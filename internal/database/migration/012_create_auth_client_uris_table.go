package migration

import (
	"log"

	"gorm.io/gorm"
)

func CreateAuthClientUrisTable(db *gorm.DB) {
	sql := `
-- CREATE TABLE
CREATE TABLE IF NOT EXISTS auth_client_uris (
    auth_client_uri_id   SERIAL PRIMARY KEY,
    auth_client_uri_uuid UUID NOT NULL UNIQUE,
    auth_client_id       INTEGER NOT NULL,
    uri                  TEXT NOT NULL,
    type                 VARCHAR(20) NOT NULL DEFAULT 'redirect-uri',
    created_at           TIMESTAMPTZ DEFAULT now(),
    updated_at           TIMESTAMPTZ DEFAULT now()
);

-- ADD CONSTRAINTS (safe)
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'fk_auth_client_uris_auth_client_id'
    ) THEN
        ALTER TABLE auth_client_uris
            ADD CONSTRAINT fk_auth_client_uris_auth_client_id FOREIGN KEY (auth_client_id)
            REFERENCES auth_clients(auth_client_id) ON DELETE CASCADE;
    END IF;

    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'chk_auth_client_uris_type'
    ) THEN
        ALTER TABLE auth_client_uris
            ADD CONSTRAINT chk_auth_client_uris_type CHECK (type IN ('redirect-uri', 'origin-uri', 'logout-uri', 'login-uri', 'cors-origin-uri'));
    END IF;
END$$;

-- ADD INDEXES
CREATE INDEX IF NOT EXISTS idx_auth_client_uris_uuid 
    ON auth_client_uris (auth_client_uri_uuid);
CREATE INDEX IF NOT EXISTS idx_auth_client_uris_auth_client_id 
    ON auth_client_uris (auth_client_id);
CREATE INDEX IF NOT EXISTS idx_auth_client_uris_uri 
    ON auth_client_uris (uri);
CREATE INDEX IF NOT EXISTS idx_auth_client_uris_type 
    ON auth_client_uris (type);
CREATE INDEX IF NOT EXISTS idx_auth_client_uris_auth_client_id_type 
    ON auth_client_uris (auth_client_id, type);
`
	if err := db.Exec(sql).Error; err != nil {
		log.Fatalf("❌ Failed to run migration 012_create_auth_client_uris_table: %v", err)
	}

	log.Println("✅ Migration 012_create_auth_client_uris_table executed")
}
