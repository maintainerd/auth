package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type API struct {
	APIID           int64     `gorm:"column:api_id;primaryKey"`
	APIUUID         uuid.UUID `gorm:"column:api_uuid"`
	APIName         string    `gorm:"column:api_name"`
	DisplayName     string    `gorm:"column:display_name"`
	APIType         string    `gorm:"column:api_type"`
	Description     string    `gorm:"column:description"`
	Identifier      string    `gorm:"column:identifier"`
	IsActive        bool      `gorm:"column:is_active"`
	IsDefault       bool      `gorm:"column:is_default"`
	ServiceID       int64     `gorm:"column:service_id"`
	AuthContainerID int64     `gorm:"column:auth_container_id"`
	CreatedAt       time.Time `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt       time.Time `gorm:"column:updated_at;autoUpdateTime"`

	// Relationships
	Service       *Service       `gorm:"foreignKey:ServiceID;references:ServiceID"`
	AuthContainer *AuthContainer `gorm:"foreignKey:AuthContainerID;references:AuthContainerID"`
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
