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
	AuthClientID     int64          `gorm:"column:auth_client_id"`
	Provider         string         `gorm:"column:provider"`
	Sub              string         `gorm:"column:sub"`
	UserData         datatypes.JSON `gorm:"column:user_data"`
	CreatedAt        time.Time      `gorm:"column:created_at;autoCreateTime"`

	// Relationships
	User       *User       `gorm:"foreignKey:UserID;references:UserID;constraint:OnDelete:CASCADE"`
	AuthClient *AuthClient `gorm:"foreignKey:AuthClientID;references:AuthClientID;constraint:OnDelete:CASCADE"` // new relationship
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
