package oauth

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOAuthCIBARequestDTO_Validate(t *testing.T) {
	cases := []struct {
		name    string
		dto     OAuthCIBARequestDTO
		wantErr string
	}{
		{"valid with login_hint", OAuthCIBARequestDTO{ClientID: "app", Scope: "openid", LoginHint: "user@example.com"}, ""},
		{"missing client_id", OAuthCIBARequestDTO{Scope: "openid", LoginHint: "user"}, "client_id"},
		{"missing scope", OAuthCIBARequestDTO{ClientID: "app", LoginHint: "user"}, "scope"},
		{"missing login hint", OAuthCIBARequestDTO{ClientID: "app", Scope: "openid"}, "login_hint"},
		{"scope too long", OAuthCIBARequestDTO{ClientID: "app", Scope: strings.Repeat("x", 2048), LoginHint: "user"}, "scope"},
		{"binding_message too long", OAuthCIBARequestDTO{ClientID: "app", Scope: "openid", LoginHint: "user", BindingMessage: strings.Repeat("x", 200)}, "binding_message"},
		{"with login_hint_token instead of login_hint", OAuthCIBARequestDTO{ClientID: "app", Scope: "openid", LoginHintToken: "token123"}, ""},
		{"with id_token_hint instead of login_hint", OAuthCIBARequestDTO{ClientID: "app", Scope: "openid", IDTokenHint: "eyJ..."}, ""},
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

func TestOAuthCIBATokenRequestDTO_Validate(t *testing.T) {
	cases := []struct {
		name    string
		dto     OAuthCIBATokenRequestDTO
		wantErr string
	}{
		{"valid", OAuthCIBATokenRequestDTO{AuthReqID: "req123", ClientID: "app"}, ""},
		{"missing auth_req_id", OAuthCIBATokenRequestDTO{ClientID: "app"}, "auth_req_id"},
		{"missing client_id", OAuthCIBATokenRequestDTO{AuthReqID: "req123"}, "client_id"},
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
