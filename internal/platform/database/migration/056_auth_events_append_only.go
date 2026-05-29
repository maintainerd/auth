package migration

import "gorm.io/gorm"

// CreateAuthEventsAppendOnlyRule adds a PostgreSQL RULE that silently discards
// any UPDATE on the auth_events table. This enforces append-only semantics at
// the database level — audit records can never be modified after creation.
//
// DELETE is intentionally not blocked so that the retention runner can prune
// old events via DeleteOlderThan. Inserts are unaffected.
func CreateAuthEventsAppendOnlyRule(db *gorm.DB) error {
	return db.Exec(`
		CREATE OR REPLACE RULE no_update_auth_events
		AS ON UPDATE TO auth_events DO INSTEAD NOTHING
	`).Error
}
