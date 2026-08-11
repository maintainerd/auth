package mfa

import (
	"testing"

	"github.com/maintainerd/maintainerd-auth/internal/platform/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewWebAuthnService_ConfiguresSharedLocalRP(t *testing.T) {
	original := config.AppPublicHostname
	t.Cleanup(func() { config.AppPublicHostname = original })
	t.Setenv("WEBAUTHN_RP_ID", "auth.maintainerd.local")
	// Deliberately set to prove it is now IGNORED: origin validation is per-request
	// (waForOrigin accepts any origin under the RP ID), so the static Begin-only
	// instance no longer consumes an extra-origins list.
	t.Setenv("WEBAUTHN_EXTRA_ORIGINS", "https://console.auth.maintainerd.local,https://identity.auth.maintainerd.local")
	config.AppPublicHostname = "https://public-api.auth.maintainerd.local"
	db, _ := newMockGormDB(t)

	got, err := NewWebAuthnService(db, &mockUserRepo{}, &mockMFAWebAuthnCredentialRepo{}, &mockWebAuthnSessionStore{}, &mockAuthEventService{}, &mockWebAuthnChallengeRepo{})

	require.NoError(t, err)
	configured := got.(*webAuthnService).wa.Config
	assert.Equal(t, "auth.maintainerd.local", configured.RPID)
	// RP ID override still applies; WEBAUTHN_EXTRA_ORIGINS is ignored, so the static
	// instance carries only the public hostname.
	assert.Equal(t, []string{"https://public-api.auth.maintainerd.local"}, configured.RPOrigins)
}
