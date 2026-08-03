package branding

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/maintainerd/maintainerd-auth/internal/platform/apperror"
	"github.com/maintainerd/maintainerd-auth/internal/platform/cache"
	"github.com/maintainerd/maintainerd-auth/internal/platform/middleware"
	resp "github.com/maintainerd/maintainerd-auth/internal/platform/response"
)

// BrandingHandler handles tenant branding configuration endpoints.
type BrandingHandler struct {
	brandingService BrandingService
	appCache        *cache.Cache
}

const brandingLogoCacheTTL = time.Hour

type cachedBrandingLogo struct {
	Data        []byte `json:"data"`
	ContentType string `json:"content_type"`
}

// NewBrandingHandler creates a new BrandingHandler.
func NewBrandingHandler(brandingService BrandingService, appCache ...*cache.Cache) *BrandingHandler {
	h := &BrandingHandler{brandingService: brandingService}
	if len(appCache) > 0 {
		h.appCache = appCache[0]
	}
	return h
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
		req.Name, req.Layout, req.CompanyName, req.LogoLabel, req.ShowLogoLabelOrDefault(), req.LogoURL, req.FaviconURL,
		req.Metadata,
		req.SupportURL, req.PrivacyPolicyURL, req.TermsOfServiceURL,
	)
	if err != nil {
		resp.HandleServiceError(w, r, "Failed to create branding", err)
		return
	}

	if req.LogoData != "" {
		if err := h.storeLogoUpload(r, result.BrandingUUID, req); err != nil {
			resp.HandleServiceError(w, r, "Failed to store logo", err)
			return
		}
		result.LogoURL = fmt.Sprintf("/public/branding/%s/logo", result.BrandingUUID)
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

	if cached := h.cachedLogo(r.Context(), brandingUUID); cached != nil {
		h.writeLogo(w, brandingUUID, cached.Data, cached.ContentType)
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

	h.cacheLogo(r.Context(), brandingUUID, data, contentType)
	h.writeLogo(w, brandingUUID, data, contentType)
}

func (h *BrandingHandler) writeLogo(w http.ResponseWriter, brandingUUID uuid.UUID, data []byte, contentType string) {
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Cache-Control", "public, max-age=3600")
	w.Header().Set("ETag", `"`+brandingUUID.String()+`"`)
	_, _ = w.Write(data)
}

func (h *BrandingHandler) storeLogoUpload(r *http.Request, brandingUUID uuid.UUID, req BrandingUpdateRequestDTO) error {
	logoBytes, err := base64.StdEncoding.DecodeString(req.LogoData)
	if err != nil {
		return apperror.NewValidation("Logo data must be base64-encoded")
	}
	if err := h.brandingService.SetLogoData(r.Context(), brandingUUID, logoBytes, req.LogoContentType); err != nil {
		return err
	}
	h.deleteLogoCache(r.Context(), brandingUUID)
	return nil
}

func brandingLogoCacheKey(brandingUUID uuid.UUID) string {
	return "branding:logo:" + brandingUUID.String()
}

func (h *BrandingHandler) cachedLogo(ctx context.Context, brandingUUID uuid.UUID) *cachedBrandingLogo {
	if h.appCache == nil {
		return nil
	}
	var cached cachedBrandingLogo
	if err := h.appCache.GetSession(ctx, brandingLogoCacheKey(brandingUUID), &cached); err != nil {
		return nil
	}
	if len(cached.Data) == 0 || cached.ContentType == "" {
		return nil
	}
	return &cached
}

func (h *BrandingHandler) cacheLogo(ctx context.Context, brandingUUID uuid.UUID, data []byte, contentType string) {
	if h.appCache == nil {
		return
	}
	_ = h.appCache.SetSession(ctx, brandingLogoCacheKey(brandingUUID), cachedBrandingLogo{
		Data:        data,
		ContentType: contentType,
	}, brandingLogoCacheTTL)
}

func (h *BrandingHandler) deleteLogoCache(ctx context.Context, brandingUUID uuid.UUID) {
	if h.appCache == nil {
		return
	}
	_ = h.appCache.DeleteSession(ctx, brandingLogoCacheKey(brandingUUID))
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
		req.Name, req.Layout, req.CompanyName, req.LogoLabel, req.ShowLogoLabelOrDefault(), req.LogoURL, req.FaviconURL,
		req.Metadata,
		req.SupportURL, req.PrivacyPolicyURL, req.TermsOfServiceURL,
	)
	if err != nil {
		resp.HandleServiceError(w, r, "Failed to update branding", err)
		return
	}

	h.deleteLogoCache(r.Context(), brandingUUID)
	if req.LogoData != "" {
		if err := h.storeLogoUpload(r, brandingUUID, req); err != nil {
			resp.HandleServiceError(w, r, "Failed to store logo", err)
			return
		}
		result.LogoURL = fmt.Sprintf("/public/branding/%s/logo", brandingUUID)
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

// RestoreSystem resets a system branding theme to its seeded defaults.
//
// PATCH /branding/{branding_uuid}/restore
func (h *BrandingHandler) RestoreSystem(w http.ResponseWriter, r *http.Request) {
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

	result, err := h.brandingService.RestoreSystem(r.Context(), brandingUUID, tenant.TenantID)
	if err != nil {
		resp.HandleServiceError(w, r, "Failed to restore branding", err)
		return
	}

	h.deleteLogoCache(r.Context(), brandingUUID)
	resp.Success(w, toBrandingResponseDTO(result), "Branding restored successfully")
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

	h.deleteLogoCache(r.Context(), brandingUUID)
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
		LogoLabel:         b.LogoLabel,
		ShowLogoLabel:     b.ShowLogoLabel,
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
