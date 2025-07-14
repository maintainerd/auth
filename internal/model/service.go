package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type Service struct {
	ServiceID   int64          `gorm:"column:service_id;primaryKey"`
	ServiceUUID uuid.UUID      `gorm:"column:service_uuid;type:uuid;not null;unique;index:idx_services_service_uuid"`
	ServiceName string         `gorm:"column:service_name;type:varchar(100);not null;index:idx_services_service_name"` // e.g., "auth"
	DisplayName string         `gorm:"column:display_name;type:text;not null;index:idx_services_display_name"`
	Description string         `gorm:"column:description;type:text;not null"`
	ServiceType string         `gorm:"column:service_type;type:text;not null;index:idx_services_service_type"` // e.g., "default"
	Version     string         `gorm:"column:version;type:varchar(20);not null"`
	Config      datatypes.JSON `gorm:"column:config;type:jsonb"`
	IsActive    bool           `gorm:"column:is_active;type:boolean;default:false"`
	IsDefault   bool           `gorm:"column:is_default;type:boolean;default:false"`
	CreatedAt   time.Time      `gorm:"column:created_at;type:timestamptz;autoCreateTime"`
	UpdatedAt   time.Time      `gorm:"column:updated_at;type:timestamptz;autoUpdateTime"`
}

func (Service) TableName() string {
	return "services"
}

func (s *Service) BeforeCreate(tx *gorm.DB) (err error) {
	if s.ServiceUUID == uuid.Nil {
		s.ServiceUUID = uuid.New()
	}
	return
}
