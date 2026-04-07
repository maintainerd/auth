package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ClientPermission struct {
	ClientPermissionID   int64     `gorm:"column:client_permission_id;primaryKey"`
	ClientPermissionUUID uuid.UUID `gorm:"column:client_permission_uuid"`
	ClientAPIID          int64     `gorm:"column:client_api_id"`
	PermissionID         int64     `gorm:"column:permission_id"`
	CreatedAt            time.Time `gorm:"column:created_at;autoCreateTime"`

	// Relationships
	ClientAPI  *ClientAPI  `gorm:"foreignKey:ClientAPIID;references:ClientAPIID"`
	Permission *Permission `gorm:"foreignKey:PermissionID;references:PermissionID"`
}

func (ClientPermission) TableName() string {
	return "client_permissions"
}

func (acp *ClientPermission) BeforeCreate(tx *gorm.DB) (err error) {
	if acp.ClientPermissionUUID == uuid.Nil {
		acp.ClientPermissionUUID = uuid.New()
	}
	return
}
