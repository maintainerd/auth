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

func isUnsafeWebhookIP(ip net.IP) bool {
	return ip.IsLoopback() ||
		ip.IsPrivate() ||
		ip.IsLinkLocalUnicast() ||
		ip.IsLinkLocalMulticast() ||
		ip.IsUnspecified() ||
		ip.IsMulticast()
}
