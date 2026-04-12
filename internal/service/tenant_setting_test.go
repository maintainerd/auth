package service

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/datatypes"
)

func newTenantSettingSvc(repo *mockTenantSettingRepo) TenantSettingService {
	return NewTenantSettingService(repo)
}

func newTenantSetting(tenantID int64) *model.TenantSetting {
	return &model.TenantSetting{
		TenantSettingID:   1,
		TenantSettingUUID: uuid.New(),
		TenantID:          tenantID,
		RateLimitConfig:   datatypes.JSON([]byte(`{"max_rps":100}`)),
		AuditConfig:       datatypes.JSON([]byte(`{"enabled":true}`)),
		MaintenanceConfig: datatypes.JSON([]byte(`{"active":false}`)),
		FeatureFlags:      datatypes.JSON([]byte(`{"dark_mode":true}`)),
	}
}

// ---------------------------------------------------------------------------
// Get
// ---------------------------------------------------------------------------

func TestTenantSettingService_Get(t *testing.T) {
	t.Run("existing record", func(t *testing.T) {
		ts := newTenantSetting(1)
		svc := newTenantSettingSvc(&mockTenantSettingRepo{
			findByTenantIDFn: func(_ int64) (*model.TenantSetting, error) { return ts, nil },
		})
		res, err := svc.Get(context.Background(), 1)
		require.NoError(t, err)
		assert.Equal(t, ts.TenantSettingUUID, res.TenantSettingUUID)
		assert.NotNil(t, res.RateLimitConfig)
	})

	t.Run("auto-creates default when not found", func(t *testing.T) {
		svc := newTenantSettingSvc(&mockTenantSettingRepo{
			findByTenantIDFn: func(_ int64) (*model.TenantSetting, error) { return nil, nil },
			createFn: func(e *model.TenantSetting) (*model.TenantSetting, error) {
				e.TenantSettingUUID = uuid.New()
				return e, nil
			},
		})
		res, err := svc.Get(context.Background(), 1)
		require.NoError(t, err)
		assert.NotNil(t, res)
	})

	t.Run("FindByTenantID error", func(t *testing.T) {
		svc := newTenantSettingSvc(&mockTenantSettingRepo{
			findByTenantIDFn: func(_ int64) (*model.TenantSetting, error) { return nil, errors.New("db") },
		})
		_, err := svc.Get(context.Background(), 1)
		require.Error(t, err)
	})

	t.Run("create default error", func(t *testing.T) {
		svc := newTenantSettingSvc(&mockTenantSettingRepo{
			findByTenantIDFn: func(_ int64) (*model.TenantSetting, error) { return nil, nil },
			createFn: func(_ *model.TenantSetting) (*model.TenantSetting, error) {
				return nil, errors.New("create fail")
			},
		})
		_, err := svc.Get(context.Background(), 1)
		require.Error(t, err)
	})
}

// ---------------------------------------------------------------------------
// GetRateLimitConfig / GetAuditConfig / GetMaintenanceConfig / GetFeatureFlags
// ---------------------------------------------------------------------------

func TestTenantSettingService_GetRateLimitConfig(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		ts := newTenantSetting(1)
		svc := newTenantSettingSvc(&mockTenantSettingRepo{
			findByTenantIDFn: func(_ int64) (*model.TenantSetting, error) { return ts, nil },
		})
		cfg, err := svc.GetRateLimitConfig(context.Background(), 1)
		require.NoError(t, err)
		assert.Equal(t, float64(100), cfg["max_rps"])
	})

	t.Run("repo error", func(t *testing.T) {
		svc := newTenantSettingSvc(&mockTenantSettingRepo{
			findByTenantIDFn: func(_ int64) (*model.TenantSetting, error) { return nil, errors.New("fail") },
		})
		_, err := svc.GetRateLimitConfig(context.Background(), 1)
		require.Error(t, err)
	})

	t.Run("auto-creates when missing", func(t *testing.T) {
		svc := newTenantSettingSvc(&mockTenantSettingRepo{
			findByTenantIDFn: func(_ int64) (*model.TenantSetting, error) { return nil, nil },
			createFn: func(e *model.TenantSetting) (*model.TenantSetting, error) {
				e.TenantSettingUUID = uuid.New()
				return e, nil
			},
		})
		cfg, err := svc.GetRateLimitConfig(context.Background(), 1)
		require.NoError(t, err)
		assert.NotNil(t, cfg)
	})
}

func TestTenantSettingService_GetAuditConfig(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		ts := newTenantSetting(1)
		svc := newTenantSettingSvc(&mockTenantSettingRepo{
			findByTenantIDFn: func(_ int64) (*model.TenantSetting, error) { return ts, nil },
		})
		cfg, err := svc.GetAuditConfig(context.Background(), 1)
		require.NoError(t, err)
		assert.Equal(t, true, cfg["enabled"])
	})

	t.Run("repo error", func(t *testing.T) {
		svc := newTenantSettingSvc(&mockTenantSettingRepo{
			findByTenantIDFn: func(_ int64) (*model.TenantSetting, error) { return nil, errors.New("fail") },
		})
		_, err := svc.GetAuditConfig(context.Background(), 1)
		require.Error(t, err)
	})
}

func TestTenantSettingService_GetMaintenanceConfig(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		ts := newTenantSetting(1)
		svc := newTenantSettingSvc(&mockTenantSettingRepo{
			findByTenantIDFn: func(_ int64) (*model.TenantSetting, error) { return ts, nil },
		})
		cfg, err := svc.GetMaintenanceConfig(context.Background(), 1)
		require.NoError(t, err)
		assert.Equal(t, false, cfg["active"])
	})

	t.Run("repo error", func(t *testing.T) {
		svc := newTenantSettingSvc(&mockTenantSettingRepo{
			findByTenantIDFn: func(_ int64) (*model.TenantSetting, error) { return nil, errors.New("fail") },
		})
		_, err := svc.GetMaintenanceConfig(context.Background(), 1)
		require.Error(t, err)
	})
}

func TestTenantSettingService_GetFeatureFlags(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		ts := newTenantSetting(1)
		svc := newTenantSettingSvc(&mockTenantSettingRepo{
			findByTenantIDFn: func(_ int64) (*model.TenantSetting, error) { return ts, nil },
		})
		cfg, err := svc.GetFeatureFlags(context.Background(), 1)
		require.NoError(t, err)
		assert.Equal(t, true, cfg["dark_mode"])
	})

	t.Run("repo error", func(t *testing.T) {
		svc := newTenantSettingSvc(&mockTenantSettingRepo{
			findByTenantIDFn: func(_ int64) (*model.TenantSetting, error) { return nil, errors.New("fail") },
		})
		_, err := svc.GetFeatureFlags(context.Background(), 1)
		require.Error(t, err)
	})
}

// ---------------------------------------------------------------------------
// UpdateRateLimitConfig / UpdateAuditConfig / UpdateMaintenanceConfig /
// UpdateFeatureFlags
// ---------------------------------------------------------------------------

func TestTenantSettingService_UpdateRateLimitConfig(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		ts := newTenantSetting(1)
		svc := newTenantSettingSvc(&mockTenantSettingRepo{
			findByTenantIDFn: func(_ int64) (*model.TenantSetting, error) { return ts, nil },
			createOrUpdateFn: func(e *model.TenantSetting) (*model.TenantSetting, error) { return e, nil },
		})
		res, err := svc.UpdateRateLimitConfig(context.Background(), 1, map[string]any{"max_rps": 200})
		require.NoError(t, err)
		assert.NotNil(t, res)
	})

	t.Run("getOrCreate error", func(t *testing.T) {
		svc := newTenantSettingSvc(&mockTenantSettingRepo{
			findByTenantIDFn: func(_ int64) (*model.TenantSetting, error) { return nil, errors.New("db") },
		})
		_, err := svc.UpdateRateLimitConfig(context.Background(), 1, map[string]any{})
		require.Error(t, err)
	})

	t.Run("CreateOrUpdate error", func(t *testing.T) {
		ts := newTenantSetting(1)
		svc := newTenantSettingSvc(&mockTenantSettingRepo{
			findByTenantIDFn: func(_ int64) (*model.TenantSetting, error) { return ts, nil },
			createOrUpdateFn: func(_ *model.TenantSetting) (*model.TenantSetting, error) {
				return nil, errors.New("save err")
			},
		})
		_, err := svc.UpdateRateLimitConfig(context.Background(), 1, map[string]any{})
		require.Error(t, err)
	})
}

func TestTenantSettingService_UpdateAuditConfig(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		ts := newTenantSetting(1)
		svc := newTenantSettingSvc(&mockTenantSettingRepo{
			findByTenantIDFn: func(_ int64) (*model.TenantSetting, error) { return ts, nil },
			createOrUpdateFn: func(e *model.TenantSetting) (*model.TenantSetting, error) { return e, nil },
		})
		res, err := svc.UpdateAuditConfig(context.Background(), 1, map[string]any{"enabled": false})
		require.NoError(t, err)
		assert.NotNil(t, res)
	})

	t.Run("getOrCreate error", func(t *testing.T) {
		svc := newTenantSettingSvc(&mockTenantSettingRepo{
			findByTenantIDFn: func(_ int64) (*model.TenantSetting, error) { return nil, errors.New("db") },
		})
		_, err := svc.UpdateAuditConfig(context.Background(), 1, map[string]any{})
		require.Error(t, err)
	})

	t.Run("CreateOrUpdate error", func(t *testing.T) {
		ts := newTenantSetting(1)
		svc := newTenantSettingSvc(&mockTenantSettingRepo{
			findByTenantIDFn: func(_ int64) (*model.TenantSetting, error) { return ts, nil },
			createOrUpdateFn: func(_ *model.TenantSetting) (*model.TenantSetting, error) {
				return nil, errors.New("save")
			},
		})
		_, err := svc.UpdateAuditConfig(context.Background(), 1, map[string]any{})
		require.Error(t, err)
	})
}

func TestTenantSettingService_UpdateMaintenanceConfig(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		ts := newTenantSetting(1)
		svc := newTenantSettingSvc(&mockTenantSettingRepo{
			findByTenantIDFn: func(_ int64) (*model.TenantSetting, error) { return ts, nil },
			createOrUpdateFn: func(e *model.TenantSetting) (*model.TenantSetting, error) { return e, nil },
		})
		res, err := svc.UpdateMaintenanceConfig(context.Background(), 1, map[string]any{"active": true})
		require.NoError(t, err)
		assert.NotNil(t, res)
	})

	t.Run("getOrCreate error", func(t *testing.T) {
		svc := newTenantSettingSvc(&mockTenantSettingRepo{
			findByTenantIDFn: func(_ int64) (*model.TenantSetting, error) { return nil, errors.New("db") },
		})
		_, err := svc.UpdateMaintenanceConfig(context.Background(), 1, map[string]any{})
		require.Error(t, err)
	})

	t.Run("CreateOrUpdate error", func(t *testing.T) {
		ts := newTenantSetting(1)
		svc := newTenantSettingSvc(&mockTenantSettingRepo{
			findByTenantIDFn: func(_ int64) (*model.TenantSetting, error) { return ts, nil },
			createOrUpdateFn: func(_ *model.TenantSetting) (*model.TenantSetting, error) {
				return nil, errors.New("save")
			},
		})
		_, err := svc.UpdateMaintenanceConfig(context.Background(), 1, map[string]any{})
		require.Error(t, err)
	})
}

func TestTenantSettingService_UpdateFeatureFlags(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		ts := newTenantSetting(1)
		svc := newTenantSettingSvc(&mockTenantSettingRepo{
			findByTenantIDFn: func(_ int64) (*model.TenantSetting, error) { return ts, nil },
			createOrUpdateFn: func(e *model.TenantSetting) (*model.TenantSetting, error) { return e, nil },
		})
		res, err := svc.UpdateFeatureFlags(context.Background(), 1, map[string]any{"dark_mode": false})
		require.NoError(t, err)
		assert.NotNil(t, res)
	})

	t.Run("getOrCreate error", func(t *testing.T) {
		svc := newTenantSettingSvc(&mockTenantSettingRepo{
			findByTenantIDFn: func(_ int64) (*model.TenantSetting, error) { return nil, errors.New("db") },
		})
		_, err := svc.UpdateFeatureFlags(context.Background(), 1, map[string]any{})
		require.Error(t, err)
	})

	t.Run("CreateOrUpdate error", func(t *testing.T) {
		ts := newTenantSetting(1)
		svc := newTenantSettingSvc(&mockTenantSettingRepo{
			findByTenantIDFn: func(_ int64) (*model.TenantSetting, error) { return ts, nil },
			createOrUpdateFn: func(_ *model.TenantSetting) (*model.TenantSetting, error) {
				return nil, errors.New("save")
			},
		})
		_, err := svc.UpdateFeatureFlags(context.Background(), 1, map[string]any{})
		require.Error(t, err)
	})
}

// ---------------------------------------------------------------------------
// updateConfig — invalid configType (covers the default switch branch)
// ---------------------------------------------------------------------------

func TestTenantSettingService_updateConfig_invalidConfigType(t *testing.T) {
	ts := newTenantSetting(1)
	repo := &mockTenantSettingRepo{
		findByTenantIDFn: func(_ int64) (*model.TenantSetting, error) { return ts, nil },
	}

	// Access the concrete type to call the unexported helper directly.
	svc := &tenantSettingService{tenantSettingRepo: repo}
	_, err := svc.updateConfig(context.Background(), 1, "bogus", map[string]any{"x": 1})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid config type")
}

func TestTenantSettingService_updateConfig_marshalError(t *testing.T) {
	ts := newTenantSetting(1)
	repo := &mockTenantSettingRepo{
		findByTenantIDFn: func(_ int64) (*model.TenantSetting, error) { return ts, nil },
	}

	svc := &tenantSettingService{tenantSettingRepo: repo}
	// A channel value cannot be marshalled to JSON.
	_, err := svc.updateConfig(context.Background(), 1, "rate_limit", map[string]any{"bad": make(chan int)})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid config payload")
}
