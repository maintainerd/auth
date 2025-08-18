package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Role struct {
	RoleID          int64     `gorm:"column:role_id;primaryKey"`
	RoleUUID        uuid.UUID `gorm:"column:role_uuid;unique"`
	Name            string    `gorm:"column:name;unique"`
	Description     string    `gorm:"column:description"`
	IsActive        bool      `gorm:"column:is_active;default:false"`
	IsDefault       bool      `gorm:"column:is_default;default:false"`
	AuthContainerID int64     `gorm:"column:auth_container_id"`
	CreatedAt       time.Time `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt       time.Time `gorm:"column:updated_at;autoUpdateTime"`

	// Relationships
	AuthContainer *AuthContainer `gorm:"foreignKey:AuthContainerID;references:AuthContainerID"`
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
