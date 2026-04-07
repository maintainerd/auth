package model

import (
	"time"

	"github.com/google/uuid"
)

type ClientAPI struct {
	ClientAPIID   int64     `gorm:"column:client_api_id;primaryKey;autoIncrement"`
	ClientAPIUUID uuid.UUID `gorm:"column:client_api_uuid;type:uuid;not null;uniqueIndex"`
	ClientID      int64     `gorm:"column:client_id;not null;index"`
	APIID         int64     `gorm:"column:api_id;not null;index"`
	CreatedAt     time.Time `gorm:"column:created_at;default:now()"`

	// Relationships
	Client      Client             `gorm:"foreignKey:ClientID;references:ClientID"`
	API         API                `gorm:"foreignKey:APIID;references:APIID"`
	Permissions []ClientPermission `gorm:"foreignKey:ClientAPIID;references:ClientAPIID"`
}

func (ClientAPI) TableName() string {
	return "client_apis"
}
