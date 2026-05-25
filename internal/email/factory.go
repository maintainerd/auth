package email

import (
	"context"
	"fmt"

	"github.com/maintainerd/auth/internal/config"
)

// NewSystemProvider constructs a Provider from the application's global config.
// Provider is selected by the EMAIL_PROVIDER env var (default "smtp").
func NewSystemProvider(ctx context.Context) (Provider, error) {
	cfg := ProviderConfig{
		Provider: config.EmailProvider,
		Host:     config.SMTPHost,
		Port:     config.SMTPPort,
		Username: config.SMTPUser,
		Password: config.SMTPPass,
		APIKey:   config.EmailAPIKey,
		Domain:   config.EmailDomain,
		Region:   config.EmailRegion,
	}
	return NewProvider(ctx, cfg)
}

// NewProvider returns the Provider implementation for cfg.Provider.
func NewProvider(ctx context.Context, cfg ProviderConfig) (Provider, error) {
	switch cfg.Provider {
	case "smtp", "":
		return newSMTPProvider(cfg), nil
	case "ses":
		return newSESProvider(ctx, cfg)
	case "sendgrid":
		return newSendGridProvider(cfg), nil
	case "postmark":
		return newPostmarkProvider(cfg), nil
	case "mailgun":
		return newMailgunProvider(cfg), nil
	case "resend":
		return newResendProvider(cfg), nil
	default:
		return nil, fmt.Errorf("email: unknown provider %q", cfg.Provider)
	}
}
