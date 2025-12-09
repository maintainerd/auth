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

type APIKeyHandler struct {
	apiKeyService service.APIKeyService
}

func NewAPIKeyHandler(apiKeyService service.APIKeyService) *APIKeyHandler {
	return &APIKeyHandler{
		apiKeyService: apiKeyService,
	}
}

// Get API keys with pagination and filtering
func (h *APIKeyHandler) Get(w http.ResponseWriter, r *http.Request) {
	// Get authentication context
	requestingUser := r.Context().Value(middleware.UserContextKey).(*model.User)

	// Parse query parameters
	var reqParams dto.APIKeyGetRequestDto
	reqParams.Page, _ = strconv.Atoi(r.URL.Query().Get("page"))
	reqParams.Limit, _ = strconv.Atoi(r.URL.Query().Get("limit"))
	reqParams.SortBy = r.URL.Query().Get("sort_by")
	reqParams.SortOrder = r.URL.Query().Get("sort_order")

	if name := r.URL.Query().Get("name"); name != "" {
		reqParams.Name = &name
	}
	if description := r.URL.Query().Get("description"); description != "" {
		reqParams.Description = &description
	}
	if status := r.URL.Query().Get("status"); status != "" {
		reqParams.Status = &status
	}
	// UserUUID parameter removed

	// Set defaults
	if reqParams.Page == 0 {
		reqParams.Page = 1
	}
	if reqParams.Limit == 0 {
		reqParams.Limit = 10
	}
	if reqParams.SortBy == "" {
		reqParams.SortBy = "created_at"
	}
	if reqParams.SortOrder == "" {
		reqParams.SortOrder = "desc"
	}

	// Validate request
	if err := reqParams.Validate(); err != nil {
		util.ValidationError(w, err)
		return
	}

	// Build service filter
	serviceFilter := service.APIKeyServiceGetFilter{
		Name:        reqParams.Name,
		Description: reqParams.Description,
		Status:      reqParams.Status,
		Page:        reqParams.Page,
		Limit:       reqParams.Limit,
		SortBy:      reqParams.SortBy,
		SortOrder:   reqParams.SortOrder,
	}

	// Fetch API keys
	result, err := h.apiKeyService.Get(serviceFilter, requestingUser.UserUUID)
	if err != nil {
		util.Error(w, http.StatusInternalServerError, "Failed to fetch API keys", err.Error())
		return
	}

	// Convert to DTOs
	var dtoResults []dto.APIKeyResponseDto
	for _, apiKey := range result.Data {
		dtoResult := toAPIKeyResponseDto(apiKey)
		dtoResults = append(dtoResults, dtoResult)
	}

	// Build paginated response
	response := dto.PaginatedResponseDto[dto.APIKeyResponseDto]{
		Rows:       dtoResults,
		Total:      result.Total,
		Page:       result.Page,
		Limit:      result.Limit,
		TotalPages: result.TotalPages,
	}

	util.Success(w, response, "API keys fetched successfully")
}

// Get API key by UUID
func (h *APIKeyHandler) GetByUUID(w http.ResponseWriter, r *http.Request) {
	// Get authentication context
	requestingUser := r.Context().Value(middleware.UserContextKey).(*model.User)

	apiKeyUUIDStr := chi.URLParam(r, "api_key_uuid")
	apiKeyUUID, err := uuid.Parse(apiKeyUUIDStr)
	if err != nil {
		util.Error(w, http.StatusBadRequest, "Invalid API key UUID")
		return
	}

	apiKey, err := h.apiKeyService.GetByUUID(apiKeyUUID, requestingUser.UserUUID)
	if err != nil {
		util.Error(w, http.StatusNotFound, "API key not found")
		return
	}

	// Build response data
	dtoRes := toAPIKeyResponseDto(*apiKey)

	util.Success(w, dtoRes, "API key fetched successfully")
}

// Get API key config by UUID
func (h *APIKeyHandler) GetConfigByUUID(w http.ResponseWriter, r *http.Request) {
	apiKeyUUID, err := uuid.Parse(chi.URLParam(r, "api_key_uuid"))
	if err != nil {
		util.Error(w, http.StatusBadRequest, "Invalid API key UUID", err.Error())
		return
	}

	apiKeyConfig, err := h.apiKeyService.GetConfigByUUID(apiKeyUUID)
	if err != nil {
		util.Error(w, http.StatusNotFound, "API key not found", err.Error())
		return
	}

	// Return config directly as data (not wrapped in DTO)
	util.Success(w, apiKeyConfig, "API key config fetched successfully")
}

// Create API key
func (h *APIKeyHandler) Create(w http.ResponseWriter, r *http.Request) {
	// Get authentication context (not needed for API key creation anymore)
	_ = r.Context().Value(middleware.UserContextKey).(*model.User)

	var req dto.APIKeyCreateRequestDto
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		util.Error(w, http.StatusBadRequest, "Invalid request", err.Error())
		return
	}

	if err := req.Validate(); err != nil {
		util.ValidationError(w, err)
		return
	}

	// Set default status if not provided
	status := "active"
	if req.Status != "" {
		status = req.Status
	}

	apiKey, plainKey, err := h.apiKeyService.Create(req.Name, req.Description, req.Config, req.ExpiresAt, req.RateLimit, status)
	if err != nil {
		util.Error(w, http.StatusInternalServerError, "Failed to create API key", err.Error())
		return
	}

	// Build response data using flat DTO structure
	response := dto.APIKeyCreateResponseDto{
		APIKeyID:    apiKey.APIKeyUUID,
		Name:        apiKey.Name,
		Description: apiKey.Description,
		KeyPrefix:   apiKey.KeyPrefix,
		Key:         plainKey, // This is the actual API key that should be stored securely
		ExpiresAt:   apiKey.ExpiresAt,
		LastUsedAt:  apiKey.LastUsedAt,
		UsageCount:  apiKey.UsageCount,
		RateLimit:   apiKey.RateLimit,
		Status:      apiKey.Status,
		CreatedAt:   apiKey.CreatedAt,
		UpdatedAt:   apiKey.UpdatedAt,
	}

	util.Created(w, response, "API key created successfully")
}

// Update API key
func (h *APIKeyHandler) Update(w http.ResponseWriter, r *http.Request) {
	// Get authentication context
	updaterUser := r.Context().Value(middleware.UserContextKey).(*model.User)

	apiKeyUUIDStr := chi.URLParam(r, "api_key_uuid")
	apiKeyUUID, err := uuid.Parse(apiKeyUUIDStr)
	if err != nil {
		util.Error(w, http.StatusBadRequest, "Invalid API key UUID")
		return
	}

	var req dto.APIKeyUpdateRequestDto
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		util.Error(w, http.StatusBadRequest, "Invalid request", err.Error())
		return
	}

	if err := req.Validate(); err != nil {
		util.ValidationError(w, err)
		return
	}

	apiKey, err := h.apiKeyService.Update(apiKeyUUID, req.Name, req.Description, req.Config, req.ExpiresAt, req.RateLimit, req.Status, updaterUser.UserUUID)
	if err != nil {
		util.Error(w, http.StatusInternalServerError, "Failed to update API key", err.Error())
		return
	}

	// Build response data
	dtoRes := toAPIKeyResponseDto(*apiKey)

	util.Success(w, dtoRes, "API key updated successfully")
}

// Delete API key
func (h *APIKeyHandler) Delete(w http.ResponseWriter, r *http.Request) {
	// Get authentication context
	deleterUser := r.Context().Value(middleware.UserContextKey).(*model.User)

	apiKeyUUIDStr := chi.URLParam(r, "api_key_uuid")
	apiKeyUUID, err := uuid.Parse(apiKeyUUIDStr)
	if err != nil {
		util.Error(w, http.StatusBadRequest, "Invalid API key UUID")
		return
	}

	apiKey, err := h.apiKeyService.Delete(apiKeyUUID, deleterUser.UserUUID)
	if err != nil {
		util.Error(w, http.StatusInternalServerError, "Failed to delete API key", err.Error())
		return
	}

	// Build response data
	dtoRes := toAPIKeyResponseDto(*apiKey)

	util.Success(w, dtoRes, "API key deleted successfully")
}

// Convert service result to DTO
func toAPIKeyResponseDto(r service.APIKeyServiceDataResult) dto.APIKeyResponseDto {
	result := dto.APIKeyResponseDto{
		APIKeyID:    r.APIKeyUUID,
		Name:        r.Name,
		Description: r.Description,
		KeyPrefix:   r.KeyPrefix,
		ExpiresAt:   r.ExpiresAt,
		LastUsedAt:  r.LastUsedAt,
		UsageCount:  r.UsageCount,
		RateLimit:   r.RateLimit,
		Status:      r.Status,
		CreatedAt:   r.CreatedAt,
		UpdatedAt:   r.UpdatedAt,
	}

	// User and Tenant relationships removed

	return result
}

// Get APIs assigned to API key with pagination
func (h *APIKeyHandler) GetApis(w http.ResponseWriter, r *http.Request) {
	apiKeyUUID, err := uuid.Parse(chi.URLParam(r, "api_key_uuid"))
	if err != nil {
		util.Error(w, http.StatusBadRequest, "Invalid API key UUID")
		return
	}

	// Parse query parameters
	q := r.URL.Query()

	// Parse pagination
	page, _ := strconv.Atoi(q.Get("page"))
	limit, _ := strconv.Atoi(q.Get("limit"))

	// Set defaults
	if page <= 0 {
		page = 1
	}
	if limit <= 0 {
		limit = 10
	}

	// Build request DTO
	reqParams := dto.APIKeyApisGetRequestDto{
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

	// Get API key APIs with pagination
	result, err := h.apiKeyService.GetAPIKeyApis(apiKeyUUID, reqParams.Page, reqParams.Limit, reqParams.SortBy, reqParams.SortOrder)
	if err != nil {
		util.Error(w, http.StatusInternalServerError, "Failed to get API key APIs")
		return
	}

	// Convert to DTO
	apiDtos := make([]dto.APIKeyApiResponseDto, len(result.Data))
	for i, api := range result.Data {
		// Convert API
		apiDto := dto.APIResponseDto{
			APIUUID:     api.Api.APIUUID,
			Name:        api.Api.Name,
			DisplayName: api.Api.DisplayName,
			Description: api.Api.Description,
			APIType:     api.Api.APIType,
			Identifier:  api.Api.Identifier,
			Status:      api.Api.Status,
			IsDefault:   api.Api.IsDefault,
			IsSystem:    api.Api.IsSystem,
			CreatedAt:   api.Api.CreatedAt,
			UpdatedAt:   api.Api.UpdatedAt,
		}

		// Convert permissions
		permissions := make([]dto.PermissionResponseDto, len(api.Permissions))
		for j, perm := range api.Permissions {
			permissions[j] = dto.PermissionResponseDto{
				PermissionUUID: perm.PermissionUUID,
				Name:           perm.Name,
				Description:    perm.Description,
				Status:         perm.Status,
				IsDefault:      perm.IsDefault,
				IsSystem:       perm.IsSystem,
				CreatedAt:      perm.CreatedAt,
				UpdatedAt:      perm.UpdatedAt,
			}
		}

		apiDtos[i] = dto.APIKeyApiResponseDto{
			APIKeyApiID: api.APIKeyApiUUID,
			Api:         apiDto,
			Permissions: permissions,
			CreatedAt:   api.CreatedAt,
		}
	}

	// Build paginated response
	response := dto.PaginatedResponseDto[dto.APIKeyApiResponseDto]{
		Rows:       apiDtos,
		Total:      result.Total,
		Page:       result.Page,
		Limit:      result.Limit,
		TotalPages: result.TotalPages,
	}

	util.Success(w, response, "API key APIs retrieved successfully")
}

// Add APIs to API key
func (h *APIKeyHandler) AddApis(w http.ResponseWriter, r *http.Request) {
	apiKeyUUID, err := uuid.Parse(chi.URLParam(r, "api_key_uuid"))
	if err != nil {
		util.Error(w, http.StatusBadRequest, "Invalid API key UUID")
		return
	}

	var req dto.AddAPIKeyApisRequestDto
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		util.Error(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if err := req.Validate(); err != nil {
		util.ValidationError(w, err)
		return
	}

	// Add APIs to API key
	err = h.apiKeyService.AddAPIKeyApis(apiKeyUUID, req.ApiUUIDs)
	if err != nil {
		util.Error(w, http.StatusInternalServerError, "Failed to add APIs to API key")
		return
	}

	util.Success(w, nil, "APIs added to API key successfully")
}

// Remove API from API key
func (h *APIKeyHandler) RemoveApi(w http.ResponseWriter, r *http.Request) {
	apiKeyUUID, err := uuid.Parse(chi.URLParam(r, "api_key_uuid"))
	if err != nil {
		util.Error(w, http.StatusBadRequest, "Invalid API key UUID")
		return
	}

	apiUUID, err := uuid.Parse(chi.URLParam(r, "api_uuid"))
	if err != nil {
		util.Error(w, http.StatusBadRequest, "Invalid API UUID")
		return
	}

	// Remove API from API key
	err = h.apiKeyService.RemoveAPIKeyApi(apiKeyUUID, apiUUID)
	if err != nil {
		util.Error(w, http.StatusInternalServerError, "Failed to remove API from API key")
		return
	}

	util.Success(w, nil, "API removed from API key successfully")
}

// Get permissions for a specific API assigned to API key
func (h *APIKeyHandler) GetApiPermissions(w http.ResponseWriter, r *http.Request) {
	apiKeyUUID, err := uuid.Parse(chi.URLParam(r, "api_key_uuid"))
	if err != nil {
		util.Error(w, http.StatusBadRequest, "Invalid API key UUID")
		return
	}

	apiUUID, err := uuid.Parse(chi.URLParam(r, "api_uuid"))
	if err != nil {
		util.Error(w, http.StatusBadRequest, "Invalid API UUID")
		return
	}

	// Get API key API permissions
	permissions, err := h.apiKeyService.GetAPIKeyApiPermissions(apiKeyUUID, apiUUID)
	if err != nil {
		util.Error(w, http.StatusInternalServerError, "Failed to get API key API permissions")
		return
	}

	// Convert to DTO
	permissionDtos := make([]dto.PermissionResponseDto, len(permissions))
	for i, perm := range permissions {
		permissionDtos[i] = dto.PermissionResponseDto{
			PermissionUUID: perm.PermissionUUID,
			Name:           perm.Name,
			Description:    perm.Description,
			Status:         perm.Status,
			IsDefault:      perm.IsDefault,
			IsSystem:       perm.IsSystem,
			CreatedAt:      perm.CreatedAt,
			UpdatedAt:      perm.UpdatedAt,
		}
	}

	// Wrap in structured response DTO
	response := dto.APIKeyApiPermissionsResponseDto{
		Permissions: permissionDtos,
	}

	util.Success(w, response, "API key API permissions retrieved successfully")
}

// Add permissions to a specific API for API key
func (h *APIKeyHandler) AddApiPermissions(w http.ResponseWriter, r *http.Request) {
	apiKeyUUID, err := uuid.Parse(chi.URLParam(r, "api_key_uuid"))
	if err != nil {
		util.Error(w, http.StatusBadRequest, "Invalid API key UUID")
		return
	}

	apiUUID, err := uuid.Parse(chi.URLParam(r, "api_uuid"))
	if err != nil {
		util.Error(w, http.StatusBadRequest, "Invalid API UUID")
		return
	}

	var req dto.AddAPIKeyPermissionsRequestDto
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		util.Error(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if err := req.Validate(); err != nil {
		util.ValidationError(w, err)
		return
	}

	// Add permissions to API key API
	err = h.apiKeyService.AddAPIKeyApiPermissions(apiKeyUUID, apiUUID, req.Permissions)
	if err != nil {
		util.Error(w, http.StatusInternalServerError, "Failed to add permissions to API key API")
		return
	}

	util.Success(w, nil, "Permissions added to API key API successfully")
}

// Remove permission from a specific API for API key
func (h *APIKeyHandler) RemoveApiPermission(w http.ResponseWriter, r *http.Request) {
	apiKeyUUID, err := uuid.Parse(chi.URLParam(r, "api_key_uuid"))
	if err != nil {
		util.Error(w, http.StatusBadRequest, "Invalid API key UUID")
		return
	}

	apiUUID, err := uuid.Parse(chi.URLParam(r, "api_uuid"))
	if err != nil {
		util.Error(w, http.StatusBadRequest, "Invalid API UUID")
		return
	}

	permissionUUID, err := uuid.Parse(chi.URLParam(r, "permission_uuid"))
	if err != nil {
		util.Error(w, http.StatusBadRequest, "Invalid permission UUID")
		return
	}

	// Remove permission from API key API
	err = h.apiKeyService.RemoveAPIKeyApiPermission(apiKeyUUID, apiUUID, permissionUUID)
	if err != nil {
		util.Error(w, http.StatusInternalServerError, "Failed to remove permission from API key API")
		return
	}

	util.Success(w, nil, "Permission removed from API key API successfully")
}
