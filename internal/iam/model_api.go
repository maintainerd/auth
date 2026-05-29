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
	APIType     string         `gorm:"column:api_type"`
	Identifier  string         `gorm:"column:identifier"`
	Status      string         `gorm:"column:status;default:'inactive'"`
	IsSystem    bool           `gorm:"column:is_system;default:false"`
	CreatedBy   *int64         `gorm:"column:created_by"`
	UpdatedBy   *int64         `gorm:"column:updated_by"`
	CreatedAt   time.Time      `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt   time.Time      `gorm:"column:updated_at;autoUpdateTime"`
	DeletedAt   gorm.DeletedAt `gorm:"column:deleted_at;index"`

	// Relationships
	Service *Service `gorm:"foreignKey:ServiceID;references:ServiceID"`
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
