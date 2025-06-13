package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Organization struct {
	OrganizationID      int64     `gorm:"column:organization_id;primaryKey"`
	OrganizationUUID    uuid.UUID `gorm:"column:organization_uuid;type:uuid;not null;unique;index:idx_organizations_organization_uuid"`
	Name                string    `gorm:"column:name;type:varchar(255);not null;index:idx_organizations_name"`
	Description         *string   `gorm:"column:description;type:text"`
	Email               *string   `gorm:"column:email;type:varchar(255);index:idx_organizations_email"`
	PhoneNumber         *string   `gorm:"column:phone_number;type:varchar(50);index:idx_organizations_phone_number"`
	WebsiteURL          *string   `gorm:"column:website_url;type:text"`
	LogoURL             *string   `gorm:"column:logo_url;type:text"`
	ExternalReferenceID *string   `gorm:"column:external_reference_id;type:varchar(255);index:idx_organizations_external_reference_id"`
	IsDefault           bool      `gorm:"column:is_default;type:boolean;default:false;index:idx_organizations_is_default"`
	IsActive            bool      `gorm:"column:is_active;type:boolean;default:true;index:idx_organizations_is_active"`
	CreatedAt           time.Time `gorm:"column:created_at;type:timestamptz;autoCreateTime"`
	UpdatedAt           time.Time `gorm:"column:updated_at;type:timestamptz;autoUpdateTime"`
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
