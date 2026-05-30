package oauth

import (
	"github.com/maintainerd/auth/internal/platform/apperror"
	"gorm.io/gorm"
)

func authenticateOAuthClient(db *gorm.DB, creds OAuthClientCredentials) (*Client, *apperror.OAuthError) {
	if creds.ClientID == "" {
		return nil, apperror.NewOAuthInvalidClient("client_id is required")
	}
	client, err := findActiveClientByIdentifier(db, creds.ClientID)
	if err != nil {
		return nil, apperror.NewOAuthServerError("an unexpected error occurred")
	}
	if client == nil {
		return nil, apperror.NewOAuthInvalidClient("client authentication failed")
	}
	if client.TokenEndpointAuthMethod == TokenAuthMethodNone {
		return client, nil
	}
	if client.TokenEndpointAuthMethod == TokenAuthMethodSecretBasic || client.TokenEndpointAuthMethod == TokenAuthMethodSecretPost {
		if !clientSecretMatches(client, creds.ClientSecret) {
			return nil, apperror.NewOAuthInvalidClient("client authentication failed")
		}
		return client, nil
	}
	return nil, apperror.NewOAuthInvalidClient("unsupported token_endpoint_auth_method")
}

func clientHasGrant(client *Client, grantType string) bool {
	if client.GrantTypes == nil {
		return false
	}
	for _, g := range client.GrantTypes {
		if g == grantType {
			return true
		}
	}
	return false
}
