package migration

import "gorm.io/gorm"

func CreateAPIKeysTable(db *gorm.DB) error {
	// api_keys was determined to be redundant with M2M OAuth (client credentials flow).
	// The authorization model (scoped APIs and permissions per tenant) is identical.
	// Programmatic management access uses a system client with client_secret instead.
	return nil
}
