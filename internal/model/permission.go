package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Permission struct {
	PermissionID   int64     `gorm:"column:permission_id;primaryKey"`
	PermissionUUID uuid.UUID `gorm:"column:permission_uuid"`
	Name           string    `gorm:"column:name;unique"`
	Description    string    `gorm:"column:description"`
	APIID          int64     `gorm:"column:api_id"`
	Status         string    `gorm:"column:status;default:'active'"`
	IsDefault      bool      `gorm:"column:is_default;default:false"`
	IsSystem       bool      `gorm:"column:is_system;default:false"`
	CreatedAt      time.Time `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt      time.Time `gorm:"column:updated_at;autoUpdateTime"`

	// Relationships
	API                   *API                   `gorm:"foreignKey:APIID;references:APIID"`
	Roles                 []Role                 `gorm:"many2many:role_permissions;joinForeignKey:PermissionID;joinReferences:RoleID"`
	AuthClientPermissions []AuthClientPermission `gorm:"foreignKey:PermissionID;references:PermissionID"`
}

func (Permission) TableName() string {
	return "permissions"
}

func (p *Permission) BeforeCreate(tx *gorm.DB) (err error) {
	if p.PermissionUUID == uuid.Nil {
		p.PermissionUUID = uuid.New()
	}
	return
}
