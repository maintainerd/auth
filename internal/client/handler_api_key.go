package client

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/maintainerd/maintainerd-auth/internal/platform/middleware"
	"github.com/maintainerd/maintainerd-auth/internal/platform/pagination"
	resp "github.com/maintainerd/maintainerd-auth/internal/platform/response"
	"github.com/maintainerd/maintainerd-auth/internal/shared"
)

type APIKeyHandler struct {
	apiKeyService APIKeyService
}

func NewAPIKeyHandler(apiKeyService APIKeyService) *APIKeyHandler {
	return &APIKeyHandler{
		apiKeyService: apiKeyService,
	}
}

// Get API keys with pagination and filtering
func (h *APIKeyHandler) Get(w http.ResponseWriter, r *http.Request) {
	// Get authentication context
	requestingUser := middleware.AuthFromRequest(r).User

	// Get tenant from context
	tenant := middleware.AuthFromRequest(r).Tenant
	if tenant == nil {
		resp.Error(w, http.StatusUnauthorized, "Tenant not found in context")
		return
	}

	// Parse query parameters
	var reqParams APIKeyGetRequestDTO
	reqParams.PaginationRequestDTO = pagination.ParseQuery(r)

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
	serviceFilter := APIKeyServiceGetFilter{
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
	var dtoResults []APIKeyResponseDTO
	for _, apiKey := range result.Data {
		dtoResult := toAPIKeyResponseDTO(apiKey)
		dtoResults = append(dtoResults, dtoResult)
	}

	// Build paginated response
	response := PaginatedResponseDTO[APIKeyResponseDTO]{
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
	requestingUser := middleware.AuthFromRequest(r).User

	// Get tenant from context
	tenant := middleware.AuthFromRequest(r).Tenant
	if tenant == nil {
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
	tenant := middleware.AuthFromRequest(r).Tenant
	if tenant == nil {
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
	// Get tenant from context
	tenant := middleware.AuthFromRequest(r).Tenant
	if tenant == nil {
		resp.Error(w, http.StatusUnauthorized, "Tenant not found in context")
		return
	}

	var req APIKeyCreateRequestDTO
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		resp.BadRequestBody(w)
		return
	}

	if err := req.Validate(); err != nil {
		resp.ValidationError(w, err)
		return
	}

	// Set default status if not provided
	status := shared.StatusActive
	if req.Status != "" {
		status = req.Status
	}

	apiKey, plainKey, err := h.apiKeyService.Create(r.Context(), tenant.TenantID, req.Name, req.Description, req.Config, req.ExpiresAt, status)
	if err != nil {
		resp.HandleServiceError(w, r, "Failed to create API key", err)
		return
	}

	// Build response data using flat DTO structure
	response := APIKeyCreateResponseDTO{
		APIKeyID:    apiKey.APIKeyUUID,
		Name:        apiKey.Name,
		Description: apiKey.Description,
		KeyPrefix:   apiKey.KeyPrefix,
		Key:         plainKey, // This is the actual API key that should be stored securely
		ExpiresAt:   apiKey.ExpiresAt,
		Status:      apiKey.Status,
		CreatedAt:   apiKey.CreatedAt,
		UpdatedAt:   apiKey.UpdatedAt,
	}

	resp.Created(w, response, "API key created successfully")
}

// Update API key
func (h *APIKeyHandler) Update(w http.ResponseWriter, r *http.Request) {
	// Get authentication context
	updaterUser := middleware.AuthFromRequest(r).User

	// Get tenant from context
	tenant := middleware.AuthFromRequest(r).Tenant
	if tenant == nil {
		resp.Error(w, http.StatusUnauthorized, "Tenant not found in context")
		return
	}

	apiKeyUUIDStr := chi.URLParam(r, "api_key_uuid")
	apiKeyUUID, err := uuid.Parse(apiKeyUUIDStr)
	if err != nil {
		resp.Error(w, http.StatusBadRequest, "Invalid API key UUID")
		return
	}

	var req APIKeyUpdateRequestDTO
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		resp.BadRequestBody(w)
		return
	}

	if err := req.Validate(); err != nil {
		resp.ValidationError(w, err)
		return
	}

	apiKey, err := h.apiKeyService.Update(r.Context(), apiKeyUUID, tenant.TenantID, req.Name, req.Description, req.Config, req.ExpiresAt, req.Status, updaterUser.UserUUID)
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
	tenant := middleware.AuthFromRequest(r).Tenant
	if tenant == nil {
		resp.Error(w, http.StatusUnauthorized, "Tenant not found in context")
		return
	}

	apiKeyUUID, err := uuid.Parse(chi.URLParam(r, "api_key_uuid"))
	if err != nil {
		resp.Error(w, http.StatusBadRequest, "Invalid API key UUID")
		return
	}

	// Parse request body
	var req APIKeyStatusUpdateDTO
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		resp.BadRequestBody(w)
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
	deleterUser := middleware.AuthFromRequest(r).User

	// Get tenant from context
	tenant := middleware.AuthFromRequest(r).Tenant
	if tenant == nil {
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
func toAPIKeyResponseDTO(r APIKeyServiceDataResult) APIKeyResponseDTO {
	result := APIKeyResponseDTO{
		APIKeyID:    r.APIKeyUUID,
		Name:        r.Name,
		Description: r.Description,
		KeyPrefix:   r.KeyPrefix,
		ExpiresAt:   r.ExpiresAt,
		Status:      r.Status,
		CreatedAt:   r.CreatedAt,
		UpdatedAt:   r.UpdatedAt,
	}

	// User and Tenant relationships removed

	return result
}

// GetAPIs retrieves APIs assigned to API key with pagination.
func (h *APIKeyHandler) GetAPIs(w http.ResponseWriter, r *http.Request) {
	tenant := middleware.AuthFromRequest(r).Tenant
	if tenant == nil {
		resp.Error(w, http.StatusUnauthorized, "Tenant not found in context")
		return
	}

	apiKeyUUID, err := uuid.Parse(chi.URLParam(r, "api_key_uuid"))
	if err != nil {
		resp.Error(w, http.StatusBadRequest, "Invalid API key UUID")
		return
	}

	// Build request DTO
	reqParams := APIKeyAPIsGetRequestDTO{
		PaginationRequestDTO: pagination.ParseQuery(r),
	}

	if err := reqParams.Validate(); err != nil {
		resp.ValidationError(w, err)
		return
	}

	// Get API key APIs with pagination
	result, err := h.apiKeyService.GetAPIKeyAPIs(r.Context(), tenant.TenantID, apiKeyUUID, reqParams.Page, reqParams.Limit, reqParams.SortBy, reqParams.SortOrder)
	if err != nil {
		resp.HandleServiceError(w, r, "Failed to get API key APIs", err)
		return
	}

	// Convert to DTO - return just the API objects
	apiDtos := make([]APIResponseDTO, len(result.Data))
	for i, api := range result.Data {
		// Convert API directly
		apiDtos[i] = APIResponseDTO{
			APIUUID:     api.Api.APIUUID,
			Name:        api.Api.Name,
			DisplayName: api.Api.DisplayName,
			Description: api.Api.Description,
			Identifier:  api.Api.Identifier,
			Status:      api.Api.Status,
			IsSystem:    api.Api.IsSystem,
			CreatedAt:   api.Api.CreatedAt,
			UpdatedAt:   api.Api.UpdatedAt,
		}
	}

	// Build paginated response
	response := PaginatedResponseDTO[APIResponseDTO]{
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
	tenant := middleware.AuthFromRequest(r).Tenant
	if tenant == nil {
		resp.Error(w, http.StatusUnauthorized, "Tenant not found in context")
		return
	}

	apiKeyUUID, err := uuid.Parse(chi.URLParam(r, "api_key_uuid"))
	if err != nil {
		resp.Error(w, http.StatusBadRequest, "Invalid API key UUID")
		return
	}

	var req AddAPIKeyAPIsRequestDTO
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		resp.BadRequestBody(w)
		return
	}

	if err := req.Validate(); err != nil {
		resp.ValidationError(w, err)
		return
	}

	// Add APIs to API key
	err = h.apiKeyService.AddAPIKeyAPIs(r.Context(), tenant.TenantID, apiKeyUUID, req.APIUUIDs)
	if err != nil {
		resp.HandleServiceError(w, r, "Failed to add APIs to API key", err)
		return
	}

	resp.Success(w, nil, "APIs added to API key successfully")
}

// RemoveAPI removes an API from API key.
func (h *APIKeyHandler) RemoveAPI(w http.ResponseWriter, r *http.Request) {
	tenant := middleware.AuthFromRequest(r).Tenant
	if tenant == nil {
		resp.Error(w, http.StatusUnauthorized, "Tenant not found in context")
		return
	}

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
	err = h.apiKeyService.RemoveAPIKeyAPI(r.Context(), tenant.TenantID, apiKeyUUID, apiUUID)
	if err != nil {
		resp.HandleServiceError(w, r, "Failed to remove API from API key", err)
		return
	}

	resp.Success(w, nil, "API removed from API key successfully")
}

// GetAPIPermissions retrieves permissions for a specific API assigned to API key.
func (h *APIKeyHandler) GetAPIPermissions(w http.ResponseWriter, r *http.Request) {
	tenant := middleware.AuthFromRequest(r).Tenant
	if tenant == nil {
		resp.Error(w, http.StatusUnauthorized, "Tenant not found in context")
		return
	}

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
	permissions, err := h.apiKeyService.GetAPIKeyAPIPermissions(r.Context(), tenant.TenantID, apiKeyUUID, apiUUID)
	if err != nil {
		resp.HandleServiceError(w, r, "Failed to get API key API permissions", err)
		return
	}

	// Convert to DTO
	permissionDtos := make([]PermissionResponseDTO, len(permissions))
	for i, perm := range permissions {
		permissionDtos[i] = PermissionResponseDTO{
			PermissionUUID: perm.PermissionUUID,
			Name:           perm.Name,
			Description:    perm.Description,
			Status:         perm.Status,
			IsSystem:       perm.IsSystem,
			CreatedAt:      perm.CreatedAt,
			UpdatedAt:      perm.UpdatedAt,
		}
	}

	// Wrap in structured response DTO
	response := APIKeyAPIPermissionsResponseDTO{
		Permissions: permissionDtos,
	}

	resp.Success(w, response, "API key API permissions retrieved successfully")
}

// AddAPIPermissions adds permissions to a specific API for API key.
func (h *APIKeyHandler) AddAPIPermissions(w http.ResponseWriter, r *http.Request) {
	tenant := middleware.AuthFromRequest(r).Tenant
	if tenant == nil {
		resp.Error(w, http.StatusUnauthorized, "Tenant not found in context")
		return
	}

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

	var req AddAPIKeyPermissionsRequestDTO
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		resp.BadRequestBody(w)
		return
	}

	if err := req.Validate(); err != nil {
		resp.ValidationError(w, err)
		return
	}

	// Add permissions to API key API
	err = h.apiKeyService.AddAPIKeyAPIPermissions(r.Context(), tenant.TenantID, apiKeyUUID, apiUUID, req.PermissionUUIDs)
	if err != nil {
		resp.HandleServiceError(w, r, "Failed to add permissions to API key API", err)
		return
	}

	resp.Success(w, nil, "Permissions added to API key API successfully")
}

// RemoveAPIPermission removes a permission from a specific API for API key.
func (h *APIKeyHandler) RemoveAPIPermission(w http.ResponseWriter, r *http.Request) {
	tenant := middleware.AuthFromRequest(r).Tenant
	if tenant == nil {
		resp.Error(w, http.StatusUnauthorized, "Tenant not found in context")
		return
	}

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
	err = h.apiKeyService.RemoveAPIKeyAPIPermission(r.Context(), tenant.TenantID, apiKeyUUID, apiUUID, permissionUUID)
	if err != nil {
		resp.HandleServiceError(w, r, "Failed to remove permission from API key API", err)
		return
	}

	resp.Success(w, nil, "Permission removed from API key API successfully")
}
