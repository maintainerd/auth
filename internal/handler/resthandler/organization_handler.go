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

type OrganizationHandler struct {
	organizationService service.OrganizationService
}

func NewOrganizationHandler(organizationService service.OrganizationService) *OrganizationHandler {
	return &OrganizationHandler{organizationService}
}

// GetAll roles with pagination
func (h *OrganizationHandler) Get(w http.ResponseWriter, r *http.Request) {
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
	reqParams := dto.OrganizationFilterDto{
		Name:        util.PtrOrNil(q.Get("name")),
		Description: util.PtrOrNil(q.Get("description")),
		Email:       util.PtrOrNil(q.Get("email")),
		Phone:       util.PtrOrNil(q.Get("phone")),
		IsActive:    isActive,
		IsDefault:   isDefault,
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

	// Get organization filters
	orgFilter := service.OrganizationServiceGetFilter{
		Name:        reqParams.Name,
		Description: reqParams.Description,
		Email:       reqParams.Email,
		Phone:       reqParams.Phone,
		IsDefault:   reqParams.IsDefault,
		IsActive:    reqParams.IsActive,
		Page:        reqParams.Page,
		Limit:       reqParams.Limit,
		SortBy:      reqParams.SortBy,
		SortOrder:   reqParams.SortOrder,
	}

	// Fetch organizations
	result, err := h.organizationService.Get(orgFilter)
	if err != nil {
		util.Error(w, http.StatusInternalServerError, "Failed to fetch organizations", err.Error())
		return
	}

	// Map result to dto
	rows := make([]dto.OrganizationResponseDto, len(result.Data))
	for i, r := range result.Data {
		rows[i] = toOrganizationResponseDto(r)
	}

	// Build response data
	response := dto.PaginatedResponseDto[dto.OrganizationResponseDto]{
		Rows:       rows,
		Total:      result.Total,
		Page:       result.Page,
		Limit:      result.Limit,
		TotalPages: result.TotalPages,
	}

	util.Success(w, response, "Organization fetched successfully")
}

// Get organization by UUID
func (h *OrganizationHandler) GetByUUID(w http.ResponseWriter, r *http.Request) {
	orgUUID, err := uuid.Parse(chi.URLParam(r, "organization_uuid"))
	if err != nil {
		util.Error(w, http.StatusBadRequest, "Invalid organization UUID")
		return
	}

	// Fetch organization
	org, err := h.organizationService.GetByUUID(orgUUID)
	if err != nil {
		util.Error(w, http.StatusNotFound, "Organization not found")
		return
	}

	// Build response data
	dtoRes := toOrganizationResponseDto(*org)

	util.Success(w, dtoRes, "Organization fetched successfully")
}

// Create organization
func (h *OrganizationHandler) Create(w http.ResponseWriter, r *http.Request) {
	// Validate request body
	var req dto.OrganizationCreateOrUpdateRequestDto
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		util.Error(w, http.StatusBadRequest, "Invalid request", err.Error())
		return
	}

	if err := req.Validate(); err != nil {
		util.ValidationError(w, err)
		return
	}

	// Create organization
	org, err := h.organizationService.Create(req.Name, *req.Description, *req.Email, *req.Phone, req.IsActive, false)
	if err != nil {
		util.Error(w, http.StatusInternalServerError, "Failed to create organization", err.Error())
		return
	}

	// Build response data
	dtoRes := toOrganizationResponseDto(*org)

	util.Created(w, dtoRes, "Organization created successfully")
}

// Update organization
func (h *OrganizationHandler) Update(w http.ResponseWriter, r *http.Request) {
	// Get authentication context
	user := r.Context().Value(middleware.UserContextKey).(*model.User)

	// Validate organization_uuid
	orgUUID, err := uuid.Parse(chi.URLParam(r, "organization_uuid"))
	if err != nil {
		util.Error(w, http.StatusBadRequest, "Invalid organization UUID")
		return
	}

	// Validate request body
	var req dto.OrganizationCreateOrUpdateRequestDto
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		util.Error(w, http.StatusBadRequest, "Invalid request", err.Error())
		return
	}

	if err := req.Validate(); err != nil {
		util.ValidationError(w, err)
		return
	}

	// Update organization
	org, err := h.organizationService.Update(orgUUID, req.Name, *req.Description, *req.Email, *req.Phone, req.IsActive, false, user.UserUUID)
	if err != nil {
		util.Error(w, http.StatusInternalServerError, "Failed to update organization", err.Error())
		return
	}

	// Build response data
	dtoRes := toOrganizationResponseDto(*org)

	util.Success(w, dtoRes, "Organization updated successfully")
}

// Set organization status
func (h *OrganizationHandler) SetStatus(w http.ResponseWriter, r *http.Request) {
	// Get authentication context
	user := r.Context().Value(middleware.UserContextKey).(*model.User)

	// Validate organization_uuid
	orgUUID, err := uuid.Parse(chi.URLParam(r, "organization_uuid"))
	if err != nil {
		util.Error(w, http.StatusBadRequest, "Invalid organization UUID")
		return
	}

	// Update organization
	org, err := h.organizationService.SetActiveStatusByUUID(orgUUID, user.UserUUID)
	if err != nil {
		util.Error(w, http.StatusInternalServerError, "Failed to update organization", err.Error())
		return
	}

	// Build response data
	dtoRes := toOrganizationResponseDto(*org)

	util.Success(w, dtoRes, "Organization updated successfully")
}

// Delete organization
func (h *OrganizationHandler) Delete(w http.ResponseWriter, r *http.Request) {
	// Get authentication context
	user := r.Context().Value(middleware.UserContextKey).(*model.User)

	// Validate organization_uuid
	orgUUID, err := uuid.Parse(chi.URLParam(r, "organization_uuid"))
	if err != nil {
		util.Error(w, http.StatusBadRequest, "Invalid organization UUID")
		return
	}

	// Delete organization
	org, err := h.organizationService.DeleteByUUID(orgUUID, user.UserUUID)
	if err != nil {
		util.Error(w, http.StatusInternalServerError, "Failed to delete organization", err.Error())
		return
	}

	// Build response data
	dtoRes := toOrganizationResponseDto(*org)

	util.Success(w, dtoRes, "Organization deleted successfully")
}

// Convert organization result to dto
func toOrganizationResponseDto(r service.OrganizationServiceDataResult) dto.OrganizationResponseDto {
	return dto.OrganizationResponseDto{
		OrganizationUUID: r.OrganizationUUID,
		Name:             r.Name,
		Description:      *r.Description,
		Email:            *r.Email,
		Phone:            *r.Phone,
		IsActive:         r.IsActive,
		IsDefault:        r.IsDefault,
		CreatedAt:        r.CreatedAt,
		UpdatedAt:        r.UpdatedAt,
	}
}
