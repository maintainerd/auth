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

type ServiceHandler struct {
	service service.ServiceService
}

func NewServiceHandler(service service.ServiceService) *ServiceHandler {
	return &ServiceHandler{service}
}

// Get all services with pagination & filters
func (h *ServiceHandler) Get(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()

	// Parse pagination
	page, _ := strconv.Atoi(q.Get("page"))
	limit, _ := strconv.Atoi(q.Get("limit"))

	// Parse bools safely
	var isDefault, isActive, isPublic *bool
	if v := q.Get("is_default"); v != "" {
		if parsed, err := strconv.ParseBool(v); err == nil {
			isDefault = &parsed
		}
	}
	if v := q.Get("is_active"); v != "" {
		if parsed, err := strconv.ParseBool(v); err == nil {
			isActive = &parsed
		}
	}
	if v := q.Get("is_public"); v != "" {
		if parsed, err := strconv.ParseBool(v); err == nil {
			isPublic = &parsed
		}
	}

	// Build request DTO (for validation)
	reqParams := dto.ServiceFilterDto{
		Name:        util.PtrOrNil(q.Get("name")),
		DisplayName: util.PtrOrNil(q.Get("display_name")),
		Description: util.PtrOrNil(q.Get("description")),
		Version:     util.PtrOrNil(q.Get("version")),
		IsActive:    isActive,
		IsPublic:    isPublic,
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

	// Build service filter
	filter := service.ServiceServiceGetFilter{
		Name:        reqParams.Name,
		DisplayName: reqParams.DisplayName,
		Description: reqParams.Description,
		Version:     reqParams.Version,
		IsActive:    reqParams.IsActive,
		IsPublic:    reqParams.IsPublic,
		IsDefault:   reqParams.IsDefault,
		Page:        reqParams.Page,
		Limit:       reqParams.Limit,
		SortBy:      reqParams.SortBy,
		SortOrder:   reqParams.SortOrder,
	}

	// Fetch services
	result, err := h.service.Get(filter)
	if err != nil {
		util.Error(w, http.StatusInternalServerError, "Failed to fetch services", err.Error())
		return
	}

	// Map service result to dto
	rows := make([]dto.ServiceResponseDto, len(result.Data))
	for i, s := range result.Data {
		rows[i] = toServiceResponseDto(s)
	}

	// Build response data
	response := dto.PaginatedResponseDto[dto.ServiceResponseDto]{
		Rows:       rows,
		Total:      result.Total,
		Page:       result.Page,
		Limit:      result.Limit,
		TotalPages: result.TotalPages,
	}

	util.Success(w, response, "Services fetched successfully")
}

// Get service by UUID
func (h *ServiceHandler) GetByUUID(w http.ResponseWriter, r *http.Request) {
	// Validate service_uuid
	serviceUUID, err := uuid.Parse(chi.URLParam(r, "service_uuid"))
	if err != nil {
		util.Error(w, http.StatusBadRequest, "Invalid service UUID")
		return
	}

	// Validate request body
	svc, err := h.service.GetByUUID(serviceUUID)
	if err != nil {
		util.Error(w, http.StatusNotFound, "Service not found")
		return
	}

	// Build response
	dtoRes := toServiceResponseDto(*svc)

	util.Success(w, dtoRes, "Service fetched successfully")
}

// Create service
func (h *ServiceHandler) Create(w http.ResponseWriter, r *http.Request) {
	// Get authentication context
	user := r.Context().Value(middleware.UserContextKey).(*model.User)

	// Validate request body
	var req dto.ServiceCreateOrUpdateRequestDto
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		util.Error(w, http.StatusBadRequest, "Invalid request", err.Error())
		return
	}

	if err := req.Validate(); err != nil {
		util.ValidationError(w, err)
		return
	}

	svc, err := h.service.Create(
		req.Name,
		req.DisplayName,
		req.Description,
		req.Version,
		false,
		req.IsActive,
		req.IsPublic,
		user.TenantID,
	)
	if err != nil {
		util.Error(w, http.StatusInternalServerError, "Failed to create service", err.Error())
		return
	}

	dtoRes := toServiceResponseDto(*svc)
	util.Created(w, dtoRes, "Service created successfully")
}

// Update service
func (h *ServiceHandler) Update(w http.ResponseWriter, r *http.Request) {
	serviceUUID, err := uuid.Parse(chi.URLParam(r, "service_uuid"))
	if err != nil {
		util.Error(w, http.StatusBadRequest, "Invalid service UUID")
		return
	}

	var req dto.ServiceCreateOrUpdateRequestDto
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		util.Error(w, http.StatusBadRequest, "Invalid request", err.Error())
		return
	}

	if err := req.Validate(); err != nil {
		util.ValidationError(w, err)
		return
	}

	svc, err := h.service.Update(
		serviceUUID,
		req.Name,
		req.DisplayName,
		req.Description,
		req.Version,
		false,
		req.IsActive,
		req.IsPublic,
	)
	if err != nil {
		util.Error(w, http.StatusInternalServerError, "Failed to update service", err.Error())
		return
	}

	dtoRes := toServiceResponseDto(*svc)
	util.Success(w, dtoRes, "Service updated successfully")
}

// Set service status
func (h *ServiceHandler) SetStatus(w http.ResponseWriter, r *http.Request) {
	// Validate role_uuid
	serviceUUID, err := uuid.Parse(chi.URLParam(r, "service_uuid"))
	if err != nil {
		util.Error(w, http.StatusBadRequest, "Invalid service UUID")
		return
	}

	// Update service
	service, err := h.service.SetActiveStatusByUUID(serviceUUID)
	if err != nil {
		util.Error(w, http.StatusInternalServerError, "Failed to update service", err.Error())
		return
	}

	// Build response data
	dtoRes := toServiceResponseDto(*service)

	util.Success(w, dtoRes, "Service updated successfully")
}

// Set service public
func (h *ServiceHandler) SetPublic(w http.ResponseWriter, r *http.Request) {
	// Validate role_uuid
	serviceUUID, err := uuid.Parse(chi.URLParam(r, "service_uuid"))
	if err != nil {
		util.Error(w, http.StatusBadRequest, "Invalid service UUID")
		return
	}

	// Update service
	service, err := h.service.SetPublicStatusByUUID(serviceUUID)
	if err != nil {
		util.Error(w, http.StatusInternalServerError, "Failed to update service", err.Error())
		return
	}

	// Build response data
	dtoRes := toServiceResponseDto(*service)

	util.Success(w, dtoRes, "Service updated successfully")
}

// Delete service
func (h *ServiceHandler) Delete(w http.ResponseWriter, r *http.Request) {
	serviceUUID, err := uuid.Parse(chi.URLParam(r, "service_uuid"))
	if err != nil {
		util.Error(w, http.StatusBadRequest, "Invalid service UUID")
		return
	}

	svc, err := h.service.DeleteByUUID(serviceUUID)
	if err != nil {
		util.Error(w, http.StatusInternalServerError, "Failed to delete service", err.Error())
		return
	}

	dtoRes := toServiceResponseDto(*svc)
	util.Success(w, dtoRes, "Service deleted successfully")
}

// Convert service result to dto
func toServiceResponseDto(s service.ServiceServiceDataResult) dto.ServiceResponseDto {
	return dto.ServiceResponseDto{
		ServiceUUID: s.ServiceUUID,
		Name:        s.Name,
		DisplayName: s.DisplayName,
		Description: s.Description,
		Version:     s.Version,
		IsActive:    s.IsActive,
		IsPublic:    s.IsPublic,
		IsDefault:   s.IsDefault,
		CreatedAt:   s.CreatedAt,
		UpdatedAt:   s.UpdatedAt,
	}
}
