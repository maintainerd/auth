package oauth

import (
	"sync"
	"time"
)

// assertionReplayWindow is how long a spent client-assertion `jti` is remembered.
// It matches the maximum age an assertion may have (assertionMaxAge), so an entry
// is only ever dropped once the assertion it guards has expired on its own.
const assertionReplayWindow = assertionMaxAge

// clientAssertionReplayGuard remembers the `jti` of every client assertion that
// has already authenticated a client.
//
// RFC 7523 §3 point 7 requires the authorization server to reject a repeated
// jti within the assertion's validity window. Without it a single captured
// assertion — from a proxy log, an error report, a compromised TLS terminator —
// authenticates the client over and over for the whole five-minute window, which
// defeats the point of using a signed assertion instead of a bearer secret.
//
// This store is per-process. A multi-replica deployment can still accept the
// same assertion once per replica; closing that needs a shared store, which is
// tracked separately. Per-process is strictly better than the previous
// behaviour, which accepted an unlimited number of replays everywhere.
type clientAssertionReplayGuard struct {
	mu   sync.Mutex
	seen map[string]time.Time
}

var assertionReplayGuard = &clientAssertionReplayGuard{seen: map[string]time.Time{}}

// remember records jti and reports whether it was unused. A false return means
// the assertion is a replay and the caller must refuse it.
func (g *clientAssertionReplayGuard) remember(jti string, now time.Time) bool {
	if jti == "" {
		return false
	}

	g.mu.Lock()
	defer g.mu.Unlock()

	// Sweep here rather than on a timer: the map only grows on authentication, so
	// the write path is exactly where the expired entries are worth dropping.
	for seenJTI, expiresAt := range g.seen {
		if !expiresAt.After(now) {
			delete(g.seen, seenJTI)
		}
	}

	if expiresAt, ok := g.seen[jti]; ok && expiresAt.After(now) {
		return false
	}

	g.seen[jti] = now.Add(assertionReplayWindow)
	return true
}

// reset clears the guard. Test-only: package tests reuse jti values across cases.
	//nolint:unused
func (g *clientAssertionReplayGuard) reset() {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.seen = map[string]time.Time{}
}
