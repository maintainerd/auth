package migration

import "gorm.io/gorm"

func CreateTenantServicesTable(db *gorm.DB) error {
	// tenant_services was determined to be redundant with services.tenant_id.
	// Every service is tenant-scoped via services.tenant_id NOT NULL; there is
	// no cross-tenant service sharing. This migration intentionally creates nothing.
	return nil
}
