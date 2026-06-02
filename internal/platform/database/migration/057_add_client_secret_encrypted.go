package migration

import "gorm.io/gorm"

// AddClientSecretEncrypted adds encrypted client-secret storage for
// client_secret_jwt HMAC verification while retaining bcrypt hashes for
// client_secret_basic/client_secret_post checks.
func AddClientSecretEncrypted(db *gorm.DB) error {
	return db.Exec(`
ALTER TABLE clients
    ADD COLUMN IF NOT EXISTS secret_encrypted TEXT,
    ADD COLUMN IF NOT EXISTS previous_secret_encrypted TEXT;
`).Error
}
