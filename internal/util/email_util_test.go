package util

import (
	"testing"

	"github.com/maintainerd/auth/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setSMTPConfig sets up config fields and restores them after the test.
func setSMTPConfig(t *testing.T, host string, port int, user, pass, fromEmail, fromName string) {
	t.Helper()
	origHost, origPort := config.SMTPHost, config.SMTPPort
	origUser, origPass := config.SMTPUser, config.SMTPPass
	origFrom, origName := config.SMTPFromEmail, config.SMTPFromName

	config.SMTPHost = host
	config.SMTPPort = port
	config.SMTPUser = user
	config.SMTPPass = pass
	config.SMTPFromEmail = fromEmail
	config.SMTPFromName = fromName

	t.Cleanup(func() {
		config.SMTPHost = origHost
		config.SMTPPort = origPort
		config.SMTPUser = origUser
		config.SMTPPass = origPass
		config.SMTPFromEmail = origFrom
		config.SMTPFromName = origName
	})
}

func TestSendEmail_FailsWhenSMTPUnreachable(t *testing.T) {
	// Point SMTP at localhost:1 — guaranteed to be closed / unreachable
	setSMTPConfig(t, "127.0.0.1", 1, "", "", "noreply@example.com", "Test")

	err := SendEmail(SendEmailParams{
		To:       "user@example.com",
		Subject:  "Hello",
		BodyHTML: "<p>Hello</p>",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to send email")
}

func TestSendEmail_FailsWhenSMTPUnreachable_WithFrom(t *testing.T) {
	setSMTPConfig(t, "127.0.0.1", 1, "", "", "noreply@example.com", "Test")

	err := SendEmail(SendEmailParams{
		To:       "user@example.com",
		From:     "custom@sender.com",
		Subject:  "Custom From",
		BodyHTML: "<p>body</p>",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to send email")
}

func TestSendEmail_FailsWhenSMTPUnreachable_WithPlainText(t *testing.T) {
	setSMTPConfig(t, "127.0.0.1", 1, "", "", "noreply@example.com", "Test")

	err := SendEmail(SendEmailParams{
		To:        "user@example.com",
		Subject:   "Plain + HTML",
		BodyHTML:  "<p>Hello</p>",
		BodyPlain: "Hello",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to send email")
}

func TestSendEmail_FailsWithBadHost(t *testing.T) {
	setSMTPConfig(t, "this-host-does-not-exist.invalid", 587, "", "", "noreply@example.com", "Test")

	err := SendEmail(SendEmailParams{
		To:       "user@example.com",
		Subject:  "Bad host",
		BodyHTML: "<p>body</p>",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to send email")
}

