package resthandler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/dto"
	"github.com/maintainerd/auth/internal/service"
	"github.com/maintainerd/auth/internal/util"
)

type PermissionHandler struct {
	permissionService service.PermissionService
}

func NewPermissionHandler(permissionService service.PermissionService) *PermissionHandler {
	return &PermissionHandler{permissionService}
}

// Get permissions with pagination
func (h *PermissionHandler) Get(w http.ResponseWriter, r *http.Request) {
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

	// Build request DTO
	reqParams := dto.PermissionFilterDto{
		Name:           util.PtrOrNil(q.Get("name")),
		Description:    util.PtrOrNil(q.Get("description")),
		APIUUID:        util.PtrOrNil(q.Get("api_uuid")),
		RoleUUID:       util.PtrOrNil(q.Get("role_uuid")),
		AuthClientUUID: util.PtrOrNil(q.Get("auth_client_uuid")),
		IsActive:       isActive,
		IsDefault:      isDefault,
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
	permissionFilter := service.PermissionServiceGetFilter{
		Name:           reqParams.Name,
		Description:    reqParams.Description,
		APIUUID:        reqParams.APIUUID,
		RoleUUID:       reqParams.RoleUUID,
		AuthClientUUID: reqParams.AuthClientUUID,
		IsActive:       reqParams.IsActive,
		IsDefault:      reqParams.IsDefault,
		Page:           reqParams.Page,
		Limit:          reqParams.Limit,
		SortBy:         reqParams.SortBy,
		SortOrder:      reqParams.SortOrder,
	}

	// Fetch permissions
	result, err := h.permissionService.Get(permissionFilter)
	if err != nil {
		util.Error(w, http.StatusInternalServerError, "Failed to fetch permissions", err.Error())
		return
	}

	// Map permissions result to DTO
	rows := make([]dto.PermissionResponseDto, len(result.Data))
	for i, r := range result.Data {
		rows[i] = toPermissionResponseDto(r)
	}

	// Build response data
	response := dto.PaginatedResponseDto[dto.PermissionResponseDto]{
		Rows:       rows,
		Total:      result.Total,
		Page:       result.Page,
		Limit:      result.Limit,
		TotalPages: result.TotalPages,
	}

	util.Success(w, response, "Permissions fetched successfully")
}

// Get permission by UUID
func (h *PermissionHandler) GetByUUID(w http.ResponseWriter, r *http.Request) {
	permissonUUID, err := uuid.Parse(chi.URLParam(r, "permission_uuid"))
	if err != nil {
		util.Error(w, http.StatusBadRequest, "Invalid permission UUID")
		return
	}

	permission, err := h.permissionService.GetByUUID(permissonUUID)
	if err != nil {
		util.Error(w, http.StatusNotFound, "Permission not found")
		return
	}

	dtoRes := toPermissionResponseDto(*permission)

	util.Success(w, dtoRes, "Permission fetched successfully")
}

// Create permission
func (h *PermissionHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req dto.PermissionCreateRequestDto
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		util.Error(w, http.StatusBadRequest, "Invalid request", err.Error())
		return
	}

	if err := req.Validate(); err != nil {
		util.ValidationError(w, err)
		return
	}

	permission, err := h.permissionService.Create(req.Name, req.Description, req.IsActive, false, req.APIUUID)
	if err != nil {
		util.Error(w, http.StatusInternalServerError, "Failed to create permission", err.Error())
		return
	}

	dtoRes := toPermissionResponseDto(*permission)

	util.Created(w, dtoRes, "Permission created successfully")
}

// Update permission
func (h *PermissionHandler) Update(w http.ResponseWriter, r *http.Request) {
	permissionUUID, err := uuid.Parse(chi.URLParam(r, "permission_uuid"))
	if err != nil {
		util.Error(w, http.StatusBadRequest, "Invalid permission UUID")
		return
	}

	var req dto.PermissionUpdateRequestDto
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		util.Error(w, http.StatusBadRequest, "Invalid request", err.Error())
		return
	}

	if err := req.Validate(); err != nil {
		util.ValidationError(w, err)
		return
	}

	permission, err := h.permissionService.Update(permissionUUID, req.Name, req.Description, req.IsActive, false)
	if err != nil {
		util.Error(w, http.StatusInternalServerError, "Failed to update auth container", err.Error())
		return
	}

	dtoRes := toPermissionResponseDto(*permission)

	util.Success(w, dtoRes, "Permission updated successfully")
}

// Set permission status
func (h *PermissionHandler) SetStatus(w http.ResponseWriter, r *http.Request) {
	permissionUUID, err := uuid.Parse(chi.URLParam(r, "permission_uuid"))
	if err != nil {
		util.Error(w, http.StatusBadRequest, "Invalid permission UUID")
		return
	}

	permission, err := h.permissionService.SetActiveStatusByUUID(permissionUUID)
	if err != nil {
		util.Error(w, http.StatusInternalServerError, "Failed to update permission", err.Error())
		return
	}

	dtoRes := toPermissionResponseDto(*permission)

	util.Success(w, dtoRes, "Permission status updated successfully")
}

// Delete permission
func (h *PermissionHandler) Delete(w http.ResponseWriter, r *http.Request) {
	permissionUUID, err := uuid.Parse(chi.URLParam(r, "permission_uuid"))
	if err != nil {
		util.Error(w, http.StatusBadRequest, "Invalid permission UUID")
		return
	}

	permission, err := h.permissionService.DeleteByUUID(permissionUUID)
	if err != nil {
		util.Error(w, http.StatusInternalServerError, "Failed to delete auth container", err.Error())
		return
	}

	dtoRes := toPermissionResponseDto(*permission)

	util.Success(w, dtoRes, "Permission deleted successfully")
}

// Convert permission result to DTO
func toPermissionResponseDto(r service.PermissionServiceDataResult) dto.PermissionResponseDto {
	result := dto.PermissionResponseDto{
		PermissionUUID: r.PermissionUUID,
		Name:           r.Name,
		Description:    r.Description,
		IsActive:       r.IsActive,
		IsDefault:      r.IsDefault,
		CreatedAt:      r.CreatedAt,
		UpdatedAt:      r.UpdatedAt,
	}

	if r.API != nil {
		result.API = &dto.APIResponseDto{
			APIUUID:     r.API.APIUUID,
			Name:        r.API.Name,
			DisplayName: r.API.DisplayName,
			Description: r.API.Description,
			APIType:     r.API.APIType,
			Identifier:  r.API.Identifier,
			Status:      r.API.Status,
			IsDefault:   r.API.IsDefault,
			CreatedAt:   r.API.CreatedAt,
			UpdatedAt:   r.API.UpdatedAt,
		}
	}

	return result
}
