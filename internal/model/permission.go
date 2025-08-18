package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Permission struct {
	PermissionID    int64     `gorm:"column:permission_id;primaryKey"`
	PermissionUUID  uuid.UUID `gorm:"column:permission_uuid"`
	Name            string    `gorm:"column:name"`
	Description     string    `gorm:"column:description"`
	IsActive        bool      `gorm:"column:is_active"`
	IsDefault       bool      `gorm:"column:is_default"`
	APIID           int64     `gorm:"column:api_id"`
	AuthContainerID int64     `gorm:"column:auth_container_id"`
	CreatedAt       time.Time `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt       time.Time `gorm:"column:updated_at;autoUpdateTime"`

	// Relationships
	API           *API           `gorm:"foreignKey:APIID;references:APIID"`
	AuthContainer *AuthContainer `gorm:"foreignKey:AuthContainerID;references:AuthContainerID"`
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
