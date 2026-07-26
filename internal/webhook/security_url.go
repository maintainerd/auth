package webhook

import (
	"context"
	"fmt"
	"net"
	"net/url"
)

// ipLookup abstracts DNS resolution so tests can inject a stub.
type ipLookup interface {
	LookupIPAddr(ctx context.Context, host string) ([]net.IPAddr, error)
}

// ValidateDeliveryURL validates a webhook destination URL at delivery time:
// it resolves DNS and rejects loopback/private/link-local/metadata addresses.
// Use this on every outbound request (and on each redirect hop) to close the
// DNS-rebinding / TOCTOU SSRF window left by registration-time validation.
func ValidateDeliveryURL(ctx context.Context, raw string) error {
	return validateWebhookURL(ctx, raw, true, net.DefaultResolver)
}

func validateWebhookURL(ctx context.Context, raw string, resolve bool, resolver ipLookup) error {
	parsed, err := url.Parse(raw)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return fmt.Errorf("webhook URL must be valid")
	}
	if parsed.Scheme != "https" {
		return fmt.Errorf("webhook URL must use https")
	}

	host := parsed.Hostname()
	if host == "" {
		return fmt.Errorf("webhook URL host is required")
	}
	if ip := net.ParseIP(host); ip != nil {
		if isUnsafeWebhookIP(ip) {
			return fmt.Errorf("webhook URL host is not allowed")
		}
		return nil
	}
	if !resolve {
		return nil
	}

	addrs, err := resolver.LookupIPAddr(ctx, host)
	if err != nil {
		return fmt.Errorf("resolve webhook URL host: %w", err)
	}
	for _, addr := range addrs {
		if isUnsafeWebhookIP(addr.IP) {
			return fmt.Errorf("webhook URL host resolves to a private address")
		}
	}
	return nil
}

// extraBlockedNets covers ranges the net.IP helpers do NOT classify but which
// are still unsafe SSRF targets: CGNAT/shared address space (RFC 6598), the
// NAT64 well-known prefix (can reach IPv4 privates through a NAT64 gateway),
// IETF protocol assignments, and the benchmarking range.
var extraBlockedNets = func() []*net.IPNet {
	cidrs := []string{
		"100.64.0.0/10",  // RFC 6598 CGNAT / shared address space
		"192.0.0.0/24",   // RFC 6890 IETF protocol assignments
		"198.18.0.0/15",  // RFC 2544 benchmarking
		"64:ff9b::/96",   // RFC 6052 NAT64 well-known prefix
		"64:ff9b:1::/48", // RFC 8215 NAT64 local-use prefix
	}
	nets := make([]*net.IPNet, 0, len(cidrs))
	for _, c := range cidrs {
		if _, n, err := net.ParseCIDR(c); err == nil {
			nets = append(nets, n)
		}
	}
	return nets
}()

// ValidateResolvedHostSafe resolves the URL's host best-effort and rejects only
// when a resolved address is unsafe. A resolution FAILURE returns nil (not a
// rejection): transient DNS must never block a valid registration, and
// delivery-time pinned dialing is the authoritative guard. Literal-IP hosts are
// already covered by validateWebhookURL, so they short-circuit here.
func ValidateResolvedHostSafe(ctx context.Context, raw string) error {
	parsed, err := url.Parse(raw)
	if err != nil || parsed.Hostname() == "" {
		return nil
	}
	host := parsed.Hostname()
	if net.ParseIP(host) != nil {
		return nil
	}
	addrs, err := net.DefaultResolver.LookupIPAddr(ctx, host)
	if err != nil {
		return nil
	}
	for _, a := range addrs {
		if isUnsafeWebhookIP(a.IP) {
			return fmt.Errorf("webhook URL resolves to a disallowed address")
		}
	}
	return nil
}

func isUnsafeWebhookIP(ip net.IP) bool {
	if ip.IsLoopback() ||
		ip.IsPrivate() ||
		ip.IsLinkLocalUnicast() ||
		ip.IsLinkLocalMulticast() ||
		ip.IsUnspecified() ||
		ip.IsMulticast() {
		return true
	}
	for _, n := range extraBlockedNets {
		if n.Contains(ip) {
			return true
		}
	}
	return false
}
