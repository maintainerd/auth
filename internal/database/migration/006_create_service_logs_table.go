package migration

import (
	"log"

	"gorm.io/gorm"
)

func CreateServiceLogsTable(db *gorm.DB) {
	sql := `
-- CREATE TABLE
CREATE TABLE IF NOT EXISTS service_logs (
    service_log_id			SERIAL PRIMARY KEY,
    service_log_uuid		UUID NOT NULL UNIQUE,
    service_id					INTEGER NOT NULL,
    level								VARCHAR(20) NOT NULL CHECK (level IN ('INFO', 'WARN', 'ERROR', 'DEBUG')),
    message							TEXT NOT NULL,
    metadata						JSONB,
    created_at					TIMESTAMPTZ DEFAULT now()
);

-- ADD CONSTRAINTS
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'fk_service_logs_service_id'
    ) THEN
        ALTER TABLE service_logs
            ADD CONSTRAINT fk_service_logs_service_id FOREIGN KEY (service_id)
            REFERENCES services(service_id) ON DELETE CASCADE;
    END IF;
END$$;

-- ADD INDEXES
CREATE INDEX IF NOT EXISTS idx_service_logs_uuid ON service_logs (service_log_uuid);
CREATE INDEX IF NOT EXISTS idx_service_logs_service_id ON service_logs (service_id);
CREATE INDEX IF NOT EXISTS idx_service_logs_level ON service_logs (level);
CREATE INDEX IF NOT EXISTS idx_service_logs_metadata ON service_logs (metadata);
CREATE INDEX IF NOT EXISTS idx_service_logs_created_at ON service_logs (created_at);
`

	if err := db.Exec(sql).Error; err != nil {
		log.Fatalf("❌ Failed to run migration 006_create_service_logs_table: %v", err)
	}

	log.Println("✅ Migration 006_create_service_logs_table executed")
}
