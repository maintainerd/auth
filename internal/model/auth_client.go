package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type AuthClient struct {
	AuthClientID       int64          `gorm:"column:auth_client_id;primaryKey"`
	AuthClientUUID     uuid.UUID      `gorm:"column:auth_client_uuid"`
	Name               string         `gorm:"column:name"`
	DisplayName        string         `gorm:"column:display_name"`
	ClientType         string         `gorm:"column:client_type"`
	Domain             *string        `gorm:"column:domain"`
	ClientID           *string        `gorm:"column:client_id"`
	ClientSecret       *string        `gorm:"column:client_secret"`
	RedirectURI        *string        `gorm:"column:redirect_uri"`
	Config             datatypes.JSON `gorm:"column:config"`
	IsActive           bool           `gorm:"column:is_active;default:false"`
	IsDefault          bool           `gorm:"column:is_default;default:false"`
	IdentityProviderID int64          `gorm:"column:identity_provider_id"`
	CreatedAt          time.Time      `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt          time.Time      `gorm:"column:updated_at;autoUpdateTime"`

	// Relationships
	IdentityProvider *IdentityProvider `gorm:"foreignKey:IdentityProviderID;references:IdentityProviderID"`
}

func (AuthClient) TableName() string {
	return "auth_clients"
}

func (ac *AuthClient) BeforeCreate(tx *gorm.DB) (err error) {
	if ac.AuthClientUUID == uuid.Nil {
		ac.AuthClientUUID = uuid.New()
	}
	return
}
