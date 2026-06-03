package mfa

import (
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/secpolicy"
	"github.com/pquerna/otp"
	"github.com/pquerna/otp/totp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
			svc := &mfaService{secSettingRepo: &mockSecuritySettingRepo{findByUserPoolID: tt.setting, findByUserPoolIDErr: tt.err}}

			got, err := svc.GetMFAPolicy(t.Context(), 7)

			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestMFAService_IsMFARequired(t *testing.T) {
	svc := &mfaService{secSettingRepo: &mockSecuritySettingRepo{
		findByUserPoolID: &secpolicy.SecuritySetting{
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
}

func (m *mockUserRepo) WithTx(*gorm.DB) UserRepository { return m }

func (m *mockUserRepo) FindByID(any, ...string) (*User, error) {
	return m.findByID, m.findByIDErr
}

func (m *mockUserRepo) FindByUUID(any, ...string) (*User, error) {
	return m.findByUUID, nil
}

type mockBackupCodeRepo struct {
	mockBaseRepositoryMethods[UserBackupCode]
	findUnused    []UserBackupCode
	findUnusedErr error
}

func (m *mockBackupCodeRepo) WithTx(*gorm.DB) UserBackupCodeRepository { return m }
func (m *mockBackupCodeRepo) CreateBulk([]*UserBackupCode) error       { return nil }
func (m *mockBackupCodeRepo) FindUnusedByUserID(int64) ([]UserBackupCode, error) {
	return m.findUnused, m.findUnusedErr
}
func (m *mockBackupCodeRepo) FindByUserIDAndCodeHash(int64, string) (*UserBackupCode, error) {
	return nil, nil
}
func (m *mockBackupCodeRepo) MarkUsed(int64) error          { return nil }
func (m *mockBackupCodeRepo) DeleteAllByUserID(int64) error { return nil }

type mockWebAuthnCredentialRepo struct {
	mockBaseRepositoryMethods[UserWebAuthnCredential]
	findByUserID    []UserWebAuthnCredential
	findByUserIDErr error
}

func (m *mockWebAuthnCredentialRepo) WithTx(*gorm.DB) UserWebAuthnCredentialRepository { return m }
func (m *mockWebAuthnCredentialRepo) FindByUserID(int64) ([]UserWebAuthnCredential, error) {
	return m.findByUserID, m.findByUserIDErr
}
func (m *mockWebAuthnCredentialRepo) FindByCredentialKeyID(string) (*UserWebAuthnCredential, error) {
	return nil, nil
}
func (m *mockWebAuthnCredentialRepo) CreateCredential(*UserWebAuthnCredential) error { return nil }
func (m *mockWebAuthnCredentialRepo) UpdateSignCount(int64, int64) error             { return nil }
func (m *mockWebAuthnCredentialRepo) UpdateLastUsed(int64) error                     { return nil }
func (m *mockWebAuthnCredentialRepo) DeleteCredentialByID(int64, int64) error        { return nil }
func (m *mockWebAuthnCredentialRepo) DeleteAllByUserID(int64) error                  { return nil }

type mockSecuritySettingRepo struct {
	mockBaseRepositoryMethods[secpolicy.SecuritySetting]
	findByUserPoolID    *secpolicy.SecuritySetting
	findByUserPoolIDErr error
}

func (m *mockSecuritySettingRepo) WithTx(*gorm.DB) secpolicy.SecuritySettingRepository { return m }
func (m *mockSecuritySettingRepo) FindByUserPoolID(int64) (*secpolicy.SecuritySetting, error) {
	return m.findByUserPoolID, m.findByUserPoolIDErr
}
func (m *mockSecuritySettingRepo) FindDefaultByTenantID(int64) (*secpolicy.SecuritySetting, error) {
	return nil, nil
}
func (m *mockSecuritySettingRepo) FindPaginated(secpolicy.SecuritySettingRepositoryGetFilter) (*PaginationResult[secpolicy.SecuritySetting], error) {
	return nil, nil
}
func (m *mockSecuritySettingRepo) IncrementVersion(int64) error { return nil }
