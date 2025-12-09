package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type APIKeyApi struct {
	APIKeyApiID   int64     `gorm:"column:api_key_api_id;primaryKey;autoIncrement"`
	APIKeyApiUUID uuid.UUID `gorm:"column:api_key_api_uuid;type:uuid;not null;uniqueIndex"`
	APIKeyID      int64     `gorm:"column:api_key_id;not null;index"`
	APIID         int64     `gorm:"column:api_id;not null;index"`
	CreatedAt     time.Time `gorm:"column:created_at;default:now()"`

	// Relationships
	APIKey      APIKey               `gorm:"foreignKey:APIKeyID;references:APIKeyID"`
	API         API                  `gorm:"foreignKey:APIID;references:APIID"`
	Permissions []APIKeyPermission   `gorm:"foreignKey:APIKeyApiID;references:APIKeyApiID"`
}

func (APIKeyApi) TableName() string {
	return "api_key_apis"
}

func (a *APIKeyApi) BeforeCreate(tx *gorm.DB) (err error) {
	if a.APIKeyApiUUID == uuid.Nil {
		a.APIKeyApiUUID = uuid.New()
	}
	return
}
