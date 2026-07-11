package mfa

import (
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"gorm.io/gorm"
)

// UserMFAWebAuthnCredential stores a registered WebAuthn / FIDO2 credential.
// A user may have multiple credentials (e.g. phone passkey + hardware key).
type UserMFAWebAuthnCredential struct {
	CredentialID             int64          `gorm:"column:credential_id;primaryKey"`
	CredentialUUID           uuid.UUID      `gorm:"column:credential_uuid"`
	UserID                   int64          `gorm:"column:user_id;not null"`
	CredentialKeyID          string         `gorm:"column:credential_key_id;not null;unique"`
	PublicKey                []byte         `gorm:"column:public_key;not null"`
	AAGUID                   *uuid.UUID     `gorm:"column:aaguid;type:uuid"`
	SignCount                int64          `gorm:"column:sign_count;default:0"`
	Transport                pq.StringArray `gorm:"column:transport;type:text[];default:'{}'"`
	IsBackupEligible         bool           `gorm:"column:is_backup_eligible;default:false"`
	IsBackupActive           bool           `gorm:"column:is_backup_active;default:false"`
	IsDiscoverableCredential bool           `gorm:"column:is_discoverable_credential;not null;default:false"`
	Name                     string         `gorm:"column:name;default:'Security Key'"`
	LastUsedAt               *time.Time     `gorm:"column:last_used_at"`
	CreatedAt                time.Time      `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt                time.Time      `gorm:"column:updated_at;autoUpdateTime"`
}

func (UserMFAWebAuthnCredential) TableName() string { return "user_mfa_webauthn_credentials" }

func (c *UserMFAWebAuthnCredential) BeforeCreate(tx *gorm.DB) error {
	if c.CredentialUUID == uuid.Nil {
		c.CredentialUUID = uuid.New()
	}
	return nil
}
