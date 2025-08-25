package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Organization struct {
	OrganizationID      int64     `gorm:"column:organization_id;primaryKey"`
	OrganizationUUID    uuid.UUID `gorm:"column:organization_uuid"`
	Name                string    `gorm:"column:name"`
	Description         *string   `gorm:"column:description"`
	Email               *string   `gorm:"column:email"`
	PhoneNumber         *string   `gorm:"column:phone_number"`
	WebsiteURL          *string   `gorm:"column:website_url"`
	LogoURL             *string   `gorm:"column:logo_url"`
	ExternalReferenceID *string   `gorm:"column:external_reference_id"`
	IsDefault           bool      `gorm:"column:is_default"`
	IsActive            bool      `gorm:"column:is_active"`
	CreatedAt           time.Time `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt           time.Time `gorm:"column:updated_at;autoUpdateTime"`

	// Relationships
	Services []*Service `gorm:"many2many:organization_services;joinForeignKey:OrganizationID;joinReferences:ServiceID"`
}

func (Organization) TableName() string {
	return "organizations"
}

func (o *Organization) BeforeCreate(tx *gorm.DB) (err error) {
	if o.OrganizationUUID == uuid.Nil {
		o.OrganizationUUID = uuid.New()
	}
	return
}
