package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type UserPermission struct {
	UserPermissionID   int64      `gorm:"column:user_permission_id;primaryKey"`
	UserPermissionUUID uuid.UUID  `gorm:"column:user_permission_uuid;type:uuid;not null;unique;index:idx_user_permissions_uuid"`
	UserID             int64      `gorm:"column:user_id;type:integer;not null;index:idx_user_permissions_user_id"`
	PermissionID       int64      `gorm:"column:permission_id;type:integer;not null;index:idx_user_permissions_permission_id"`
	IsDefault          bool       `gorm:"column:is_default;type:boolean;default:false"`
	CreatedAt          time.Time  `gorm:"column:created_at;type:timestamptz;autoCreateTime"`
	UpdatedAt          *time.Time `gorm:"column:updated_at;type:timestamptz;autoUpdateTime"`

	// Relationships
	User       *User       `gorm:"foreignKey:UserID;references:UserID;constraint:OnDelete:CASCADE"`
	Permission *Permission `gorm:"foreignKey:PermissionID;references:PermissionID;constraint:OnDelete:CASCADE"`
}

func (UserPermission) TableName() string {
	return "user_permissions"
}

func (up *UserPermission) BeforeCreate(tx *gorm.DB) (err error) {
	if up.UserPermissionUUID == uuid.Nil {
		up.UserPermissionUUID = uuid.New()
	}
	return
}
