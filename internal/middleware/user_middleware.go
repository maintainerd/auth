package middleware

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/maintainerd/auth/internal/model"
	resp "github.com/maintainerd/auth/internal/rest/response"
	"github.com/redis/go-redis/v9"
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
	redisClient *redis.Client,
) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Get sub and client_id from JWT context
			sub, _ := r.Context().Value(SubKey).(string)
			clientID, _ := r.Context().Value(ClientIDKey).(string)

			// Create cache key
			cacheKey := "user:" + sub + ":" + clientID

			// Use the request context so Redis calls respect cancellation/timeouts
			ctx := r.Context()

			// Define cache structure for user context
			type UserContextCache struct {
				User     *model.User             `json:"user"`
				Tenant   *model.Tenant           `json:"tenant"`
				Provider *model.IdentityProvider `json:"provider"`
				Client   *model.Client           `json:"client"`
			}

			var userContextCache *UserContextCache

			// Check and set auth information from cache
			cachedData, err := redisClient.Get(ctx, cacheKey).Result()
			if err == nil {
				if err := json.Unmarshal([]byte(cachedData), &userContextCache); err == nil {
					// Set all context information
					reqCtx := context.WithValue(r.Context(), UserContextKey, userContextCache.User)
					reqCtx = context.WithValue(reqCtx, TenantContextKey, userContextCache.Tenant)
					reqCtx = context.WithValue(reqCtx, ProviderContextKey, userContextCache.Provider)
					reqCtx = context.WithValue(reqCtx, ClientContextKey, userContextCache.Client)
					next.ServeHTTP(w, r.WithContext(reqCtx))
					return
				}
			}

			// Get auth information from database
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

			// Create cache structure
			userContextCache = &UserContextCache{
				User:     user,
				Tenant:   tenant,
				Provider: provider,
				Client:   client,
			}

			// Cache user context for 10 minutes
			if jsonData, err := json.Marshal(userContextCache); err == nil {
				_ = redisClient.Set(ctx, cacheKey, jsonData, 10*time.Minute).Err()
			}

			// Set all context information
			reqCtx := context.WithValue(r.Context(), UserContextKey, user)
			reqCtx = context.WithValue(reqCtx, TenantContextKey, tenant)
			reqCtx = context.WithValue(reqCtx, ProviderContextKey, provider)
			reqCtx = context.WithValue(reqCtx, ClientContextKey, client)
			next.ServeHTTP(w, r.WithContext(reqCtx))
		})
	}
}
