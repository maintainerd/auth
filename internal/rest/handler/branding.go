package handler

import (
	"encoding/json"
	"net/http"

	"github.com/maintainerd/auth/internal/dto"
	"github.com/maintainerd/auth/internal/middleware"
	resp "github.com/maintainerd/auth/internal/rest/response"
	"github.com/maintainerd/auth/internal/service"
)

// BrandingHandler handles tenant branding configuration endpoints.
type BrandingHandler struct {
	brandingService service.BrandingService
}

// NewBrandingHandler creates a new BrandingHandler.
func NewBrandingHandler(brandingService service.BrandingService) *BrandingHandler {
	return &BrandingHandler{brandingService: brandingService}
}

// Get retrieves the branding for the authenticated tenant.
//
// GET /branding
func (h *BrandingHandler) Get(w http.ResponseWriter, r *http.Request) {
	tenant := middleware.AuthFromRequest(r).Tenant
	if tenant == nil {
		resp.Error(w, http.StatusUnauthorized, "Tenant not found in context")
		return
	}

	result, err := h.brandingService.Get(r.Context(), tenant.TenantID)
	if err != nil {
		resp.HandleServiceError(w, r, "Failed to get branding", err)
		return
	}

	resp.Success(w, toBrandingResponseDTO(result), "Branding retrieved successfully")
}

// Update upserts the branding for the authenticated tenant.
//
// PUT /branding
func (h *BrandingHandler) Update(w http.ResponseWriter, r *http.Request) {
	tenant := middleware.AuthFromRequest(r).Tenant
	if tenant == nil {
		resp.Error(w, http.StatusUnauthorized, "Tenant not found in context")
		return
	}

	var req dto.BrandingUpdateRequestDTO
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		resp.Error(w, http.StatusBadRequest, "Invalid request")
		return
	}

	if err := req.Validate(); err != nil {
		resp.ValidationError(w, err)
		return
	}

	result, err := h.brandingService.Update(
		r.Context(), tenant.TenantID,
		req.CompanyName, req.LogoURL, req.FaviconURL,
		req.PrimaryColor, req.SecondaryColor, req.AccentColor,
		req.FontFamily, req.CustomCSS,
		req.SupportURL, req.PrivacyPolicyURL, req.TermsOfServiceURL,
	)
	if err != nil {
		resp.HandleServiceError(w, r, "Failed to update branding", err)
		return
	}

	resp.Success(w, toBrandingResponseDTO(result), "Branding updated successfully")
}

func toBrandingResponseDTO(b *service.BrandingServiceDataResult) dto.BrandingResponseDTO {
	return dto.BrandingResponseDTO{
		BrandingID:        b.BrandingUUID.String(),
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
