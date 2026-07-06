package migration

import "gorm.io/gorm"

func CreateApiPermissionTable(db *gorm.DB) error {
	// api_permissions was determined to be redundant with permissions.api_id (the FK
	// on permissions already establishes the 1:M relationship between apis and permissions).
	// The M:N junction creates a potential data model contradiction and is not used.
	// This migration intentionally creates nothing; the table is not part of the schema.
	return nil
}
