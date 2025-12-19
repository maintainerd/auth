package resthandler

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/dto"
	"github.com/maintainerd/auth/internal/middleware"
	"github.com/maintainerd/auth/internal/model"
	"github.com/maintainerd/auth/internal/service"
	"github.com/maintainerd/auth/internal/util"
)

type IdentityProviderHandler struct {
	idpService service.IdentityProviderService
}

func NewIdentityProviderHandler(idpService service.IdentityProviderService) *IdentityProviderHandler {
	return &IdentityProviderHandler{idpService}
}

// Get identity provider with pagination
func (h *IdentityProviderHandler) Get(w http.ResponseWriter, r *http.Request) {
	// Parse query parameters
	q := r.URL.Query()

	// Parse pagination
	page, _ := strconv.Atoi(q.Get("page"))
	limit, _ := strconv.Atoi(q.Get("limit"))

	// Parse bools safely
	var isDefault, isSystem *bool
	if v := q.Get("is_default"); v != "" {
		parsed, err := strconv.ParseBool(v)
		if err == nil {
			isDefault = &parsed
		}
	}
	if v := q.Get("is_system"); v != "" {
		parsed, err := strconv.ParseBool(v)
		if err == nil {
			isSystem = &parsed
		}
	}

	// Parse status (comma-separated values)
	var status []string
	if statusParam := q.Get("status"); statusParam != "" {
		status = strings.Split(strings.ReplaceAll(statusParam, " ", ""), ",")
	}

	// Parse provider (comma-separated values)
	var provider []string
	if providerParam := q.Get("provider"); providerParam != "" {
		provider = strings.Split(strings.ReplaceAll(providerParam, " ", ""), ",")
	}

	// Build request DTO
	reqParams := dto.IdentityProviderFilterDto{
		Name:         util.PtrOrNil(q.Get("name")),
		DisplayName:  util.PtrOrNil(q.Get("display_name")),
		Provider:     provider,
		ProviderType: util.PtrOrNil(q.Get("provider_type")),
		Identifier:   util.PtrOrNil(q.Get("identifier")),
		Status:       status,
		TenantUUID:   util.PtrOrNil(q.Get("tenant_id")),
		IsDefault:    isDefault,
		IsSystem:     isSystem,
		PaginationRequestDto: dto.PaginationRequestDto{
			Page:      page,
			Limit:     limit,
			SortBy:    q.Get("sort_by"),
			SortOrder: q.Get("sort_order"),
		},
	}

	if err := reqParams.Validate(); err != nil {
		util.ValidationError(w, err)
		return
	}

	// Build permission filter
	idpFilter := service.IdentityProviderServiceGetFilter{
		Name:         reqParams.Name,
		DisplayName:  reqParams.DisplayName,
		Provider:     reqParams.Provider,
		ProviderType: reqParams.ProviderType,
		Identifier:   reqParams.Identifier,
		TenantUUID:   reqParams.TenantUUID,
		Status:       reqParams.Status,
		IsDefault:    reqParams.IsDefault,
		IsSystem:     reqParams.IsSystem,
		Page:         reqParams.Page,
		Limit:        reqParams.Limit,
		SortBy:       reqParams.SortBy,
		SortOrder:    reqParams.SortOrder,
	}

	// Fetch permissions
	result, err := h.idpService.Get(idpFilter)
	if err != nil {
		util.Error(w, http.StatusInternalServerError, "Failed to fetch identity providers", err.Error())
		return
	}

	// Map results to DTO (list response without config and tenant)
	rows := make([]dto.IdentityProviderResponseDto, len(result.Data))
	for i, r := range result.Data {
		rows[i] = toIdpListResponseDto(r)
	}

	// Build response data
	response := dto.PaginatedResponseDto[dto.IdentityProviderResponseDto]{
		Rows:       rows,
		Total:      result.Total,
		Page:       result.Page,
		Limit:      result.Limit,
		TotalPages: result.TotalPages,
	}

	util.Success(w, response, "Identity providers fetched successfully")
}

// Get identity provider by UUID
func (h *IdentityProviderHandler) GetByUUID(w http.ResponseWriter, r *http.Request) {
	idpUUID, err := uuid.Parse(chi.URLParam(r, "identity_provider_uuid"))
	if err != nil {
		util.Error(w, http.StatusBadRequest, "Invalid identity provider UUID")
		return
	}

	idp, err := h.idpService.GetByUUID(idpUUID)
	if err != nil {
		util.Error(w, http.StatusNotFound, "Identity provider not found")
		return
	}

	dtoRes := toIdpDetailResponseDto(*idp)

	util.Success(w, dtoRes, "Identity provider fetched successfully")
}

// Create identity provider
func (h *IdentityProviderHandler) Create(w http.ResponseWriter, r *http.Request) {
	// Get authentication context
	user := r.Context().Value(middleware.UserContextKey).(*model.User)

	var req dto.IdentityProviderCreateRequestDto
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		util.Error(w, http.StatusBadRequest, "Invalid request", err.Error())
		return
	}

	if err := req.Validate(); err != nil {
		util.ValidationError(w, err)
		return
	}

	idp, err := h.idpService.Create(req.Name, req.DisplayName, req.Provider, req.ProviderType, req.Config, req.Status, req.TenantUUID, user.UserUUID)
	if err != nil {
		util.Error(w, http.StatusInternalServerError, "Failed to create permission", err.Error())
		return
	}

	dtoRes := toIdpDetailResponseDto(*idp)

	util.Created(w, dtoRes, "Identity provider created successfully")
}

// Update identity provider
func (h *IdentityProviderHandler) Update(w http.ResponseWriter, r *http.Request) {
	// Get authentication context
	user := r.Context().Value(middleware.UserContextKey).(*model.User)

	idpUUID, err := uuid.Parse(chi.URLParam(r, "identity_provider_uuid"))
	if err != nil {
		util.Error(w, http.StatusBadRequest, "Invalid identity provider UUID")
		return
	}

	var req dto.IdentityProviderUpdateRequestDto
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		util.Error(w, http.StatusBadRequest, "Invalid request", err.Error())
		return
	}

	if err := req.Validate(); err != nil {
		util.ValidationError(w, err)
		return
	}

	idp, err := h.idpService.Update(idpUUID, req.Name, req.DisplayName, req.Provider, req.ProviderType, req.Config, req.Status, user.UserUUID)
	if err != nil {
		util.Error(w, http.StatusInternalServerError, "Failed to update identity provider", err.Error())
		return
	}

	dtoRes := toIdpDetailResponseDto(*idp)

	util.Success(w, dtoRes, "Identity provider updated successfully")
}

// Set identity provider status
func (h *IdentityProviderHandler) SetStatus(w http.ResponseWriter, r *http.Request) {
	// Get authentication context
	user := r.Context().Value(middleware.UserContextKey).(*model.User)

	idpUUID, err := uuid.Parse(chi.URLParam(r, "identity_provider_uuid"))
	if err != nil {
		util.Error(w, http.StatusBadRequest, "Invalid identity provider UUID")
		return
	}

	// Parse and validate request body
	var req dto.IdentityProviderStatusUpdateDto
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		util.Error(w, http.StatusBadRequest, "Invalid request", err.Error())
		return
	}

	if err := req.Validate(); err != nil {
		util.ValidationError(w, err)
		return
	}

	idp, err := h.idpService.SetStatusByUUID(idpUUID, req.Status, user.UserUUID)
	if err != nil {
		util.Error(w, http.StatusInternalServerError, "Failed to update identity provider", err.Error())
		return
	}

	dtoRes := toIdpDetailResponseDto(*idp)

	util.Success(w, dtoRes, "Identity provider status updated successfully")
}

// Delete identity provider
func (h *IdentityProviderHandler) Delete(w http.ResponseWriter, r *http.Request) {
	// Get authentication context
	user := r.Context().Value(middleware.UserContextKey).(*model.User)

	idpUUID, err := uuid.Parse(chi.URLParam(r, "identity_provider_uuid"))
	if err != nil {
		util.Error(w, http.StatusBadRequest, "Invalid identity provider UUID")
		return
	}

	idp, err := h.idpService.DeleteByUUID(idpUUID, user.UserUUID)
	if err != nil {
		util.Error(w, http.StatusInternalServerError, "Failed to delete identity provider", err.Error())
		return
	}

	dtoRes := toIdpDetailResponseDto(*idp)

	util.Success(w, dtoRes, "Identity provider deleted successfully")
}

// Convert identity provider result to list DTO (without config and tenant)
func toIdpListResponseDto(r service.IdentityProviderServiceDataResult) dto.IdentityProviderResponseDto {
	return dto.IdentityProviderResponseDto{
		IdentityProviderUUID: r.IdentityProviderUUID,
		Name:                 r.Name,
		DisplayName:          r.DisplayName,
		Provider:             r.Provider,
		ProviderType:         r.ProviderType,
		Identifier:           r.Identifier,
		Status:               r.Status,
		IsDefault:            r.IsDefault,
		IsSystem:             r.IsSystem,
		CreatedAt:            r.CreatedAt,
		UpdatedAt:            r.UpdatedAt,
	}
}

// Convert identity provider result to detail DTO (with config and tenant)
func toIdpDetailResponseDto(r service.IdentityProviderServiceDataResult) dto.IdentityProviderDetailResponseDto {
	result := dto.IdentityProviderDetailResponseDto{
		IdentityProviderUUID: r.IdentityProviderUUID,
		Name:                 r.Name,
		DisplayName:          r.DisplayName,
		Provider:             r.Provider,
		ProviderType:         r.ProviderType,
		Identifier:           r.Identifier,
		Config:               r.Config,
		Status:               r.Status,
		IsDefault:            r.IsDefault,
		IsSystem:             r.IsSystem,
		CreatedAt:            r.CreatedAt,
		UpdatedAt:            r.UpdatedAt,
	}

	if r.Tenant != nil {
		result.Tenant = &dto.TenantResponseDto{
			TenantUUID:  r.Tenant.TenantUUID,
			Name:        r.Tenant.Name,
			Description: r.Tenant.Description,
			Identifier:  r.Tenant.Identifier,
			Status:      r.Tenant.Status,
			IsPublic:    r.Tenant.IsPublic,
			IsDefault:   r.Tenant.IsDefault,
			CreatedAt:   r.Tenant.CreatedAt,
			UpdatedAt:   r.Tenant.UpdatedAt,
		}
	}

	return result
}
