package safehttp

import (
	"context"
	"net"
	"testing"
)

type staticResolver map[string][]net.IPAddr

func (r staticResolver) LookupIPAddr(_ context.Context, host string) ([]net.IPAddr, error) {
	return r[host], nil
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
