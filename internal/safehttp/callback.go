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
	out := *http.DefaultClient
	if base != nil {
		out = *base
	}
	out.Transport = NewCallbackTransport(out.Transport, netResolver{})
	return &out
}

func NewCallbackTransport(base http.RoundTripper, resolver Resolver) http.RoundTripper {
	baseTransport, _ := base.(*http.Transport)
	if baseTransport == nil {
		baseTransport = http.DefaultTransport.(*http.Transport)
	}
	transport := baseTransport.Clone()
	if resolver == nil {
		resolver = netResolver{}
	}
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
		var lastErr error
		for _, ip := range ips {
			if !isAllowedCallbackIP(ip) {
				return nil, fmt.Errorf("unsafe callback host %s resolved to %s", host, ip.String())
			}
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
	transport.DialTLSContext = nil
	if transport.TLSClientConfig != nil {
		transport.TLSClientConfig = transport.TLSClientConfig.Clone()
	} else {
		transport.TLSClientConfig = &tls.Config{}
	}
	return transport
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
