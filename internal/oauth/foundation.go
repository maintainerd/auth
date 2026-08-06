package oauth

import (
	"net/http"
	"strings"
	"time"

	"github.com/lib/pq"
	"github.com/maintainerd/maintainerd-auth/internal/authctx"
	"github.com/maintainerd/maintainerd-auth/internal/platform/apperror"
	"github.com/maintainerd/maintainerd-auth/internal/platform/database"
	"github.com/maintainerd/maintainerd-auth/internal/platform/pagination"
	"github.com/maintainerd/maintainerd-auth/internal/platform/security"
)

type BaseRepository[T any] = database.BaseRepository[T]
type BaseRepositoryMethods[T any] = database.BaseRepositoryMethods[T]
type PaginationResult[T any] = database.PaginationResult[T]

type PaginationRequestDTO = pagination.PaginationRequestDTO
type PaginatedResponseDTO[T any] = pagination.PaginatedResponseDTO[T]
type SuccessResponseDTO = pagination.SuccessResponseDTO

const (
	SortOrderAsc  = pagination.SortOrderAsc
	SortOrderDesc = pagination.SortOrderDesc
)

// ---------------------------------------------------------------------------
// Type aliases — cache auth types stored in authctx.AuthContext
// ---------------------------------------------------------------------------

// User and Profile are aliases for the cache auth types so that handlers can
// inject rich user data into the auth context and read it back.
type User = authctx.AuthUser
type Profile = authctx.AuthProfile

// ---------------------------------------------------------------------------
// OAuth constants (defined locally to avoid importing the client package)
// ---------------------------------------------------------------------------

const (
	GrantTypeAuthorizationCode = "authorization_code"
	GrantTypeClientCredentials = "client_credentials"
	GrantTypeRefreshToken      = "refresh_token"

	GrantTypeDeviceCode    = "urn:ietf:params:oauth:grant-type:device_code"
	GrantTypeTokenExchange = "urn:ietf:params:oauth:grant-type:token-exchange"
	GrantTypeCIBA          = "urn:ietf:params:oauth:grant-type:ciba"

	TokenAuthMethodSecretBasic = "client_secret_basic"
	TokenAuthMethodSecretPost  = "client_secret_post"
	TokenAuthMethodNone        = "none"

	TokenAuthMethodPrivateKeyJWT   = "private_key_jwt"
	TokenAuthMethodClientSecretJWT = "client_secret_jwt"
	// Accepted by the client registry and its CHECK constraint, but there is no
	// certificate-binding implementation behind them yet.
	TokenAuthMethodTLSClientAuth           = "tls_client_auth"
	TokenAuthMethodSelfSignedTLSClientAuth = "self_signed_tls_client_auth"

	ResponseTypeCode = "code"
)

// Subject-type labels (the `sub_type` claim) for grants that produce a user
// token with no browser session behind it. Session validation uses them to tell
// "legitimately sessionless" apart from "unbound, and therefore unrevokable" —
// see middleware.sessionlessSubjectTypes.
const (
	subjectTypeDevice   = "device"   // RFC 8628 device authorization grant
	subjectTypeCIBA     = "ciba"     // OIDC CIBA
	subjectTypeExchange = "exchange" // RFC 8693 token exchange
)

// ---------------------------------------------------------------------------
// Client credentials
// ---------------------------------------------------------------------------

// OAuthClientCredentials holds the resolved client_id and client_secret from
// either the Authorization header (Basic auth) or the POST body.
type OAuthClientCredentials struct {
	ClientID            string
	ClientSecret        string
	ClientAssertionType string
	ClientAssertion     string
}

// extractOAuthClientCredentials resolves client_id / client_secret from either
// HTTP Basic auth (RFC 6749 §2.3.1) or the request body form fields.
func extractOAuthClientCredentials(r *http.Request, clientID, clientSecret string) OAuthClientCredentials {
	if id, secret, ok := r.BasicAuth(); ok {
		return OAuthClientCredentials{ClientID: id, ClientSecret: secret}
	}
	return OAuthClientCredentials{ClientID: clientID, ClientSecret: clientSecret}
}

func clientSecretMatches(client *Client, plaintext string) bool {
	if plaintext == "" {
		return false
	}
	if client.SecretHash != nil && security.CompareClientSecret(plaintext, *client.SecretHash) {
		return true
	}
	if client.PreviousSecretHash == nil || client.PreviousSecretExpiresAt == nil {
		return false
	}
	if !client.PreviousSecretExpiresAt.After(time.Now()) {
		return false
	}
	return security.CompareClientSecret(plaintext, *client.PreviousSecretHash)
}

// baselineScopes are the scopes this server implements itself and grants to any
// client without the operator having to enumerate them: the OIDC Core §5.4
// standard scopes plus RFC 6749 §1.5 offline_access. They confer only the
// caller's own profile data and a refresh token — never authority over anything
// else — which is what makes them safe as the floor.
//
// Everything outside this set is an API scope defined by the deployment, and an
// API scope must be granted explicitly.
var baselineScopes = map[string]struct{}{
	"openid":         {},
	"profile":        {},
	"email":          {},
	"phone":          {},
	"address":        {},
	"offline_access": {},
}

// validateClientAllowedScopes enforces RFC 6749 §3.3: the authorization server
// decides which scopes a client may receive, and a scope outside that set is
// invalid_scope.
//
// An EMPTY allowlist used to short-circuit to "allowed", i.e. it meant "every
// scope". The column defaults to '{}', so every client shipped that way —
// including consent-free seeded system clients — would mint a token asserting
// whatever the caller typed. Requesting "openid admin:write tenants:delete"
// against a seeded public SPA client returned a signed token carrying those
// scopes with no consent screen and no client secret. Nothing else in the
// codebase asserted that a requested scope even existed.
//
// An empty allowlist now means "the baseline OIDC scopes only", never "all", and
// a scope that is neither baseline nor explicitly allowlisted is rejected
// whether or not the client has an allowlist — that is the "is this a known
// scope" check that was missing everywhere.
func validateClientAllowedScopes(client *Client, scope string) *apperror.OAuthError {
	requested := parseScopeFields(scope)
	if len(requested) == 0 {
		return nil
	}
	if client == nil {
		return apperror.NewOAuthInvalidScope("requested scope is not allowed for this client")
	}

	// An explicit allowlist is authoritative — an operator who narrowed a client
	// to "openid email" meant to exclude the rest, baseline or not. Only an
	// EMPTY allowlist falls back to the baseline, and that fallback is the whole
	// point: it is a floor, never a blanket grant.
	allowed := scopeSet(client.AllowedScopes)
	if len(allowed) == 0 {
		allowed = baselineScopes
	}
	for _, r := range requested {
		if _, ok := allowed[r]; !ok {
			// Naming the offending scope is explicitly sanctioned by RFC 6749 §5.2
			// (error_description) and leaks nothing: the caller already knows what
			// it asked for.
			return apperror.NewOAuthInvalidScope("requested scope is not allowed for this client: " + r)
		}
	}
	return nil
}

func validateRequestedScopesSubset(requestedScope, grantedScope string) *apperror.OAuthError {
	if strings.TrimSpace(requestedScope) == "" {
		return nil
	}

	granted := scopeSet(parseScopeFields(grantedScope))
	for _, requested := range parseScopeFields(requestedScope) {
		if _, ok := granted[requested]; !ok {
			return apperror.NewOAuthInvalidScope("requested scope exceeds the original grant")
		}
	}
	return nil
}

func scopeSet(scopes []string) map[string]struct{} {
	set := make(map[string]struct{}, len(scopes))
	for _, scope := range scopes {
		scope = strings.TrimSpace(scope)
		if scope != "" {
			set[scope] = struct{}{}
		}
	}
	return set
}

func parseScopeFields(scope string) pq.StringArray {
	return pq.StringArray(strings.Fields(scope))
}
