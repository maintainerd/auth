package migration

import (
	"log"

	"gorm.io/gorm"
)

func CreateUserSettingsTable(db *gorm.DB) {
	sql := `
-- CREATE TABLE
CREATE TABLE IF NOT EXISTS user_settings (
    user_setting_id         SERIAL PRIMARY KEY,
    user_setting_uuid       UUID NOT NULL UNIQUE,
    user_id                 INTEGER NOT NULL UNIQUE,
    timezone                VARCHAR(50),
    preferred_language      VARCHAR(10),
    locale                  VARCHAR(10),
    social_links            JSONB DEFAULT '{}',
    preferred_contact_method VARCHAR(20),
    marketing_email_consent BOOLEAN DEFAULT FALSE,
    sms_notifications_consent BOOLEAN DEFAULT FALSE,
    push_notifications_consent BOOLEAN DEFAULT FALSE,
    profile_visibility      VARCHAR(20) DEFAULT 'private',
    data_processing_consent BOOLEAN DEFAULT FALSE,
    terms_accepted_at       TIMESTAMPTZ,
    privacy_policy_accepted_at TIMESTAMPTZ,
    emergency_contact_name  VARCHAR(200),
    emergency_contact_phone VARCHAR(20),
    emergency_contact_email VARCHAR(255),
    emergency_contact_relation VARCHAR(50),
    created_at              TIMESTAMPTZ DEFAULT now(),
    updated_at              TIMESTAMPTZ DEFAULT now()
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
CREATE INDEX IF NOT EXISTS idx_user_settings_preferred_language ON user_settings (preferred_language);
CREATE INDEX IF NOT EXISTS idx_user_settings_profile_visibility ON user_settings (profile_visibility);
CREATE INDEX IF NOT EXISTS idx_user_settings_created_at ON user_settings (created_at);

-- ADD CHECK CONSTRAINTS FOR DATA INTEGRITY
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'chk_user_settings_preferred_contact_method'
    ) THEN
        ALTER TABLE user_settings
            ADD CONSTRAINT chk_user_settings_preferred_contact_method
            CHECK (preferred_contact_method IN ('email', 'phone', 'sms'));
    END IF;
END$$;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'chk_user_settings_visibility'
    ) THEN
        ALTER TABLE user_settings
            ADD CONSTRAINT chk_user_settings_visibility
            CHECK (profile_visibility IN ('public', 'private', 'friends'));
    END IF;
END$$;

-- ADD COMMENTS FOR DOCUMENTATION
COMMENT ON COLUMN user_settings.social_links IS 'JSON object containing social media links and profiles';
COMMENT ON COLUMN user_settings.preferred_contact_method IS 'Preferred method of contact: email, phone, sms';
COMMENT ON COLUMN user_settings.profile_visibility IS 'Profile visibility setting: public, private, friends';
COMMENT ON COLUMN user_settings.timezone IS 'User timezone (e.g., America/New_York, Europe/London)';
COMMENT ON COLUMN user_settings.preferred_language IS 'ISO 639-1 language code (e.g., en, es, fr)';
COMMENT ON COLUMN user_settings.locale IS 'Locale code (e.g., en_US, es_ES, fr_FR)';
`
	if err := db.Exec(sql).Error; err != nil {
		log.Fatalf("❌ Failed to run migration 021_create_user_settings_table: %v", err)
	}

	log.Println("✅ Migration 021_create_user_settings_table executed")
}
