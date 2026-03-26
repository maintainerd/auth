package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type Client struct {
	ClientID           int64          `gorm:"column:client_id;primaryKey"`
	ClientUUID         uuid.UUID      `gorm:"column:client_uuid"`
	TenantID           int64          `gorm:"column:tenant_id;not null"`
	IdentityProviderID int64          `gorm:"column:identity_provider_id"`
	Name               string         `gorm:"column:name"`
	DisplayName        string         `gorm:"column:display_name"`
	ClientType         string         `gorm:"column:client_type"`
	Domain             *string        `gorm:"column:domain"`
	Identifier         *string        `gorm:"column:identifier"`
	Secret             *string        `gorm:"column:secret"`
	Config             datatypes.JSON `gorm:"column:config"`
	Status             string         `gorm:"column:status;default:'inactive'"`
	IsDefault          bool           `gorm:"column:is_default;default:false"`
	IsSystem           bool           `gorm:"column:is_system;default:false"`
	CreatedAt          time.Time      `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt          time.Time      `gorm:"column:updated_at;autoUpdateTime"`

	// Relationships
	IdentityProvider *IdentityProvider `gorm:"foreignKey:IdentityProviderID;references:IdentityProviderID"`
	ClientURIs       *[]ClientURI      `gorm:"foreignKey:ClientID;references:ClientID"`
	ClientApis       *[]ClientApi      `gorm:"foreignKey:ClientID;references:ClientID"`
}

func (Client) TableName() string {
	return "clients"
}

func (ac *Client) BeforeCreate(tx *gorm.DB) (err error) {
	if ac.ClientUUID == uuid.Nil {
		ac.ClientUUID = uuid.New()
	}
	return
}
