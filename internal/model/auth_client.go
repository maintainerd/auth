package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type AuthClient struct {
	AuthClientID       int64          `gorm:"column:auth_client_id;primaryKey"`
	AuthClientUUID     uuid.UUID      `gorm:"column:auth_client_uuid;type:uuid;not null;unique"`
	ClientName         string         `gorm:"column:client_name;type:varchar(100);not null"`
	DisplayName        string         `gorm:"column:display_name;type:text;not null"`
	ClientType         string         `gorm:"column:client_type;type:varchar(100);not null"`
	Domain             *string        `gorm:"column:domain;type:text"`
	ClientID           *string        `gorm:"column:client_id;type:text"`
	ClientSecret       *string        `gorm:"column:client_secret;type:text"`
	RedirectURI        *string        `gorm:"column:redirect_uri;type:text"`
	Config             datatypes.JSON `gorm:"column:config;type:jsonb"`
	IsActive           bool           `gorm:"column:is_active;type:boolean;default:false"`
	IsDefault          bool           `gorm:"column:is_default;type:boolean;default:false"`
	IdentityProviderID int64          `gorm:"column:identity_provider_id;type:integer;not null;index:idx_auth_clients_identity_provider_id"`
	AuthContainerID    int64          `gorm:"column:auth_container_id;type:integer;not null;index:idx_auth_clients_auth_container_id"`
	CreatedAt          time.Time      `gorm:"column:created_at;type:timestamptz;autoCreateTime"`
	UpdatedAt          time.Time      `gorm:"column:updated_at;type:timestamptz;autoUpdateTime"`

	// Relationships
	IdentityProvider *IdentityProvider `gorm:"foreignKey:IdentityProviderID;references:IdentityProviderID;constraint:OnDelete:CASCADE"`
	AuthContainer    *AuthContainer    `gorm:"foreignKey:AuthContainerID;references:AuthContainerID;constraint:OnDelete:CASCADE"`
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
