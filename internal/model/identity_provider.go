package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type IdentityProvider struct {
	IdentityProviderID   int64          `gorm:"column:identity_provider_id;primaryKey"`
	IdentityProviderUUID uuid.UUID      `gorm:"column:identity_provider_uuid"`
	Name                 string         `gorm:"column:name"`
	DisplayName          string         `gorm:"column:display_name"`
	ProviderType         string         `gorm:"column:provider_type"`
	Identifier           string         `gorm:"column:identifier"`
	Config               datatypes.JSON `gorm:"column:config"`
	IsActive             bool           `gorm:"column:is_active;default:false"`
	IsDefault            bool           `gorm:"column:is_default;default:false"`
	TenantID             int64          `gorm:"column:tenant_id"`
	CreatedAt            time.Time      `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt            time.Time      `gorm:"column:updated_at;autoUpdateTime"`

	// Relationships
	Tenant *Tenant `gorm:"foreignKey:TenantID;references:TenantID"`
}

func (IdentityProvider) TableName() string {
	return "identity_providers"
}

func (ip *IdentityProvider) BeforeCreate(tx *gorm.DB) (err error) {
	if ip.IdentityProviderUUID == uuid.Nil {
		ip.IdentityProviderUUID = uuid.New()
	}
	return
}
