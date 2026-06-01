package oauth

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOAuthClientRegistrationRequestDTO_Validate(t *testing.T) {
	cases := []struct {
		name    string
		dto     OAuthClientRegistrationRequestDTO
		wantErr string
	}{
		{
			"valid",
			OAuthClientRegistrationRequestDTO{
				ClientName:            "My App",
				RedirectURIs:          []string{"https://example.com/cb"},
				TokenEndpointAuthMethod: "client_secret_basic",
				IdentityProviderID:    1,
			},
			"",
		},
		{"missing client_name", OAuthClientRegistrationRequestDTO{RedirectURIs: []string{"https://x.com"}, IdentityProviderID: 1}, "client_name"},
		{"client_name too long", OAuthClientRegistrationRequestDTO{ClientName: strings.Repeat("x", 300), RedirectURIs: []string{"https://x.com"}, IdentityProviderID: 1}, "client_name"},
		{"missing redirect_uris", OAuthClientRegistrationRequestDTO{ClientName: "App", IdentityProviderID: 1}, "redirect_uris"},
		{"empty redirect_uris", OAuthClientRegistrationRequestDTO{ClientName: "App", RedirectURIs: []string{}, IdentityProviderID: 1}, "redirect_uris"},
		{"too many redirect_uris", OAuthClientRegistrationRequestDTO{ClientName: "App", RedirectURIs: make([]string, 11), IdentityProviderID: 1}, "redirect_uris"},
		{"invalid auth method", OAuthClientRegistrationRequestDTO{ClientName: "App", RedirectURIs: []string{"https://x.com"}, TokenEndpointAuthMethod: "private_key_jwt", IdentityProviderID: 1}, "token_endpoint_auth_method"},
		{"scope too long", OAuthClientRegistrationRequestDTO{ClientName: "App", RedirectURIs: []string{"https://x.com"}, Scope: strings.Repeat("x", 2000), IdentityProviderID: 1}, "scope"},
		{"missing identity_provider_id", OAuthClientRegistrationRequestDTO{ClientName: "App", RedirectURIs: []string{"https://x.com"}}, "identity_provider_id"},
		{"identity_provider_id zero", OAuthClientRegistrationRequestDTO{ClientName: "App", RedirectURIs: []string{"https://x.com"}, IdentityProviderID: 0}, "identity_provider_id"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.dto.Validate()
			if tc.wantErr == "" {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.wantErr)
			}
		})
	}
}
