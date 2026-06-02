package tenant

import (
	"context"
	"log/slog"
	"time"

	"gorm.io/gorm"
)

const (
	DefaultTenantRetentionPeriod   = 30 * 24 * time.Hour
	DefaultTenantRetentionInterval = 24 * time.Hour
)

func StartRetentionRunner(ctx context.Context, db *gorm.DB, retentionPeriod, interval time.Duration) {
	if db == nil {
		return
	}
	if retentionPeriod <= 0 {
		retentionPeriod = DefaultTenantRetentionPeriod
	}
	if interval <= 0 {
		interval = DefaultTenantRetentionInterval
	}

	runTenantRetention(ctx, db, retentionPeriod)
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			runTenantRetention(ctx, db, retentionPeriod)
		}
	}
}

func runTenantRetention(ctx context.Context, db *gorm.DB, retentionPeriod time.Duration) {
	cutoff := time.Now().Add(-retentionPeriod)
	result := db.WithContext(ctx).
		Unscoped().
		Where("deleted_at IS NOT NULL AND deleted_at < ? AND is_system = false", cutoff).
		Delete(&Tenant{})
	if result.Error != nil {
		slog.Error("tenant retention purge failed", "error", result.Error)
		return
	}
	if result.RowsAffected > 0 {
		slog.Info("tenant retention purged deleted tenants", "count", result.RowsAffected)
	}
}
