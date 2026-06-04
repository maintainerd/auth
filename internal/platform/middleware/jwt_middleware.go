package middleware

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/authctx"
	"github.com/maintainerd/auth/internal/platform/jwt"
	resp "github.com/maintainerd/auth/internal/platform/response"
)

// jwtKey is the unexported context key type for JWTClaims, preventing key
// collisions with other packages.
type jwtKey struct{}

type APIKeyAuthenticator interface {
	AuthenticateAPIKey(ctx context.Context, rawKey string) (*authctx.AuthContext, error)
}

var apiKeyAuthenticator APIKeyAuthenticator

func SetAPIKeyAuthenticator(authenticator APIKeyAuthenticator) {
	apiKeyAuthenticator = authenticator
}

// JWTClaims holds the parsed claims extracted from a validated JWT.
// It is stored once by JWTAuthMiddleware and retrieved by downstream
// middleware and handlers via JWTClaimsFromRequest.
type JWTClaims struct {
	Sub         string
	UserUUID    uuid.UUID
	Service     string
	SubjectType string
	Scope       string
	Audience    string
	Issuer      string
	JTI         string
	ClientID    string
	ProviderID  string
	SessionID   string
	AMR         []string
	ACR         string
}

// JWTClaimsFromRequest returns the JWTClaims stored in the request context
// by JWTAuthMiddleware, or nil if the middleware has not run.
func JWTClaimsFromRequest(r *http.Request) *JWTClaims {
	claims, _ := r.Context().Value(jwtKey{}).(*JWTClaims)
	return claims
}

func JWTClaimsFromContext(ctx context.Context) *JWTClaims {
	claims, _ := ctx.Value(jwtKey{}).(*JWTClaims)
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
		var apiKey string

		if authHeader != "" {
			// Use Bearer token if present
			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) == 2 && strings.ToLower(parts[0]) == "bearer" {
				token = parts[1]
			}
		}
		if strings.HasPrefix(token, "ak_") {
			apiKey = token
			token = ""
		}
		if apiKey == "" {
			apiKey = strings.TrimSpace(r.Header.Get("X-API-Key"))
		}
		if apiKey == "" && token == "" {
			// Fallback to cookie if no Authorization header
			if cookie, err := r.Cookie("access_token"); err == nil {
				token = cookie.Value
			}
		}

		if apiKey != "" {
			if apiKeyAuthenticator == nil {
				resp.Error(w, http.StatusUnauthorized, "API key authentication is not configured")
				return
			}
			auth, err := apiKeyAuthenticator.AuthenticateAPIKey(r.Context(), apiKey)
			if err != nil {
				status := http.StatusUnauthorized
				if errors.Is(err, errAPIKeyForbidden) {
					status = http.StatusForbidden
				}
				resp.Error(w, status, err.Error())
				return
			}
			next.ServeHTTP(w, WithAuthContext(r, auth))
			return
		}

		if token == "" {
			resp.Error(w, http.StatusUnauthorized, "No valid authentication found")
			return
		}

		// Validate token
		rawClaims, err := jwt.ValidateTokenWithContext(r.Context(), token)
		if err != nil {
			resp.Error(w, http.StatusUnauthorized, "Invalid or expired token", err.Error())
			return
		}

		// Extract subject — ValidateToken already guarantees sub is non-empty.
		sub, _ := rawClaims["sub"].(string)

		// Authn tokens often use user UUIDs, while OAuth/OIDC tokens may use a
		// pairwise subject. Preserve sub either way and populate UserUUID only
		// when the subject happens to be a UUID.
		userUUID, _ := uuid.Parse(sub)

		scope, _ := rawClaims["scope"].(string)
		aud, _ := rawClaims["aud"].(string)
		iss, _ := rawClaims["iss"].(string)
		jti, _ := rawClaims["jti"].(string)
		clientID, _ := rawClaims["client_id"].(string)
		providerID, _ := rawClaims["provider_id"].(string)
		sessionID, _ := rawClaims["sid"].(string)
		service, _ := rawClaims["svc"].(string)
		subjectType, _ := rawClaims["sub_type"].(string)
		acr, _ := rawClaims["acr"].(string)
		amr := stringSliceClaim(rawClaims["amr"])

		claims := &JWTClaims{
			Sub:         sub,
			UserUUID:    userUUID,
			Service:     service,
			SubjectType: subjectType,
			Scope:       scope,
			Audience:    aud,
			Issuer:      iss,
			JTI:         jti,
			ClientID:    clientID,
			ProviderID:  providerID,
			SessionID:   sessionID,
			AMR:         amr,
			ACR:         acr,
		}

		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), jwtKey{}, claims)))
	})
}

var errAPIKeyForbidden = errors.New("API key is not allowed")

func stringSliceClaim(raw any) []string {
	values, ok := raw.([]any)
	if !ok {
		return nil
	}
	out := make([]string, 0, len(values))
	for _, value := range values {
		if s, ok := value.(string); ok && s != "" {
			out = append(out, s)
		}
	}
	return out
}

// RequireStepUp requires an elevated token with acr=2. It must run after
// JWTAuthMiddleware so JWTClaims are already present in the request context.
func RequireStepUp(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		claims := JWTClaimsFromRequest(r)
		if claims == nil {
			resp.Error(w, http.StatusUnauthorized, "No valid authentication found")
			return
		}
		if claims.ACR != jwt.ACRLevel2 {
			resp.Error(w, http.StatusForbidden, "Step-up authentication required")
			return
		}
		next.ServeHTTP(w, r)
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
