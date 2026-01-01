package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Role struct {
	RoleID      int64     `gorm:"column:role_id;primaryKey"`
	RoleUUID    uuid.UUID `gorm:"column:role_uuid;unique"`
	TenantID    int64     `gorm:"column:tenant_id;not null"`
	Name        string    `gorm:"column:name"`
	Description string    `gorm:"column:description"`
	Status      string    `gorm:"column:status;type:varchar(16);default:'inactive'"`
	IsDefault   bool      `gorm:"column:is_default;default:false"`
	IsSystem    bool      `gorm:"column:is_system;default:false"`
	CreatedAt   time.Time `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt   time.Time `gorm:"column:updated_at;autoUpdateTime"`

	// Relationships
	Tenant      *Tenant      `gorm:"foreignKey:TenantID;references:TenantID"`
	Permissions []Permission `gorm:"many2many:role_permissions;joinForeignKey:RoleID;joinReferences:PermissionID"`
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
