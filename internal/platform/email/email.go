package email

import (
	"context"

	"github.com/maintainerd/auth/internal/platform/config"
	"gopkg.in/gomail.v2"
)

// SendEmailParams holds the parameters for sending an email.
// Kept for backwards compatibility with existing callers.
type SendEmailParams struct {
	To        string
	From      string // optional; falls back to config.SMTPFromEmail
	Subject   string
	BodyHTML  string
	BodyPlain string
}

// SendEmail is the default email sender. It delegates to the system provider
// selected by the EMAIL_PROVIDER env var. It can be replaced in tests.
var SendEmail = sendEmail

func sendEmail(ctx context.Context, params SendEmailParams) error {
	from := params.From
	if from == "" {
		from = gomail.NewMessage().FormatAddress(config.SMTPFromEmail, config.SMTPFromName)
	}

	provider, err := NewSystemProvider(ctx)
	if err != nil {
		return err
	}
	return provider.Send(ctx, SendParams{
		To:        params.To,
		From:      from,
		Subject:   params.Subject,
		BodyHTML:  params.BodyHTML,
		BodyPlain: params.BodyPlain,
	})
}
