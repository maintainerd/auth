package seeder

import (
	model "github.com/maintainerd/maintainerd-auth/internal/tenant"
	"gorm.io/gorm"
)

func SeedTenantService(db *gorm.DB, tenantID, serviceID int64) (model.TenantServiceLink, error) {
	// DEPRECATED: tenant_services table has been removed.
	// services.tenant_id is the authoritative relationship.
	return model.TenantServiceLink{}, nil
}
