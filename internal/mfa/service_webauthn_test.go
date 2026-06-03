package mfa

import (
	"testing"

	"github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/webauthn"
	"github.com/stretchr/testify/assert"
)

func TestWebAuthnUserAdapter(t *testing.T) {
	creds := []webauthn.Credential{{ID: []byte("credential-id")}}
	user := &webAuthnUser{
		user:  &User{UserID: 42, Email: "user@example.com"},
		creds: creds,
	}

	assert.Equal(t, []byte("42"), user.WebAuthnID())
	assert.Equal(t, "user@example.com", user.WebAuthnName())
	assert.Equal(t, "user@example.com", user.WebAuthnDisplayName())
	assert.Equal(t, creds, user.WebAuthnCredentials())
}

func TestRPIDFromHostname(t *testing.T) {
	tests := []struct {
		name     string
		hostname string
		want     string
	}{
		{name: "https hostname", hostname: "https://auth.example.com", want: "auth.example.com"},
		{name: "http hostname", hostname: "http://localhost:8080", want: "localhost"},
		{name: "bare hostname", hostname: "auth.example.com", want: "auth.example.com"},
		{name: "bare hostname with port", hostname: "auth.example.com:8443", want: "auth.example.com"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, rpIDFromHostname(tt.hostname))
		})
	}
}

func TestJoinTransports(t *testing.T) {
	assert.Equal(t, "", joinTransports(nil))
	assert.Equal(t, "usb,nfc", joinTransports([]protocol.AuthenticatorTransport{
		protocol.USB,
		protocol.NFC,
	}))
}
