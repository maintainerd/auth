package middleware

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/google/uuid"
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
			// Get user UUID from context (set by JWTAuthMiddleware)
			val := r.Context().Value(UserUUIDKey)
			if val == nil {
				util.Error(w, http.StatusUnauthorized, "Missing user UUID in context")
				return
			}

			userUUID, ok := val.(uuid.UUID)
			if !ok {
				util.Error(w, http.StatusInternalServerError, "Invalid user UUID in context")
				return
			}

			cacheKey := "user:" + userUUID.String()
			ctx := context.Background()

			var user *model.User

			// Check Redis cache
			cachedUser, err := redisClient.Get(ctx, cacheKey).Result()
			if err == nil {
				if err := json.Unmarshal([]byte(cachedUser), &user); err == nil {
					// Cache hit and unmarshalled successfully
					reqCtx := context.WithValue(r.Context(), UserContextKey, user)
					next.ServeHTTP(w, r.WithContext(reqCtx))
					return
				}
			}

			// Load from database
			user, err = userRepo.FindByUUID(userUUID)
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
