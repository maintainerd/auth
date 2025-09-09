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

type AuthContainerHandler struct {
	authContainerService service.AuthContainerService
}

func NewAuthContainerHandler(authContainerService service.AuthContainerService) *AuthContainerHandler {
	return &AuthContainerHandler{authContainerService}
}

// Get all auth containers with pagination
func (h *AuthContainerHandler) Get(w http.ResponseWriter, r *http.Request) {
	// Parse query parameters
	q := r.URL.Query()

	// Parse pagination
	page, _ := strconv.Atoi(q.Get("page"))
	limit, _ := strconv.Atoi(q.Get("limit"))

	// Parse bools safely
	var isDefault, isActive, isPublic *bool
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
	if v := q.Get("is_public"); v != "" {
		parsed, err := strconv.ParseBool(v)
		if err == nil {
			isPublic = &parsed
		}
	}

	// Build request DTO
	reqParams := dto.AuthContainerFilterDto{
		Name:             util.PtrOrNil(q.Get("name")),
		Description:      util.PtrOrNil(q.Get("description")),
		Identifier:       util.PtrOrNil(q.Get("identifier")),
		OrganizationUUID: util.PtrOrNil(q.Get("organization_uuid")),
		IsDefault:        isDefault,
		IsPublic:         isPublic,
		IsActive:         isActive,
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

	// Build service filter
	authContainerFilter := service.AuthContainerServiceGetFilter{
		Name:             reqParams.Name,
		Description:      reqParams.Description,
		Identifier:       reqParams.Identifier,
		OrganizationUUID: reqParams.OrganizationUUID,
		IsDefault:        reqParams.IsDefault,
		IsPublic:         isPublic,
		IsActive:         reqParams.IsActive,
		Page:             reqParams.Page,
		Limit:            reqParams.Limit,
		SortBy:           reqParams.SortBy,
		SortOrder:        reqParams.SortOrder,
	}

	// Fetch Auth Containers
	result, err := h.authContainerService.Get(authContainerFilter)
	if err != nil {
		util.Error(w, http.StatusInternalServerError, "Failed to fetch auth containers", err.Error())
		return
	}

	// Map auth container result to DTO
	rows := make([]dto.AuthContainerResponseDto, len(result.Data))
	for i, r := range result.Data {
		rows[i] = toAuthContainerResponseDto(r)
	}

	// Build response data
	response := dto.PaginatedResponseDto[dto.AuthContainerResponseDto]{
		Rows:       rows,
		Total:      result.Total,
		Page:       result.Page,
		Limit:      result.Limit,
		TotalPages: result.TotalPages,
	}

	util.Success(w, response, "Auth containers fetched successfully")
}

// Get Auth container by UUID
func (h *AuthContainerHandler) GetByUUID(w http.ResponseWriter, r *http.Request) {
	authContainerUUID, err := uuid.Parse(chi.URLParam(r, "auth_container_uuid"))
	if err != nil {
		util.Error(w, http.StatusBadRequest, "Invalid Auth container UUID")
		return
	}

	authContainer, err := h.authContainerService.GetByUUID(authContainerUUID)
	if err != nil {
		util.Error(w, http.StatusNotFound, "Auth container not found")
		return
	}

	dtoRes := toAuthContainerResponseDto(*authContainer)

	util.Success(w, dtoRes, "Auth container fetched successfully")
}

// Create Auth Container
func (h *AuthContainerHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req dto.AuthContainerCreateRequestDto
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		util.Error(w, http.StatusBadRequest, "Invalid request", err.Error())
		return
	}

	if err := req.Validate(); err != nil {
		util.ValidationError(w, err)
		return
	}

	authContainer, err := h.authContainerService.Create(req.Name, req.Description, req.IsActive, req.IsPublic, false, req.OrganizationUUID)
	if err != nil {
		util.Error(w, http.StatusInternalServerError, "Failed to create auth container", err.Error())
		return
	}

	dtoRes := toAuthContainerResponseDto(*authContainer)

	util.Created(w, dtoRes, "Auth container created successfully")
}

// Update Auth Container
func (h *AuthContainerHandler) Update(w http.ResponseWriter, r *http.Request) {
	authContainerUUID, err := uuid.Parse(chi.URLParam(r, "auth_container_uuid"))
	if err != nil {
		util.Error(w, http.StatusBadRequest, "Invalid auth container UUID")
		return
	}

	var req dto.AuthContainerUpdateRequestDto
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		util.Error(w, http.StatusBadRequest, "Invalid request", err.Error())
		return
	}

	if err := req.Validate(); err != nil {
		util.ValidationError(w, err)
		return
	}

	authContainer, err := h.authContainerService.Update(authContainerUUID, req.Name, req.Description, req.IsActive, req.IsPublic, false)
	if err != nil {
		util.Error(w, http.StatusInternalServerError, "Failed to update auth container", err.Error())
		return
	}

	dtoRes := toAuthContainerResponseDto(*authContainer)

	util.Success(w, dtoRes, "Auth container updated successfully")
}

// Set Auth container status
func (h *AuthContainerHandler) SetStatus(w http.ResponseWriter, r *http.Request) {
	authContainerUUID, err := uuid.Parse(chi.URLParam(r, "auth_container_uuid"))
	if err != nil {
		util.Error(w, http.StatusBadRequest, "Invalid auth container UUID")
		return
	}

	authContainer, err := h.authContainerService.SetActiveStatusByUUID(authContainerUUID)
	if err != nil {
		util.Error(w, http.StatusInternalServerError, "Failed to update API", err.Error())
		return
	}

	dtoRes := toAuthContainerResponseDto(*authContainer)

	util.Success(w, dtoRes, "Auth container status updated successfully")
}

// Delete Auth Container
func (h *AuthContainerHandler) Delete(w http.ResponseWriter, r *http.Request) {
	authContainerUUID, err := uuid.Parse(chi.URLParam(r, "auth_container_uuid"))
	if err != nil {
		util.Error(w, http.StatusBadRequest, "Invalid Auth Container UUID")
		return
	}

	authContainer, err := h.authContainerService.DeleteByUUID(authContainerUUID)
	if err != nil {
		util.Error(w, http.StatusInternalServerError, "Failed to delete auth container", err.Error())
		return
	}

	dtoRes := toAuthContainerResponseDto(*authContainer)

	util.Success(w, dtoRes, "Auth container deleted successfully")
}

// Convert service result to DTO
func toAuthContainerResponseDto(r service.AuthContainerServiceDataResult) dto.AuthContainerResponseDto {
	return dto.AuthContainerResponseDto{
		AuthContainerUUID: r.AuthContainerUUID,
		Name:              r.Name,
		Description:       r.Description,
		Identifier:        r.Identifier,
		Organization: dto.OrganizationResponseDto{
			OrganizationUUID: r.Organization.OrganizationUUID,
			Name:             r.Organization.Name,
			Description:      *r.Organization.Description,
			Email:            *r.Organization.Email,
			Phone:            *r.Organization.Phone,
			IsActive:         r.Organization.IsActive,
			IsDefault:        r.Organization.IsDefault,
			CreatedAt:        r.Organization.CreatedAt,
			UpdatedAt:        r.Organization.UpdatedAt,
		},
		IsActive:  r.IsActive,
		IsPublic:  r.IsPublic,
		IsDefault: r.IsDefault,
		CreatedAt: r.CreatedAt,
		UpdatedAt: r.UpdatedAt,
	}
}
