package branding

import (
	"encoding/json"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type resolvedBranding struct {
	BrandingUUID      uuid.UUID
	CompanyName       string
	LogoURL           string
	FaviconURL        string
	Layout            string
	SupportURL        string
	PrivacyPolicyURL  string
	TermsOfServiceURL string
	Metadata          datatypes.JSON
}

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
		Select("branding_uuid, company_name, logo_url, favicon_url, layout, support_url, privacy_policy_url, terms_of_service_url, metadata").
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
		Select("branding_uuid, company_name, logo_url, favicon_url, layout, support_url, privacy_policy_url, terms_of_service_url, metadata").
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
		Select("branding_uuid, company_name, logo_url, favicon_url, layout, support_url, privacy_policy_url, terms_of_service_url, metadata").
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
		BrandingUUID:      b.BrandingUUID.String(),
		CompanyName:       b.CompanyName,
		LogoURL:           b.LogoURL,
		FaviconURL:        b.FaviconURL,
		Layout:            b.Layout,
		SupportURL:        b.SupportURL,
		PrivacyPolicyURL:  b.PrivacyPolicyURL,
		TermsOfServiceURL: b.TermsOfServiceURL,
		Metadata:          metadata,
	}
}
