package safehttp

import (
	"context"
	"crypto/tls"
	"io"
	"net"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"
)

type staticResolver map[string][]net.IPAddr

func (r staticResolver) LookupIPAddr(_ context.Context, host string) ([]net.IPAddr, error) {
	return r[host], nil
}

type sequenceResolver struct {
	mu      sync.Mutex
	answers [][]net.IPAddr
}

func (r *sequenceResolver) LookupIPAddr(_ context.Context, _ string) ([]net.IPAddr, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	answer := r.answers[0]
	if len(r.answers) > 1 {
		r.answers = r.answers[1:]
	}
	return answer, nil
}

type memoryRoundTripFunc func(*http.Request) (*http.Response, error)

func (f memoryRoundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func (memoryRoundTripFunc) SafeHTTPNonNetworkTransport() {}

type unmarkedRoundTripFunc func(*http.Request) (*http.Response, error)

func (f unmarkedRoundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func TestValidateCallbackURLRejectsUnsafeTargets(t *testing.T) {
	resolver := staticResolver{
		"loopback.example":  {{IP: net.ParseIP("127.0.0.1")}},
		"private.example":   {{IP: net.ParseIP("10.0.0.1")}},
		"linklocal.example": {{IP: net.ParseIP("169.254.169.254")}},
		"unspecified.test":  {{IP: net.ParseIP("0.0.0.0")}},
		"ipv6local.test":    {{IP: net.ParseIP("::1")}},
	}
	for host := range resolver {
		raw := "http://" + host + "/callback"
		if err := ValidateCallbackURLWithResolver(context.Background(), raw, resolver); err == nil {
			t.Fatalf("ValidateCallbackURLWithResolver(%q) succeeded, want rejection", raw)
		}
	}
}

func TestAllowedCallbackIPUsesReviewedSpecialPurposePolicy(t *testing.T) {
	tests := []struct {
		ip      string
		allowed bool
	}{
		{"8.8.8.8", true},
		{"100.63.255.255", true},
		{"100.64.0.0", false},
		{"100.127.255.255", false},
		{"100.128.0.0", true},
		{"198.17.255.255", true},
		{"198.18.0.0", false},
		{"198.19.255.255", false},
		{"198.20.0.0", true},
		{"203.0.114.1", true},
		{"0.0.0.0", false},
		{"10.1.2.3", false},
		{"127.0.0.1", false},
		{"169.254.169.254", false},
		{"172.31.255.255", false},
		{"192.0.0.9", false},
		{"192.0.2.1", false},
		{"192.88.99.2", false},
		{"192.168.1.1", false},
		{"198.51.100.1", false},
		{"203.0.113.1", false},
		{"224.0.0.1", false},
		{"240.0.0.1", false},
		{"255.255.255.255", false},
		{"2001:4860:4860::8888", true},
		{"2606:4700:4700::1111", true},
		{"::", false},
		{"::1", false},
		{"64:ff9b::c000:201", false},
		{"100::1", false},
		{"2001::1", false},
		{"2001:2::1", false},
		{"2001:10::1", false},
		{"2001:20::1", false},
		{"2001:db8::1", false},
		{"2002:c000:0201::1", false},
		{"3fff::1", false},
		{"fc00::1", false},
		{"fe80::1", false},
		{"ff02::1", false},
		{"::ffff:127.0.0.1", false},
		{"::ffff:8.8.8.8", true},
	}
	for _, test := range tests {
		t.Run(test.ip, func(t *testing.T) {
			if got := isAllowedCallbackIP(net.ParseIP(test.ip)); got != test.allowed {
				t.Fatalf("isAllowedCallbackIP(%s) = %t, want %t", test.ip, got, test.allowed)
			}
		})
	}
}

func TestValidateCallbackURLAllowsPublicTargets(t *testing.T) {
	resolver := staticResolver{
		"public.example": {{IP: net.ParseIP("93.184.216.34")}},
	}
	if err := ValidateCallbackURLWithResolver(context.Background(), "https://public.example/callback", resolver); err != nil {
		t.Fatalf("public callback rejected: %v", err)
	}
}

func TestValidateCallbackURLRejectsMixedDNSResults(t *testing.T) {
	resolver := staticResolver{
		"mixed.example": {
			{IP: net.ParseIP("93.184.216.34")},
			{IP: net.ParseIP("127.0.0.1")},
		},
	}
	if err := ValidateCallbackURLWithResolver(context.Background(), "https://mixed.example/callback", resolver); err == nil {
		t.Fatal("mixed public/private DNS result succeeded, want rejection")
	}
}

func TestValidateCallbackURLRejectsEmbeddedCredentials(t *testing.T) {
	err := ValidateCallbackURLWithResolver(context.Background(), "https://user:secret@8.8.8.8/callback", nil)
	if err == nil || !strings.Contains(err.Error(), "credentials") {
		t.Fatalf("credential-bearing URL error = %v, want rejection", err)
	}
}

func TestCallbackTransportRejectsRebindWithMixedDialAnswer(t *testing.T) {
	resolver := &sequenceResolver{answers: [][]net.IPAddr{
		{{IP: net.ParseIP("93.184.216.34")}},
		{{IP: net.ParseIP("93.184.216.34")}, {IP: net.ParseIP("127.0.0.1")}},
	}}
	client := newCallbackClientWithResolver(nil, resolver)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://rebind.example/callback", nil)
	if err != nil {
		t.Fatal(err)
	}
	_, err = client.Do(req)
	if err == nil || !strings.Contains(err.Error(), "unsafe callback host") {
		t.Fatalf("rebind request error = %v, want mixed-answer rejection", err)
	}
}

func TestCallbackTransportDisablesProxyAndUnsafeTLSHooks(t *testing.T) {
	base := &http.Transport{
		Proxy:                  http.ProxyFromEnvironment,
		DialTLSContext:         func(context.Context, string, string) (net.Conn, error) { return nil, nil },
		TLSClientConfig:        &tls.Config{MinVersion: tls.VersionTLS10, ServerName: "override.example"},
		MaxResponseHeaderBytes: 1 << 20,
	}
	guarded, ok := NewCallbackTransport(base, nil).(*callbackRoundTripper)
	if !ok {
		t.Fatalf("transport type = %T, want guarded transport", NewCallbackTransport(base, nil))
	}
	network, ok := guarded.base.(*http.Transport)
	if !ok {
		t.Fatalf("guarded base = %T, want *http.Transport", guarded.base)
	}
	if network.Proxy != nil || network.DialTLSContext != nil {
		t.Fatal("proxy or custom TLS dial hook survived safe transport wrapping")
	}
	if network.TLSClientConfig == nil || network.TLSClientConfig.MinVersion != tls.VersionTLS12 {
		t.Fatalf("TLS minimum = %#x, want TLS 1.2", network.TLSClientConfig.MinVersion)
	}
	if network.TLSClientConfig.ServerName != "" {
		t.Fatalf("TLS server-name override = %q, want request-derived identity", network.TLSClientConfig.ServerName)
	}
	if network.MaxResponseHeaderBytes != maxCallbackResponseHeaderSize {
		t.Fatalf("response header limit = %d, want %d", network.MaxResponseHeaderBytes, maxCallbackResponseHeaderSize)
	}
}

func TestCallbackTransportRejectsInsecureTLSAndUnmarkedCustomTransport(t *testing.T) {
	for name, transport := range map[string]http.RoundTripper{
		"insecure TLS": &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}}, // #nosec G402 -- rejection fixture
		"custom network": unmarkedRoundTripFunc(func(*http.Request) (*http.Response, error) {
			return nil, nil
		}),
	} {
		t.Run(name, func(t *testing.T) {
			req, err := http.NewRequest(http.MethodGet, "https://8.8.8.8/callback", nil)
			if err != nil {
				t.Fatal(err)
			}
			_, err = NewCallbackTransport(transport, nil).RoundTrip(req)
			if err == nil {
				t.Fatal("unsafe injected transport was accepted")
			}
		})
	}
}

func TestCallbackClientValidatesMarkedInMemoryTransport(t *testing.T) {
	calls := 0
	base := &http.Client{Transport: memoryRoundTripFunc(func(req *http.Request) (*http.Response, error) {
		calls++
		return callbackTestResponse(req, http.StatusNoContent, ""), nil
	})}
	client := NewCallbackClient(base)
	if _, err := client.Get("http://127.0.0.1/callback"); err == nil {
		t.Fatal("in-memory transport bypassed denied target validation")
	}
	if calls != 0 {
		t.Fatalf("in-memory transport calls = %d, want zero for denied target", calls)
	}
	if _, err := client.Get("https://8.8.8.8/callback"); err != nil {
		t.Fatalf("public target through in-memory transport failed: %v", err)
	}
	if calls != 1 {
		t.Fatalf("in-memory transport calls = %d, want one", calls)
	}
}

func TestCallbackClientRejectsUnsafeRedirectAndStripsCrossAuthorityCredentials(t *testing.T) {
	var redirected http.Header
	base := &http.Client{Transport: memoryRoundTripFunc(func(req *http.Request) (*http.Response, error) {
		switch req.URL.Host {
		case "8.8.8.8":
			return callbackTestResponse(req, http.StatusFound, "https://1.1.1.1/next"), nil
		case "1.1.1.1":
			redirected = req.Header.Clone()
			return callbackTestResponse(req, http.StatusNoContent, ""), nil
		default:
			t.Fatalf("unexpected redirect host %q", req.URL.Host)
			return nil, nil
		}
	})}
	client := NewCallbackClient(base)
	req, err := http.NewRequest(http.MethodGet, "https://8.8.8.8/start", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer secret")
	req.Header.Set("X-Api-Key", "secret")
	req.Header.Set("Accept", "application/json")
	if _, err := client.Do(req); err != nil {
		t.Fatal(err)
	}
	if redirected.Get("Authorization") != "" || redirected.Get("X-Api-Key") != "" {
		t.Fatalf("redirect retained credentials: %#v", redirected)
	}
	if len(redirected) != 0 {
		t.Fatalf("cross-authority redirect retained headers: %#v", redirected)
	}

	calls := 0
	client = NewCallbackClient(&http.Client{Transport: memoryRoundTripFunc(func(req *http.Request) (*http.Response, error) {
		calls++
		return callbackTestResponse(req, http.StatusFound, "http://127.0.0.1/private"), nil
	})})
	if _, err := client.Get("https://8.8.8.8/start"); err == nil || !strings.Contains(err.Error(), "unsafe callback host") {
		t.Fatalf("unsafe redirect error = %v, want rejection", err)
	}
	if calls != 1 {
		t.Fatalf("transport calls = %d, want denied redirect stopped before second hop", calls)
	}
}

func TestCallbackClientReappliesPolicyAfterInjectedRedirectHook(t *testing.T) {
	calls := 0
	base := &http.Client{
		Transport: memoryRoundTripFunc(func(req *http.Request) (*http.Response, error) {
			calls++
			return callbackTestResponse(req, http.StatusFound, "https://1.1.1.1/next"), nil
		}),
		CheckRedirect: func(req *http.Request, _ []*http.Request) error {
			req.URL.Host = "127.0.0.1"
			req.Header.Set("Authorization", "restored secret")
			return nil
		},
	}
	_, err := NewCallbackClient(base).Get("https://8.8.8.8/start")
	if err == nil || !strings.Contains(err.Error(), "unsafe callback host") {
		t.Fatalf("mutated redirect error = %v, want mandatory policy rejection", err)
	}
	if calls != 1 {
		t.Fatalf("transport calls = %d, want injected hook stopped before second hop", calls)
	}
}

func TestCallbackClientPreservesTimeoutAndBoundsRedirects(t *testing.T) {
	timed := NewCallbackClient(&http.Client{
		Timeout: 20 * time.Millisecond,
		Transport: memoryRoundTripFunc(func(req *http.Request) (*http.Response, error) {
			<-req.Context().Done()
			return nil, req.Context().Err()
		}),
	})
	started := time.Now()
	if _, err := timed.Get("https://8.8.8.8/wait"); err == nil {
		t.Fatal("injected client timeout was not preserved")
	}
	if elapsed := time.Since(started); elapsed > 500*time.Millisecond {
		t.Fatalf("client timeout elapsed = %s, want prompt cancellation", elapsed)
	}

	calls := 0
	redirecting := NewCallbackClient(&http.Client{Transport: memoryRoundTripFunc(func(req *http.Request) (*http.Response, error) {
		calls++
		return callbackTestResponse(req, http.StatusFound, "/again"), nil
	})})
	if _, err := redirecting.Get("https://8.8.8.8/start"); err == nil || !strings.Contains(err.Error(), "10 redirects") {
		t.Fatalf("redirect limit error = %v", err)
	}
	if calls != maxCallbackRedirects {
		t.Fatalf("redirect transport calls = %d, want %d", calls, maxCallbackRedirects)
	}
}

func callbackTestResponse(req *http.Request, status int, location string) *http.Response {
	header := make(http.Header)
	if location != "" {
		header.Set("Location", location)
	}
	return &http.Response{
		StatusCode: status,
		Status:     http.StatusText(status),
		Header:     header,
		Body:       io.NopCloser(strings.NewReader("")),
		Request:    req,
	}
}
