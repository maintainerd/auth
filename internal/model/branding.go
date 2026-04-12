package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// Branding holds tenant-level UI customisation consumed by auth-console
// (port 8080). Each tenant has at most one branding row.
type Branding struct {
	BrandingID        int64          `gorm:"column:branding_id;primaryKey;autoIncrement" json:"branding_id"`
	BrandingUUID      uuid.UUID      `gorm:"column:branding_uuid;type:uuid;uniqueIndex;not null" json:"branding_uuid"`
	TenantID          int64          `gorm:"column:tenant_id;not null" json:"tenant_id"`
	CompanyName       string         `gorm:"column:company_name;type:varchar(255)" json:"company_name"`
	LogoURL           string         `gorm:"column:logo_url;type:text" json:"logo_url"`
	FaviconURL        string         `gorm:"column:favicon_url;type:text" json:"favicon_url"`
	PrimaryColor      string         `gorm:"column:primary_color;type:varchar(20)" json:"primary_color"`
	SecondaryColor    string         `gorm:"column:secondary_color;type:varchar(20)" json:"secondary_color"`
	AccentColor       string         `gorm:"column:accent_color;type:varchar(20)" json:"accent_color"`
	FontFamily        string         `gorm:"column:font_family;type:varchar(100)" json:"font_family"`
	CustomCSS         string         `gorm:"column:custom_css;type:text" json:"custom_css"`
	SupportURL        string         `gorm:"column:support_url;type:text" json:"support_url"`
	PrivacyPolicyURL  string         `gorm:"column:privacy_policy_url;type:text" json:"privacy_policy_url"`
	TermsOfServiceURL string         `gorm:"column:terms_of_service_url;type:text" json:"terms_of_service_url"`
	Metadata          datatypes.JSON `gorm:"column:metadata;type:jsonb;default:'{}'" json:"metadata"`
	CreatedAt         time.Time      `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt         time.Time      `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`

	// Relationships
	Tenant *Tenant `gorm:"foreignKey:TenantID;references:TenantID"`
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
