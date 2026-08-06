package client

import (
	"context"
	"log/slog"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/maintainerd/maintainerd-auth/internal/platform/middleware"
	"github.com/maintainerd/maintainerd-auth/internal/shared"
	"gorm.io/gorm"
)

// corsOriginCacheTTL bounds how long a registration change takes to take
// effect. Short enough that an operator adding an origin in the admin console
// sees it work almost immediately, long enough that the token endpoint is not
// doing a DB round trip per preflight.
const corsOriginCacheTTL = 30 * time.Second

// systemTenantScope is the cache key for origins registered under the system
// tenant. A literal that cannot be a DNS label, so it can never collide with a
// tenant slug.
const systemTenantScope = "\x00system"

// CORSOriginResolver answers whether an Origin has been registered for
// cross-origin access by an active OAuth client OF THE TENANT THE REQUEST IS ON.
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
//
// The allow-list is scoped PER TENANT, not global. A match makes the CORS
// middleware emit Access-Control-Allow-Credentials: true, so a listed origin can
// read cookie-authenticated responses (profile, sessions, MFA state) — and GETs
// are exempt from CSRF, so nothing else stands in the way. A flat allow-list
// therefore meant any tenant could register an origin and have it read every
// OTHER tenant's users' responses: self-service client registration became a
// cross-tenant read primitive. The tenant that registered an origin is now the
// only tenant whose surface that origin may talk to.
type CORSOriginResolver struct {
	db *gorm.DB

	mu sync.RWMutex
	// cache maps a tenant scope (slug, or systemTenantScope) to that tenant's
	// registered origins.
	cache  map[string]map[string]bool
	loaded time.Time
}

func NewCORSOriginResolver(db *gorm.DB) *CORSOriginResolver {
	return &CORSOriginResolver{db: db}
}

// IsAllowedCORSOrigin reports whether origin was registered as a
// `cors_origin_uri` by a currently-active client belonging to the tenant this
// request is on. Fails CLOSED: an unparseable origin, a request whose tenant
// cannot be determined from its own host, or any lookup error denies.
func (r *CORSOriginResolver) IsAllowedCORSOrigin(ctx context.Context, origin string) bool {
	normalized := normalizeOrigin(origin)
	if normalized == "" || r == nil || r.db == nil {
		return false
	}

	scope, ok := requestTenantScope(ctx)
	if !ok {
		// Fail CLOSED. Granting a credentialed origin is a per-tenant decision and
		// an unrecognized host gives no tenant to make it for. Operator-owned
		// origins are still served by CORS_ALLOWED_ORIGINS, which needs no tenant.
		return false
	}

	if scoped, fresh := r.cached(); fresh {
		return scoped[scope][normalized]
	}
	return r.refresh(ctx)[scope][normalized]
}

// requestTenantScope derives the cache key for the tenant whose surface this
// request arrived on. The tenant comes from the REQUEST's own host — never from
// a caller-supplied id — which is what makes the scoping unforgeable.
func requestTenantScope(ctx context.Context) (string, bool) {
	rt, ok := middleware.RequestTenantFromContext(ctx)
	if !ok {
		// NOTHING in the context — as opposed to a RequestTenant with OK=false —
		// means middleware.RequestTenantMiddleware never ran ahead of CORS on this
		// route, not that the host is unrecognized. That misconfiguration fails
		// closed so quietly that it is indistinguishable from "the operator typed
		// the origin wrong": every registered cors_origin_uri stops matching and
		// nothing is logged. Say so, once.
		warnRequestTenantMiddlewareMissing()
		return "", false
	}
	if !rt.OK {
		return "", false
	}
	if rt.IsSystem {
		return systemTenantScope, true
	}
	slug := strings.ToLower(strings.TrimSpace(rt.Slug))
	if slug == "" {
		return "", false
	}
	return slug, true
}

// requestTenantMiddlewareWarning keeps the mount warning to one line per
// process; CORS runs on every cross-origin request and this must not become a
// log flood.
var requestTenantMiddlewareWarning sync.Once

func warnRequestTenantMiddlewareMissing() {
	requestTenantMiddlewareWarning.Do(func() {
		slog.Warn("operator-registered CORS origins (cors_origin_uri) are being ignored: " +
			"the allow-list is scoped to the request's tenant, but middleware.RequestTenantMiddleware " +
			"has not run ahead of CORSMiddleware on this route, so there is no tenant to scope to and " +
			"every registered origin is denied. Mount RequestTenantMiddleware before CORSMiddleware.")
	})
}

func (r *CORSOriginResolver) cached() (map[string]map[string]bool, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if r.cache == nil || time.Since(r.loaded) > corsOriginCacheTTL {
		return nil, false
	}
	return r.cache, true
}

// corsOriginRow is one registered origin plus the tenant that owns it.
type corsOriginRow struct {
	TenantName string
	IsSystem   bool
	URI        string
}

// refresh reloads the whole allow-list in one query, grouped by tenant. The set
// is small (one row per registered origin per client) and this runs at most once
// per TTL, which is far cheaper than a per-request lookup keyed on the Origin
// header.
func (r *CORSOriginResolver) refresh(ctx context.Context) map[string]map[string]bool {
	var rows []corsOriginRow
	err := r.db.WithContext(ctx).
		Model(&ClientURI{}).
		Select("tenants.name AS tenant_name, tenants.is_system AS is_system, client_uris.uri AS uri").
		Joins("JOIN clients ON clients.client_id = client_uris.client_id").
		Joins("JOIN tenants ON tenants.tenant_id = clients.tenant_id").
		Where("client_uris.type = ?", shared.ClientURITypeCORSOrigin).
		Where("client_uris.deleted_at IS NULL").
		// The clients side is joined with a raw string, so GORM's soft-delete scope
		// covers client_uris only. Without this predicate a soft-deleted client's
		// origin stays in the allow-list; that it currently does not is incidental
		// (DeleteByUUID soft-deletes the ClientURI rows first), not a guarantee.
		Where("clients.deleted_at IS NULL").
		Where("clients.status = ?", shared.StatusActive).
		// Same for the tenant: a suspended or deleted tenant must not keep handing
		// out credentialed cross-origin access on a surface that still resolves.
		Where("tenants.deleted_at IS NULL").
		Where("tenants.status = ?", shared.StatusActive).
		Scan(&rows).Error

	scoped := make(map[string]map[string]bool)
	if err != nil {
		// Fail closed, and do NOT cache the failure — a transient DB blip should
		// not blackhole CORS for the whole TTL.
		return scoped
	}
	for _, row := range rows {
		normalized := normalizeOrigin(row.URI)
		if normalized == "" {
			continue
		}
		for _, scope := range tenantScopes(row) {
			if scoped[scope] == nil {
				scoped[scope] = make(map[string]bool)
			}
			scoped[scope][normalized] = true
		}
	}

	r.mu.Lock()
	r.cache = scoped
	r.loaded = time.Now()
	r.mu.Unlock()
	return scoped
}

// tenantScopes lists every cache key under which a tenant's origins must be
// indexed, i.e. every request scope requestTenantScope can return for a host
// that belongs to this tenant.
//
// `tenants.name` IS the DNS label: shared.ResolveTenantHost strips it off the
// request host and hands it back as RequestTenant.Slug, and the authorize
// endpoint binds it with ResolveTenantIDByName(rt.Slug). Both sides of this
// lookup therefore key on the same lowercased value.
//
// The system tenant is the exception, and it has to be indexed under BOTH keys.
// It answers on the bare base host (shared.ResolveTenantHost → IsSystem, no
// slug) AND — like any other tenant — on {its name}.<base>, which resolves to a
// plain slug. Indexing it only under systemTenantScope silently dropped every
// origin it registered whenever the browser was on its named subdomain, which is
// the surface the console actually links to.
func tenantScopes(row corsOriginRow) []string {
	name := strings.ToLower(strings.TrimSpace(row.TenantName))
	if !row.IsSystem {
		if name == "" {
			return nil
		}
		return []string{name}
	}
	scopes := []string{systemTenantScope}
	if name != "" {
		scopes = append(scopes, name)
	}
	return scopes
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
