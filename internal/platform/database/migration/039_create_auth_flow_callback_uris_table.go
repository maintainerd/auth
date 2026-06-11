package migration

import (
	"gorm.io/gorm"
)

func CreateAuthFlowCallbackURITable(db *gorm.DB) error {
	sql := `
-- CREATE TABLE
CREATE TABLE IF NOT EXISTS auth_flow_callback_uris (
    auth_flow_callback_uri_id   BIGSERIAL PRIMARY KEY,
    auth_flow_callback_uri_uuid UUID NOT NULL UNIQUE,
    auth_flow_id                BIGINT NOT NULL,
    client_uri_id               BIGINT NOT NULL,
    created_at                  TIMESTAMPTZ DEFAULT now()
);

-- ADD CONSTRAINTS
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'fk_auth_flow_callback_uris_auth_flow_id'
    ) THEN
        ALTER TABLE auth_flow_callback_uris
            ADD CONSTRAINT fk_auth_flow_callback_uris_auth_flow_id FOREIGN KEY (auth_flow_id)
            REFERENCES auth_flows(auth_flow_id) ON DELETE CASCADE;
    END IF;

    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'fk_auth_flow_callback_uris_client_uri_id'
    ) THEN
        ALTER TABLE auth_flow_callback_uris
            ADD CONSTRAINT fk_auth_flow_callback_uris_client_uri_id FOREIGN KEY (client_uri_id)
            REFERENCES client_uris(client_uri_id) ON DELETE CASCADE;
    END IF;

    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'uq_auth_flow_callback_uris_flow_uri'
    ) THEN
        ALTER TABLE auth_flow_callback_uris
            ADD CONSTRAINT uq_auth_flow_callback_uris_flow_uri UNIQUE (auth_flow_id, client_uri_id);
    END IF;
END$$;

-- ADD INDEXES
CREATE INDEX IF NOT EXISTS idx_auth_flow_callback_uris_uuid ON auth_flow_callback_uris (auth_flow_callback_uri_uuid);
CREATE INDEX IF NOT EXISTS idx_auth_flow_callback_uris_auth_flow_id ON auth_flow_callback_uris (auth_flow_id);
CREATE INDEX IF NOT EXISTS idx_auth_flow_callback_uris_client_uri_id ON auth_flow_callback_uris (client_uri_id);
`
	return db.Exec(sql).Error
}
