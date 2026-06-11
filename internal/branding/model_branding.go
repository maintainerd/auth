package branding

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type Branding struct {
	BrandingID        int64          `gorm:"column:branding_id;primaryKey;autoIncrement" json:"branding_id"`
	BrandingUUID      uuid.UUID      `gorm:"column:branding_uuid;type:uuid;uniqueIndex;not null" json:"branding_uuid"`
	TenantID          int64          `gorm:"column:tenant_id;not null" json:"tenant_id"`
	Name              string         `gorm:"column:name;type:varchar(100)" json:"name"`
	IsSystem          bool           `gorm:"column:is_system;not null;default:false" json:"is_system"`
	IsActive          bool           `gorm:"column:is_active;not null;default:false" json:"is_active"`
	CompanyName       string         `gorm:"column:company_name;type:varchar(255)" json:"company_name"`
	LogoURL           string         `gorm:"column:logo_url;type:text" json:"logo_url"`
	FaviconURL        string         `gorm:"column:favicon_url;type:text" json:"favicon_url"`
	SupportURL        string         `gorm:"column:support_url;type:text" json:"support_url"`
	PrivacyPolicyURL  string         `gorm:"column:privacy_policy_url;type:text" json:"privacy_policy_url"`
	TermsOfServiceURL string         `gorm:"column:terms_of_service_url;type:text" json:"terms_of_service_url"`
	// Metadata holds all theme tokens (colors, fonts, panel backgrounds, …) as a
	// flexible JSON object so the palette can extend without schema changes.
	Metadata          datatypes.JSON `gorm:"column:metadata;type:jsonb;default:'{}'" json:"metadata"`
	CreatedBy         *int64         `gorm:"column:created_by" json:"created_by,omitempty"`
	UpdatedBy         *int64         `gorm:"column:updated_by" json:"updated_by,omitempty"`
	CreatedAt         time.Time      `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt         time.Time      `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
	DeletedAt         gorm.DeletedAt `gorm:"column:deleted_at;index" json:"deleted_at,omitempty"`

	// Relationships
}

// TableName returns the database table name for Branding.
func (Branding) TableName() string {
	return "branding"
}

// BeforeCreate sets a new UUID on the Branding before it is inserted into the
// database if one has not already been assigned.
func (b *Branding) BeforeCreate(tx *gorm.DB) error {
	if b.BrandingUUID == uuid.Nil {
		b.BrandingUUID = uuid.New()
	}
	return nil
}
