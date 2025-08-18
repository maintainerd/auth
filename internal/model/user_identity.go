package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type UserIdentity struct {
	UserIdentityID   int64          `gorm:"column:user_identity_id;primaryKey"`
	UserIdentityUUID uuid.UUID      `gorm:"column:user_identity_uuid;unique"`
	UserID           int64          `gorm:"column:user_id"`
	ProviderName     string         `gorm:"column:provider_name"`
	ProviderUserID   string         `gorm:"column:provider_user_id"`
	Email            *string        `gorm:"column:email"`
	RawProfile       datatypes.JSON `gorm:"column:raw_profile"`
	CreatedAt        time.Time      `gorm:"column:created_at;autoCreateTime"`

	// Relationships
	User *User `gorm:"foreignKey:UserID;references:UserID;constraint:OnDelete:CASCADE"`
}

func (UserIdentity) TableName() string {
	return "user_identities"
}

func (ui *UserIdentity) BeforeCreate(tx *gorm.DB) (err error) {
	if ui.UserIdentityUUID == uuid.Nil {
		ui.UserIdentityUUID = uuid.New()
	}
	return
}
