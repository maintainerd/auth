package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Tenant struct {
	TenantID    int64     `gorm:"column:tenant_id;primaryKey"`
	TenantUUID  uuid.UUID `gorm:"column:tenant_uuid"`
	Name        string    `gorm:"column:name"`
	Description string    `gorm:"column:description"`
	Identifier  string    `gorm:"column:identifier"`
	IsActive    bool      `gorm:"column:is_active;default:false"`
	IsPublic    bool      `gorm:"column:is_public;default:false"`
	IsDefault   bool      `gorm:"column:is_default;default:false"`
	CreatedAt   time.Time `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt   time.Time `gorm:"column:updated_at;autoUpdateTime"`

	// Relationships
	Services          []Service           `gorm:"many2many:tenant_services;joinForeignKey:TenantID;joinReferences:ServiceID"`
	IdentityProviders []*IdentityProvider `gorm:"foreignKey:TenantID;references:TenantID"`
	Roles             []*Role             `gorm:"foreignKey:TenantID;references:TenantID"`
	Users             []*User             `gorm:"foreignKey:TenantID;references:TenantID"`
	AuthLogs          []*AuthLog          `gorm:"foreignKey:TenantID;references:TenantID"`
}

func (Tenant) TableName() string {
	return "tenants"
}

func (t *Tenant) BeforeCreate(tx *gorm.DB) (err error) {
	if t.TenantUUID == uuid.Nil {
		t.TenantUUID = uuid.New()
	}
	return
}
