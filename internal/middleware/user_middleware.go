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
	FindBySubAndClientID(sub string, clientID string) (*model.User, error)
}

// Context keys for accessing user-related information
type userContextKey string

const (
	UserContextKey     userContextKey = "user"
	TenantContextKey   userContextKey = "tenant"
	ProviderContextKey userContextKey = "provider"
	ClientContextKey   userContextKey = "client"
)

func UserContextMiddleware(
	userProvider UserContextProvider,
	appCache *cache.Cache,
) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Get sub and client_id from JWT context
			sub, _ := r.Context().Value(SubKey).(string)
			clientID, _ := r.Context().Value(ClientIDKey).(string)

			ctx := r.Context()

			// Try cache first
			if uc := appCache.GetUserContext(ctx, sub, clientID); uc != nil {
				reqCtx := context.WithValue(r.Context(), UserContextKey, uc.User)
				reqCtx = context.WithValue(reqCtx, TenantContextKey, uc.Tenant)
				reqCtx = context.WithValue(reqCtx, ProviderContextKey, uc.Provider)
				reqCtx = context.WithValue(reqCtx, ClientContextKey, uc.Client)
				next.ServeHTTP(w, r.WithContext(reqCtx))
				return
			}

			// Cache miss — load from database
			user, err := userProvider.FindBySubAndClientID(sub, clientID)
			if err != nil {
				resp.Error(w, http.StatusInternalServerError, "Failed to load user from database")
				return
			}
			if user == nil {
				resp.Error(w, http.StatusUnauthorized, "User not found")
				return
			}

			// Extract tenant, provider, and client information from user relationships
			var tenant *model.Tenant
			var provider *model.IdentityProvider
			var client *model.Client

			// Get tenant, provider and client from user identities
			if len(user.UserIdentities) > 0 {
				for _, identity := range user.UserIdentities {
					if identity.Client != nil && identity.Client.Identifier != nil && *identity.Client.Identifier == clientID {
						// Get tenant from this identity
						if identity.Tenant != nil {
							tenant = identity.Tenant
						}
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

			// Set all context information
			reqCtx := context.WithValue(r.Context(), UserContextKey, user)
			reqCtx = context.WithValue(reqCtx, TenantContextKey, tenant)
			reqCtx = context.WithValue(reqCtx, ProviderContextKey, provider)
			reqCtx = context.WithValue(reqCtx, ClientContextKey, client)
			next.ServeHTTP(w, r.WithContext(reqCtx))
		})
	}
}
