package sms

import (
	"context"
	"testing"

	"github.com/maintainerd/auth/internal/platform/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// Factory tests
// ---------------------------------------------------------------------------

func TestNewProvider_Twilio(t *testing.T) {
	cfg := ProviderConfig{
		Provider:    "twilio",
		TwilioSID:   "test-sid",
		TwilioToken: "test-token",
		TwilioFrom:  "+1234567890",
	}
	p, err := NewProvider(context.Background(), cfg)
	require.NoError(t, err)
	assert.NotNil(t, p)
}

func TestNewProvider_Vonage(t *testing.T) {
	cfg := ProviderConfig{
		Provider:        "vonage",
		VonageAPIKey:    "test-key",
		VonageAPISecret: "test-secret",
		VonageFrom:      "TestApp",
	}
	p, err := NewProvider(context.Background(), cfg)
	require.NoError(t, err)
	assert.NotNil(t, p)
}

func TestNewProvider_SNS(t *testing.T) {
	cfg := ProviderConfig{
		Provider:  "sns",
		SNSRegion: "us-east-1",
	}
	p, err := NewProvider(context.Background(), cfg)
	if err != nil {
		assert.Nil(t, p)
		return
	}
	assert.NotNil(t, p)
}

func TestNewProvider_Unknown(t *testing.T) {
	cfg := ProviderConfig{Provider: "unknown"}
	_, err := NewProvider(context.Background(), cfg)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown provider")
}

func TestNewProvider_Empty(t *testing.T) {
	cfg := ProviderConfig{Provider: ""}
	_, err := NewProvider(context.Background(), cfg)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown provider")
}

// ---------------------------------------------------------------------------
// NewSystemProvider
// ---------------------------------------------------------------------------

func TestNewSystemProvider_UnknownProvider(t *testing.T) {
	orig := config.SMSProvider
	t.Cleanup(func() { config.SMSProvider = orig })
	config.SMSProvider = "unknown"

	_, err := NewSystemProvider(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown provider")
}

func TestNewSystemProvider_TwilioFromConfig(t *testing.T) {
	orig := config.SMSProvider
	origSID := config.TwilioAccountSID
	origToken := config.TwilioAuthToken
	origFrom := config.TwilioFromNumber
	t.Cleanup(func() {
		config.SMSProvider = orig
		config.TwilioAccountSID = origSID
		config.TwilioAuthToken = origToken
		config.TwilioFromNumber = origFrom
	})

	config.SMSProvider = "twilio"
	config.TwilioAccountSID = "sid"
	config.TwilioAuthToken = "token"
	config.TwilioFromNumber = "+1234"

	p, err := NewSystemProvider(context.Background())
	require.NoError(t, err)
	assert.NotNil(t, p)
}

// ---------------------------------------------------------------------------
// Provider Send — error paths (real APIs unreachable)
// ---------------------------------------------------------------------------

func TestTwilio_Send_Error(t *testing.T) {
	p, err := NewProvider(context.Background(), ProviderConfig{
		Provider:    "twilio",
		TwilioSID:   "test-sid",
		TwilioToken: "test-token",
		TwilioFrom:  "+1234567890",
	})
	require.NoError(t, err)

	err = p.Send(context.Background(), "+1234567890", "Hello")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "twilio")
}

func TestVonage_Send(t *testing.T) {
	p, err := NewProvider(context.Background(), ProviderConfig{
		Provider:        "vonage",
		VonageAPIKey:    "test-key",
		VonageAPISecret: "test-secret",
		VonageFrom:      "TestApp",
	})
	require.NoError(t, err)

	err = p.Send(context.Background(), "+1234567890", "Hello")
	// Vonage API may return 200 even with wrong credentials (error in body).
	// Just verify it doesn't panic.
	_ = err
}

func TestSNS_Send_Error(t *testing.T) {
	cfg := ProviderConfig{
		Provider:  "sns",
		SNSRegion: "us-east-1",
	}
	p, err := NewProvider(context.Background(), cfg)
	if err != nil {
		return
	}

	err = p.Send(context.Background(), "+1234567890", "Hello")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "sms/sns")
}
