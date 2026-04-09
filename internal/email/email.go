package email

import (
	"context"
	"crypto/tls"
	"fmt"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"

	"github.com/maintainerd/auth/internal/config"
	"gopkg.in/gomail.v2"
)

// SendEmailParams holds the parameters for sending an email.
type SendEmailParams struct {
	To        string
	From      string // Optional, if empty use config.SMTPFrom
	Subject   string
	BodyHTML  string
	BodyPlain string // Optional plain text fallback
}

// SendEmail is the default email sender. It can be replaced in tests.
var SendEmail = sendEmail

// sendEmail sends an email with the given parameters.
func sendEmail(ctx context.Context, params SendEmailParams) error {
	_, span := otel.Tracer("email").Start(ctx, "smtp.send")
	defer span.End()

	span.SetAttributes(
		attribute.String("smtp.host", config.SMTPHost),
		attribute.Int("smtp.port", config.SMTPPort),
		attribute.String("email.to", params.To),
		attribute.String("email.subject", params.Subject),
	)

	from := params.From
	if from == "" {
		from = gomail.NewMessage().FormatAddress(config.SMTPFromEmail, config.SMTPFromName)
	}

	m := gomail.NewMessage()
	m.SetHeader("From", from)
	m.SetHeader("To", params.To)
	m.SetHeader("Subject", params.Subject)

	if params.BodyPlain != "" {
		m.SetBody("text/plain", params.BodyPlain)
		m.AddAlternative("text/html", params.BodyHTML)
	} else {
		m.SetBody("text/html", params.BodyHTML)
	}

	d := gomail.NewDialer(config.SMTPHost, config.SMTPPort, config.SMTPUser, config.SMTPPass)
	d.TLSConfig = &tls.Config{
		MinVersion: tls.VersionTLS12,
		ServerName: config.SMTPHost,
	}
	if err := d.DialAndSend(m); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "smtp send failed")
		return fmt.Errorf("failed to send email: %w", err)
	}

	span.SetStatus(codes.Ok, "sent")
	return nil
}
