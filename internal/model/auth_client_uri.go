package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type AuthClientURI struct {
	AuthClientURIID   int64     `gorm:"column:auth_client_uri_id;primaryKey"`
	AuthClientURIUUID uuid.UUID `gorm:"column:auth_client_uri_uuid"`
	AuthClientID      int64     `gorm:"column:auth_client_id"`
	URI               string    `gorm:"column:uri"`
	Type              string    `gorm:"column:type;default:'redirect-uri'"`
	CreatedAt         time.Time `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt         time.Time `gorm:"column:updated_at;autoUpdateTime"`

	// Relationships
	AuthClient *AuthClient `gorm:"foreignKey:AuthClientID;references:AuthClientID"`
}

func (AuthClientURI) TableName() string {
	return "auth_client_uris"
}

func (acu *AuthClientURI) BeforeCreate(tx *gorm.DB) (err error) {
	if acu.AuthClientURIUUID == uuid.Nil {
		acu.AuthClientURIUUID = uuid.New()
	}
	return
}
