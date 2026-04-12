package dto

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func validEmailConfigUpdate() EmailConfigUpdateRequestDTO {
	return EmailConfigUpdateRequestDTO{
		Provider:    "smtp",
		Host:        "smtp.example.com",
		Port:        587,
		Username:    "user",
		Password:    "pass",
		FromAddress: "noreply@example.com",
		FromName:    "Acme",
		ReplyTo:     "support@example.com",
		Encryption:  "tls",
	}
}

func TestEmailConfigUpdateRequestDTO_Validate(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		assert.NoError(t, validEmailConfigUpdate().Validate())
	})

	t.Run("valid all providers", func(t *testing.T) {
		for _, p := range []string{"smtp", "ses", "sendgrid", "mailgun", "postmark", "resend"} {
			d := validEmailConfigUpdate()
			d.Provider = p
			assert.NoError(t, d.Validate(), "provider: %s", p)
		}
	})

	t.Run("valid all encryption types", func(t *testing.T) {
		for _, e := range []string{"tls", "ssl", "none"} {
			d := validEmailConfigUpdate()
			d.Encryption = e
			assert.NoError(t, d.Validate(), "encryption: %s", e)
		}
	})

	t.Run("empty encryption allowed", func(t *testing.T) {
		d := validEmailConfigUpdate()
		d.Encryption = ""
		assert.NoError(t, d.Validate())
	})

	t.Run("missing provider", func(t *testing.T) {
		d := validEmailConfigUpdate()
		d.Provider = ""
		require.Error(t, d.Validate())
	})

	t.Run("invalid provider", func(t *testing.T) {
		d := validEmailConfigUpdate()
		d.Provider = "unknown"
		require.Error(t, d.Validate())
	})

	t.Run("missing from_address", func(t *testing.T) {
		d := validEmailConfigUpdate()
		d.FromAddress = ""
		require.Error(t, d.Validate())
	})

	t.Run("invalid from_address", func(t *testing.T) {
		d := validEmailConfigUpdate()
		d.FromAddress = "not-an-email"
		require.Error(t, d.Validate())
	})

	t.Run("from_address too long", func(t *testing.T) {
		d := validEmailConfigUpdate()
		d.FromAddress = strings.Repeat("a", 250) + "@b.com"
		require.Error(t, d.Validate())
	})

	t.Run("from_name too long", func(t *testing.T) {
		d := validEmailConfigUpdate()
		d.FromName = strings.Repeat("a", 256)
		require.Error(t, d.Validate())
	})

	t.Run("invalid reply_to", func(t *testing.T) {
		d := validEmailConfigUpdate()
		d.ReplyTo = "not-an-email"
		require.Error(t, d.Validate())
	})

	t.Run("reply_to too long", func(t *testing.T) {
		d := validEmailConfigUpdate()
		d.ReplyTo = strings.Repeat("a", 250) + "@b.com"
		require.Error(t, d.Validate())
	})

	t.Run("invalid encryption", func(t *testing.T) {
		d := validEmailConfigUpdate()
		d.Encryption = "starttls"
		require.Error(t, d.Validate())
	})

	t.Run("host too long", func(t *testing.T) {
		d := validEmailConfigUpdate()
		d.Host = strings.Repeat("a", 256)
		require.Error(t, d.Validate())
	})

	t.Run("username too long", func(t *testing.T) {
		d := validEmailConfigUpdate()
		d.Username = strings.Repeat("a", 256)
		require.Error(t, d.Validate())
	})

	t.Run("empty reply_to allowed", func(t *testing.T) {
		d := validEmailConfigUpdate()
		d.ReplyTo = ""
		assert.NoError(t, d.Validate())
	})

	t.Run("port zero allowed", func(t *testing.T) {
		d := validEmailConfigUpdate()
		d.Port = 0
		assert.NoError(t, d.Validate())
	})

	t.Run("port valid", func(t *testing.T) {
		for _, p := range []int{1, 25, 465, 587, 65535} {
			d := validEmailConfigUpdate()
			d.Port = p
			assert.NoError(t, d.Validate(), "port: %d", p)
		}
	})

	t.Run("port exceeds max", func(t *testing.T) {
		d := validEmailConfigUpdate()
		d.Port = 65536
		require.Error(t, d.Validate())
	})

	t.Run("port negative", func(t *testing.T) {
		d := validEmailConfigUpdate()
		d.Port = -1
		require.Error(t, d.Validate())
	})
}
