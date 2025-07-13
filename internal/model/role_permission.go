package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type RolePermission struct {
	RolePermissionID   int64      `gorm:"column:role_permission_id;primaryKey"`
	RolePermissionUUID uuid.UUID  `gorm:"column:role_permission_uuid;type:uuid;not null;unique;index:idx_role_permissions_uuid"`
	RoleID             int64      `gorm:"column:role_id;type:integer;not null;index:idx_role_permissions_role_id"`
	PermissionID       int64      `gorm:"column:permission_id;type:integer;not null;index:idx_role_permissions_permission_id"`
	IsDefault          bool       `gorm:"column:is_default;type:boolean;default:false"`
	CreatedAt          time.Time  `gorm:"column:created_at;type:timestamptz;autoCreateTime"`
	UpdatedAt          *time.Time `gorm:"column:updated_at;type:timestamptz;autoUpdateTime"`

	// Relationships
	Role       *Role       `gorm:"foreignKey:RoleID;references:RoleID;constraint:OnDelete:CASCADE"`
	Permission *Permission `gorm:"foreignKey:PermissionID;references:PermissionID;constraint:OnDelete:CASCADE"`
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
