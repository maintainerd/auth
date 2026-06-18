package middleware

import (
	"context"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	resp "github.com/maintainerd/auth/internal/platform/response"
)

const (
	tenantMaintenanceConfigCacheTTL = 5 * time.Second
	defaultMaintenanceMessage       = "The system is currently undergoing maintenance. Please try again later."
)

// TenantMaintenanceReader reads a tenant's stored maintenance_config.
type TenantMaintenanceReader interface {
	GetMaintenanceConfig(ctx context.Context, tenantID int64) (map[string]any, error)
}

type MaintenanceConfig struct {
	Enabled        bool
	Message        string
	ScheduledStart *time.Time
	ScheduledEnd   *time.Time
}

// TenantMaintenanceMiddleware applies tenant_settings.maintenance_config after
// auth middleware has populated the request AuthContext.
func TenantMaintenanceMiddleware(reader TenantMaintenanceReader) func(http.Handler) http.Handler {
	configCache := newTenantMaintenanceConfigCache(tenantMaintenanceConfigCacheTTL)
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			auth := AuthFromRequest(r)
			if reader == nil || auth.Tenant == nil || isMaintenanceExcludedPath(r.URL.Path) {
				next.ServeHTTP(w, r)
				return
			}

			cfg := configCache.get(r.Context(), reader, auth.Tenant.TenantID)
			if cfg == nil || !cfg.isActive(time.Now()) {
				next.ServeHTTP(w, r)
				return
			}

			if cfg.ScheduledEnd != nil && time.Now().Before(*cfg.ScheduledEnd) {
				retryAfter := int(time.Until(*cfg.ScheduledEnd).Seconds())
				if retryAfter > 0 {
					w.Header().Set("Retry-After", strconv.Itoa(retryAfter))
				}
			}
			resp.ErrorWithCode(w, http.StatusServiceUnavailable, "maintenance_mode", cfg.Message)
		})
	}
}

func (cfg *MaintenanceConfig) isActive(now time.Time) bool {
	if cfg == nil || !cfg.Enabled {
		return false
	}
	if cfg.ScheduledStart != nil && now.Before(*cfg.ScheduledStart) {
		return false
	}
	if cfg.ScheduledEnd != nil && !now.Before(*cfg.ScheduledEnd) {
		return false
	}
	return true
}

func mapToMaintenanceConfig(raw map[string]any) *MaintenanceConfig {
	cfg := &MaintenanceConfig{
		Enabled: boolFromAny(raw["enabled"]),
		Message: stringFromAny(raw["message"]),
	}
	if strings.TrimSpace(cfg.Message) == "" {
		cfg.Message = defaultMaintenanceMessage
	}
	cfg.ScheduledStart = timePtrFromAny(raw["scheduled_start"])
	cfg.ScheduledEnd = timePtrFromAny(raw["scheduled_end"])
	return cfg
}

func stringFromAny(v any) string {
	s, _ := v.(string)
	return s
}

func timePtrFromAny(v any) *time.Time {
	switch value := v.(type) {
	case time.Time:
		return &value
	case string:
		if strings.TrimSpace(value) == "" {
			return nil
		}
		parsed, err := time.Parse(time.RFC3339, value)
		if err != nil {
			return nil
		}
		return &parsed
	default:
		return nil
	}
}

func isMaintenanceExcludedPath(path string) bool {
	switch strings.TrimRight(path, "/") {
	case "/health", "/healthz", "/ready", "/readyz", "/livez":
		return true
	default:
		return false
	}
}

type tenantMaintenanceConfigCache struct {
	ttl     time.Duration
	mu      sync.Mutex
	entries map[int64]tenantMaintenanceConfigCacheEntry
}

type tenantMaintenanceConfigCacheEntry struct {
	config    *MaintenanceConfig
	expiresAt time.Time
}

func newTenantMaintenanceConfigCache(ttl time.Duration) *tenantMaintenanceConfigCache {
	return &tenantMaintenanceConfigCache{
		ttl:     ttl,
		entries: map[int64]tenantMaintenanceConfigCacheEntry{},
	}
}

func (c *tenantMaintenanceConfigCache) get(ctx context.Context, reader TenantMaintenanceReader, tenantID int64) *MaintenanceConfig {
	now := time.Now()
	c.mu.Lock()
	entry, ok := c.entries[tenantID]
	if ok && now.Before(entry.expiresAt) {
		c.mu.Unlock()
		return entry.config
	}
	c.mu.Unlock()

	raw, err := reader.GetMaintenanceConfig(ctx, tenantID)
	if err != nil {
		return nil
	}
	config := mapToMaintenanceConfig(raw)

	c.mu.Lock()
	c.entries[tenantID] = tenantMaintenanceConfigCacheEntry{
		config:    config,
		expiresAt: now.Add(c.ttl),
	}
	c.mu.Unlock()

	return config
}
