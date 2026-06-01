package oauth

import (
	"net/http"
	"time"

	"github.com/maintainerd/auth/internal/platform/cache"
	"github.com/maintainerd/auth/internal/platform/database"
	"github.com/maintainerd/auth/internal/platform/pagination"
	"github.com/maintainerd/auth/internal/platform/security"
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
// Type aliases — cache auth types stored in middleware.AuthContext
// ---------------------------------------------------------------------------

// User and Profile are aliases for the cache auth types so that handlers can
// inject rich user data into the auth context and read it back.
type User = cache.AuthUser
type Profile = cache.AuthProfile

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
