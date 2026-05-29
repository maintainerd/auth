package branding

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/platform/apperror"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// BrandingServiceDataResult is the service-layer representation of a branding
// record, decoupled from the persistence
type BrandingServiceDataResult struct {
	BrandingUUID      uuid.UUID
	CompanyName       string
	LogoURL           string
	FaviconURL        string
	PrimaryColor      string
	SecondaryColor    string
	AccentColor       string
	FontFamily        string
	CustomCSS         string
	SupportURL        string
	PrivacyPolicyURL  string
	TermsOfServiceURL string
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

// BrandingService defines business operations on tenant branding.
type BrandingService interface {
	Get(ctx context.Context, tenantID int64) (*BrandingServiceDataResult, error)
	Update(ctx context.Context, tenantID int64, companyName, logoURL, faviconURL, primaryColor, secondaryColor, accentColor, fontFamily, customCSS, supportURL, privacyPolicyURL, termsOfServiceURL string) (*BrandingServiceDataResult, error)
}

type brandingService struct {
	brandingRepo BrandingRepository
}

// NewBrandingService creates a new BrandingService.
func NewBrandingService(brandingRepo BrandingRepository) BrandingService {
	return &brandingService{brandingRepo: brandingRepo}
}

// toBrandingServiceDataResult converts a Branding into its service-layer
// representation.
func toBrandingServiceDataResult(b *Branding) *BrandingServiceDataResult {
	return &BrandingServiceDataResult{
		BrandingUUID:      b.BrandingUUID,
		CompanyName:       b.CompanyName,
		LogoURL:           b.LogoURL,
		FaviconURL:        b.FaviconURL,
		PrimaryColor:      b.PrimaryColor,
		SecondaryColor:    b.SecondaryColor,
		AccentColor:       b.AccentColor,
		FontFamily:        b.FontFamily,
		CustomCSS:         b.CustomCSS,
		SupportURL:        b.SupportURL,
		PrivacyPolicyURL:  b.PrivacyPolicyURL,
		TermsOfServiceURL: b.TermsOfServiceURL,
		CreatedAt:         b.CreatedAt,
		UpdatedAt:         b.UpdatedAt,
	}
}

// Get retrieves the branding for a tenant, auto-creating a default record if
// none exists.
func (s *brandingService) Get(ctx context.Context, tenantID int64) (*BrandingServiceDataResult, error) {
	_, span := otel.Tracer("service").Start(ctx, "branding.get")
	defer span.End()
	span.SetAttributes(attribute.Int64("tenant.id", tenantID))

	branding, err := s.getOrCreate(tenantID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "get branding failed")
		return nil, err
	}

	span.SetStatus(codes.Ok, "")
	return toBrandingServiceDataResult(branding), nil
}

// Update upserts the branding record for a tenant.
func (s *brandingService) Update(ctx context.Context, tenantID int64, companyName, logoURL, faviconURL, primaryColor, secondaryColor, accentColor, fontFamily, customCSS, supportURL, privacyPolicyURL, termsOfServiceURL string) (*BrandingServiceDataResult, error) {
	_, span := otel.Tracer("service").Start(ctx, "branding.update")
	defer span.End()
	span.SetAttributes(attribute.Int64("tenant.id", tenantID))

	branding, err := s.getOrCreate(tenantID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "get branding for update failed")
		return nil, err
	}

	branding.CompanyName = companyName
	branding.LogoURL = logoURL
	branding.FaviconURL = faviconURL
	branding.PrimaryColor = primaryColor
	branding.SecondaryColor = secondaryColor
	branding.AccentColor = accentColor
	branding.FontFamily = fontFamily
	branding.CustomCSS = customCSS
	branding.SupportURL = supportURL
	branding.PrivacyPolicyURL = privacyPolicyURL
	branding.TermsOfServiceURL = termsOfServiceURL

	updated, err := s.brandingRepo.CreateOrUpdate(branding)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "update branding failed")
		return nil, err
	}

	span.SetStatus(codes.Ok, "")
	return toBrandingServiceDataResult(updated), nil
}

// getOrCreate retrieves the branding record for the given tenant,
// automatically creating a default empty record if none exists.
func (s *brandingService) getOrCreate(tenantID int64) (*Branding, error) {
	branding, err := s.brandingRepo.FindByTenantID(tenantID)
	if err != nil {
		return nil, err
	}
	if branding != nil {
		return branding, nil
	}

	branding = &Branding{TenantID: tenantID}
	created, err := s.brandingRepo.Create(branding)
	if err != nil {
		return nil, apperror.NewInternal("failed to create default branding", err)
	}
	return created, nil
}

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
