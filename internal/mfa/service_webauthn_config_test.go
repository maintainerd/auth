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
	t.Setenv("WEBAUTHN_EXTRA_ORIGINS", "https://console.auth.maintainerd.local,https://identity.auth.maintainerd.local")
	config.AppPublicHostname = "https://public-api.auth.maintainerd.local"
	db, _ := newMockGormDB(t)

	got, err := NewWebAuthnService(db, &mockUserRepo{}, &mockMFAWebAuthnCredentialRepo{}, &mockWebAuthnSessionStore{}, &mockAuthEventService{}, nil)

	require.NoError(t, err)
	configured := got.(*webAuthnService).wa.Config
	assert.Equal(t, "auth.maintainerd.local", configured.RPID)
	assert.ElementsMatch(t, []string{
		"https://public-api.auth.maintainerd.local",
		"https://console.auth.maintainerd.local",
		"https://identity.auth.maintainerd.local",
	}, configured.RPOrigins)
}
