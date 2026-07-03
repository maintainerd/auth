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
	"github.com/maintainerd/maintainerd-auth/internal/notifier"
	"github.com/maintainerd/maintainerd-auth/internal/platform/apperror"
	"github.com/maintainerd/maintainerd-auth/internal/platform/config"
	"github.com/maintainerd/maintainerd-auth/internal/platform/crypto"
	authjwt "github.com/maintainerd/maintainerd-auth/internal/platform/jwt"
	"github.com/maintainerd/maintainerd-auth/internal/platform/middleware"
	"github.com/maintainerd/maintainerd-auth/internal/secpolicy"
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

	step, ok, err := validateTOTPAndStep(code, secret, at, totpDigits, totpPeriod)
	require.NoError(t, err)
	assert.True(t, ok)
	assert.Equal(t, at.Unix()/int64(totpPeriod), step)
}

func TestValidateTOTPAndStepRejectsInvalidCode(t *testing.T) {
	step, ok, err := validateTOTPAndStep("000000", "JBSWY3DPEHPK3PXP", time.Unix(1_700_000_000, 0).UTC(), totpDigits, totpPeriod)
	require.NoError(t, err)
	assert.False(t, ok)
	assert.Zero(t, step)

	step, ok, err = validateTOTPAndStep("000000", "JBSWY3DPEHPK3PXP", time.Unix(-1, 0).UTC(), totpDigits, totpPeriod)
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

func TestMFAMethodAllowedPolicy(t *testing.T) {
	tests := []struct {
		name   string
		policy *secpolicy.MFAPolicy
		method string
		want   bool
	}{
		{name: "nil policy allows configured method", method: "totp", want: true},
		{name: "disabled mode blocks configured method", policy: &secpolicy.MFAPolicy{Mode: "disabled", AllowedMethods: []string{"totp"}}, method: "totp", want: false},
		{name: "sms requires explicit sms permission flag", policy: &secpolicy.MFAPolicy{AllowedMethods: []string{"sms"}, AllowSMS: false}, method: "sms", want: false},
		{name: "sms gate applies even without allowed method list", policy: &secpolicy.MFAPolicy{AllowSMS: false}, method: "sms", want: false},
		{name: "allowed method passes", policy: &secpolicy.MFAPolicy{AllowedMethods: []string{"webauthn"}}, method: "webauthn", want: true},
		{name: "unlisted method blocked", policy: &secpolicy.MFAPolicy{AllowedMethods: []string{"totp"}}, method: "webauthn", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, methodAllowed(tt.policy, tt.method))
		})
	}
}

func TestMFAService_GetBackupCodesCount(t *testing.T) {
	tests := []struct {
		name    string
		codes   []UserMFABackupCode
		err     error
		want    int
		wantErr string
	}{
		{name: "counts unused codes", codes: []UserMFABackupCode{{BackupCodeID: 1}, {BackupCodeID: 2}}, want: 2},
		{name: "repo error", err: errors.New("db down"), wantErr: "backup code lookup failed"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &mfaService{mfaBackupCodeRepo: &mockMFABackupCodeRepo{findUnused: tt.codes, findUnusedErr: tt.err}}

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
		codes      []UserMFABackupCode
		creds      []UserMFAWebAuthnCredential
		credsErr   error
		wantErr    string
		assertResp func(*testing.T, *MFAStatusResponseDTO)
	}{
		{
			name:  "success maps factors and timestamps",
			user:  &User{UserID: 42, IsTOTPEnabled: true, IsWebAuthnEnabled: true, MFAEnabledAt: &enabledAt},
			codes: []UserMFABackupCode{{BackupCodeID: 1}, {BackupCodeID: 2}},
			creds: []UserMFAWebAuthnCredential{{
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
				userRepo:            &mockUserRepo{findByID: tt.user, findByIDErr: tt.userErr},
				mfaBackupCodeRepo:   &mockMFABackupCodeRepo{findUnused: tt.codes},
				mfaWebAuthnCredRepo: &mockMFAWebAuthnCredentialRepo{findByUserID: tt.creds, findByUserIDErr: tt.credsErr},
				mfaPhoneRepo:        &mockMFAPhoneRepo{}, emailOTPRepo: &mockMFAEmailRepo{},
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
			want:    &MFAPolicyDTO{Required: false, AllowedMethods: []string{"totp", "webauthn", "backup_code"}},
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
			want: &MFAPolicyDTO{Required: false, AllowedMethods: []string{"totp", "webauthn", "backup_code"}},
		},
		{
			name: "missing allowed methods keeps required flag with default methods",
			setting: &secpolicy.SecuritySetting{
				MFAConfig: datatypes.JSON([]byte(`{"required":true}`)),
			},
			want: &MFAPolicyDTO{Required: true, AllowedMethods: []string{"totp", "webauthn", "backup_code"}},
		},
		{
			name: "canonical mode and recovery code are normalized",
			setting: &secpolicy.SecuritySetting{
				MFAConfig: datatypes.JSON([]byte(`{"mode":"enforced","allowed_methods":["totp","recovery_code"]}`)),
			},
			want: &MFAPolicyDTO{Required: true, AllowedMethods: []string{"totp", "backup_code"}},
		},
		{
			name: "disabled mode exposes no usable methods",
			setting: &secpolicy.SecuritySetting{
				MFAConfig: datatypes.JSON([]byte(`{"mode":"disabled","allowed_methods":["totp","webauthn"]}`)),
			},
			want: &MFAPolicyDTO{Required: false, AllowedMethods: []string{}},
		},
		{
			name: "repo error uses default policy",
			err:  errors.New("db down"),
			want: &MFAPolicyDTO{Required: false, AllowedMethods: []string{"totp", "webauthn", "backup_code"}},
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
	svc := NewMFAService(db, &mockUserRepo{}, &mockMFATOTPSecretRepo{}, &mockMFAWebAuthnCredentialRepo{}, &mockWebAuthnService{}, &mockMFABackupCodeRepo{}, &mockMFAPhoneRepo{}, &mockMFAEmailRepo{}, &mockSMSOtpRepo{}, &mockSecuritySettingRepo{}, &mockAuthEventService{})
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
				userRepo:    &mockUserRepo{findByID: tt.user, findByIDErr: tt.userErr},
				mfaTotpRepo: &mockMFATOTPSecretRepo{upsertErr: tt.upsert},
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
			userRepo:    &mockUserRepo{findByID: &User{UserID: mfaTestUserID, Email: "user@example.com"}},
			mfaTotpRepo: &mockMFATOTPSecretRepo{},
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
			db:                db,
			mfaTotpRepo:       &mockMFATOTPSecretRepo{findByUserID: &UserMFATOTPSecret{Secret: secret, IsEnabled: false}},
			mfaBackupCodeRepo: &mockMFABackupCodeRepo{},
			authEventService:  events,
		}

		got, err := svc.FinishTOTPEnrollment(t.Context(), mfaTestUserID, code)

		require.NoError(t, err)
		assert.Len(t, got, mfaBackupCodeCount)
		assert.Len(t, events.inputs, 1)
		assertExpectationsMet(t, mock)
	})

	tests := []struct {
		name    string
		record  *UserMFATOTPSecret
		repoErr error
		enable  error
		dbErr   bool
		wantErr string
	}{
		{name: "lookup error", repoErr: errors.New("db down"), wantErr: "TOTP secret lookup failed"},
		{name: "no pending record", wantErr: "no pending TOTP enrollment found"},
		{name: "already enabled", record: &UserMFATOTPSecret{Secret: secret, IsEnabled: true}, wantErr: "no pending TOTP enrollment found"},
		{name: "invalid code", record: &UserMFATOTPSecret{Secret: secret}, wantErr: "invalid TOTP code"},
		{name: "enable error", record: &UserMFATOTPSecret{Secret: secret}, enable: errors.New("db down"), wantErr: "failed to enable TOTP"},
		{name: "user update error", record: &UserMFATOTPSecret{Secret: secret}, dbErr: true, wantErr: "failed to update user MFA status"},
		{name: "backup code delete error", record: &UserMFATOTPSecret{Secret: secret}, wantErr: "failed to delete existing backup codes"},
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
				db:                db,
				mfaTotpRepo:       &mockMFATOTPSecretRepo{findByUserID: tt.record, findByUserIDErr: tt.repoErr, enableErr: tt.enable},
				mfaBackupCodeRepo: &mockMFABackupCodeRepo{deleteAllErr: map[bool]error{true: errors.New("db down")}[tt.name == "backup code delete error"]},
				authEventService:  &mockAuthEventService{},
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
		record   *UserMFATOTPSecret
		repoErr  error
		accepted bool
		markErr  error
		code     string
		want     bool
		wantErr  string
	}{
		{name: "valid code accepted", record: &UserMFATOTPSecret{Secret: secret, IsEnabled: true}, accepted: true, code: code, want: true},
		{name: "replay rejected without error", record: &UserMFATOTPSecret{Secret: secret, IsEnabled: true}, accepted: false, code: code, want: false},
		{name: "invalid code", record: &UserMFATOTPSecret{Secret: secret, IsEnabled: true}, code: "000000", want: false},
		{name: "lookup error", repoErr: errors.New("db down"), wantErr: "TOTP lookup failed"},
		{name: "not enabled", record: &UserMFATOTPSecret{Secret: secret, IsEnabled: false}, wantErr: "TOTP is not enabled"},
		{name: "mark step error", record: &UserMFATOTPSecret{Secret: secret, IsEnabled: true}, code: code, markErr: errors.New("db down"), wantErr: "failed to update TOTP last-used step"},
		{name: "invalid secret", record: &UserMFATOTPSecret{Secret: "%", IsEnabled: true}, code: code, wantErr: "invalid TOTP code"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &mfaService{mfaTotpRepo: &mockMFATOTPSecretRepo{findByUserID: tt.record, findByUserIDErr: tt.repoErr, markAccepted: tt.accepted, markStepErr: tt.markErr}}

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
		svc := &mfaService{mfaTotpRepo: &mockMFATOTPSecretRepo{findByUserID: &UserMFATOTPSecret{Secret: secret, IsEnabled: true}}}

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
			db:                db,
			userRepo:          &mockUserRepo{findByID: &User{UserID: mfaTestUserID, IsTOTPEnabled: true}},
			mfaBackupCodeRepo: &mockMFABackupCodeRepo{deleteAllErr: errors.New("must not be called")},
			mfaPhoneRepo:      &mockMFAPhoneRepo{}, emailOTPRepo: &mockMFAEmailRepo{},
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
			db:                db,
			userRepo:          &mockUserRepo{findByID: &User{UserID: mfaTestUserID}},
			mfaBackupCodeRepo: &mockMFABackupCodeRepo{},
			mfaPhoneRepo:      &mockMFAPhoneRepo{}, emailOTPRepo: &mockMFAEmailRepo{},
		}

		require.NoError(t, svc.SyncMFAState(t.Context(), mfaTestUserID))
		assertExpectationsMet(t, mock)
	})

	t.Run("backup code delete error surfaces", func(t *testing.T) {
		svc := &mfaService{
			userRepo:          &mockUserRepo{findByID: &User{UserID: mfaTestUserID}},
			mfaBackupCodeRepo: &mockMFABackupCodeRepo{deleteAllErr: errors.New("db down")},
			mfaPhoneRepo:      &mockMFAPhoneRepo{}, emailOTPRepo: &mockMFAEmailRepo{},
		}

		err := svc.SyncMFAState(t.Context(), mfaTestUserID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to delete backup codes")
	})
}

func TestMFAService_EnrolledMFAMethods(t *testing.T) {
	t.Run("returns all active factors in canonical order", func(t *testing.T) {
		svc := &mfaService{
			userRepo:     &mockUserRepo{findByID: &User{UserID: mfaTestUserID, IsTOTPEnabled: true, IsWebAuthnEnabled: true}},
			mfaPhoneRepo: &mockMFAPhoneRepo{findByUserID: &UserMFAPhone{Phone: "+15550001111", IsVerified: true}}, emailOTPRepo: &mockMFAEmailRepo{},
			mfaBackupCodeRepo: &mockMFABackupCodeRepo{findUnused: []UserMFABackupCode{{BackupCodeID: 1}}},
		}
		got, err := svc.EnrolledMFAMethods(t.Context(), mfaTestUserID)
		require.NoError(t, err)
		assert.Equal(t, []string{"totp", "webauthn", "sms", "backup_code"}, got)
	})

	t.Run("no active factor returns empty", func(t *testing.T) {
		svc := &mfaService{
			userRepo:     &mockUserRepo{findByID: &User{UserID: mfaTestUserID}},
			mfaPhoneRepo: &mockMFAPhoneRepo{}, emailOTPRepo: &mockMFAEmailRepo{},
			mfaBackupCodeRepo: &mockMFABackupCodeRepo{},
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

	t.Run("policy disabled blocks stale SMS verification", func(t *testing.T) {
		svc := &mfaService{secSettingRepo: &mockSecuritySettingRepo{findByTenantID: &secpolicy.SecuritySetting{
			MFAConfig: datatypes.JSON([]byte(`{"mode":"disabled","allowed_methods":["sms"],"allow_sms":true}`)),
		}}}

		_, err := svc.verifyFactor(t.Context(), mfaTestUserID, "sms", "123456", nil)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "sms MFA is not permitted by tenant policy")
	})

	t.Run("verifies against the MFA phone record, not users.phone", func(t *testing.T) {
		svc := &mfaService{
			mfaPhoneRepo: &mockMFAPhoneRepo{findByUserID: &UserMFAPhone{Phone: "+15550001111", IsVerified: true}}, emailOTPRepo: &mockMFAEmailRepo{},
			smsOtpRepo: &mockSMSOtpRepo{findValid: &notifier.UserOTP{UserOTPID: 1, OTPHash: crypto.HashAuthorizationCode("123456")}},
		}
		amr, err := svc.verifyFactor(t.Context(), mfaTestUserID, "sms", "123456", nil)
		require.NoError(t, err)
		assert.Equal(t, []string{"pwd", "sms"}, amr)
	})

	t.Run("no verified MFA phone is rejected", func(t *testing.T) {
		svc := &mfaService{mfaPhoneRepo: &mockMFAPhoneRepo{findByUserID: nil}, emailOTPRepo: &mockMFAEmailRepo{}}
		_, err := svc.verifyFactor(t.Context(), mfaTestUserID, "sms", "123456", nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "no verified phone on file")
	})

	t.Run("invalid code is rejected", func(t *testing.T) {
		svc := &mfaService{
			mfaPhoneRepo: &mockMFAPhoneRepo{findByUserID: &UserMFAPhone{Phone: "+15550001111", IsVerified: true}}, emailOTPRepo: &mockMFAEmailRepo{},
			smsOtpRepo: &mockSMSOtpRepo{findValid: &notifier.UserOTP{UserOTPID: 1, OTPHash: crypto.HashAuthorizationCode("999999")}},
		}
		_, err := svc.verifyFactor(t.Context(), mfaTestUserID, "sms", "123456", nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid SMS code")
	})
}

func TestMFAService_BeginWebAuthnLoginPolicyGate(t *testing.T) {
	t.Run("disabled policy blocks login ceremony", func(t *testing.T) {
		svc := &mfaService{
			secSettingRepo: &mockSecuritySettingRepo{findByTenantID: &secpolicy.SecuritySetting{
				MFAConfig: datatypes.JSON([]byte(`{"mode":"disabled","allowed_methods":["webauthn"]}`)),
			}},
			webAuthnSvc: &mockWebAuthnService{beginAuthenticationFn: func(context.Context, int64) (*protocol.CredentialAssertion, error) {
				t.Fatal("WebAuthn ceremony should not start when disabled")
				return nil, nil
			}},
		}

		_, err := svc.BeginWebAuthnLogin(t.Context(), mfaTestUserID)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "WebAuthn MFA is not permitted by tenant policy")
	})

	t.Run("allowed policy starts login ceremony", func(t *testing.T) {
		svc := &mfaService{
			secSettingRepo: &mockSecuritySettingRepo{findByTenantID: &secpolicy.SecuritySetting{
				MFAConfig: datatypes.JSON([]byte(`{"mode":"optional","allowed_methods":["webauthn"]}`)),
			}},
			webAuthnSvc: &mockWebAuthnService{beginAuthenticationFn: func(_ context.Context, userID int64) (*protocol.CredentialAssertion, error) {
				assert.Equal(t, mfaTestUserID, userID)
				return &protocol.CredentialAssertion{}, nil
			}},
		}

		got, err := svc.BeginWebAuthnLogin(t.Context(), mfaTestUserID)

		require.NoError(t, err)
		assert.NotEmpty(t, got)
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
			db:                db,
			userRepo:          &mockUserRepo{findByID: &User{UserID: mfaTestUserID}},
			mfaTotpRepo:       &mockMFATOTPSecretRepo{},
			mfaBackupCodeRepo: &mockMFABackupCodeRepo{},
			mfaPhoneRepo:      &mockMFAPhoneRepo{}, emailOTPRepo: &mockMFAEmailRepo{},
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
			svc := &mfaService{db: db, mfaTotpRepo: &mockMFATOTPSecretRepo{disableErr: tt.totpErr}, mfaBackupCodeRepo: &mockMFABackupCodeRepo{deleteAllErr: tt.codeErr}, authEventService: &mockAuthEventService{}}

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
		svc := &mfaService{mfaBackupCodeRepo: &mockMFABackupCodeRepo{}}
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

		_, err := (&mfaService{mfaBackupCodeRepo: &mockMFABackupCodeRepo{}}).RegenerateBackupCodes(t.Context(), mfaTestUserID)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "backup code hashing failed")
	})

	t.Run("regenerate delete and storage errors", func(t *testing.T) {
		_, err := (&mfaService{mfaBackupCodeRepo: &mockMFABackupCodeRepo{deleteAllErr: errors.New("db down")}}).RegenerateBackupCodes(t.Context(), mfaTestUserID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to delete existing backup codes")

		_, err = (&mfaService{mfaBackupCodeRepo: &mockMFABackupCodeRepo{createBulkErr: errors.New("db down")}}).RegenerateBackupCodes(t.Context(), mfaTestUserID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "backup code storage failed")
	})
}

func TestMFAService_AdminResetMFA(t *testing.T) {
	t.Run("success clears all factors", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		expectUserTenantIDLookup(mock, 1)
		mock.ExpectBegin()
		expectMFAUpdate(mock, "users").WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectCommit()
		events := &mockAuthEventService{}
		svc := &mfaService{
			db:                  db,
			userRepo:            &mockUserRepo{findByUUID: &User{UserID: mfaTestUserID}},
			mfaTotpRepo:         &mockMFATOTPSecretRepo{},
			mfaBackupCodeRepo:   &mockMFABackupCodeRepo{},
			mfaWebAuthnCredRepo: &mockMFAWebAuthnCredentialRepo{},
			mfaPhoneRepo:        &mockMFAPhoneRepo{}, emailOTPRepo: &mockMFAEmailRepo{},
			authEventService: events,
		}

		require.NoError(t, svc.AdminResetMFA(t.Context(), mfaTestUserUUID.String(), 99, 1))

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
			// A resolvable target user reaches the (now fail-closed) tenant guard,
			// which requires the target's tenant to match the caller's.
			if tt.user != nil {
				expectUserTenantIDLookup(mock, 1)
			}
			if tt.dbErr {
				mock.ExpectBegin()
				expectMFAUpdate(mock, "users").WillReturnError(errors.New("db down"))
				mock.ExpectRollback()
			}
			svc := &mfaService{
				db:                  db,
				userRepo:            &mockUserRepo{findByUUID: tt.user, findUUIDErr: tt.userErr},
				mfaTotpRepo:         &mockMFATOTPSecretRepo{disableErr: tt.totpErr},
				mfaBackupCodeRepo:   &mockMFABackupCodeRepo{deleteAllErr: tt.codeErr},
				mfaWebAuthnCredRepo: &mockMFAWebAuthnCredentialRepo{deleteAllErr: tt.webErr},
				mfaPhoneRepo:        &mockMFAPhoneRepo{}, emailOTPRepo: &mockMFAEmailRepo{},
				authEventService: &mockAuthEventService{},
			}

			err := svc.AdminResetMFA(t.Context(), mfaTestUserUUID.String(), 99, 1)

			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.wantErr)
			assertExpectationsMet(t, mock)
		})
	}
}

func TestMFAService_AdminResetMFAMethod(t *testing.T) {
	t.Run("success resets a single factor and syncs state", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		expectUserTenantIDLookup(mock, 1)
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
			db:                  db,
			userRepo:            &mockUserRepo{findByUUID: &User{UserID: mfaTestUserID}, findByID: &User{UserID: mfaTestUserID}},
			mfaTotpRepo:         &mockMFATOTPSecretRepo{},
			mfaBackupCodeRepo:   &mockMFABackupCodeRepo{},
			mfaWebAuthnCredRepo: &mockMFAWebAuthnCredentialRepo{},
			mfaPhoneRepo:        &mockMFAPhoneRepo{}, emailOTPRepo: &mockMFAEmailRepo{},
			authEventService: events,
		}

		require.NoError(t, svc.AdminResetMFAMethod(t.Context(), mfaTestUserUUID.String(), "totp", 99, 1))

		assert.Len(t, events.inputs, 1)
		assertExpectationsMet(t, mock)
	})

	t.Run("missing user", func(t *testing.T) {
		svc := &mfaService{userRepo: &mockUserRepo{findByUUID: nil}}
		err := svc.AdminResetMFAMethod(t.Context(), mfaTestUserUUID.String(), "totp", 99, 1)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "target user not found")
	})

	t.Run("unsupported method", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		expectUserTenantIDLookup(mock, 1)
		svc := &mfaService{db: db, userRepo: &mockUserRepo{findByUUID: &User{UserID: mfaTestUserID}}}
		err := svc.AdminResetMFAMethod(t.Context(), mfaTestUserUUID.String(), "carrier-pigeon", 99, 1)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "unsupported MFA method")
		assertExpectationsMet(t, mock)
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
			db, mock := newMockGormDB(t)
			expectUserTenantIDLookup(mock, 1)
			svc := &mfaService{
				db:                  db,
				userRepo:            &mockUserRepo{findByUUID: &User{UserID: mfaTestUserID}},
				mfaTotpRepo:         &mockMFATOTPSecretRepo{disableErr: tt.totpErr},
				mfaWebAuthnCredRepo: &mockMFAWebAuthnCredentialRepo{deleteAllErr: tt.webErr},
				mfaPhoneRepo:        &mockMFAPhoneRepo{deleteErr: tt.smsErr}, emailOTPRepo: &mockMFAEmailRepo{},
				mfaBackupCodeRepo: &mockMFABackupCodeRepo{deleteAllErr: tt.codeErr},
				authEventService:  &mockAuthEventService{},
			}
			err := svc.AdminResetMFAMethod(t.Context(), mfaTestUserUUID.String(), tt.method, 99, 1)
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.wantErr)
			assertExpectationsMet(t, mock)
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
			db:                  db,
			mfaTotpRepo:         &mockMFATOTPSecretRepo{},
			mfaBackupCodeRepo:   &mockMFABackupCodeRepo{},
			mfaWebAuthnCredRepo: &mockMFAWebAuthnCredentialRepo{},
			mfaPhoneRepo:        &mockMFAPhoneRepo{}, emailOTPRepo: &mockMFAEmailRepo{},
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
				db:                  db,
				mfaTotpRepo:         &mockMFATOTPSecretRepo{disableErr: tt.totpErr},
				mfaBackupCodeRepo:   &mockMFABackupCodeRepo{deleteAllErr: tt.codeErr},
				mfaWebAuthnCredRepo: &mockMFAWebAuthnCredentialRepo{deleteAllErr: tt.webErr},
				mfaPhoneRepo:        &mockMFAPhoneRepo{deleteErr: tt.smsErr}, emailOTPRepo: &mockMFAEmailRepo{},
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
		svc := &mfaService{userRepo: &mockUserRepo{findByUUID: &User{UserID: mfaTestUserID}}}
		got, err := svc.IssueStepUpChallenge(t.Context(), mfaTestUserUUID.String(), []string{"totp"})
		require.NoError(t, err)
		assert.Equal(t, "challenge", got.ChallengeToken)

		generateStepUpChallengeToken = func(context.Context, string, time.Duration, ...[]string) (string, error) {
			return "", errors.New("jwt down")
		}
		_, err = svc.IssueStepUpChallenge(t.Context(), mfaTestUserUUID.String(), []string{"totp"})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "step-up challenge generation failed")
	})

	t.Run("issue challenge filters disabled methods and prefers configured method", func(t *testing.T) {
		original := generateStepUpChallengeToken
		t.Cleanup(func() { generateStepUpChallengeToken = original })
		generateStepUpChallengeToken = func(_ context.Context, userUUID string, ttl time.Duration, allowedMethods ...[]string) (string, error) {
			assert.Equal(t, mfaTestUserUUID.String(), userUUID)
			assert.Equal(t, []string{"webauthn", "totp"}, allowedMethods[0])
			return "challenge", nil
		}
		svc := &mfaService{
			userRepo: &mockUserRepo{findByUUID: &User{UserID: mfaTestUserID}},
			secSettingRepo: &mockSecuritySettingRepo{findByTenantID: &secpolicy.SecuritySetting{
				MFAConfig: datatypes.JSON([]byte(`{"mode":"optional","allowed_methods":["totp","webauthn"],"preferred_method":"webauthn"}`)),
			}},
		}

		got, err := svc.IssueStepUpChallenge(t.Context(), mfaTestUserUUID.String(), []string{"totp", "sms", "webauthn"})

		require.NoError(t, err)
		assert.Equal(t, []string{"webauthn", "totp"}, got.AllowedMethods)
	})

	t.Run("issue challenge rejects when policy disables every requested method", func(t *testing.T) {
		svc := &mfaService{
			userRepo: &mockUserRepo{findByUUID: &User{UserID: mfaTestUserID}},
			secSettingRepo: &mockSecuritySettingRepo{findByTenantID: &secpolicy.SecuritySetting{
				MFAConfig: datatypes.JSON([]byte(`{"mode":"disabled","allowed_methods":["totp"]}`)),
			}},
		}

		_, err := svc.IssueStepUpChallenge(t.Context(), mfaTestUserUUID.String(), []string{"totp"})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "no MFA methods are permitted by tenant policy")
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
			userRepo:          &mockUserRepo{findByUUID: &User{UserID: mfaTestUserID}, findByID: &User{UserID: mfaTestUserID, UserUUID: mfaTestUserUUID}},
			mfaBackupCodeRepo: &mockMFABackupCodeRepo{findUnused: []UserMFABackupCode{{BackupCodeID: 1, CodeHash: string(hash)}}},
			authEventService:  events,
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
			mfaTotpRepo:      &mockMFATOTPSecretRepo{findByUserID: &UserMFATOTPSecret{Secret: secret, IsEnabled: true}, markAccepted: true},
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
			userRepo:          &mockUserRepo{findByUUID: &User{UserID: mfaTestUserID}},
			mfaBackupCodeRepo: &mockMFABackupCodeRepo{findUnusedErr: errors.New("db down")},
		}
		_, err := svc.VerifyStepUp(t.Context(), StepUpVerifyRequestDTO{ChallengeToken: "challenge", Method: "backup_code", Code: "bad"}, mfaTestUserID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "backup code lookup failed")

		svc.mfaBackupCodeRepo = &mockMFABackupCodeRepo{findUnused: []UserMFABackupCode{{BackupCodeID: 1, CodeHash: "bad-hash"}}}
		_, err = svc.VerifyStepUp(t.Context(), StepUpVerifyRequestDTO{ChallengeToken: "challenge", Method: "backup_code", Code: "bad"}, mfaTestUserID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid backup code")

		hash, hashErr := bcrypt.GenerateFromPassword([]byte("backup-code"), bcrypt.DefaultCost)
		require.NoError(t, hashErr)
		svc.mfaBackupCodeRepo = &mockMFABackupCodeRepo{findUnused: []UserMFABackupCode{{BackupCodeID: 1, CodeHash: string(hash)}}, markUsedErr: errors.New("db down")}
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
			userRepo:          &mockUserRepo{findByUUID: &User{UserID: mfaTestUserID}, findByIDErr: errors.New("db down")},
			mfaBackupCodeRepo: &mockMFABackupCodeRepo{findUnused: []UserMFABackupCode{{BackupCodeID: 1, CodeHash: string(hash)}}},
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
			mfaTotpRepo:      &mockMFATOTPSecretRepo{findByUserID: &UserMFATOTPSecret{Secret: secret, IsEnabled: true}, markAccepted: true},
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
			webAuthnSvc: &mockWebAuthnService{finishAuthenticationFn: func(context.Context, int64, *protocol.ParsedCredentialAssertionData) (*UserMFAWebAuthnCredential, error) {
				return &UserMFAWebAuthnCredential{IsBackupEligible: true}, nil
			}},
			authEventService: &mockAuthEventService{},
		}
		got, err := svc.VerifyStepUp(t.Context(), StepUpVerifyRequestDTO{ChallengeToken: "challenge", Method: "webauthn", Assertion: []byte(`{}`)}, mfaTestUserID)
		require.NoError(t, err)
		assert.Equal(t, "access", got.AccessToken)
		assert.Equal(t, []string{"pwd", "user", "swk"}, gotAMR)

		// Device-bound passkey → hardware key (hwk).
		svc.webAuthnSvc = &mockWebAuthnService{finishAuthenticationFn: func(context.Context, int64, *protocol.ParsedCredentialAssertionData) (*UserMFAWebAuthnCredential, error) {
			return &UserMFAWebAuthnCredential{IsBackupEligible: false}, nil
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
		svcFail.webAuthnSvc = &mockWebAuthnService{finishAuthenticationFn: func(context.Context, int64, *protocol.ParsedCredentialAssertionData) (*UserMFAWebAuthnCredential, error) {
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

type mockMFABackupCodeRepo struct {
	mockBaseRepositoryMethods[UserMFABackupCode]
	findUnused    []UserMFABackupCode
	findUnusedErr error
	createBulkErr error
	deleteAllErr  error
	markUsedErr   error
}

func (m *mockMFABackupCodeRepo) WithTx(*gorm.DB) UserMFABackupCodeRepository { return m }
func (m *mockMFABackupCodeRepo) CreateBulk([]*UserMFABackupCode) error       { return m.createBulkErr }
func (m *mockMFABackupCodeRepo) FindUnusedByUserID(int64) ([]UserMFABackupCode, error) {
	return m.findUnused, m.findUnusedErr
}
func (m *mockMFABackupCodeRepo) FindByUserIDAndCodeHash(int64, string) (*UserMFABackupCode, error) {
	return nil, nil
}
func (m *mockMFABackupCodeRepo) MarkUsed(int64) error          { return m.markUsedErr }
func (m *mockMFABackupCodeRepo) DeleteAllByUserID(int64) error { return m.deleteAllErr }

type mockMFAWebAuthnCredentialRepo struct {
	mockBaseRepositoryMethods[UserMFAWebAuthnCredential]
	findByUserID    []UserMFAWebAuthnCredential
	findByUserIDErr error
	deleteAllErr    error
	deleteErr       error
	createErr       error
	findByKeyID     *UserMFAWebAuthnCredential
	findByKeyIDErr  error
	signCountErr    error
	lastUsedErr     error
}

func (m *mockMFAWebAuthnCredentialRepo) WithTx(*gorm.DB) UserMFAWebAuthnCredentialRepository {
	return m
}
func (m *mockMFAWebAuthnCredentialRepo) FindByUserID(int64) ([]UserMFAWebAuthnCredential, error) {
	return m.findByUserID, m.findByUserIDErr
}
func (m *mockMFAWebAuthnCredentialRepo) FindByCredentialKeyID(string) (*UserMFAWebAuthnCredential, error) {
	return m.findByKeyID, m.findByKeyIDErr
}
func (m *mockMFAWebAuthnCredentialRepo) CreateCredential(*UserMFAWebAuthnCredential) error {
	return m.createErr
}
func (m *mockMFAWebAuthnCredentialRepo) UpdateSignCount(int64, int64) error      { return m.signCountErr }
func (m *mockMFAWebAuthnCredentialRepo) UpdateLastUsed(int64) error              { return m.lastUsedErr }
func (m *mockMFAWebAuthnCredentialRepo) DeleteCredentialByID(int64, int64) error { return m.deleteErr }
func (m *mockMFAWebAuthnCredentialRepo) DeleteAllByUserID(int64) error           { return m.deleteAllErr }

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

type mockMFATOTPSecretRepo struct {
	mockBaseRepositoryMethods[UserMFATOTPSecret]
	findByUserID    *UserMFATOTPSecret
	findByUserIDErr error
	upsertErr       error
	enableErr       error
	disableErr      error
	markAccepted    bool
	markStepErr     error
}

func (m *mockMFATOTPSecretRepo) WithTx(*gorm.DB) UserMFATOTPSecretRepository { return m }
func (m *mockMFATOTPSecretRepo) FindByUserID(int64) (*UserMFATOTPSecret, error) {
	return m.findByUserID, m.findByUserIDErr
}
func (m *mockMFATOTPSecretRepo) Upsert(*UserMFATOTPSecret) error { return m.upsertErr }
func (m *mockMFATOTPSecretRepo) Enable(int64) error              { return m.enableErr }
func (m *mockMFATOTPSecretRepo) Disable(int64) error             { return m.disableErr }
func (m *mockMFATOTPSecretRepo) UpdateLastUsed(int64) error      { return nil }
func (m *mockMFATOTPSecretRepo) MarkStepUsed(int64, int64) (bool, error) {
	return m.markAccepted, m.markStepErr
}
func (m *mockMFATOTPSecretRepo) DeleteByUserID(int64) error { return nil }

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
func (m *mockSMSOtpRepo) RecordFailure(int64, int) error         { return m.recordFailErr }
func (m *mockSMSOtpRepo) MarkUsed(int64) error                   { return m.markUsedErr }
func (m *mockSMSOtpRepo) DeleteExpired(time.Time) (int64, error) { return 0, nil }

type mockMFAPhoneRepo struct {
	mockBaseRepositoryMethods[UserMFAPhone]
	findByUserID    *UserMFAPhone
	findByUserIDErr error
	deleteErr       error
}

func (m *mockMFAPhoneRepo) WithTx(*gorm.DB) UserMFAPhoneRepository { return m }
func (m *mockMFAPhoneRepo) FindByUserID(int64) (*UserMFAPhone, error) {
	return m.findByUserID, m.findByUserIDErr
}
func (m *mockMFAPhoneRepo) DeleteByUserID(int64) error { return m.deleteErr }

type mockMFAEmailRepo struct {
	findByUserID    *UserMFAEmail
	findByUserIDErr error
	deleteErr       error
}

func (m *mockMFAEmailRepo) FindByUserID(int64) (*UserMFAEmail, error) {
	return m.findByUserID, m.findByUserIDErr
}
func (m *mockMFAEmailRepo) Create(r *UserMFAEmail) (*UserMFAEmail, error) { return r, nil }
func (m *mockMFAEmailRepo) Save(*UserMFAEmail) error                      { return nil }
func (m *mockMFAEmailRepo) DeleteByUserID(int64) error                    { return m.deleteErr }
