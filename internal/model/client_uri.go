package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ClientURI struct {
	ClientURIID   int64     `gorm:"column:client_uri_id;primaryKey"`
	ClientURIUUID uuid.UUID `gorm:"column:client_uri_uuid"`
	TenantID      int64     `gorm:"column:tenant_id;not null"`
	ClientID      int64     `gorm:"column:client_id"`
	URI           string    `gorm:"column:uri"`
	Type          string    `gorm:"column:type;default:'redirect-uri'"`
	CreatedAt     time.Time `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt     time.Time `gorm:"column:updated_at;autoUpdateTime"`

	// Relationships
	Client *Client `gorm:"foreignKey:ClientID;references:ClientID"`
}

func (ClientURI) TableName() string {
	return "client_uris"
}

func (acu *ClientURI) BeforeCreate(tx *gorm.DB) (err error) {
	if acu.ClientURIUUID == uuid.Nil {
		acu.ClientURIUUID = uuid.New()
	}
	return
}
