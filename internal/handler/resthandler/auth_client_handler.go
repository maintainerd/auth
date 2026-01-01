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

type AuthClientHandler struct {
	authClientService service.AuthClientService
}

func NewAuthClientHandler(authClientService service.AuthClientService) *AuthClientHandler {
	return &AuthClientHandler{authClientService}
}

// Get all auth clients with pagination
func (h *AuthClientHandler) Get(w http.ResponseWriter, r *http.Request) {
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
	reqParams := dto.AuthClientFilterDto{
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
	authClientFilter := service.AuthClientServiceGetFilter{
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
	result, err := h.authClientService.Get(authClientFilter)
	if err != nil {
		util.Error(w, http.StatusInternalServerError, "Failed to fetch auth clients", err.Error())
		return
	}

	// Map auth client result to DTO
	rows := make([]dto.AuthClientResponseDto, len(result.Data))
	for i, r := range result.Data {
		rows[i] = toAuthClientResponseDto(r)
	}

	// Build response data
	response := dto.PaginatedResponseDto[dto.AuthClientResponseDto]{
		Rows:       rows,
		Total:      result.Total,
		Page:       result.Page,
		Limit:      result.Limit,
		TotalPages: result.TotalPages,
	}

	util.Success(w, response, "Auth clients fetched successfully")
}

// Get Auth client by UUID
func (h *AuthClientHandler) GetByUUID(w http.ResponseWriter, r *http.Request) {
	// Get tenant from context
	tenant, ok := r.Context().Value(middleware.TenantContextKey).(*model.Tenant)
	if !ok || tenant == nil {
		util.Error(w, http.StatusUnauthorized, "Tenant not found in context")
		return
	}

	authClientUUID, err := uuid.Parse(chi.URLParam(r, "auth_client_uuid"))
	if err != nil {
		util.Error(w, http.StatusBadRequest, "Invalid Auth client UUID")
		return
	}

	authClient, err := h.authClientService.GetByUUID(authClientUUID, tenant.TenantID)
	if err != nil {
		util.Error(w, http.StatusNotFound, "Auth client not found")
		return
	}

	dtoRes := toAuthClientResponseDto(*authClient)

	util.Success(w, dtoRes, "Auth client fetched successfully")
}

// Get Auth client secret by UUID
func (h *AuthClientHandler) GetSecretByUUID(w http.ResponseWriter, r *http.Request) {
	// Get tenant from context
	tenant, ok := r.Context().Value(middleware.TenantContextKey).(*model.Tenant)
	if !ok || tenant == nil {
		util.Error(w, http.StatusUnauthorized, "Tenant not found in context")
		return
	}

	authClientUUID, err := uuid.Parse(chi.URLParam(r, "auth_client_uuid"))
	if err != nil {
		util.Error(w, http.StatusBadRequest, "Invalid Auth client UUID")
		return
	}

	authClient, err := h.authClientService.GetSecretByUUID(authClientUUID, tenant.TenantID)
	if err != nil {
		util.Error(w, http.StatusNotFound, "Auth client not found")
		return
	}

	dtoRes := dto.AuthClientSecretResponseDto{
		ClientID:     authClient.ClientID,
		ClientSecret: authClient.ClientSecret,
	}

	util.Success(w, dtoRes, "Auth client secret fetched successfully")
}

// Get Auth client config by UUID
func (h *AuthClientHandler) GetConfigByUUID(w http.ResponseWriter, r *http.Request) {
	// Get tenant from context
	tenant, ok := r.Context().Value(middleware.TenantContextKey).(*model.Tenant)
	if !ok || tenant == nil {
		util.Error(w, http.StatusUnauthorized, "Tenant not found in context")
		return
	}

	authClientUUID, err := uuid.Parse(chi.URLParam(r, "auth_client_uuid"))
	if err != nil {
		util.Error(w, http.StatusBadRequest, "Invalid Auth client UUID")
		return
	}

	authClientConfig, err := h.authClientService.GetConfigByUUID(authClientUUID, tenant.TenantID)
	if err != nil {
		util.Error(w, http.StatusNotFound, "Auth client not found")
		return
	}

	// Return config directly as data (not wrapped in DTO)
	util.Success(w, authClientConfig, "Auth client config fetched successfully")
}

// Create Auth Client
func (h *AuthClientHandler) Create(w http.ResponseWriter, r *http.Request) {
	// Get tenant from context
	tenant, ok := r.Context().Value(middleware.TenantContextKey).(*model.Tenant)
	if !ok || tenant == nil {
		util.Error(w, http.StatusUnauthorized, "Tenant not found in context")
		return
	}

	// Get authentication context
	user := r.Context().Value(middleware.UserContextKey).(*model.User)

	var req dto.AuthClientCreateRequestDto
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		util.Error(w, http.StatusBadRequest, "Invalid request", err.Error())
		return
	}

	if err := req.Validate(); err != nil {
		util.ValidationError(w, err)
		return
	}

	authClient, err := h.authClientService.Create(tenant.TenantID, req.Name, req.DisplayName, req.ClientType, req.Domain, req.Config, req.Status, false, req.IdentityProviderUUID, user.UserUUID)
	if err != nil {
		util.Error(w, http.StatusInternalServerError, "Failed to create auth client", err.Error())
		return
	}

	dtoRes := toAuthClientResponseDto(*authClient)

	util.Created(w, dtoRes, "Auth client created successfully")
}

// Update Auth Client
func (h *AuthClientHandler) Update(w http.ResponseWriter, r *http.Request) {
	// Get tenant from context
	tenant, ok := r.Context().Value(middleware.TenantContextKey).(*model.Tenant)
	if !ok || tenant == nil {
		util.Error(w, http.StatusUnauthorized, "Tenant not found in context")
		return
	}

	// Get authentication context
	user := r.Context().Value(middleware.UserContextKey).(*model.User)

	authClientUUID, err := uuid.Parse(chi.URLParam(r, "auth_client_uuid"))
	if err != nil {
		util.Error(w, http.StatusBadRequest, "Invalid auth client UUID")
		return
	}

	var req dto.AuthClientUpdateRequestDto
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		util.Error(w, http.StatusBadRequest, "Invalid request", err.Error())
		return
	}

	if err := req.Validate(); err != nil {
		util.ValidationError(w, err)
		return
	}

	authClient, err := h.authClientService.Update(authClientUUID, tenant.TenantID, req.Name, req.DisplayName, req.ClientType, req.Domain, req.Config, req.Status, false, user.UserUUID)
	if err != nil {
		util.Error(w, http.StatusInternalServerError, "Failed to update auth client", err.Error())
		return
	}

	dtoRes := toAuthClientResponseDto(*authClient)

	util.Success(w, dtoRes, "Auth client updated successfully")
}

// Set Auth client status
func (h *AuthClientHandler) SetStatus(w http.ResponseWriter, r *http.Request) {
	// Get tenant from context
	tenant, ok := r.Context().Value(middleware.TenantContextKey).(*model.Tenant)
	if !ok || tenant == nil {
		util.Error(w, http.StatusUnauthorized, "Tenant not found in context")
		return
	}

	// Get authentication context
	user := r.Context().Value(middleware.UserContextKey).(*model.User)

	authClientUUID, err := uuid.Parse(chi.URLParam(r, "auth_client_uuid"))
	if err != nil {
		util.Error(w, http.StatusBadRequest, "Invalid auth client UUID")
		return
	}

	// Toggle status between active and inactive
	newStatus := "active"
	// We need to get current status first to toggle it
	currentClient, err := h.authClientService.GetByUUID(authClientUUID, tenant.TenantID)
	if err != nil {
		util.Error(w, http.StatusNotFound, "Auth client not found")
		return
	}
	if currentClient.Status == "active" {
		newStatus = "inactive"
	}

	authClient, err := h.authClientService.SetStatusByUUID(authClientUUID, tenant.TenantID, newStatus, user.UserUUID)
	if err != nil {
		util.Error(w, http.StatusInternalServerError, "Failed to update API", err.Error())
		return
	}

	dtoRes := toAuthClientResponseDto(*authClient)

	util.Success(w, dtoRes, "Auth client status updated successfully")
}

// Delete Auth Client
func (h *AuthClientHandler) Delete(w http.ResponseWriter, r *http.Request) {
	// Get tenant from context
	tenant, ok := r.Context().Value(middleware.TenantContextKey).(*model.Tenant)
	if !ok || tenant == nil {
		util.Error(w, http.StatusUnauthorized, "Tenant not found in context")
		return
	}

	// Get authentication context
	user := r.Context().Value(middleware.UserContextKey).(*model.User)

	authClientUUID, err := uuid.Parse(chi.URLParam(r, "auth_client_uuid"))
	if err != nil {
		util.Error(w, http.StatusBadRequest, "Invalid Auth Client UUID")
		return
	}

	authClient, err := h.authClientService.DeleteByUUID(authClientUUID, tenant.TenantID, user.UserUUID)
	if err != nil {
		util.Error(w, http.StatusInternalServerError, "Failed to delete auth client", err.Error())
		return
	}

	dtoRes := toAuthClientResponseDto(*authClient)

	util.Success(w, dtoRes, "Auth client deleted successfully")
}

func (h *AuthClientHandler) GetURIs(w http.ResponseWriter, r *http.Request) {
	// Get tenant from context
	tenant, ok := r.Context().Value(middleware.TenantContextKey).(*model.Tenant)
	if !ok || tenant == nil {
		util.Error(w, http.StatusUnauthorized, "Tenant not found in context")
		return
	}

	authClientUUID, err := uuid.Parse(chi.URLParam(r, "auth_client_uuid"))
	if err != nil {
		util.Error(w, http.StatusBadRequest, "Invalid auth client UUID")
		return
	}

	authClient, err := h.authClientService.GetByUUID(authClientUUID, tenant.TenantID)
	if err != nil {
		util.Error(w, http.StatusNotFound, "Auth client not found", err.Error())
		return
	}

	// Convert URIs to response format
	var uris []dto.AuthClientURIResponseDto
	if authClient.AuthClientURIs != nil {
		uris = make([]dto.AuthClientURIResponseDto, len(*authClient.AuthClientURIs))
		for i, uri := range *authClient.AuthClientURIs {
			uris[i] = dto.AuthClientURIResponseDto{
				AuthClientURIUUID: uri.AuthClientURIUUID,
				URI:               uri.URI,
				Type:              uri.Type,
				CreatedAt:         uri.CreatedAt,
				UpdatedAt:         uri.UpdatedAt,
			}
		}
	}

	response := dto.AuthClientURIsResponseDto{
		URIs: uris,
	}

	util.Success(w, response, "URIs retrieved successfully")
}

func (h *AuthClientHandler) CreateURI(w http.ResponseWriter, r *http.Request) {
	// Get tenant from context
	tenant, ok := r.Context().Value(middleware.TenantContextKey).(*model.Tenant)
	if !ok || tenant == nil {
		util.Error(w, http.StatusUnauthorized, "Tenant not found in context")
		return
	}

	// Get authentication context
	user := r.Context().Value(middleware.UserContextKey).(*model.User)

	authClientUUID, err := uuid.Parse(chi.URLParam(r, "auth_client_uuid"))
	if err != nil {
		util.Error(w, http.StatusBadRequest, "Invalid auth client UUID")
		return
	}

	var req dto.AuthClientURICreateOrUpdateRequestDto
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		util.Error(w, http.StatusBadRequest, "Invalid request", err.Error())
		return
	}

	if err := req.Validate(); err != nil {
		util.ValidationError(w, err)
		return
	}

	uri, err := h.authClientService.CreateURI(authClientUUID, tenant.TenantID, req.URI, req.Type, user.UserUUID)
	if err != nil {
		util.Error(w, http.StatusInternalServerError, "Failed to create URI", err.Error())
		return
	}

	dtoRes := dto.AuthClientURIResponseDto{
		AuthClientURIUUID: (*uri.AuthClientURIs)[0].AuthClientURIUUID,
		URI:               (*uri.AuthClientURIs)[0].URI,
		Type:              (*uri.AuthClientURIs)[0].Type,
		CreatedAt:         (*uri.AuthClientURIs)[0].CreatedAt,
		UpdatedAt:         (*uri.AuthClientURIs)[0].UpdatedAt,
	}

	util.Created(w, dtoRes, "URI created successfully")
}

func (h *AuthClientHandler) UpdateURI(w http.ResponseWriter, r *http.Request) {
	// Get tenant from context
	tenant, ok := r.Context().Value(middleware.TenantContextKey).(*model.Tenant)
	if !ok || tenant == nil {
		util.Error(w, http.StatusUnauthorized, "Tenant not found in context")
		return
	}

	// Get authentication context
	user := r.Context().Value(middleware.UserContextKey).(*model.User)

	authClientUUID, err := uuid.Parse(chi.URLParam(r, "auth_client_uuid"))
	if err != nil {
		util.Error(w, http.StatusBadRequest, "Invalid auth client UUID")
		return
	}

	authClientURIUUID, err := uuid.Parse(chi.URLParam(r, "auth_client_uri_uuid"))
	if err != nil {
		util.Error(w, http.StatusBadRequest, "Invalid auth client URI UUID")
		return
	}

	var req dto.AuthClientURICreateOrUpdateRequestDto
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		util.Error(w, http.StatusBadRequest, "Invalid request", err.Error())
		return
	}

	if err := req.Validate(); err != nil {
		util.ValidationError(w, err)
		return
	}

	uri, err := h.authClientService.UpdateURI(authClientUUID, tenant.TenantID, authClientURIUUID, req.URI, req.Type, user.UserUUID)
	if err != nil {
		util.Error(w, http.StatusInternalServerError, "Failed to update URI", err.Error())
		return
	}

	// Find the updated URI in the response
	var updatedURI *service.AuthClientURIServiceDataResult
	if uri.AuthClientURIs != nil {
		for _, u := range *uri.AuthClientURIs {
			if u.AuthClientURIUUID == authClientURIUUID {
				updatedURI = &u
				break
			}
		}
	}

	if updatedURI == nil {
		util.Error(w, http.StatusInternalServerError, "Updated URI not found in response")
		return
	}

	dtoRes := dto.AuthClientURIResponseDto{
		AuthClientURIUUID: updatedURI.AuthClientURIUUID,
		URI:               updatedURI.URI,
		Type:              updatedURI.Type,
		CreatedAt:         updatedURI.CreatedAt,
		UpdatedAt:         updatedURI.UpdatedAt,
	}

	util.Success(w, dtoRes, "URI updated successfully")
}

func (h *AuthClientHandler) DeleteURI(w http.ResponseWriter, r *http.Request) {
	// Get tenant from context
	tenant, ok := r.Context().Value(middleware.TenantContextKey).(*model.Tenant)
	if !ok || tenant == nil {
		util.Error(w, http.StatusUnauthorized, "Tenant not found in context")
		return
	}

	// Get authentication context
	user := r.Context().Value(middleware.UserContextKey).(*model.User)

	authClientUUID, err := uuid.Parse(chi.URLParam(r, "auth_client_uuid"))
	if err != nil {
		util.Error(w, http.StatusBadRequest, "Invalid auth client UUID")
		return
	}

	authClientURIUUID, err := uuid.Parse(chi.URLParam(r, "auth_client_uri_uuid"))
	if err != nil {
		util.Error(w, http.StatusBadRequest, "Invalid auth client URI UUID")
		return
	}

	authClient, err := h.authClientService.DeleteURI(authClientUUID, tenant.TenantID, authClientURIUUID, user.UserUUID)
	if err != nil {
		util.Error(w, http.StatusInternalServerError, "Failed to delete URI", err.Error())
		return
	}

	dtoRes := toAuthClientResponseDto(*authClient)

	util.Success(w, dtoRes, "URI deleted successfully")
}

// Get APIs assigned to auth client
func (h *AuthClientHandler) GetApis(w http.ResponseWriter, r *http.Request) {
	// Get authentication context
	user := r.Context().Value(middleware.UserContextKey).(*model.User)

	authClientUUID, err := uuid.Parse(chi.URLParam(r, "auth_client_uuid"))
	if err != nil {
		util.Error(w, http.StatusBadRequest, "Invalid auth client UUID")
		return
	}

	// Get auth client APIs
	authClientApis, err := h.authClientService.GetAuthClientApis(user.TenantID, authClientUUID)
	if err != nil {
		util.Error(w, http.StatusInternalServerError, "Failed to get auth client APIs")
		return
	}

	// Convert to DTO
	apis := make([]dto.AuthClientApiResponseDto, len(authClientApis))
	for i, api := range authClientApis {
		// Convert API service data to DTO
		apiDto := dto.APIResponseDto{
			APIUUID:     api.Api.APIUUID,
			Name:        api.Api.Name,
			DisplayName: api.Api.DisplayName,
			Description: api.Api.Description,
			Status:      api.Api.Status,
			IsDefault:   api.Api.IsDefault,
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

		apis[i] = dto.AuthClientApiResponseDto{
			AuthClientApiUUID: api.AuthClientApiUUID,
			Api:               apiDto,
			Permissions:       permissions,
			CreatedAt:         api.CreatedAt,
		}
	}

	response := dto.AuthClientApisResponseDto{
		APIs: apis,
	}

	util.Success(w, response, "Auth client APIs retrieved successfully")
}

// Add APIs to auth client
func (h *AuthClientHandler) AddApis(w http.ResponseWriter, r *http.Request) {
	// Get authentication context
	user := r.Context().Value(middleware.UserContextKey).(*model.User)

	authClientUUID, err := uuid.Parse(chi.URLParam(r, "auth_client_uuid"))
	if err != nil {
		util.Error(w, http.StatusBadRequest, "Invalid auth client UUID")
		return
	}

	var req dto.AddAuthClientApisRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		util.Error(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Add APIs to auth client
	err = h.authClientService.AddAuthClientApis(user.TenantID, authClientUUID, req.ApiUUIDs)
	if err != nil {
		util.Error(w, http.StatusInternalServerError, "Failed to add APIs to auth client")
		return
	}

	response := dto.SuccessResponseDto{
		Message: "APIs added to auth client successfully",
	}

	util.Success(w, response, "APIs added to auth client successfully")
}

// Remove API from auth client
func (h *AuthClientHandler) RemoveApi(w http.ResponseWriter, r *http.Request) {
	// Get authentication context
	user := r.Context().Value(middleware.UserContextKey).(*model.User)

	authClientUUID, err := uuid.Parse(chi.URLParam(r, "auth_client_uuid"))
	if err != nil {
		util.Error(w, http.StatusBadRequest, "Invalid auth client UUID")
		return
	}

	apiUUID, err := uuid.Parse(chi.URLParam(r, "api_uuid"))
	if err != nil {
		util.Error(w, http.StatusBadRequest, "Invalid API UUID")
		return
	}

	// Remove API from auth client
	err = h.authClientService.RemoveAuthClientApi(user.TenantID, authClientUUID, apiUUID)
	if err != nil {
		util.Error(w, http.StatusInternalServerError, "Failed to remove API from auth client")
		return
	}

	response := dto.SuccessResponseDto{
		Message: "API removed from auth client successfully",
	}

	util.Success(w, response, "API removed from auth client successfully")
}

// Get permissions for a specific API assigned to auth client
func (h *AuthClientHandler) GetApiPermissions(w http.ResponseWriter, r *http.Request) {
	// Get authentication context
	user := r.Context().Value(middleware.UserContextKey).(*model.User)

	authClientUUID, err := uuid.Parse(chi.URLParam(r, "auth_client_uuid"))
	if err != nil {
		util.Error(w, http.StatusBadRequest, "Invalid auth client UUID")
		return
	}

	apiUUID, err := uuid.Parse(chi.URLParam(r, "api_uuid"))
	if err != nil {
		util.Error(w, http.StatusBadRequest, "Invalid API UUID")
		return
	}

	// Get auth client API permissions
	permissions, err := h.authClientService.GetAuthClientApiPermissions(user.TenantID, authClientUUID, apiUUID)
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

	response := dto.AuthClientApiPermissionsResponseDto{
		Permissions: permissionDtos,
	}

	util.Success(w, response, "Auth client API permissions retrieved successfully")
}

// Add permissions to a specific API for auth client
func (h *AuthClientHandler) AddApiPermissions(w http.ResponseWriter, r *http.Request) {
	// Get authentication context
	user := r.Context().Value(middleware.UserContextKey).(*model.User)

	authClientUUID, err := uuid.Parse(chi.URLParam(r, "auth_client_uuid"))
	if err != nil {
		util.Error(w, http.StatusBadRequest, "Invalid auth client UUID")
		return
	}

	apiUUID, err := uuid.Parse(chi.URLParam(r, "api_uuid"))
	if err != nil {
		util.Error(w, http.StatusBadRequest, "Invalid API UUID")
		return
	}

	var req dto.AddAuthClientApiPermissionsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		util.Error(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Add permissions to auth client API
	err = h.authClientService.AddAuthClientApiPermissions(user.TenantID, authClientUUID, apiUUID, req.PermissionUUIDs)
	if err != nil {
		util.Error(w, http.StatusInternalServerError, "Failed to add permissions to auth client API")
		return
	}

	response := dto.SuccessResponseDto{
		Message: "Permissions added to auth client API successfully",
	}

	util.Success(w, response, "Permissions added to auth client API successfully")
}

// Remove permission from a specific API for auth client
func (h *AuthClientHandler) RemoveApiPermission(w http.ResponseWriter, r *http.Request) {
	// Get authentication context
	user := r.Context().Value(middleware.UserContextKey).(*model.User)

	authClientUUID, err := uuid.Parse(chi.URLParam(r, "auth_client_uuid"))
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

	// Remove permission from auth client API
	err = h.authClientService.RemoveAuthClientApiPermission(user.TenantID, authClientUUID, apiUUID, permissionUUID)
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
func toAuthClientResponseDto(r service.AuthClientServiceDataResult) dto.AuthClientResponseDto {
	result := dto.AuthClientResponseDto{
		AuthClientUUID: r.AuthClientUUID,
		Name:           r.Name,
		DisplayName:    r.DisplayName,
		ClientType:     r.ClientType,
		Domain:         r.Domain,
		Status:         r.Status,
		IsDefault:      r.IsDefault,
		IsSystem:       r.IsSystem,
		CreatedAt:      r.CreatedAt,
		UpdatedAt:      r.UpdatedAt,
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

	if r.AuthClientURIs != nil && len(*r.AuthClientURIs) > 0 {
		result.URIs = make([]dto.AuthClientURIResponseDto, len(*r.AuthClientURIs))
		for i, uri := range *r.AuthClientURIs {
			result.URIs[i] = dto.AuthClientURIResponseDto{
				AuthClientURIUUID: uri.AuthClientURIUUID,
				URI:               uri.URI,
				Type:              uri.Type,
				CreatedAt:         uri.CreatedAt,
				UpdatedAt:         uri.UpdatedAt,
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
