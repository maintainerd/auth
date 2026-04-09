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
	resp "github.com/maintainerd/auth/internal/rest/response"
	"github.com/maintainerd/auth/internal/service"
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

	// Get tenant from context
	tenant, ok := r.Context().Value(middleware.TenantContextKey).(*model.Tenant)
	if !ok || tenant == nil {
		resp.Error(w, http.StatusUnauthorized, "Tenant not found in context")
		return
	}

	// Parse query parameters
	var reqParams dto.APIKeyGetRequestDTO
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
		resp.ValidationError(w, err)
		return
	}

	// Build service filter
	serviceFilter := service.APIKeyServiceGetFilter{
		TenantID:    tenant.TenantID,
		Name:        reqParams.Name,
		Description: reqParams.Description,
		Status:      reqParams.Status,
		Page:        reqParams.Page,
		Limit:       reqParams.Limit,
		SortBy:      reqParams.SortBy,
		SortOrder:   reqParams.SortOrder,
	}

	// Fetch API keys
	result, err := h.apiKeyService.Get(r.Context(), serviceFilter, requestingUser.UserUUID)
	if err != nil {
		resp.HandleServiceError(w, r, "Failed to fetch API keys", err)
		return
	}

	// Convert to DTOs
	var dtoResults []dto.APIKeyResponseDTO
	for _, apiKey := range result.Data {
		dtoResult := toAPIKeyResponseDTO(apiKey)
		dtoResults = append(dtoResults, dtoResult)
	}

	// Build paginated response
	response := dto.PaginatedResponseDTO[dto.APIKeyResponseDTO]{
		Rows:       dtoResults,
		Total:      result.Total,
		Page:       result.Page,
		Limit:      result.Limit,
		TotalPages: result.TotalPages,
	}

	resp.Success(w, response, "API keys fetched successfully")
}

// Get API key by UUID
func (h *APIKeyHandler) GetByUUID(w http.ResponseWriter, r *http.Request) {
	// Get authentication context
	requestingUser := r.Context().Value(middleware.UserContextKey).(*model.User)

	// Get tenant from context
	tenant, ok := r.Context().Value(middleware.TenantContextKey).(*model.Tenant)
	if !ok || tenant == nil {
		resp.Error(w, http.StatusUnauthorized, "Tenant not found in context")
		return
	}

	apiKeyUUIDStr := chi.URLParam(r, "api_key_uuid")
	apiKeyUUID, err := uuid.Parse(apiKeyUUIDStr)
	if err != nil {
		resp.Error(w, http.StatusBadRequest, "Invalid API key UUID")
		return
	}

	apiKey, err := h.apiKeyService.GetByUUID(r.Context(), apiKeyUUID, tenant.TenantID, requestingUser.UserUUID)
	if err != nil {
		resp.HandleServiceError(w, r, "API key not found", err)
		return
	}

	// Build response data
	dtoRes := toAPIKeyResponseDTO(*apiKey)

	resp.Success(w, dtoRes, "API key fetched successfully")
}

// Get API key config by UUID
func (h *APIKeyHandler) GetConfigByUUID(w http.ResponseWriter, r *http.Request) {
	// Get tenant from context
	tenant, ok := r.Context().Value(middleware.TenantContextKey).(*model.Tenant)
	if !ok || tenant == nil {
		resp.Error(w, http.StatusUnauthorized, "Tenant not found in context")
		return
	}

	apiKeyUUID, err := uuid.Parse(chi.URLParam(r, "api_key_uuid"))
	if err != nil {
		resp.Error(w, http.StatusBadRequest, "Invalid API key UUID")
		return
	}

	apiKeyConfig, err := h.apiKeyService.GetConfigByUUID(r.Context(), apiKeyUUID, tenant.TenantID)
	if err != nil {
		resp.HandleServiceError(w, r, "API key not found", err)
		return
	}

	// Return config directly as data (not wrapped in DTO)
	resp.Success(w, apiKeyConfig, "API key config fetched successfully")
}

// Create API key
func (h *APIKeyHandler) Create(w http.ResponseWriter, r *http.Request) {
	// Get authentication context (not needed for API key creation anymore)
	_ = r.Context().Value(middleware.UserContextKey).(*model.User)

	// Get tenant from context
	tenant, ok := r.Context().Value(middleware.TenantContextKey).(*model.Tenant)
	if !ok || tenant == nil {
		resp.Error(w, http.StatusUnauthorized, "Tenant not found in context")
		return
	}

	var req dto.APIKeyCreateRequestDTO
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		resp.Error(w, http.StatusBadRequest, "Invalid request")
		return
	}

	if err := req.Validate(); err != nil {
		resp.ValidationError(w, err)
		return
	}

	// Set default status if not provided
	status := model.StatusActive
	if req.Status != "" {
		status = req.Status
	}

	apiKey, plainKey, err := h.apiKeyService.Create(r.Context(), tenant.TenantID, req.Name, req.Description, req.Config, req.ExpiresAt, req.RateLimit, status)
	if err != nil {
		resp.HandleServiceError(w, r, "Failed to create API key", err)
		return
	}

	// Build response data using flat DTO structure
	response := dto.APIKeyCreateResponseDTO{
		APIKeyID:    apiKey.APIKeyUUID,
		Name:        apiKey.Name,
		Description: apiKey.Description,
		KeyPrefix:   apiKey.KeyPrefix,
		Key:         plainKey, // This is the actual API key that should be stored securely
		ExpiresAt:   apiKey.ExpiresAt,
		RateLimit:   apiKey.RateLimit,
		Status:      apiKey.Status,
		CreatedAt:   apiKey.CreatedAt,
		UpdatedAt:   apiKey.UpdatedAt,
	}

	resp.Created(w, response, "API key created successfully")
}

// Update API key
func (h *APIKeyHandler) Update(w http.ResponseWriter, r *http.Request) {
	// Get authentication context
	updaterUser := r.Context().Value(middleware.UserContextKey).(*model.User)

	// Get tenant from context
	tenant, ok := r.Context().Value(middleware.TenantContextKey).(*model.Tenant)
	if !ok || tenant == nil {
		resp.Error(w, http.StatusUnauthorized, "Tenant not found in context")
		return
	}

	apiKeyUUIDStr := chi.URLParam(r, "api_key_uuid")
	apiKeyUUID, err := uuid.Parse(apiKeyUUIDStr)
	if err != nil {
		resp.Error(w, http.StatusBadRequest, "Invalid API key UUID")
		return
	}

	var req dto.APIKeyUpdateRequestDTO
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		resp.Error(w, http.StatusBadRequest, "Invalid request")
		return
	}

	if err := req.Validate(); err != nil {
		resp.ValidationError(w, err)
		return
	}

	apiKey, err := h.apiKeyService.Update(r.Context(), apiKeyUUID, tenant.TenantID, req.Name, req.Description, req.Config, req.ExpiresAt, req.RateLimit, req.Status, updaterUser.UserUUID)
	if err != nil {
		resp.HandleServiceError(w, r, "Failed to update API key", err)
		return
	}

	// Build response data
	dtoRes := toAPIKeyResponseDTO(*apiKey)

	resp.Success(w, dtoRes, "API key updated successfully")
}

// Set API key status
func (h *APIKeyHandler) SetStatus(w http.ResponseWriter, r *http.Request) {
	// Get tenant from context
	tenant, ok := r.Context().Value(middleware.TenantContextKey).(*model.Tenant)
	if !ok || tenant == nil {
		resp.Error(w, http.StatusUnauthorized, "Tenant not found in context")
		return
	}

	apiKeyUUID, err := uuid.Parse(chi.URLParam(r, "api_key_uuid"))
	if err != nil {
		resp.Error(w, http.StatusBadRequest, "Invalid API key UUID")
		return
	}

	// Parse request body
	var req dto.APIKeyStatusUpdateDTO
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		resp.Error(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Validate request
	if err := req.Validate(); err != nil {
		resp.ValidationError(w, err)
		return
	}

	apiKey, err := h.apiKeyService.SetStatusByUUID(r.Context(), apiKeyUUID, tenant.TenantID, req.Status)
	if err != nil {
		resp.HandleServiceError(w, r, "Failed to update API key status", err)
		return
	}

	response := toAPIKeyResponseDTO(*apiKey)

	resp.Success(w, response, "API key status updated successfully")
}

// Delete API key
func (h *APIKeyHandler) Delete(w http.ResponseWriter, r *http.Request) {
	// Get authentication context
	deleterUser := r.Context().Value(middleware.UserContextKey).(*model.User)

	// Get tenant from context
	tenant, ok := r.Context().Value(middleware.TenantContextKey).(*model.Tenant)
	if !ok || tenant == nil {
		resp.Error(w, http.StatusUnauthorized, "Tenant not found in context")
		return
	}

	apiKeyUUIDStr := chi.URLParam(r, "api_key_uuid")
	apiKeyUUID, err := uuid.Parse(apiKeyUUIDStr)
	if err != nil {
		resp.Error(w, http.StatusBadRequest, "Invalid API key UUID")
		return
	}

	apiKey, err := h.apiKeyService.Delete(r.Context(), apiKeyUUID, tenant.TenantID, deleterUser.UserUUID)
	if err != nil {
		resp.HandleServiceError(w, r, "Failed to delete API key", err)
		return
	}

	// Build response data
	dtoRes := toAPIKeyResponseDTO(*apiKey)

	resp.Success(w, dtoRes, "API key deleted successfully")
}

// Convert service result to DTO
func toAPIKeyResponseDTO(r service.APIKeyServiceDataResult) dto.APIKeyResponseDTO {
	result := dto.APIKeyResponseDTO{
		APIKeyID:    r.APIKeyUUID,
		Name:        r.Name,
		Description: r.Description,
		KeyPrefix:   r.KeyPrefix,
		ExpiresAt:   r.ExpiresAt,
		RateLimit:   r.RateLimit,
		Status:      r.Status,
		CreatedAt:   r.CreatedAt,
		UpdatedAt:   r.UpdatedAt,
	}

	// User and Tenant relationships removed

	return result
}

// GetAPIs retrieves APIs assigned to API key with pagination.
func (h *APIKeyHandler) GetAPIs(w http.ResponseWriter, r *http.Request) {
	apiKeyUUID, err := uuid.Parse(chi.URLParam(r, "api_key_uuid"))
	if err != nil {
		resp.Error(w, http.StatusBadRequest, "Invalid API key UUID")
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
	reqParams := dto.APIKeyAPIsGetRequestDTO{
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

	// Get API key APIs with pagination
	result, err := h.apiKeyService.GetAPIKeyAPIs(r.Context(), apiKeyUUID, reqParams.Page, reqParams.Limit, reqParams.SortBy, reqParams.SortOrder)
	if err != nil {
		resp.HandleServiceError(w, r, "Failed to get API key APIs", err)
		return
	}

	// Convert to DTO - return just the API objects
	apiDtos := make([]dto.APIResponseDTO, len(result.Data))
	for i, api := range result.Data {
		// Convert API directly
		apiDtos[i] = dto.APIResponseDTO{
			APIUUID:     api.Api.APIUUID,
			Name:        api.Api.Name,
			DisplayName: api.Api.DisplayName,
			Description: api.Api.Description,
			APIType:     api.Api.APIType,
			Identifier:  api.Api.Identifier,
			Status:      api.Api.Status,
			IsSystem:    api.Api.IsSystem,
			CreatedAt:   api.Api.CreatedAt,
			UpdatedAt:   api.Api.UpdatedAt,
		}
	}

	// Build paginated response
	response := dto.PaginatedResponseDTO[dto.APIResponseDTO]{
		Rows:       apiDtos,
		Total:      result.Total,
		Page:       result.Page,
		Limit:      result.Limit,
		TotalPages: result.TotalPages,
	}

	resp.Success(w, response, "API key APIs retrieved successfully")
}

// AddAPIs adds APIs to API key.
func (h *APIKeyHandler) AddAPIs(w http.ResponseWriter, r *http.Request) {
	apiKeyUUID, err := uuid.Parse(chi.URLParam(r, "api_key_uuid"))
	if err != nil {
		resp.Error(w, http.StatusBadRequest, "Invalid API key UUID")
		return
	}

	var req dto.AddAPIKeyAPIsRequestDTO
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		resp.Error(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if err := req.Validate(); err != nil {
		resp.ValidationError(w, err)
		return
	}

	// Add APIs to API key
	err = h.apiKeyService.AddAPIKeyAPIs(r.Context(), apiKeyUUID, req.APIUUIDs)
	if err != nil {
		resp.HandleServiceError(w, r, "Failed to add APIs to API key", err)
		return
	}

	resp.Success(w, nil, "APIs added to API key successfully")
}

// RemoveAPI removes an API from API key.
func (h *APIKeyHandler) RemoveAPI(w http.ResponseWriter, r *http.Request) {
	apiKeyUUID, err := uuid.Parse(chi.URLParam(r, "api_key_uuid"))
	if err != nil {
		resp.Error(w, http.StatusBadRequest, "Invalid API key UUID")
		return
	}

	apiUUID, err := uuid.Parse(chi.URLParam(r, "api_uuid"))
	if err != nil {
		resp.Error(w, http.StatusBadRequest, "Invalid API UUID")
		return
	}

	// Remove API from API key
	err = h.apiKeyService.RemoveAPIKeyAPI(r.Context(), apiKeyUUID, apiUUID)
	if err != nil {
		resp.HandleServiceError(w, r, "Failed to remove API from API key", err)
		return
	}

	resp.Success(w, nil, "API removed from API key successfully")
}

// GetAPIPermissions retrieves permissions for a specific API assigned to API key.
func (h *APIKeyHandler) GetAPIPermissions(w http.ResponseWriter, r *http.Request) {
	apiKeyUUID, err := uuid.Parse(chi.URLParam(r, "api_key_uuid"))
	if err != nil {
		resp.Error(w, http.StatusBadRequest, "Invalid API key UUID")
		return
	}

	apiUUID, err := uuid.Parse(chi.URLParam(r, "api_uuid"))
	if err != nil {
		resp.Error(w, http.StatusBadRequest, "Invalid API UUID")
		return
	}

	// Get API key API permissions
	permissions, err := h.apiKeyService.GetAPIKeyAPIPermissions(r.Context(), apiKeyUUID, apiUUID)
	if err != nil {
		resp.HandleServiceError(w, r, "Failed to get API key API permissions", err)
		return
	}

	// Convert to DTO
	permissionDtos := make([]dto.PermissionResponseDTO, len(permissions))
	for i, perm := range permissions {
		permissionDtos[i] = dto.PermissionResponseDTO{
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
	response := dto.APIKeyAPIPermissionsResponseDTO{
		Permissions: permissionDtos,
	}

	resp.Success(w, response, "API key API permissions retrieved successfully")
}

// AddAPIPermissions adds permissions to a specific API for API key.
func (h *APIKeyHandler) AddAPIPermissions(w http.ResponseWriter, r *http.Request) {
	apiKeyUUID, err := uuid.Parse(chi.URLParam(r, "api_key_uuid"))
	if err != nil {
		resp.Error(w, http.StatusBadRequest, "Invalid API key UUID")
		return
	}

	apiUUID, err := uuid.Parse(chi.URLParam(r, "api_uuid"))
	if err != nil {
		resp.Error(w, http.StatusBadRequest, "Invalid API UUID")
		return
	}

	var req dto.AddAPIKeyPermissionsRequestDTO
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		resp.Error(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if err := req.Validate(); err != nil {
		resp.ValidationError(w, err)
		return
	}

	// Add permissions to API key API
	err = h.apiKeyService.AddAPIKeyAPIPermissions(r.Context(), apiKeyUUID, apiUUID, req.PermissionUUIDs)
	if err != nil {
		resp.HandleServiceError(w, r, "Failed to add permissions to API key API", err)
		return
	}

	resp.Success(w, nil, "Permissions added to API key API successfully")
}

// RemoveAPIPermission removes a permission from a specific API for API key.
func (h *APIKeyHandler) RemoveAPIPermission(w http.ResponseWriter, r *http.Request) {
	apiKeyUUID, err := uuid.Parse(chi.URLParam(r, "api_key_uuid"))
	if err != nil {
		resp.Error(w, http.StatusBadRequest, "Invalid API key UUID")
		return
	}

	apiUUID, err := uuid.Parse(chi.URLParam(r, "api_uuid"))
	if err != nil {
		resp.Error(w, http.StatusBadRequest, "Invalid API UUID")
		return
	}

	permissionUUID, err := uuid.Parse(chi.URLParam(r, "permission_uuid"))
	if err != nil {
		resp.Error(w, http.StatusBadRequest, "Invalid permission UUID")
		return
	}

	// Remove permission from API key API
	err = h.apiKeyService.RemoveAPIKeyAPIPermission(r.Context(), apiKeyUUID, apiUUID, permissionUUID)
	if err != nil {
		resp.HandleServiceError(w, r, "Failed to remove permission from API key API", err)
		return
	}

	resp.Success(w, nil, "Permission removed from API key API successfully")
}
