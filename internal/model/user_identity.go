package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type UserIdentity struct {
	UserIdentityID   int64          `gorm:"column:user_identity_id;primaryKey"`
	UserIdentityUUID uuid.UUID      `gorm:"column:user_identity_uuid;type:uuid;not null;unique"`
	UserID           int64          `gorm:"column:user_id;type:integer;not null"`
	ProviderName     string         `gorm:"column:provider_name;type:varchar(100);not null"`
	ProviderUserID   string         `gorm:"column:provider_user_id;type:varchar(255);not null"`
	Email            *string        `gorm:"column:email;type:varchar(255)"`
	RawProfile       datatypes.JSON `gorm:"column:raw_profile;type:jsonb"`
	CreatedAt        time.Time      `gorm:"column:created_at;type:timestamptz;autoCreateTime"`

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
