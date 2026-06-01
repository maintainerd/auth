package oauth

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOAuthPARRequestDTO_Validate(t *testing.T) {
	cases := []struct {
		name    string
		dto     OAuthPARRequestDTO
		wantErr string
	}{
		{
			"valid",
			OAuthPARRequestDTO{
				ResponseType:        "code",
				ClientID:            "app",
				RedirectURI:         "https://example.com/cb",
				CodeChallenge:       "dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk",
				CodeChallengeMethod: "S256",
			},
			"",
		},
		{"missing response_type", OAuthPARRequestDTO{ClientID: "app", RedirectURI: "https://x.com", CodeChallenge: strings.Repeat("x", 43), CodeChallengeMethod: "S256"}, "response_type"},
		{"invalid response_type", OAuthPARRequestDTO{ResponseType: "token", ClientID: "app", RedirectURI: "https://x.com", CodeChallenge: strings.Repeat("x", 43), CodeChallengeMethod: "S256"}, "response_type"},
		{"missing client_id", OAuthPARRequestDTO{ResponseType: "code", RedirectURI: "https://x.com", CodeChallenge: strings.Repeat("x", 43), CodeChallengeMethod: "S256"}, "client_id"},
		{"client_id too long", OAuthPARRequestDTO{ResponseType: "code", ClientID: strings.Repeat("x", 300), RedirectURI: "https://x.com", CodeChallenge: strings.Repeat("x", 43), CodeChallengeMethod: "S256"}, "client_id"},
		{"missing redirect_uri", OAuthPARRequestDTO{ResponseType: "code", ClientID: "app", CodeChallenge: strings.Repeat("x", 43), CodeChallengeMethod: "S256"}, "redirect_uri"},
		{"redirect_uri too long", OAuthPARRequestDTO{ResponseType: "code", ClientID: "app", RedirectURI: "https://x.com/" + strings.Repeat("y", 3000), CodeChallenge: strings.Repeat("x", 43), CodeChallengeMethod: "S256"}, "redirect_uri"},
		{"missing code_challenge", OAuthPARRequestDTO{ResponseType: "code", ClientID: "app", RedirectURI: "https://x.com", CodeChallengeMethod: "S256"}, "code_challenge"},
		{"code_challenge too short", OAuthPARRequestDTO{ResponseType: "code", ClientID: "app", RedirectURI: "https://x.com", CodeChallenge: "short", CodeChallengeMethod: "S256"}, "code_challenge"},
		{"code_challenge too long", OAuthPARRequestDTO{ResponseType: "code", ClientID: "app", RedirectURI: "https://x.com", CodeChallenge: strings.Repeat("x", 200), CodeChallengeMethod: "S256"}, "code_challenge"},
		{"missing code_challenge_method", OAuthPARRequestDTO{ResponseType: "code", ClientID: "app", RedirectURI: "https://x.com", CodeChallenge: strings.Repeat("x", 43)}, "code_challenge_method"},
		{"invalid code_challenge_method", OAuthPARRequestDTO{ResponseType: "code", ClientID: "app", RedirectURI: "https://x.com", CodeChallenge: strings.Repeat("x", 43), CodeChallengeMethod: "plain"}, "code_challenge_method"},
		{"state too long", OAuthPARRequestDTO{ResponseType: "code", ClientID: "app", RedirectURI: "https://x.com", CodeChallenge: strings.Repeat("x", 43), CodeChallengeMethod: "S256", State: strings.Repeat("s", 600)}, "state"},
		{"scope too long", OAuthPARRequestDTO{ResponseType: "code", ClientID: "app", RedirectURI: "https://x.com", CodeChallenge: strings.Repeat("x", 43), CodeChallengeMethod: "S256", Scope: strings.Repeat("s", 2048)}, "scope"},
		{"nonce too long", OAuthPARRequestDTO{ResponseType: "code", ClientID: "app", RedirectURI: "https://x.com", CodeChallenge: strings.Repeat("x", 43), CodeChallengeMethod: "S256", Nonce: strings.Repeat("n", 600)}, "nonce"},
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
