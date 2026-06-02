package authn

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSMSLoginSendDTO_Validate(t *testing.T) {
	tests := []struct {
		name     string
		phone    string
		client   string
		provider string
		wantErr  bool
	}{
		{"valid", "+1234567890", "myapp", "default", false},
		{"missing phone", "", "myapp", "default", true},
		{"missing client_id", "+1234567890", "", "default", true},
		{"missing provider_id", "+1234567890", "myapp", "", true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			dto := &SMSLoginSendDTO{Phone: tc.phone, ClientID: tc.client, ProviderID: tc.provider}
			err := dto.Validate()
			if tc.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestSMSLoginVerifyDTO_Validate(t *testing.T) {
	tests := []struct {
		name     string
		phone    string
		otp      string
		client   string
		provider string
		wantErr  bool
	}{
		{"valid", "+1234567890", "123456", "myapp", "default", false},
		{"missing phone", "", "123456", "myapp", "default", true},
		{"missing otp", "+1234567890", "", "myapp", "default", true},
		{"otp not 6 chars", "+1234567890", "12345", "myapp", "default", true},
		{"otp > 6 chars", "+1234567890", "1234567", "myapp", "default", true},
		{"missing client_id", "+1234567890", "123456", "", "default", true},
		{"missing provider_id", "+1234567890", "123456", "myapp", "", true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			dto := &SMSLoginVerifyDTO{Phone: tc.phone, OTP: tc.otp, ClientID: tc.client, ProviderID: tc.provider}
			err := dto.Validate()
			if tc.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestSMSLoginSendDTO_Validate_PhoneLength(t *testing.T) {
	dto := &SMSLoginSendDTO{Phone: strings.Repeat("1", 21), ClientID: "app", ProviderID: "idp"}
	err := dto.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "phone")
}
