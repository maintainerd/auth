package dto

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func validSMSConfigUpdate() SMSConfigUpdateRequestDTO {
	return SMSConfigUpdateRequestDTO{
		Provider:   "twilio",
		AccountSID: "AC123",
		AuthToken:  "token123",
		FromNumber: "+15551234567",
		SenderID:   "MySender",
	}
}

func TestSMSConfigUpdateRequestDTO_Validate(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		assert.NoError(t, validSMSConfigUpdate().Validate())
	})

	t.Run("valid all providers", func(t *testing.T) {
		for _, p := range []string{"twilio", "sns", "vonage", "messagebird"} {
			d := validSMSConfigUpdate()
			d.Provider = p
			assert.NoError(t, d.Validate(), "provider: %s", p)
		}
	})

	t.Run("missing provider", func(t *testing.T) {
		d := validSMSConfigUpdate()
		d.Provider = ""
		require.Error(t, d.Validate())
	})

	t.Run("invalid provider", func(t *testing.T) {
		d := validSMSConfigUpdate()
		d.Provider = "unknown"
		require.Error(t, d.Validate())
	})

	t.Run("account_sid too long", func(t *testing.T) {
		d := validSMSConfigUpdate()
		d.AccountSID = strings.Repeat("a", 256)
		require.Error(t, d.Validate())
	})

	t.Run("from_number too long", func(t *testing.T) {
		d := validSMSConfigUpdate()
		d.FromNumber = strings.Repeat("1", 51)
		require.Error(t, d.Validate())
	})

	t.Run("sender_id too long", func(t *testing.T) {
		d := validSMSConfigUpdate()
		d.SenderID = strings.Repeat("a", 51)
		require.Error(t, d.Validate())
	})
}
