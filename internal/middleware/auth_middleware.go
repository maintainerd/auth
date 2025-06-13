package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/repository"
	"github.com/maintainerd/auth/internal/util"
)

const (
	UserUUIDKey    string = "user_uuid"
	UserContextKey string = "user"
)

func JWTAuthMiddleware(userRepo repository.UserRepository) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				util.Error(w, http.StatusUnauthorized, "Authorization header missing")
				return
			}

			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
				util.Error(w, http.StatusUnauthorized, "Invalid Authorization header format")
				return
			}

			tokenString := parts[1]
			claims, err := util.ValidateToken(tokenString)
			if err != nil {
				util.Error(w, http.StatusUnauthorized, "Invalid or expired token", err.Error())
				return
			}

			sub, ok := claims["sub"].(string)
			if !ok || sub == "" {
				util.Error(w, http.StatusUnauthorized, "Token missing subject (user_uuid)")
				return
			}

			userUUID, err := uuid.Parse(sub)
			if err != nil {
				util.Error(w, http.StatusBadRequest, "Invalid User UUID format", err.Error())
				return
			}

			user, err := userRepo.FindByUUID(userUUID)
			if err != nil {
				util.Error(w, http.StatusInternalServerError, "Failed to load user from database", err.Error())
				return
			}
			if user == nil {
				util.Error(w, http.StatusUnauthorized, "User not found")
				return
			}

			ctx := context.WithValue(r.Context(), UserUUIDKey, sub)
			ctx = context.WithValue(ctx, UserContextKey, user)

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
