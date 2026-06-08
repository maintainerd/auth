package migration

import (
	"gorm.io/gorm"
)

// CreateEventTypesTable creates the event_types canonical catalog table.
func CreateEventTypesTable(db *gorm.DB) error {
	sql := `
-- CREATE TABLE
CREATE TABLE IF NOT EXISTS event_types (
    event_type_id           BIGSERIAL PRIMARY KEY,
    event_type_uuid         UUID NOT NULL UNIQUE,
    key                     VARCHAR(100) NOT NULL UNIQUE,
    category                VARCHAR(50) NOT NULL,
    description             TEXT,
    version                 INTEGER NOT NULL DEFAULT 1,
    is_active               BOOLEAN NOT NULL DEFAULT true,
    created_at              TIMESTAMPTZ DEFAULT now(),
    updated_at              TIMESTAMPTZ DEFAULT now()
);

-- CREATE INDEXES
CREATE INDEX IF NOT EXISTS idx_event_types_key ON event_types (key);
CREATE INDEX IF NOT EXISTS idx_event_types_category ON event_types (category);
CREATE INDEX IF NOT EXISTS idx_event_types_is_active ON event_types (is_active);
`

	return db.Exec(sql).Error
}
