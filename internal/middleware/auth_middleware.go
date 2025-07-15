package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/maintainerd/auth/internal/util"
)

type contextKey string

const UserUUIDKey contextKey = "user_uuid"

func JWTAuthMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				http.Error(w, `{"error":"Authorization header missing"}`, http.StatusUnauthorized)
				return
			}

			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
				http.Error(w, `{"error":"Invalid Authorization header format"}`, http.StatusUnauthorized)
				return
			}

			tokenString := parts[1]

			claims, err := util.ValidateToken(tokenString)
			if err != nil {
				http.Error(w, `{"error":"Invalid or expired token"}`, http.StatusUnauthorized)
				return
			}

			// Add user_uuid to request context
			ctx := context.WithValue(r.Context(), UserUUIDKey, claims["user_uuid"])
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
