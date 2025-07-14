package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type UserRole struct {
	UserRoleID   int64      `gorm:"column:user_role_id;primaryKey"`
	UserRoleUUID uuid.UUID  `gorm:"column:user_role_uuid;type:uuid;not null;unique;index:idx_user_roles_user_role_uuid"`
	UserID       int64      `gorm:"column:user_id;type:integer;not null;index:idx_user_roles_user_id"`
	RoleID       int64      `gorm:"column:role_id;type:integer;not null;index:idx_user_roles_role_id"`
	IsDefault    bool       `gorm:"column:is_default;type:boolean;default:false"`
	CreatedAt    time.Time  `gorm:"column:created_at;type:timestamptz;autoCreateTime"`
	UpdatedAt    *time.Time `gorm:"column:updated_at;type:timestamptz;autoUpdateTime"`

	// Relationships
	User *User `gorm:"foreignKey:UserID;references:UserID;constraint:OnDelete:CASCADE"`
	Role *Role `gorm:"foreignKey:RoleID;references:RoleID;constraint:OnDelete:CASCADE"`
}

func (UserRole) TableName() string {
	return "user_roles"
}

func (ur *UserRole) BeforeCreate(tx *gorm.DB) (err error) {
	if ur.UserRoleUUID == uuid.Nil {
		ur.UserRoleUUID = uuid.New()
	}
	return
}
