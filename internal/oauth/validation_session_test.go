package oauth

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOAuthBackchannelLogoutRequestDTO_Validate(t *testing.T) {
	cases := []struct {
		name    string
		dto     OAuthBackchannelLogoutRequestDTO
		wantErr string
	}{
		{"valid", OAuthBackchannelLogoutRequestDTO{LogoutToken: "eyJ..."}, ""},
		{"missing logout_token", OAuthBackchannelLogoutRequestDTO{}, "logout_token"},
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

func TestOAuthEndSessionRequestDTO_Validate(t *testing.T) {
	cases := []struct {
		name    string
		dto     OAuthEndSessionRequestDTO
		wantErr string
	}{
		{"valid", OAuthEndSessionRequestDTO{}, ""},
		{"post_logout_redirect_uri too long", OAuthEndSessionRequestDTO{PostLogoutRedirectURI: strings.Repeat("x", 3000)}, "post_logout_redirect_uri"},
		{"state too long", OAuthEndSessionRequestDTO{State: strings.Repeat("s", 600)}, "state"},
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
