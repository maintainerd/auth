package authn

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSMSLoginSendDTO_Validate(t *testing.T) {
	tests := []struct {
		name    string
		phone   string
		wantErr bool
	}{
		{"valid", "+1234567890", false},
		{"missing phone", "", true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			dto := &SMSLoginSendDTO{Phone: tc.phone}
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
		name    string
		phone   string
		otp     string
		wantErr bool
	}{
		{"valid", "+1234567890", "123456", false},
		{"missing phone", "", "123456", true},
		{"missing otp", "+1234567890", "", true},
		{"otp not 6 chars", "+1234567890", "12345", true},
		{"otp > 6 chars", "+1234567890", "1234567", true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			dto := &SMSLoginVerifyDTO{Phone: tc.phone, OTP: tc.otp}
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
	dto := &SMSLoginSendDTO{Phone: strings.Repeat("1", 21)}
	err := dto.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "phone")
}
