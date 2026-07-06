package migration

import "gorm.io/gorm"

// CreateUserPasswordHistoryTable stores previous password hashes per user so
// services can enforce PasswordPolicy.HistoryCount (no re-use of last N passwords).
func CreateUserPasswordHistoryTable(db *gorm.DB) error {
	return db.Exec(`
CREATE TABLE IF NOT EXISTS user_password_history (
    history_id    BIGSERIAL   PRIMARY KEY,
    history_uuid  UUID        NOT NULL UNIQUE DEFAULT gen_random_uuid(),
    user_id       BIGINT      NOT NULL REFERENCES users(user_id) ON DELETE CASCADE,
    password_hash TEXT        NOT NULL,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_uph_user_id_created ON user_password_history(user_id, created_at DESC);

-- Append-only guard: prevents accidental mutation of password history records.
CREATE OR REPLACE FUNCTION deny_password_history_update()
RETURNS TRIGGER LANGUAGE plpgsql AS $$
BEGIN
    RAISE EXCEPTION 'user_password_history is append-only; updates are not permitted';
END;
$$;

DROP TRIGGER IF EXISTS trg_deny_uph_update ON user_password_history;
CREATE TRIGGER trg_deny_uph_update
    BEFORE UPDATE ON user_password_history
    FOR EACH ROW EXECUTE FUNCTION deny_password_history_update();
`).Error
}
