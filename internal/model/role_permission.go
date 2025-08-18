package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type RolePermission struct {
	RolePermissionID   int64      `gorm:"column:role_permission_id;primaryKey"`
	RolePermissionUUID uuid.UUID  `gorm:"column:role_permission_uuid;unique"`
	RoleID             int64      `gorm:"column:role_id"`
	PermissionID       int64      `gorm:"column:permission_id"`
	IsDefault          bool       `gorm:"column:is_default;default:false"`
	CreatedAt          time.Time  `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt          *time.Time `gorm:"column:updated_at;autoUpdateTime"`

	// Relationships
	Role       *Role       `gorm:"foreignKey:RoleID;references:RoleID"`
	Permission *Permission `gorm:"foreignKey:PermissionID;references:PermissionID"`
}

func (RolePermission) TableName() string {
	return "role_permissions"
}

func (rp *RolePermission) BeforeCreate(tx *gorm.DB) (err error) {
	if rp.RolePermissionUUID == uuid.Nil {
		rp.RolePermissionUUID = uuid.New()
	}
	return
}
