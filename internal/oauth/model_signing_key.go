package oauth

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type SigningKey struct {
	SigningKeyID        int64      `gorm:"column:signing_key_id;primaryKey"`
	SigningKeyUUID      uuid.UUID  `gorm:"column:signing_key_uuid;unique;not null"`
	TenantID            *int64     `gorm:"column:tenant_id"`
	KID                 string     `gorm:"column:kid;unique;not null"`
	Algorithm           string     `gorm:"column:algorithm;not null"`
	Use                 string     `gorm:"column:use;not null"`
	Status              string     `gorm:"column:status;not null;default:'active'"`
	PublicKeyPEM        string     `gorm:"column:public_key_pem;not null"`
	PrivateKeyEncrypted []byte     `gorm:"column:private_key_encrypted;not null"`
	KeyEncryptionKeyID  string     `gorm:"column:key_encryption_key_id;not null"`
	RotatedAt           *time.Time `gorm:"column:rotated_at"`
	ExpiresAt           *time.Time `gorm:"column:expires_at"`
	CreatedBy           *int64     `gorm:"column:created_by"`
	CreatedAt           time.Time  `gorm:"column:created_at;not null;autoCreateTime"`
	UpdatedAt           time.Time  `gorm:"column:updated_at;not null;autoUpdateTime"`
}

func (SigningKey) TableName() string {
	return "signing_keys"
}

func (k *SigningKey) BeforeCreate(tx *gorm.DB) (err error) {
	if k.SigningKeyUUID == uuid.Nil {
		k.SigningKeyUUID = uuid.New()
	}
	return
}
