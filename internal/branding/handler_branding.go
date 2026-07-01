package branding

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/platform/middleware"
	resp "github.com/maintainerd/auth/internal/platform/response"
)

// BrandingHandler handles tenant branding configuration endpoints.
type BrandingHandler struct {
	brandingService BrandingService
}

// NewBrandingHandler creates a new BrandingHandler.
func NewBrandingHandler(brandingService BrandingService) *BrandingHandler {
	return &BrandingHandler{brandingService: brandingService}
}

// List returns all branding themes for the authenticated tenant (the active one
// is flagged via is_active).
//
// GET /branding
func (h *BrandingHandler) List(w http.ResponseWriter, r *http.Request) {
	tenant := middleware.AuthFromRequest(r).Tenant
	if tenant == nil {
		resp.Error(w, http.StatusUnauthorized, "Tenant not found in context")
		return
	}

	results, err := h.brandingService.List(r.Context(), tenant.TenantID)
	if err != nil {
		resp.HandleServiceError(w, r, "Failed to list branding", err)
		return
	}

	rows := make([]BrandingResponseDTO, 0, len(results))
	for _, b := range results {
		rows = append(rows, toBrandingResponseDTO(b))
	}
	resp.Success(w, rows, "Branding retrieved successfully")
}

// Create adds a new custom branding theme.
//
// POST /branding
func (h *BrandingHandler) Create(w http.ResponseWriter, r *http.Request) {
	tenant := middleware.AuthFromRequest(r).Tenant
	if tenant == nil {
		resp.Error(w, http.StatusUnauthorized, "Tenant not found in context")
		return
	}

	var req BrandingUpdateRequestDTO
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		resp.BadRequestBody(w)
		return
	}
	if err := req.Validate(); err != nil {
		resp.ValidationError(w, err)
		return
	}

	result, err := h.brandingService.Create(
		r.Context(), tenant.TenantID,
		req.Name, req.Layout, req.CompanyName, req.LogoURL, req.FaviconURL,
		req.Metadata,
		req.SupportURL, req.PrivacyPolicyURL, req.TermsOfServiceURL,
	)
	if err != nil {
		resp.HandleServiceError(w, r, "Failed to create branding", err)
		return
	}

	if req.LogoData != "" && req.LogoContentType != "" {
		logoBytes, err := base64.StdEncoding.DecodeString(req.LogoData)
		if err == nil {
			brandingUUID := result.BrandingUUID
			if err := h.brandingService.SetLogoData(r.Context(), brandingUUID, logoBytes, req.LogoContentType); err == nil {
				result.LogoURL = fmt.Sprintf("/public/branding/%s/logo", result.BrandingUUID)
			}
		}
	}

	resp.Success(w, toBrandingResponseDTO(result), "Branding created successfully")
}

// GetPublic returns the active branding for unauthenticated access.
//
// GET /public/branding?tenant_id=<id>&client_id=<identifier>
func (h *BrandingHandler) GetPublic(w http.ResponseWriter, r *http.Request) {
	tenantID := int64(0)
	if v := r.URL.Query().Get("tenant_id"); v != "" {
		if tid, err := strconv.ParseInt(v, 10, 64); err == nil {
			tenantID = tid
		}
	}
	result, err := h.brandingService.GetPublic(r.Context(), tenantID)
	if err != nil {
		resp.HandleServiceError(w, r, "Failed to get public branding", err)
		return
	}
	resp.Success(w, toBrandingResponseDTO(result), "Branding retrieved successfully")
}

// ServeLogo streams the stored logo bytes for a branding record.
//
// GET /public/branding/{branding_id}/logo
func (h *BrandingHandler) ServeLogo(w http.ResponseWriter, r *http.Request) {
	brandingUUID, err := uuid.Parse(chi.URLParam(r, "branding_id"))
	if err != nil {
		resp.Error(w, http.StatusBadRequest, "Invalid branding UUID")
		return
	}

	data, contentType, err := h.brandingService.GetLogoData(r.Context(), brandingUUID)
	if err != nil {
		resp.HandleServiceError(w, r, "Logo not found", err)
		return
	}

	if len(data) == 0 {
		resp.Error(w, http.StatusNotFound, "No logo stored for this branding")
		return
	}

	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Cache-Control", "public, max-age=3600")
	w.Header().Set("ETag", `"`+brandingUUID.String()+`"`)
	w.Write(data)
}

// Update modifies a specific branding theme.
//
// PUT /branding/{branding_uuid}
func (h *BrandingHandler) Update(w http.ResponseWriter, r *http.Request) {
	tenant := middleware.AuthFromRequest(r).Tenant
	if tenant == nil {
		resp.Error(w, http.StatusUnauthorized, "Tenant not found in context")
		return
	}

	brandingUUID, err := uuid.Parse(chi.URLParam(r, "branding_uuid"))
	if err != nil {
		resp.Error(w, http.StatusBadRequest, "Invalid branding UUID")
		return
	}

	var req BrandingUpdateRequestDTO
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		resp.BadRequestBody(w)
		return
	}

	if err := req.Validate(); err != nil {
		resp.ValidationError(w, err)
		return
	}

	result, err := h.brandingService.UpdateByUUID(
		r.Context(), brandingUUID, tenant.TenantID,
		req.Name, req.Layout, req.CompanyName, req.LogoURL, req.FaviconURL,
		req.Metadata,
		req.SupportURL, req.PrivacyPolicyURL, req.TermsOfServiceURL,
	)
	if err != nil {
		resp.HandleServiceError(w, r, "Failed to update branding", err)
		return
	}

	resp.Success(w, toBrandingResponseDTO(result), "Branding updated successfully")
}

// Activate sets the given branding as the active one for the tenant.
//
// PATCH /branding/{branding_uuid}/activate
func (h *BrandingHandler) Activate(w http.ResponseWriter, r *http.Request) {
	tenant := middleware.AuthFromRequest(r).Tenant
	if tenant == nil {
		resp.Error(w, http.StatusUnauthorized, "Tenant not found in context")
		return
	}

	brandingUUID, err := uuid.Parse(chi.URLParam(r, "branding_uuid"))
	if err != nil {
		resp.Error(w, http.StatusBadRequest, "Invalid branding UUID")
		return
	}

	result, err := h.brandingService.Activate(r.Context(), brandingUUID, tenant.TenantID)
	if err != nil {
		resp.HandleServiceError(w, r, "Failed to activate branding", err)
		return
	}

	resp.Success(w, toBrandingResponseDTO(result), "Branding activated successfully")
}

// Delete removes a non-system branding record.
//
// DELETE /branding/{branding_uuid}
func (h *BrandingHandler) Delete(w http.ResponseWriter, r *http.Request) {
	tenant := middleware.AuthFromRequest(r).Tenant
	if tenant == nil {
		resp.Error(w, http.StatusUnauthorized, "Tenant not found in context")
		return
	}

	brandingUUID, err := uuid.Parse(chi.URLParam(r, "branding_uuid"))
	if err != nil {
		resp.Error(w, http.StatusBadRequest, "Invalid branding UUID")
		return
	}

	if err := h.brandingService.Delete(r.Context(), brandingUUID, tenant.TenantID); err != nil {
		resp.HandleServiceError(w, r, "Failed to delete branding", err)
		return
	}

	resp.Success(w, nil, "Branding deleted successfully")
}

func toBrandingResponseDTO(b *BrandingServiceDataResult) BrandingResponseDTO {
	return BrandingResponseDTO{
		BrandingID:        b.BrandingUUID.String(),
		Name:              b.Name,
		IsSystem:          b.IsSystem,
		IsActive:          b.IsActive,
		Layout:            b.Layout,
		CompanyName:       b.CompanyName,
		LogoURL:           b.LogoURL,
		FaviconURL:        b.FaviconURL,
		Metadata:          b.Metadata,
		SupportURL:        b.SupportURL,
		PrivacyPolicyURL:  b.PrivacyPolicyURL,
		TermsOfServiceURL: b.TermsOfServiceURL,
		CreatedAt:         b.CreatedAt,
		UpdatedAt:         b.UpdatedAt,
	}
}
