package migration

import (
	"gorm.io/gorm"
)

func CreateClientAPIsTable(db *gorm.DB) error {
	sql := `
-- CREATE TABLE
CREATE TABLE IF NOT EXISTS client_apis (
    client_api_id   		SERIAL PRIMARY KEY,
    client_api_uuid		UUID NOT NULL UNIQUE,
    client_id              INTEGER NOT NULL,
    api_id                      INTEGER NOT NULL,
    created_at                  TIMESTAMPTZ DEFAULT now()
);

-- ADD CONSTRAINTS
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'fk_client_apis_client_id'
    ) THEN
        ALTER TABLE client_apis
            ADD CONSTRAINT fk_client_apis_client_id FOREIGN KEY (client_id)
            REFERENCES clients(client_id) ON DELETE CASCADE;
    END IF;

    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'fk_client_apis_api_id'
    ) THEN
        ALTER TABLE client_apis
            ADD CONSTRAINT fk_client_apis_api_id FOREIGN KEY (api_id)
            REFERENCES apis(api_id) ON DELETE CASCADE;
    END IF;

    -- Add unique constraint to prevent duplicate client + api combinations
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'uq_client_apis_client_api'
    ) THEN
        ALTER TABLE client_apis
            ADD CONSTRAINT uq_client_apis_client_api UNIQUE (client_id, api_id);
    END IF;
END$$;

-- ADD INDEXES
CREATE INDEX IF NOT EXISTS idx_client_apis_uuid ON client_apis (client_api_uuid);
CREATE INDEX IF NOT EXISTS idx_client_apis_client_id ON client_apis (client_id);
CREATE INDEX IF NOT EXISTS idx_client_apis_api_id ON client_apis (api_id);
`

	return db.Exec(sql).Error
}
