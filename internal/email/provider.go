package email

import "context"

// Provider is the contract for sending a single email message.
// All implementations must be safe for concurrent use.
type Provider interface {
	Send(ctx context.Context, params SendParams) error
}

// SendParams carries the data needed to deliver one email.
type SendParams struct {
	To        string
	From      string // "addr" or "Name <addr>"
	Subject   string
	BodyHTML  string
	BodyPlain string // optional plain-text fallback
}

// ProviderConfig is a normalised view of tenant or system email config.
type ProviderConfig struct {
	Provider string // smtp | ses | sendgrid | mailgun | postmark | resend
	// SMTP / SES SMTP
	Host     string
	Port     int
	Username string
	Password string
	// SaaS providers (SendGrid / Postmark / Mailgun / Resend)
	APIKey  string
	Domain  string // Mailgun domain
	Region  string // SES region
}
