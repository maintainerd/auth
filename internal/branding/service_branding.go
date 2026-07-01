package branding

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/platform/apperror"
	"github.com/maintainerd/auth/internal/shared"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"gorm.io/datatypes"
)

// BrandingServiceDataResult is the service-layer representation of a branding
// record, decoupled from the persistence layer.
type BrandingServiceDataResult struct {
	BrandingUUID      uuid.UUID
	Name              string
	IsSystem          bool
	IsActive          bool
	Layout            string
	CompanyName       string
	LogoURL           string
	FaviconURL        string
	SupportURL        string
	PrivacyPolicyURL  string
	TermsOfServiceURL string
	Metadata          datatypes.JSON
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

// BrandingService defines business operations on tenant branding.
type BrandingService interface {
	Get(ctx context.Context, tenantID int64) (*BrandingServiceDataResult, error)
	Update(ctx context.Context, tenantID int64, name, companyName, logoURL, faviconURL string, metadata datatypes.JSON, supportURL, privacyPolicyURL, termsOfServiceURL string) (*BrandingServiceDataResult, error)
	List(ctx context.Context, tenantID int64) ([]*BrandingServiceDataResult, error)
	Create(ctx context.Context, tenantID int64, name, layout, companyName, logoURL, faviconURL string, metadata datatypes.JSON, supportURL, privacyPolicyURL, termsOfServiceURL string) (*BrandingServiceDataResult, error)
	UpdateByUUID(ctx context.Context, brandingUUID uuid.UUID, tenantID int64, name, layout, companyName, logoURL, faviconURL string, metadata datatypes.JSON, supportURL, privacyPolicyURL, termsOfServiceURL string) (*BrandingServiceDataResult, error)
	Activate(ctx context.Context, brandingUUID uuid.UUID, tenantID int64) (*BrandingServiceDataResult, error)
	Delete(ctx context.Context, brandingUUID uuid.UUID, tenantID int64) error
	GetPublic(ctx context.Context, tenantID int64) (*BrandingServiceDataResult, error)
	GetLogoData(ctx context.Context, brandingUUID uuid.UUID) ([]byte, string, error)
	SetLogoData(ctx context.Context, brandingUUID uuid.UUID, data []byte, contentType string) error
}

type brandingService struct {
	brandingRepo BrandingRepository
}

func NewBrandingService(brandingRepo BrandingRepository) BrandingService {
	return &brandingService{brandingRepo: brandingRepo}
}

func toBrandingServiceDataResult(b *Branding) *BrandingServiceDataResult {
	return &BrandingServiceDataResult{
		BrandingUUID:      b.BrandingUUID,
		Name:              b.Name,
		IsSystem:          b.IsSystem,
		IsActive:          b.IsActive,
		Layout:            brandingLayoutOrDefault(b.Layout),
		CompanyName:       b.CompanyName,
		LogoURL:           b.LogoURL,
		FaviconURL:        b.FaviconURL,
		SupportURL:        b.SupportURL,
		PrivacyPolicyURL:  b.PrivacyPolicyURL,
		TermsOfServiceURL: b.TermsOfServiceURL,
		Metadata:          b.Metadata,
		CreatedAt:         b.CreatedAt,
		UpdatedAt:         b.UpdatedAt,
	}
}

func brandingLayoutOrDefault(layout string) string {
	if layout == "" {
		return shared.BrandingLayoutCentered
	}
	return layout
}

func validateBrandingLayout(layout string) error {
	switch layout {
	case shared.BrandingLayoutCentered, shared.BrandingLayoutFullPage, shared.BrandingLayoutSplit:
		return nil
	default:
		return apperror.NewValidation("layout must be centered, full_page, or split")
	}
}

// Get retrieves the active branding for a tenant, falling back to the system
// branding if no custom active record exists.
func (s *brandingService) Get(ctx context.Context, tenantID int64) (*BrandingServiceDataResult, error) {
	_, span := otel.Tracer("service").Start(ctx, "branding.get")
	defer span.End()
	span.SetAttributes(attribute.Int64("tenant.id", tenantID))

	b, err := s.brandingRepo.FindActive(tenantID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "get branding failed")
		return nil, err
	}

	span.SetStatus(codes.Ok, "")
	return toBrandingServiceDataResult(b), nil
}

// Update is used to upsert a custom (non-system) branding record. If no custom
// record exists, one is created. The system branding record cannot be modified
// through this method.
func (s *brandingService) Update(ctx context.Context, tenantID int64, name, companyName, logoURL, faviconURL string, metadata datatypes.JSON, supportURL, privacyPolicyURL, termsOfServiceURL string) (*BrandingServiceDataResult, error) {
	_, span := otel.Tracer("service").Start(ctx, "branding.update")
	defer span.End()
	span.SetAttributes(attribute.Int64("tenant.id", tenantID))

	branding, err := s.brandingRepo.FindByTenantID(tenantID)
	if err != nil {
		return nil, err
	}

	if branding != nil && branding.IsSystem {
		return nil, apperror.NewValidation("system branding cannot be modified — create a custom branding instead")
	}

	isNew := branding == nil
	if isNew {
		branding = &Branding{TenantID: tenantID, Layout: shared.BrandingLayoutCentered}
	} else if branding.Layout == "" {
		branding.Layout = shared.BrandingLayoutCentered
	}

	branding.Name = name
	branding.CompanyName = companyName
	branding.LogoURL = logoURL
	branding.FaviconURL = faviconURL
	if metadata != nil {
		branding.Metadata = metadata
	}
	branding.SupportURL = supportURL
	branding.PrivacyPolicyURL = privacyPolicyURL
	branding.TermsOfServiceURL = termsOfServiceURL

	var updated *Branding
	if isNew {
		updated, err = s.brandingRepo.Create(branding)
	} else {
		updated, err = s.brandingRepo.CreateOrUpdate(branding)
	}
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "update branding failed")
		return nil, err
	}

	span.SetStatus(codes.Ok, "")
	return toBrandingServiceDataResult(updated), nil
}

// Activate sets a specific branding record as the active one for the tenant.
// Deactivates all other non-system records for the tenant first, then activates
// the requested one. The system branding record cannot be activated/deactivated
// through this method (it auto-activates as fallback).
func (s *brandingService) Activate(ctx context.Context, brandingUUID uuid.UUID, tenantID int64) (*BrandingServiceDataResult, error) {
	_, span := otel.Tracer("service").Start(ctx, "branding.activate")
	defer span.End()

	b, err := s.brandingRepo.FindByUUID(brandingUUID)
	if err != nil || b == nil || b.TenantID != tenantID {
		return nil, apperror.NewNotFoundWithReason("branding not found")
	}

	// Any branding can be activated — including system themes (e.g. light/dark);
	// activating one deactivates the rest so exactly one is active.
	if err := s.brandingRepo.DeactivateAll(tenantID); err != nil {
		return nil, err
	}

	b.IsActive = true
	updated, err := s.brandingRepo.CreateOrUpdate(b)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "activate branding failed")
		return nil, err
	}

	span.SetStatus(codes.Ok, "")
	return toBrandingServiceDataResult(updated), nil
}

// Delete removes a non-system branding record. The system branding cannot be
// deleted.
func (s *brandingService) Delete(ctx context.Context, brandingUUID uuid.UUID, tenantID int64) error {
	_, span := otel.Tracer("service").Start(ctx, "branding.delete")
	defer span.End()

	b, err := s.brandingRepo.FindByUUID(brandingUUID)
	if err != nil || b == nil {
		return apperror.NewNotFoundWithReason("branding not found")
	}
	if b.IsSystem {
		return apperror.NewValidation("system branding cannot be deleted")
	}
	if b.TenantID != tenantID {
		return apperror.NewNotFoundWithReason("branding not found")
	}

	return s.brandingRepo.DeleteByUUID(brandingUUID)
}

// GetPublic returns the active branding for public (unauthenticated) consumption.
func (s *brandingService) GetPublic(ctx context.Context, tenantID int64) (*BrandingServiceDataResult, error) {
	return s.Get(ctx, tenantID)
}

func (s *brandingService) GetLogoData(ctx context.Context, brandingUUID uuid.UUID) ([]byte, string, error) {
	b, err := s.brandingRepo.FindByUUID(brandingUUID)
	if err != nil || b == nil {
		return nil, "", apperror.NewNotFound("branding not found")
	}
	return b.LogoData, b.LogoContentType, nil
}

func (s *brandingService) SetLogoData(ctx context.Context, brandingUUID uuid.UUID, data []byte, contentType string) error {
	b, err := s.brandingRepo.FindByUUID(brandingUUID)
	if err != nil || b == nil {
		return apperror.NewNotFound("branding not found")
	}

	allowedTypes := map[string]bool{"image/png": true, "image/jpeg": true, "image/webp": true}
	if !allowedTypes[contentType] {
		return apperror.NewValidation("logo must be PNG, JPEG, or WebP")
	}

	const maxSize = 256 * 1024
	if len(data) > maxSize {
		return apperror.NewValidation("logo must be under 256 KB")
	}

	b.LogoData = data
	b.LogoContentType = contentType
	b.LogoURL = fmt.Sprintf("/public/branding/%s/logo", brandingUUID.String())

	_, err = s.brandingRepo.CreateOrUpdate(b)
	return err
}

// List returns every branding record (themes) for a tenant.
func (s *brandingService) List(ctx context.Context, tenantID int64) ([]*BrandingServiceDataResult, error) {
	_, span := otel.Tracer("service").Start(ctx, "branding.list")
	defer span.End()
	span.SetAttributes(attribute.Int64("tenant.id", tenantID))

	rows, err := s.brandingRepo.FindAllByTenantID(tenantID)
	if err != nil {
		span.RecordError(err)
		return nil, err
	}
	out := make([]*BrandingServiceDataResult, 0, len(rows))
	for i := range rows {
		out = append(out, toBrandingServiceDataResult(&rows[i]))
	}
	return out, nil
}

// Create adds a new custom branding theme for a tenant (never active or system
// on creation — activate it explicitly).
func (s *brandingService) Create(ctx context.Context, tenantID int64, name, layout, companyName, logoURL, faviconURL string, metadata datatypes.JSON, supportURL, privacyPolicyURL, termsOfServiceURL string) (*BrandingServiceDataResult, error) {
	_, span := otel.Tracer("service").Start(ctx, "branding.create")
	defer span.End()
	span.SetAttributes(attribute.Int64("tenant.id", tenantID))

	layout = brandingLayoutOrDefault(layout)
	if err := validateBrandingLayout(layout); err != nil {
		return nil, err
	}

	b := &Branding{
		TenantID:          tenantID,
		Name:              name,
		Layout:            layout,
		CompanyName:       companyName,
		LogoURL:           logoURL,
		FaviconURL:        faviconURL,
		SupportURL:        supportURL,
		PrivacyPolicyURL:  privacyPolicyURL,
		TermsOfServiceURL: termsOfServiceURL,
	}
	if metadata != nil {
		b.Metadata = metadata
	}

	created, err := s.brandingRepo.Create(b)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "create branding failed")
		return nil, err
	}
	return toBrandingServiceDataResult(created), nil
}

// UpdateByUUID updates a specific branding theme. System themes can be edited
// (e.g. tweak colors) but never deleted. is_active/is_system are not changed
// here — activation is a separate action.
func (s *brandingService) UpdateByUUID(ctx context.Context, brandingUUID uuid.UUID, tenantID int64, name, layout, companyName, logoURL, faviconURL string, metadata datatypes.JSON, supportURL, privacyPolicyURL, termsOfServiceURL string) (*BrandingServiceDataResult, error) {
	_, span := otel.Tracer("service").Start(ctx, "branding.update_by_uuid")
	defer span.End()

	b, err := s.brandingRepo.FindByUUID(brandingUUID)
	if err != nil || b == nil || b.TenantID != tenantID {
		return nil, apperror.NewNotFoundWithReason("branding not found")
	}
	if layout != "" {
		if err := validateBrandingLayout(layout); err != nil {
			return nil, err
		}
		b.Layout = layout
	} else if b.Layout == "" {
		b.Layout = shared.BrandingLayoutCentered
	}

	b.Name = name
	b.CompanyName = companyName
	b.LogoURL = logoURL
	b.FaviconURL = faviconURL
	if metadata != nil {
		b.Metadata = metadata
	}
	b.SupportURL = supportURL
	b.PrivacyPolicyURL = privacyPolicyURL
	b.TermsOfServiceURL = termsOfServiceURL

	updated, err := s.brandingRepo.CreateOrUpdate(b)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "update branding failed")
		return nil, err
	}
	return toBrandingServiceDataResult(updated), nil
}
