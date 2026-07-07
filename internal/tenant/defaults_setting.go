package tenant

import (
	"encoding/json"

	"gorm.io/datatypes"
)

var defaultTenantSettingConfigs = map[string]map[string]any{
	"rate_limit": {
		"enabled":                 false,
		"requests_per_window":     100,
		"window_duration_seconds": 60,
		"per_ip":                  true,
		"exempt_ips":              []string{},
		"endpoint_overrides":      map[string]any{},
	},
	"audit": {
		"enabled":        true,
		"retention_days": 90,
		"pii_masking":    true,
		"log_level":      "info",
		"event_types":    []string{},
	},
	"maintenance": {
		"enabled":         false,
		"message":         "The system is currently undergoing maintenance. Please try again later.",
		"scheduled_start": nil,
		"scheduled_end":   nil,
	},
}

func DefaultTenantSettingConfig(configType string) (map[string]any, bool) {
	cfg, ok := defaultTenantSettingConfigs[configType]
	if !ok {
		return nil, false
	}
	out := make(map[string]any, len(cfg))
	for k, v := range cfg {
		switch values := v.(type) {
		case []string:
			out[k] = append([]string(nil), values...)
		case map[string]any:
			cp := make(map[string]any, len(values))
			for mk, mv := range values {
				cp[mk] = mv
			}
			out[k] = cp
		default:
			out[k] = v
		}
	}
	return out, true
}

func MustDefaultTenantSettingConfig(configType string) map[string]any {
	cfg, ok := DefaultTenantSettingConfig(configType)
	if !ok {
		panic("unknown tenant setting config type: " + configType)
	}
	return cfg
}

func DefaultTenantSettingJSON(configType string) datatypes.JSON {
	cfg := MustDefaultTenantSettingConfig(configType)
	b, err := json.Marshal(cfg)
	if err != nil {
		panic(err)
	}
	return datatypes.JSON(b)
}

func NewDefaultTenantSetting(tenantID int64) TenantSetting {
	return TenantSetting{
		TenantID:          tenantID,
		RateLimitConfig:   DefaultTenantSettingJSON("rate_limit"),
		AuditConfig:       DefaultTenantSettingJSON("audit"),
		MaintenanceConfig: DefaultTenantSettingJSON("maintenance"),
	}
}
