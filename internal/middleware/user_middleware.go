package middleware

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/maintainerd/auth/internal/model"
	"github.com/maintainerd/auth/internal/repository"
	"github.com/maintainerd/auth/internal/util"
	"github.com/redis/go-redis/v9"
)

func UserContextMiddleware(
	userRepo repository.UserRepository,
	redisClient *redis.Client,
) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Get sub and client_id
			sub, _ := r.Context().Value(SubKey).(string)
			clientID, _ := r.Context().Value(ClientIDKey).(string)

			// Validate sub and client id
			if sub == "" || clientID == "" {
				util.Error(w, http.StatusUnauthorized, "Missing sub or client_id in token")
				return
			}

			// Create cache key
			cacheKey := "user:" + sub

			// Initialize context
			ctx := context.Background()

			var user *model.User

			// Check and set auth information from cache
			cachedUser, err := redisClient.Get(ctx, cacheKey).Result()
			if err == nil {
				if err := json.Unmarshal([]byte(cachedUser), &user); err == nil {
					// Set auth information to context
					reqCtx := context.WithValue(r.Context(), UserContextKey, user)
					next.ServeHTTP(w, r.WithContext(reqCtx))
					return
				}
			}

			// Get auth information from database
			user, err = userRepo.FindBySubAndClientID(sub, clientID)
			if err != nil {
				util.Error(w, http.StatusInternalServerError, "Failed to load user from database", err.Error())
				return
			}
			if user == nil {
				util.Error(w, http.StatusUnauthorized, "User not found")
				return
			}

			// Cache user for 10 minutes
			if jsonData, err := json.Marshal(user); err == nil {
				_ = redisClient.Set(ctx, cacheKey, jsonData, 10*time.Minute).Err()
			}

			// Set in context
			reqCtx := context.WithValue(r.Context(), UserContextKey, user)
			next.ServeHTTP(w, r.WithContext(reqCtx))
		})
	}
}
