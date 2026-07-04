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
    social_links               JSONB DEFAULT '{}',
    preferred_contact_method   VARCHAR(20),
    marketing_email_consent    BOOLEAN DEFAULT FALSE,
    sms_notifications_consent  BOOLEAN DEFAULT FALSE,
    push_notifications_consent BOOLEAN DEFAULT FALSE,
    profile_visibility         VARCHAR(20) DEFAULT 'private',
    data_processing_consent    BOOLEAN DEFAULT FALSE,
    terms_accepted_at          TIMESTAMPTZ,
    privacy_policy_accepted_at TIMESTAMPTZ,
    emergency_contact_name     VARCHAR(200),
    emergency_contact_phone    VARCHAR(20),
    emergency_contact_email    VARCHAR(255),
    emergency_contact_relation VARCHAR(50),
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

    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'chk_user_settings_preferred_contact_method'
    ) THEN
        ALTER TABLE user_settings
            ADD CONSTRAINT chk_user_settings_preferred_contact_method
            CHECK (preferred_contact_method IN ('email', 'phone', 'sms'));
    END IF;

    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'chk_user_settings_visibility'
    ) THEN
        ALTER TABLE user_settings
            ADD CONSTRAINT chk_user_settings_visibility
            CHECK (profile_visibility IN ('public', 'private', 'friends'));
    END IF;
END$$;

-- ADD INDEXES
CREATE INDEX IF NOT EXISTS idx_user_settings_uuid ON user_settings (user_setting_uuid);
CREATE INDEX IF NOT EXISTS idx_user_settings_user_id ON user_settings (user_id);
CREATE INDEX IF NOT EXISTS idx_user_settings_locale ON user_settings (locale);
CREATE INDEX IF NOT EXISTS idx_user_settings_profile_visibility ON user_settings (profile_visibility);
CREATE INDEX IF NOT EXISTS idx_user_settings_created_at ON user_settings (created_at);

COMMENT ON COLUMN user_settings.social_links IS 'JSON object containing social media links and profiles';
COMMENT ON COLUMN user_settings.preferred_contact_method IS 'Preferred method of contact: email, phone, sms';
COMMENT ON COLUMN user_settings.profile_visibility IS 'Profile visibility setting: public, private, friends';
COMMENT ON COLUMN user_settings.timezone IS 'User timezone (e.g., America/New_York, Europe/London)';
COMMENT ON COLUMN user_settings.locale IS 'BCP-47 locale code (e.g., en-US, es-ES, fr-FR)';
`
	return db.Exec(sql).Error
}
