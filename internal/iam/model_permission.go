package iam

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Permission struct {
	PermissionID   int64          `gorm:"column:permission_id;primaryKey"`
	PermissionUUID uuid.UUID      `gorm:"column:permission_uuid"`
	TenantID       int64          `gorm:"column:tenant_id;not null"`
	APIID          int64          `gorm:"column:api_id"`
	Name           string         `gorm:"column:name"`
	Description    string         `gorm:"column:description"`
	Status         string         `gorm:"column:status;not null;default:'active'"`
	IsSystem       bool           `gorm:"column:is_system;not null;default:false"`
	CreatedBy      *int64         `gorm:"column:created_by"`
	UpdatedBy      *int64         `gorm:"column:updated_by"`
	CreatedAt      time.Time      `gorm:"column:created_at;not null;autoCreateTime"`
	UpdatedAt      time.Time      `gorm:"column:updated_at;not null;autoUpdateTime"`
	DeletedAt      gorm.DeletedAt `gorm:"column:deleted_at;index"`

	API   *API   `gorm:"foreignKey:APIID;references:APIID"`
	Roles []Role `gorm:"many2many:role_permissions;joinForeignKey:PermissionID;joinReferences:RoleID"`
}

func (Permission) TableName() string {
	return "permissions"
}

func (p *Permission) BeforeCreate(_ *gorm.DB) error {
	if p.PermissionUUID == uuid.Nil {
		p.PermissionUUID = uuid.New()
	}
	return nil
}
