package service

import (
	"errors"
	"testing"

	"github.com/maintainerd/auth/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/datatypes"
)

func newSecuritySettingSvc(repo *mockSecuritySettingRepo, auditRepo *mockSecuritySettingsAuditRepo) SecuritySettingService {
	return NewSecuritySettingService(nil, repo, auditRepo)
}

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
				return &model.SecuritySetting{
					TenantID:       tid,
					GeneralConfig:  datatypes.JSON([]byte(`{}`)),
					PasswordConfig: datatypes.JSON([]byte(`{}`)),
					SessionConfig:  datatypes.JSON([]byte(`{}`)),
					ThreatConfig:   datatypes.JSON([]byte(`{}`)),
					IpConfig:       datatypes.JSON([]byte(`{}`)),
				}, nil
			},
		}, &mockSecuritySettingsAuditRepo{})
		res, err := svc.GetByTenantID(1)
		require.NoError(t, err)
		assert.Equal(t, int64(1), res.TenantID)
	})
}

func TestSecuritySettingService_GetGeneralConfig(t *testing.T) {
	t.Run("repo error", func(t *testing.T) {
		svc := newSecuritySettingSvc(&mockSecuritySettingRepo{
			findByTenantIDFn: func(_ int64) (*model.SecuritySetting, error) { return nil, errors.New("fail") },
		}, &mockSecuritySettingsAuditRepo{})
		_, err := svc.GetGeneralConfig(1)
		require.Error(t, err)
	})

	t.Run("creates default when not found", func(t *testing.T) {
		// findByTenantIDFn returns nil → service creates a default via Create
		svc := newSecuritySettingSvc(&mockSecuritySettingRepo{
			findByTenantIDFn: func(_ int64) (*model.SecuritySetting, error) { return nil, nil },
			// Create is implemented by default in mockSecuritySettingRepo (returns e, nil)
		}, &mockSecuritySettingsAuditRepo{})
		cfg, err := svc.GetGeneralConfig(1)
		require.NoError(t, err)
		assert.NotNil(t, cfg)
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

func TestSecuritySettingService_GetPasswordConfig(t *testing.T) {
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
}

func TestSecuritySettingService_GetSessionConfig(t *testing.T) {
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
}

func TestSecuritySettingService_GetThreatConfig(t *testing.T) {
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
}

func TestSecuritySettingService_GetIpConfig(t *testing.T) {
	svc := newSecuritySettingSvc(&mockSecuritySettingRepo{
		findByTenantIDFn: func(_ int64) (*model.SecuritySetting, error) {
			return &model.SecuritySetting{
				IpConfig: datatypes.JSON([]byte(`{"enabled":true}`)),
			}, nil
		},
	}, &mockSecuritySettingsAuditRepo{})
	cfg, err := svc.GetIpConfig(1)
	require.NoError(t, err)
	assert.Equal(t, true, cfg["enabled"])
}

