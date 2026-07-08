package seeder

import (
	"log/slog"

	"github.com/maintainerd/maintainerd-auth/internal/tenant"
	"gorm.io/gorm"
)

// RunAll performs first-run bootstrap seeding. It seeds the globally-scoped
// records once (the "auth" service and the integration event-type catalog),
// then delegates the per-tenant baseline to SeedTenant for the system tenant.
//
// The per-tenant baseline (roles, permissions, clients, idp, branding, etc.)
// lives in SeedTenant so that admin-side tenant creation reuses the exact same
// seeding path.
func RunAll(db *gorm.DB, appVersion string) error {
	slog.Info("Running default seeders")

	// Locate the system tenant (created just before seeding during bootstrap).
	var sysTenant tenant.Tenant
	if err := db.Where("is_system = ?", true).First(&sysTenant).Error; err != nil {
		slog.Error("Failed to find system tenant", "error", err)
		return err
	}
	slog.Info("Found system tenant", "tenant_id", sysTenant.TenantID)

	// Per-tenant baseline for the system tenant (same path new tenants use).
	// Everything — including the tenant's own service and event-type catalog —
	// is seeded per tenant by SeedTenant.
	if err := SeedTenant(db, sysTenant.TenantID, appVersion); err != nil {
		slog.Error("Failed to seed system tenant baseline", "error", err)
		return err
	}

	slog.Info("Default seeding process completed")
	return nil
}
