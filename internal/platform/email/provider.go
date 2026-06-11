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
	// Default sender used when a SendParams.From is not supplied.
	FromAddress string
	FromName    string
	// SMTP / SES SMTP
	Host     string
	Port     int
	Username string
	Password string
	// SaaS providers (SendGrid / Postmark / Mailgun / Resend)
	APIKey string
	Domain string // Mailgun domain
	Region string // SES region
}

// ResolveFrom returns the From header to use: the explicit per-send value when
// set, otherwise the provider's configured sender (as "Name <addr>" or "addr").
func (c ProviderConfig) ResolveFrom(from string) string {
	if from != "" {
		return from
	}
	if c.FromAddress == "" {
		return ""
	}
	if c.FromName != "" {
		return c.FromName + " <" + c.FromAddress + ">"
	}
	return c.FromAddress
}
