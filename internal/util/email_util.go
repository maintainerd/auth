package util

import (
	"fmt"

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

// SendEmail sends an email with the given parameters.
func SendEmail(params SendEmailParams) error {
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
	if err := d.DialAndSend(m); err != nil {
		return fmt.Errorf("failed to send email: %w", err)
	}

	return nil
}
