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
