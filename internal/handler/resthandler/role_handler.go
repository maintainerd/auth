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
	var isDefault, isActive *bool
	if v := q.Get("is_default"); v != "" {
		parsed, err := strconv.ParseBool(v)
		if err == nil {
			isDefault = &parsed
		}
	}
	if v := q.Get("is_active"); v != "" {
		parsed, err := strconv.ParseBool(v)
		if err == nil {
			isActive = &parsed
		}
	}

	// Build request DTO (for validation)
	reqParams := dto.RoleFilterDto{
		Name:        util.PtrOrNil(q.Get("name")),
		Description: util.PtrOrNil(q.Get("description")),
		IsDefault:   isDefault,
		IsActive:    isActive,
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
		Name:            reqParams.Name,
		Description:     reqParams.Description,
		IsDefault:       reqParams.IsDefault,
		IsActive:        reqParams.IsActive,
		AuthContainerID: user.AuthContainerID,
		Page:            reqParams.Page,
		Limit:           reqParams.Limit,
		SortBy:          reqParams.SortBy,
		SortOrder:       reqParams.SortOrder,
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
	role, err := h.service.Create(req.Name, req.Description, false, req.IsActive, user.AuthContainerID)
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
	role, err := h.service.Update(roleUUID, req.Name, req.Description, false, req.IsActive, user.AuthContainerID)
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
	// Validate role_uuid
	roleUUID, err := uuid.Parse(chi.URLParam(r, "role_uuid"))
	if err != nil {
		util.Error(w, http.StatusBadRequest, "Invalid role UUID")
		return
	}

	// Update role
	role, err := h.service.SetActiveStatusByUUID(roleUUID)
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
	roleUUID, err := uuid.Parse(chi.URLParam(r, "role_uuid"))
	if err != nil {
		util.Error(w, http.StatusBadRequest, "Invalid role UUID")
		return
	}

	// Delete role
	role, err := h.service.DeleteByUUID(roleUUID)
	if err != nil {
		util.Error(w, http.StatusInternalServerError, "Failed to delete role", err.Error())
		return
	}

	// Build response data
	dtoRes := toRoleResponseDto(*role)

	util.Success(w, dtoRes, "Role deleted successfully")
}

// Convert service result to dto
func toRoleResponseDto(r service.RoleServiceDataResult) dto.RoleResponseDto {
	return dto.RoleResponseDto{
		RoleUUID:    r.RoleUUID,
		Name:        r.Name,
		Description: r.Description,
		IsDefault:   r.IsDefault,
		IsActive:    r.IsActive,
		CreatedAt:   r.CreatedAt,
		UpdatedAt:   r.UpdatedAt,
	}
}
