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

type AuthClientHandler struct {
	authClientService service.AuthClientService
}

func NewAuthClientHandler(authClientService service.AuthClientService) *AuthClientHandler {
	return &AuthClientHandler{authClientService}
}

// Get all auth clients with pagination
func (h *AuthClientHandler) Get(w http.ResponseWriter, r *http.Request) {
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
	reqParams := dto.AuthClientFilterDto{
		Name:                 util.PtrOrNil(q.Get("name")),
		DisplayName:          util.PtrOrNil(q.Get("display_name")),
		ClientType:           util.PtrOrNil(q.Get("client_type")),
		IdentityProviderUUID: util.PtrOrNil(q.Get("identity_provider_id")),
		IsDefault:            isDefault,
		IsActive:             isActive,
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
		Name:                 reqParams.Name,
		DisplayName:          reqParams.DisplayName,
		ClientType:           reqParams.ClientType,
		IdentityProviderUUID: reqParams.IdentityProviderUUID,
		IsDefault:            reqParams.IsDefault,
		IsActive:             reqParams.IsActive,
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

	util.Success(w, response, "Auth containers fetched successfully")
}

// Get Auth client by UUID
func (h *AuthClientHandler) GetByUUID(w http.ResponseWriter, r *http.Request) {
	authClientUUID, err := uuid.Parse(chi.URLParam(r, "auth_client_uuid"))
	if err != nil {
		util.Error(w, http.StatusBadRequest, "Invalid Auth client UUID")
		return
	}

	authClient, err := h.authClientService.GetByUUID(authClientUUID)
	if err != nil {
		util.Error(w, http.StatusNotFound, "Auth client not found")
		return
	}

	dtoRes := toAuthClientResponseDto(*authClient)

	util.Success(w, dtoRes, "Auth client fetched successfully")
}

// Get Auth client secret by UUID
func (h *AuthClientHandler) GetSecretByUUID(w http.ResponseWriter, r *http.Request) {
	authClientUUID, err := uuid.Parse(chi.URLParam(r, "auth_client_uuid"))
	if err != nil {
		util.Error(w, http.StatusBadRequest, "Invalid Auth client UUID")
		return
	}

	authClient, err := h.authClientService.GetSecretByUUID(authClientUUID)
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
	authClientUUID, err := uuid.Parse(chi.URLParam(r, "auth_client_uuid"))
	if err != nil {
		util.Error(w, http.StatusBadRequest, "Invalid Auth client UUID")
		return
	}

	authClient, err := h.authClientService.GetConfigByUUID(authClientUUID)
	if err != nil {
		util.Error(w, http.StatusNotFound, "Auth client not found")
		return
	}

	dtoRes := dto.AuthClientConfigResponseDto{
		Config: authClient.Config,
	}

	util.Success(w, dtoRes, "Auth client config fetched successfully")
}

// Create Auth Client
func (h *AuthClientHandler) Create(w http.ResponseWriter, r *http.Request) {
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

	authClient, err := h.authClientService.Create(req.Name, req.DisplayName, req.ClientType, req.Domain, req.Config, req.IsActive, false, req.IdentityProviderUUID, user.UserUUID)
	if err != nil {
		util.Error(w, http.StatusInternalServerError, "Failed to create auth client", err.Error())
		return
	}

	dtoRes := toAuthClientResponseDto(*authClient)

	util.Created(w, dtoRes, "Auth client created successfully")
}

// Update Auth Client
func (h *AuthClientHandler) Update(w http.ResponseWriter, r *http.Request) {
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

	authClient, err := h.authClientService.Update(authClientUUID, req.Name, req.DisplayName, req.ClientType, req.Domain, req.Config, req.IsActive, false, user.UserUUID)
	if err != nil {
		util.Error(w, http.StatusInternalServerError, "Failed to update auth client", err.Error())
		return
	}

	dtoRes := toAuthClientResponseDto(*authClient)

	util.Success(w, dtoRes, "Auth client updated successfully")
}

// Set Auth client status
func (h *AuthClientHandler) SetStatus(w http.ResponseWriter, r *http.Request) {
	// Get authentication context
	user := r.Context().Value(middleware.UserContextKey).(*model.User)

	authClientUUID, err := uuid.Parse(chi.URLParam(r, "auth_client_uuid"))
	if err != nil {
		util.Error(w, http.StatusBadRequest, "Invalid auth client UUID")
		return
	}

	authClient, err := h.authClientService.SetActiveStatusByUUID(authClientUUID, user.UserUUID)
	if err != nil {
		util.Error(w, http.StatusInternalServerError, "Failed to update API", err.Error())
		return
	}

	dtoRes := toAuthClientResponseDto(*authClient)

	util.Success(w, dtoRes, "Auth client status updated successfully")
}

// Delete Auth Client
func (h *AuthClientHandler) Delete(w http.ResponseWriter, r *http.Request) {
	// Get authentication context
	user := r.Context().Value(middleware.UserContextKey).(*model.User)

	authClientUUID, err := uuid.Parse(chi.URLParam(r, "auth_client_uuid"))
	if err != nil {
		util.Error(w, http.StatusBadRequest, "Invalid Auth Client UUID")
		return
	}

	authClient, err := h.authClientService.DeleteByUUID(authClientUUID, user.UserUUID)
	if err != nil {
		util.Error(w, http.StatusInternalServerError, "Failed to delete auth client", err.Error())
		return
	}

	dtoRes := toAuthClientResponseDto(*authClient)

	util.Success(w, dtoRes, "Auth client deleted successfully")
}

func (h *AuthClientHandler) CreateRedirectURI(w http.ResponseWriter, r *http.Request) {
	// Get authentication context
	user := r.Context().Value(middleware.UserContextKey).(*model.User)

	authClientUUID, err := uuid.Parse(chi.URLParam(r, "auth_client_uuid"))
	if err != nil {
		util.Error(w, http.StatusBadRequest, "Invalid auth client UUID")
		return
	}

	var req dto.AuthClientRedirectURICreateOrUpdateRequestDto
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		util.Error(w, http.StatusBadRequest, "Invalid request", err.Error())
		return
	}

	if err := req.Validate(); err != nil {
		util.ValidationError(w, err)
		return
	}

	redirectURI, err := h.authClientService.CreateRedirectURI(authClientUUID, req.RedirectURI, user.UserUUID)
	if err != nil {
		util.Error(w, http.StatusInternalServerError, "Failed to create redirect URI", err.Error())
		return
	}

	dtoRes := toAuthClientResponseDto(*redirectURI)

	util.Created(w, dtoRes, "Redirect URI created successfully")
}

func (h *AuthClientHandler) UpdateRedirectURI(w http.ResponseWriter, r *http.Request) {
	// Get authentication context
	user := r.Context().Value(middleware.UserContextKey).(*model.User)

	authClientUUID, err := uuid.Parse(chi.URLParam(r, "auth_client_uuid"))
	if err != nil {
		util.Error(w, http.StatusBadRequest, "Invalid auth client UUID")
		return
	}

	authClientRedirectURIUUID, err := uuid.Parse(chi.URLParam(r, "auth_client_redirect_uri_uuid"))
	if err != nil {
		util.Error(w, http.StatusBadRequest, "Invalid auth client redirect URI UUID")
		return
	}

	var req dto.AuthClientRedirectURICreateOrUpdateRequestDto
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		util.Error(w, http.StatusBadRequest, "Invalid request", err.Error())
		return
	}

	if err := req.Validate(); err != nil {
		util.ValidationError(w, err)
		return
	}

	redirectURI, err := h.authClientService.UpdateRedirectURI(authClientUUID, authClientRedirectURIUUID, req.RedirectURI, user.UserUUID)
	if err != nil {
		util.Error(w, http.StatusInternalServerError, "Failed to update redirect URI", err.Error())
		return
	}

	dtoRes := toAuthClientResponseDto(*redirectURI)

	util.Success(w, dtoRes, "Redirect URI updated successfully")
}

func (h *AuthClientHandler) DeleteRedirectURI(w http.ResponseWriter, r *http.Request) {
	// Get authentication context
	user := r.Context().Value(middleware.UserContextKey).(*model.User)

	authClientUUID, err := uuid.Parse(chi.URLParam(r, "auth_client_uuid"))
	if err != nil {
		util.Error(w, http.StatusBadRequest, "Invalid auth client UUID")
		return
	}

	authClientRedirectURIUUID, err := uuid.Parse(chi.URLParam(r, "auth_client_redirect_uri_uuid"))
	if err != nil {
		util.Error(w, http.StatusBadRequest, "Invalid auth client redirect URI UUID")
		return
	}

	authClient, err := h.authClientService.DeleteRedirectURI(authClientUUID, authClientRedirectURIUUID, user.UserUUID)
	if err != nil {
		util.Error(w, http.StatusInternalServerError, "Failed to delete redirect URI", err.Error())
		return
	}

	dtoRes := toAuthClientResponseDto(*authClient)

	util.Success(w, dtoRes, "Redirect URI deleted successfully")
}

// Add permissions to auth client
func (h *AuthClientHandler) AddPermissions(w http.ResponseWriter, r *http.Request) {
	// Get authentication context
	user := r.Context().Value(middleware.UserContextKey).(*model.User)

	authClientUUID, err := uuid.Parse(chi.URLParam(r, "auth_client_uuid"))
	if err != nil {
		util.Error(w, http.StatusBadRequest, "Invalid auth client UUID")
		return
	}

	var req dto.AuthClientAddPermissionsRequestDto
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		util.Error(w, http.StatusBadRequest, "Invalid request", err.Error())
		return
	}

	if err := req.Validate(); err != nil {
		util.ValidationError(w, err)
		return
	}

	authClient, err := h.authClientService.AddAuthClientPermissions(authClientUUID, req.Permissions, user.UserUUID)
	if err != nil {
		util.Error(w, http.StatusInternalServerError, "Failed to add permissions to auth client", err.Error())
		return
	}

	dtoRes := toAuthClientResponseDto(*authClient)

	util.Success(w, dtoRes, "Permissions added to auth client successfully")
}

// Remove permission from auth client
func (h *AuthClientHandler) RemovePermission(w http.ResponseWriter, r *http.Request) {
	// Validate auth_client_uuid
	authClientUUID, err := uuid.Parse(chi.URLParam(r, "auth_client_uuid"))
	if err != nil {
		util.Error(w, http.StatusBadRequest, "Invalid auth client UUID")
		return
	}

	// Validate permission_uuid
	permissionUUID, err := uuid.Parse(chi.URLParam(r, "permission_uuid"))
	if err != nil {
		util.Error(w, http.StatusBadRequest, "Invalid permission UUID")
		return
	}

	// Get authentication context
	user := r.Context().Value(middleware.UserContextKey).(*model.User)

	// Remove permission from auth client
	authClient, err := h.authClientService.RemoveAuthClientPermissions(authClientUUID, permissionUUID, user.UserUUID)
	if err != nil {
		util.Error(w, http.StatusInternalServerError, "Failed to remove permission from auth client", err.Error())
		return
	}

	// Build response data
	dtoRes := toAuthClientResponseDto(*authClient)

	util.Success(w, dtoRes, "Permission removed from auth client successfully")
}

// Convert result to DTO
func toAuthClientResponseDto(r service.AuthClientServiceDataResult) dto.AuthClientResponseDto {
	result := dto.AuthClientResponseDto{
		AuthClientUUID: r.AuthClientUUID,
		Name:           r.Name,
		DisplayName:    r.DisplayName,
		ClientType:     r.ClientType,
		Domain:         r.Domain,
		IsActive:       r.IsActive,
		IsDefault:      r.IsDefault,
		CreatedAt:      r.CreatedAt,
		UpdatedAt:      r.UpdatedAt,
	}

	if r.IdentityProvider != nil {
		result.IdentityProvider = &dto.IdentityProviderResponseDto{
			IdentityProviderUUID: r.IdentityProvider.IdentityProviderUUID,
			Name:                 r.IdentityProvider.Name,
			DisplayName:          r.IdentityProvider.DisplayName,
			ProviderType:         r.IdentityProvider.ProviderType,
			Identifier:           r.IdentityProvider.Identifier,
			IsActive:             r.IdentityProvider.IsActive,
			IsDefault:            r.IdentityProvider.IsDefault,
			CreatedAt:            r.IdentityProvider.CreatedAt,
			UpdatedAt:            r.IdentityProvider.UpdatedAt,
		}
	}

	if r.AuthClientRedirectURIs != nil && len(*r.AuthClientRedirectURIs) > 0 {
		result.RedirectURIs = make([]dto.AuthClientRedirectURIResponseDto, len(*r.AuthClientRedirectURIs))
		for i, uri := range *r.AuthClientRedirectURIs {
			result.RedirectURIs[i] = dto.AuthClientRedirectURIResponseDto{
				AuthClientRedirectURIUUID: uri.AuthClientRedirectURIUUID,
				RedirectURI:               uri.RedirectURI,
				CreatedAt:                 uri.CreatedAt,
				UpdatedAt:                 uri.UpdatedAt,
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
