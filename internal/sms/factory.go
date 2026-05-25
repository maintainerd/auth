package sms

import (
	"context"
	"fmt"

	"github.com/maintainerd/auth/internal/config"
)

// NewSystemProvider constructs a Provider from the application's global config.
func NewSystemProvider(ctx context.Context) (Provider, error) {
	cfg := ProviderConfig{
		Provider:        config.SMSProvider,
		TwilioSID:       config.TwilioAccountSID,
		TwilioToken:     config.TwilioAuthToken,
		TwilioFrom:      config.TwilioFromNumber,
		SNSRegion:       config.SNSRegion,
		VonageAPIKey:    config.VonageAPIKey,
		VonageAPISecret: config.VonageAPISecret,
		VonageFrom:      config.VonageFrom,
	}
	return NewProvider(ctx, cfg)
}

// NewProvider returns the Provider implementation for cfg.Provider.
func NewProvider(ctx context.Context, cfg ProviderConfig) (Provider, error) {
	switch cfg.Provider {
	case "twilio":
		return newTwilioProvider(cfg), nil
	case "sns":
		return newSNSProvider(ctx, cfg)
	case "vonage":
		return newVonageProvider(cfg), nil
	default:
		return nil, fmt.Errorf("sms: unknown provider %q", cfg.Provider)
	}
}
