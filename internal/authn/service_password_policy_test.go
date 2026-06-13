package authn

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/maintainerd/auth/internal/platform/security"
	"github.com/maintainerd/auth/internal/secpolicy"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type mockPasswordHistoryRepo struct {
	addEntryFn         func(int64, string) error
	findRecentHashesFn func(int64, int) ([]string, error)
	pruneExcessFn      func(int64, int) error
}

func (m *mockPasswordHistoryRepo) WithTx(_ *gorm.DB) UserPasswordHistoryRepository { return m }
func (m *mockPasswordHistoryRepo) AddEntry(userID int64, hash string) error {
	if m.addEntryFn != nil {
		return m.addEntryFn(userID, hash)
	}
	return nil
}
func (m *mockPasswordHistoryRepo) FindRecentHashes(userID int64, count int) ([]string, error) {
	if m.findRecentHashesFn != nil {
		return m.findRecentHashesFn(userID, count)
	}
	return nil, nil
}
func (m *mockPasswordHistoryRepo) PruneExcess(userID int64, keepCount int) error {
	if m.pruneExcessFn != nil {
		return m.pruneExcessFn(userID, keepCount)
	}
	return nil
}

type mockSecuritySettingRepo struct {
	findDefaultByTenantIDFn func(int64) (*secpolicy.SecuritySetting, error)
}

func (m *mockSecuritySettingRepo) WithTx(_ *gorm.DB) secpolicy.SecuritySettingRepository { return m }
func (m *mockSecuritySettingRepo) FindDefaultByTenantID(tenantID int64) (*secpolicy.SecuritySetting, error) {
	if m.findDefaultByTenantIDFn != nil {
		return m.findDefaultByTenantIDFn(tenantID)
	}
	return nil, nil
}
func (m *mockSecuritySettingRepo) FindByTenantID(tenantID int64) (*secpolicy.SecuritySetting, error) {
	if m.findDefaultByTenantIDFn != nil {
		return m.findDefaultByTenantIDFn(tenantID)
	}
	return nil, nil
}
func (m *mockSecuritySettingRepo) FindPaginated(f secpolicy.SecuritySettingRepositoryGetFilter) (*PaginationResult[secpolicy.SecuritySetting], error) {
	return nil, nil
}
func (m *mockSecuritySettingRepo) IncrementVersion(id int64) error { return nil }
func (m *mockSecuritySettingRepo) Create(e *secpolicy.SecuritySetting) (*secpolicy.SecuritySetting, error) {
	return e, nil
}
func (m *mockSecuritySettingRepo) CreateOrUpdate(e *secpolicy.SecuritySetting) (*secpolicy.SecuritySetting, error) {
	return e, nil
}
func (m *mockSecuritySettingRepo) FindAll(...string) ([]secpolicy.SecuritySetting, error) {
	return nil, nil
}
func (m *mockSecuritySettingRepo) FindByUUID(any, ...string) (*secpolicy.SecuritySetting, error) {
	return nil, nil
}
func (m *mockSecuritySettingRepo) FindByUUIDs([]string, ...string) ([]secpolicy.SecuritySetting, error) {
	return nil, nil
}
func (m *mockSecuritySettingRepo) FindByID(any, ...string) (*secpolicy.SecuritySetting, error) {
	return nil, nil
}
func (m *mockSecuritySettingRepo) UpdateByUUID(any, any) (*secpolicy.SecuritySetting, error) {
	return nil, nil
}
func (m *mockSecuritySettingRepo) UpdateByID(any, any) (*secpolicy.SecuritySetting, error) {
	return nil, nil
}
func (m *mockSecuritySettingRepo) DeleteByUUID(any) error { return nil }
func (m *mockSecuritySettingRepo) DeleteByID(any) error   { return nil }
func (m *mockSecuritySettingRepo) Paginate(map[string]any, int, int, ...string) (*PaginationResult[secpolicy.SecuritySetting], error) {
	return nil, nil
}

func TestLoadPolicy(t *testing.T) {
	t.Run("nil repo returns default", func(t *testing.T) {
		policy := secpolicy.LoadPasswordPolicy(nil, 1)
		assert.Equal(t, expectedBusinessPasswordPolicy(), policy)
	})

	t.Run("repo error returns default", func(t *testing.T) {
		repo := &mockSecuritySettingRepo{
			findDefaultByTenantIDFn: func(int64) (*secpolicy.SecuritySetting, error) {
				return nil, errors.New("db error")
			},
		}
		policy := secpolicy.LoadPasswordPolicy(repo, 1)
		assert.Equal(t, expectedBusinessPasswordPolicy(), policy)
	})

	t.Run("repo returns nil returns default", func(t *testing.T) {
		repo := &mockSecuritySettingRepo{
			findDefaultByTenantIDFn: func(int64) (*secpolicy.SecuritySetting, error) {
				return nil, nil
			},
		}
		policy := secpolicy.LoadPasswordPolicy(repo, 1)
		assert.Equal(t, expectedBusinessPasswordPolicy(), policy)
	})
}

func expectedBusinessPasswordPolicy() security.PasswordPolicy {
	return security.PasswordPolicy{
		MinLength:                 12,
		MaxLength:                 128,
		RequireUpper:              false,
		RequireLower:              false,
		RequireDigit:              false,
		RequireSpecial:            false,
		BlocklistEnabled:          true,
		HistoryCount:              5,
		ExpiryDays:                0,
		CheckHIBP:                 true,
		MinStrengthScore:          2,
		HashAlgorithm:             "argon2id",
		TempPasswordValidityHours: 72,
	}
}

func TestCheckPasswordHistory(t *testing.T) {
	t.Run("nil repo returns nil", func(t *testing.T) {
		err := checkPasswordHistory(nil, 1, 5, "password")
		require.NoError(t, err)
	})

	t.Run("history count 0 returns nil", func(t *testing.T) {
		err := checkPasswordHistory(&mockPasswordHistoryRepo{}, 1, 0, "password")
		require.NoError(t, err)
	})

	t.Run("no matching hashes returns nil", func(t *testing.T) {
		hash, _ := bcrypt.GenerateFromPassword([]byte("oldpass"), bcrypt.MinCost)
		repo := &mockPasswordHistoryRepo{
			findRecentHashesFn: func(int64, int) ([]string, error) {
				return []string{string(hash)}, nil
			},
		}
		err := checkPasswordHistory(repo, 1, 5, "newpassword")
		require.NoError(t, err)
	})

	t.Run("matching hash returns error", func(t *testing.T) {
		hash, _ := bcrypt.GenerateFromPassword([]byte("reuseme"), bcrypt.MinCost)
		repo := &mockPasswordHistoryRepo{
			findRecentHashesFn: func(int64, int) ([]string, error) {
				return []string{string(hash)}, nil
			},
		}
		err := checkPasswordHistory(repo, 1, 5, "reuseme")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "recently")
	})

	t.Run("find recent hashes error returns error", func(t *testing.T) {
		repo := &mockPasswordHistoryRepo{
			findRecentHashesFn: func(int64, int) ([]string, error) {
				return nil, errors.New("db error")
			},
		}
		err := checkPasswordHistory(repo, 1, 5, "password")
		require.Error(t, err)
	})
}

func TestRecordPasswordHistory(t *testing.T) {
	t.Run("nil repo is no-op", func(t *testing.T) {
		secpolicy.RecordPasswordHistory(nil, 1, 5, "hash")
	})

	t.Run("history count 0 is no-op", func(t *testing.T) {
		secpolicy.RecordPasswordHistory(&mockPasswordHistoryRepo{}, 1, 0, "hash")
	})
}

func TestCheckPasswordExpiry(t *testing.T) {
	t.Run("nil securitySettingRepo is no-op", func(t *testing.T) {
		user := &User{UserID: 1}
		svc := &loginService{userRepo: &mockUserRepo{}}
		svc.checkPasswordExpiry(context.Background(), user, 1)
		assert.False(t, user.ForcePasswordChange)
	})

	t.Run("nil PasswordChangedAt is no-op", func(t *testing.T) {
		repo := &mockSecuritySettingRepo{
			findDefaultByTenantIDFn: func(int64) (*secpolicy.SecuritySetting, error) {
				return &secpolicy.SecuritySetting{PasswordConfig: datatypes.JSON([]byte(`{"expiry_days":90}`))}, nil
			},
		}
		user := &User{UserID: 1}
		svc := &loginService{securitySettingRepo: repo, userRepo: &mockUserRepo{}}
		svc.checkPasswordExpiry(context.Background(), user, 1)
		assert.False(t, user.ForcePasswordChange)
	})

	t.Run("expiry days 0 is no-op", func(t *testing.T) {
		repo := &mockSecuritySettingRepo{
			findDefaultByTenantIDFn: func(int64) (*secpolicy.SecuritySetting, error) {
				return &secpolicy.SecuritySetting{PasswordConfig: datatypes.JSON([]byte(`{"expiry_days":0}`))}, nil
			},
		}
		pwdChanged := time.Now()
		user := &User{UserID: 1, PasswordChangedAt: &pwdChanged}
		svc := &loginService{securitySettingRepo: repo, userRepo: &mockUserRepo{}}
		svc.checkPasswordExpiry(context.Background(), user, 1)
		assert.False(t, user.ForcePasswordChange)
	})

	t.Run("password expired sets ForcePasswordChange", func(t *testing.T) {
		repo := &mockSecuritySettingRepo{
			findDefaultByTenantIDFn: func(int64) (*secpolicy.SecuritySetting, error) {
				return &secpolicy.SecuritySetting{PasswordConfig: datatypes.JSON([]byte(`{"expiry_days":1}`))}, nil
			},
		}
		pwdChanged := time.Now().AddDate(0, 0, -2)
		user := &User{UserID: 1, PasswordChangedAt: &pwdChanged}
		ur := &mockUserRepo{
			updateByIDFn: func(_ any, data any) (*User, error) { return nil, nil },
		}
		svc := &loginService{securitySettingRepo: repo, userRepo: ur}
		svc.checkPasswordExpiry(context.Background(), user, 1)
		assert.True(t, user.ForcePasswordChange)
	})

	t.Run("password not expired does nothing", func(t *testing.T) {
		repo := &mockSecuritySettingRepo{
			findDefaultByTenantIDFn: func(int64) (*secpolicy.SecuritySetting, error) {
				return &secpolicy.SecuritySetting{PasswordConfig: datatypes.JSON([]byte(`{"expiry_days":90}`))}, nil
			},
		}
		pwdChanged := time.Now()
		user := &User{UserID: 1, PasswordChangedAt: &pwdChanged}
		svc := &loginService{securitySettingRepo: repo, userRepo: &mockUserRepo{}}
		svc.checkPasswordExpiry(context.Background(), user, 1)
		assert.False(t, user.ForcePasswordChange)
	})
}

func TestCheckTemporaryPasswordExpiry(t *testing.T) {
	t.Run("no temporary password expiry is no-op", func(t *testing.T) {
		user := &User{UserID: 1, ForcePasswordChange: true}
		svc := &loginService{}
		assert.NoError(t, svc.checkTemporaryPasswordExpiry(context.Background(), user, 1))
	})

	t.Run("unexpired temporary password is allowed to continue to password change", func(t *testing.T) {
		expiresAt := time.Now().Add(time.Hour)
		user := &User{UserID: 1, ForcePasswordChange: true, TemporaryPasswordExpiresAt: &expiresAt}
		svc := &loginService{}
		assert.NoError(t, svc.checkTemporaryPasswordExpiry(context.Background(), user, 1))
	})

	t.Run("expired temporary password is rejected", func(t *testing.T) {
		expiresAt := time.Now().Add(-time.Hour)
		user := &User{UserID: 1, ForcePasswordChange: true, TemporaryPasswordExpiresAt: &expiresAt}
		svc := &loginService{}
		err := svc.checkTemporaryPasswordExpiry(context.Background(), user, 1)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "temporary password has expired")
	})
}

func TestErrPasswordReusedError(t *testing.T) {
	assert.Contains(t, errPasswordReused.Error(), "recently")
}
