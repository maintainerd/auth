package idp

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"
)

// idpHTTPClient returns a validated, timed HTTP client for all outbound IdP
// calls (OIDC discovery, OAuth2 code exchange, token exchange, userinfo). It
// blocks connections to loopback, link-local, cloud-metadata, multicast,
// reserved, and RFC-1918 private ranges, and enforces a hard timeout so a
// slow/hostile provider cannot pin the request goroutine indefinitely.
func idpHTTPClient() *http.Client {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.DialContext = resolveAndValidate
	return &http.Client{
		Timeout: 30 * time.Second,
		Transport: &idpValidatedTransport{
			wrapped: transport,
		},
	}
}

var idpHTTPClientFactory = idpHTTPClient

// idpValidatedTransport resolves the target hostname to its IP(s) and rejects
// any connection to a restricted address before the dial, preventing SSRF to
// internal infrastructure. The wrapped transport handles the actual connection.
type idpValidatedTransport struct {
	wrapped http.RoundTripper
}

// restrictedCIDRs enumerates the IPv4 and IPv6 prefixes that outbound IdP
// connections must never reach. This includes RFC-1918 private space,
// loopback, link-local (which covers AWS/GCP/Azure cloud-metadata at
// 169.254.169.254), multicast, and reserved ranges.
var restrictedCIDRs = []string{
	"0.0.0.0/8",      // "This host on this network" (RFC 1122 §3.2.1.3)
	"10.0.0.0/8",     // RFC 1918 private
	"100.64.0.0/10",  // RFC 6598 CGN
	"127.0.0.0/8",    // Loopback
	"169.254.0.0/16", // Link-local / cloud metadata
	"172.16.0.0/12",  // RFC 1918 private
	"192.168.0.0/16", // RFC 1918 private
	"198.18.0.0/15",  // Benchmarking (RFC 2544)
	"224.0.0.0/4",    // Multicast
	"240.0.0.0/4",    // Reserved (RFC 1112 §4)
	"::1/128",        // IPv6 loopback
	"fc00::/7",       // IPv6 unique local
	"fe80::/10",      // IPv6 link-local
}

func resolveAndValidate(ctx context.Context, network, addr string) (net.Conn, error) {
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		host = addr
	}
	ips, err := net.DefaultResolver.LookupIPAddr(ctx, host)
	if err != nil {
		return nil, fmt.Errorf("idp validated transport: host resolution failed for %s: %w", host, err)
	}
	if len(ips) == 0 {
		return nil, fmt.Errorf("idp validated transport: no addresses resolved for %s", host)
	}
	for _, ipAddr := range ips {
		for _, cidr := range restrictedCIDRs {
			_, network, err := net.ParseCIDR(cidr)
			if err != nil {
				continue
			}
			if network.Contains(ipAddr.IP) {
				return nil, fmt.Errorf("idp validated transport: %s resolves to restricted address %s (blocked range %s)", host, ipAddr.IP, cidr)
			}
		}
	}
	dialer := &net.Dialer{Timeout: 10 * time.Second}
	return dialer.DialContext(ctx, network, addr)
}

func (t *idpValidatedTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if req.URL == nil {
		return nil, errors.New("idp validated transport: request has no URL")
	}
	host := req.URL.Hostname()
	if host == "" {
		return nil, errors.New("idp validated transport: request has empty hostname")
	}
	if strings.TrimSpace(host) != host {
		return nil, errors.New("idp validated transport: hostname is not valid")
	}
	ips, err := net.DefaultResolver.LookupIPAddr(req.Context(), host)
	if err != nil {
		return nil, fmt.Errorf("idp validated transport: host resolution failed for %s: %w", host, err)
	}
	for _, ipAddr := range ips {
		for _, cidr := range restrictedCIDRs {
			_, network, _ := net.ParseCIDR(cidr)
			if network != nil && network.Contains(ipAddr.IP) {
				return nil, fmt.Errorf("idp validated transport: %s resolves to %s (blocked range %s)", host, ipAddr.IP, cidr)
			}
		}
	}
	if t.wrapped == nil {
		return http.DefaultTransport.RoundTrip(req)
	}
	return t.wrapped.RoundTrip(req)
}
