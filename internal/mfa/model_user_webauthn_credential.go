package mfa

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// UserWebAuthnCredential stores a registered WebAuthn / FIDO2 credential.
// A user may have multiple credentials (e.g. phone passkey + hardware key).
type UserWebAuthnCredential struct {
	CredentialID     int64      `gorm:"column:credential_id;primaryKey"`
	CredentialUUID   uuid.UUID  `gorm:"column:credential_uuid"`
	UserID           int64      `gorm:"column:user_id;not null"`
	CredentialKeyID  string     `gorm:"column:credential_key_id;not null;unique"`
	PublicKey        []byte     `gorm:"column:public_key;not null"`
	AAGUID           *uuid.UUID `gorm:"column:aaguid;type:uuid"`
	SignCount        int64      `gorm:"column:sign_count;default:0"`
	Transport        string     `gorm:"column:transport"`
	IsBackupEligible bool       `gorm:"column:is_backup_eligible;default:false"`
	IsBackupState    bool       `gorm:"column:is_backup_state;default:false"`
	Name             string     `gorm:"column:name;default:'Security Key'"`
	LastUsedAt       *time.Time `gorm:"column:last_used_at"`
	CreatedAt        time.Time  `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt        time.Time  `gorm:"column:updated_at;autoUpdateTime"`
}

func (UserWebAuthnCredential) TableName() string { return "user_webauthn_credentials" }

func (c *UserWebAuthnCredential) BeforeCreate(tx *gorm.DB) error {
	if c.CredentialUUID == uuid.Nil {
		c.CredentialUUID = uuid.New()
	}
	return nil
}
