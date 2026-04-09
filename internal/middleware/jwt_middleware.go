package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/jwt"
	resp "github.com/maintainerd/auth/internal/rest/response"
)

// jwtKey is the unexported context key type for JWTClaims, preventing key
// collisions with other packages.
type jwtKey struct{}

// JWTClaims holds the parsed claims extracted from a validated JWT.
// It is stored once by JWTAuthMiddleware and retrieved by downstream
// middleware and handlers via JWTClaimsFromRequest.
type JWTClaims struct {
	Sub        string
	UserUUID   uuid.UUID
	Scope      string
	Audience   string
	Issuer     string
	JTI        string
	ClientID   string
	ProviderID string
}

// JWTClaimsFromRequest returns the JWTClaims stored in the request context
// by JWTAuthMiddleware, or nil if the middleware has not run.
func JWTClaimsFromRequest(r *http.Request) *JWTClaims {
	claims, _ := r.Context().Value(jwtKey{}).(*JWTClaims)
	return claims
}

// WithJWTClaims returns a shallow copy of r with the given JWTClaims stored
// in its context. It is intended for use in tests.
func WithJWTClaims(r *http.Request, claims *JWTClaims) *http.Request {
	return r.WithContext(context.WithValue(r.Context(), jwtKey{}, claims))
}

// JWTAuthMiddleware validates the Bearer token (or access_token cookie) and
// stores the parsed JWT claims in the request context for downstream use.
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
			resp.Error(w, http.StatusUnauthorized, "No valid authentication found")
			return
		}

		// Validate token
		rawClaims, err := jwt.ValidateToken(token)
		if err != nil {
			resp.Error(w, http.StatusUnauthorized, "Invalid or expired token", err.Error())
			return
		}

		// Extract subject — ValidateToken already guarantees sub is non-empty.
		sub, _ := rawClaims["sub"].(string)

		// Parse sub into a UUID.
		userUUID, err := uuid.Parse(sub)
		if err != nil {
			resp.Error(w, http.StatusBadRequest, "Invalid User UUID format", err.Error())
			return
		}

		scope, _ := rawClaims["scope"].(string)
		aud, _ := rawClaims["aud"].(string)
		iss, _ := rawClaims["iss"].(string)
		jti, _ := rawClaims["jti"].(string)
		clientID, _ := rawClaims["client_id"].(string)
		providerID, _ := rawClaims["provider_id"].(string)

		claims := &JWTClaims{
			Sub:        sub,
			UserUUID:   userUUID,
			Scope:      scope,
			Audience:   aud,
			Issuer:     iss,
			JTI:        jti,
			ClientID:   clientID,
			ProviderID: providerID,
		}

		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), jwtKey{}, claims)))
	})
}

// GetClientIDFromContext extracts the client_id from the JWT claims stored in
// the request context. Returns an empty string when claims are absent.
func GetClientIDFromContext(r *http.Request) string {
	if claims := JWTClaimsFromRequest(r); claims != nil {
		return claims.ClientID
	}
	return ""
}

// GetProviderIDFromContext extracts the provider_id from the JWT claims stored
// in the request context. Returns an empty string when claims are absent.
func GetProviderIDFromContext(r *http.Request) string {
	if claims := JWTClaimsFromRequest(r); claims != nil {
		return claims.ProviderID
	}
	return ""
}
