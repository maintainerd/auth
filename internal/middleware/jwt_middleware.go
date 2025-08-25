package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/util"
)

type contextKey string

const (
	UserContextKey contextKey = "user"
	UserUUIDKey    contextKey = "user_uuid"
)

func JWTAuthMiddleware(next http.Handler) http.Handler {
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

		token := parts[1]
		claims, err := util.ValidateToken(token)
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

		ctx := context.WithValue(r.Context(), UserUUIDKey, userUUID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
