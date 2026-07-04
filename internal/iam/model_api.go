package iam

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type API struct {
	APIID       int64          `gorm:"column:api_id;primaryKey"`
	APIUUID     uuid.UUID      `gorm:"column:api_uuid"`
	TenantID    int64          `gorm:"column:tenant_id;not null"`
	ServiceID   int64          `gorm:"column:service_id"`
	Name        string         `gorm:"column:name"`
	DisplayName string         `gorm:"column:display_name"`
	Description string         `gorm:"column:description"`
	Identifier  string         `gorm:"column:identifier"`
	Status      string         `gorm:"column:status;not null;default:'inactive'"`
	IsSystem    bool           `gorm:"column:is_system;not null;default:false"`
	CreatedBy   *int64         `gorm:"column:created_by"`
	UpdatedBy   *int64         `gorm:"column:updated_by"`
	CreatedAt   time.Time      `gorm:"column:created_at;not null;autoCreateTime"`
	UpdatedAt   time.Time      `gorm:"column:updated_at;not null;autoUpdateTime"`
	DeletedAt   gorm.DeletedAt `gorm:"column:deleted_at;index"`

	Service *Service `gorm:"foreignKey:ServiceID;references:ServiceID"`
}

func (API) TableName() string {
	return "apis"
}

func (a *API) BeforeCreate(_ *gorm.DB) error {
	if a.APIUUID == uuid.Nil {
		a.APIUUID = uuid.New()
	}
	return nil
}
