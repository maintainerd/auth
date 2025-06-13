package model

import (
	"time"

	"github.com/google/uuid"
)

type UserPermission struct {
	UserPermissionID   int64     `gorm:"primaryKey;column:user_permission_id"`
	UserPermissionUUID uuid.UUID `gorm:"type:uuid;not null;unique;column:user_permission_uuid;index:idx_user_permissions_uuid"`

	UserID       int64 `gorm:"not null;index:idx_user_permissions_user_id"`
	PermissionID int64 `gorm:"not null;index:idx_user_permissions_permission_id"`

	IsDefault bool       `gorm:"default:false"`
	CreatedAt time.Time  `gorm:"autoCreateTime"`
	UpdatedAt *time.Time `gorm:"autoUpdateTime"`

	// Relations (optional, for eager loading)
	User       User       `gorm:"foreignKey:UserID;references:UserID;constraint:OnDelete:CASCADE"`
	Permission Permission `gorm:"foreignKey:PermissionID;references:PermissionID;constraint:OnDelete:CASCADE"`
}

// TableName overrides the default table name
func (UserPermission) TableName() string {
	return "user_permissions"
}
