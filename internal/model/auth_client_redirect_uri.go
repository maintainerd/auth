package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type AuthClientRedirectURI struct {
	AuthClientRedirectURIID   int64     `gorm:"column:auth_client_redirect_uri_id;primaryKey"`
	AuthClientRedirectURIUUID uuid.UUID `gorm:"column:auth_client_redirect_uri_uuid"`
	AuthClientID              int64     `gorm:"column:auth_client_id"`
	RedirectURI               string    `gorm:"column:redirect_uri"`
	CreatedAt                 time.Time `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt                 time.Time `gorm:"column:updated_at;autoUpdateTime"`

	// Relationships
	AuthClient *AuthClient `gorm:"foreignKey:AuthClientID;references:AuthClientID"`
}

func (AuthClientRedirectURI) TableName() string {
	return "auth_client_redirect_uris"
}

func (acru *AuthClientRedirectURI) BeforeCreate(tx *gorm.DB) (err error) {
	if acru.AuthClientRedirectURIUUID == uuid.Nil {
		acru.AuthClientRedirectURIUUID = uuid.New()
	}
	return
}
