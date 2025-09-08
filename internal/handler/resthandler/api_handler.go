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

type APIHandler struct {
	apiService service.APIService
}

func NewAPIHandler(apiService service.APIService) *APIHandler {
	return &APIHandler{apiService}
}

// GetAll APIs with pagination
func (h *APIHandler) Get(w http.ResponseWriter, r *http.Request) {
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
	reqParams := dto.APIFilterDto{
		Name:        util.PtrOrNil(q.Get("name")),
		DisplayName: util.PtrOrNil(q.Get("display_name")),
		APIType:     util.PtrOrNil(q.Get("api_type")),
		Identifier:  util.PtrOrNil(q.Get("identifier")),
		ServiceUUID: util.PtrOrNil(q.Get("service_uuid")),
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

	// Build service filter
	apiFilter := service.APIServiceGetFilter{
		Name:        reqParams.Name,
		DisplayName: reqParams.DisplayName,
		APIType:     reqParams.APIType,
		Identifier:  reqParams.Identifier,
		ServiceID:   nil, // optional, skip mapping
		IsDefault:   reqParams.IsDefault,
		IsActive:    reqParams.IsActive,
		Page:        reqParams.Page,
		Limit:       reqParams.Limit,
		SortBy:      reqParams.SortBy,
		SortOrder:   reqParams.SortOrder,
	}

	// Fetch APIs
	result, err := h.apiService.Get(apiFilter)
	if err != nil {
		util.Error(w, http.StatusInternalServerError, "Failed to fetch APIs", err.Error())
		return
	}

	// Map service result to DTO
	rows := make([]dto.APIResponseDto, len(result.Data))
	for i, r := range result.Data {
		rows[i] = toAPIResponseDto(r)
	}

	// Build response data
	response := dto.PaginatedResponseDto[dto.APIResponseDto]{
		Rows:       rows,
		Total:      result.Total,
		Page:       result.Page,
		Limit:      result.Limit,
		TotalPages: result.TotalPages,
	}

	util.Success(w, response, "APIs fetched successfully")
}

// Get API by UUID
func (h *APIHandler) GetByUUID(w http.ResponseWriter, r *http.Request) {
	apiUUID, err := uuid.Parse(chi.URLParam(r, "api_uuid"))
	if err != nil {
		util.Error(w, http.StatusBadRequest, "Invalid API UUID")
		return
	}

	api, err := h.apiService.GetByUUID(apiUUID)
	if err != nil {
		util.Error(w, http.StatusNotFound, "API not found")
		return
	}

	dtoRes := toAPIResponseDto(*api)

	util.Success(w, dtoRes, "API fetched successfully")
}

// Create API
func (h *APIHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req dto.APICreateRequestDto
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		util.Error(w, http.StatusBadRequest, "Invalid request", err.Error())
		return
	}

	if err := req.Validate(); err != nil {
		util.ValidationError(w, err)
		return
	}

	api, err := h.apiService.Create(req.Name, req.DisplayName, req.Description, req.APIType, req.IsActive, req.IsDefault, req.ServiceUUID)
	if err != nil {
		util.Error(w, http.StatusInternalServerError, "Failed to create API", err.Error())
		return
	}

	dtoRes := toAPIResponseDto(*api)

	util.Created(w, dtoRes, "API created successfully")
}

// Update API
func (h *APIHandler) Update(w http.ResponseWriter, r *http.Request) {
	apiUUID, err := uuid.Parse(chi.URLParam(r, "api_uuid"))
	if err != nil {
		util.Error(w, http.StatusBadRequest, "Invalid API UUID")
		return
	}

	var req dto.APIUpdateRequestDto
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		util.Error(w, http.StatusBadRequest, "Invalid request", err.Error())
		return
	}

	if err := req.Validate(); err != nil {
		util.ValidationError(w, err)
		return
	}

	api, err := h.apiService.Update(apiUUID, req.Name, req.DisplayName, req.Description, req.APIType, req.IsActive, req.IsDefault)
	if err != nil {
		util.Error(w, http.StatusInternalServerError, "Failed to update API", err.Error())
		return
	}

	dtoRes := toAPIResponseDto(*api)

	util.Success(w, dtoRes, "API updated successfully")
}

// Set API status
func (h *APIHandler) SetStatus(w http.ResponseWriter, r *http.Request) {
	apiUUID, err := uuid.Parse(chi.URLParam(r, "api_uuid"))
	if err != nil {
		util.Error(w, http.StatusBadRequest, "Invalid API UUID")
		return
	}

	api, err := h.apiService.SetActiveStatusByUUID(apiUUID)
	if err != nil {
		util.Error(w, http.StatusInternalServerError, "Failed to update API", err.Error())
		return
	}

	dtoRes := toAPIResponseDto(*api)

	util.Success(w, dtoRes, "API status updated successfully")
}

// Delete API
func (h *APIHandler) Delete(w http.ResponseWriter, r *http.Request) {
	apiUUID, err := uuid.Parse(chi.URLParam(r, "api_uuid"))
	if err != nil {
		util.Error(w, http.StatusBadRequest, "Invalid API UUID")
		return
	}

	api, err := h.apiService.DeleteByUUID(apiUUID)
	if err != nil {
		util.Error(w, http.StatusInternalServerError, "Failed to delete API", err.Error())
		return
	}

	dtoRes := toAPIResponseDto(*api)

	util.Success(w, dtoRes, "API deleted successfully")
}

// Convert service result to DTO
func toAPIResponseDto(r service.APIServiceDataResult) dto.APIResponseDto {
	return dto.APIResponseDto{
		APIUUID:     r.APIUUID,
		Name:        r.Name,
		DisplayName: r.DisplayName,
		Description: r.Description,
		APIType:     r.APIType,
		Identifier:  r.Identifier,
		Service: dto.ServiceResponseDto{
			ServiceUUID: r.Service.ServiceUUID,
			Name:        r.Service.Name,
			DisplayName: r.Service.DisplayName,
			Description: r.Service.Description,
			Version:     r.Service.Version,
			IsDefault:   r.Service.IsDefault,
			IsActive:    r.Service.IsActive,
			IsPublic:    r.Service.IsPublic,
			CreatedAt:   r.Service.CreatedAt,
			UpdatedAt:   r.Service.UpdatedAt,
		},
		IsActive:  r.IsActive,
		IsDefault: r.IsDefault,
		CreatedAt: r.CreatedAt,
		UpdatedAt: r.UpdatedAt,
	}
}
