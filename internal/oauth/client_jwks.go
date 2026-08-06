package oauth

import (
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

// clientJWKSHTTPClient is a var so tests can point it at an httptest server.
var clientJWKSHTTPClient = &http.Client{Timeout: clientJWKSTimeout}

// resetClientJWKSCache clears the fetched-JWKS cache. Test-only.
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
