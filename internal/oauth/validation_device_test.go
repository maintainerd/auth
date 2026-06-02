package oauth

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOAuthDeviceAuthorizationRequestDTO_Validate(t *testing.T) {
	cases := []struct {
		name    string
		dto     OAuthDeviceAuthorizationRequestDTO
		wantErr string
	}{
		{"valid", OAuthDeviceAuthorizationRequestDTO{ClientID: "app", Scope: "openid"}, ""},
		{"missing client_id", OAuthDeviceAuthorizationRequestDTO{}, "client_id"},
		{"client_id too long", OAuthDeviceAuthorizationRequestDTO{ClientID: strings.Repeat("x", 300)}, "client_id"},
		{"scope too long", OAuthDeviceAuthorizationRequestDTO{ClientID: "app", Scope: strings.Repeat("x", 2000)}, "scope"},
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

func TestOAuthDeviceVerifyRequestDTO_Validate(t *testing.T) {
	cases := []struct {
		name     string
		userCode string
		wantErr  string
	}{
		{"valid 8-char code", "ABCD-123", ""},
		{"valid 9-char code", "ABCD-1234", ""},
		{"missing user_code", "", "user_code"},
		{"too short", "AB-CD-1", "user_code"},
		{"too long", "ABCD-12345", "user_code"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dto := &OAuthDeviceVerifyRequestDTO{UserCode: tc.userCode}
			err := dto.Validate()
			if tc.wantErr == "" {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.wantErr)
			}
		})
	}
}

func TestOAuthDeviceTokenRequestDTO_Validate(t *testing.T) {
	cases := []struct {
		name    string
		dto     OAuthDeviceTokenRequestDTO
		wantErr string
	}{
		{"valid", OAuthDeviceTokenRequestDTO{DeviceCode: "dev123", ClientID: "app"}, ""},
		{"missing device_code", OAuthDeviceTokenRequestDTO{ClientID: "app"}, "device_code"},
		{"missing client_id", OAuthDeviceTokenRequestDTO{DeviceCode: "dev123"}, "client_id"},
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
