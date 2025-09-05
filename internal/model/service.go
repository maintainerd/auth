package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Service struct {
	ServiceID   int64     `gorm:"column:service_id;primaryKey"`
	ServiceUUID uuid.UUID `gorm:"column:service_uuid;unique"`
	Name        string    `gorm:"column:name"`
	DisplayName string    `gorm:"column:display_name"`
	Description string    `gorm:"column:description"`
	Version     string    `gorm:"column:version"`
	IsActive    bool      `gorm:"column:is_active;default:false"`
	IsDefault   bool      `gorm:"column:is_default;default:false"`
	IsPublic    bool      `gorm:"column:is_public;default:false"`
	CreatedAt   time.Time `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt   time.Time `gorm:"column:updated_at;autoUpdateTime"`
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
