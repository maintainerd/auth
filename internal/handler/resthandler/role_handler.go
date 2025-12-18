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

type RoleHandler struct {
	service service.RoleService
}

func NewRoleHandler(service service.RoleService) *RoleHandler {
	return &RoleHandler{service}
}

// GetAll roles with pagination
func (h *RoleHandler) Get(w http.ResponseWriter, r *http.Request) {
	// Get authentication context
	user := r.Context().Value(middleware.UserContextKey).(*model.User)

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
	var status *string
	if v := q.Get("status"); v != "" {
		status = &v
	}

	// Build request DTO (for validation)
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

	if err := reqParams.Validate(); err != nil {
		util.ValidationError(w, err)
		return
	}

	// Get role filters
	roleFilter := service.RoleServiceGetFilter{
		Name:        reqParams.Name,
		Description: reqParams.Description,
		IsDefault:   reqParams.IsDefault,
		IsSystem:    reqParams.IsSystem,
		Status:      reqParams.Status,
		TenantID:    user.TenantID,
		Page:        reqParams.Page,
		Limit:       reqParams.Limit,
		SortBy:      reqParams.SortBy,
		SortOrder:   reqParams.SortOrder,
	}

	// Fetch roles
	result, err := h.service.Get(roleFilter)
	if err != nil {
		util.Error(w, http.StatusInternalServerError, "Failed to fetch roles", err.Error())
		return
	}

	// Map service result to dto
	rows := make([]dto.RoleResponseDto, len(result.Data))
	for i, r := range result.Data {
		rows[i] = toRoleResponseDto(r)
	}

	// Build response data
	response := dto.PaginatedResponseDto[dto.RoleResponseDto]{
		Rows:       rows,
		Total:      result.Total,
		Page:       result.Page,
		Limit:      result.Limit,
		TotalPages: result.TotalPages,
	}

	util.Success(w, response, "Roles fetched successfully")
}

// Get role by UUID
func (h *RoleHandler) GetByUUID(w http.ResponseWriter, r *http.Request) {
	roleUUID, err := uuid.Parse(chi.URLParam(r, "role_uuid"))
	if err != nil {
		util.Error(w, http.StatusBadRequest, "Invalid role UUID")
		return
	}

	// Fetch role
	role, err := h.service.GetByUUID(roleUUID)
	if err != nil {
		util.Error(w, http.StatusNotFound, "Role not found")
		return
	}

	// Build response data
	dtoRes := toRoleResponseDto(*role)

	util.Success(w, dtoRes, "Role fetched successfully")
}

// Create role
func (h *RoleHandler) Create(w http.ResponseWriter, r *http.Request) {
	// Get authentication context
	user := r.Context().Value(middleware.UserContextKey).(*model.User)

	// Validate request body
	var req dto.RoleCreateOrUpdateRequestDto
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		util.Error(w, http.StatusBadRequest, "Invalid request", err.Error())
		return
	}

	if err := req.Validate(); err != nil {
		util.ValidationError(w, err)
		return
	}

	// Create role
	role, err := h.service.Create(req.Name, req.Description, false, false, req.Status, user.Tenant.TenantUUID.String(), user.UserUUID)
	if err != nil {
		util.Error(w, http.StatusInternalServerError, "Failed to create role", err.Error())
		return
	}

	// Build response data
	dtoRes := toRoleResponseDto(*role)

	util.Created(w, dtoRes, "Role created successfully")
}

// Update role
func (h *RoleHandler) Update(w http.ResponseWriter, r *http.Request) {
	// Get authentication context
	user := r.Context().Value(middleware.UserContextKey).(*model.User)

	// Validate role_uuid
	roleUUID, err := uuid.Parse(chi.URLParam(r, "role_uuid"))
	if err != nil {
		util.Error(w, http.StatusBadRequest, "Invalid role UUID")
		return
	}

	// Validate request body
	var req dto.RoleCreateOrUpdateRequestDto
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		util.Error(w, http.StatusBadRequest, "Invalid request", err.Error())
		return
	}

	if err := req.Validate(); err != nil {
		util.ValidationError(w, err)
		return
	}

	// Update role
	role, err := h.service.Update(roleUUID, req.Name, req.Description, false, false, req.Status, user.UserUUID)
	if err != nil {
		util.Error(w, http.StatusInternalServerError, "Failed to update role", err.Error())
		return
	}

	// Build response data
	dtoRes := toRoleResponseDto(*role)

	util.Success(w, dtoRes, "Role updated successfully")
}

// Set role status
func (h *RoleHandler) SetStatus(w http.ResponseWriter, r *http.Request) {
	// Get authentication context
	user := r.Context().Value(middleware.UserContextKey).(*model.User)

	// Validate role_uuid
	roleUUID, err := uuid.Parse(chi.URLParam(r, "role_uuid"))
	if err != nil {
		util.Error(w, http.StatusBadRequest, "Invalid role UUID")
		return
	}

	// Validate request body
	var req struct {
		Status string `json:"status"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		util.Error(w, http.StatusBadRequest, "Invalid request", err.Error())
		return
	}
	if req.Status != "active" && req.Status != "inactive" {
		util.Error(w, http.StatusBadRequest, "Status must be 'active' or 'inactive'")
		return
	}

	// Update role
	role, err := h.service.SetStatusByUUID(roleUUID, req.Status, user.UserUUID)
	if err != nil {
		util.Error(w, http.StatusInternalServerError, "Failed to update role", err.Error())
		return
	}

	// Build response data
	dtoRes := toRoleResponseDto(*role)

	util.Success(w, dtoRes, "Role updated successfully")
}

// Delete role
func (h *RoleHandler) Delete(w http.ResponseWriter, r *http.Request) {
	// Get authentication context
	user := r.Context().Value(middleware.UserContextKey).(*model.User)

	roleUUID, err := uuid.Parse(chi.URLParam(r, "role_uuid"))
	if err != nil {
		util.Error(w, http.StatusBadRequest, "Invalid role UUID")
		return
	}

	// Delete role
	role, err := h.service.DeleteByUUID(roleUUID, user.UserUUID)
	if err != nil {
		util.Error(w, http.StatusInternalServerError, "Failed to delete role", err.Error())
		return
	}

	// Build response data
	dtoRes := toRoleResponseDto(*role)

	util.Success(w, dtoRes, "Role deleted successfully")
}

// Get permissions for a role
func (h *RoleHandler) GetPermissions(w http.ResponseWriter, r *http.Request) {
	// Get authentication context
	user := r.Context().Value(middleware.UserContextKey).(*model.User)

	// Validate role_uuid
	roleUUID, err := uuid.Parse(chi.URLParam(r, "role_uuid"))
	if err != nil {
		util.Error(w, http.StatusBadRequest, "Invalid role UUID")
		return
	}

	// Parse query parameters
	q := r.URL.Query()

	// Parse pagination
	page, _ := strconv.Atoi(q.Get("page"))
	limit, _ := strconv.Atoi(q.Get("limit"))

	// Parse filters
	var status *string
	if v := q.Get("status"); v != "" {
		status = &v
	}

	// Build request DTO
	reqParams := dto.PaginationRequestDto{
		Page:      page,
		Limit:     limit,
		SortBy:    q.Get("sort_by"),
		SortOrder: q.Get("sort_order"),
	}

	if err := reqParams.Validate(); err != nil {
		util.ValidationError(w, err)
		return
	}

	// Build service filter
	filter := service.RoleServiceGetPermissionsFilter{
		RoleUUID:  roleUUID,
		Status:    status,
		TenantID:  user.TenantID,
		Page:      reqParams.Page,
		Limit:     reqParams.Limit,
		SortBy:    reqParams.SortBy,
		SortOrder: reqParams.SortOrder,
	}

	// Fetch permissions
	result, err := h.service.GetRolePermissions(filter)
	if err != nil {
		util.Error(w, http.StatusInternalServerError, "Failed to fetch role permissions", err.Error())
		return
	}

	// Map service result to dto
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
		// Include API if available
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

	// Build response data
	response := dto.PaginatedResponseDto[dto.PermissionResponseDto]{
		Rows:       rows,
		Total:      result.Total,
		Page:       result.Page,
		Limit:      result.Limit,
		TotalPages: result.TotalPages,
	}

	util.Success(w, response, "Role permissions fetched successfully")
}

// Add permissions to role
func (h *RoleHandler) AddPermissions(w http.ResponseWriter, r *http.Request) {
	// Get authentication context
	user := r.Context().Value(middleware.UserContextKey).(*model.User)

	// Validate role_uuid
	roleUUID, err := uuid.Parse(chi.URLParam(r, "role_uuid"))
	if err != nil {
		util.Error(w, http.StatusBadRequest, "Invalid role UUID")
		return
	}

	// Validate request body
	var req dto.RoleAddPermissionsRequestDto
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		util.Error(w, http.StatusBadRequest, "Invalid request", err.Error())
		return
	}

	if err := req.Validate(); err != nil {
		util.ValidationError(w, err)
		return
	}

	// Add permissions to role
	role, err := h.service.AddRolePermissions(roleUUID, req.Permissions, user.UserUUID)
	if err != nil {
		util.Error(w, http.StatusInternalServerError, "Failed to add permissions to role", err.Error())
		return
	}

	// Build response data
	dtoRes := toRoleResponseDto(*role)

	util.Success(w, dtoRes, "Permissions added to role successfully")
}

// Remove permission from role
func (h *RoleHandler) RemovePermission(w http.ResponseWriter, r *http.Request) {
	// Get authentication context
	user := r.Context().Value(middleware.UserContextKey).(*model.User)

	// Validate role_uuid
	roleUUID, err := uuid.Parse(chi.URLParam(r, "role_uuid"))
	if err != nil {
		util.Error(w, http.StatusBadRequest, "Invalid role UUID")
		return
	}

	// Validate permission_uuid
	permissionUUID, err := uuid.Parse(chi.URLParam(r, "permission_uuid"))
	if err != nil {
		util.Error(w, http.StatusBadRequest, "Invalid permission UUID")
		return
	}

	// Remove permission from role
	role, err := h.service.RemoveRolePermissions(roleUUID, permissionUUID, user.UserUUID)
	if err != nil {
		util.Error(w, http.StatusInternalServerError, "Failed to remove permission from role", err.Error())
		return
	}

	// Build response data
	dtoRes := toRoleResponseDto(*role)

	util.Success(w, dtoRes, "Permission removed from role successfully")
}

// Convert service result to dto
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
	// Map Permissions if present
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
