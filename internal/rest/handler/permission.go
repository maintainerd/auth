package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/dto"
	"github.com/maintainerd/auth/internal/middleware"
	"github.com/maintainerd/auth/internal/model"
	"github.com/maintainerd/auth/internal/ptr"
	resp "github.com/maintainerd/auth/internal/rest/response"
	"github.com/maintainerd/auth/internal/service"
)

type PermissionHandler struct {
	permissionService service.PermissionService
}

func NewPermissionHandler(permissionService service.PermissionService) *PermissionHandler {
	return &PermissionHandler{permissionService}
}

// Get permissions with pagination
func (h *PermissionHandler) Get(w http.ResponseWriter, r *http.Request) {
	// Get tenant from context
	tenant, ok := r.Context().Value(middleware.TenantContextKey).(*model.Tenant)
	if !ok || tenant == nil {
		resp.Error(w, http.StatusUnauthorized, "Tenant not found in context")
		return
	}

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

	// Build request DTO
	reqParams := dto.PermissionFilterDTO{
		Name:        ptr.PtrOrNil(q.Get("name")),
		Description: ptr.PtrOrNil(q.Get("description")),
		APIUUID:     ptr.PtrOrNil(q.Get("api_id")),
		RoleUUID:    ptr.PtrOrNil(q.Get("role_id")),
		ClientUUID:  ptr.PtrOrNil(q.Get("client_id")),
		Status:      ptr.PtrOrNil(q.Get("status")),
		IsDefault:   isDefault,
		IsSystem:    isSystem,
		PaginationRequestDTO: dto.PaginationRequestDTO{
			Page:      page,
			Limit:     limit,
			SortBy:    q.Get("sort_by"),
			SortOrder: q.Get("sort_order"),
		},
	}

	if err := reqParams.Validate(); err != nil {
		resp.ValidationError(w, err)
		return
	}

	// Build permission filter
	permissionFilter := service.PermissionServiceGetFilter{
		TenantID:    tenant.TenantID,
		Name:        reqParams.Name,
		Description: reqParams.Description,
		APIUUID:     reqParams.APIUUID,
		RoleUUID:    reqParams.RoleUUID,
		ClientUUID:  reqParams.ClientUUID,
		Status:      reqParams.Status,
		IsDefault:   reqParams.IsDefault,
		IsSystem:    reqParams.IsSystem,
		Page:        reqParams.Page,
		Limit:       reqParams.Limit,
		SortBy:      reqParams.SortBy,
		SortOrder:   reqParams.SortOrder,
	}

	// Fetch permissions
	result, err := h.permissionService.Get(r.Context(), permissionFilter)
	if err != nil {
		resp.HandleServiceError(w, r, "Failed to fetch permissions", err)
		return
	}

	// Map permissions result to DTO
	rows := make([]dto.PermissionResponseDTO, len(result.Data))
	for i, r := range result.Data {
		rows[i] = toPermissionResponseDTO(r)
	}

	// Build response data
	response := dto.PaginatedResponseDTO[dto.PermissionResponseDTO]{
		Rows:       rows,
		Total:      result.Total,
		Page:       result.Page,
		Limit:      result.Limit,
		TotalPages: result.TotalPages,
	}

	resp.Success(w, response, "Permissions fetched successfully")
}

// Get permission by UUID
func (h *PermissionHandler) GetByUUID(w http.ResponseWriter, r *http.Request) {
	// Get tenant from context
	tenant, ok := r.Context().Value(middleware.TenantContextKey).(*model.Tenant)
	if !ok || tenant == nil {
		resp.Error(w, http.StatusUnauthorized, "Tenant not found in context")
		return
	}

	permissonUUID, err := uuid.Parse(chi.URLParam(r, "permission_uuid"))
	if err != nil {
		resp.Error(w, http.StatusBadRequest, "Invalid permission UUID")
		return
	}

	permission, err := h.permissionService.GetByUUID(r.Context(), permissonUUID, tenant.TenantID)
	if err != nil {
		resp.HandleServiceError(w, r, "Permission not found", err)
		return
	}

	dtoRes := toPermissionResponseDTO(*permission)

	resp.Success(w, dtoRes, "Permission fetched successfully")
}

// Create permission
func (h *PermissionHandler) Create(w http.ResponseWriter, r *http.Request) {
	// Get tenant from context
	tenant, ok := r.Context().Value(middleware.TenantContextKey).(*model.Tenant)
	if !ok || tenant == nil {
		resp.Error(w, http.StatusUnauthorized, "Tenant not found in context")
		return
	}

	var req dto.PermissionCreateRequestDTO
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		resp.Error(w, http.StatusBadRequest, "Invalid request")
		return
	}

	if err := req.Validate(); err != nil {
		resp.ValidationError(w, err)
		return
	}

	permission, err := h.permissionService.Create(r.Context(), tenant.TenantID, req.Name, req.Description, req.Status, false, req.APIUUID)
	if err != nil {
		resp.HandleServiceError(w, r, "Failed to create permission", err)
		return
	}

	dtoRes := toPermissionResponseDTO(*permission)

	resp.Created(w, dtoRes, "Permission created successfully")
}

// Update permission
func (h *PermissionHandler) Update(w http.ResponseWriter, r *http.Request) {
	// Get tenant from context
	tenant, ok := r.Context().Value(middleware.TenantContextKey).(*model.Tenant)
	if !ok || tenant == nil {
		resp.Error(w, http.StatusUnauthorized, "Tenant not found in context")
		return
	}

	permissionUUID, err := uuid.Parse(chi.URLParam(r, "permission_uuid"))
	if err != nil {
		resp.Error(w, http.StatusBadRequest, "Invalid permission UUID")
		return
	}

	var req dto.PermissionUpdateRequestDTO
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		resp.Error(w, http.StatusBadRequest, "Invalid request")
		return
	}

	if err := req.Validate(); err != nil {
		resp.ValidationError(w, err)
		return
	}

	permission, err := h.permissionService.Update(r.Context(), permissionUUID, tenant.TenantID, req.Name, req.Description, req.Status)
	if err != nil {
		resp.HandleServiceError(w, r, "Failed to update permission", err)
		return
	}

	dtoRes := toPermissionResponseDTO(*permission)

	resp.Success(w, dtoRes, "Permission updated successfully")
}

// Set permission status
func (h *PermissionHandler) SetStatus(w http.ResponseWriter, r *http.Request) {
	// Get tenant from context
	tenant, ok := r.Context().Value(middleware.TenantContextKey).(*model.Tenant)
	if !ok || tenant == nil {
		resp.Error(w, http.StatusUnauthorized, "Tenant not found in context")
		return
	}

	permissionUUID, err := uuid.Parse(chi.URLParam(r, "permission_uuid"))
	if err != nil {
		resp.Error(w, http.StatusBadRequest, "Invalid permission UUID")
		return
	}

	var req dto.PermissionStatusUpdateDTO
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		resp.Error(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if err := req.Validate(); err != nil {
		resp.ValidationError(w, err)
		return
	}

	permission, err := h.permissionService.SetStatus(r.Context(), permissionUUID, tenant.TenantID, req.Status)
	if err != nil {
		resp.HandleServiceError(w, r, "Failed to update permission status", err)
		return
	}

	dtoRes := toPermissionResponseDTO(*permission)
	resp.Success(w, dtoRes, "Permission status updated successfully")
}

// Delete permission
func (h *PermissionHandler) Delete(w http.ResponseWriter, r *http.Request) {
	// Get tenant from context
	tenant, ok := r.Context().Value(middleware.TenantContextKey).(*model.Tenant)
	if !ok || tenant == nil {
		resp.Error(w, http.StatusUnauthorized, "Tenant not found in context")
		return
	}

	permissionUUID, err := uuid.Parse(chi.URLParam(r, "permission_uuid"))
	if err != nil {
		resp.Error(w, http.StatusBadRequest, "Invalid permission UUID")
		return
	}

	permission, err := h.permissionService.DeleteByUUID(r.Context(), permissionUUID, tenant.TenantID)
	if err != nil {
		resp.HandleServiceError(w, r, "Failed to delete permission", err)
		return
	}

	dtoRes := toPermissionResponseDTO(*permission)

	resp.Success(w, dtoRes, "Permission deleted successfully")
}

// Convert permission result to DTO
func toPermissionResponseDTO(r service.PermissionServiceDataResult) dto.PermissionResponseDTO {
	result := dto.PermissionResponseDTO{
		PermissionUUID: r.PermissionUUID,
		Name:           r.Name,
		Description:    r.Description,
		Status:         r.Status,
		IsDefault:      r.IsDefault,
		IsSystem:       r.IsSystem,
		CreatedAt:      r.CreatedAt,
		UpdatedAt:      r.UpdatedAt,
	}

	if r.API != nil {
		result.API = &dto.APIResponseDTO{
			APIUUID:     r.API.APIUUID,
			Name:        r.API.Name,
			DisplayName: r.API.DisplayName,
			Description: r.API.Description,
			APIType:     r.API.APIType,
			Identifier:  r.API.Identifier,
			Status:      r.API.Status,
			CreatedAt:   r.API.CreatedAt,
			UpdatedAt:   r.API.UpdatedAt,
		}
	}

	return result
}
