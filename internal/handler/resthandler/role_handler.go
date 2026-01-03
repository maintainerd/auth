package resthandler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/dto"
	"github.com/maintainerd/auth/internal/middleware"
	"github.com/maintainerd/auth/internal/model"
	"github.com/maintainerd/auth/internal/service"
	"github.com/maintainerd/auth/internal/util"
)

// RoleHandler handles HTTP requests for role management.
// All endpoints are tenant-scoped - the middleware validates user access to the tenant
// and sets it in the request context. The service layer ensures roles belong to the tenant.
type RoleHandler struct {
	service service.RoleService
}

// NewRoleHandler creates a new instance of RoleHandler.
func NewRoleHandler(service service.RoleService) *RoleHandler {
	return &RoleHandler{service}
}

// Get retrieves all roles for the tenant with optional filtering and pagination.
// Tenant access is validated by middleware; this handler only needs to extract tenant from context.
// The service layer filters roles by tenant_id to ensure data isolation.
func (h *RoleHandler) Get(w http.ResponseWriter, r *http.Request) {
	// Tenant is already validated by middleware - just extract from context
	tenant, ok := r.Context().Value(middleware.TenantContextKey).(*model.Tenant)
	if !ok || tenant == nil {
		util.Error(w, http.StatusUnauthorized, "Tenant not found in context")
		return
	}

	// Extract query parameters
	q := r.URL.Query()

	// Parse pagination parameters
	page, _ := strconv.Atoi(q.Get("page"))
	limit, _ := strconv.Atoi(q.Get("limit"))

	// Parse boolean filters for default and system roles
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

	// Parse status filter
	var status *string
	if v := q.Get("status"); v != "" {
		status = &v
	}

	// Build filter DTO with all query parameters
	reqParams := dto.RoleFilterDto{
		Name:        util.PtrOrNil(q.Get("name")),
		Description: util.PtrOrNil(q.Get("description")),
		IsDefault:   isDefault,
		IsSystem:    isSystem,
		Status:      status,
		PaginationRequestDto: dto.PaginationRequestDto{
			Page:      page,
			Limit:     limit,
			SortBy:    q.Get("sort_by"),
			SortOrder: q.Get("sort_order"),
		},
	}

	// Validate filter parameters
	if err := reqParams.Validate(); err != nil {
		util.ValidationError(w, err)
		return
	}

	// Build service filter
	roleFilter := service.RoleServiceGetFilter{
		Name:        reqParams.Name,
		Description: reqParams.Description,
		IsDefault:   reqParams.IsDefault,
		IsSystem:    reqParams.IsSystem,
		Status:      reqParams.Status,
		TenantID:    tenant.TenantID,
		Page:        reqParams.Page,
		Limit:       reqParams.Limit,
		SortBy:      reqParams.SortBy,
		SortOrder:   reqParams.SortOrder,
	}

	// Fetch roles from service - service filters by tenant_id
	result, err := h.service.Get(roleFilter)
	if err != nil {
		util.Error(w, http.StatusInternalServerError, "Failed to fetch roles", err.Error())
		return
	}

	// Convert service results to DTOs
	rows := make([]dto.RoleResponseDto, len(result.Data))
	for i, r := range result.Data {
		rows[i] = toRoleResponseDto(r)
	}

	// Build paginated response
	response := dto.PaginatedResponseDto[dto.RoleResponseDto]{
		Rows:       rows,
		Total:      result.Total,
		Page:       result.Page,
		Limit:      result.Limit,
		TotalPages: result.TotalPages,
	}

	util.Success(w, response, "Roles fetched successfully")
}

// GetByUUID retrieves a specific role by UUID.
// Tenant access is validated by middleware.
// The service layer verifies the role belongs to the tenant.
func (h *RoleHandler) GetByUUID(w http.ResponseWriter, r *http.Request) {
	// Tenant is already validated by middleware - just extract from context
	tenant, ok := r.Context().Value(middleware.TenantContextKey).(*model.Tenant)
	if !ok || tenant == nil {
		util.Error(w, http.StatusUnauthorized, "Tenant not found in context")
		return
	}

	// Extract and validate role UUID from URL parameter
	roleUUID, err := uuid.Parse(chi.URLParam(r, "role_uuid"))
	if err != nil {
		util.Error(w, http.StatusBadRequest, "Invalid role UUID")
		return
	}

	// Fetch role - service validates it belongs to tenant
	role, err := h.service.GetByUUID(roleUUID, tenant.TenantID)
	if err != nil {
		util.Error(w, http.StatusNotFound, "Role not found")
		return
	}

	util.Success(w, toRoleResponseDto(*role), "Role fetched successfully")
}

// Create creates a new role for the tenant.
// Tenant access is validated by middleware.
// The role is automatically associated with the tenant from context.
func (h *RoleHandler) Create(w http.ResponseWriter, r *http.Request) {
	// Tenant is already validated by middleware - just extract from context
	tenant, ok := r.Context().Value(middleware.TenantContextKey).(*model.Tenant)
	if !ok || tenant == nil {
		util.Error(w, http.StatusUnauthorized, "Tenant not found in context")
		return
	}

	// Extract authenticated user from context (needed for audit tracking)
	user, ok := r.Context().Value(middleware.UserContextKey).(*model.User)
	if !ok || user == nil {
		util.Error(w, http.StatusUnauthorized, "User not found in context")
		return
	}

	// Decode request body
	var req dto.RoleCreateOrUpdateRequestDto
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		util.Error(w, http.StatusBadRequest, "Invalid request", err.Error())
		return
	}

	// Validate request data
	if err := req.Validate(); err != nil {
		util.ValidationError(w, err)
		return
	}

	// Create role associated with tenant
	role, err := h.service.Create(req.Name, req.Description, false, false, req.Status, tenant.TenantUUID.String(), user.UserUUID)
	if err != nil {
		util.Error(w, http.StatusInternalServerError, "Failed to create role", err.Error())
		return
	}

	util.Created(w, toRoleResponseDto(*role), "Role created successfully")
}

// Update updates an existing role.
// Tenant access is validated by middleware.
// The service layer verifies the role belongs to the tenant before updating.
func (h *RoleHandler) Update(w http.ResponseWriter, r *http.Request) {
	// Tenant is already validated by middleware - just extract from context
	tenant, ok := r.Context().Value(middleware.TenantContextKey).(*model.Tenant)
	if !ok || tenant == nil {
		util.Error(w, http.StatusUnauthorized, "Tenant not found in context")
		return
	}

	// Extract authenticated user from context (needed for audit tracking)
	user, ok := r.Context().Value(middleware.UserContextKey).(*model.User)
	if !ok || user == nil {
		util.Error(w, http.StatusUnauthorized, "User not found in context")
		return
	}

	// Extract and validate role UUID from URL parameter
	roleUUID, err := uuid.Parse(chi.URLParam(r, "role_uuid"))
	if err != nil {
		util.Error(w, http.StatusBadRequest, "Invalid role UUID")
		return
	}

	// Decode request body
	var req dto.RoleCreateOrUpdateRequestDto
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		util.Error(w, http.StatusBadRequest, "Invalid request", err.Error())
		return
	}

	// Validate request data
	if err := req.Validate(); err != nil {
		util.ValidationError(w, err)
		return
	}

	// Update role - service validates it belongs to tenant
	role, err := h.service.Update(roleUUID, tenant.TenantID, req.Name, req.Description, false, false, req.Status, user.UserUUID)
	if err != nil {
		util.Error(w, http.StatusInternalServerError, "Failed to update role", err.Error())
		return
	}

	util.Success(w, toRoleResponseDto(*role), "Role updated successfully")
}

// SetStatus updates the status of a role (active/inactive).
// Tenant access is validated by middleware.
// The service layer verifies the role belongs to the tenant before updating status.
func (h *RoleHandler) SetStatus(w http.ResponseWriter, r *http.Request) {
	// Tenant is already validated by middleware - just extract from context
	tenant, ok := r.Context().Value(middleware.TenantContextKey).(*model.Tenant)
	if !ok || tenant == nil {
		util.Error(w, http.StatusUnauthorized, "Tenant not found in context")
		return
	}

	// Extract authenticated user from context (needed for audit tracking)
	user, ok := r.Context().Value(middleware.UserContextKey).(*model.User)
	if !ok || user == nil {
		util.Error(w, http.StatusUnauthorized, "User not found in context")
		return
	}

	// Extract and validate role UUID from URL parameter
	roleUUID, err := uuid.Parse(chi.URLParam(r, "role_uuid"))
	if err != nil {
		util.Error(w, http.StatusBadRequest, "Invalid role UUID")
		return
	}

	// Decode request body
	var req struct {
		Status string `json:"status"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		util.Error(w, http.StatusBadRequest, "Invalid request", err.Error())
		return
	}

	// Validate status value
	if req.Status != "active" && req.Status != "inactive" {
		util.Error(w, http.StatusBadRequest, "Status must be 'active' or 'inactive'")
		return
	}

	// Update role status - service validates it belongs to tenant
	role, err := h.service.SetStatusByUUID(roleUUID, tenant.TenantID, req.Status, user.UserUUID)
	if err != nil {
		util.Error(w, http.StatusInternalServerError, "Failed to update role", err.Error())
		return
	}

	util.Success(w, toRoleResponseDto(*role), "Role updated successfully")
}

// Delete soft-deletes a role.
// Tenant access is validated by middleware.
// The service layer verifies the role belongs to the tenant before deletion.
func (h *RoleHandler) Delete(w http.ResponseWriter, r *http.Request) {
	// Tenant is already validated by middleware - just extract from context
	tenant, ok := r.Context().Value(middleware.TenantContextKey).(*model.Tenant)
	if !ok || tenant == nil {
		util.Error(w, http.StatusUnauthorized, "Tenant not found in context")
		return
	}

	// Extract authenticated user from context (needed for audit tracking)
	user, ok := r.Context().Value(middleware.UserContextKey).(*model.User)
	if !ok || user == nil {
		util.Error(w, http.StatusUnauthorized, "User not found in context")
		return
	}

	// Extract and validate role UUID from URL parameter
	roleUUID, err := uuid.Parse(chi.URLParam(r, "role_uuid"))
	if err != nil {
		util.Error(w, http.StatusBadRequest, "Invalid role UUID")
		return
	}

	// Delete role - service validates it belongs to tenant
	role, err := h.service.DeleteByUUID(roleUUID, tenant.TenantID, user.UserUUID)
	if err != nil {
		util.Error(w, http.StatusInternalServerError, "Failed to delete role", err.Error())
		return
	}

	util.Success(w, toRoleResponseDto(*role), "Role deleted successfully")
}

// GetPermissions retrieves all permissions assigned to a role with optional filtering and pagination.
// Tenant access is validated by middleware.
// The service layer verifies the role belongs to the tenant.
func (h *RoleHandler) GetPermissions(w http.ResponseWriter, r *http.Request) {
	// Tenant is already validated by middleware - just extract from context
	tenant, ok := r.Context().Value(middleware.TenantContextKey).(*model.Tenant)
	if !ok || tenant == nil {
		util.Error(w, http.StatusUnauthorized, "Tenant not found in context")
		return
	}

	// Extract and validate role UUID from URL parameter
	roleUUID, err := uuid.Parse(chi.URLParam(r, "role_uuid"))
	if err != nil {
		util.Error(w, http.StatusBadRequest, "Invalid role UUID")
		return
	}

	// Extract query parameters
	q := r.URL.Query()

	// Parse pagination parameters
	page, _ := strconv.Atoi(q.Get("page"))
	limit, _ := strconv.Atoi(q.Get("limit"))

	// Parse status filter
	var status *string
	if v := q.Get("status"); v != "" {
		status = &v
	}

	// Build pagination DTO
	reqParams := dto.PaginationRequestDto{
		Page:      page,
		Limit:     limit,
		SortBy:    q.Get("sort_by"),
		SortOrder: q.Get("sort_order"),
	}

	// Validate pagination parameters
	if err := reqParams.Validate(); err != nil {
		util.ValidationError(w, err)
		return
	}

	// Build service filter
	filter := service.RoleServiceGetPermissionsFilter{
		RoleUUID:  roleUUID,
		Status:    status,
		TenantID:  tenant.TenantID,
		Page:      reqParams.Page,
		Limit:     reqParams.Limit,
		SortBy:    reqParams.SortBy,
		SortOrder: reqParams.SortOrder,
	}

	// Fetch permissions from service - service validates role belongs to tenant
	result, err := h.service.GetRolePermissions(filter)
	if err != nil {
		util.Error(w, http.StatusInternalServerError, "Failed to fetch role permissions", err.Error())
		return
	}

	// Convert service results to DTOs
	rows := make([]dto.PermissionResponseDto, len(result.Data))
	for i, p := range result.Data {
		permDto := dto.PermissionResponseDto{
			PermissionUUID: p.PermissionUUID,
			Name:           p.Name,
			Description:    p.Description,
			Status:         p.Status,
			IsDefault:      p.IsDefault,
			IsSystem:       p.IsSystem,
			CreatedAt:      p.CreatedAt,
			UpdatedAt:      p.UpdatedAt,
		}
		// Include API details if available
		if p.API != nil {
			permDto.API = &dto.APIResponseDto{
				APIUUID:     p.API.APIUUID,
				Name:        p.API.Name,
				DisplayName: p.API.DisplayName,
				Description: p.API.Description,
				APIType:     p.API.APIType,
				Identifier:  p.API.Identifier,
				Status:      p.API.Status,
				IsDefault:   p.API.IsDefault,
				CreatedAt:   p.API.CreatedAt,
				UpdatedAt:   p.API.UpdatedAt,
			}
		}
		rows[i] = permDto
	}

	// Build paginated response
	response := dto.PaginatedResponseDto[dto.PermissionResponseDto]{
		Rows:       rows,
		Total:      result.Total,
		Page:       result.Page,
		Limit:      result.Limit,
		TotalPages: result.TotalPages,
	}

	util.Success(w, response, "Role permissions fetched successfully")
}

// AddPermissions adds one or more permissions to a role.
// Tenant access is validated by middleware.
// The service layer verifies the role belongs to the tenant before adding permissions.
func (h *RoleHandler) AddPermissions(w http.ResponseWriter, r *http.Request) {
	// Tenant is already validated by middleware - just extract from context
	tenant, ok := r.Context().Value(middleware.TenantContextKey).(*model.Tenant)
	if !ok || tenant == nil {
		util.Error(w, http.StatusUnauthorized, "Tenant not found in context")
		return
	}

	// Extract authenticated user from context (needed for audit tracking)
	user, ok := r.Context().Value(middleware.UserContextKey).(*model.User)
	if !ok || user == nil {
		util.Error(w, http.StatusUnauthorized, "User not found in context")
		return
	}

	// Extract and validate role UUID from URL parameter
	roleUUID, err := uuid.Parse(chi.URLParam(r, "role_uuid"))
	if err != nil {
		util.Error(w, http.StatusBadRequest, "Invalid role UUID")
		return
	}

	// Decode request body
	var req dto.RoleAddPermissionsRequestDto
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		util.Error(w, http.StatusBadRequest, "Invalid request", err.Error())
		return
	}

	// Validate request data
	if err := req.Validate(); err != nil {
		util.ValidationError(w, err)
		return
	}

	// Add permissions to role - service validates role belongs to tenant
	role, err := h.service.AddRolePermissions(roleUUID, tenant.TenantID, req.Permissions, user.UserUUID)
	if err != nil {
		util.Error(w, http.StatusInternalServerError, "Failed to add permissions to role", err.Error())
		return
	}

	util.Success(w, toRoleResponseDto(*role), "Permissions added to role successfully")
}

// RemovePermission removes a specific permission from a role.
// Tenant access is validated by middleware.
// The service layer verifies the role belongs to the tenant before removing the permission.
func (h *RoleHandler) RemovePermission(w http.ResponseWriter, r *http.Request) {
	// Tenant is already validated by middleware - just extract from context
	tenant, ok := r.Context().Value(middleware.TenantContextKey).(*model.Tenant)
	if !ok || tenant == nil {
		util.Error(w, http.StatusUnauthorized, "Tenant not found in context")
		return
	}

	// Extract authenticated user from context (needed for audit tracking)
	user, ok := r.Context().Value(middleware.UserContextKey).(*model.User)
	if !ok || user == nil {
		util.Error(w, http.StatusUnauthorized, "User not found in context")
		return
	}

	// Extract and validate role UUID from URL parameter
	roleUUID, err := uuid.Parse(chi.URLParam(r, "role_uuid"))
	if err != nil {
		util.Error(w, http.StatusBadRequest, "Invalid role UUID")
		return
	}

	// Extract and validate permission UUID from URL parameter
	permissionUUID, err := uuid.Parse(chi.URLParam(r, "permission_uuid"))
	if err != nil {
		util.Error(w, http.StatusBadRequest, "Invalid permission UUID")
		return
	}

	// Remove permission from role - service validates role belongs to tenant
	role, err := h.service.RemoveRolePermissions(roleUUID, tenant.TenantID, permissionUUID, user.UserUUID)
	if err != nil {
		util.Error(w, http.StatusInternalServerError, "Failed to remove permission from role", err.Error())
		return
	}

	util.Success(w, toRoleResponseDto(*role), "Permission removed from role successfully")
}

// Helper function for converting service data to response DTO

// toRoleResponseDto converts a service result to a role response DTO.
func toRoleResponseDto(r service.RoleServiceDataResult) dto.RoleResponseDto {
	result := dto.RoleResponseDto{
		RoleUUID:    r.RoleUUID,
		Name:        r.Name,
		Description: r.Description,
		IsDefault:   r.IsDefault,
		IsSystem:    r.IsSystem,
		Status:      r.Status,
		CreatedAt:   r.CreatedAt,
		UpdatedAt:   r.UpdatedAt,
	}
	// Map permissions if present
	if r.Permissions != nil {
		permissions := make([]dto.PermissionResponseDto, len(*r.Permissions))
		for i, permission := range *r.Permissions {
			permissions[i] = dto.PermissionResponseDto{
				PermissionUUID: permission.PermissionUUID,
				Name:           permission.Name,
				Description:    permission.Description,
				Status:         permission.Status,
				IsDefault:      permission.IsDefault,
				IsSystem:       permission.IsSystem,
				CreatedAt:      permission.CreatedAt,
				UpdatedAt:      permission.UpdatedAt,
			}
		}
		result.Permissions = &permissions
	}
	return result
}
