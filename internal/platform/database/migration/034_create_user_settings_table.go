package migration

import (
	"gorm.io/gorm"
)

// CreateUserSettingsTable creates the user_settings table (1:1 with user).
// Cascade-deletes with the parent user, so no soft delete is needed.
// preferred_language was removed — use locale (BCP-47) as the single source of truth.
func CreateUserSettingsTable(db *gorm.DB) error {
	sql := `
-- CREATE TABLE
CREATE TABLE IF NOT EXISTS user_settings (
    user_setting_id            BIGSERIAL PRIMARY KEY,
    user_setting_uuid          UUID NOT NULL UNIQUE,
    user_id                    BIGINT NOT NULL UNIQUE,
    timezone                   VARCHAR(50),
    locale                     VARCHAR(10),
    created_at                 TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at                 TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- ADD CONSTRAINTS
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'fk_user_settings_user_id'
    ) THEN
        ALTER TABLE user_settings
            ADD CONSTRAINT fk_user_settings_user_id FOREIGN KEY (user_id)
            REFERENCES users(user_id) ON DELETE CASCADE;
    END IF;

END$$;

-- ADD INDEXES
CREATE INDEX IF NOT EXISTS idx_user_settings_uuid ON user_settings (user_setting_uuid);
CREATE INDEX IF NOT EXISTS idx_user_settings_user_id ON user_settings (user_id);
CREATE INDEX IF NOT EXISTS idx_user_settings_locale ON user_settings (locale);
CREATE INDEX IF NOT EXISTS idx_user_settings_created_at ON user_settings (created_at);

COMMENT ON COLUMN user_settings.timezone IS 'User timezone (e.g., America/New_York, Europe/London)';
COMMENT ON COLUMN user_settings.locale IS 'BCP-47 locale code (e.g., en-US, es-ES, fr-FR)';
`
	return db.Exec(sql).Error
}
