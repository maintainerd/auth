package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Role struct {
	RoleID          int64     `gorm:"column:role_id;primaryKey"`
	RoleUUID        uuid.UUID `gorm:"column:role_uuid;type:uuid;not null;unique;index:idx_roles_role_uuid"`
	Name            string    `gorm:"column:name;type:varchar(255);not null;unique;index:idx_roles_name"`
	Description     string    `gorm:"column:description;type:text;not null"`
	IsActive        bool      `gorm:"column:is_active;type:boolean;default:false"`
	IsDefault       bool      `gorm:"column:is_default;type:boolean;default:false"`
	AuthContainerID int64     `gorm:"column:auth_container_id;type:integer;not null;index:idx_roles_auth_container_id"`
	CreatedAt       time.Time `gorm:"column:created_at;type:timestamptz;autoCreateTime"`
	UpdatedAt       time.Time `gorm:"column:updated_at;type:timestamptz;autoUpdateTime"`

	// Relationships
	AuthContainer *AuthContainer `gorm:"foreignKey:AuthContainerID;references:AuthContainerID;constraint:OnDelete:CASCADE"`
	Permissions   []Permission   `gorm:"many2many:role_permissions;joinForeignKey:RoleID;joinReferences:PermissionID"`
}

func (Role) TableName() string {
	return "roles"
}

func (r *Role) BeforeCreate(tx *gorm.DB) (err error) {
	if r.RoleUUID == uuid.Nil {
		r.RoleUUID = uuid.New()
	}
	return
}
