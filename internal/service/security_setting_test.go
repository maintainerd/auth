package service

import (
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
func newSecSetting(tenantID int64) *model.SecuritySetting {
	return &model.SecuritySetting{
		SecuritySettingID:   1,
		SecuritySettingUUID: uuid.New(),
		TenantID:            tenantID,
		GeneralConfig:       datatypes.JSON([]byte(`{}`)),
		PasswordConfig:      datatypes.JSON([]byte(`{}`)),
		SessionConfig:       datatypes.JSON([]byte(`{}`)),
		ThreatConfig:        datatypes.JSON([]byte(`{}`)),
		IPConfig:            datatypes.JSON([]byte(`{}`)),
		Version:             1,
	}
}

// ---------------------------------------------------------------------------
// GetByTenantID
// ---------------------------------------------------------------------------

func TestSecuritySettingService_GetByTenantID(t *testing.T) {
	t.Run("not found", func(t *testing.T) {
		svc := newSecuritySettingSvc(&mockSecuritySettingRepo{
			findByTenantIDFn: func(_ int64) (*model.SecuritySetting, error) { return nil, nil },
		}, &mockSecuritySettingsAuditRepo{})
		_, err := svc.GetByTenantID(1)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})

	t.Run("repo error", func(t *testing.T) {
		svc := newSecuritySettingSvc(&mockSecuritySettingRepo{
			findByTenantIDFn: func(_ int64) (*model.SecuritySetting, error) { return nil, errors.New("db") },
		}, &mockSecuritySettingsAuditRepo{})
		_, err := svc.GetByTenantID(1)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "db")
	})

	t.Run("success", func(t *testing.T) {
		svc := newSecuritySettingSvc(&mockSecuritySettingRepo{
			findByTenantIDFn: func(tid int64) (*model.SecuritySetting, error) {
				return newSecSetting(tid), nil
			},
		}, &mockSecuritySettingsAuditRepo{})
		res, err := svc.GetByTenantID(1)
		require.NoError(t, err)
		assert.Equal(t, int64(1), res.TenantID)
	})
}

// ---------------------------------------------------------------------------
// GetGeneralConfig
// ---------------------------------------------------------------------------

func TestSecuritySettingService_GetGeneralConfig(t *testing.T) {
	t.Run("repo error", func(t *testing.T) {
		svc := newSecuritySettingSvc(&mockSecuritySettingRepo{
			findByTenantIDFn: func(_ int64) (*model.SecuritySetting, error) { return nil, errors.New("fail") },
		}, &mockSecuritySettingsAuditRepo{})
		_, err := svc.GetGeneralConfig(1)
		require.Error(t, err)
	})

	t.Run("creates default when not found", func(t *testing.T) {
		svc := newSecuritySettingSvc(&mockSecuritySettingRepo{
			findByTenantIDFn: func(_ int64) (*model.SecuritySetting, error) { return nil, nil },
		}, &mockSecuritySettingsAuditRepo{})
		cfg, err := svc.GetGeneralConfig(1)
		require.NoError(t, err)
		assert.NotNil(t, cfg)
	})

	t.Run("create default error", func(t *testing.T) {
		svc := newSecuritySettingSvc(&mockSecuritySettingRepo{
			findByTenantIDFn: func(_ int64) (*model.SecuritySetting, error) { return nil, nil },
			createFn: func(_ *model.SecuritySetting) (*model.SecuritySetting, error) {
				return nil, errors.New("create error")
			},
		}, &mockSecuritySettingsAuditRepo{})
		_, err := svc.GetGeneralConfig(1)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "create error")
	})

	t.Run("success with existing", func(t *testing.T) {
		svc := newSecuritySettingSvc(&mockSecuritySettingRepo{
			findByTenantIDFn: func(_ int64) (*model.SecuritySetting, error) {
				return &model.SecuritySetting{
					GeneralConfig: datatypes.JSON([]byte(`{"key":"val"}`)),
				}, nil
			},
		}, &mockSecuritySettingsAuditRepo{})
		cfg, err := svc.GetGeneralConfig(1)
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
			findByTenantIDFn: func(_ int64) (*model.SecuritySetting, error) {
				return &model.SecuritySetting{
					PasswordConfig: datatypes.JSON([]byte(`{"min_length":8}`)),
				}, nil
			},
		}, &mockSecuritySettingsAuditRepo{})
		cfg, err := svc.GetPasswordConfig(1)
		require.NoError(t, err)
		assert.EqualValues(t, float64(8), cfg["min_length"])
	})

	t.Run("error", func(t *testing.T) {
		svc := newSecuritySettingSvc(&mockSecuritySettingRepo{
			findByTenantIDFn: func(_ int64) (*model.SecuritySetting, error) { return nil, errors.New("fail") },
		}, &mockSecuritySettingsAuditRepo{})
		_, err := svc.GetPasswordConfig(1)
		require.Error(t, err)
	})
}

// ---------------------------------------------------------------------------
// GetSessionConfig
// ---------------------------------------------------------------------------

func TestSecuritySettingService_GetSessionConfig(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		svc := newSecuritySettingSvc(&mockSecuritySettingRepo{
			findByTenantIDFn: func(_ int64) (*model.SecuritySetting, error) {
				return &model.SecuritySetting{
					SessionConfig: datatypes.JSON([]byte(`{"timeout":3600}`)),
				}, nil
			},
		}, &mockSecuritySettingsAuditRepo{})
		cfg, err := svc.GetSessionConfig(1)
		require.NoError(t, err)
		assert.EqualValues(t, float64(3600), cfg["timeout"])
	})

	t.Run("error", func(t *testing.T) {
		svc := newSecuritySettingSvc(&mockSecuritySettingRepo{
			findByTenantIDFn: func(_ int64) (*model.SecuritySetting, error) { return nil, errors.New("fail") },
		}, &mockSecuritySettingsAuditRepo{})
		_, err := svc.GetSessionConfig(1)
		require.Error(t, err)
	})
}

// ---------------------------------------------------------------------------
// GetThreatConfig
// ---------------------------------------------------------------------------

func TestSecuritySettingService_GetThreatConfig(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		svc := newSecuritySettingSvc(&mockSecuritySettingRepo{
			findByTenantIDFn: func(_ int64) (*model.SecuritySetting, error) {
				return &model.SecuritySetting{
					ThreatConfig: datatypes.JSON([]byte(`{"max_attempts":5}`)),
				}, nil
			},
		}, &mockSecuritySettingsAuditRepo{})
		cfg, err := svc.GetThreatConfig(1)
		require.NoError(t, err)
		assert.EqualValues(t, float64(5), cfg["max_attempts"])
	})

	t.Run("error", func(t *testing.T) {
		svc := newSecuritySettingSvc(&mockSecuritySettingRepo{
			findByTenantIDFn: func(_ int64) (*model.SecuritySetting, error) { return nil, errors.New("fail") },
		}, &mockSecuritySettingsAuditRepo{})
		_, err := svc.GetThreatConfig(1)
		require.Error(t, err)
	})
}

// ---------------------------------------------------------------------------
// GetIPConfig
// ---------------------------------------------------------------------------

func TestSecuritySettingService_GetIPConfig(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		svc := newSecuritySettingSvc(&mockSecuritySettingRepo{
			findByTenantIDFn: func(_ int64) (*model.SecuritySetting, error) {
				return &model.SecuritySetting{
					IPConfig: datatypes.JSON([]byte(`{"enabled":true}`)),
				}, nil
			},
		}, &mockSecuritySettingsAuditRepo{})
		cfg, err := svc.GetIPConfig(1)
		require.NoError(t, err)
		assert.Equal(t, true, cfg["enabled"])
	})

	t.Run("error", func(t *testing.T) {
		svc := newSecuritySettingSvc(&mockSecuritySettingRepo{
			findByTenantIDFn: func(_ int64) (*model.SecuritySetting, error) { return nil, errors.New("fail") },
		}, &mockSecuritySettingsAuditRepo{})
		_, err := svc.GetIPConfig(1)
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
// UpdateGeneralConfig – transactional (delegates to updateConfig)
// ---------------------------------------------------------------------------

func TestSecuritySettingService_UpdateGeneralConfig(t *testing.T) {
	tenantID := int64(1)
	updatedBy := int64(10)
	cfg := map[string]any{"enforce_mfa": true}

	t.Run("FindByTenantID error → rollback", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		svc := NewSecuritySettingService(db, &mockSecuritySettingRepo{
			findByTenantIDFn: func(_ int64) (*model.SecuritySetting, error) { return nil, errors.New("db") },
		}, &mockSecuritySettingsAuditRepo{})
		_, err := svc.UpdateGeneralConfig(tenantID, cfg, updatedBy, "1.2.3.4", "agent")
		require.Error(t, err)
	})

	t.Run("new setting → create + audit + success", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectCommit()
		settingUUID := uuid.New()
		svc := NewSecuritySettingService(db, &mockSecuritySettingRepo{
			findByTenantIDFn: func(_ int64) (*model.SecuritySetting, error) { return nil, nil },
			createFn: func(e *model.SecuritySetting) (*model.SecuritySetting, error) {
				e.SecuritySettingUUID = settingUUID
				e.SecuritySettingID = 1
				return e, nil
			},
			findByUUIDFn: func(_ any, _ ...string) (*model.SecuritySetting, error) {
				ss := newSecSetting(tenantID)
				ss.SecuritySettingUUID = settingUUID
				ss.GeneralConfig = datatypes.JSON([]byte(`{"enforce_mfa":true}`))
				ss.Version = 2
				return ss, nil
			},
		}, &mockSecuritySettingsAuditRepo{})
		res, err := svc.UpdateGeneralConfig(tenantID, cfg, updatedBy, "1.2.3.4", "agent")
		require.NoError(t, err)
		assert.Equal(t, 2, res.Version)
	})

	t.Run("existing setting → CreateOrUpdate + success", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectCommit()
		existing := newSecSetting(tenantID)
		svc := NewSecuritySettingService(db, &mockSecuritySettingRepo{
			findByTenantIDFn: func(_ int64) (*model.SecuritySetting, error) { return existing, nil },
			createOrUpdateFn: func(e *model.SecuritySetting) (*model.SecuritySetting, error) { return e, nil },
			findByUUIDFn: func(_ any, _ ...string) (*model.SecuritySetting, error) {
				return existing, nil
			},
		}, &mockSecuritySettingsAuditRepo{})
		res, err := svc.UpdateGeneralConfig(tenantID, cfg, updatedBy, "1.2.3.4", "agent")
		require.NoError(t, err)
		assert.NotNil(t, res)
	})

	t.Run("marshal error → rollback", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		existing := newSecSetting(tenantID)
		svc := NewSecuritySettingService(db, &mockSecuritySettingRepo{
			findByTenantIDFn: func(_ int64) (*model.SecuritySetting, error) { return existing, nil },
		}, &mockSecuritySettingsAuditRepo{})
		badCfg := map[string]any{"bad": math.Inf(1)}
		_, err := svc.UpdateGeneralConfig(tenantID, badCfg, updatedBy, "1.2.3.4", "agent")
		require.Error(t, err)
	})

	t.Run("CreateOrUpdate error → rollback", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		existing := newSecSetting(tenantID)
		svc := NewSecuritySettingService(db, &mockSecuritySettingRepo{
			findByTenantIDFn: func(_ int64) (*model.SecuritySetting, error) { return existing, nil },
			createOrUpdateFn: func(_ *model.SecuritySetting) (*model.SecuritySetting, error) {
				return nil, errors.New("save error")
			},
		}, &mockSecuritySettingsAuditRepo{})
		_, err := svc.UpdateGeneralConfig(tenantID, cfg, updatedBy, "1.2.3.4", "agent")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "save error")
	})

	t.Run("IncrementVersion error → rollback", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		existing := newSecSetting(tenantID)
		svc := NewSecuritySettingService(db, &mockSecuritySettingRepo{
			findByTenantIDFn:   func(_ int64) (*model.SecuritySetting, error) { return existing, nil },
			createOrUpdateFn:   func(e *model.SecuritySetting) (*model.SecuritySetting, error) { return e, nil },
			incrementVersionFn: func(_ int64) error { return errors.New("version error") },
		}, &mockSecuritySettingsAuditRepo{})
		_, err := svc.UpdateGeneralConfig(tenantID, cfg, updatedBy, "1.2.3.4", "agent")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "version error")
	})

	t.Run("audit Create error → rollback", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		existing := newSecSetting(tenantID)
		svc := NewSecuritySettingService(db, &mockSecuritySettingRepo{
			findByTenantIDFn: func(_ int64) (*model.SecuritySetting, error) { return existing, nil },
			createOrUpdateFn: func(e *model.SecuritySetting) (*model.SecuritySetting, error) { return e, nil },
		}, &mockSecuritySettingsAuditRepo{
			createFn: func(_ *model.SecuritySettingsAudit) (*model.SecuritySettingsAudit, error) {
				return nil, errors.New("audit error")
			},
		})
		_, err := svc.UpdateGeneralConfig(tenantID, cfg, updatedBy, "1.2.3.4", "agent")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "audit error")
	})

	t.Run("FindByUUID refresh error → returns error", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectCommit()
		existing := newSecSetting(tenantID)
		svc := NewSecuritySettingService(db, &mockSecuritySettingRepo{
			findByTenantIDFn: func(_ int64) (*model.SecuritySetting, error) { return existing, nil },
			createOrUpdateFn: func(e *model.SecuritySetting) (*model.SecuritySetting, error) { return e, nil },
			findByUUIDFn: func(_ any, _ ...string) (*model.SecuritySetting, error) {
				return nil, errors.New("refresh error")
			},
		}, &mockSecuritySettingsAuditRepo{})
		_, err := svc.UpdateGeneralConfig(tenantID, cfg, updatedBy, "1.2.3.4", "agent")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "refresh error")
	})

	t.Run("new setting → Create error → rollback", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		svc := NewSecuritySettingService(db, &mockSecuritySettingRepo{
			findByTenantIDFn: func(_ int64) (*model.SecuritySetting, error) { return nil, nil },
			createFn: func(_ *model.SecuritySetting) (*model.SecuritySetting, error) {
				return nil, errors.New("create error")
			},
		}, &mockSecuritySettingsAuditRepo{})
		_, err := svc.UpdateGeneralConfig(tenantID, cfg, updatedBy, "1.2.3.4", "agent")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "create error")
	})
}

// ---------------------------------------------------------------------------
// UpdatePasswordConfig
// ---------------------------------------------------------------------------

func TestSecuritySettingService_UpdatePasswordConfig(t *testing.T) {
	tenantID := int64(1)
	cfg := map[string]any{"min_length": 12}

	t.Run("success", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectCommit()
		existing := newSecSetting(tenantID)
		svc := NewSecuritySettingService(db, &mockSecuritySettingRepo{
			findByTenantIDFn: func(_ int64) (*model.SecuritySetting, error) { return existing, nil },
			createOrUpdateFn: func(e *model.SecuritySetting) (*model.SecuritySetting, error) { return e, nil },
			findByUUIDFn:     func(_ any, _ ...string) (*model.SecuritySetting, error) { return existing, nil },
		}, &mockSecuritySettingsAuditRepo{})
		res, err := svc.UpdatePasswordConfig(tenantID, cfg, 10, "1.2.3.4", "agent")
		require.NoError(t, err)
		assert.NotNil(t, res)
	})

	t.Run("error", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		svc := NewSecuritySettingService(db, &mockSecuritySettingRepo{
			findByTenantIDFn: func(_ int64) (*model.SecuritySetting, error) { return nil, errors.New("db") },
		}, &mockSecuritySettingsAuditRepo{})
		_, err := svc.UpdatePasswordConfig(tenantID, cfg, 10, "1.2.3.4", "agent")
		require.Error(t, err)
	})
}

// ---------------------------------------------------------------------------
// UpdateSessionConfig
// ---------------------------------------------------------------------------

func TestSecuritySettingService_UpdateSessionConfig(t *testing.T) {
	tenantID := int64(1)
	cfg := map[string]any{"timeout": 7200}

	t.Run("success", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectCommit()
		existing := newSecSetting(tenantID)
		svc := NewSecuritySettingService(db, &mockSecuritySettingRepo{
			findByTenantIDFn: func(_ int64) (*model.SecuritySetting, error) { return existing, nil },
			createOrUpdateFn: func(e *model.SecuritySetting) (*model.SecuritySetting, error) { return e, nil },
			findByUUIDFn:     func(_ any, _ ...string) (*model.SecuritySetting, error) { return existing, nil },
		}, &mockSecuritySettingsAuditRepo{})
		res, err := svc.UpdateSessionConfig(tenantID, cfg, 10, "1.2.3.4", "agent")
		require.NoError(t, err)
		assert.NotNil(t, res)
	})

	t.Run("error", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		svc := NewSecuritySettingService(db, &mockSecuritySettingRepo{
			findByTenantIDFn: func(_ int64) (*model.SecuritySetting, error) { return nil, errors.New("db") },
		}, &mockSecuritySettingsAuditRepo{})
		_, err := svc.UpdateSessionConfig(tenantID, cfg, 10, "1.2.3.4", "agent")
		require.Error(t, err)
	})
}

// ---------------------------------------------------------------------------
// UpdateThreatConfig
// ---------------------------------------------------------------------------

func TestSecuritySettingService_UpdateThreatConfig(t *testing.T) {
	tenantID := int64(1)
	cfg := map[string]any{"max_attempts": 10}

	t.Run("success", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectCommit()
		existing := newSecSetting(tenantID)
		svc := NewSecuritySettingService(db, &mockSecuritySettingRepo{
			findByTenantIDFn: func(_ int64) (*model.SecuritySetting, error) { return existing, nil },
			createOrUpdateFn: func(e *model.SecuritySetting) (*model.SecuritySetting, error) { return e, nil },
			findByUUIDFn:     func(_ any, _ ...string) (*model.SecuritySetting, error) { return existing, nil },
		}, &mockSecuritySettingsAuditRepo{})
		res, err := svc.UpdateThreatConfig(tenantID, cfg, 10, "1.2.3.4", "agent")
		require.NoError(t, err)
		assert.NotNil(t, res)
	})

	t.Run("error", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		svc := NewSecuritySettingService(db, &mockSecuritySettingRepo{
			findByTenantIDFn: func(_ int64) (*model.SecuritySetting, error) { return nil, errors.New("db") },
		}, &mockSecuritySettingsAuditRepo{})
		_, err := svc.UpdateThreatConfig(tenantID, cfg, 10, "1.2.3.4", "agent")
		require.Error(t, err)
	})
}

// ---------------------------------------------------------------------------
// UpdateIPConfig
// ---------------------------------------------------------------------------

func TestSecuritySettingService_UpdateIPConfig(t *testing.T) {
	tenantID := int64(1)
	cfg := map[string]any{"enabled": false}

	t.Run("success", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectCommit()
		existing := newSecSetting(tenantID)
		svc := NewSecuritySettingService(db, &mockSecuritySettingRepo{
			findByTenantIDFn: func(_ int64) (*model.SecuritySetting, error) { return existing, nil },
			createOrUpdateFn: func(e *model.SecuritySetting) (*model.SecuritySetting, error) { return e, nil },
			findByUUIDFn:     func(_ any, _ ...string) (*model.SecuritySetting, error) { return existing, nil },
		}, &mockSecuritySettingsAuditRepo{})
		res, err := svc.UpdateIPConfig(tenantID, cfg, 10, "1.2.3.4", "agent")
		require.NoError(t, err)
		assert.NotNil(t, res)
	})

	t.Run("error", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		svc := NewSecuritySettingService(db, &mockSecuritySettingRepo{
			findByTenantIDFn: func(_ int64) (*model.SecuritySetting, error) { return nil, errors.New("db") },
		}, &mockSecuritySettingsAuditRepo{})
		_, err := svc.UpdateIPConfig(tenantID, cfg, 10, "1.2.3.4", "agent")
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
			findByTenantIDFn: func(_ int64) (*model.SecuritySetting, error) {
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
