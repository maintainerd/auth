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
	ProviderName         string         `gorm:"column:provider_name"`
	DisplayName          string         `gorm:"column:display_name"`
	ProviderType         string         `gorm:"column:provider_type"`
	Identifier           *string        `gorm:"column:identifier"`
	Config               datatypes.JSON `gorm:"column:config"`
	IsActive             bool           `gorm:"column:is_active"`
	IsDefault            bool           `gorm:"column:is_default"`
	AuthContainerID      int64          `gorm:"column:auth_container_id"`
	CreatedAt            time.Time      `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt            time.Time      `gorm:"column:updated_at;autoUpdateTime"`

	// Relationships
	AuthContainer *AuthContainer `gorm:"foreignKey:AuthContainerID;references:AuthContainerID"`
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
