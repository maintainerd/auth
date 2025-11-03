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
	UserUUIDKey contextKey = "user_uuid"
	SubKey      contextKey = "sub"

	// standard jwt fields
	ScopeKey    contextKey = "scope"
	AudienceKey contextKey = "audience"
	IssuerKey   contextKey = "issuer"
	JTIKey      contextKey = "jti"
)

func JWTAuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Get authorization header first
		authHeader := r.Header.Get("Authorization")
		var token string

		if authHeader != "" {
			// Use Bearer token if present
			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) == 2 && strings.ToLower(parts[0]) == "bearer" {
				token = parts[1]
			}
		} else {
			// Fallback to cookie if no Authorization header
			if cookie, err := r.Cookie("access_token"); err == nil {
				token = cookie.Value
			}
		}

		if token == "" {
			util.Error(w, http.StatusUnauthorized, "No valid authentication found")
			return
		}

		// Validate token
		claims, err := util.ValidateToken(token)
		if err != nil {
			util.Error(w, http.StatusUnauthorized, "Invalid or expired token", err.Error())
			return
		}

		// Extract subject (user_uuid)
		sub, ok := claims["sub"].(string)
		if !ok || sub == "" {
			util.Error(w, http.StatusUnauthorized, "Token missing subject (user_uuid)")
			return
		}

		// Parse sub into uuid
		userUUID, err := uuid.Parse(sub)
		if err != nil {
			util.Error(w, http.StatusBadRequest, "Invalid User UUID format", err.Error())
			return
		}

		// Standard JWT fields
		scope, _ := claims["scope"].(string)
		aud, _ := claims["aud"].(string)
		iss, _ := claims["iss"].(string)
		jti, _ := claims["jti"].(string)

		// Build new context with all needed values
		ctx := context.WithValue(r.Context(), SubKey, sub)
		ctx = context.WithValue(ctx, ScopeKey, scope)
		ctx = context.WithValue(ctx, AudienceKey, aud)
		ctx = context.WithValue(ctx, IssuerKey, iss)
		ctx = context.WithValue(ctx, JTIKey, jti)
		ctx = context.WithValue(ctx, UserUUIDKey, userUUID)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
