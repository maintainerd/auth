package authn

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	platformjwt "github.com/maintainerd/maintainerd-auth/internal/platform/jwt"
	"github.com/maintainerd/maintainerd-auth/internal/shared"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockMFAAuthenticator implements MFAFactorAuthenticator for login MFA tests.
type mockMFAAuthenticator struct {
	verifyFactorFn func(userID int64, method, code string, assertion []byte) ([]string, error)
	sendSMSFn      func(userID int64) error
	beginWAFn      func(userID int64) (json.RawMessage, error)
	enrolledFn     func(userID int64) ([]string, error)
}

func (m *mockMFAAuthenticator) EnrolledMFAMethods(_ context.Context, userID int64) ([]string, error) {
	if m.enrolledFn != nil {
		return m.enrolledFn(userID)
	}
	return nil, nil
}

func (m *mockMFAAuthenticator) VerifyFactor(_ context.Context, userID int64, method, code string, assertion []byte) ([]string, error) {
	if m.verifyFactorFn != nil {
		return m.verifyFactorFn(userID, method, code, assertion)
	}
	return []string{"pwd", "otp"}, nil
}

func (m *mockMFAAuthenticator) SendSMSChallenge(_ context.Context, userID int64) error {
	if m.sendSMSFn != nil {
		return m.sendSMSFn(userID)
	}
	return nil
}

func (m *mockMFAAuthenticator) SendEmailOTPChallenge(_ context.Context, _ int64) error {
	return nil
}

func (m *mockMFAAuthenticator) BeginWebAuthnLogin(_ context.Context, userID int64) (json.RawMessage, error) {
	if m.beginWAFn != nil {
		return m.beginWAFn(userID)
	}
	return json.RawMessage(`{}`), nil
}

func TestLoginMFAMethodAllowed(t *testing.T) {
	tests := []struct {
		name   string
		raw    any
		method string
		want   bool
	}{
		{name: "empty method rejected", raw: []any{"totp"}, method: "", want: false},
		{name: "listed method allowed", raw: []any{"totp", "webauthn"}, method: "webauthn", want: true},
		{name: "unlisted method rejected", raw: []any{"totp"}, method: "sms", want: false},
		{name: "non-list claim rejected", raw: "totp", method: "totp", want: false},
		{name: "nil claim rejected", raw: nil, method: "totp", want: false},
		{name: "non-string values ignored", raw: []any{123, "totp"}, method: "sms", want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, loginMFAMethodAllowed(tt.raw, tt.method))
		})
	}
}

func TestCompleteMFALogin_Guards(t *testing.T) {
	t.Run("nil authenticator is not configured", func(t *testing.T) {
		svc := &loginService{}
		_, err := svc.CompleteMFALogin(context.Background(), "tok", "totp", "123456", nil, nil, nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "MFA is not configured")
	})

	t.Run("invalid challenge token rejected", func(t *testing.T) {
		svc := &loginService{mfaAuthenticator: &mockMFAAuthenticator{}}
		_, err := svc.CompleteMFALogin(context.Background(), "not-a-jwt", "totp", "123456", nil, nil, nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid or expired MFA challenge")
	})
}

func TestCompleteMFALogin_PreservesMagicLinkAsPrimaryAMR(t *testing.T) {
	initTestJWTKeysService(t)
	userUUID := uuid.New()
	user := &User{UserID: 42, UserUUID: userUUID, Username: "magic-user", Status: shared.StatusActive}
	challenge, err := platformjwt.GenerateStepUpChallengeTokenForAuthMethodWithContext(
		context.Background(), userUUID.String(), time.Minute, platformjwt.AMRMagicLink, []string{"totp"},
	)
	require.NoError(t, err)
	clientID := "test-client"
	svc := &loginService{
		userRepo: &mockUserRepo{findByUUIDFn: func(any, ...string) (*User, error) { return user, nil }},
		clientRepo: &mockClientRepo{findByIdentifierFn: func(string) (*Client, error) {
			return buildActiveClient(), nil
		}},
		userIdentityRepo: &mockUserIdentityRepo{findByUserIDAndClientIDFn: func(_, _ int64) (*UserIdentity, error) {
			return &UserIdentity{Sub: "magic-sub"}, nil
		}},
		authEventService: &mockAuthEventService{},
		mfaAuthenticator: &mockMFAAuthenticator{verifyFactorFn: func(int64, string, string, []byte) ([]string, error) {
			return []string{platformjwt.AMRPassword, platformjwt.AMROTP}, nil
		}},
	}

	resp, err := svc.CompleteMFALogin(context.Background(), challenge, "totp", "123456", nil, &clientID, nil)

	require.NoError(t, err)
	claims, err := platformjwt.ValidateToken(resp.AccessToken)
	require.NoError(t, err)
	assert.Equal(t, []any{platformjwt.AMRMagicLink, platformjwt.AMROTP}, claims["amr"])
	assert.Equal(t, platformjwt.ACRLevel2, claims["acr"])
}

func TestCompleteMFALogin_CreatesMaintainerdIdentityWhenMissing(t *testing.T) {
	initTestJWTKeysService(t)
	userUUID := uuid.New()
	user := &User{UserID: 42, UserUUID: userUUID, Username: "mfa-user", Status: shared.StatusActive}
	challenge, err := platformjwt.GenerateStepUpChallengeTokenWithContext(context.Background(), userUUID.String(), time.Minute, []string{"webauthn"})
	require.NoError(t, err)
	clientID := "test-client"
	maintainerdIDP := &IdentityProvider{IdentityProviderID: 7, Provider: shared.IDPProviderMaintainerd, ProviderType: shared.IDPTypeSystem, IsSystem: true}
	googleIDP := &IdentityProvider{IdentityProviderID: 8, Provider: shared.IDPProviderGoogle}
	connections := []ClientIdentityProvider{
		{IdentityProviderID: 8, Enabled: true, IsDefault: true, IdentityProvider: googleIDP},
		{IdentityProviderID: 7, Enabled: true, IdentityProvider: maintainerdIDP},
	}
	client := buildActiveClient()
	client.ClientID = 99
	client.IdentityProviderID = 8
	client.IdentityProvider = googleIDP
	client.ConnectedProviders = &connections
	var created *UserIdentity
	svc := &loginService{
		userRepo: &mockUserRepo{findByUUIDFn: func(any, ...string) (*User, error) { return user, nil }},
		clientRepo: &mockClientRepo{findByIdentifierFn: func(string) (*Client, error) {
			return client, nil
		}},
		userIdentityRepo: &mockUserIdentityRepo{
			findByUserIDAndClientIDFn: func(_, _ int64) (*UserIdentity, error) { return nil, nil },
			createFn: func(identity *UserIdentity) (*UserIdentity, error) {
				created = identity
				return identity, nil
			},
		},
		authEventService: &mockAuthEventService{},
		mfaAuthenticator: &mockMFAAuthenticator{verifyFactorFn: func(int64, string, string, []byte) ([]string, error) {
			return []string{platformjwt.AMRPassword, "hwk"}, nil
		}},
	}

	resp, err := svc.CompleteMFALogin(context.Background(), challenge, "webauthn", "", []byte(`{}`), &clientID, nil)

	require.NoError(t, err)
	require.NotNil(t, created)
	assert.Equal(t, int64(1), created.TenantID)
	assert.Equal(t, user.UserID, created.UserID)
	// The identity is created against the client's connected identity provider,
	// never against the client itself — user_identities has no client_id.
	assert.Equal(t, int64(7), created.IdentityProviderID)
	assert.Equal(t, shared.ProviderMaintainerd, created.Provider)
	assert.NotEmpty(t, created.Sub)
	claims, err := platformjwt.ValidateToken(resp.AccessToken)
	require.NoError(t, err)
	assert.Equal(t, created.Sub, claims["sub"])
	assert.Equal(t, platformjwt.ACRLevel2, claims["acr"])
}

func TestSendMFALoginSMS_Guards(t *testing.T) {
	t.Run("nil authenticator", func(t *testing.T) {
		svc := &loginService{}
		err := svc.SendMFALoginSMS(context.Background(), "tok")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "MFA is not configured")
	})

	t.Run("invalid challenge token", func(t *testing.T) {
		svc := &loginService{mfaAuthenticator: &mockMFAAuthenticator{}}
		err := svc.SendMFALoginSMS(context.Background(), "not-a-jwt")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid or expired MFA challenge")
	})

	t.Run("challenge allowed methods must include sms", func(t *testing.T) {
		initTestJWTKeysService(t)
		userUUID := uuid.New()
		const userID int64 = 42
		challenge, err := platformjwt.GenerateStepUpChallengeTokenWithContext(context.Background(), userUUID.String(), time.Minute, []string{"totp"})
		require.NoError(t, err)
		svc := &loginService{
			userRepo:         &mockUserRepo{findByUUIDFn: func(any, ...string) (*User, error) { return &User{UserID: userID, UserUUID: userUUID}, nil }},
			mfaAuthenticator: &mockMFAAuthenticator{},
		}

		err = svc.SendMFALoginSMS(context.Background(), challenge)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "MFA method not allowed: sms")
	})
}

func TestBeginMFALoginWebAuthn_Guards(t *testing.T) {
	t.Run("invalid challenge token", func(t *testing.T) {
		svc := &loginService{mfaAuthenticator: &mockMFAAuthenticator{}}
		_, err := svc.BeginMFALoginWebAuthn(context.Background(), "not-a-jwt")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid or expired MFA challenge")
	})

	t.Run("challenge allowed methods must include webauthn", func(t *testing.T) {
		initTestJWTKeysService(t)
		userUUID := uuid.New()
		const userID int64 = 42
		challenge, err := platformjwt.GenerateStepUpChallengeTokenWithContext(context.Background(), userUUID.String(), time.Minute, []string{"totp"})
		require.NoError(t, err)
		svc := &loginService{
			userRepo:         &mockUserRepo{findByUUIDFn: func(any, ...string) (*User, error) { return &User{UserID: userID, UserUUID: userUUID}, nil }},
			mfaAuthenticator: &mockMFAAuthenticator{},
		}

		_, err = svc.BeginMFALoginWebAuthn(context.Background(), challenge)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "MFA method not allowed: webauthn")
	})
}

func TestSetMFAFactorAuthenticator(t *testing.T) {
	svc := &loginService{}
	svc.SetMFAFactorAuthenticator(&mockMFAAuthenticator{})
	assert.NotNil(t, svc.mfaAuthenticator)
}
