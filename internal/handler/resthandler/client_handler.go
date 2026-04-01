package resthandler

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
	"github.com/maintainerd/auth/internal/service"
	"github.com/maintainerd/auth/internal/util"
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
		util.Error(w, http.StatusUnauthorized, "Tenant not found in context")
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
	reqParams := dto.ClientFilterDto{
		Name:                 util.PtrOrNil(q.Get("name")),
		DisplayName:          util.PtrOrNil(q.Get("display_name")),
		ClientType:           clientType,
		IdentityProviderUUID: util.PtrOrNil(q.Get("identity_provider_id")),
		Status:               status,
		IsDefault:            isDefault,
		IsSystem:             isSystem,
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
	result, err := h.ClientService.Get(ClientFilter)
	if err != nil {
		util.Error(w, http.StatusInternalServerError, "Failed to fetch auth clients", err.Error())
		return
	}

	// Map auth client result to DTO
	rows := make([]dto.ClientResponseDto, len(result.Data))
	for i, r := range result.Data {
		rows[i] = toClientResponseDto(r)
	}

	// Build response data
	response := dto.PaginatedResponseDto[dto.ClientResponseDto]{
		Rows:       rows,
		Total:      result.Total,
		Page:       result.Page,
		Limit:      result.Limit,
		TotalPages: result.TotalPages,
	}

	util.Success(w, response, "Auth clients fetched successfully")
}

// Get Auth client by UUID
func (h *ClientHandler) GetByUUID(w http.ResponseWriter, r *http.Request) {
	// Get tenant from context
	tenant, ok := r.Context().Value(middleware.TenantContextKey).(*model.Tenant)
	if !ok || tenant == nil {
		util.Error(w, http.StatusUnauthorized, "Tenant not found in context")
		return
	}

	ClientUUID, err := uuid.Parse(chi.URLParam(r, "client_uuid"))
	if err != nil {
		util.Error(w, http.StatusBadRequest, "Invalid Auth client UUID")
		return
	}

	Client, err := h.ClientService.GetByUUID(ClientUUID, tenant.TenantID)
	if err != nil {
		util.Error(w, http.StatusNotFound, "Auth client not found")
		return
	}

	dtoRes := toClientResponseDto(*Client)

	util.Success(w, dtoRes, "Auth client fetched successfully")
}

// Get Auth client secret by UUID
func (h *ClientHandler) GetSecretByUUID(w http.ResponseWriter, r *http.Request) {
	// Get tenant from context
	tenant, ok := r.Context().Value(middleware.TenantContextKey).(*model.Tenant)
	if !ok || tenant == nil {
		util.Error(w, http.StatusUnauthorized, "Tenant not found in context")
		return
	}

	ClientUUID, err := uuid.Parse(chi.URLParam(r, "client_uuid"))
	if err != nil {
		util.Error(w, http.StatusBadRequest, "Invalid Auth client UUID")
		return
	}

	Client, err := h.ClientService.GetSecretByUUID(ClientUUID, tenant.TenantID)
	if err != nil {
		util.Error(w, http.StatusNotFound, "Auth client not found")
		return
	}

	dtoRes := dto.ClientSecretResponseDto{
		ClientID:     Client.ClientID,
		ClientSecret: Client.ClientSecret,
	}

	util.Success(w, dtoRes, "Auth client secret fetched successfully")
}

// Get Auth client config by UUID
func (h *ClientHandler) GetConfigByUUID(w http.ResponseWriter, r *http.Request) {
	// Get tenant from context
	tenant, ok := r.Context().Value(middleware.TenantContextKey).(*model.Tenant)
	if !ok || tenant == nil {
		util.Error(w, http.StatusUnauthorized, "Tenant not found in context")
		return
	}

	ClientUUID, err := uuid.Parse(chi.URLParam(r, "client_uuid"))
	if err != nil {
		util.Error(w, http.StatusBadRequest, "Invalid Auth client UUID")
		return
	}

	ClientConfig, err := h.ClientService.GetConfigByUUID(ClientUUID, tenant.TenantID)
	if err != nil {
		util.Error(w, http.StatusNotFound, "Auth client not found")
		return
	}

	// Return config directly as data (not wrapped in DTO)
	util.Success(w, ClientConfig, "Auth client config fetched successfully")
}

// Create Auth Client
func (h *ClientHandler) Create(w http.ResponseWriter, r *http.Request) {
	// Get tenant from context
	tenant, ok := r.Context().Value(middleware.TenantContextKey).(*model.Tenant)
	if !ok || tenant == nil {
		util.Error(w, http.StatusUnauthorized, "Tenant not found in context")
		return
	}

	// Get authentication context
	user := r.Context().Value(middleware.UserContextKey).(*model.User)

	var req dto.ClientCreateRequestDto
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		util.Error(w, http.StatusBadRequest, "Invalid request", err.Error())
		return
	}

	if err := req.Validate(); err != nil {
		util.ValidationError(w, err)
		return
	}

	Client, err := h.ClientService.Create(tenant.TenantID, req.Name, req.DisplayName, req.ClientType, req.Domain, req.Config, req.Status, false, req.IdentityProviderUUID, user.UserUUID)
	if err != nil {
		util.Error(w, http.StatusInternalServerError, "Failed to create auth client", err.Error())
		return
	}

	dtoRes := toClientResponseDto(*Client)

	util.Created(w, dtoRes, "Auth client created successfully")
}

// Update Auth Client
func (h *ClientHandler) Update(w http.ResponseWriter, r *http.Request) {
	// Get tenant from context
	tenant, ok := r.Context().Value(middleware.TenantContextKey).(*model.Tenant)
	if !ok || tenant == nil {
		util.Error(w, http.StatusUnauthorized, "Tenant not found in context")
		return
	}

	// Get authentication context
	user := r.Context().Value(middleware.UserContextKey).(*model.User)

	ClientUUID, err := uuid.Parse(chi.URLParam(r, "client_uuid"))
	if err != nil {
		util.Error(w, http.StatusBadRequest, "Invalid auth client UUID")
		return
	}

	var req dto.ClientUpdateRequestDto
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		util.Error(w, http.StatusBadRequest, "Invalid request", err.Error())
		return
	}

	if err := req.Validate(); err != nil {
		util.ValidationError(w, err)
		return
	}

	Client, err := h.ClientService.Update(ClientUUID, tenant.TenantID, req.Name, req.DisplayName, req.ClientType, req.Domain, req.Config, req.Status, false, user.UserUUID)
	if err != nil {
		util.Error(w, http.StatusInternalServerError, "Failed to update auth client", err.Error())
		return
	}

	dtoRes := toClientResponseDto(*Client)

	util.Success(w, dtoRes, "Auth client updated successfully")
}

// Set Auth client status
func (h *ClientHandler) SetStatus(w http.ResponseWriter, r *http.Request) {
	// Get tenant from context
	tenant, ok := r.Context().Value(middleware.TenantContextKey).(*model.Tenant)
	if !ok || tenant == nil {
		util.Error(w, http.StatusUnauthorized, "Tenant not found in context")
		return
	}

	// Get authentication context
	user := r.Context().Value(middleware.UserContextKey).(*model.User)

	ClientUUID, err := uuid.Parse(chi.URLParam(r, "client_uuid"))
	if err != nil {
		util.Error(w, http.StatusBadRequest, "Invalid auth client UUID")
		return
	}

	// Toggle status between active and inactive
	newStatus := model.StatusActive
	// We need to get current status first to toggle it
	currentClient, err := h.ClientService.GetByUUID(ClientUUID, tenant.TenantID)
	if err != nil {
		util.Error(w, http.StatusNotFound, "Auth client not found")
		return
	}
	if currentClient.Status == model.StatusActive {
		newStatus = model.StatusInactive
	}

	Client, err := h.ClientService.SetStatusByUUID(ClientUUID, tenant.TenantID, newStatus, user.UserUUID)
	if err != nil {
		util.Error(w, http.StatusInternalServerError, "Failed to update API", err.Error())
		return
	}

	dtoRes := toClientResponseDto(*Client)

	util.Success(w, dtoRes, "Auth client status updated successfully")
}

// Delete Auth Client
func (h *ClientHandler) Delete(w http.ResponseWriter, r *http.Request) {
	// Get tenant from context
	tenant, ok := r.Context().Value(middleware.TenantContextKey).(*model.Tenant)
	if !ok || tenant == nil {
		util.Error(w, http.StatusUnauthorized, "Tenant not found in context")
		return
	}

	// Get authentication context
	user := r.Context().Value(middleware.UserContextKey).(*model.User)

	ClientUUID, err := uuid.Parse(chi.URLParam(r, "client_uuid"))
	if err != nil {
		util.Error(w, http.StatusBadRequest, "Invalid Auth Client UUID")
		return
	}

	Client, err := h.ClientService.DeleteByUUID(ClientUUID, tenant.TenantID, user.UserUUID)
	if err != nil {
		util.Error(w, http.StatusInternalServerError, "Failed to delete auth client", err.Error())
		return
	}

	dtoRes := toClientResponseDto(*Client)

	util.Success(w, dtoRes, "Auth client deleted successfully")
}

func (h *ClientHandler) GetURIs(w http.ResponseWriter, r *http.Request) {
	// Get tenant from context
	tenant, ok := r.Context().Value(middleware.TenantContextKey).(*model.Tenant)
	if !ok || tenant == nil {
		util.Error(w, http.StatusUnauthorized, "Tenant not found in context")
		return
	}

	ClientUUID, err := uuid.Parse(chi.URLParam(r, "client_uuid"))
	if err != nil {
		util.Error(w, http.StatusBadRequest, "Invalid auth client UUID")
		return
	}

	Client, err := h.ClientService.GetByUUID(ClientUUID, tenant.TenantID)
	if err != nil {
		util.Error(w, http.StatusNotFound, "Auth client not found", err.Error())
		return
	}

	// Convert URIs to response format
	var uris []dto.ClientURIResponseDto
	if Client.ClientURIs != nil {
		uris = make([]dto.ClientURIResponseDto, len(*Client.ClientURIs))
		for i, uri := range *Client.ClientURIs {
			uris[i] = dto.ClientURIResponseDto{
				ClientURIUUID: uri.ClientURIUUID,
				URI:           uri.URI,
				Type:          uri.Type,
				CreatedAt:     uri.CreatedAt,
				UpdatedAt:     uri.UpdatedAt,
			}
		}
	}

	response := dto.ClientURIsResponseDto{
		URIs: uris,
	}

	util.Success(w, response, "URIs retrieved successfully")
}

func (h *ClientHandler) CreateURI(w http.ResponseWriter, r *http.Request) {
	// Get tenant from context
	tenant, ok := r.Context().Value(middleware.TenantContextKey).(*model.Tenant)
	if !ok || tenant == nil {
		util.Error(w, http.StatusUnauthorized, "Tenant not found in context")
		return
	}

	// Get authentication context
	user := r.Context().Value(middleware.UserContextKey).(*model.User)

	ClientUUID, err := uuid.Parse(chi.URLParam(r, "client_uuid"))
	if err != nil {
		util.Error(w, http.StatusBadRequest, "Invalid auth client UUID")
		return
	}

	var req dto.ClientURICreateOrUpdateRequestDto
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		util.Error(w, http.StatusBadRequest, "Invalid request", err.Error())
		return
	}

	if err := req.Validate(); err != nil {
		util.ValidationError(w, err)
		return
	}

	uri, err := h.ClientService.CreateURI(ClientUUID, tenant.TenantID, req.URI, req.Type, user.UserUUID)
	if err != nil {
		util.Error(w, http.StatusInternalServerError, "Failed to create URI", err.Error())
		return
	}

	dtoRes := dto.ClientURIResponseDto{
		ClientURIUUID: (*uri.ClientURIs)[0].ClientURIUUID,
		URI:           (*uri.ClientURIs)[0].URI,
		Type:          (*uri.ClientURIs)[0].Type,
		CreatedAt:     (*uri.ClientURIs)[0].CreatedAt,
		UpdatedAt:     (*uri.ClientURIs)[0].UpdatedAt,
	}

	util.Created(w, dtoRes, "URI created successfully")
}

func (h *ClientHandler) UpdateURI(w http.ResponseWriter, r *http.Request) {
	// Get tenant from context
	tenant, ok := r.Context().Value(middleware.TenantContextKey).(*model.Tenant)
	if !ok || tenant == nil {
		util.Error(w, http.StatusUnauthorized, "Tenant not found in context")
		return
	}

	// Get authentication context
	user := r.Context().Value(middleware.UserContextKey).(*model.User)

	ClientUUID, err := uuid.Parse(chi.URLParam(r, "client_uuid"))
	if err != nil {
		util.Error(w, http.StatusBadRequest, "Invalid auth client UUID")
		return
	}

	ClientURIUUID, err := uuid.Parse(chi.URLParam(r, "client_uri_uuid"))
	if err != nil {
		util.Error(w, http.StatusBadRequest, "Invalid auth client URI UUID")
		return
	}

	var req dto.ClientURICreateOrUpdateRequestDto
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		util.Error(w, http.StatusBadRequest, "Invalid request", err.Error())
		return
	}

	if err := req.Validate(); err != nil {
		util.ValidationError(w, err)
		return
	}

	uri, err := h.ClientService.UpdateURI(ClientUUID, tenant.TenantID, ClientURIUUID, req.URI, req.Type, user.UserUUID)
	if err != nil {
		util.Error(w, http.StatusInternalServerError, "Failed to update URI", err.Error())
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
		util.Error(w, http.StatusInternalServerError, "Updated URI not found in response")
		return
	}

	dtoRes := dto.ClientURIResponseDto{
		ClientURIUUID: updatedURI.ClientURIUUID,
		URI:           updatedURI.URI,
		Type:          updatedURI.Type,
		CreatedAt:     updatedURI.CreatedAt,
		UpdatedAt:     updatedURI.UpdatedAt,
	}

	util.Success(w, dtoRes, "URI updated successfully")
}

func (h *ClientHandler) DeleteURI(w http.ResponseWriter, r *http.Request) {
	// Get tenant from context
	tenant, ok := r.Context().Value(middleware.TenantContextKey).(*model.Tenant)
	if !ok || tenant == nil {
		util.Error(w, http.StatusUnauthorized, "Tenant not found in context")
		return
	}

	// Get authentication context
	user := r.Context().Value(middleware.UserContextKey).(*model.User)

	ClientUUID, err := uuid.Parse(chi.URLParam(r, "client_uuid"))
	if err != nil {
		util.Error(w, http.StatusBadRequest, "Invalid auth client UUID")
		return
	}

	ClientURIUUID, err := uuid.Parse(chi.URLParam(r, "client_uri_uuid"))
	if err != nil {
		util.Error(w, http.StatusBadRequest, "Invalid auth client URI UUID")
		return
	}

	Client, err := h.ClientService.DeleteURI(ClientUUID, tenant.TenantID, ClientURIUUID, user.UserUUID)
	if err != nil {
		util.Error(w, http.StatusInternalServerError, "Failed to delete URI", err.Error())
		return
	}

	dtoRes := toClientResponseDto(*Client)

	util.Success(w, dtoRes, "URI deleted successfully")
}

// GetAPIs retrieves APIs assigned to auth client.
func (h *ClientHandler) GetAPIs(w http.ResponseWriter, r *http.Request) {
	// Get authentication context
	// Get tenant from context
	tenant, ok := r.Context().Value(middleware.TenantContextKey).(*model.Tenant)
	if !ok || tenant == nil {
		util.Error(w, http.StatusUnauthorized, "Tenant not found in context")
		return
	}

	ClientUUID, err := uuid.Parse(chi.URLParam(r, "client_uuid"))
	if err != nil {
		util.Error(w, http.StatusBadRequest, "Invalid auth client UUID")
		return
	}

	// Get auth client APIs
	ClientApis, err := h.ClientService.GetClientApis(tenant.TenantID, ClientUUID)
	if err != nil {
		util.Error(w, http.StatusInternalServerError, "Failed to get auth client APIs")
		return
	}

	// Convert to DTO
	apis := make([]dto.ClientAPIResponseDto, len(ClientApis))
	for i, api := range ClientApis {
		// Convert API service data to DTO
		apiDto := dto.APIResponseDto{
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

		apis[i] = dto.ClientAPIResponseDto{
			ClientAPIUUID: api.ClientApiUUID,
			API:           apiDto,
			Permissions:   permissions,
			CreatedAt:     api.CreatedAt,
		}
	}

	response := dto.ClientAPIsResponseDto{
		APIs: apis,
	}

	util.Success(w, response, "Auth client APIs retrieved successfully")
}

// AddAPIs adds APIs to auth client.
func (h *ClientHandler) AddAPIs(w http.ResponseWriter, r *http.Request) {
	// Get authentication context
	ClientUUID, err := uuid.Parse(chi.URLParam(r, "client_uuid"))
	if err != nil {
		util.Error(w, http.StatusBadRequest, "Invalid auth client UUID")
		return
	}

	var req dto.AddClientAPIsRequestDto
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		util.Error(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Get tenant from context
	tenant, ok := r.Context().Value(middleware.TenantContextKey).(*model.Tenant)
	if !ok || tenant == nil {
		util.Error(w, http.StatusUnauthorized, "Tenant not found in context")
		return
	}

	// Add APIs to auth client
	err = h.ClientService.AddClientApis(tenant.TenantID, ClientUUID, req.APIUUIDs)
	if err != nil {
		util.Error(w, http.StatusInternalServerError, "Failed to add APIs to auth client")
		return
	}

	response := dto.SuccessResponseDto{
		Message: "APIs added to auth client successfully",
	}

	util.Success(w, response, "APIs added to auth client successfully")
}

// RemoveAPI removes an API from auth client.
func (h *ClientHandler) RemoveAPI(w http.ResponseWriter, r *http.Request) {
	ClientUUID, err := uuid.Parse(chi.URLParam(r, "client_uuid"))
	if err != nil {
		util.Error(w, http.StatusBadRequest, "Invalid auth client UUID")
		return
	}

	apiUUID, err := uuid.Parse(chi.URLParam(r, "api_uuid"))
	if err != nil {
		util.Error(w, http.StatusBadRequest, "Invalid API UUID")
		return
	}

	// Get tenant from context
	tenant, ok := r.Context().Value(middleware.TenantContextKey).(*model.Tenant)
	if !ok || tenant == nil {
		util.Error(w, http.StatusUnauthorized, "Tenant not found in context")
		return
	}

	// Remove API from auth client
	err = h.ClientService.RemoveClientApi(tenant.TenantID, ClientUUID, apiUUID)
	if err != nil {
		util.Error(w, http.StatusInternalServerError, "Failed to remove API from auth client")
		return
	}

	response := dto.SuccessResponseDto{
		Message: "API removed from auth client successfully",
	}

	util.Success(w, response, "API removed from auth client successfully")
}

// GetAPIPermissions retrieves permissions for a specific API assigned to auth client.
func (h *ClientHandler) GetAPIPermissions(w http.ResponseWriter, r *http.Request) {
	ClientUUID, err := uuid.Parse(chi.URLParam(r, "client_uuid"))
	if err != nil {
		util.Error(w, http.StatusBadRequest, "Invalid auth client UUID")
		return
	}

	apiUUID, err := uuid.Parse(chi.URLParam(r, "api_uuid"))
	if err != nil {
		util.Error(w, http.StatusBadRequest, "Invalid API UUID")
		return
	}

	// Get tenant from context
	tenant, ok := r.Context().Value(middleware.TenantContextKey).(*model.Tenant)
	if !ok || tenant == nil {
		util.Error(w, http.StatusUnauthorized, "Tenant not found in context")
		return
	}

	// Get auth client API permissions
	permissions, err := h.ClientService.GetClientApiPermissions(tenant.TenantID, ClientUUID, apiUUID)
	if err != nil {
		util.Error(w, http.StatusInternalServerError, "Failed to get auth client API permissions")
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

	response := dto.ClientAPIPermissionsResponseDto{
		Permissions: permissionDtos,
	}

	util.Success(w, response, "Auth client API permissions retrieved successfully")
}

// AddAPIPermissions adds permissions to a specific API for auth client.
func (h *ClientHandler) AddAPIPermissions(w http.ResponseWriter, r *http.Request) {
	// Get authentication context
	ClientUUID, err := uuid.Parse(chi.URLParam(r, "client_uuid"))
	if err != nil {
		util.Error(w, http.StatusBadRequest, "Invalid auth client UUID")
		return
	}

	apiUUID, err := uuid.Parse(chi.URLParam(r, "api_uuid"))
	if err != nil {
		util.Error(w, http.StatusBadRequest, "Invalid API UUID")
		return
	}

	var req dto.AddClientAPIPermissionsRequestDto
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		util.Error(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Get tenant from context
	tenant, ok := r.Context().Value(middleware.TenantContextKey).(*model.Tenant)
	if !ok || tenant == nil {
		util.Error(w, http.StatusUnauthorized, "Tenant not found in context")
		return
	}

	// Add permissions to auth client API
	err = h.ClientService.AddClientApiPermissions(tenant.TenantID, ClientUUID, apiUUID, req.PermissionUUIDs)
	if err != nil {
		util.Error(w, http.StatusInternalServerError, "Failed to add permissions to auth client API")
		return
	}

	response := dto.SuccessResponseDto{
		Message: "Permissions added to auth client API successfully",
	}

	util.Success(w, response, "Permissions added to auth client API successfully")
}

// RemoveAPIPermission removes a permission from a specific API for auth client.
func (h *ClientHandler) RemoveAPIPermission(w http.ResponseWriter, r *http.Request) {
	// Get authentication context
	ClientUUID, err := uuid.Parse(chi.URLParam(r, "client_uuid"))
	if err != nil {
		util.Error(w, http.StatusBadRequest, "Invalid auth client UUID")
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

	// Get tenant from context
	tenant, ok := r.Context().Value(middleware.TenantContextKey).(*model.Tenant)
	if !ok || tenant == nil {
		util.Error(w, http.StatusUnauthorized, "Tenant not found in context")
		return
	}

	// Remove permission from auth client API
	err = h.ClientService.RemoveClientApiPermission(tenant.TenantID, ClientUUID, apiUUID, permissionUUID)
	if err != nil {
		util.Error(w, http.StatusInternalServerError, "Failed to remove permission from auth client API")
		return
	}

	response := dto.SuccessResponseDto{
		Message: "Permission removed from auth client API successfully",
	}

	util.Success(w, response, "Permission removed from auth client API successfully")
}

// Convert result to DTO
func toClientResponseDto(r service.ClientServiceDataResult) dto.ClientResponseDto {
	result := dto.ClientResponseDto{
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
		result.IdentityProvider = &dto.IdentityProviderResponseDto{
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
		result.URIs = make([]dto.ClientURIResponseDto, len(*r.ClientURIs))
		for i, uri := range *r.ClientURIs {
			result.URIs[i] = dto.ClientURIResponseDto{
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
		permissions := make([]dto.PermissionResponseDto, len(*r.Permissions))
		for i, permission := range *r.Permissions {
			permissions[i] = dto.PermissionResponseDto{
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
