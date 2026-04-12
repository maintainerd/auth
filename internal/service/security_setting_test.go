package service

import (
	"context"
	"errors"
	"math"
	"testing"

	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/datatypes"
)

func newSecuritySettingSvc(repo *mockSecuritySettingRepo, auditRepo *mockSecuritySettingsAuditRepo) SecuritySettingService {
	return NewSecuritySettingService(nil, repo, auditRepo)
}

// helper: minimal SecuritySetting fixture
func newSecSetting(userPoolID int64) *model.SecuritySetting {
	return &model.SecuritySetting{
		SecuritySettingID:   1,
		SecuritySettingUUID: uuid.New(),
		UserPoolID:          userPoolID,
		MFAConfig:           datatypes.JSON([]byte(`{}`)),
		PasswordConfig:      datatypes.JSON([]byte(`{}`)),
		SessionConfig:       datatypes.JSON([]byte(`{}`)),
		ThreatConfig:        datatypes.JSON([]byte(`{}`)),
		LockoutConfig:       datatypes.JSON([]byte(`{}`)),
		RegistrationConfig:  datatypes.JSON([]byte(`{}`)),
		TokenConfig:         datatypes.JSON([]byte(`{}`)),
		Version:             1,
	}
}

// ---------------------------------------------------------------------------
// GetByUserPoolID
// ---------------------------------------------------------------------------

func TestSecuritySettingService_GetByUserPoolID(t *testing.T) {
	t.Run("not found", func(t *testing.T) {
		svc := newSecuritySettingSvc(&mockSecuritySettingRepo{
			findByUserPoolIDFn: func(_ int64) (*model.SecuritySetting, error) { return nil, nil },
		}, &mockSecuritySettingsAuditRepo{})
		_, err := svc.GetByUserPoolID(context.Background(), 1)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})

	t.Run("repo error", func(t *testing.T) {
		svc := newSecuritySettingSvc(&mockSecuritySettingRepo{
			findByUserPoolIDFn: func(_ int64) (*model.SecuritySetting, error) { return nil, errors.New("db") },
		}, &mockSecuritySettingsAuditRepo{})
		_, err := svc.GetByUserPoolID(context.Background(), 1)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "db")
	})

	t.Run("success", func(t *testing.T) {
		svc := newSecuritySettingSvc(&mockSecuritySettingRepo{
			findByUserPoolIDFn: func(tid int64) (*model.SecuritySetting, error) {
				return newSecSetting(tid), nil
			},
		}, &mockSecuritySettingsAuditRepo{})
		res, err := svc.GetByUserPoolID(context.Background(), 1)
		require.NoError(t, err)
		assert.Equal(t, int64(1), res.UserPoolID)
	})
}

// ---------------------------------------------------------------------------
// GetMFAConfig
// ---------------------------------------------------------------------------

func TestSecuritySettingService_GetMFAConfig(t *testing.T) {
	t.Run("repo error", func(t *testing.T) {
		svc := newSecuritySettingSvc(&mockSecuritySettingRepo{
			findByUserPoolIDFn: func(_ int64) (*model.SecuritySetting, error) { return nil, errors.New("fail") },
		}, &mockSecuritySettingsAuditRepo{})
		_, err := svc.GetMFAConfig(context.Background(), 1)
		require.Error(t, err)
	})

	t.Run("creates default when not found", func(t *testing.T) {
		svc := newSecuritySettingSvc(&mockSecuritySettingRepo{
			findByUserPoolIDFn: func(_ int64) (*model.SecuritySetting, error) { return nil, nil },
		}, &mockSecuritySettingsAuditRepo{})
		cfg, err := svc.GetMFAConfig(context.Background(), 1)
		require.NoError(t, err)
		assert.NotNil(t, cfg)
	})

	t.Run("create default error", func(t *testing.T) {
		svc := newSecuritySettingSvc(&mockSecuritySettingRepo{
			findByUserPoolIDFn: func(_ int64) (*model.SecuritySetting, error) { return nil, nil },
			createFn: func(_ *model.SecuritySetting) (*model.SecuritySetting, error) {
				return nil, errors.New("create error")
			},
		}, &mockSecuritySettingsAuditRepo{})
		_, err := svc.GetMFAConfig(context.Background(), 1)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "create error")
	})

	t.Run("success with existing", func(t *testing.T) {
		svc := newSecuritySettingSvc(&mockSecuritySettingRepo{
			findByUserPoolIDFn: func(_ int64) (*model.SecuritySetting, error) {
				return &model.SecuritySetting{
					MFAConfig: datatypes.JSON([]byte(`{"key":"val"}`)),
				}, nil
			},
		}, &mockSecuritySettingsAuditRepo{})
		cfg, err := svc.GetMFAConfig(context.Background(), 1)
		require.NoError(t, err)
		assert.Equal(t, "val", cfg["key"])
	})
}

// ---------------------------------------------------------------------------
// GetPasswordConfig
// ---------------------------------------------------------------------------

func TestSecuritySettingService_GetPasswordConfig(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		svc := newSecuritySettingSvc(&mockSecuritySettingRepo{
			findByUserPoolIDFn: func(_ int64) (*model.SecuritySetting, error) {
				return &model.SecuritySetting{
					PasswordConfig: datatypes.JSON([]byte(`{"min_length":8}`)),
				}, nil
			},
		}, &mockSecuritySettingsAuditRepo{})
		cfg, err := svc.GetPasswordConfig(context.Background(), 1)
		require.NoError(t, err)
		assert.EqualValues(t, float64(8), cfg["min_length"])
	})

	t.Run("error", func(t *testing.T) {
		svc := newSecuritySettingSvc(&mockSecuritySettingRepo{
			findByUserPoolIDFn: func(_ int64) (*model.SecuritySetting, error) { return nil, errors.New("fail") },
		}, &mockSecuritySettingsAuditRepo{})
		_, err := svc.GetPasswordConfig(context.Background(), 1)
		require.Error(t, err)
	})
}

// ---------------------------------------------------------------------------
// GetSessionConfig
// ---------------------------------------------------------------------------

func TestSecuritySettingService_GetSessionConfig(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		svc := newSecuritySettingSvc(&mockSecuritySettingRepo{
			findByUserPoolIDFn: func(_ int64) (*model.SecuritySetting, error) {
				return &model.SecuritySetting{
					SessionConfig: datatypes.JSON([]byte(`{"timeout":3600}`)),
				}, nil
			},
		}, &mockSecuritySettingsAuditRepo{})
		cfg, err := svc.GetSessionConfig(context.Background(), 1)
		require.NoError(t, err)
		assert.EqualValues(t, float64(3600), cfg["timeout"])
	})

	t.Run("error", func(t *testing.T) {
		svc := newSecuritySettingSvc(&mockSecuritySettingRepo{
			findByUserPoolIDFn: func(_ int64) (*model.SecuritySetting, error) { return nil, errors.New("fail") },
		}, &mockSecuritySettingsAuditRepo{})
		_, err := svc.GetSessionConfig(context.Background(), 1)
		require.Error(t, err)
	})
}

// ---------------------------------------------------------------------------
// GetThreatConfig
// ---------------------------------------------------------------------------

func TestSecuritySettingService_GetThreatConfig(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		svc := newSecuritySettingSvc(&mockSecuritySettingRepo{
			findByUserPoolIDFn: func(_ int64) (*model.SecuritySetting, error) {
				return &model.SecuritySetting{
					ThreatConfig: datatypes.JSON([]byte(`{"max_attempts":5}`)),
				}, nil
			},
		}, &mockSecuritySettingsAuditRepo{})
		cfg, err := svc.GetThreatConfig(context.Background(), 1)
		require.NoError(t, err)
		assert.EqualValues(t, float64(5), cfg["max_attempts"])
	})

	t.Run("error", func(t *testing.T) {
		svc := newSecuritySettingSvc(&mockSecuritySettingRepo{
			findByUserPoolIDFn: func(_ int64) (*model.SecuritySetting, error) { return nil, errors.New("fail") },
		}, &mockSecuritySettingsAuditRepo{})
		_, err := svc.GetThreatConfig(context.Background(), 1)
		require.Error(t, err)
	})
}

// ---------------------------------------------------------------------------
// GetLockoutConfig
// ---------------------------------------------------------------------------

func TestSecuritySettingService_GetLockoutConfig(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		svc := newSecuritySettingSvc(&mockSecuritySettingRepo{
			findByUserPoolIDFn: func(_ int64) (*model.SecuritySetting, error) {
				return &model.SecuritySetting{
					LockoutConfig: datatypes.JSON([]byte(`{"enabled":true}`)),
				}, nil
			},
		}, &mockSecuritySettingsAuditRepo{})
		cfg, err := svc.GetLockoutConfig(context.Background(), 1)
		require.NoError(t, err)
		assert.Equal(t, true, cfg["enabled"])
	})

	t.Run("error", func(t *testing.T) {
		svc := newSecuritySettingSvc(&mockSecuritySettingRepo{
			findByUserPoolIDFn: func(_ int64) (*model.SecuritySetting, error) { return nil, errors.New("fail") },
		}, &mockSecuritySettingsAuditRepo{})
		_, err := svc.GetLockoutConfig(context.Background(), 1)
		require.Error(t, err)
	})
}

// ---------------------------------------------------------------------------
// unmarshalJSON
// ---------------------------------------------------------------------------

func TestUnmarshalJSON(t *testing.T) {
	t.Run("nil/empty → empty map", func(t *testing.T) {
		result := unmarshalJSON(nil)
		assert.NotNil(t, result)
		assert.Empty(t, result)
	})

	t.Run("invalid JSON → empty map", func(t *testing.T) {
		result := unmarshalJSON(datatypes.JSON([]byte("not-json")))
		assert.NotNil(t, result)
		assert.Empty(t, result)
	})

	t.Run("valid JSON", func(t *testing.T) {
		result := unmarshalJSON(datatypes.JSON([]byte(`{"a":"b"}`)))
		assert.Equal(t, "b", result["a"])
	})
}

// ---------------------------------------------------------------------------
// UpdateMFAConfig – transactional (delegates to updateConfig)
// ---------------------------------------------------------------------------

func TestSecuritySettingService_UpdateMFAConfig(t *testing.T) {
	userPoolID := int64(1)
	updatedBy := int64(10)
	cfg := map[string]any{"enforce_mfa": true}

	t.Run("FindByUserPoolID error → rollback", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		svc := NewSecuritySettingService(db, &mockSecuritySettingRepo{
			findByUserPoolIDFn: func(_ int64) (*model.SecuritySetting, error) { return nil, errors.New("db") },
		}, &mockSecuritySettingsAuditRepo{})
		_, err := svc.UpdateMFAConfig(context.Background(), userPoolID, cfg, updatedBy, "1.2.3.4", "agent")
		require.Error(t, err)
	})

	t.Run("new setting → create + audit + success", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectCommit()
		settingUUID := uuid.New()
		svc := NewSecuritySettingService(db, &mockSecuritySettingRepo{
			findByUserPoolIDFn: func(_ int64) (*model.SecuritySetting, error) { return nil, nil },
			createFn: func(e *model.SecuritySetting) (*model.SecuritySetting, error) {
				e.SecuritySettingUUID = settingUUID
				e.SecuritySettingID = 1
				return e, nil
			},
			findByUUIDFn: func(_ any, _ ...string) (*model.SecuritySetting, error) {
				ss := newSecSetting(userPoolID)
				ss.SecuritySettingUUID = settingUUID
				ss.MFAConfig = datatypes.JSON([]byte(`{"enforce_mfa":true}`))
				ss.Version = 2
				return ss, nil
			},
		}, &mockSecuritySettingsAuditRepo{})
		res, err := svc.UpdateMFAConfig(context.Background(), userPoolID, cfg, updatedBy, "1.2.3.4", "agent")
		require.NoError(t, err)
		assert.Equal(t, 2, res.Version)
	})

	t.Run("existing setting → CreateOrUpdate + success", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectCommit()
		existing := newSecSetting(userPoolID)
		svc := NewSecuritySettingService(db, &mockSecuritySettingRepo{
			findByUserPoolIDFn: func(_ int64) (*model.SecuritySetting, error) { return existing, nil },
			createOrUpdateFn:   func(e *model.SecuritySetting) (*model.SecuritySetting, error) { return e, nil },
			findByUUIDFn: func(_ any, _ ...string) (*model.SecuritySetting, error) {
				return existing, nil
			},
		}, &mockSecuritySettingsAuditRepo{})
		res, err := svc.UpdateMFAConfig(context.Background(), userPoolID, cfg, updatedBy, "1.2.3.4", "agent")
		require.NoError(t, err)
		assert.NotNil(t, res)
	})

	t.Run("marshal error → rollback", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		existing := newSecSetting(userPoolID)
		svc := NewSecuritySettingService(db, &mockSecuritySettingRepo{
			findByUserPoolIDFn: func(_ int64) (*model.SecuritySetting, error) { return existing, nil },
		}, &mockSecuritySettingsAuditRepo{})
		badCfg := map[string]any{"bad": math.Inf(1)}
		_, err := svc.UpdateMFAConfig(context.Background(), userPoolID, badCfg, updatedBy, "1.2.3.4", "agent")
		require.Error(t, err)
	})

	t.Run("CreateOrUpdate error → rollback", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		existing := newSecSetting(userPoolID)
		svc := NewSecuritySettingService(db, &mockSecuritySettingRepo{
			findByUserPoolIDFn: func(_ int64) (*model.SecuritySetting, error) { return existing, nil },
			createOrUpdateFn: func(_ *model.SecuritySetting) (*model.SecuritySetting, error) {
				return nil, errors.New("save error")
			},
		}, &mockSecuritySettingsAuditRepo{})
		_, err := svc.UpdateMFAConfig(context.Background(), userPoolID, cfg, updatedBy, "1.2.3.4", "agent")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "save error")
	})

	t.Run("IncrementVersion error → rollback", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		existing := newSecSetting(userPoolID)
		svc := NewSecuritySettingService(db, &mockSecuritySettingRepo{
			findByUserPoolIDFn: func(_ int64) (*model.SecuritySetting, error) { return existing, nil },
			createOrUpdateFn:   func(e *model.SecuritySetting) (*model.SecuritySetting, error) { return e, nil },
			incrementVersionFn: func(_ int64) error { return errors.New("version error") },
		}, &mockSecuritySettingsAuditRepo{})
		_, err := svc.UpdateMFAConfig(context.Background(), userPoolID, cfg, updatedBy, "1.2.3.4", "agent")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "version error")
	})

	t.Run("audit Create error → rollback", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		existing := newSecSetting(userPoolID)
		svc := NewSecuritySettingService(db, &mockSecuritySettingRepo{
			findByUserPoolIDFn: func(_ int64) (*model.SecuritySetting, error) { return existing, nil },
			createOrUpdateFn:   func(e *model.SecuritySetting) (*model.SecuritySetting, error) { return e, nil },
		}, &mockSecuritySettingsAuditRepo{
			createFn: func(_ *model.SecuritySettingsAudit) (*model.SecuritySettingsAudit, error) {
				return nil, errors.New("audit error")
			},
		})
		_, err := svc.UpdateMFAConfig(context.Background(), userPoolID, cfg, updatedBy, "1.2.3.4", "agent")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "audit error")
	})

	t.Run("FindByUUID refresh error → returns error", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectCommit()
		existing := newSecSetting(userPoolID)
		svc := NewSecuritySettingService(db, &mockSecuritySettingRepo{
			findByUserPoolIDFn: func(_ int64) (*model.SecuritySetting, error) { return existing, nil },
			createOrUpdateFn:   func(e *model.SecuritySetting) (*model.SecuritySetting, error) { return e, nil },
			findByUUIDFn: func(_ any, _ ...string) (*model.SecuritySetting, error) {
				return nil, errors.New("refresh error")
			},
		}, &mockSecuritySettingsAuditRepo{})
		_, err := svc.UpdateMFAConfig(context.Background(), userPoolID, cfg, updatedBy, "1.2.3.4", "agent")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "refresh error")
	})

	t.Run("new setting → Create error → rollback", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		svc := NewSecuritySettingService(db, &mockSecuritySettingRepo{
			findByUserPoolIDFn: func(_ int64) (*model.SecuritySetting, error) { return nil, nil },
			createFn: func(_ *model.SecuritySetting) (*model.SecuritySetting, error) {
				return nil, errors.New("create error")
			},
		}, &mockSecuritySettingsAuditRepo{})
		_, err := svc.UpdateMFAConfig(context.Background(), userPoolID, cfg, updatedBy, "1.2.3.4", "agent")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "create error")
	})
}

// ---------------------------------------------------------------------------
// UpdatePasswordConfig
// ---------------------------------------------------------------------------

func TestSecuritySettingService_UpdatePasswordConfig(t *testing.T) {
	userPoolID := int64(1)
	cfg := map[string]any{"min_length": 12}

	t.Run("success", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectCommit()
		existing := newSecSetting(userPoolID)
		svc := NewSecuritySettingService(db, &mockSecuritySettingRepo{
			findByUserPoolIDFn: func(_ int64) (*model.SecuritySetting, error) { return existing, nil },
			createOrUpdateFn:   func(e *model.SecuritySetting) (*model.SecuritySetting, error) { return e, nil },
			findByUUIDFn:       func(_ any, _ ...string) (*model.SecuritySetting, error) { return existing, nil },
		}, &mockSecuritySettingsAuditRepo{})
		res, err := svc.UpdatePasswordConfig(context.Background(), userPoolID, cfg, 10, "1.2.3.4", "agent")
		require.NoError(t, err)
		assert.NotNil(t, res)
	})

	t.Run("error", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		svc := NewSecuritySettingService(db, &mockSecuritySettingRepo{
			findByUserPoolIDFn: func(_ int64) (*model.SecuritySetting, error) { return nil, errors.New("db") },
		}, &mockSecuritySettingsAuditRepo{})
		_, err := svc.UpdatePasswordConfig(context.Background(), userPoolID, cfg, 10, "1.2.3.4", "agent")
		require.Error(t, err)
	})
}

// ---------------------------------------------------------------------------
// UpdateSessionConfig
// ---------------------------------------------------------------------------

func TestSecuritySettingService_UpdateSessionConfig(t *testing.T) {
	userPoolID := int64(1)
	cfg := map[string]any{"timeout": 7200}

	t.Run("success", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectCommit()
		existing := newSecSetting(userPoolID)
		svc := NewSecuritySettingService(db, &mockSecuritySettingRepo{
			findByUserPoolIDFn: func(_ int64) (*model.SecuritySetting, error) { return existing, nil },
			createOrUpdateFn:   func(e *model.SecuritySetting) (*model.SecuritySetting, error) { return e, nil },
			findByUUIDFn:       func(_ any, _ ...string) (*model.SecuritySetting, error) { return existing, nil },
		}, &mockSecuritySettingsAuditRepo{})
		res, err := svc.UpdateSessionConfig(context.Background(), userPoolID, cfg, 10, "1.2.3.4", "agent")
		require.NoError(t, err)
		assert.NotNil(t, res)
	})

	t.Run("error", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		svc := NewSecuritySettingService(db, &mockSecuritySettingRepo{
			findByUserPoolIDFn: func(_ int64) (*model.SecuritySetting, error) { return nil, errors.New("db") },
		}, &mockSecuritySettingsAuditRepo{})
		_, err := svc.UpdateSessionConfig(context.Background(), userPoolID, cfg, 10, "1.2.3.4", "agent")
		require.Error(t, err)
	})
}

// ---------------------------------------------------------------------------
// UpdateThreatConfig
// ---------------------------------------------------------------------------

func TestSecuritySettingService_UpdateThreatConfig(t *testing.T) {
	userPoolID := int64(1)
	cfg := map[string]any{"max_attempts": 10}

	t.Run("success", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectCommit()
		existing := newSecSetting(userPoolID)
		svc := NewSecuritySettingService(db, &mockSecuritySettingRepo{
			findByUserPoolIDFn: func(_ int64) (*model.SecuritySetting, error) { return existing, nil },
			createOrUpdateFn:   func(e *model.SecuritySetting) (*model.SecuritySetting, error) { return e, nil },
			findByUUIDFn:       func(_ any, _ ...string) (*model.SecuritySetting, error) { return existing, nil },
		}, &mockSecuritySettingsAuditRepo{})
		res, err := svc.UpdateThreatConfig(context.Background(), userPoolID, cfg, 10, "1.2.3.4", "agent")
		require.NoError(t, err)
		assert.NotNil(t, res)
	})

	t.Run("error", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		svc := NewSecuritySettingService(db, &mockSecuritySettingRepo{
			findByUserPoolIDFn: func(_ int64) (*model.SecuritySetting, error) { return nil, errors.New("db") },
		}, &mockSecuritySettingsAuditRepo{})
		_, err := svc.UpdateThreatConfig(context.Background(), userPoolID, cfg, 10, "1.2.3.4", "agent")
		require.Error(t, err)
	})
}

// ---------------------------------------------------------------------------
// UpdateLockoutConfig
// ---------------------------------------------------------------------------

func TestSecuritySettingService_UpdateLockoutConfig(t *testing.T) {
	userPoolID := int64(1)
	cfg := map[string]any{"enabled": false}

	t.Run("success", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectCommit()
		existing := newSecSetting(userPoolID)
		svc := NewSecuritySettingService(db, &mockSecuritySettingRepo{
			findByUserPoolIDFn: func(_ int64) (*model.SecuritySetting, error) { return existing, nil },
			createOrUpdateFn:   func(e *model.SecuritySetting) (*model.SecuritySetting, error) { return e, nil },
			findByUUIDFn:       func(_ any, _ ...string) (*model.SecuritySetting, error) { return existing, nil },
		}, &mockSecuritySettingsAuditRepo{})
		res, err := svc.UpdateLockoutConfig(context.Background(), userPoolID, cfg, 10, "1.2.3.4", "agent")
		require.NoError(t, err)
		assert.NotNil(t, res)
	})

	t.Run("error", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		svc := NewSecuritySettingService(db, &mockSecuritySettingRepo{
			findByUserPoolIDFn: func(_ int64) (*model.SecuritySetting, error) { return nil, errors.New("db") },
		}, &mockSecuritySettingsAuditRepo{})
		_, err := svc.UpdateLockoutConfig(context.Background(), userPoolID, cfg, 10, "1.2.3.4", "agent")
		require.Error(t, err)
	})
}

// ---------------------------------------------------------------------------
// updateConfig – unreachable default case
// ---------------------------------------------------------------------------

func TestSecuritySettingService_UpdateConfig_InvalidConfigType(t *testing.T) {
	db, mock := newMockGormDB(t)
	mock.ExpectBegin()
	mock.ExpectRollback()
	svc := &securitySettingService{
		db: db,
		securitySettingRepo: &mockSecuritySettingRepo{
			findByUserPoolIDFn: func(_ int64) (*model.SecuritySetting, error) {
				return newSecSetting(1), nil
			},
		},
		securitySettingsAuditRepo: &mockSecuritySettingsAuditRepo{},
	}
	_, err := svc.updateConfig(1, "invalid_type", map[string]any{"key": "val"}, 10, "1.2.3.4", "agent")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid config type")
	assert.NoError(t, mock.ExpectationsWereMet())
}
