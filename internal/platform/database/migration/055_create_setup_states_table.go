package migration

import "gorm.io/gorm"

// CreateSetupStatesTable stores process-level bootstrap locks independently
// from tenant and service domain records.
func CreateSetupStatesTable(db *gorm.DB) error {
	return db.Exec(`
CREATE TABLE IF NOT EXISTS setup_states (
    setup_state_id BIGSERIAL PRIMARY KEY,
    key            TEXT NOT NULL UNIQUE,
    is_complete    BOOLEAN NOT NULL DEFAULT FALSE,
    completed_at   TIMESTAMPTZ,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_setup_states_key_complete ON setup_states(key, is_complete);
`).Error
}
