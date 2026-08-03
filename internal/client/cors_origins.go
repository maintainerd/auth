package client

import (
	"context"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/maintainerd/maintainerd-auth/internal/shared"
	"gorm.io/gorm"
)

// corsOriginCacheTTL bounds how long a registration change takes to take
// effect. Short enough that an operator adding an origin in the admin console
// sees it work almost immediately, long enough that the token endpoint is not
// doing a DB round trip per preflight.
const corsOriginCacheTTL = 30 * time.Second

// CORSOriginResolver answers whether an Origin has been registered for
// cross-origin access by an active OAuth client.
//
// It exists because the client registry has always accepted a `cors_origin_uri`
// URI type — it is in the table's CHECK constraint, write validation accepts
// it, and the admin console lets an operator add one — but nothing ever read
// those rows at request time. CORS was decided solely by the CORS_ALLOWED_ORIGINS
// env var plus maintainerd's own console/identity hostnames, so a third-party
// SPA on its own domain could complete the redirect leg of authorization-code +
// PKCE (a top-level navigation, which CORS does not police) and then have its
// fetch() to /oauth/token blocked by the browser. The authorization code was
// valid and the exchange would have succeeded; the failure was entirely
// client-side and invisible in server logs. Registering the origin in the
// console did nothing — the only fix was editing a server env var and
// restarting, which defeats the point of self-service client registration.
type CORSOriginResolver struct {
	db *gorm.DB

	mu     sync.RWMutex
	cache  map[string]bool
	loaded time.Time
}

func NewCORSOriginResolver(db *gorm.DB) *CORSOriginResolver {
	return &CORSOriginResolver{db: db}
}

// IsAllowedCORSOrigin reports whether origin was registered as a
// `cors_origin_uri` by a client that is currently active. Fails CLOSED: any
// lookup error, or an unparseable origin, denies.
func (r *CORSOriginResolver) IsAllowedCORSOrigin(ctx context.Context, origin string) bool {
	normalized := normalizeOrigin(origin)
	if normalized == "" || r == nil || r.db == nil {
		return false
	}

	if origins, ok := r.cached(); ok {
		return origins[normalized]
	}
	return r.refresh(ctx)[normalized]
}

func (r *CORSOriginResolver) cached() (map[string]bool, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if r.cache == nil || time.Since(r.loaded) > corsOriginCacheTTL {
		return nil, false
	}
	return r.cache, true
}

// refresh reloads the whole allow-list in one query. The set is small (one row
// per registered origin per client) and this runs at most once per TTL, which
// is far cheaper than a per-request lookup keyed on the Origin header.
func (r *CORSOriginResolver) refresh(ctx context.Context) map[string]bool {
	var uris []string
	err := r.db.WithContext(ctx).
		Model(&ClientURI{}).
		Joins("JOIN clients ON clients.client_id = client_uris.client_id").
		Where("client_uris.type = ?", shared.ClientURITypeCORSOrigin).
		Where("client_uris.deleted_at IS NULL").
		Where("clients.status = ?", shared.StatusActive).
		Pluck("client_uris.uri", &uris).Error

	origins := make(map[string]bool, len(uris))
	if err != nil {
		// Fail closed, and do NOT cache the failure — a transient DB blip should
		// not blackhole CORS for the whole TTL.
		return origins
	}
	for _, u := range uris {
		if n := normalizeOrigin(u); n != "" {
			origins[n] = true
		}
	}

	r.mu.Lock()
	r.cache = origins
	r.loaded = time.Now()
	r.mu.Unlock()
	return origins
}

// normalizeOrigin reduces a URL to the scheme://host[:port] form a browser puts
// in the Origin header, so a registration stored with a trailing slash, a path,
// or mixed-case host still matches. Returns "" for anything that is not an
// absolute http(s) URL — notably the literal "null" origin, which sandboxed
// iframes and some file:// contexts send and which must never be allowed.
func normalizeOrigin(raw string) string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" || strings.EqualFold(trimmed, "null") {
		return ""
	}
	u, err := url.Parse(trimmed)
	if err != nil {
		return ""
	}
	scheme := strings.ToLower(u.Scheme)
	if scheme != "http" && scheme != "https" {
		return ""
	}
	if u.Host == "" {
		return ""
	}
	return scheme + "://" + strings.ToLower(u.Host)
}
