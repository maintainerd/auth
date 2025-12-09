package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type APIKeyPermission struct {
	APIKeyPermissionID   int64     `gorm:"column:api_key_permission_id;primaryKey"`
	APIKeyPermissionUUID uuid.UUID `gorm:"column:api_key_permission_uuid;unique"`
	APIKeyApiID          int64     `gorm:"column:api_key_api_id;uniqueIndex:idx_api_key_permission_unique"`
	PermissionID         int64     `gorm:"column:permission_id;uniqueIndex:idx_api_key_permission_unique"`
	CreatedAt            time.Time `gorm:"column:created_at;autoCreateTime"`

	// Relationships
	APIKeyApi  *APIKeyApi  `gorm:"foreignKey:APIKeyApiID;references:APIKeyApiID;constraint:OnDelete:CASCADE"`
	Permission *Permission `gorm:"foreignKey:PermissionID;references:PermissionID;constraint:OnDelete:CASCADE"`
}

func (APIKeyPermission) TableName() string {
	return "api_key_permissions"
}

func (akp *APIKeyPermission) BeforeCreate(tx *gorm.DB) (err error) {
	if akp.APIKeyPermissionUUID == uuid.Nil {
		akp.APIKeyPermissionUUID = uuid.New()
	}
	return
}
