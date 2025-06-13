package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Permission struct {
	PermissionID    int64     `gorm:"column:permission_id;primaryKey"`
	PermissionUUID  uuid.UUID `gorm:"column:permission_uuid;type:uuid;not null;unique;index:idx_permissions_permission_uuid"`
	Name            string    `gorm:"column:name;type:varchar(255);not null;unique;index:idx_permissions_name"`
	Description     string    `gorm:"column:description;type:text;not null"`
	IsActive        bool      `gorm:"column:is_active;type:boolean;default:false"`
	IsDefault       bool      `gorm:"column:is_default;type:boolean;default:false"`
	APIID           int64     `gorm:"column:api_id;type:integer;not null;index:idx_permissions_api_id"`
	AuthContainerID int64     `gorm:"column:auth_container_id;type:integer;not null;index:idx_permissions_auth_container_id"`
	CreatedAt       time.Time `gorm:"column:created_at;type:timestamptz;autoCreateTime"`
	UpdatedAt       time.Time `gorm:"column:updated_at;type:timestamptz;autoUpdateTime"`

	// Relationships
	API           *API           `gorm:"foreignKey:APIID;references:APIID;constraint:OnDelete:CASCADE"`
	AuthContainer *AuthContainer `gorm:"foreignKey:AuthContainerID;references:AuthContainerID;constraint:OnDelete:CASCADE"`
	Roles         []Role         `gorm:"many2many:role_permissions;joinForeignKey:PermissionID;joinReferences:RoleID"`
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
