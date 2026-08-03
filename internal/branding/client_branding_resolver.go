package branding

import (
	"encoding/json"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// resolvedBranding mirrors the columns actually present on the branding table.
//
// There is deliberately no Layout field: the hosted-login layout is stored in
// the metadata JSONB under BrandingMetadataLayout, not in a column (see
// types.go and service_branding.go). This struct used to declare one and the
// queries below selected it, so every lookup failed with
// `column "layout" does not exist` (SQLSTATE 42703) — silently, because each
// caller treats an error as "no branding". The login form therefore rendered
// with EMPTY branding on every request while the DB row was perfectly fine.
type resolvedBranding struct {
	BrandingUUID      uuid.UUID
	CompanyName       string
	LogoURL           string
	FaviconURL        string
	SupportURL        string
	PrivacyPolicyURL  string
	TermsOfServiceURL string
	Metadata          datatypes.JSON
}

// brandingColumns is shared by all three lookups so they cannot drift apart
// again — the previous copies were identical strings maintained by hand.
const brandingColumns = "branding_uuid, company_name, logo_url, favicon_url, " +
	"support_url, privacy_policy_url, terms_of_service_url, metadata"

type ClientBrandingResponse struct {
	BrandingUUID      string          `json:"branding_id"`
	CompanyName       string          `json:"company_name"`
	LogoURL           string          `json:"logo_url"`
	FaviconURL        string          `json:"favicon_url"`
	Layout            string          `json:"layout"`
	SupportURL        string          `json:"support_url"`
	PrivacyPolicyURL  string          `json:"privacy_policy_url"`
	TermsOfServiceURL string          `json:"terms_of_service_url"`
	Metadata          json.RawMessage `json:"metadata"`
}

type ClientBrandingResolver struct {
	db *gorm.DB
}

func NewClientBrandingResolver(db *gorm.DB) *ClientBrandingResolver {
	return &ClientBrandingResolver{db: db}
}

func (r *ClientBrandingResolver) ResolveForClient(brandingID *int64, tenantID int64) *ClientBrandingResponse {
	if brandingID != nil {
		b := r.resolveByID(*brandingID, tenantID)
		if b != nil {
			return b
		}
	}
	if tenantID > 0 {
		b := r.resolveActiveForTenant(tenantID)
		if b != nil {
			return b
		}
	}
	return r.systemFallback()
}

func (r *ClientBrandingResolver) resolveByID(brandingID int64, tenantID int64) *ClientBrandingResponse {
	var b resolvedBranding
	err := r.db.Table("branding").
		Where("branding_id = ? AND tenant_id = ? AND deleted_at IS NULL", brandingID, tenantID).
		Select(brandingColumns).
		First(&b).Error
	if err != nil {
		return nil
	}
	return toBrandingResponse(&b)
}

func (r *ClientBrandingResolver) resolveActiveForTenant(tenantID int64) *ClientBrandingResponse {
	var b resolvedBranding
	err := r.db.Table("branding").
		Where("tenant_id = ? AND is_active = true AND deleted_at IS NULL", tenantID).
		Select(brandingColumns).
		First(&b).Error
	if err != nil {
		return nil
	}
	return toBrandingResponse(&b)
}

func (r *ClientBrandingResolver) systemFallback() *ClientBrandingResponse {
	var b resolvedBranding
	err := r.db.Table("branding").
		Where("is_system = true AND is_active = true AND deleted_at IS NULL").
		Select(brandingColumns).
		First(&b).Error
	if err != nil {
		return &ClientBrandingResponse{}
	}
	return toBrandingResponse(&b)
}

func toBrandingResponse(b *resolvedBranding) *ClientBrandingResponse {
	if b == nil {
		return nil
	}
	metadata, _ := b.Metadata.MarshalJSON()
	return &ClientBrandingResponse{
		BrandingUUID: b.BrandingUUID.String(),
		CompanyName:  b.CompanyName,
		LogoURL:      b.LogoURL,
		FaviconURL:   b.FaviconURL,
		// Read from metadata and default the same way service_branding.go does,
		// so the connections endpoint and the branding API agree on the layout.
		Layout:            brandingLayoutOrDefault(metadataString(b.Metadata, BrandingMetadataLayout)),
		SupportURL:        b.SupportURL,
		PrivacyPolicyURL:  b.PrivacyPolicyURL,
		TermsOfServiceURL: b.TermsOfServiceURL,
		Metadata:          metadata,
	}
}
