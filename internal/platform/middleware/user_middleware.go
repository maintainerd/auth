package middleware

import (
	"context"
	"net/http"

	"github.com/maintainerd/maintainerd-auth/internal/authctx"
	"github.com/maintainerd/maintainerd-auth/internal/platform/cache"
	resp "github.com/maintainerd/maintainerd-auth/internal/platform/response"
	"github.com/maintainerd/maintainerd-auth/internal/shared"
)

// UserContextProvider is the minimal interface required by UserContextMiddleware
// to resolve an authenticated subject from a JWT sub claim and client ID. It
// returns the full UserContext (user plus the tenant, identity provider, and
// client tied to the identity that authenticated via clientID) so the
// middleware can populate the request AuthContext. This is intentionally narrow
// so the middleware does not depend on a raw repository or the full
// UserService interface.
type UserContextProvider interface {
	FindBySubAndClientID(ctx context.Context, sub string, clientID string) (*authctx.UserContext, error)
}

// authKey is the unexported context key type for AuthContext, preventing key
// collisions with other packages.
type authKey struct{}

// AuthFromRequest returns the AuthContext stored in the request context by
// UserContextMiddleware. It never returns nil — fields inside the struct may
// be nil when the middleware has not populated them.
// AuthFromContext returns the authenticated principal from a bare context.
// Transports that do not carry an *http.Request — gRPC, and services handed
// only ctx — need the same authority as AuthFromRequest.
func AuthFromContext(ctx context.Context) *authctx.AuthContext {
	if auth, ok := ctx.Value(authKey{}).(*authctx.AuthContext); ok {
		return auth
	}
	return &authctx.AuthContext{}
}

// WithAuthContextValue stores an AuthContext on a bare context. Used by gRPC
// interceptors and tests, which have no *http.Request to hang it off.
func WithAuthContextValue(ctx context.Context, auth *authctx.AuthContext) context.Context {
	return context.WithValue(ctx, authKey{}, auth)
}

func AuthFromRequest(r *http.Request) *authctx.AuthContext {
	if auth, ok := r.Context().Value(authKey{}).(*authctx.AuthContext); ok {
		return auth
	}
	return &authctx.AuthContext{}
}

// WithAuthContext returns a shallow copy of r with the given AuthContext stored
// in its context. It is intended for use in tests.
func WithAuthContext(r *http.Request, auth *authctx.AuthContext) *http.Request {
	return r.WithContext(context.WithValue(r.Context(), authKey{}, auth))
}

// UserContextMiddleware resolves the authenticated user, tenant, provider, and
// client from the JWT claims already stored by JWTAuthMiddleware, populates an
// AuthContext, and stores it in the request context for downstream handlers.
func UserContextMiddleware(
	userProvider UserContextProvider,
	appCache *cache.Cache,
) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if auth := AuthFromRequest(r); auth.User != nil {
				if !ValidateSessionFromRequest(w, r) {
					return
				}
				next.ServeHTTP(w, r)
				return
			}

			var sub, clientID string
			if c := JWTClaimsFromRequest(r); c != nil {
				sub, clientID = c.Sub, c.ClientID
			}

			ctx := r.Context()

			// Try cache first
			if uc := appCache.GetUserContext(ctx, sub, clientID); uc != nil {
				auth := &authctx.AuthContext{
					User:     uc.User,
					Tenant:   uc.Tenant,
					Provider: uc.Provider,
					Client:   uc.Client,
				}
				if !userStatusGrantsAccess(w, auth.User) {
					return
				}
				req := r.WithContext(context.WithValue(ctx, authKey{}, auth))
				if !ValidateSessionFromRequest(w, req) {
					return
				}
				next.ServeHTTP(w, req)
				return
			}

			// Cache miss — load from database. The provider resolves the user
			// along with the tenant, identity provider, and client tied to the
			// identity that authenticated via clientID.
			uc, err := userProvider.FindBySubAndClientID(ctx, sub, clientID)
			if err != nil {
				resp.Error(w, http.StatusInternalServerError, "Failed to load user from database")
				return
			}
			if uc == nil || uc.User == nil {
				resp.Error(w, http.StatusUnauthorized, "User not found")
				return
			}

			if !userStatusGrantsAccess(w, uc.User) {
				return
			}

			// Write through to cache
			appCache.SetUserContext(ctx, sub, clientID, uc)

			auth := &authctx.AuthContext{
				User:     uc.User,
				Tenant:   uc.Tenant,
				Provider: uc.Provider,
				Client:   uc.Client,
			}
			req := r.WithContext(context.WithValue(ctx, authKey{}, auth))
			if !ValidateSessionFromRequest(w, req) {
				return
			}
			next.ServeHTTP(w, req)
		})
	}
}

// userStatusGrantsAccess refuses a request whose subject is no longer an active
// account.
//
// Deactivating, suspending or soft-deleting a user has to take effect on the
// next request. Without this it only took effect when the access token happened
// to expire, so a disabled account stayed fully usable for the remainder of its
// token lifetime — and the same applied to a cached user context, which is why
// both the cache-hit and database branches are gated.
//
// An empty status is treated as active: several projections legitimately do not
// select the column, and failing closed there would lock every user out rather
// than deny one.
func userStatusGrantsAccess(w http.ResponseWriter, u *authctx.AuthUser) bool {
	if u == nil || u.Status == "" || u.Status == shared.StatusActive {
		return true
	}
	resp.Error(w, http.StatusUnauthorized, "This account is no longer active")
	return false
}
