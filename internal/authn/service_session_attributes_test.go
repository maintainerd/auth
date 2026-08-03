package authn

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/maintainerd/maintainerd-auth/internal/secpolicy"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// user_sessions carries client_id, identity_provider_id, idp_session_id, amr and
// acr. Every one of them was previously left NULL/'{}'/"1", which made sessions
// indistinguishable from one another: an MFA-completed login looked like a
// password-only one, and an upstream back-channel logout had no way to find the
// session it had just ended.
func TestCreateSessionWithPolicy_RecordsAuthenticationFacts(t *testing.T) {
	clientID := int64(42)
	idpID := int64(7)
	upstreamSID := "upstream-session-abc"

	var got *UserSession
	repo := &mockUserSessionRepo{
		createFn: func(s *UserSession) error {
			s.UserSessionUUID = uuid.New()
			got = s
			return nil
		},
	}

	svc := NewSessionService(repo).(*sessionService)
	_, err := svc.CreateSessionWithPolicy(
		context.Background(), 1, 1, "203.0.113.7", "Mozilla/5.0",
		secpolicy.EffectiveSessionPolicy{IdleTimeoutSeconds: 600, AbsoluteTimeoutSeconds: 7200},
		SessionAttributes{
			AMR:                []string{"pwd", "otp"},
			ACR:                "2",
			ClientID:           &clientID,
			IdentityProviderID: &idpID,
			IDPSessionID:       &upstreamSID,
		},
	)
	require.NoError(t, err)
	require.NotNil(t, got)

	assert.Equal(t, []string{"pwd", "otp"}, []string(got.AMR))
	assert.Equal(t, "2", got.ACR, "an MFA-completed login must not be recorded as acr=1")
	require.NotNil(t, got.ClientID)
	assert.Equal(t, clientID, *got.ClientID)
	require.NotNil(t, got.IdentityProviderID)
	assert.Equal(t, idpID, *got.IdentityProviderID)
	require.NotNil(t, got.IDPSessionID)
	assert.Equal(t, upstreamSID, *got.IDPSessionID)
}

// An unspecified acr must still be a valid value, not an empty string — the
// column is NOT NULL and consumers compare against "1"/"2".
func TestCreateSessionWithPolicy_DefaultsACRWhenUnset(t *testing.T) {
	var got *UserSession
	repo := &mockUserSessionRepo{
		createFn: func(s *UserSession) error {
			s.UserSessionUUID = uuid.New()
			got = s
			return nil
		},
	}

	svc := NewSessionService(repo).(*sessionService)
	_, err := svc.CreateSessionWithPolicy(
		context.Background(), 1, 1, "", "",
		secpolicy.EffectiveSessionPolicy{IdleTimeoutSeconds: 600, AbsoluteTimeoutSeconds: 7200},
		SessionAttributes{},
	)
	require.NoError(t, err)
	assert.Equal(t, "1", got.ACR)
	assert.Nil(t, got.ClientID)
	assert.Nil(t, got.IDPSessionID)
}
