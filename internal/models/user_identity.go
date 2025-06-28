package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
)

type UserIdentity struct {
	UserIdentityID   int64     `gorm:"primaryKey;column:user_identity_id"`
	UserIdentityUUID uuid.UUID `gorm:"type:uuid;not null;unique;column:user_identity_uuid"`

	UserID         int64  `gorm:"not null"`
	ProviderName   string `gorm:"type:varchar(100);not null"`
	ProviderUserID string `gorm:"type:varchar(255);not null"`

	Email      *string        `gorm:"type:varchar(255)"`
	RawProfile datatypes.JSON `gorm:"type:jsonb"`

	CreatedAt time.Time `gorm:"autoCreateTime"`

	// Relation
	User User `gorm:"foreignKey:UserID;references:UserID;constraint:OnDelete:CASCADE"`
}

// TableName overrides the default table name
func (UserIdentity) TableName() string {
	return "user_identities"
}
