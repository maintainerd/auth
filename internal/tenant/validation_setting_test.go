package tenant

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTenantSettingUpdateConfigRequestDTO_Validate(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		d := TenantSettingUpdateConfigRequestDTO{"max_rps": 100}
		assert.NoError(t, d.Validate())
	})

	t.Run("empty map rejected", func(t *testing.T) {
		d := TenantSettingUpdateConfigRequestDTO{}
		require.Error(t, d.Validate())
	})

	t.Run("nil map rejected", func(t *testing.T) {
		var d TenantSettingUpdateConfigRequestDTO
		require.Error(t, d.Validate())
	})
}

func TestTenantSettingUpdateConfigRequestDTO_ValidateRateLimitConfig(t *testing.T) {
	t.Run("valid full config", func(t *testing.T) {
		dto := TenantSettingUpdateConfigRequestDTO{
			"enabled":                 true,
			"requests_per_window":     float64(100),
			"window_duration_seconds": float64(60),
			"per_ip":                  true,
			"exempt_ips":              []any{"10.0.0.1"},
			"endpoint_overrides":      map[string]any{"/login": float64(10)},
		}
		assert.NoError(t, dto.ValidateRateLimitConfig())
	})

	t.Run("invalid requests_per_window zero", func(t *testing.T) {
		dto := TenantSettingUpdateConfigRequestDTO{"requests_per_window": float64(0)}
		require.Error(t, dto.ValidateRateLimitConfig())
	})

	t.Run("invalid window_duration_seconds too large", func(t *testing.T) {
		dto := TenantSettingUpdateConfigRequestDTO{"window_duration_seconds": float64(9999)}
		require.Error(t, dto.ValidateRateLimitConfig())
	})

	t.Run("unknown field rejected", func(t *testing.T) {
		dto := TenantSettingUpdateConfigRequestDTO{"max_rps": float64(100)}
		require.Error(t, dto.ValidateRateLimitConfig())
	})

	t.Run("endpoint_overrides must have positive values", func(t *testing.T) {
		dto := TenantSettingUpdateConfigRequestDTO{"endpoint_overrides": map[string]any{"/login": float64(0)}}
		require.Error(t, dto.ValidateRateLimitConfig())
	})

	t.Run("exempt_ips must be strings", func(t *testing.T) {
		dto := TenantSettingUpdateConfigRequestDTO{"exempt_ips": []any{123}}
		require.Error(t, dto.ValidateRateLimitConfig())
	})
}

func TestTenantSettingUpdateConfigRequestDTO_ValidateAuditConfig(t *testing.T) {
	t.Run("valid full config", func(t *testing.T) {
		dto := TenantSettingUpdateConfigRequestDTO{
			"enabled":        true,
			"retention_days": float64(365),
			"pii_masking":    true,
			"log_level":      "warn",
			"event_types":    []any{"authn_login_success"},
		}
		assert.NoError(t, dto.ValidateAuditConfig())
	})

	t.Run("invalid retention range", func(t *testing.T) {
		dto := TenantSettingUpdateConfigRequestDTO{"retention_days": float64(0)}
		require.Error(t, dto.ValidateAuditConfig())
	})

	t.Run("unknown field rejected", func(t *testing.T) {
		dto := TenantSettingUpdateConfigRequestDTO{"unexpected": true}
		require.Error(t, dto.ValidateAuditConfig())
	})

	t.Run("event types must be strings", func(t *testing.T) {
		dto := TenantSettingUpdateConfigRequestDTO{"event_types": []any{"authn_login_success", 123}}
		require.Error(t, dto.ValidateAuditConfig())
	})
}

func TestTenantSettingUpdateConfigRequestDTO_ValidateMaintenanceConfig(t *testing.T) {
	t.Run("valid full config", func(t *testing.T) {
		dto := TenantSettingUpdateConfigRequestDTO{
			"enabled":         true,
			"message":         "Maintenance in progress",
			"scheduled_start": "2026-01-01T00:00:00Z",
			"scheduled_end":   "2026-01-01T01:00:00Z",
		}
		assert.NoError(t, dto.ValidateMaintenanceConfig())
	})

	t.Run("enabled must be boolean", func(t *testing.T) {
		dto := TenantSettingUpdateConfigRequestDTO{"enabled": "true"}
		require.Error(t, dto.ValidateMaintenanceConfig())
	})

	t.Run("message cannot be empty", func(t *testing.T) {
		dto := TenantSettingUpdateConfigRequestDTO{"message": "   "}
		require.Error(t, dto.ValidateMaintenanceConfig())
	})

	t.Run("scheduled start must be RFC3339 or null", func(t *testing.T) {
		dto := TenantSettingUpdateConfigRequestDTO{"scheduled_start": "tomorrow"}
		require.Error(t, dto.ValidateMaintenanceConfig())
	})

	t.Run("scheduled end must be after start", func(t *testing.T) {
		dto := TenantSettingUpdateConfigRequestDTO{
			"scheduled_start": "2026-01-01T01:00:00Z",
			"scheduled_end":   "2026-01-01T00:00:00Z",
		}
		require.Error(t, dto.ValidateMaintenanceConfig())
	})

	t.Run("removed bypass_ips field rejected", func(t *testing.T) {
		dto := TenantSettingUpdateConfigRequestDTO{"bypass_ips": []any{"127.0.0.1"}}
		require.Error(t, dto.ValidateMaintenanceConfig())
	})

	t.Run("removed admin_bypass_roles field rejected", func(t *testing.T) {
		dto := TenantSettingUpdateConfigRequestDTO{"admin_bypass_roles": []any{"super-admin"}}
		require.Error(t, dto.ValidateMaintenanceConfig())
	})
}
