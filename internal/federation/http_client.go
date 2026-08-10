package federation

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"
)

// federationHTTPClient returns a validated, timed HTTP client for outbound OIDC
// calls (issuer discovery, JWKS fetch). It blocks connections to loopback,
// link-local, cloud-metadata, multicast, reserved, and RFC-1918 private ranges,
// and enforces a hard timeout so a slow/hostile issuer cannot pin the request
// goroutine. Mirrors the SSRF protection used by the idp package.
func federationHTTPClient() *http.Client {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.DialContext = federationResolveAndValidate
	return &http.Client{
		Timeout:   15 * time.Second,
		Transport: &federationValidatedTransport{wrapped: transport},
	}
}

var federationHTTPClientFactory = federationHTTPClient

type federationValidatedTransport struct {
	wrapped http.RoundTripper
}

// federationRestrictedCIDRs enumerates the prefixes outbound OIDC connections
// must never reach (RFC-1918 private space, loopback, link-local / cloud
// metadata at 169.254.169.254, multicast, and reserved ranges).
var federationRestrictedCIDRs = []string{
	"0.0.0.0/8",
	"10.0.0.0/8",
	"100.64.0.0/10",
	"127.0.0.0/8",
	"169.254.0.0/16",
	"172.16.0.0/12",
	"192.168.0.0/16",
	"198.18.0.0/15",
	"224.0.0.0/4",
	"240.0.0.0/4",
	"::1/128",
	"fc00::/7",
	"fe80::/10",
}

func federationIsRestricted(ip net.IP) (string, bool) {
	for _, cidr := range federationRestrictedCIDRs {
		_, network, err := net.ParseCIDR(cidr)
		if err != nil {
			continue
		}
		if network.Contains(ip) {
			return cidr, true
		}
	}
	return "", false
}

func federationResolveAndValidate(ctx context.Context, network, addr string) (net.Conn, error) {
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return nil, fmt.Errorf("federation validated transport: invalid dial address %q: %w", addr, err)
	}
	ips, err := net.DefaultResolver.LookupIPAddr(ctx, host)
	if err != nil {
		return nil, fmt.Errorf("federation validated transport: host resolution failed for %s: %w", host, err)
	}
	if len(ips) == 0 {
		return nil, fmt.Errorf("federation validated transport: no addresses resolved for %s", host)
	}
	for _, ipAddr := range ips {
		if cidr, blocked := federationIsRestricted(ipAddr.IP); blocked {
			return nil, fmt.Errorf("federation validated transport: %s resolves to restricted address %s (blocked range %s)", host, ipAddr.IP, cidr)
		}
	}
	// Dial the validated IP(s), never the hostname, to close the DNS-rebinding
	// window between validation and connect. The transport still sets TLS
	// ServerName from the URL host, so certificate verification is unaffected.
	dialer := &net.Dialer{Timeout: 10 * time.Second}
	var lastErr error
	for _, ipAddr := range ips {
		conn, derr := dialer.DialContext(ctx, network, net.JoinHostPort(ipAddr.IP.String(), port))
		if derr != nil {
			lastErr = derr
			continue
		}
		return conn, nil
	}
	if lastErr == nil {
		lastErr = fmt.Errorf("federation validated transport: no usable address for %s", host)
	}
	return nil, lastErr
}

func (t *federationValidatedTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if req.URL == nil {
		return nil, errors.New("federation validated transport: request has no URL")
	}
	host := req.URL.Hostname()
	if host == "" || strings.TrimSpace(host) != host {
		return nil, errors.New("federation validated transport: invalid hostname")
	}
	ips, err := net.DefaultResolver.LookupIPAddr(req.Context(), host)
	if err != nil {
		return nil, fmt.Errorf("federation validated transport: host resolution failed for %s: %w", host, err)
	}
	for _, ipAddr := range ips {
		if cidr, blocked := federationIsRestricted(ipAddr.IP); blocked {
			return nil, fmt.Errorf("federation validated transport: %s resolves to %s (blocked range %s)", host, ipAddr.IP, cidr)
		}
	}
	if t.wrapped == nil {
		return http.DefaultTransport.RoundTrip(req)
	}
	return t.wrapped.RoundTrip(req)
}
