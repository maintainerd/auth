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
	runTenantRetentionLoop(ctx, db, retentionPeriod, ticker.C)
}

func runTenantRetentionLoop(ctx context.Context, db *gorm.DB, retentionPeriod time.Duration, ticks <-chan time.Time) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticks:
			runTenantRetention(ctx, db, retentionPeriod)
		}
	}
}

func runTenantRetention(ctx context.Context, db *gorm.DB, retentionPeriod time.Duration) {
	cutoff := time.Now().Add(-retentionPeriod)

	// The physical purge hard-deletes the tenant row, which cascades ON DELETE to
	// every tenant-scoped table — including the append-only audit tables
	// (auth_events, management_audit_log) whose immutability triggers block
	// deletes unless a sanctioned-lifecycle GUC is set. Run the purge in one
	// transaction that sets both GUCs (transaction-local) so the cascade can
	// complete; without them the cascade raises and NO tenant is ever purged
	// (soft-deleted tenants + their PII would accumulate forever — a retention /
	// GDPR-erasure failure).
	var purged int64
	err := db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec("SELECT set_config('maintainerd.allow_auth_event_delete', 'tenant_delete', true)").Error; err != nil {
			return err
		}
		if err := tx.Exec("SELECT set_config('maintainerd.allow_management_audit_log_delete', 'tenant_delete', true)").Error; err != nil {
			return err
		}
		result := tx.Unscoped().
			Where("deleted_at IS NOT NULL AND deleted_at < ? AND is_system = false", cutoff).
			Delete(&Tenant{})
		if result.Error != nil {
			return result.Error
		}
		purged = result.RowsAffected
		return nil
	})
	if err != nil {
		slog.Error("tenant retention purge failed", "error", err)
		return
	}
	if purged > 0 {
		slog.Info("tenant retention purged deleted tenants", "count", purged)
	}
}
