package app

import (
	"context"

	"github.com/maintainerd/maintainerd-auth/internal/setup/seeder"
	"gorm.io/gorm"
)

// tenantSeederAdapter satisfies tenant.TenantSeeder, letting the tenant service
// run the per-tenant baseline seed inside its create transaction. It lives in
// the app layer because the tenant package cannot import internal/setup/seeder
// directly (the seeder imports tenant, which would be an import cycle).
type tenantSeederAdapter struct{}

func (tenantSeederAdapter) SeedTenant(_ context.Context, tx *gorm.DB, tenantID int64) error {
	return seeder.SeedTenant(tx, tenantID)
}
