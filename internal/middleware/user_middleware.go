package middleware

import (
	"context"
	"net/http"

	"github.com/maintainerd/auth/internal/cache"
	"github.com/maintainerd/auth/internal/model"
	resp "github.com/maintainerd/auth/internal/rest/response"
)

// UserContextProvider is the minimal interface required by UserContextMiddleware
// to resolve a user from a JWT sub claim and client ID. This is intentionally
// narrow so the middleware does not depend on a raw repository or the full
// UserService interface.
type UserContextProvider interface {
	FindBySubAndClientID(ctx context.Context, sub string, clientID string) (*model.User, error)
}

// authKey is the unexported context key type for AuthContext, preventing key
// collisions with other packages.
type authKey struct{}

// AuthContext holds the authenticated principal and their associated tenant,
// identity provider, and client. It is set once by UserContextMiddleware and
// retrieved by downstream middleware and handlers via AuthFromRequest.
type AuthContext struct {
	User     *model.User
	Tenant   *model.Tenant
	Provider *model.IdentityProvider
	Client   *model.Client
}

// AuthFromRequest returns the AuthContext stored in the request context by
// UserContextMiddleware. It never returns nil — fields inside the struct may
// be nil when the middleware has not populated them.
func AuthFromRequest(r *http.Request) *AuthContext {
	if auth, ok := r.Context().Value(authKey{}).(*AuthContext); ok {
		return auth
	}
	return &AuthContext{}
}

// WithAuthContext returns a shallow copy of r with the given AuthContext stored
// in its context. It is intended for use in tests.
func WithAuthContext(r *http.Request, auth *AuthContext) *http.Request {
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
			var sub, clientID string
			if c := JWTClaimsFromRequest(r); c != nil {
				sub, clientID = c.Sub, c.ClientID
			}

			ctx := r.Context()

			// Try cache first
			if uc := appCache.GetUserContext(ctx, sub, clientID); uc != nil {
				auth := &AuthContext{
					User:     uc.User,
					Tenant:   uc.Tenant,
					Provider: uc.Provider,
					Client:   uc.Client,
				}
				next.ServeHTTP(w, r.WithContext(context.WithValue(ctx, authKey{}, auth)))
				return
			}

			// Cache miss — load from database
			user, err := userProvider.FindBySubAndClientID(ctx, sub, clientID)
			if err != nil {
				resp.Error(w, http.StatusInternalServerError, "Failed to load user from database")
				return
			}
			if user == nil {
				resp.Error(w, http.StatusUnauthorized, "User not found")
				return
			}

			// Extract tenant from user identities matching the current client.
			var tenant *model.Tenant
			var provider *model.IdentityProvider
			var client *model.Client

			for _, identity := range user.UserIdentities {
				if identity.Client != nil && identity.Client.Identifier != nil && *identity.Client.Identifier == clientID {
					if identity.Tenant != nil {
						tenant = identity.Tenant
					}
				}
			}

			// Write through to cache
			appCache.SetUserContext(ctx, sub, clientID, &cache.UserContext{
				User:     user,
				Tenant:   tenant,
				Provider: provider,
				Client:   client,
			})

			auth := &AuthContext{
				User:     user,
				Tenant:   tenant,
				Provider: provider,
				Client:   client,
			}
			next.ServeHTTP(w, r.WithContext(context.WithValue(ctx, authKey{}, auth)))
		})
	}
}
