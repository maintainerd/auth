package migration

import "gorm.io/gorm"

func CreateAPIKeyPermissionsTable(db *gorm.DB) error {
	// api_key_permissions removed — api_keys is a no-op; this child table is not created.
	return nil
}
