package scim

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/maintainerd/maintainerd-auth/internal/platform/middleware"
	resp "github.com/maintainerd/maintainerd-auth/internal/platform/response"
)

type SCIMConfigurationHandler struct {
	svc SCIMConfigurationService
}

func NewSCIMConfigurationHandler(svc SCIMConfigurationService) *SCIMConfigurationHandler {
	return &SCIMConfigurationHandler{svc: svc}
}

func (h *SCIMConfigurationHandler) GetAll(w http.ResponseWriter, r *http.Request) {
	auth := middleware.AuthFromRequest(r)
	if auth == nil || auth.Tenant == nil {
		resp.Error(w, http.StatusUnauthorized, "Unauthorized")
		return
	}
	tenantID := int64(auth.Tenant.TenantID)

	var req SCIMConfigurationFilterDTO
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		req = SCIMConfigurationFilterDTO{
			Page:      1,
			Limit:     20,
			SortBy:    "created_at",
			SortOrder: "desc",
		}
	}

	result, err := h.svc.List(r.Context(), tenantID, SCIMConfigurationFilter{
		Search:    derefString(req.Search),
		IsActive:  req.IsActive,
		Page:      req.Page,
		Limit:     req.Limit,
		SortBy:    req.SortBy,
		SortOrder: req.SortOrder,
	})
	if err != nil {
		resp.HandleServiceError(w, r, "Failed to list SCIM configurations", err)
		return
	}

	dtos := make([]SCIMConfigurationResponseDTO, len(result.Data))
	for i, cfg := range result.Data {
		dtos[i] = toSCIMConfigResponseDTO(&cfg)
	}

	response := PaginatedResponseDTO[SCIMConfigurationResponseDTO]{
		Rows:       dtos,
		Total:      result.Total,
		Page:       result.Page,
		Limit:      result.Limit,
		TotalPages: result.TotalPages,
	}
	resp.Success(w, response, "SCIM configurations retrieved successfully")
}

func (h *SCIMConfigurationHandler) Get(w http.ResponseWriter, r *http.Request) {
	auth := middleware.AuthFromRequest(r)
	if auth == nil || auth.Tenant == nil {
		resp.Error(w, http.StatusUnauthorized, "Unauthorized")
		return
	}
	tenantID := int64(auth.Tenant.TenantID)

	scimUUID, err := uuid.Parse(chi.URLParam(r, "scim_configuration_uuid"))
	if err != nil {
		resp.Error(w, http.StatusBadRequest, "Invalid SCIM configuration UUID")
		return
	}

	result, err := h.svc.GetByUUID(r.Context(), scimUUID, tenantID)
	if err != nil {
		resp.HandleServiceError(w, r, "Failed to get SCIM configuration", err)
		return
	}

	resp.Success(w, toSCIMConfigResponseDTO(result), "SCIM configuration retrieved successfully")
}

func (h *SCIMConfigurationHandler) Create(w http.ResponseWriter, r *http.Request) {
	auth := middleware.AuthFromRequest(r)
	if auth == nil || auth.Tenant == nil {
		resp.Error(w, http.StatusUnauthorized, "Unauthorized")
		return
	}
	tenantID := int64(auth.Tenant.TenantID)

	var req SCIMConfigurationCreateDTO
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		resp.BadRequestBody(w)
		return
	}
	req.Sanitize()
	if err := req.Validate(); err != nil {
		resp.ValidationError(w, err)
		return
	}

	syncUsers := true
	if req.SyncUsers != nil {
		syncUsers = *req.SyncUsers
	}
	syncGroups := false
	if req.SyncGroups != nil {
		syncGroups = *req.SyncGroups
	}
	syncDirection := "inbound"
	if req.SyncDirection != nil {
		syncDirection = *req.SyncDirection
	}
	isActive := true
	if req.IsActive != nil {
		isActive = *req.IsActive
	}
	var attrMapping json.RawMessage
	if req.AttributeMapping != nil {
		attrMapping = *req.AttributeMapping
	}

	result, err := h.svc.Create(r.Context(), tenantID, SCIMConfigurationCreateInput{
		IdentityProviderID: req.IdentityProviderID,
		DisplayName:        req.DisplayName,
		BaseURL:            req.BaseURL,
		BearerToken:        req.BearerToken,
		SyncUsers:          syncUsers,
		SyncGroups:         syncGroups,
		SyncDirection:      syncDirection,
		AttributeMapping:   attrMapping,
		IsActive:           isActive,
	})
	if err != nil {
		resp.HandleServiceError(w, r, "Failed to create SCIM configuration", err)
		return
	}

	resp.Created(w, toSCIMConfigResponseDTO(result), "SCIM configuration created successfully")
}

func (h *SCIMConfigurationHandler) Update(w http.ResponseWriter, r *http.Request) {
	auth := middleware.AuthFromRequest(r)
	if auth == nil || auth.Tenant == nil {
		resp.Error(w, http.StatusUnauthorized, "Unauthorized")
		return
	}
	tenantID := int64(auth.Tenant.TenantID)

	scimUUID, err := uuid.Parse(chi.URLParam(r, "scim_configuration_uuid"))
	if err != nil {
		resp.Error(w, http.StatusBadRequest, "Invalid SCIM configuration UUID")
		return
	}

	var req SCIMConfigurationUpdateDTO
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		resp.BadRequestBody(w)
		return
	}
	req.Sanitize()
	if err := req.Validate(); err != nil {
		resp.ValidationError(w, err)
		return
	}

	result, err := h.svc.Update(r.Context(), scimUUID, tenantID, SCIMConfigurationUpdateInput{
		IdentityProviderID: req.IdentityProviderID,
		DisplayName:        req.DisplayName,
		BaseURL:            req.BaseURL,
		BearerToken:        req.BearerToken,
		SyncUsers:          req.SyncUsers,
		SyncGroups:         req.SyncGroups,
		SyncDirection:      req.SyncDirection,
		AttributeMapping:   req.AttributeMapping,
		IsActive:           req.IsActive,
	})
	if err != nil {
		resp.HandleServiceError(w, r, "Failed to update SCIM configuration", err)
		return
	}

	resp.Success(w, toSCIMConfigResponseDTO(result), "SCIM configuration updated successfully")
}

func (h *SCIMConfigurationHandler) Delete(w http.ResponseWriter, r *http.Request) {
	auth := middleware.AuthFromRequest(r)
	if auth == nil || auth.Tenant == nil {
		resp.Error(w, http.StatusUnauthorized, "Unauthorized")
		return
	}
	tenantID := int64(auth.Tenant.TenantID)

	scimUUID, err := uuid.Parse(chi.URLParam(r, "scim_configuration_uuid"))
	if err != nil {
		resp.Error(w, http.StatusBadRequest, "Invalid SCIM configuration UUID")
		return
	}

	if err := h.svc.Delete(r.Context(), scimUUID, tenantID); err != nil {
		resp.HandleServiceError(w, r, "Failed to delete SCIM configuration", err)
		return
	}

	resp.Success(w, nil, "SCIM configuration deleted successfully")
}

func toSCIMConfigResponseDTO(result *SCIMConfigurationServiceDataResult) SCIMConfigurationResponseDTO {
	dto := SCIMConfigurationResponseDTO{
		SCIMConfigurationUUID: result.SCIMConfigurationUUID,
		TenantID:              result.TenantID,
		IdentityProviderID:    result.IdentityProviderID,
		DisplayName:           result.DisplayName,
		BaseURL:               result.BaseURL,
		SyncUsers:             result.SyncUsers,
		SyncGroups:            result.SyncGroups,
		SyncDirection:         result.SyncDirection,
		AttributeMapping:      json.RawMessage(result.AttributeMapping),
		IsActive:              result.IsActive,
		LastSyncAt:            result.LastSyncAt,
		LastSyncStatus:        result.LastSyncStatus,
		LastSyncError:         result.LastSyncError,
		CreatedAt:             result.CreatedAt,
		UpdatedAt:             result.UpdatedAt,
	}
	return dto
}
