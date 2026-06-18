package authevent

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"
)

const (
	auditConfigCacheTTL     = 5 * time.Second
	defaultAuditRetention   = 90
	legacyAuditRetention    = 365
	maxAuthEventExportLimit = 10000
)

// AuditConfigReader supplies tenant-scoped audit_config values from
// tenant_settings without coupling authevent to the tenant package.
type AuditConfigReader interface {
	GetAuditConfig(ctx context.Context, tenantID int64) (map[string]any, error)
}

type auditConfig struct {
	Enabled       bool
	RetentionDays int
	PIIMasking    bool
	LogLevel      string
	EventTypes    map[string]struct{}
}

type auditConfigCacheEntry struct {
	config    auditConfig
	expiresAt time.Time
}

func legacyAuditConfig() auditConfig {
	return auditConfig{
		Enabled:       true,
		RetentionDays: legacyAuditRetention,
		PIIMasking:    true,
		LogLevel:      "info",
		EventTypes:    map[string]struct{}{},
	}
}

func defaultTenantAuditConfig() auditConfig {
	return auditConfig{
		Enabled:       true,
		RetentionDays: defaultAuditRetention,
		PIIMasking:    true,
		LogLevel:      "info",
		EventTypes:    map[string]struct{}{},
	}
}

func parseAuditConfig(raw map[string]any) auditConfig {
	cfg := defaultTenantAuditConfig()
	if raw == nil {
		return cfg
	}
	cfg.Enabled = boolValue(raw["enabled"], cfg.Enabled)
	cfg.RetentionDays = intValue(raw["retention_days"], cfg.RetentionDays)
	cfg.PIIMasking = boolValue(raw["pii_masking"], cfg.PIIMasking)
	cfg.LogLevel = strings.ToLower(stringValue(raw["log_level"], cfg.LogLevel))
	cfg.EventTypes = eventTypeSet(raw["event_types"])
	return cfg
}

func boolValue(v any, fallback bool) bool {
	switch value := v.(type) {
	case bool:
		return value
	case string:
		parsed, err := strconv.ParseBool(value)
		if err == nil {
			return parsed
		}
	}
	return fallback
}

func intValue(v any, fallback int) int {
	switch value := v.(type) {
	case int:
		return value
	case int64:
		return int(value)
	case float64:
		return int(value)
	case jsonNumber:
		parsed, err := strconv.Atoi(value.String())
		if err == nil {
			return parsed
		}
	case string:
		parsed, err := strconv.Atoi(value)
		if err == nil {
			return parsed
		}
	}
	return fallback
}

type jsonNumber interface {
	String() string
}

func stringValue(v any, fallback string) string {
	if value, ok := v.(string); ok && value != "" {
		return value
	}
	return fallback
}

func eventTypeSet(v any) map[string]struct{} {
	result := map[string]struct{}{}
	switch values := v.(type) {
	case []string:
		for _, value := range values {
			if value != "" {
				result[value] = struct{}{}
			}
		}
	case []any:
		for _, item := range values {
			if value, ok := item.(string); ok && value != "" {
				result[value] = struct{}{}
			}
		}
	}
	return result
}

func (cfg auditConfig) allowsEvent(eventType string) bool {
	if len(cfg.EventTypes) == 0 {
		return true
	}
	_, ok := cfg.EventTypes[eventType]
	return ok
}

func (cfg auditConfig) allowsSeverity(severity string) bool {
	return severityRank(severity) >= severityRank(cfg.LogLevel)
}

func (cfg auditConfig) masksPII() bool {
	return cfg.PIIMasking
}

func severityRank(severity string) int {
	switch strings.ToLower(severity) {
	case "debug":
		return 0
	case "info":
		return 1
	case "warn", "warning":
		return 2
	case "critical", "error":
		return 3
	default:
		return 1
	}
}

func normalizeExportFormat(format string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(format)) {
	case "", "json":
		return "json", nil
	case "csv":
		return "csv", nil
	default:
		return "", fmt.Errorf("unsupported auth event export format: %s", format)
	}
}
