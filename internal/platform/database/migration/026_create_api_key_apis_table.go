package migration

import "gorm.io/gorm"

func CreateAPIKeyAPITable(db *gorm.DB) error {
	// api_key_apis removed — api_keys is a no-op; this child table is not created.
	return nil
}
