package models

import (
	"time"

	"github.com/google/uuid"
)

type RolePermission struct {
	RolePermissionID   int64     `gorm:"primaryKey;column:role_permission_id"`
	RolePermissionUUID uuid.UUID `gorm:"type:uuid;not null;unique;column:role_permission_uuid;index:idx_role_permissions_uuid"`

	RoleID       int64 `gorm:"not null;index:idx_role_permissions_role_id"`
	PermissionID int64 `gorm:"not null;index:idx_role_permissions_permission_id"`

	IsDefault bool       `gorm:"default:false"`
	CreatedAt time.Time  `gorm:"autoCreateTime"`
	UpdatedAt *time.Time `gorm:"autoUpdateTime"`

	// Relations (optional, if using eager loading)
	Role       Role       `gorm:"foreignKey:RoleID;references:RoleID;constraint:OnDelete:CASCADE"`
	Permission Permission `gorm:"foreignKey:PermissionID;references:PermissionID;constraint:OnDelete:CASCADE"`
}

// TableName overrides the default table name
func (RolePermission) TableName() string {
	return "role_permissions"
}
