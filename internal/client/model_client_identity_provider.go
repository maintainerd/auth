package client

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ClientIdentityProvider enables an identity provider connection for a client.
// Clients model downstream OAuth apps; identity providers model upstream login
// systems. This join controls which login options the hosted identity app shows.
type ClientIdentityProvider struct {
	ClientIdentityProviderID   int64     `gorm:"column:client_identity_provider_id;primaryKey"`
	ClientIdentityProviderUUID uuid.UUID `gorm:"column:client_identity_provider_uuid"`
	TenantID                   int64     `gorm:"column:tenant_id;not null"`
	ClientID                   int64     `gorm:"column:client_id;not null"`
	IdentityProviderID         int64     `gorm:"column:identity_provider_id;not null"`
	IsDefault                  bool      `gorm:"column:is_default;default:false"`
	Enabled                    bool      `gorm:"column:enabled;default:true"`
	DisplayOrder               int       `gorm:"column:display_order;default:0"`
	CreatedBy                  *int64    `gorm:"column:created_by"`
	UpdatedBy                  *int64    `gorm:"column:updated_by"`
	CreatedAt                  time.Time `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt                  time.Time `gorm:"column:updated_at;autoUpdateTime"`
	DeletedAt                  gorm.DeletedAt

	Client           *Client           `gorm:"foreignKey:ClientID;references:ClientID"`
	IdentityProvider *IdentityProvider `gorm:"foreignKey:IdentityProviderID;references:IdentityProviderID"`
}

func (ClientIdentityProvider) TableName() string {
	return "client_identity_providers"
}

func (cip *ClientIdentityProvider) BeforeCreate(tx *gorm.DB) error {
	if cip.ClientIdentityProviderUUID == uuid.Nil {
		cip.ClientIdentityProviderUUID = uuid.New()
	}
	return nil
}
