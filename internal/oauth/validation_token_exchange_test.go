package oauth

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOAuthTokenExchangeRequestDTO_Validate(t *testing.T) {
	validTokenType := "urn:ietf:params:oauth:token-type:access_token"
	cases := []struct {
		name    string
		dto     OAuthTokenExchangeRequestDTO
		wantErr string
	}{
		{
			"valid",
			OAuthTokenExchangeRequestDTO{
				SubjectToken:       "tok123",
				SubjectTokenType:   validTokenType,
				RequestedTokenType: validTokenType,
				ClientID:           "app",
			},
			"",
		},
		{"missing subject_token", OAuthTokenExchangeRequestDTO{SubjectTokenType: validTokenType, ClientID: "app"}, "subject_token"},
		{"missing subject_token_type", OAuthTokenExchangeRequestDTO{SubjectToken: "tok", ClientID: "app"}, "subject_token_type"},
		{"invalid subject_token_type", OAuthTokenExchangeRequestDTO{SubjectToken: "tok", SubjectTokenType: "invalid", ClientID: "app"}, "subject_token_type"},
		{"invalid requested_token_type", OAuthTokenExchangeRequestDTO{SubjectToken: "tok", SubjectTokenType: validTokenType, RequestedTokenType: "invalid", ClientID: "app"}, "requested_token_type"},
		{"missing client_id", OAuthTokenExchangeRequestDTO{SubjectToken: "tok", SubjectTokenType: validTokenType}, "client_id"},
		{"scope too long", OAuthTokenExchangeRequestDTO{SubjectToken: "tok", SubjectTokenType: validTokenType, ClientID: "app", Scope: strings.Repeat("x", 2000)}, "scope"},
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
