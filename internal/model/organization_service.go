package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type OrganizationService struct {
	OrganizationServiceID   int64     `gorm:"column:organization_service_id;primaryKey"`
	OrganizationServiceUUID uuid.UUID `gorm:"column:organization_service_uuid;unique"`
	OrganizationID          int64     `gorm:"column:organization_id"`
	ServiceID               int64     `gorm:"column:service_id"`
	CreatedAt               time.Time `gorm:"column:created_at;autoCreateTime"`

	// Relationships
	Organization *Organization `gorm:"foreignKey:OrganizationID;references:OrganizationID"`
	Service      *Service      `gorm:"foreignKey:ServiceID;references:ServiceID"`
}

func (OrganizationService) TableName() string {
	return "organization_services"
}

func (os *OrganizationService) BeforeCreate(tx *gorm.DB) (err error) {
	if os.OrganizationServiceUUID == uuid.Nil {
		os.OrganizationServiceUUID = uuid.New()
	}
	return
}
