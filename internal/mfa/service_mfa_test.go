package mfa

import (
	"context"
	"errors"
	"io"
	"testing"
	"time"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/go-webauthn/webauthn/protocol"
	jwtlib "github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/notifier"
	"github.com/maintainerd/auth/internal/platform/apperror"
	"github.com/maintainerd/auth/internal/platform/config"
	"github.com/maintainerd/auth/internal/platform/crypto"
	authjwt "github.com/maintainerd/auth/internal/platform/jwt"
	"github.com/maintainerd/auth/internal/platform/middleware"
	"github.com/maintainerd/auth/internal/secpolicy"
	"github.com/pquerna/otp"
	"github.com/pquerna/otp/totp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

func TestValidateTOTPAndStep(t *testing.T) {
	secret := "JBSWY3DPEHPK3PXP"
	at := time.Unix(1_700_000_000, 0).UTC()
	code, err := totp.GenerateCodeCustom(secret, at, totp.ValidateOpts{
		Period:    totpPeriod,
		Skew:      1,
		Digits:    totpDigits,
		Algorithm: otp.AlgorithmSHA1,
	})
	require.NoError(t, err)

	step, ok, err := validateTOTPAndStep(code, secret, at)
	require.NoError(t, err)
	assert.True(t, ok)
	assert.Equal(t, at.Unix()/totpPeriod, step)
}

func TestValidateTOTPAndStepRejectsInvalidCode(t *testing.T) {
	step, ok, err := validateTOTPAndStep("000000", "JBSWY3DPEHPK3PXP", time.Unix(1_700_000_000, 0).UTC())
	require.NoError(t, err)
	assert.False(t, ok)
	assert.Zero(t, step)

	step, ok, err = validateTOTPAndStep("000000", "JBSWY3DPEHPK3PXP", time.Unix(-1, 0).UTC())
	require.NoError(t, err)
	assert.False(t, ok)
	assert.Zero(t, step)
}

func TestStepUpMethodAllowed(t *testing.T) {
	tests := []struct {
		name   string
		raw    any
		method string
		want   bool
	}{
		{name: "empty method rejected", raw: []any{"totp"}, method: "", want: false},
		{name: "non-list claim allows backward-compatible token", raw: "totp", method: "totp", want: true},
		{name: "missing claim allows backward-compatible token", raw: nil, method: "totp", want: true},
		{name: "listed method allowed", raw: []any{"totp", "backup_code"}, method: "backup_code", want: true},
		{name: "unlisted method rejected", raw: []any{"totp"}, method: "backup_code", want: false},
		{name: "non-string list values ignored", raw: []any{123, "totp"}, method: "backup_code", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, stepUpMethodAllowed(tt.raw, tt.method))
		})
	}
}

func TestMFAService_GetBackupCodesCount(t *testing.T) {
	tests := []struct {
		name    string
		codes   []UserBackupCode
		err     error
		want    int
		wantErr string
	}{
		{name: "counts unused codes", codes: []UserBackupCode{{BackupCodeID: 1}, {BackupCodeID: 2}}, want: 2},
		{name: "repo error", err: errors.New("db down"), wantErr: "backup code lookup failed"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &mfaService{backupCodeRepo: &mockBackupCodeRepo{findUnused: tt.codes, findUnusedErr: tt.err}}

			got, err := svc.GetBackupCodesCount(t.Context(), 42)

			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestMFAService_GetMFAStatus(t *testing.T) {
	enabledAt := time.Unix(1_700_000_000, 0).UTC()
	lastUsed := enabledAt.Add(time.Hour)
	credUUID := uuid.MustParse("00000000-0000-0000-0000-000000000099")

	tests := []struct {
		name       string
		user       *User
		userErr    error
		codes      []UserBackupCode
		creds      []UserWebAuthnCredential
		credsErr   error
		wantErr    string
		assertResp func(*testing.T, *MFAStatusResponseDTO)
	}{
		{
			name:  "success maps factors and timestamps",
			user:  &User{UserID: 42, IsTOTPEnabled: true, IsWebAuthnEnabled: true, MFAEnabledAt: &enabledAt},
			codes: []UserBackupCode{{BackupCodeID: 1}, {BackupCodeID: 2}},
			creds: []UserWebAuthnCredential{{
				CredentialUUID: credUUID,
				Name:           "Laptop",
				Transport:      "usb",
				LastUsedAt:     &lastUsed,
				CreatedAt:      enabledAt,
			}},
			assertResp: func(t *testing.T, got *MFAStatusResponseDTO) {
				t.Helper()
				assert.True(t, got.IsTOTPEnabled)
				assert.True(t, got.IsWebAuthnEnabled)
				assert.Equal(t, 2, got.BackupCodesCount)
				require.Len(t, got.WebAuthnKeys, 1)
				assert.Equal(t, credUUID.String(), got.WebAuthnKeys[0].CredentialUUID)
				assert.Equal(t, "Laptop", got.WebAuthnKeys[0].Name)
				require.NotNil(t, got.WebAuthnKeys[0].LastUsedAt)
				require.NotNil(t, got.MFAEnabledAt)
			},
		},
		{name: "missing user", user: nil, wantErr: "user not found"},
		{name: "user repo error", userErr: errors.New("db down"), wantErr: "user not found"},
		{name: "credential repo error", user: &User{UserID: 42}, credsErr: errors.New("db down"), wantErr: "credential lookup failed"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &mfaService{
				userRepo:         &mockUserRepo{findByID: tt.user, findByIDErr: tt.userErr},
				backupCodeRepo:   &mockBackupCodeRepo{findUnused: tt.codes},
				webAuthnCredRepo: &mockWebAuthnCredentialRepo{findByUserID: tt.creds, findByUserIDErr: tt.credsErr},
				smsPhoneRepo:     &mockSMSPhoneRepo{},
			}

			got, err := svc.GetMFAStatus(t.Context(), 42)

			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
				return
			}
			require.NoError(t, err)
			tt.assertResp(t, got)
		})
	}
}

func TestMFAService_GetMFAPolicy(t *testing.T) {
	tests := []struct {
		name    string
		setting *secpolicy.SecuritySetting
		err     error
		want    *MFAPolicyDTO
	}{
		{
			name:    "repo miss uses default policy",
			setting: nil,
			want:    &MFAPolicyDTO{Required: false, AllowedMethods: []string{"totp", "sms", "webauthn", "backup_code"}},
		},
		{
			name: "valid json policy",
			setting: &secpolicy.SecuritySetting{
				MFAConfig: datatypes.JSON([]byte(`{"required":true,"allowed_methods":["totp"]}`)),
			},
			want: &MFAPolicyDTO{Required: true, AllowedMethods: []string{"totp"}},
		},
		{
			name: "invalid json uses default policy",
			setting: &secpolicy.SecuritySetting{
				MFAConfig: datatypes.JSON([]byte(`{bad json`)),
			},
			want: &MFAPolicyDTO{Required: false, AllowedMethods: []string{"totp", "sms", "webauthn", "backup_code"}},
		},
		{
			name: "missing allowed methods uses default policy",
			setting: &secpolicy.SecuritySetting{
				MFAConfig: datatypes.JSON([]byte(`{"required":true}`)),
			},
			want: &MFAPolicyDTO{Required: false, AllowedMethods: []string{"totp", "sms", "webauthn", "backup_code"}},
		},
		{
			name: "repo error uses default policy",
			err:  errors.New("db down"),
			want: &MFAPolicyDTO{Required: false, AllowedMethods: []string{"totp", "sms", "webauthn", "backup_code"}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &mfaService{secSettingRepo: &mockSecuritySettingRepo{findByTenantID: tt.setting, findByTenantIDErr: tt.err}}

			got, err := svc.GetMFAPolicy(t.Context(), 7)

			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestMFAService_IsMFARequired(t *testing.T) {
	svc := &mfaService{secSettingRepo: &mockSecuritySettingRepo{
		findByTenantID: &secpolicy.SecuritySetting{
			MFAConfig: datatypes.JSON([]byte(`{"required":true,"allowed_methods":["totp"]}`)),
		},
	}}

	required, err := svc.IsMFARequired(t.Context(), 7)

	require.NoError(t, err)
	assert.True(t, required)
}

func TestMFAService_UserHasMFA(t *testing.T) {
	tests := []struct {
		name string
		user *User
		err  error
		want bool
	}{
		{name: "totp enabled", user: &User{IsTOTPEnabled: true}, want: true},
		{name: "webauthn enabled", user: &User{IsWebAuthnEnabled: true}, want: true},
		{name: "no factors", user: &User{}, want: false},
		{name: "missing user", user: nil, want: false},
		{name: "repo error", err: errors.New("db down"), want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &mfaService{userRepo: &mockUserRepo{findByID: tt.user, findByIDErr: tt.err}}

			got, err := svc.UserHasMFA(t.Context(), 42)

			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestNewMFAService(t *testing.T) {
	db, _ := newMockGormDB(t)
	svc := NewMFAService(db, &mockUserRepo{}, &mockTOTPSecretRepo{}, &mockWebAuthnCredentialRepo{}, &mockWebAuthnService{}, &mockBackupCodeRepo{}, &mockSMSPhoneRepo{}, &mockSMSOtpRepo{}, &mockSecuritySettingRepo{}, &mockAuthEventService{})
	require.NotNil(t, svc)
	assert.IsType(t, &mfaService{}, svc)
}

func TestMFAService_BeginTOTPEnrollment(t *testing.T) {
	originalKey := config.AppEncryptionKey
	config.AppEncryptionKey = []byte("12345678901234567890123456789012")
	t.Cleanup(func() { config.AppEncryptionKey = originalKey })

	tests := []struct {
		name    string
		user    *User
		userErr error
		upsert  error
		wantErr string
	}{
		{name: "success", user: &User{UserID: mfaTestUserID, Email: "user@example.com"}},
		{name: "missing user", wantErr: "user not found"},
		{name: "user repo error", userErr: errors.New("db down"), wantErr: "user not found"},
		{name: "upsert error", user: &User{UserID: mfaTestUserID, Email: "user@example.com"}, upsert: errors.New("db down"), wantErr: "failed to store TOTP secret"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &mfaService{
				userRepo: &mockUserRepo{findByID: tt.user, findByIDErr: tt.userErr},
				totpRepo: &mockTOTPSecretRepo{upsertErr: tt.upsert},
			}

			got, err := svc.BeginTOTPEnrollment(t.Context(), mfaTestUserID)

			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
				return
			}
			require.NoError(t, err)
			assert.NotEmpty(t, got.Secret)
			assert.Contains(t, got.QRCodeURL, "otpauth://totp/")
		})
	}

	t.Run("totp generation error", func(t *testing.T) {
		original := generateTOTPKey
		t.Cleanup(func() { generateTOTPKey = original })
		generateTOTPKey = func(totp.GenerateOpts) (*otp.Key, error) {
			return nil, errors.New("totp down")
		}
		svc := &mfaService{userRepo: &mockUserRepo{findByID: &User{UserID: mfaTestUserID, Email: "user@example.com"}}}

		_, err := svc.BeginTOTPEnrollment(t.Context(), mfaTestUserID)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "TOTP key generation failed")
	})

	t.Run("encryption error", func(t *testing.T) {
		originalKey := config.AppEncryptionKey
		config.AppEncryptionKey = []byte("bad-key")
		t.Cleanup(func() { config.AppEncryptionKey = originalKey })
		svc := &mfaService{
			userRepo: &mockUserRepo{findByID: &User{UserID: mfaTestUserID, Email: "user@example.com"}},
			totpRepo: &mockTOTPSecretRepo{},
		}

		_, err := svc.BeginTOTPEnrollment(t.Context(), mfaTestUserID)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to encrypt TOTP secret")
	})
}

func TestMFAService_FinishTOTPEnrollment(t *testing.T) {
	secret := "JBSWY3DPEHPK3PXP"
	code, err := totp.GenerateCode(secret, time.Now())
	require.NoError(t, err)

	t.Run("success enables TOTP updates user and creates backup codes", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		expectMFAUpdate(mock, "users").WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectCommit()
		events := &mockAuthEventService{}
		svc := &mfaService{
			db:               db,
			totpRepo:         &mockTOTPSecretRepo{findByUserID: &UserTOTPSecret{Secret: secret, IsEnabled: false}},
			backupCodeRepo:   &mockBackupCodeRepo{},
			authEventService: events,
		}

		got, err := svc.FinishTOTPEnrollment(t.Context(), mfaTestUserID, code)

		require.NoError(t, err)
		assert.Len(t, got, mfaBackupCodeCount)
		assert.Len(t, events.inputs, 1)
		assertExpectationsMet(t, mock)
	})

	tests := []struct {
		name    string
		record  *UserTOTPSecret
		repoErr error
		enable  error
		dbErr   bool
		wantErr string
	}{
		{name: "lookup error", repoErr: errors.New("db down"), wantErr: "TOTP secret lookup failed"},
		{name: "no pending record", wantErr: "no pending TOTP enrollment found"},
		{name: "already enabled", record: &UserTOTPSecret{Secret: secret, IsEnabled: true}, wantErr: "no pending TOTP enrollment found"},
		{name: "invalid code", record: &UserTOTPSecret{Secret: secret}, wantErr: "invalid TOTP code"},
		{name: "enable error", record: &UserTOTPSecret{Secret: secret}, enable: errors.New("db down"), wantErr: "failed to enable TOTP"},
		{name: "user update error", record: &UserTOTPSecret{Secret: secret}, dbErr: true, wantErr: "failed to update user MFA status"},
		{name: "backup code delete error", record: &UserTOTPSecret{Secret: secret}, wantErr: "failed to delete existing backup codes"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock := newMockGormDB(t)
			if tt.dbErr || tt.name == "backup code delete error" {
				mock.ExpectBegin()
				if tt.dbErr {
					expectMFAUpdate(mock, "users").WillReturnError(errors.New("db down"))
					mock.ExpectRollback()
				} else {
					expectMFAUpdate(mock, "users").WillReturnResult(sqlmock.NewResult(0, 1))
					mock.ExpectCommit()
				}
			}
			svc := &mfaService{
				db:               db,
				totpRepo:         &mockTOTPSecretRepo{findByUserID: tt.record, findByUserIDErr: tt.repoErr, enableErr: tt.enable},
				backupCodeRepo:   &mockBackupCodeRepo{deleteAllErr: map[bool]error{true: errors.New("db down")}[tt.name == "backup code delete error"]},
				authEventService: &mockAuthEventService{},
			}
			useCode := code
			if tt.name == "invalid code" {
				useCode = "000000"
			}

			_, err := svc.FinishTOTPEnrollment(t.Context(), mfaTestUserID, useCode)

			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.wantErr)
			assertExpectationsMet(t, mock)
		})
	}
}

func TestMFAService_VerifyTOTP(t *testing.T) {
	secret := "JBSWY3DPEHPK3PXP"
	code, err := totp.GenerateCode(secret, time.Now())
	require.NoError(t, err)

	tests := []struct {
		name     string
		record   *UserTOTPSecret
		repoErr  error
		accepted bool
		markErr  error
		code     string
		want     bool
		wantErr  string
	}{
		{name: "valid code accepted", record: &UserTOTPSecret{Secret: secret, IsEnabled: true}, accepted: true, code: code, want: true},
		{name: "replay rejected without error", record: &UserTOTPSecret{Secret: secret, IsEnabled: true}, accepted: false, code: code, want: false},
		{name: "invalid code", record: &UserTOTPSecret{Secret: secret, IsEnabled: true}, code: "000000", want: false},
		{name: "lookup error", repoErr: errors.New("db down"), wantErr: "TOTP lookup failed"},
		{name: "not enabled", record: &UserTOTPSecret{Secret: secret, IsEnabled: false}, wantErr: "TOTP is not enabled"},
		{name: "mark step error", record: &UserTOTPSecret{Secret: secret, IsEnabled: true}, code: code, markErr: errors.New("db down"), wantErr: "failed to update TOTP last-used step"},
		{name: "invalid secret", record: &UserTOTPSecret{Secret: "%", IsEnabled: true}, code: code, wantErr: "invalid TOTP code"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &mfaService{totpRepo: &mockTOTPSecretRepo{findByUserID: tt.record, findByUserIDErr: tt.repoErr, markAccepted: tt.accepted, markStepErr: tt.markErr}}

			got, err := svc.VerifyTOTP(t.Context(), mfaTestUserID, tt.code)

			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}

	t.Run("rate limited", func(t *testing.T) {
		original := checkMFARateLimit
		t.Cleanup(func() { checkMFARateLimit = original })
		checkMFARateLimit = func(string) error { return errors.New("locked") }
		svc := &mfaService{totpRepo: &mockTOTPSecretRepo{findByUserID: &UserTOTPSecret{Secret: secret, IsEnabled: true}}}

		_, err := svc.VerifyTOTP(t.Context(), mfaTestUserID, code)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "too many attempts")
	})
}

func TestMFAService_SyncMFAState(t *testing.T) {
	t.Run("no-op while a primary factor remains", func(t *testing.T) {
		// No DB ops expected: a TOTP factor is still active.
		db, mock := newMockGormDB(t)
		svc := &mfaService{
			db:             db,
			userRepo:       &mockUserRepo{findByID: &User{UserID: mfaTestUserID, IsTOTPEnabled: true}},
			backupCodeRepo: &mockBackupCodeRepo{deleteAllErr: errors.New("must not be called")},
			smsPhoneRepo:   &mockSMSPhoneRepo{},
		}

		require.NoError(t, svc.SyncMFAState(t.Context(), mfaTestUserID))
		assertExpectationsMet(t, mock)
	})

	t.Run("purges backup codes and clears mfa_enabled_at when no factor remains", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		expectMFAUpdate(mock, "users").WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectCommit()
		svc := &mfaService{
			db:             db,
			userRepo:       &mockUserRepo{findByID: &User{UserID: mfaTestUserID}},
			backupCodeRepo: &mockBackupCodeRepo{},
			smsPhoneRepo:   &mockSMSPhoneRepo{},
		}

		require.NoError(t, svc.SyncMFAState(t.Context(), mfaTestUserID))
		assertExpectationsMet(t, mock)
	})

	t.Run("backup code delete error surfaces", func(t *testing.T) {
		svc := &mfaService{
			userRepo:       &mockUserRepo{findByID: &User{UserID: mfaTestUserID}},
			backupCodeRepo: &mockBackupCodeRepo{deleteAllErr: errors.New("db down")},
			smsPhoneRepo:   &mockSMSPhoneRepo{},
		}

		err := svc.SyncMFAState(t.Context(), mfaTestUserID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to delete backup codes")
	})
}

func TestMFAService_EnrolledMFAMethods(t *testing.T) {
	t.Run("returns all active factors in canonical order", func(t *testing.T) {
		svc := &mfaService{
			userRepo:       &mockUserRepo{findByID: &User{UserID: mfaTestUserID, IsTOTPEnabled: true, IsWebAuthnEnabled: true}},
			smsPhoneRepo:   &mockSMSPhoneRepo{findByUserID: &UserSMSPhone{Phone: "+15550001111", IsVerified: true}},
			backupCodeRepo: &mockBackupCodeRepo{findUnused: []UserBackupCode{{BackupCodeID: 1}}},
		}
		got, err := svc.EnrolledMFAMethods(t.Context(), mfaTestUserID)
		require.NoError(t, err)
		assert.Equal(t, []string{"totp", "webauthn", "sms", "backup_code"}, got)
	})

	t.Run("no active factor returns empty", func(t *testing.T) {
		svc := &mfaService{
			userRepo:       &mockUserRepo{findByID: &User{UserID: mfaTestUserID}},
			smsPhoneRepo:   &mockSMSPhoneRepo{},
			backupCodeRepo: &mockBackupCodeRepo{},
		}
		got, err := svc.EnrolledMFAMethods(t.Context(), mfaTestUserID)
		require.NoError(t, err)
		assert.Empty(t, got)
	})
}

func TestMFAService_VerifyFactorSMS(t *testing.T) {
	originalRL := checkMFARateLimit
	t.Cleanup(func() { checkMFARateLimit = originalRL })
	checkMFARateLimit = func(string) error { return nil }

	t.Run("verifies against the MFA phone record, not users.phone", func(t *testing.T) {
		svc := &mfaService{
			smsPhoneRepo: &mockSMSPhoneRepo{findByUserID: &UserSMSPhone{Phone: "+15550001111", IsVerified: true}},
			smsOtpRepo:   &mockSMSOtpRepo{findValid: &notifier.UserOTP{UserOTPID: 1, OTPHash: crypto.HashAuthorizationCode("123456")}},
		}
		amr, err := svc.verifyFactor(t.Context(), mfaTestUserID, "sms", "123456", nil)
		require.NoError(t, err)
		assert.Equal(t, []string{"pwd", "sms"}, amr)
	})

	t.Run("no verified MFA phone is rejected", func(t *testing.T) {
		svc := &mfaService{smsPhoneRepo: &mockSMSPhoneRepo{findByUserID: nil}}
		_, err := svc.verifyFactor(t.Context(), mfaTestUserID, "sms", "123456", nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "no verified phone on file")
	})

	t.Run("invalid code is rejected", func(t *testing.T) {
		svc := &mfaService{
			smsPhoneRepo: &mockSMSPhoneRepo{findByUserID: &UserSMSPhone{Phone: "+15550001111", IsVerified: true}},
			smsOtpRepo:   &mockSMSOtpRepo{findValid: &notifier.UserOTP{UserOTPID: 1, OTPHash: crypto.HashAuthorizationCode("999999")}},
		}
		_, err := svc.verifyFactor(t.Context(), mfaTestUserID, "sms", "123456", nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid SMS code")
	})
}

func TestMFAService_DisableAndRegenerateBackupCodes(t *testing.T) {
	t.Run("disable success", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		// 1) is_totp_enabled = false
		mock.ExpectBegin()
		expectMFAUpdate(mock, "users").WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectCommit()
		// 2) SyncMFAState clears mfa_enabled_at when no primary factor remains.
		mock.ExpectBegin()
		expectMFAUpdate(mock, "users").WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectCommit()
		events := &mockAuthEventService{}
		svc := &mfaService{
			db:               db,
			userRepo:         &mockUserRepo{findByID: &User{UserID: mfaTestUserID}},
			totpRepo:         &mockTOTPSecretRepo{},
			backupCodeRepo:   &mockBackupCodeRepo{},
			smsPhoneRepo:     &mockSMSPhoneRepo{},
			authEventService: events,
		}

		require.NoError(t, svc.DisableTOTP(t.Context(), mfaTestUserID))

		assert.Len(t, events.inputs, 1)
		assertExpectationsMet(t, mock)
	})

	for _, tt := range []struct {
		name    string
		totpErr error
		codeErr error
		dbErr   bool
		wantErr string
	}{
		{name: "disable repo error", totpErr: errors.New("db down"), wantErr: "failed to disable TOTP"},
		{name: "delete codes error", codeErr: errors.New("db down"), wantErr: "failed to delete backup codes"},
		{name: "update user error", dbErr: true, wantErr: "failed to update user TOTP state"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			db, mock := newMockGormDB(t)
			if tt.dbErr {
				mock.ExpectBegin()
				expectMFAUpdate(mock, "users").WillReturnError(errors.New("db down"))
				mock.ExpectRollback()
			}
			svc := &mfaService{db: db, totpRepo: &mockTOTPSecretRepo{disableErr: tt.totpErr}, backupCodeRepo: &mockBackupCodeRepo{deleteAllErr: tt.codeErr}, authEventService: &mockAuthEventService{}}

			err := svc.DisableTOTP(t.Context(), mfaTestUserID)

			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.wantErr)
			assertExpectationsMet(t, mock)
		})
	}

	t.Run("regenerate success and generator error", func(t *testing.T) {
		original := generateBackupCodeString
		t.Cleanup(func() { generateBackupCodeString = original })
		generateBackupCodeString = func(int) (string, error) { return "backup-code", nil }
		svc := &mfaService{backupCodeRepo: &mockBackupCodeRepo{}}
		got, err := svc.RegenerateBackupCodes(t.Context(), mfaTestUserID)
		require.NoError(t, err)
		assert.Len(t, got, mfaBackupCodeCount)

		generateBackupCodeString = func(int) (string, error) { return "", errors.New("rng down") }
		_, err = svc.RegenerateBackupCodes(t.Context(), mfaTestUserID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "backup code generation failed")
	})

	t.Run("regenerate hash error", func(t *testing.T) {
		original := hashBackupCodePassword
		t.Cleanup(func() { hashBackupCodePassword = original })
		hashBackupCodePassword = func([]byte, int) ([]byte, error) { return nil, errors.New("bcrypt down") }

		_, err := (&mfaService{backupCodeRepo: &mockBackupCodeRepo{}}).RegenerateBackupCodes(t.Context(), mfaTestUserID)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "backup code hashing failed")
	})

	t.Run("regenerate delete and storage errors", func(t *testing.T) {
		_, err := (&mfaService{backupCodeRepo: &mockBackupCodeRepo{deleteAllErr: errors.New("db down")}}).RegenerateBackupCodes(t.Context(), mfaTestUserID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to delete existing backup codes")

		_, err = (&mfaService{backupCodeRepo: &mockBackupCodeRepo{createBulkErr: errors.New("db down")}}).RegenerateBackupCodes(t.Context(), mfaTestUserID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "backup code storage failed")
	})
}

func TestMFAService_AdminResetMFA(t *testing.T) {
	t.Run("success clears all factors", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		expectMFAUpdate(mock, "users").WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectCommit()
		events := &mockAuthEventService{}
		svc := &mfaService{
			db:               db,
			userRepo:         &mockUserRepo{findByUUID: &User{UserID: mfaTestUserID}},
			totpRepo:         &mockTOTPSecretRepo{},
			backupCodeRepo:   &mockBackupCodeRepo{},
			webAuthnCredRepo: &mockWebAuthnCredentialRepo{},
			smsPhoneRepo:     &mockSMSPhoneRepo{},
			authEventService: events,
		}

		require.NoError(t, svc.AdminResetMFA(t.Context(), mfaTestUserUUID.String(), 99))

		assert.Len(t, events.inputs, 1)
		assertExpectationsMet(t, mock)
	})

	tests := []struct {
		name    string
		user    *User
		userErr error
		totpErr error
		codeErr error
		webErr  error
		dbErr   bool
		wantErr string
	}{
		{name: "missing user", wantErr: "target user not found"},
		{name: "user error", userErr: errors.New("db down"), wantErr: "target user not found"},
		{name: "totp error", user: &User{UserID: mfaTestUserID}, totpErr: errors.New("db down"), wantErr: "failed to disable target TOTP"},
		{name: "backup code error", user: &User{UserID: mfaTestUserID}, codeErr: errors.New("db down"), wantErr: "failed to delete target backup codes"},
		{name: "webauthn error", user: &User{UserID: mfaTestUserID}, webErr: errors.New("db down"), wantErr: "failed to delete target WebAuthn credentials"},
		{name: "db update error", user: &User{UserID: mfaTestUserID}, dbErr: true, wantErr: "failed to reset user MFA status"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock := newMockGormDB(t)
			if tt.dbErr {
				mock.ExpectBegin()
				expectMFAUpdate(mock, "users").WillReturnError(errors.New("db down"))
				mock.ExpectRollback()
			}
			svc := &mfaService{
				db:               db,
				userRepo:         &mockUserRepo{findByUUID: tt.user, findUUIDErr: tt.userErr},
				totpRepo:         &mockTOTPSecretRepo{disableErr: tt.totpErr},
				backupCodeRepo:   &mockBackupCodeRepo{deleteAllErr: tt.codeErr},
				webAuthnCredRepo: &mockWebAuthnCredentialRepo{deleteAllErr: tt.webErr},
				smsPhoneRepo:     &mockSMSPhoneRepo{},
				authEventService: &mockAuthEventService{},
			}

			err := svc.AdminResetMFA(t.Context(), mfaTestUserUUID.String(), 99)

			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.wantErr)
			assertExpectationsMet(t, mock)
		})
	}
}

func TestMFAService_AdminResetMFAMethod(t *testing.T) {
	t.Run("success resets a single factor and syncs state", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		// 1) is_totp_enabled = false
		mock.ExpectBegin()
		expectMFAUpdate(mock, "users").WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectCommit()
		// 2) SyncMFAState clears mfa_enabled_at when no primary factor remains.
		mock.ExpectBegin()
		expectMFAUpdate(mock, "users").WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectCommit()
		events := &mockAuthEventService{}
		svc := &mfaService{
			db:               db,
			userRepo:         &mockUserRepo{findByUUID: &User{UserID: mfaTestUserID}, findByID: &User{UserID: mfaTestUserID}},
			totpRepo:         &mockTOTPSecretRepo{},
			backupCodeRepo:   &mockBackupCodeRepo{},
			webAuthnCredRepo: &mockWebAuthnCredentialRepo{},
			smsPhoneRepo:     &mockSMSPhoneRepo{},
			authEventService: events,
		}

		require.NoError(t, svc.AdminResetMFAMethod(t.Context(), mfaTestUserUUID.String(), "totp", 99))

		assert.Len(t, events.inputs, 1)
		assertExpectationsMet(t, mock)
	})

	t.Run("missing user", func(t *testing.T) {
		svc := &mfaService{userRepo: &mockUserRepo{findByUUID: nil}}
		err := svc.AdminResetMFAMethod(t.Context(), mfaTestUserUUID.String(), "totp", 99)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "target user not found")
	})

	t.Run("unsupported method", func(t *testing.T) {
		svc := &mfaService{userRepo: &mockUserRepo{findByUUID: &User{UserID: mfaTestUserID}}}
		err := svc.AdminResetMFAMethod(t.Context(), mfaTestUserUUID.String(), "carrier-pigeon", 99)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "unsupported MFA method")
	})

	tests := []struct {
		name    string
		method  string
		totpErr error
		webErr  error
		smsErr  error
		codeErr error
		wantErr string
	}{
		{name: "totp error", method: "totp", totpErr: errors.New("db down"), wantErr: "failed to disable target TOTP"},
		{name: "webauthn error", method: "webauthn", webErr: errors.New("db down"), wantErr: "failed to delete target WebAuthn credentials"},
		{name: "sms error", method: "sms", smsErr: errors.New("db down"), wantErr: "failed to delete target SMS phone"},
		{name: "backup code error", method: "backup_code", codeErr: errors.New("db down"), wantErr: "failed to delete target backup codes"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &mfaService{
				userRepo:         &mockUserRepo{findByUUID: &User{UserID: mfaTestUserID}},
				totpRepo:         &mockTOTPSecretRepo{disableErr: tt.totpErr},
				webAuthnCredRepo: &mockWebAuthnCredentialRepo{deleteAllErr: tt.webErr},
				smsPhoneRepo:     &mockSMSPhoneRepo{deleteErr: tt.smsErr},
				backupCodeRepo:   &mockBackupCodeRepo{deleteAllErr: tt.codeErr},
				authEventService: &mockAuthEventService{},
			}
			err := svc.AdminResetMFAMethod(t.Context(), mfaTestUserUUID.String(), tt.method, 99)
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.wantErr)
		})
	}
}

func TestMFAService_SelfResetMFA(t *testing.T) {
	t.Run("success clears all own factors", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		expectMFAUpdate(mock, "users").WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectCommit()
		events := &mockAuthEventService{}
		svc := &mfaService{
			db:               db,
			totpRepo:         &mockTOTPSecretRepo{},
			backupCodeRepo:   &mockBackupCodeRepo{},
			webAuthnCredRepo: &mockWebAuthnCredentialRepo{},
			smsPhoneRepo:     &mockSMSPhoneRepo{},
			authEventService: events,
		}

		require.NoError(t, svc.SelfResetMFA(t.Context(), mfaTestUserID))

		assert.Len(t, events.inputs, 1)
		assertExpectationsMet(t, mock)
	})

	tests := []struct {
		name    string
		totpErr error
		codeErr error
		webErr  error
		smsErr  error
		dbErr   bool
		wantErr string
	}{
		{name: "totp error", totpErr: errors.New("db down"), wantErr: "failed to disable TOTP"},
		{name: "backup code error", codeErr: errors.New("db down"), wantErr: "failed to delete backup codes"},
		{name: "webauthn error", webErr: errors.New("db down"), wantErr: "failed to delete WebAuthn credentials"},
		{name: "sms error", smsErr: errors.New("db down"), wantErr: "failed to delete SMS phone"},
		{name: "db update error", dbErr: true, wantErr: "failed to reset MFA status"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock := newMockGormDB(t)
			if tt.dbErr {
				mock.ExpectBegin()
				expectMFAUpdate(mock, "users").WillReturnError(errors.New("db down"))
				mock.ExpectRollback()
			}
			svc := &mfaService{
				db:               db,
				totpRepo:         &mockTOTPSecretRepo{disableErr: tt.totpErr},
				backupCodeRepo:   &mockBackupCodeRepo{deleteAllErr: tt.codeErr},
				webAuthnCredRepo: &mockWebAuthnCredentialRepo{deleteAllErr: tt.webErr},
				smsPhoneRepo:     &mockSMSPhoneRepo{deleteErr: tt.smsErr},
				authEventService: &mockAuthEventService{},
			}

			err := svc.SelfResetMFA(t.Context(), mfaTestUserID)

			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.wantErr)
			assertExpectationsMet(t, mock)
		})
	}
}

func TestMFAService_StepUp(t *testing.T) {
	t.Run("issue challenge success and error", func(t *testing.T) {
		original := generateStepUpChallengeToken
		t.Cleanup(func() { generateStepUpChallengeToken = original })
		generateStepUpChallengeToken = func(_ context.Context, userUUID string, ttl time.Duration, allowedMethods ...[]string) (string, error) {
			assert.Equal(t, mfaTestUserUUID.String(), userUUID)
			assert.Equal(t, stepUpChallengeTTL, ttl)
			assert.Equal(t, []string{"totp"}, allowedMethods[0])
			return "challenge", nil
		}
		got, err := (&mfaService{}).IssueStepUpChallenge(t.Context(), mfaTestUserUUID.String(), []string{"totp"})
		require.NoError(t, err)
		assert.Equal(t, "challenge", got.ChallengeToken)

		generateStepUpChallengeToken = func(context.Context, string, time.Duration, ...[]string) (string, error) {
			return "", errors.New("jwt down")
		}
		_, err = (&mfaService{}).IssueStepUpChallenge(t.Context(), mfaTestUserUUID.String(), []string{"totp"})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "step-up challenge generation failed")
	})

	t.Run("verify backup code success and errors", func(t *testing.T) {
		hash, err := bcrypt.GenerateFromPassword([]byte("backup-code"), bcrypt.DefaultCost)
		require.NoError(t, err)
		originalValidate := validateStepUpChallengeToken
		originalAccess := generateStepUpAccessToken
		t.Cleanup(func() {
			validateStepUpChallengeToken = originalValidate
			generateStepUpAccessToken = originalAccess
		})
		validateStepUpChallengeToken = func(string) (jwtlib.MapClaims, error) {
			return jwtlib.MapClaims{"sub": mfaTestUserUUID.String(), "allowed_methods": []any{"backup_code"}}, nil
		}
		generateStepUpAccessToken = func(context.Context, string, string, string, string, string, string, *authjwt.AccessTokenOptions) (string, error) {
			return "access", nil
		}
		events := &mockAuthEventService{}
		svc := &mfaService{
			userRepo:         &mockUserRepo{findByUUID: &User{UserID: mfaTestUserID}, findByID: &User{UserID: mfaTestUserID, UserUUID: mfaTestUserUUID}},
			backupCodeRepo:   &mockBackupCodeRepo{findUnused: []UserBackupCode{{BackupCodeID: 1, CodeHash: string(hash)}}},
			authEventService: events,
		}

		got, err := svc.VerifyStepUp(t.Context(), StepUpVerifyRequestDTO{ChallengeToken: "challenge", Method: "backup_code", Code: "backup-code"}, mfaTestUserID)
		require.NoError(t, err)
		assert.Equal(t, "access", got.AccessToken)
		assert.Len(t, events.inputs, 1)

		validateStepUpChallengeToken = func(string) (jwtlib.MapClaims, error) { return nil, errors.New("bad token") }
		_, err = svc.VerifyStepUp(t.Context(), StepUpVerifyRequestDTO{ChallengeToken: "bad", Method: "backup_code"}, mfaTestUserID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid or expired")

		validateStepUpChallengeToken = func(string) (jwtlib.MapClaims, error) { return jwtlib.MapClaims{}, nil }
		_, err = svc.VerifyStepUp(t.Context(), StepUpVerifyRequestDTO{ChallengeToken: "challenge", Method: "backup_code"}, mfaTestUserID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "missing sub")
	})

	t.Run("verify step-up rejects subject and method problems", func(t *testing.T) {
		originalValidate := validateStepUpChallengeToken
		t.Cleanup(func() { validateStepUpChallengeToken = originalValidate })
		validateStepUpChallengeToken = func(string) (jwtlib.MapClaims, error) {
			return jwtlib.MapClaims{"sub": mfaTestUserUUID.String(), "allowed_methods": []any{"totp"}}, nil
		}

		svc := &mfaService{userRepo: &mockUserRepo{findByUUID: nil}}
		_, err := svc.VerifyStepUp(t.Context(), StepUpVerifyRequestDTO{ChallengeToken: "challenge", Method: "totp"}, mfaTestUserID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "unknown user")

		svc.userRepo = &mockUserRepo{findByUUID: &User{UserID: 99}}
		_, err = svc.VerifyStepUp(t.Context(), StepUpVerifyRequestDTO{ChallengeToken: "challenge", Method: "totp"}, mfaTestUserID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "does not match")

		svc.userRepo = &mockUserRepo{findByUUID: &User{UserID: mfaTestUserID}}
		_, err = svc.VerifyStepUp(t.Context(), StepUpVerifyRequestDTO{ChallengeToken: "challenge", Method: "backup_code"}, mfaTestUserID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "method not allowed")
	})

	t.Run("verify totp path success and failure", func(t *testing.T) {
		secret := "JBSWY3DPEHPK3PXP"
		code, err := totp.GenerateCode(secret, time.Now())
		require.NoError(t, err)
		originalValidate := validateStepUpChallengeToken
		originalAccess := generateStepUpAccessToken
		t.Cleanup(func() {
			validateStepUpChallengeToken = originalValidate
			generateStepUpAccessToken = originalAccess
		})
		validateStepUpChallengeToken = func(string) (jwtlib.MapClaims, error) {
			return jwtlib.MapClaims{"sub": mfaTestUserUUID.String(), "allowed_methods": []any{"totp"}}, nil
		}
		generateStepUpAccessToken = func(context.Context, string, string, string, string, string, string, *authjwt.AccessTokenOptions) (string, error) {
			return "access", nil
		}
		svc := &mfaService{
			userRepo:         &mockUserRepo{findByUUID: &User{UserID: mfaTestUserID}, findByID: &User{UserID: mfaTestUserID, UserUUID: mfaTestUserUUID}},
			totpRepo:         &mockTOTPSecretRepo{findByUserID: &UserTOTPSecret{Secret: secret, IsEnabled: true}, markAccepted: true},
			authEventService: &mockAuthEventService{},
		}
		got, err := svc.VerifyStepUp(t.Context(), StepUpVerifyRequestDTO{ChallengeToken: "challenge", Method: "totp", Code: code}, mfaTestUserID)
		require.NoError(t, err)
		assert.Equal(t, "access", got.AccessToken)

		_, err = svc.VerifyStepUp(t.Context(), StepUpVerifyRequestDTO{ChallengeToken: "challenge", Method: "totp", Code: "000000"}, mfaTestUserID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid TOTP code")
	})

	t.Run("verify backup code error paths after allowed method", func(t *testing.T) {
		originalValidate := validateStepUpChallengeToken
		t.Cleanup(func() { validateStepUpChallengeToken = originalValidate })
		validateStepUpChallengeToken = func(string) (jwtlib.MapClaims, error) {
			return jwtlib.MapClaims{"sub": mfaTestUserUUID.String(), "allowed_methods": []any{"backup_code"}}, nil
		}
		svc := &mfaService{
			userRepo:       &mockUserRepo{findByUUID: &User{UserID: mfaTestUserID}},
			backupCodeRepo: &mockBackupCodeRepo{findUnusedErr: errors.New("db down")},
		}
		_, err := svc.VerifyStepUp(t.Context(), StepUpVerifyRequestDTO{ChallengeToken: "challenge", Method: "backup_code", Code: "bad"}, mfaTestUserID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "backup code lookup failed")

		svc.backupCodeRepo = &mockBackupCodeRepo{findUnused: []UserBackupCode{{BackupCodeID: 1, CodeHash: "bad-hash"}}}
		_, err = svc.VerifyStepUp(t.Context(), StepUpVerifyRequestDTO{ChallengeToken: "challenge", Method: "backup_code", Code: "bad"}, mfaTestUserID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid backup code")

		hash, hashErr := bcrypt.GenerateFromPassword([]byte("backup-code"), bcrypt.DefaultCost)
		require.NoError(t, hashErr)
		svc.backupCodeRepo = &mockBackupCodeRepo{findUnused: []UserBackupCode{{BackupCodeID: 1, CodeHash: string(hash)}}, markUsedErr: errors.New("db down")}
		_, err = svc.VerifyStepUp(t.Context(), StepUpVerifyRequestDTO{ChallengeToken: "challenge", Method: "backup_code", Code: "backup-code"}, mfaTestUserID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to mark backup code used")
	})

	t.Run("verify backup code rate limited", func(t *testing.T) {
		originalValidate := validateStepUpChallengeToken
		originalRateLimit := checkMFARateLimit
		t.Cleanup(func() {
			validateStepUpChallengeToken = originalValidate
			checkMFARateLimit = originalRateLimit
		})
		validateStepUpChallengeToken = func(string) (jwtlib.MapClaims, error) {
			return jwtlib.MapClaims{"sub": mfaTestUserUUID.String(), "allowed_methods": []any{"backup_code"}}, nil
		}
		checkMFARateLimit = func(string) error { return errors.New("locked") }
		svc := &mfaService{userRepo: &mockUserRepo{findByUUID: &User{UserID: mfaTestUserID}}}

		_, err := svc.VerifyStepUp(t.Context(), StepUpVerifyRequestDTO{ChallengeToken: "challenge", Method: "backup_code", Code: "backup-code"}, mfaTestUserID)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "too many attempts")
	})

	t.Run("verify final user lookup and token generation errors", func(t *testing.T) {
		hash, err := bcrypt.GenerateFromPassword([]byte("backup-code"), bcrypt.DefaultCost)
		require.NoError(t, err)
		originalValidate := validateStepUpChallengeToken
		originalAccess := generateStepUpAccessToken
		t.Cleanup(func() {
			validateStepUpChallengeToken = originalValidate
			generateStepUpAccessToken = originalAccess
		})
		validateStepUpChallengeToken = func(string) (jwtlib.MapClaims, error) {
			return jwtlib.MapClaims{"sub": mfaTestUserUUID.String(), "allowed_methods": []any{"backup_code"}}, nil
		}
		svc := &mfaService{
			userRepo:       &mockUserRepo{findByUUID: &User{UserID: mfaTestUserID}, findByIDErr: errors.New("db down")},
			backupCodeRepo: &mockBackupCodeRepo{findUnused: []UserBackupCode{{BackupCodeID: 1, CodeHash: string(hash)}}},
		}
		_, err = svc.VerifyStepUp(t.Context(), StepUpVerifyRequestDTO{ChallengeToken: "challenge", Method: "backup_code", Code: "backup-code"}, mfaTestUserID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "user lookup failed")

		svc.userRepo = &mockUserRepo{findByUUID: &User{UserID: mfaTestUserID}, findByID: &User{UserID: mfaTestUserID, UserUUID: mfaTestUserUUID}}
		generateStepUpAccessToken = func(context.Context, string, string, string, string, string, string, *authjwt.AccessTokenOptions) (string, error) {
			return "", errors.New("jwt down")
		}
		_, err = svc.VerifyStepUp(t.Context(), StepUpVerifyRequestDTO{ChallengeToken: "challenge", Method: "backup_code", Code: "backup-code"}, mfaTestUserID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "token generation failed")

		_, err = svc.VerifyStepUp(t.Context(), StepUpVerifyRequestDTO{ChallengeToken: "challenge", Method: "sms", Code: "123456"}, mfaTestUserID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "method not allowed")
	})

	t.Run("verify unsupported method when token allows legacy methods", func(t *testing.T) {
		originalValidate := validateStepUpChallengeToken
		t.Cleanup(func() { validateStepUpChallengeToken = originalValidate })
		validateStepUpChallengeToken = func(string) (jwtlib.MapClaims, error) {
			return jwtlib.MapClaims{"sub": mfaTestUserUUID.String()}, nil
		}
		svc := &mfaService{userRepo: &mockUserRepo{findByUUID: &User{UserID: mfaTestUserID}}}

		_, err := svc.VerifyStepUp(t.Context(), StepUpVerifyRequestDTO{ChallengeToken: "challenge", Method: "xyz", Code: "123456"}, mfaTestUserID)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "unsupported MFA method")
	})

	// Regression: the access-token generator rejects empty client_id/provider_id,
	// so the elevated token must inherit them from the caller's session claims.
	// Without this, every step-up 500s and acr=2 is unreachable.
	t.Run("verify forwards session client/provider/scope into elevated token", func(t *testing.T) {
		secret := "JBSWY3DPEHPK3PXP"
		code, err := totp.GenerateCode(secret, time.Now())
		require.NoError(t, err)
		originalValidate := validateStepUpChallengeToken
		originalAccess := generateStepUpAccessToken
		t.Cleanup(func() {
			validateStepUpChallengeToken = originalValidate
			generateStepUpAccessToken = originalAccess
		})
		validateStepUpChallengeToken = func(string) (jwtlib.MapClaims, error) {
			return jwtlib.MapClaims{"sub": mfaTestUserUUID.String(), "allowed_methods": []any{"totp"}}, nil
		}
		var gotSub, gotClient, gotProvider, gotScope, gotSession string
		generateStepUpAccessToken = func(_ context.Context, sub, scope, _, _, clientID, providerID string, opts *authjwt.AccessTokenOptions) (string, error) {
			gotSub, gotClient, gotProvider, gotScope = sub, clientID, providerID, scope
			if opts != nil {
				gotSession = opts.SessionID
			}
			return "access", nil
		}
		svc := &mfaService{
			userRepo:         &mockUserRepo{findByUUID: &User{UserID: mfaTestUserID}, findByID: &User{UserID: mfaTestUserID, UserUUID: mfaTestUserUUID}},
			totpRepo:         &mockTOTPSecretRepo{findByUserID: &UserTOTPSecret{Secret: secret, IsEnabled: true}, markAccepted: true},
			authEventService: &mockAuthEventService{},
		}

		// The original session subject (user_identities.sub) is NOT the user UUID;
		// the elevated token must keep it so UserContextMiddleware resolves the user.
		ctx := middleware.ContextWithJWTClaims(t.Context(), &middleware.JWTClaims{
			Sub:        "identity-sub-001",
			ClientID:   "client-123",
			ProviderID: "provider-456",
			Scope:      "openid profile",
			SessionID:  "session-789",
		})
		_, err = svc.VerifyStepUp(ctx, StepUpVerifyRequestDTO{ChallengeToken: "challenge", Method: "totp", Code: code}, mfaTestUserID)
		require.NoError(t, err)
		assert.Equal(t, "identity-sub-001", gotSub)
		assert.NotEqual(t, mfaTestUserUUID.String(), gotSub, "elevated token must keep the original sub, not the user UUID")
		assert.Equal(t, "client-123", gotClient)
		assert.Equal(t, "provider-456", gotProvider)
		assert.Equal(t, "openid profile", gotScope)
		assert.Equal(t, "session-789", gotSession)
	})

	t.Run("verify webauthn step-up success and amr mapping", func(t *testing.T) {
		originalValidate := validateStepUpChallengeToken
		originalAccess := generateStepUpAccessToken
		originalParse := parseWebAuthnRequestResponse
		t.Cleanup(func() {
			validateStepUpChallengeToken = originalValidate
			generateStepUpAccessToken = originalAccess
			parseWebAuthnRequestResponse = originalParse
		})
		validateStepUpChallengeToken = func(string) (jwtlib.MapClaims, error) {
			return jwtlib.MapClaims{"sub": mfaTestUserUUID.String(), "allowed_methods": []any{"webauthn"}}, nil
		}
		parseWebAuthnRequestResponse = func(io.Reader) (*protocol.ParsedCredentialAssertionData, error) {
			return &protocol.ParsedCredentialAssertionData{}, nil
		}
		var gotAMR []string
		generateStepUpAccessToken = func(_ context.Context, _, _, _, _, _, _ string, opts *authjwt.AccessTokenOptions) (string, error) {
			if opts != nil {
				gotAMR = opts.AMR
			}
			return "access", nil
		}

		// Backup-eligible (synced) passkey → software key (swk).
		svc := &mfaService{
			userRepo: &mockUserRepo{findByUUID: &User{UserID: mfaTestUserID}, findByID: &User{UserID: mfaTestUserID, UserUUID: mfaTestUserUUID}},
			webAuthnSvc: &mockWebAuthnService{finishAuthenticationFn: func(context.Context, int64, *protocol.ParsedCredentialAssertionData) (*UserWebAuthnCredential, error) {
				return &UserWebAuthnCredential{IsBackupEligible: true}, nil
			}},
			authEventService: &mockAuthEventService{},
		}
		got, err := svc.VerifyStepUp(t.Context(), StepUpVerifyRequestDTO{ChallengeToken: "challenge", Method: "webauthn", Assertion: []byte(`{}`)}, mfaTestUserID)
		require.NoError(t, err)
		assert.Equal(t, "access", got.AccessToken)
		assert.Equal(t, []string{"pwd", "user", "swk"}, gotAMR)

		// Device-bound passkey → hardware key (hwk).
		svc.webAuthnSvc = &mockWebAuthnService{finishAuthenticationFn: func(context.Context, int64, *protocol.ParsedCredentialAssertionData) (*UserWebAuthnCredential, error) {
			return &UserWebAuthnCredential{IsBackupEligible: false}, nil
		}}
		_, err = svc.VerifyStepUp(t.Context(), StepUpVerifyRequestDTO{ChallengeToken: "challenge", Method: "webauthn", Assertion: []byte(`{}`)}, mfaTestUserID)
		require.NoError(t, err)
		assert.Equal(t, []string{"pwd", "user", "hwk"}, gotAMR)
	})

	t.Run("verify webauthn step-up error paths", func(t *testing.T) {
		originalValidate := validateStepUpChallengeToken
		originalParse := parseWebAuthnRequestResponse
		t.Cleanup(func() {
			validateStepUpChallengeToken = originalValidate
			parseWebAuthnRequestResponse = originalParse
		})
		validateStepUpChallengeToken = func(string) (jwtlib.MapClaims, error) {
			return jwtlib.MapClaims{"sub": mfaTestUserUUID.String(), "allowed_methods": []any{"webauthn"}}, nil
		}
		base := func() *mfaService {
			return &mfaService{
				userRepo:    &mockUserRepo{findByUUID: &User{UserID: mfaTestUserID}, findByID: &User{UserID: mfaTestUserID, UserUUID: mfaTestUserUUID}},
				webAuthnSvc: &mockWebAuthnService{},
			}
		}

		// Missing assertion.
		_, err := base().VerifyStepUp(t.Context(), StepUpVerifyRequestDTO{ChallengeToken: "challenge", Method: "webauthn"}, mfaTestUserID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "assertion is required")

		// WebAuthn service not wired.
		svcNoWA := base()
		svcNoWA.webAuthnSvc = nil
		_, err = svcNoWA.VerifyStepUp(t.Context(), StepUpVerifyRequestDTO{ChallengeToken: "challenge", Method: "webauthn", Assertion: []byte(`{}`)}, mfaTestUserID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not available")

		// Malformed assertion.
		parseWebAuthnRequestResponse = func(io.Reader) (*protocol.ParsedCredentialAssertionData, error) {
			return nil, errors.New("bad assertion")
		}
		_, err = base().VerifyStepUp(t.Context(), StepUpVerifyRequestDTO{ChallengeToken: "challenge", Method: "webauthn", Assertion: []byte(`{`)}, mfaTestUserID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid WebAuthn assertion")

		// Assertion verification fails.
		parseWebAuthnRequestResponse = func(io.Reader) (*protocol.ParsedCredentialAssertionData, error) {
			return &protocol.ParsedCredentialAssertionData{}, nil
		}
		svcFail := base()
		svcFail.webAuthnSvc = &mockWebAuthnService{finishAuthenticationFn: func(context.Context, int64, *protocol.ParsedCredentialAssertionData) (*UserWebAuthnCredential, error) {
			return nil, apperror.NewUnauthorized("WebAuthn authentication failed")
		}}
		_, err = svcFail.VerifyStepUp(t.Context(), StepUpVerifyRequestDTO{ChallengeToken: "challenge", Method: "webauthn", Assertion: []byte(`{}`)}, mfaTestUserID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "WebAuthn authentication failed")
	})
}

type mockBaseRepositoryMethods[T any] struct{}

func (mockBaseRepositoryMethods[T]) Create(*T) (*T, error)                        { return nil, nil }
func (mockBaseRepositoryMethods[T]) CreateOrUpdate(*T) (*T, error)                { return nil, nil }
func (mockBaseRepositoryMethods[T]) FindAll(...string) ([]T, error)               { return nil, nil }
func (mockBaseRepositoryMethods[T]) FindByUUID(any, ...string) (*T, error)        { return nil, nil }
func (mockBaseRepositoryMethods[T]) FindByUUIDs([]string, ...string) ([]T, error) { return nil, nil }
func (mockBaseRepositoryMethods[T]) FindByID(any, ...string) (*T, error)          { return nil, nil }
func (mockBaseRepositoryMethods[T]) UpdateByUUID(any, any) (*T, error)            { return nil, nil }
func (mockBaseRepositoryMethods[T]) UpdateByID(any, any) (*T, error)              { return nil, nil }
func (mockBaseRepositoryMethods[T]) DeleteByUUID(any) error                       { return nil }
func (mockBaseRepositoryMethods[T]) DeleteByID(any) error                         { return nil }
func (mockBaseRepositoryMethods[T]) Paginate(map[string]any, int, int, ...string) (*PaginationResult[T], error) {
	return nil, nil
}

type mockUserRepo struct {
	mockBaseRepositoryMethods[User]
	findByID    *User
	findByIDErr error
	findByUUID  *User
	findUUIDErr error
}

func (m *mockUserRepo) WithTx(*gorm.DB) UserRepository { return m }

func (m *mockUserRepo) FindByID(any, ...string) (*User, error) {
	return m.findByID, m.findByIDErr
}

func (m *mockUserRepo) FindByUUID(any, ...string) (*User, error) {
	return m.findByUUID, m.findUUIDErr
}

type mockBackupCodeRepo struct {
	mockBaseRepositoryMethods[UserBackupCode]
	findUnused    []UserBackupCode
	findUnusedErr error
	createBulkErr error
	deleteAllErr  error
	markUsedErr   error
}

func (m *mockBackupCodeRepo) WithTx(*gorm.DB) UserBackupCodeRepository { return m }
func (m *mockBackupCodeRepo) CreateBulk([]*UserBackupCode) error       { return m.createBulkErr }
func (m *mockBackupCodeRepo) FindUnusedByUserID(int64) ([]UserBackupCode, error) {
	return m.findUnused, m.findUnusedErr
}
func (m *mockBackupCodeRepo) FindByUserIDAndCodeHash(int64, string) (*UserBackupCode, error) {
	return nil, nil
}
func (m *mockBackupCodeRepo) MarkUsed(int64) error          { return m.markUsedErr }
func (m *mockBackupCodeRepo) DeleteAllByUserID(int64) error { return m.deleteAllErr }

type mockWebAuthnCredentialRepo struct {
	mockBaseRepositoryMethods[UserWebAuthnCredential]
	findByUserID    []UserWebAuthnCredential
	findByUserIDErr error
	deleteAllErr    error
	deleteErr       error
	createErr       error
	findByKeyID     *UserWebAuthnCredential
	findByKeyIDErr  error
	signCountErr    error
	lastUsedErr     error
}

func (m *mockWebAuthnCredentialRepo) WithTx(*gorm.DB) UserWebAuthnCredentialRepository { return m }
func (m *mockWebAuthnCredentialRepo) FindByUserID(int64) ([]UserWebAuthnCredential, error) {
	return m.findByUserID, m.findByUserIDErr
}
func (m *mockWebAuthnCredentialRepo) FindByCredentialKeyID(string) (*UserWebAuthnCredential, error) {
	return m.findByKeyID, m.findByKeyIDErr
}
func (m *mockWebAuthnCredentialRepo) CreateCredential(*UserWebAuthnCredential) error {
	return m.createErr
}
func (m *mockWebAuthnCredentialRepo) UpdateSignCount(int64, int64) error      { return m.signCountErr }
func (m *mockWebAuthnCredentialRepo) UpdateLastUsed(int64) error              { return m.lastUsedErr }
func (m *mockWebAuthnCredentialRepo) DeleteCredentialByID(int64, int64) error { return m.deleteErr }
func (m *mockWebAuthnCredentialRepo) DeleteAllByUserID(int64) error           { return m.deleteAllErr }

type mockSecuritySettingRepo struct {
	mockBaseRepositoryMethods[secpolicy.SecuritySetting]
	findByTenantID    *secpolicy.SecuritySetting
	findByTenantIDErr error
}

func (m *mockSecuritySettingRepo) WithTx(*gorm.DB) secpolicy.SecuritySettingRepository { return m }
func (m *mockSecuritySettingRepo) FindByTenantID(int64) (*secpolicy.SecuritySetting, error) {
	return m.findByTenantID, m.findByTenantIDErr
}
func (m *mockSecuritySettingRepo) FindDefaultByTenantID(int64) (*secpolicy.SecuritySetting, error) {
	return nil, nil
}
func (m *mockSecuritySettingRepo) FindPaginated(secpolicy.SecuritySettingRepositoryGetFilter) (*PaginationResult[secpolicy.SecuritySetting], error) {
	return nil, nil
}
func (m *mockSecuritySettingRepo) IncrementVersion(int64) error { return nil }

type mockTOTPSecretRepo struct {
	mockBaseRepositoryMethods[UserTOTPSecret]
	findByUserID    *UserTOTPSecret
	findByUserIDErr error
	upsertErr       error
	enableErr       error
	disableErr      error
	markAccepted    bool
	markStepErr     error
}

func (m *mockTOTPSecretRepo) WithTx(*gorm.DB) UserTOTPSecretRepository { return m }
func (m *mockTOTPSecretRepo) FindByUserID(int64) (*UserTOTPSecret, error) {
	return m.findByUserID, m.findByUserIDErr
}
func (m *mockTOTPSecretRepo) Upsert(*UserTOTPSecret) error { return m.upsertErr }
func (m *mockTOTPSecretRepo) Enable(int64) error           { return m.enableErr }
func (m *mockTOTPSecretRepo) Disable(int64) error          { return m.disableErr }
func (m *mockTOTPSecretRepo) UpdateLastUsed(int64) error   { return nil }
func (m *mockTOTPSecretRepo) MarkStepUsed(int64, int64) (bool, error) {
	return m.markAccepted, m.markStepErr
}
func (m *mockTOTPSecretRepo) DeleteByUserID(int64) error { return nil }

type mockSMSOtpRepo struct {
	mockBaseRepositoryMethods[notifier.UserOTP]
	findValid     *notifier.UserOTP
	findValidErr  error
	recordFailErr error
	markUsedErr   error
}

func (m *mockSMSOtpRepo) WithTx(*gorm.DB) notifier.UserOTPRepository { return m }
func (m *mockSMSOtpRepo) FindValid(channel, recipient string) (*notifier.UserOTP, error) {
	return m.findValid, m.findValidErr
}
func (m *mockSMSOtpRepo) RecordFailure(int64, int) error { return m.recordFailErr }
func (m *mockSMSOtpRepo) MarkUsed(int64) error           { return m.markUsedErr }

type mockSMSPhoneRepo struct {
	mockBaseRepositoryMethods[UserSMSPhone]
	findByUserID    *UserSMSPhone
	findByUserIDErr error
	deleteErr       error
}

func (m *mockSMSPhoneRepo) WithTx(*gorm.DB) UserSMSPhoneRepository { return m }
func (m *mockSMSPhoneRepo) FindByUserID(int64) (*UserSMSPhone, error) {
	return m.findByUserID, m.findByUserIDErr
}
func (m *mockSMSPhoneRepo) DeleteByUserID(int64) error { return m.deleteErr }
