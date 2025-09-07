package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type AuthClientPermission struct {
	AuthClientPermissionID   int64     `gorm:"column:auth_client_permission_id;primaryKey"`
	AuthClientPermissionUUID uuid.UUID `gorm:"column:auth_client_permission_uuid"`
	AuthClientID             int64     `gorm:"column:auth_client_id"`
	PermissionID             int64     `gorm:"column:permission_id"`
	CreatedAt                time.Time `gorm:"column:created_at;autoCreateTime"`

	// Relationships
	AuthClient *AuthClient `gorm:"foreignKey:AuthClientID;references:AuthClientID"`
	Permission *Permission `gorm:"foreignKey:PermissionID;references:PermissionID"`
}

func (AuthClientPermission) TableName() string {
	return "auth_client_permissions"
}

func (acp *AuthClientPermission) BeforeCreate(tx *gorm.DB) (err error) {
	if acp.AuthClientPermissionUUID == uuid.Nil {
		acp.AuthClientPermissionUUID = uuid.New()
	}
	return
}
