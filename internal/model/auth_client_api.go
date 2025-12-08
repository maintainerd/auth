package model

import (
	"time"

	"github.com/google/uuid"
)

type AuthClientApi struct {
	AuthClientApiID   int64     `gorm:"column:auth_client_api_id;primaryKey;autoIncrement"`
	AuthClientApiUUID uuid.UUID `gorm:"column:auth_client_api_uuid;type:uuid;not null;uniqueIndex"`
	AuthClientID      int64     `gorm:"column:auth_client_id;not null;index"`
	APIID             int64     `gorm:"column:api_id;not null;index"`
	CreatedAt         time.Time `gorm:"column:created_at;default:now()"`

	// Relationships
	AuthClient  AuthClient             `gorm:"foreignKey:AuthClientID;references:AuthClientID"`
	API         API                    `gorm:"foreignKey:APIID;references:APIID"`
	Permissions []AuthClientPermission `gorm:"foreignKey:AuthClientApiID;references:AuthClientApiID"`
}

func (AuthClientApi) TableName() string {
	return "auth_client_apis"
}
