package handler

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/dto"
	"github.com/maintainerd/auth/internal/middleware"
	"github.com/maintainerd/auth/internal/model"
	"github.com/maintainerd/auth/internal/ptr"
	resp "github.com/maintainerd/auth/internal/rest/response"
	"github.com/maintainerd/auth/internal/service"
)

type ClientHandler struct {
	ClientService service.ClientService
}

func NewClientHandler(ClientService service.ClientService) *ClientHandler {
	return &ClientHandler{ClientService}
}

// Get all auth clients with pagination
func (h *ClientHandler) Get(w http.ResponseWriter, r *http.Request) {
	// Get tenant from context
	tenant, ok := r.Context().Value(middleware.TenantContextKey).(*model.Tenant)
	if !ok || tenant == nil {
		resp.Error(w, http.StatusUnauthorized, "Tenant not found in context")
		return
	}

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

	// Parse status array
	var status []string
	if v := q.Get("status"); v != "" {
		status = strings.Split(v, ",")
		// Trim whitespace from each value
		for i, s := range status {
			status[i] = strings.TrimSpace(s)
		}
	}

	// Parse client_type array
	var clientType []string
	if v := q.Get("client_type"); v != "" {
		clientType = strings.Split(v, ",")
		// Trim whitespace from each value
		for i, ct := range clientType {
			clientType[i] = strings.TrimSpace(ct)
		}
	}

	// Build request DTO
	reqParams := dto.ClientFilterDTO{
		Name:                 ptr.PtrOrNil(q.Get("name")),
		DisplayName:          ptr.PtrOrNil(q.Get("display_name")),
		ClientType:           clientType,
		IdentityProviderUUID: ptr.PtrOrNil(q.Get("identity_provider_id")),
		Status:               status,
		IsDefault:            isDefault,
		IsSystem:             isSystem,
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

	// Build service filter
	ClientFilter := service.ClientServiceGetFilter{
		TenantID:             tenant.TenantID,
		Name:                 reqParams.Name,
		DisplayName:          reqParams.DisplayName,
		ClientType:           reqParams.ClientType,
		IdentityProviderUUID: reqParams.IdentityProviderUUID,
		Status:               reqParams.Status,
		IsDefault:            reqParams.IsDefault,
		IsSystem:             reqParams.IsSystem,
		Page:                 reqParams.Page,
		Limit:                reqParams.Limit,
		SortBy:               reqParams.SortBy,
		SortOrder:            reqParams.SortOrder,
	}

	// Fetch Auth Clients
	result, err := h.ClientService.Get(r.Context(), ClientFilter)
	if err != nil {
		resp.HandleServiceError(w, r, "Failed to fetch auth clients", err)
		return
	}

	// Map auth client result to DTO
	rows := make([]dto.ClientResponseDTO, len(result.Data))
	for i, r := range result.Data {
		rows[i] = toClientResponseDTO(r)
	}

	// Build response data
	response := dto.PaginatedResponseDTO[dto.ClientResponseDTO]{
		Rows:       rows,
		Total:      result.Total,
		Page:       result.Page,
		Limit:      result.Limit,
		TotalPages: result.TotalPages,
	}

	resp.Success(w, response, "Auth clients fetched successfully")
}

// Get Auth client by UUID
func (h *ClientHandler) GetByUUID(w http.ResponseWriter, r *http.Request) {
	// Get tenant from context
	tenant, ok := r.Context().Value(middleware.TenantContextKey).(*model.Tenant)
	if !ok || tenant == nil {
		resp.Error(w, http.StatusUnauthorized, "Tenant not found in context")
		return
	}

	ClientUUID, err := uuid.Parse(chi.URLParam(r, "client_uuid"))
	if err != nil {
		resp.Error(w, http.StatusBadRequest, "Invalid Auth client UUID")
		return
	}

	Client, err := h.ClientService.GetByUUID(r.Context(), ClientUUID, tenant.TenantID)
	if err != nil {
		resp.HandleServiceError(w, r, "Auth client not found", err)
		return
	}

	dtoRes := toClientResponseDTO(*Client)

	resp.Success(w, dtoRes, "Auth client fetched successfully")
}

// Get Auth client secret by UUID
func (h *ClientHandler) GetSecretByUUID(w http.ResponseWriter, r *http.Request) {
	// Get tenant from context
	tenant, ok := r.Context().Value(middleware.TenantContextKey).(*model.Tenant)
	if !ok || tenant == nil {
		resp.Error(w, http.StatusUnauthorized, "Tenant not found in context")
		return
	}

	ClientUUID, err := uuid.Parse(chi.URLParam(r, "client_uuid"))
	if err != nil {
		resp.Error(w, http.StatusBadRequest, "Invalid Auth client UUID")
		return
	}

	Client, err := h.ClientService.GetSecretByUUID(r.Context(), ClientUUID, tenant.TenantID)
	if err != nil {
		resp.HandleServiceError(w, r, "Auth client not found", err)
		return
	}

	dtoRes := dto.ClientSecretResponseDTO{
		ClientID:     Client.ClientID,
		ClientSecret: Client.ClientSecret,
	}

	resp.Success(w, dtoRes, "Auth client secret fetched successfully")
}

// Get Auth client config by UUID
func (h *ClientHandler) GetConfigByUUID(w http.ResponseWriter, r *http.Request) {
	// Get tenant from context
	tenant, ok := r.Context().Value(middleware.TenantContextKey).(*model.Tenant)
	if !ok || tenant == nil {
		resp.Error(w, http.StatusUnauthorized, "Tenant not found in context")
		return
	}

	ClientUUID, err := uuid.Parse(chi.URLParam(r, "client_uuid"))
	if err != nil {
		resp.Error(w, http.StatusBadRequest, "Invalid Auth client UUID")
		return
	}

	ClientConfig, err := h.ClientService.GetConfigByUUID(r.Context(), ClientUUID, tenant.TenantID)
	if err != nil {
		resp.HandleServiceError(w, r, "Auth client not found", err)
		return
	}

	// Return config directly as data (not wrapped in DTO)
	resp.Success(w, ClientConfig, "Auth client config fetched successfully")
}

// Create Auth Client
func (h *ClientHandler) Create(w http.ResponseWriter, r *http.Request) {
	// Get tenant from context
	tenant, ok := r.Context().Value(middleware.TenantContextKey).(*model.Tenant)
	if !ok || tenant == nil {
		resp.Error(w, http.StatusUnauthorized, "Tenant not found in context")
		return
	}

	// Get authentication context
	user := r.Context().Value(middleware.UserContextKey).(*model.User)

	var req dto.ClientCreateRequestDTO
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		resp.Error(w, http.StatusBadRequest, "Invalid request")
		return
	}

	if err := req.Validate(); err != nil {
		resp.ValidationError(w, err)
		return
	}

	Client, err := h.ClientService.Create(r.Context(), tenant.TenantID, req.Name, req.DisplayName, req.ClientType, req.Domain, req.Config, req.Status, false, req.IdentityProviderUUID, user.UserUUID)
	if err != nil {
		resp.HandleServiceError(w, r, "Failed to create auth client", err)
		return
	}

	dtoRes := toClientResponseDTO(*Client)

	resp.Created(w, dtoRes, "Auth client created successfully")
}

// Update Auth Client
func (h *ClientHandler) Update(w http.ResponseWriter, r *http.Request) {
	// Get tenant from context
	tenant, ok := r.Context().Value(middleware.TenantContextKey).(*model.Tenant)
	if !ok || tenant == nil {
		resp.Error(w, http.StatusUnauthorized, "Tenant not found in context")
		return
	}

	// Get authentication context
	user := r.Context().Value(middleware.UserContextKey).(*model.User)

	ClientUUID, err := uuid.Parse(chi.URLParam(r, "client_uuid"))
	if err != nil {
		resp.Error(w, http.StatusBadRequest, "Invalid auth client UUID")
		return
	}

	var req dto.ClientUpdateRequestDTO
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		resp.Error(w, http.StatusBadRequest, "Invalid request")
		return
	}

	if err := req.Validate(); err != nil {
		resp.ValidationError(w, err)
		return
	}

	Client, err := h.ClientService.Update(r.Context(), ClientUUID, tenant.TenantID, req.Name, req.DisplayName, req.ClientType, req.Domain, req.Config, req.Status, false, user.UserUUID)
	if err != nil {
		resp.HandleServiceError(w, r, "Failed to update auth client", err)
		return
	}

	dtoRes := toClientResponseDTO(*Client)

	resp.Success(w, dtoRes, "Auth client updated successfully")
}

// Set Auth client status
func (h *ClientHandler) SetStatus(w http.ResponseWriter, r *http.Request) {
	// Get tenant from context
	tenant, ok := r.Context().Value(middleware.TenantContextKey).(*model.Tenant)
	if !ok || tenant == nil {
		resp.Error(w, http.StatusUnauthorized, "Tenant not found in context")
		return
	}

	// Get authentication context
	user := r.Context().Value(middleware.UserContextKey).(*model.User)

	ClientUUID, err := uuid.Parse(chi.URLParam(r, "client_uuid"))
	if err != nil {
		resp.Error(w, http.StatusBadRequest, "Invalid auth client UUID")
		return
	}

	// Toggle status between active and inactive
	newStatus := model.StatusActive
	// We need to get current status first to toggle it
	currentClient, err := h.ClientService.GetByUUID(r.Context(), ClientUUID, tenant.TenantID)
	if err != nil {
		resp.HandleServiceError(w, r, "Auth client not found", err)
		return
	}
	if currentClient.Status == model.StatusActive {
		newStatus = model.StatusInactive
	}

	Client, err := h.ClientService.SetStatusByUUID(r.Context(), ClientUUID, tenant.TenantID, newStatus, user.UserUUID)
	if err != nil {
		resp.HandleServiceError(w, r, "Failed to update auth client status", err)
		return
	}

	dtoRes := toClientResponseDTO(*Client)

	resp.Success(w, dtoRes, "Auth client status updated successfully")
}

// Delete Auth Client
func (h *ClientHandler) Delete(w http.ResponseWriter, r *http.Request) {
	// Get tenant from context
	tenant, ok := r.Context().Value(middleware.TenantContextKey).(*model.Tenant)
	if !ok || tenant == nil {
		resp.Error(w, http.StatusUnauthorized, "Tenant not found in context")
		return
	}

	// Get authentication context
	user := r.Context().Value(middleware.UserContextKey).(*model.User)

	ClientUUID, err := uuid.Parse(chi.URLParam(r, "client_uuid"))
	if err != nil {
		resp.Error(w, http.StatusBadRequest, "Invalid Auth Client UUID")
		return
	}

	Client, err := h.ClientService.DeleteByUUID(r.Context(), ClientUUID, tenant.TenantID, user.UserUUID)
	if err != nil {
		resp.HandleServiceError(w, r, "Failed to delete auth client", err)
		return
	}

	dtoRes := toClientResponseDTO(*Client)

	resp.Success(w, dtoRes, "Auth client deleted successfully")
}

func (h *ClientHandler) GetURIs(w http.ResponseWriter, r *http.Request) {
	// Get tenant from context
	tenant, ok := r.Context().Value(middleware.TenantContextKey).(*model.Tenant)
	if !ok || tenant == nil {
		resp.Error(w, http.StatusUnauthorized, "Tenant not found in context")
		return
	}

	ClientUUID, err := uuid.Parse(chi.URLParam(r, "client_uuid"))
	if err != nil {
		resp.Error(w, http.StatusBadRequest, "Invalid auth client UUID")
		return
	}

	Client, err := h.ClientService.GetByUUID(r.Context(), ClientUUID, tenant.TenantID)
	if err != nil {
		resp.HandleServiceError(w, r, "Auth client not found", err)
		return
	}

	// Convert URIs to response format
	var uris []dto.ClientURIResponseDTO
	if Client.ClientURIs != nil {
		uris = make([]dto.ClientURIResponseDTO, len(*Client.ClientURIs))
		for i, uri := range *Client.ClientURIs {
			uris[i] = dto.ClientURIResponseDTO{
				ClientURIUUID: uri.ClientURIUUID,
				URI:           uri.URI,
				Type:          uri.Type,
				CreatedAt:     uri.CreatedAt,
				UpdatedAt:     uri.UpdatedAt,
			}
		}
	}

	response := dto.ClientURIsResponseDTO{
		URIs: uris,
	}

	resp.Success(w, response, "URIs retrieved successfully")
}

func (h *ClientHandler) CreateURI(w http.ResponseWriter, r *http.Request) {
	// Get tenant from context
	tenant, ok := r.Context().Value(middleware.TenantContextKey).(*model.Tenant)
	if !ok || tenant == nil {
		resp.Error(w, http.StatusUnauthorized, "Tenant not found in context")
		return
	}

	// Get authentication context
	user := r.Context().Value(middleware.UserContextKey).(*model.User)

	ClientUUID, err := uuid.Parse(chi.URLParam(r, "client_uuid"))
	if err != nil {
		resp.Error(w, http.StatusBadRequest, "Invalid auth client UUID")
		return
	}

	var req dto.ClientURICreateOrUpdateRequestDTO
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		resp.Error(w, http.StatusBadRequest, "Invalid request")
		return
	}

	if err := req.Validate(); err != nil {
		resp.ValidationError(w, err)
		return
	}

	uri, err := h.ClientService.CreateURI(r.Context(), ClientUUID, tenant.TenantID, req.URI, req.Type, user.UserUUID)
	if err != nil {
		resp.HandleServiceError(w, r, "Failed to create URI", err)
		return
	}

	dtoRes := dto.ClientURIResponseDTO{
		ClientURIUUID: (*uri.ClientURIs)[0].ClientURIUUID,
		URI:           (*uri.ClientURIs)[0].URI,
		Type:          (*uri.ClientURIs)[0].Type,
		CreatedAt:     (*uri.ClientURIs)[0].CreatedAt,
		UpdatedAt:     (*uri.ClientURIs)[0].UpdatedAt,
	}

	resp.Created(w, dtoRes, "URI created successfully")
}

func (h *ClientHandler) UpdateURI(w http.ResponseWriter, r *http.Request) {
	// Get tenant from context
	tenant, ok := r.Context().Value(middleware.TenantContextKey).(*model.Tenant)
	if !ok || tenant == nil {
		resp.Error(w, http.StatusUnauthorized, "Tenant not found in context")
		return
	}

	// Get authentication context
	user := r.Context().Value(middleware.UserContextKey).(*model.User)

	ClientUUID, err := uuid.Parse(chi.URLParam(r, "client_uuid"))
	if err != nil {
		resp.Error(w, http.StatusBadRequest, "Invalid auth client UUID")
		return
	}

	ClientURIUUID, err := uuid.Parse(chi.URLParam(r, "client_uri_uuid"))
	if err != nil {
		resp.Error(w, http.StatusBadRequest, "Invalid auth client URI UUID")
		return
	}

	var req dto.ClientURICreateOrUpdateRequestDTO
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		resp.Error(w, http.StatusBadRequest, "Invalid request")
		return
	}

	if err := req.Validate(); err != nil {
		resp.ValidationError(w, err)
		return
	}

	uri, err := h.ClientService.UpdateURI(r.Context(), ClientUUID, tenant.TenantID, ClientURIUUID, req.URI, req.Type, user.UserUUID)
	if err != nil {
		resp.HandleServiceError(w, r, "Failed to update URI", err)
		return
	}

	// Find the updated URI in the response
	var updatedURI *service.ClientURIServiceDataResult
	if uri.ClientURIs != nil {
		for _, u := range *uri.ClientURIs {
			if u.ClientURIUUID == ClientURIUUID {
				updatedURI = &u
				break
			}
		}
	}

	if updatedURI == nil {
		resp.Error(w, http.StatusInternalServerError, "Updated URI not found in response")
		return
	}

	dtoRes := dto.ClientURIResponseDTO{
		ClientURIUUID: updatedURI.ClientURIUUID,
		URI:           updatedURI.URI,
		Type:          updatedURI.Type,
		CreatedAt:     updatedURI.CreatedAt,
		UpdatedAt:     updatedURI.UpdatedAt,
	}

	resp.Success(w, dtoRes, "URI updated successfully")
}

func (h *ClientHandler) DeleteURI(w http.ResponseWriter, r *http.Request) {
	// Get tenant from context
	tenant, ok := r.Context().Value(middleware.TenantContextKey).(*model.Tenant)
	if !ok || tenant == nil {
		resp.Error(w, http.StatusUnauthorized, "Tenant not found in context")
		return
	}

	// Get authentication context
	user := r.Context().Value(middleware.UserContextKey).(*model.User)

	ClientUUID, err := uuid.Parse(chi.URLParam(r, "client_uuid"))
	if err != nil {
		resp.Error(w, http.StatusBadRequest, "Invalid auth client UUID")
		return
	}

	ClientURIUUID, err := uuid.Parse(chi.URLParam(r, "client_uri_uuid"))
	if err != nil {
		resp.Error(w, http.StatusBadRequest, "Invalid auth client URI UUID")
		return
	}

	Client, err := h.ClientService.DeleteURI(r.Context(), ClientUUID, tenant.TenantID, ClientURIUUID, user.UserUUID)
	if err != nil {
		resp.HandleServiceError(w, r, "Failed to delete URI", err)
		return
	}

	dtoRes := toClientResponseDTO(*Client)

	resp.Success(w, dtoRes, "URI deleted successfully")
}

// GetAPIs retrieves APIs assigned to auth client.
func (h *ClientHandler) GetAPIs(w http.ResponseWriter, r *http.Request) {
	// Get authentication context
	// Get tenant from context
	tenant, ok := r.Context().Value(middleware.TenantContextKey).(*model.Tenant)
	if !ok || tenant == nil {
		resp.Error(w, http.StatusUnauthorized, "Tenant not found in context")
		return
	}

	ClientUUID, err := uuid.Parse(chi.URLParam(r, "client_uuid"))
	if err != nil {
		resp.Error(w, http.StatusBadRequest, "Invalid auth client UUID")
		return
	}

	// Get auth client APIs
	ClientAPIs, err := h.ClientService.GetClientAPIs(r.Context(), tenant.TenantID, ClientUUID)
	if err != nil {
		resp.HandleServiceError(w, r, "Failed to get auth client APIs", err)
		return
	}

	// Convert to DTO
	apis := make([]dto.ClientAPIResponseDTO, len(ClientAPIs))
	for i, api := range ClientAPIs {
		// Convert API service data to DTO
		apiDTO := dto.APIResponseDTO{
			APIUUID:     api.Api.APIUUID,
			Name:        api.Api.Name,
			DisplayName: api.Api.DisplayName,
			Description: api.Api.Description,
			Status:      api.Api.Status,
			IsSystem:    api.Api.IsSystem,
			CreatedAt:   api.Api.CreatedAt,
			UpdatedAt:   api.Api.UpdatedAt,
		}

		// Convert permissions service data to DTO
		permissions := make([]dto.PermissionResponseDTO, len(api.Permissions))
		for j, perm := range api.Permissions {
			permissions[j] = dto.PermissionResponseDTO{
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

		apis[i] = dto.ClientAPIResponseDTO{
			ClientAPIUUID: api.ClientAPIUUID,
			API:           apiDTO,
			Permissions:   permissions,
			CreatedAt:     api.CreatedAt,
		}
	}

	response := dto.ClientAPIsResponseDTO{
		APIs: apis,
	}

	resp.Success(w, response, "Auth client APIs retrieved successfully")
}

// AddAPIs adds APIs to auth client.
func (h *ClientHandler) AddAPIs(w http.ResponseWriter, r *http.Request) {
	// Get authentication context
	ClientUUID, err := uuid.Parse(chi.URLParam(r, "client_uuid"))
	if err != nil {
		resp.Error(w, http.StatusBadRequest, "Invalid auth client UUID")
		return
	}

	var req dto.AddClientAPIsRequestDTO
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		resp.Error(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Get tenant from context
	tenant, ok := r.Context().Value(middleware.TenantContextKey).(*model.Tenant)
	if !ok || tenant == nil {
		resp.Error(w, http.StatusUnauthorized, "Tenant not found in context")
		return
	}

	// Add APIs to auth client
	err = h.ClientService.AddClientAPIs(r.Context(), tenant.TenantID, ClientUUID, req.APIUUIDs)
	if err != nil {
		resp.HandleServiceError(w, r, "Failed to add APIs to auth client", err)
		return
	}

	response := dto.SuccessResponseDTO{
		Message: "APIs added to auth client successfully",
	}

	resp.Success(w, response, "APIs added to auth client successfully")
}

// RemoveAPI removes an API from auth client.
func (h *ClientHandler) RemoveAPI(w http.ResponseWriter, r *http.Request) {
	ClientUUID, err := uuid.Parse(chi.URLParam(r, "client_uuid"))
	if err != nil {
		resp.Error(w, http.StatusBadRequest, "Invalid auth client UUID")
		return
	}

	apiUUID, err := uuid.Parse(chi.URLParam(r, "api_uuid"))
	if err != nil {
		resp.Error(w, http.StatusBadRequest, "Invalid API UUID")
		return
	}

	// Get tenant from context
	tenant, ok := r.Context().Value(middleware.TenantContextKey).(*model.Tenant)
	if !ok || tenant == nil {
		resp.Error(w, http.StatusUnauthorized, "Tenant not found in context")
		return
	}

	// Remove API from auth client
	err = h.ClientService.RemoveClientAPI(r.Context(), tenant.TenantID, ClientUUID, apiUUID)
	if err != nil {
		resp.HandleServiceError(w, r, "Failed to remove API from auth client", err)
		return
	}

	response := dto.SuccessResponseDTO{
		Message: "API removed from auth client successfully",
	}

	resp.Success(w, response, "API removed from auth client successfully")
}

// GetAPIPermissions retrieves permissions for a specific API assigned to auth client.
func (h *ClientHandler) GetAPIPermissions(w http.ResponseWriter, r *http.Request) {
	ClientUUID, err := uuid.Parse(chi.URLParam(r, "client_uuid"))
	if err != nil {
		resp.Error(w, http.StatusBadRequest, "Invalid auth client UUID")
		return
	}

	apiUUID, err := uuid.Parse(chi.URLParam(r, "api_uuid"))
	if err != nil {
		resp.Error(w, http.StatusBadRequest, "Invalid API UUID")
		return
	}

	// Get tenant from context
	tenant, ok := r.Context().Value(middleware.TenantContextKey).(*model.Tenant)
	if !ok || tenant == nil {
		resp.Error(w, http.StatusUnauthorized, "Tenant not found in context")
		return
	}

	// Get auth client API permissions
	permissions, err := h.ClientService.GetClientAPIPermissions(r.Context(), tenant.TenantID, ClientUUID, apiUUID)
	if err != nil {
		resp.HandleServiceError(w, r, "Failed to get auth client API permissions", err)
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

	response := dto.ClientAPIPermissionsResponseDTO{
		Permissions: permissionDtos,
	}

	resp.Success(w, response, "Auth client API permissions retrieved successfully")
}

// AddAPIPermissions adds permissions to a specific API for auth client.
func (h *ClientHandler) AddAPIPermissions(w http.ResponseWriter, r *http.Request) {
	// Get authentication context
	ClientUUID, err := uuid.Parse(chi.URLParam(r, "client_uuid"))
	if err != nil {
		resp.Error(w, http.StatusBadRequest, "Invalid auth client UUID")
		return
	}

	apiUUID, err := uuid.Parse(chi.URLParam(r, "api_uuid"))
	if err != nil {
		resp.Error(w, http.StatusBadRequest, "Invalid API UUID")
		return
	}

	var req dto.AddClientAPIPermissionsRequestDTO
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		resp.Error(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Get tenant from context
	tenant, ok := r.Context().Value(middleware.TenantContextKey).(*model.Tenant)
	if !ok || tenant == nil {
		resp.Error(w, http.StatusUnauthorized, "Tenant not found in context")
		return
	}

	// Add permissions to auth client API
	err = h.ClientService.AddClientAPIPermissions(r.Context(), tenant.TenantID, ClientUUID, apiUUID, req.PermissionUUIDs)
	if err != nil {
		resp.HandleServiceError(w, r, "Failed to add permissions to auth client API", err)
		return
	}

	response := dto.SuccessResponseDTO{
		Message: "Permissions added to auth client API successfully",
	}

	resp.Success(w, response, "Permissions added to auth client API successfully")
}

// RemoveAPIPermission removes a permission from a specific API for auth client.
func (h *ClientHandler) RemoveAPIPermission(w http.ResponseWriter, r *http.Request) {
	// Get authentication context
	ClientUUID, err := uuid.Parse(chi.URLParam(r, "client_uuid"))
	if err != nil {
		resp.Error(w, http.StatusBadRequest, "Invalid auth client UUID")
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

	// Get tenant from context
	tenant, ok := r.Context().Value(middleware.TenantContextKey).(*model.Tenant)
	if !ok || tenant == nil {
		resp.Error(w, http.StatusUnauthorized, "Tenant not found in context")
		return
	}

	// Remove permission from auth client API
	err = h.ClientService.RemoveClientAPIPermission(r.Context(), tenant.TenantID, ClientUUID, apiUUID, permissionUUID)
	if err != nil {
		resp.HandleServiceError(w, r, "Failed to remove permission from auth client API", err)
		return
	}

	response := dto.SuccessResponseDTO{
		Message: "Permission removed from auth client API successfully",
	}

	resp.Success(w, response, "Permission removed from auth client API successfully")
}

// Convert result to DTO
func toClientResponseDTO(r service.ClientServiceDataResult) dto.ClientResponseDTO {
	result := dto.ClientResponseDTO{
		ClientUUID:  r.ClientUUID,
		Name:        r.Name,
		DisplayName: r.DisplayName,
		ClientType:  r.ClientType,
		Domain:      r.Domain,
		Status:      r.Status,
		IsDefault:   r.IsDefault,
		IsSystem:    r.IsSystem,
		CreatedAt:   r.CreatedAt,
		UpdatedAt:   r.UpdatedAt,
	}

	if r.IdentityProvider != nil {
		result.IdentityProvider = &dto.IdentityProviderResponseDTO{
			IdentityProviderUUID: r.IdentityProvider.IdentityProviderUUID,
			Name:                 r.IdentityProvider.Name,
			DisplayName:          r.IdentityProvider.DisplayName,
			Provider:             r.IdentityProvider.Provider,
			ProviderType:         r.IdentityProvider.ProviderType,
			Identifier:           r.IdentityProvider.Identifier,
			Status:               r.IdentityProvider.Status,
			IsDefault:            r.IdentityProvider.IsDefault,
			IsSystem:             r.IdentityProvider.IsSystem,
			CreatedAt:            r.IdentityProvider.CreatedAt,
			UpdatedAt:            r.IdentityProvider.UpdatedAt,
		}
	}

	if r.ClientURIs != nil && len(*r.ClientURIs) > 0 {
		result.URIs = make([]dto.ClientURIResponseDTO, len(*r.ClientURIs))
		for i, uri := range *r.ClientURIs {
			result.URIs[i] = dto.ClientURIResponseDTO{
				ClientURIUUID: uri.ClientURIUUID,
				URI:           uri.URI,
				Type:          uri.Type,
				CreatedAt:     uri.CreatedAt,
				UpdatedAt:     uri.UpdatedAt,
			}
		}
	}

	// Map Permissions if present
	if r.Permissions != nil {
		permissions := make([]dto.PermissionResponseDTO, len(*r.Permissions))
		for i, permission := range *r.Permissions {
			permissions[i] = dto.PermissionResponseDTO{
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
