package middleware

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/maintainerd/maintainerd-auth/internal/platform/jwt"
	resp "github.com/maintainerd/maintainerd-auth/internal/platform/response"
)

// jwtKey is the unexported context key type for JWTClaims, preventing key
// collisions with other packages.
type jwtKey struct{}

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
	Iat         int64
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

func ContextWithJWTClaims(ctx context.Context, claims *JWTClaims) context.Context {
	return context.WithValue(ctx, jwtKey{}, claims)
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
		var scheme string

		if authHeader != "" {
			parts := strings.SplitN(authHeader, " ", 2)
			// "DPoP" is the scheme RFC 9449 §7.1 requires for a sender-constrained
			// token; it was previously unrecognized, so such a token could not be
			// used at all under its own scheme.
			if len(parts) == 2 {
				switch strings.ToLower(parts[0]) {
				case "bearer", "dpop":
					scheme = strings.ToLower(parts[0])
					token = parts[1]
				}
			}
		}
		if token == "" {
			for _, name := range []string{"access_token", "__Host-access_token"} {
				if cookie, err := r.Cookie(name); err == nil {
					token = cookie.Value
					scheme = "cookie"
					break
				}
			}
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

		// A token carrying cnf.jkt is bound to a key and is only valid alongside a
		// proof of possession of that key.
		if err := enforceDPoPBinding(r, scheme, token, rawClaims); err != nil {
			resp.Error(w, http.StatusUnauthorized, "Invalid token", err.Error())
			return
		}

		claims := buildJWTClaims(rawClaims)

		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), jwtKey{}, claims)))
	})
}

// buildJWTClaims maps validated raw token claims onto the typed JWTClaims used
// across the request lifecycle. Shared by the hard JWTAuthMiddleware and the
// session-aware OptionalUserContextMiddleware so the mapping lives in one place.
func buildJWTClaims(rawClaims map[string]any) *JWTClaims {
	// ValidateToken already guarantees sub is non-empty. Authn tokens often use
	// user UUIDs, while OAuth/OIDC tokens may use a pairwise subject; preserve
	// sub either way and populate UserUUID only when sub happens to be a UUID.
	sub, _ := rawClaims["sub"].(string)
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
	iat := numericDateClaim(rawClaims["iat"])

	return &JWTClaims{
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
		Iat:         iat,
	}
}

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

const defaultStepUpTTLSeconds = 300

// StepUpTTLReader resolves the tenant-specific step-up TTL in seconds.
// When set via SetStepUpTTLReader, RequireStepUp uses the tenant's
// mfa_config.step_up_ttl_minutes instead of the hardcoded default.
type StepUpTTLReader interface {
	StepUpTTLSecondsByTenant(ctx context.Context, tenantID int64) int64
}

var stepUpTTLReader StepUpTTLReader

// SetStepUpTTLReader installs a tenant-aware step-up TTL provider.
// Call once during app startup before serving requests.
func SetStepUpTTLReader(reader StepUpTTLReader) {
	stepUpTTLReader = reader
}

// RequireStepUp requires an elevated token with acr=2 issued within the
// tenant's configured step-up freshness window (mfa_config.step_up_ttl_minutes).
// Falls back to 300 seconds when no reader is set or tenant is unavailable.
// It must run after JWTAuthMiddleware so JWTClaims are already present in
// the request context.
func RequireStepUp(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		claims := JWTClaimsFromRequest(r)
		if claims == nil {
			resp.Error(w, http.StatusUnauthorized, "No valid authentication found")
			return
		}
		if claims.ACR != jwt.ACRLevel2 {
			resp.ErrorWithCode(w, http.StatusForbidden, "step_up_required", "Step-up authentication required")
			return
		}
		ttl := int64(defaultStepUpTTLSeconds)
		if stepUpTTLReader != nil {
			if auth := AuthFromRequest(r); auth != nil && auth.Tenant != nil {
				if tenantTTL := stepUpTTLReader.StepUpTTLSecondsByTenant(r.Context(), auth.Tenant.TenantID); tenantTTL > 0 {
					ttl = tenantTTL
				}
			}
		}
		if claims.Iat > 0 && time.Now().Unix()-claims.Iat > ttl {
			resp.ErrorWithCode(w, http.StatusForbidden, "step_up_required", "Step-up authentication has expired; please re-authenticate")
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

func numericDateClaim(v any) int64 {
	switch t := v.(type) {
	case float64:
		return int64(t)
	case int64:
		return t
	case int:
		return int64(t)
	}
	return 0
}
