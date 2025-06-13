package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type IdentityProvider struct {
	IdentityProviderID   int64          `gorm:"column:identity_provider_id;primaryKey"`
	IdentityProviderUUID uuid.UUID      `gorm:"column:identity_provider_uuid;type:uuid;not null;unique"`
	ProviderName         string         `gorm:"column:provider_name;type:varchar(100);not null;index:idx_identity_providers_provider_name"`
	DisplayName          string         `gorm:"column:display_name;type:text;not null"`
	ProviderType         string         `gorm:"column:provider_type;type:varchar(100);not null"`
	Identifier           *string        `gorm:"column:identifier;type:text"`
	Config               datatypes.JSON `gorm:"column:config;type:jsonb"`
	IsActive             bool           `gorm:"column:is_active;type:boolean;default:false"`
	IsDefault            bool           `gorm:"column:is_default;type:boolean;default:false"`
	AuthContainerID      int64          `gorm:"column:auth_container_id;type:integer;not null;index:idx_identity_providers_auth_container_id"`
	CreatedAt            time.Time      `gorm:"column:created_at;type:timestamptz;autoCreateTime"`
	UpdatedAt            time.Time      `gorm:"column:updated_at;type:timestamptz;autoUpdateTime"`

	// Relationships
	AuthContainer *AuthContainer `gorm:"foreignKey:AuthContainerID;references:AuthContainerID;constraint:OnDelete:CASCADE"`
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
