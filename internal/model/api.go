package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type API struct {
	APIID           int64     `gorm:"column:api_id;primaryKey"`
	APIUUID         uuid.UUID `gorm:"column:api_uuid;type:uuid;not null;unique;index:idx_apis_api_uuid"`
	APIName         string    `gorm:"column:api_name;type:varchar(100);not null;index:idx_apis_api_name"`
	DisplayName     string    `gorm:"column:display_name;type:text;not null"`
	APIType         string    `gorm:"column:api_type;type:text;not null;index:idx_apis_api_type"`
	Description     string    `gorm:"column:description;type:text;not null"`
	Identifier      string    `gorm:"column:identifier;type:text;not null;index:idx_apis_identifier"`
	IsActive        bool      `gorm:"column:is_active;type:boolean;default:false"`
	IsDefault       bool      `gorm:"column:is_default;type:boolean;default:false"`
	ServiceID       int64     `gorm:"column:service_id;type:integer;not null;index:idx_apis_service_id"`
	AuthContainerID int64     `gorm:"column:auth_container_id;type:integer;not null;index:idx_apis_auth_container_id"`
	CreatedAt       time.Time `gorm:"column:created_at;type:timestamptz;autoCreateTime"`
	UpdatedAt       time.Time `gorm:"column:updated_at;type:timestamptz;autoUpdateTime"`

	// Relationships
	Service       *Service       `gorm:"foreignKey:ServiceID;references:ServiceID;constraint:OnDelete:CASCADE"`
	AuthContainer *AuthContainer `gorm:"foreignKey:AuthContainerID;references:AuthContainerID;constraint:OnDelete:CASCADE"`
}

func (API) TableName() string {
	return "apis"
}

func (a *API) BeforeCreate(tx *gorm.DB) (err error) {
	if a.APIUUID == uuid.Nil {
		a.APIUUID = uuid.New()
	}
	return
}
