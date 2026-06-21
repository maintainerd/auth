package authn

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	platformjwt "github.com/maintainerd/auth/internal/platform/jwt"
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
