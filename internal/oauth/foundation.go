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

	ResponseTypeCode = "code"
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

func validateClientAllowedScopes(client *Client, scope string) *apperror.OAuthError {
	if client == nil || len(client.AllowedScopes) == 0 || strings.TrimSpace(scope) == "" {
		return nil
	}

	allowed := scopeSet(client.AllowedScopes)
	for _, requested := range parseScopeFields(scope) {
		if _, ok := allowed[requested]; !ok {
			return apperror.NewOAuthInvalidScope("requested scope is not allowed for this client")
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
