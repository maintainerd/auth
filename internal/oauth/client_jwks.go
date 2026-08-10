package oauth

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

const (
	// clientJWKSCacheTTL bounds how stale a fetched JWKS may be. Short enough that
	// a client rotating its key recovers without operator action, long enough that
	// the token endpoint does not make an outbound request per authentication.
	clientJWKSCacheTTL = 5 * time.Minute
	// clientJWKSMaxBytes caps the response body. A jwks_uri is operator-registered
	// but still points at a third-party host, so an unbounded read is a memory
	// exhaustion primitive against the token endpoint.
	clientJWKSMaxBytes = 512 * 1024
	clientJWKSTimeout  = 5 * time.Second
)

type cachedClientJWKS struct {
	raw       []byte
	expiresAt time.Time
}

var clientJWKSCache = struct {
	mu      sync.RWMutex
	entries map[string]cachedClientJWKS
}{entries: map[string]cachedClientJWKS{}}

// clientJWKSRestrictedCIDRs are the ranges the jwks_uri fetch must never reach.
// Loopback is intentionally omitted: validateClientJWKSURI already permits an
// http loopback jwks_uri for local development, and a fetch to the server's own
// localhost is far lower risk than the cloud-metadata / internal-network pivots
// this list blocks. Everything else — RFC-1918, CGN, link-local (169.254.169.254
// cloud metadata), multicast, reserved — is denied, INCLUDING when reached via a
// DNS name that resolves into one of these ranges, and across HTTP redirects.
var clientJWKSRestrictedCIDRs = []string{
	"10.0.0.0/8",
	"100.64.0.0/10",
	"169.254.0.0/16",
	"172.16.0.0/12",
	"192.168.0.0/16",
	"198.18.0.0/15",
	"224.0.0.0/4",
	"240.0.0.0/4",
	"fc00::/7",
	"fe80::/10",
}

// clientJWKSResolveAndValidate resolves the target host, rejects any restricted
// resolution, and then dials the exact validated IP — never the hostname — so a
// DNS name cannot rebind to an internal address between the check and the
// connect. The http.Transport still derives TLS ServerName from the URL host, so
// certificate verification is unaffected.
func clientJWKSResolveAndValidate(ctx context.Context, network, addr string) (net.Conn, error) {
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return nil, fmt.Errorf("jwks_uri transport: invalid dial address")
	}
	ips, err := net.DefaultResolver.LookupIPAddr(ctx, host)
	if err != nil {
		return nil, fmt.Errorf("jwks_uri transport: host resolution failed")
	}
	if len(ips) == 0 {
		return nil, fmt.Errorf("jwks_uri transport: no addresses resolved")
	}
	for _, ipAddr := range ips {
		for _, cidr := range clientJWKSRestrictedCIDRs {
			_, ipNet, perr := net.ParseCIDR(cidr)
			if perr != nil {
				continue
			}
			if ipNet.Contains(ipAddr.IP) {
				return nil, fmt.Errorf("jwks_uri resolves to a restricted address")
			}
		}
	}
	dialer := &net.Dialer{Timeout: clientJWKSTimeout}
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
		lastErr = fmt.Errorf("jwks_uri has no usable address")
	}
	return nil, lastErr
}

func newClientJWKSHTTPClient() *http.Client {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.DialContext = clientJWKSResolveAndValidate
	return &http.Client{
		Timeout:   clientJWKSTimeout,
		Transport: transport,
		// Re-validate every redirect hop: a compliant https jwks_uri must not be
		// able to bounce the fetch to http or to an IP-literal private/loopback
		// host, and must not redirect indefinitely. The transport independently
		// blocks DNS names that resolve into a restricted range on every hop.
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 5 {
				return fmt.Errorf("jwks_uri redirected too many times")
			}
			return validateClientJWKSURI(req.URL.String())
		},
	}
}

// clientJWKSHTTPClient is a var so tests can point it at an httptest server.
var clientJWKSHTTPClient = newClientJWKSHTTPClient()

// resetClientJWKSCache clears the fetched-JWKS cache. Test-only.
	//lint:ignore U1000 pre-existing; retained for future use
func resetClientJWKSCache() {
	clientJWKSCache.mu.Lock()
	defer clientJWKSCache.mu.Unlock()
	clientJWKSCache.entries = map[string]cachedClientJWKS{}
}

// fetchClientJWKS dereferences a client's registered jwks_uri.
//
// A client may register either an inline `jwks` document or a `jwks_uri`, and
// the registry accepts both. findClientJWK only ever read the inline document,
// so private_key_jwt authentication failed unconditionally for every client that
// chose the URI form even though it had passed registration — the client had no
// way to authenticate at all.
func fetchClientJWKS(rawURI string) ([]byte, error) {
	uri := strings.TrimSpace(rawURI)
	if err := validateClientJWKSURI(uri); err != nil {
		return nil, err
	}

	clientJWKSCache.mu.RLock()
	entry, ok := clientJWKSCache.entries[uri]
	clientJWKSCache.mu.RUnlock()
	if ok && entry.expiresAt.After(time.Now()) {
		return entry.raw, nil
	}

	resp, err := clientJWKSHTTPClient.Get(uri)
	if err != nil {
		return nil, fmt.Errorf("jwks_uri fetch failed")
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("jwks_uri returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, clientJWKSMaxBytes+1))
	if err != nil {
		return nil, fmt.Errorf("jwks_uri body could not be read")
	}
	if len(body) > clientJWKSMaxBytes {
		return nil, fmt.Errorf("jwks_uri document is too large")
	}
	if !json.Valid(body) {
		return nil, fmt.Errorf("jwks_uri did not return valid JSON")
	}

	clientJWKSCache.mu.Lock()
	clientJWKSCache.entries[uri] = cachedClientJWKS{raw: body, expiresAt: time.Now().Add(clientJWKSCacheTTL)}
	clientJWKSCache.mu.Unlock()

	return body, nil
}

// validateClientJWKSURI keeps the token endpoint from being used as an SSRF
// primitive. The URI is operator-registered, but the token endpoint is
// unauthenticated up to the point this runs, so anyone who can register a client
// (or compromise the console) would otherwise be able to make this server issue
// GETs to arbitrary internal addresses and observe the outcome through the
// authentication error.
//
// https only, except loopback over http so local development still works, and
// no literal private / link-local / loopback address behind an https URL.
func validateClientJWKSURI(raw string) error {
	if raw == "" {
		return fmt.Errorf("client has no jwks_uri configured")
	}
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" {
		return fmt.Errorf("jwks_uri is not a valid absolute URL")
	}

	host := u.Hostname()
	isLoopbackName := host == "localhost" || host == "127.0.0.1" || host == "::1"

	switch u.Scheme {
	case "https":
	case "http":
		if !isLoopbackName {
			return fmt.Errorf("jwks_uri must use https")
		}
		return nil
	default:
		return fmt.Errorf("jwks_uri must use https")
	}

	if ip := net.ParseIP(host); ip != nil {
		if ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() || ip.IsUnspecified() {
			return fmt.Errorf("jwks_uri must not point at a private or loopback address")
		}
	}
	return nil
}
