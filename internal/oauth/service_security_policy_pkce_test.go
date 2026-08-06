package oauth

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func boolPtr(v bool) *bool { return &v }

// ResolveEffectiveTokenPolicy only ever ESCALATES, so before this a client row
// carrying require_pkce=false could not be raised by anything, and nothing at
// all forced PKCE on for SPA/native clients. Combined with
// token_endpoint_auth_method "none" that leaves the token endpoint with neither
// client authentication nor proof of possession. RFC 9700 §2.1.1.
func TestOAuthEffectiveTokenPolicy_PKCEForcedForPublicClients(t *testing.T) {
	cases := []struct {
		name   string
		client *Client
		want   bool
	}{
		{
			name:   "SPA with require_pkce explicitly false",
			client: &Client{ClientType: "spa", RequirePKCE: boolPtr(false), TokenEndpointAuthMethod: TokenAuthMethodSecretPost},
			want:   true,
		},
		{
			name:   "mobile with require_pkce explicitly false",
			client: &Client{ClientType: "mobile", RequirePKCE: boolPtr(false), TokenEndpointAuthMethod: TokenAuthMethodSecretPost},
			want:   true,
		},
		{
			name: "any client type registered with token_endpoint_auth_method none",
			// The declared type does not matter: authenticateOAuthClient
			// short-circuits on the auth method, so this client presents no
			// credential at redemption whatever it calls itself.
			client: &Client{ClientType: "web", RequirePKCE: boolPtr(false), TokenEndpointAuthMethod: TokenAuthMethodNone},
			want:   true,
		},
		{
			// The shipped tenant default is require_pkce: true, so a confidential
			// client is covered by that already; the opt-out only bites once a
			// tenant turns the default off, which is covered in secpolicy's
			// ResolveEffectiveTokenPolicy tests.
			name:   "confidential client can still opt in",
			client: &Client{ClientType: "web", RequirePKCE: boolPtr(true), TokenEndpointAuthMethod: TokenAuthMethodSecretPost},
			want:   true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			policy := oauthEffectiveTokenPolicy(nil, tc.client)
			assert.Equal(t, tc.want, policy.RequirePKCE)
		})
	}
}

func TestIsPublicOAuthClient(t *testing.T) {
	assert.True(t, isPublicOAuthClient(&Client{ClientType: "spa"}))
	assert.True(t, isPublicOAuthClient(&Client{ClientType: "mobile"}))
	assert.True(t, isPublicOAuthClient(&Client{ClientType: "web", TokenEndpointAuthMethod: TokenAuthMethodNone}))
	assert.False(t, isPublicOAuthClient(&Client{ClientType: "web", TokenEndpointAuthMethod: TokenAuthMethodSecretBasic}))
	assert.False(t, isPublicOAuthClient(&Client{ClientType: "m2m", TokenEndpointAuthMethod: TokenAuthMethodSecretPost}))
	assert.False(t, isPublicOAuthClient(nil))
}
