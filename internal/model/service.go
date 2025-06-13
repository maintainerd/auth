package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
)

type Service struct {
	ServiceID   int            `gorm:"column:service_id;primaryKey;autoIncrement"`
	ServiceUUID uuid.UUID      `gorm:"column:service_uuid;type:uuid;not null;unique"`
	ServiceName string         `gorm:"column:service_name;type:varchar(100);not null"`
	DisplayName string         `gorm:"column:display_name;type:text;not null"`
	Description string         `gorm:"column:description;type:text;not null"`
	ServiceType string         `gorm:"column:service_type;type:text;not null"`
	Version     string         `gorm:"column:version;type:varchar(20);not null"`
	Config      datatypes.JSON `gorm:"column:config;type:jsonb"`
	IsActive    bool           `gorm:"column:is_active;default:false"`
	IsDefault   bool           `gorm:"column:is_default;default:false"`
	CreatedAt   time.Time      `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt   time.Time      `gorm:"column:updated_at;autoUpdateTime"`
}

// TableName sets the table name explicitly
func (Service) TableName() string {
	return "services"
}
