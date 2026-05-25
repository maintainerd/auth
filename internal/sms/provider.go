package sms

import "context"

// Provider is the interface for sending SMS messages.
type Provider interface {
	Send(ctx context.Context, to, body string) error
}

// ProviderConfig holds the configuration for all SMS providers.
type ProviderConfig struct {
	Provider        string
	TwilioSID       string
	TwilioToken     string
	TwilioFrom      string
	SNSRegion       string
	VonageAPIKey    string
	VonageAPISecret string
	VonageFrom      string
}
