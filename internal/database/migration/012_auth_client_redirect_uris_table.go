package migration

import (
	"log"

	"gorm.io/gorm"
)

func CreateAuthClientRedirectUrisTable(db *gorm.DB) {
	sql := `
-- CREATE TABLE
CREATE TABLE IF NOT EXISTS auth_client_redirect_uris (
    auth_client_redirect_uri_id  SERIAL PRIMARY KEY,
    auth_client_redirect_uri_uuid UUID NOT NULL UNIQUE,
    auth_client_id               INTEGER NOT NULL,
    redirect_uri                 TEXT NOT NULL,
    created_at                   TIMESTAMPTZ DEFAULT now(),
    updated_at                   TIMESTAMPTZ DEFAULT now()
);

-- ADD CONSTRAINTS (safe)
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'fk_auth_client_redirect_uris_auth_client_id'
    ) THEN
        ALTER TABLE auth_client_redirect_uris
            ADD CONSTRAINT fk_auth_client_redirect_uris_auth_client_id FOREIGN KEY (auth_client_id)
            REFERENCES auth_clients(auth_client_id) ON DELETE CASCADE;
    END IF;
END$$;

-- ADD INDEXES
CREATE INDEX IF NOT EXISTS idx_auth_client_redirect_uris_uuid 
    ON auth_client_redirect_uris (auth_client_redirect_uri_uuid);
CREATE INDEX IF NOT EXISTS idx_auth_client_redirect_uris_auth_client_id 
    ON auth_client_redirect_uris (auth_client_id);
CREATE INDEX IF NOT EXISTS idx_auth_client_redirect_uris_redirect_uri 
    ON auth_client_redirect_uris (redirect_uri);
`
	if err := db.Exec(sql).Error; err != nil {
		log.Fatalf("❌ Failed to run migration 012_create_auth_client_redirect_uris_table: %v", err)
	}

	log.Println("✅ Migration 012_create_auth_client_redirect_uris_table executed")
}
