package model

import (
	"time"

	"github.com/google/uuid"
)

type AuthProvider struct {
	AuthProviderID   int64      `gorm:"primaryKey;column:auth_provider_id"`
	AuthProviderUUID uuid.UUID  `gorm:"type:uuid;not null;unique;column:auth_provider_uuid"`
	ProviderName     string     `gorm:"type:varchar(100);not null;index:idx_auth_providers_provider_name"`
	ClientID         *string    `gorm:"type:text"`
	ClientSecret     *string    `gorm:"type:text"`
	RedirectURI      *string    `gorm:"type:text"`
	MetadataURL      *string    `gorm:"type:text"`
	IsActive         bool       `gorm:"default:true"`
	CreatedAt        time.Time  `gorm:"autoCreateTime"`
	UpdatedAt        *time.Time `gorm:"autoUpdateTime"`
}

// TableName overrides the default table name
func (AuthProvider) TableName() string {
	return "auth_providers"
}
