package safehttp

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"strings"
	"time"
)

const (
	maxCallbackRedirects          = 10
	maxCallbackResponseHeaderSize = 64 << 10
)

// NonNetworkRoundTripper is the only custom RoundTripper accepted by the
// callback client. It is reserved for deterministic in-memory test doubles
// that do not open sockets, consult proxies, or delegate to another transport.
// Network clients must inject an *http.Transport so safehttp can replace every
// dial path while preserving its non-network TLS and pooling configuration.
type NonNetworkRoundTripper interface {
	http.RoundTripper
	SafeHTTPNonNetworkTransport()
}

type Resolver interface {
	LookupIPAddr(context.Context, string) ([]net.IPAddr, error)
}

type netResolver struct{}

func (netResolver) LookupIPAddr(ctx context.Context, host string) ([]net.IPAddr, error) {
	return net.DefaultResolver.LookupIPAddr(ctx, host)
}

func ValidateCallbackURL(ctx context.Context, raw string) error {
	return ValidateCallbackURLWithResolver(ctx, raw, netResolver{})
}

func ValidateCallbackURLWithResolver(ctx context.Context, raw string, resolver Resolver) error {
	u, err := url.Parse(raw)
	if err != nil {
		return err
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return fmt.Errorf("invalid callback scheme")
	}
	if u.Host == "" {
		return fmt.Errorf("invalid callback host")
	}
	if u.User != nil {
		return fmt.Errorf("callback URL credentials are not allowed")
	}
	host := u.Hostname()
	if host == "" {
		return fmt.Errorf("invalid callback host")
	}
	if resolver == nil {
		resolver = netResolver{}
	}
	ips, err := resolveCallbackHost(ctx, resolver, host)
	if err != nil {
		return err
	}
	for _, ip := range ips {
		if !isAllowedCallbackIP(ip) {
			return fmt.Errorf("unsafe callback host %s resolved to %s", host, ip.String())
		}
	}
	return nil
}

func NewCallbackClient(base *http.Client) *http.Client {
	return newCallbackClientWithResolver(base, netResolver{})
}

func newCallbackClientWithResolver(base *http.Client, resolver Resolver) *http.Client {
	out := *http.DefaultClient
	if base != nil {
		out = *base
	}
	if resolver == nil {
		resolver = netResolver{}
	}
	baseRedirect := out.CheckRedirect
	out.CheckRedirect = callbackRedirectPolicy(resolver, baseRedirect)
	out.Transport = NewCallbackTransport(out.Transport, resolver)
	// Outbound bidder and callback clients are stateless. Keeping an injected
	// jar would reintroduce credentials after redirect-header stripping.
	out.Jar = nil
	return &out
}

func NewCallbackTransport(base http.RoundTripper, resolver Resolver) http.RoundTripper {
	if resolver == nil {
		resolver = netResolver{}
	}
	if existing, ok := base.(*callbackRoundTripper); ok {
		return existing
	}
	if base == nil {
		base = http.DefaultTransport
	}
	if memory, ok := base.(NonNetworkRoundTripper); ok {
		return &callbackRoundTripper{base: memory, resolver: resolver}
	}
	baseTransport, ok := base.(*http.Transport)
	if !ok {
		return errorRoundTripper{err: fmt.Errorf("safehttp: custom network RoundTripper %T is not supported", base)}
	}
	transport := baseTransport.Clone()
	if transport.TLSClientConfig != nil && transport.TLSClientConfig.InsecureSkipVerify {
		return errorRoundTripper{err: fmt.Errorf("safehttp: TLS certificate verification cannot be disabled")}
	}
	transport.Proxy = nil
	dialer := &net.Dialer{Timeout: 30 * time.Second, KeepAlive: 30 * time.Second}
	transport.DialContext = func(ctx context.Context, network, address string) (net.Conn, error) {
		host, port, err := net.SplitHostPort(address)
		if err != nil {
			return nil, err
		}
		ips, err := resolveCallbackHost(ctx, resolver, host)
		if err != nil {
			return nil, err
		}
		// Validate the complete answer before trying any member. Otherwise a
		// public first result could hide a denied later result during rebinding.
		for _, ip := range ips {
			if !isAllowedCallbackIP(ip) {
				return nil, fmt.Errorf("unsafe callback host %s resolved to %s", host, ip.String())
			}
		}
		var lastErr error
		for _, ip := range ips {
			conn, err := dialer.DialContext(ctx, network, net.JoinHostPort(ip.String(), port))
			if err == nil {
				return conn, nil
			}
			lastErr = err
		}
		if lastErr != nil {
			return nil, lastErr
		}
		return nil, fmt.Errorf("callback host %s resolved to no addresses", host)
	}
	//lint:ignore SA1019 A legacy injected DialTLS hook must also be cleared or it can bypass DialContext.
	transport.DialTLS = nil
	transport.DialTLSContext = nil
	if transport.TLSClientConfig != nil {
		transport.TLSClientConfig = transport.TLSClientConfig.Clone()
	} else {
		transport.TLSClientConfig = &tls.Config{}
	}
	if transport.TLSClientConfig.MinVersion < tls.VersionTLS12 {
		transport.TLSClientConfig.MinVersion = tls.VersionTLS12
	}
	// Certificate identity must follow the request host, not an injected
	// transport-wide override.
	transport.TLSClientConfig.ServerName = ""
	if transport.MaxResponseHeaderBytes <= 0 || transport.MaxResponseHeaderBytes > maxCallbackResponseHeaderSize {
		transport.MaxResponseHeaderBytes = maxCallbackResponseHeaderSize
	}
	return &callbackRoundTripper{base: transport, resolver: resolver}
}

type callbackRoundTripper struct {
	base     http.RoundTripper
	resolver Resolver
}

func (t *callbackRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	if req == nil || req.URL == nil {
		return nil, fmt.Errorf("safehttp: request URL is required")
	}
	if err := ValidateCallbackURLWithResolver(req.Context(), req.URL.String(), t.resolver); err != nil {
		return nil, err
	}
	return t.base.RoundTrip(req)
}

type errorRoundTripper struct {
	err error
}

func (t errorRoundTripper) RoundTrip(*http.Request) (*http.Response, error) {
	return nil, t.err
}

func callbackRedirectPolicy(resolver Resolver, base func(*http.Request, []*http.Request) error) func(*http.Request, []*http.Request) error {
	return func(req *http.Request, via []*http.Request) error {
		if len(via) >= maxCallbackRedirects {
			return fmt.Errorf("stopped after %d redirects", maxCallbackRedirects)
		}
		if req == nil || req.URL == nil {
			return fmt.Errorf("redirect URL is required")
		}
		if base != nil {
			if err := base(req, via); err != nil {
				return err
			}
		}
		// Run the mandatory policy after the injected hook so it cannot mutate
		// the hop or restore credentials after validation.
		if err := ValidateCallbackURLWithResolver(req.Context(), req.URL.String(), resolver); err != nil {
			return err
		}
		for _, previous := range via {
			if previous == nil || previous.URL == nil || !sameCallbackAuthority(req.URL, previous.URL) {
				stripRedirectCredentials(req)
				break
			}
		}
		return nil
	}
}

func sameCallbackAuthority(a, b *url.URL) bool {
	if a == nil || b == nil || !strings.EqualFold(a.Scheme, b.Scheme) {
		return false
	}
	return strings.EqualFold(a.Hostname(), b.Hostname()) && callbackPort(a) == callbackPort(b)
}

func callbackPort(u *url.URL) string {
	if port := u.Port(); port != "" {
		return port
	}
	if strings.EqualFold(u.Scheme, "https") {
		return "443"
	}
	return "80"
}

func stripRedirectCredentials(req *http.Request) {
	// Credential refs may use arbitrary non-hop-by-hop header names, so a
	// denylist cannot prove that a secret was removed.
	req.Header = make(http.Header)
	req.URL.User = nil
}

func resolveCallbackHost(ctx context.Context, resolver Resolver, host string) ([]net.IP, error) {
	host = strings.TrimSpace(host)
	if host == "" {
		return nil, fmt.Errorf("invalid callback host")
	}
	if ip := net.ParseIP(host); ip != nil {
		return []net.IP{ip}, nil
	}
	addrs, err := resolver.LookupIPAddr(ctx, host)
	if err != nil {
		return nil, err
	}
	if len(addrs) == 0 {
		return nil, fmt.Errorf("callback host %s resolved to no addresses", host)
	}
	ips := make([]net.IP, 0, len(addrs))
	for _, addr := range addrs {
		ips = append(ips, addr.IP)
	}
	return ips, nil
}

func isAllowedCallbackIP(ip net.IP) bool {
	if ip == nil {
		return false
	}
	addr, ok := netip.AddrFromSlice(ip)
	if !ok {
		return false
	}
	addr = addr.Unmap()
	if !addr.IsValid() || !addr.IsGlobalUnicast() {
		return false
	}
	if addr.Is4() {
		return !prefixContains(callbackDeniedIPv4, addr)
	}
	// Only the currently assignable global-unicast block is eligible. This
	// rejects reserved/future IPv6 space without assuming IsGlobalUnicast means
	// publicly routable, then removes special-purpose subranges within it.
	if !callbackPublicIPv6.Contains(addr) {
		return false
	}
	return !prefixContains(callbackDeniedIPv6, addr)
}

var (
	callbackPublicIPv6 = netip.MustParsePrefix("2000::/3")
	callbackDeniedIPv4 = []netip.Prefix{
		netip.MustParsePrefix("0.0.0.0/8"),     // current network / unspecified
		netip.MustParsePrefix("10.0.0.0/8"),    // private use
		netip.MustParsePrefix("100.64.0.0/10"), // shared address space / CGNAT
		netip.MustParsePrefix("127.0.0.0/8"),   // loopback
		netip.MustParsePrefix("169.254.0.0/16"),
		netip.MustParsePrefix("172.16.0.0/12"),
		netip.MustParsePrefix("192.0.0.0/24"),
		netip.MustParsePrefix("192.0.2.0/24"),
		netip.MustParsePrefix("192.88.99.0/24"),
		netip.MustParsePrefix("192.168.0.0/16"),
		netip.MustParsePrefix("198.18.0.0/15"),
		netip.MustParsePrefix("198.51.100.0/24"),
		netip.MustParsePrefix("203.0.113.0/24"),
		netip.MustParsePrefix("224.0.0.0/4"),
		netip.MustParsePrefix("240.0.0.0/4"),
	}
	callbackDeniedIPv6 = []netip.Prefix{
		netip.MustParsePrefix("2001::/32"),    // Teredo
		netip.MustParsePrefix("2001:2::/48"),  // benchmarking
		netip.MustParsePrefix("2001:10::/28"), // deprecated ORCHID
		netip.MustParsePrefix("2001:20::/28"), // ORCHIDv2
		netip.MustParsePrefix("2001:db8::/32"),
		netip.MustParsePrefix("2002::/16"), // 6to4
		netip.MustParsePrefix("3fff::/20"), // documentation
	}
)

func prefixContains(prefixes []netip.Prefix, addr netip.Addr) bool {
	for _, prefix := range prefixes {
		if prefix.Contains(addr) {
			return true
		}
	}
	return false
}
