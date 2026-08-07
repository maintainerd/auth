package branding

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/maintainerd/maintainerd-auth/internal/platform/apperror"
	"github.com/maintainerd/maintainerd-auth/internal/shared"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"gorm.io/datatypes"
)

// BrandingServiceDataResult is the service-layer representation of a branding
// record, decoupled from the persistence layer.
type BrandingServiceDataResult struct {
	BrandingUUID          uuid.UUID
	Name                  string
	IsSystem              bool
	IsActive              bool
	Layout                string
	CompanyName           string
	LogoLabel             string
	LogoDetail            string
	ShowLogoLabel         bool
	IdentityLogoLabel     string
	IdentityShowLogoLabel bool
	LogoURL               string
	FaviconURL            string
	SupportURL            string
	PrivacyPolicyURL      string
	TermsOfServiceURL     string
	Metadata              datatypes.JSON
	CreatedAt             time.Time
	UpdatedAt             time.Time
}

// BrandingService defines business operations on tenant branding.
type BrandingService interface {
	Get(ctx context.Context, tenantID int64) (*BrandingServiceDataResult, error)
	Update(ctx context.Context, tenantID int64, name, companyName, logoURL, faviconURL string, metadata datatypes.JSON, supportURL, privacyPolicyURL, termsOfServiceURL string) (*BrandingServiceDataResult, error)
	List(ctx context.Context, tenantID int64) ([]*BrandingServiceDataResult, error)
	Create(ctx context.Context, tenantID int64, name, companyName, logoURL, faviconURL string, metadata datatypes.JSON, supportURL, privacyPolicyURL, termsOfServiceURL string) (*BrandingServiceDataResult, error)
	UpdateByUUID(ctx context.Context, brandingUUID uuid.UUID, tenantID int64, name, companyName, logoURL, faviconURL string, metadata datatypes.JSON, supportURL, privacyPolicyURL, termsOfServiceURL string) (*BrandingServiceDataResult, error)
	Activate(ctx context.Context, brandingUUID uuid.UUID, tenantID int64) (*BrandingServiceDataResult, error)
	RestoreSystem(ctx context.Context, brandingUUID uuid.UUID, tenantID int64) (*BrandingServiceDataResult, error)
	Delete(ctx context.Context, brandingUUID uuid.UUID, tenantID int64) error
	GetPublic(ctx context.Context, tenantID int64) (*BrandingServiceDataResult, error)
	GetPublicByID(ctx context.Context, tenantID, brandingID int64) (*BrandingServiceDataResult, error)
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
		BrandingUUID:          b.BrandingUUID,
		Name:                  b.Name,
		IsSystem:              b.IsSystem,
		IsActive:              b.IsActive,
		Layout:                brandingLayoutOrDefault(metadataString(b.Metadata, BrandingMetadataLayout)),
		CompanyName:           b.CompanyName,
		LogoLabel:             brandingLogoLabelOrDefault(metadataString(b.Metadata, BrandingMetadataLogoLabel), b.CompanyName, b.Name),
		LogoDetail:            metadataString(b.Metadata, BrandingMetadataLogoDetail),
		ShowLogoLabel:         metadataBool(b.Metadata, BrandingMetadataShowLogoLabel, true),
		IdentityLogoLabel:     metadataString(b.Metadata, BrandingMetadataIdentityLogoLabel),
		IdentityShowLogoLabel: metadataBool(b.Metadata, BrandingMetadataIdentityShowLogoLabel, true),
		LogoURL:               b.LogoURL,
		FaviconURL:            b.FaviconURL,
		SupportURL:            b.SupportURL,
		PrivacyPolicyURL:      b.PrivacyPolicyURL,
		TermsOfServiceURL:     b.TermsOfServiceURL,
		Metadata:              b.Metadata,
		CreatedAt:             b.CreatedAt,
		UpdatedAt:             b.UpdatedAt,
	}
}

// metadataString reads a string value from branding metadata.
func metadataString(metadata datatypes.JSON, key string) string {
	if len(metadata) == 0 {
		return ""
	}
	var m map[string]any
	if err := json.Unmarshal(metadata, &m); err != nil {
		return ""
	}
	value, _ := m[key].(string)
	return value
}

// metadataBool reads a boolean value from branding metadata, falling back to
// fallback when the key is absent or not a boolean.
func metadataBool(metadata datatypes.JSON, key string, fallback bool) bool {
	if len(metadata) == 0 {
		return fallback
	}
	var m map[string]any
	if err := json.Unmarshal(metadata, &m); err != nil {
		return fallback
	}
	value, ok := m[key].(bool)
	if !ok {
		return fallback
	}
	return value
}

// setMetadataString writes a string value into branding metadata, returning the
// updated payload.
func setMetadataString(metadata datatypes.JSON, key, value string) datatypes.JSON {
	m := map[string]any{}
	if len(metadata) > 0 {
		_ = json.Unmarshal(metadata, &m)
	}
	m[key] = value
	out, err := json.Marshal(m)
	if err != nil {
		return metadata
	}
	return datatypes.JSON(out)
}

func brandingLogoLabelOrDefault(label, companyName, name string) string {
	if label != "" {
		return label
	}
	if companyName != "" {
		return companyName
	}
	if name != "" {
		return name
	}
	return "Maintainerd-IAM"
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
		branding = &Branding{TenantID: tenantID}
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

// RestoreSystem resets a seeded system theme to its canonical defaults while
// preserving tenant ownership, system status, active state, and UUID.
func (s *brandingService) RestoreSystem(ctx context.Context, brandingUUID uuid.UUID, tenantID int64) (*BrandingServiceDataResult, error) {
	_, span := otel.Tracer("service").Start(ctx, "branding.restore_system")
	defer span.End()
	span.SetAttributes(attribute.Int64("tenant.id", tenantID))

	b, err := s.brandingRepo.FindByUUID(brandingUUID)
	if err != nil || b == nil || b.TenantID != tenantID {
		return nil, apperror.NewNotFoundWithReason("branding not found")
	}
	if !b.IsSystem {
		return nil, apperror.NewValidation("only system branding themes can be restored")
	}

	defaultTheme, ok := systemBrandingThemeDefaultByName(b.Name)
	if !ok {
		return nil, apperror.NewValidation("system branding theme does not have a seeded default")
	}

	b.CompanyName = "Maintainerd-Auth"
	b.LogoURL = ""
	b.LogoData = nil
	b.LogoContentType = ""
	b.FaviconURL = ""
	b.SupportURL = ""
	b.PrivacyPolicyURL = ""
	b.TermsOfServiceURL = ""
	b.Metadata = datatypes.JSON([]byte(defaultTheme.Metadata))

	updated, err := s.brandingRepo.CreateOrUpdate(b)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "restore branding failed")
		return nil, err
	}

	span.SetStatus(codes.Ok, "")
	return toBrandingServiceDataResult(updated), nil
}

func systemBrandingThemeDefaultByName(name string) (shared.SystemBrandingThemeDefault, bool) {
	normalizedName := strings.ToLower(strings.TrimSpace(name))
	for _, theme := range shared.SystemBrandingThemeDefaults() {
		if theme.Name == normalizedName {
			return theme, true
		}
	}
	return shared.SystemBrandingThemeDefault{}, false
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

// GetPublicByID returns a specific tenant-owned branding record for public
// consumption. It is used when a public client has an attached theme.
func (s *brandingService) GetPublicByID(ctx context.Context, tenantID, brandingID int64) (*BrandingServiceDataResult, error) {
	_, span := otel.Tracer("service").Start(ctx, "branding.get_public_by_id")
	defer span.End()
	span.SetAttributes(attribute.Int64("tenant.id", tenantID), attribute.Int64("branding.id", brandingID))

	// Public/theming read: deliberately excludes the logo bytes. The response
	// carries logo_url and the browser fetches the image separately, so pulling
	// 256 KB here would be read and discarded on every login page render.
	b, err := s.brandingRepo.FindPublicByID(brandingID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "get branding by id failed")
		return nil, err
	}
	if b == nil || b.TenantID != tenantID {
		return nil, apperror.NewNotFoundWithReason("branding not found")
	}

	span.SetStatus(codes.Ok, "")
	return toBrandingServiceDataResult(b), nil
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
// on creation — activate it explicitly). The hosted-login layout is read from
// metadata and defaults to centered when absent.
func (s *brandingService) Create(ctx context.Context, tenantID int64, name, companyName, logoURL, faviconURL string, metadata datatypes.JSON, supportURL, privacyPolicyURL, termsOfServiceURL string) (*BrandingServiceDataResult, error) {
	_, span := otel.Tracer("service").Start(ctx, "branding.create")
	defer span.End()
	span.SetAttributes(attribute.Int64("tenant.id", tenantID))

	layout := metadataString(metadata, BrandingMetadataLayout)
	if layout == "" {
		layout = shared.BrandingLayoutCentered
		metadata = setMetadataString(metadata, BrandingMetadataLayout, layout)
	}
	if err := validateBrandingLayout(layout); err != nil {
		return nil, err
	}

	b := &Branding{
		TenantID:          tenantID,
		Name:              name,
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
func (s *brandingService) UpdateByUUID(ctx context.Context, brandingUUID uuid.UUID, tenantID int64, name, companyName, logoURL, faviconURL string, metadata datatypes.JSON, supportURL, privacyPolicyURL, termsOfServiceURL string) (*BrandingServiceDataResult, error) {
	_, span := otel.Tracer("service").Start(ctx, "branding.update_by_uuid")
	defer span.End()

	b, err := s.brandingRepo.FindByUUID(brandingUUID)
	if err != nil || b == nil || b.TenantID != tenantID {
		return nil, apperror.NewNotFoundWithReason("branding not found")
	}

	if layout := metadataString(metadata, BrandingMetadataLayout); layout != "" {
		if err := validateBrandingLayout(layout); err != nil {
			return nil, err
		}
	}

	if !b.IsSystem {
		b.Name = name
	}
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
