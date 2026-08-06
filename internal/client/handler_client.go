package client

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/maintainerd/maintainerd-auth/internal/auditlog"
	"github.com/maintainerd/maintainerd-auth/internal/platform/middleware"
	"github.com/maintainerd/maintainerd-auth/internal/platform/pagination"
	"github.com/maintainerd/maintainerd-auth/internal/platform/ptr"
	resp "github.com/maintainerd/maintainerd-auth/internal/platform/response"
)

type ClientHandler struct {
	ClientService ClientService
	auditLogger   auditlog.ManagementAuditLogger
}

func NewClientHandler(ClientService ClientService) *ClientHandler {
	return &ClientHandler{ClientService: ClientService}
}

func (h *ClientHandler) SetAuditLogger(l auditlog.ManagementAuditLogger) { h.auditLogger = l }

func (h *ClientHandler) logAudit(r *http.Request, tenantID int64, actorUserID *int64, action, resourceType, resourceID string, resourceUUID *uuid.UUID, changes, outcome string) {
	if h.auditLogger == nil {
		return
	}
	_ = h.auditLogger.Log(r.Context(), auditlog.LogEntry{
		TenantID:     tenantID,
		ActorUserID:  actorUserID,
		Action:       action,
		ResourceType: resourceType,
		ResourceID:   resourceID,
		ResourceUUID: resourceUUID,
		Changes:      changes,
		Outcome:      outcome,
	})
}

func (h *ClientHandler) GetPublic(w http.ResponseWriter, r *http.Request) {
	identifier := strings.TrimSpace(r.URL.Query().Get("client_id"))
	if identifier == "" || r.URL.Query().Get("tenant_id") != "" {
		resp.Error(w, http.StatusBadRequest, "client_id is required and tenant_id is not accepted")
		return
	}
	publicService, ok := h.ClientService.(interface {
		GetPublicByIdentifier(context.Context, string) (*ClientPublicServiceDataResult, error)
	})
	if !ok {
		resp.Error(w, http.StatusInternalServerError, "Public client discovery is unavailable")
		return
	}
	client, err := publicService.GetPublicByIdentifier(r.Context(), identifier)
	if err != nil {
		resp.HandleServiceError(w, r, "Auth client not found", err)
		return
	}
	resp.Success(w, ClientPublicResponseDTO{
		ClientID: client.ClientID, Name: client.Name, DisplayName: client.DisplayName,
		ClientType: client.ClientType, Domain: client.Domain, TenantIdentifier: client.TenantIdentifier,
	}, "Auth client fetched successfully")
}

func (h *ClientHandler) GetPublicConsole(w http.ResponseWriter, r *http.Request) {
	tenantIdentifier := strings.TrimSpace(r.URL.Query().Get("tenant_id"))
	if tenantIdentifier == "" || r.URL.Query().Get("client_id") != "" {
		resp.Error(w, http.StatusBadRequest, "tenant_id is required and client_id is not accepted")
		return
	}
	publicService, ok := h.ClientService.(interface {
		GetPublicConsoleByTenantIdentifier(context.Context, string) (*ClientPublicServiceDataResult, error)
	})
	if !ok {
		resp.Error(w, http.StatusInternalServerError, "Public console client discovery is unavailable")
		return
	}
	client, err := publicService.GetPublicConsoleByTenantIdentifier(r.Context(), tenantIdentifier)
	if err != nil {
		resp.HandleServiceError(w, r, "Console client not found", err)
		return
	}
	resp.Success(w, ClientPublicResponseDTO{
		ClientID: client.ClientID, Name: client.Name, DisplayName: client.DisplayName,
		ClientType: client.ClientType, Domain: client.Domain, TenantIdentifier: client.TenantIdentifier,
	}, "Console client fetched successfully")
}

// Get all auth clients with pagination
func (h *ClientHandler) Get(w http.ResponseWriter, r *http.Request) {
	// Get tenant from context
	tenant := middleware.AuthFromRequest(r).Tenant
	if tenant == nil {
		resp.Error(w, http.StatusUnauthorized, "Tenant not found in context")
		return
	}

	// Parse query parameters
	q := r.URL.Query()

	// Parse pagination

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
	reqParams := ClientFilterDTO{
		Name:                 ptr.PtrOrNil(q.Get("name")),
		DisplayName:          ptr.PtrOrNil(q.Get("display_name")),
		ClientType:           clientType,
		IdentityProviderUUID: ptr.PtrOrNil(q.Get("identity_provider_id")),
		Status:               status,
		IsDefault:            isDefault,
		IsSystem:             isSystem,
		PaginationRequestDTO: pagination.ParseQuery(r),
	}

	if err := reqParams.Validate(); err != nil {
		resp.ValidationError(w, err)
		return
	}

	// Build service filter
	ClientFilter := ClientServiceGetFilter{
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
	rows := make([]ClientResponseDTO, len(result.Data))
	for i, r := range result.Data {
		rows[i] = toClientResponseDTO(r)
	}

	// Build response data
	response := PaginatedResponseDTO[ClientResponseDTO]{
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
	tenant := middleware.AuthFromRequest(r).Tenant
	if tenant == nil {
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

// Get Auth client config by UUID
func (h *ClientHandler) GetConfigByUUID(w http.ResponseWriter, r *http.Request) {
	// Get tenant from context
	tenant := middleware.AuthFromRequest(r).Tenant
	if tenant == nil {
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
	tenant := middleware.AuthFromRequest(r).Tenant
	if tenant == nil {
		resp.Error(w, http.StatusUnauthorized, "Tenant not found in context")
		return
	}

	// Get authentication context
	user := middleware.AuthFromRequest(r).User
	if user == nil {
		resp.Error(w, http.StatusUnauthorized, "User not found in context")
		return
	}

	var req ClientCreateRequestDTO
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		resp.BadRequestBody(w)
		return
	}

	if err := req.Validate(); err != nil {
		resp.ValidationError(w, err)
		return
	}

	result, err := h.ClientService.Create(r.Context(), tenant.TenantID, req.Name, req.DisplayName, req.ClientType, req.Domain, req.Config, req.Status, req.IdentityProviderUUID, parseOptionalUUID(req.BrandingUUID), boolValue(req.AllowRegistration, true), req.BackchannelLogoutURI, req.FrontchannelLogoutURI, req.BackchannelLogoutSessionRequired, req.DPoPRequired, user.UserUUID, req.ServiceUUID)
	if err != nil {
		resp.HandleServiceError(w, r, "Failed to create auth client", err)
		return
	}

	dtoRes := struct {
		Client      interface{}                   `json:"client"`
		Credentials ClientCreateSecretResponseDTO `json:"credentials"`
	}{
		Client: toClientResponseDTO(*result.Client),
		Credentials: ClientCreateSecretResponseDTO{
			ClientUUID:   result.Client.ClientUUID.String(),
			ClientID:     result.ClientIdentifier,
			ClientSecret: result.PlaintextSecret,
		},
	}

	actorUserIDCreate := &user.UserID
	createdClientUUID := result.Client.ClientUUID
	changesJSONCreate, _ := json.Marshal(map[string]any{"after": toClientResponseDTO(*result.Client)})
	h.logAudit(r, tenant.TenantID, actorUserIDCreate, "create", "client", createdClientUUID.String(), &createdClientUUID, string(changesJSONCreate), "success")

	resp.Created(w, dtoRes, "Auth client created successfully. Store the client_secret now — it will not be shown again.")
}

// Update Auth Client
func (h *ClientHandler) Update(w http.ResponseWriter, r *http.Request) {
	// Get tenant from context
	tenant := middleware.AuthFromRequest(r).Tenant
	if tenant == nil {
		resp.Error(w, http.StatusUnauthorized, "Tenant not found in context")
		return
	}

	// Get authentication context
	user := middleware.AuthFromRequest(r).User
	if user == nil {
		resp.Error(w, http.StatusUnauthorized, "User not found in context")
		return
	}

	ClientUUID, err := uuid.Parse(chi.URLParam(r, "client_uuid"))
	if err != nil {
		resp.Error(w, http.StatusBadRequest, "Invalid auth client UUID")
		return
	}

	var req ClientUpdateRequestDTO
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		resp.BadRequestBody(w)
		return
	}

	if err := req.Validate(); err != nil {
		resp.ValidationError(w, err)
		return
	}

	Client, err := h.ClientService.Update(r.Context(), ClientUUID, tenant.TenantID, req.Name, req.DisplayName, req.ClientType, req.Domain, req.Config, req.Status, parseOptionalUUID(req.BrandingUUID), req.AllowRegistration, req.AllowMagicLink, req.BackchannelLogoutURI, req.FrontchannelLogoutURI, req.BackchannelLogoutSessionRequired, req.DPoPRequired, user.UserUUID, req.ExpectedUpdatedAt, req.ServiceUUID)
	if err != nil {
		resp.HandleServiceError(w, r, "Failed to update auth client", err)
		return
	}

	dtoRes := toClientResponseDTO(*Client)

	actorUserIDUpdate := &user.UserID
	changesJSONUpdate, _ := json.Marshal(map[string]any{"update": req, "after": dtoRes})
	h.logAudit(r, tenant.TenantID, actorUserIDUpdate, "update", "client", ClientUUID.String(), &ClientUUID, string(changesJSONUpdate), "success")

	resp.Success(w, dtoRes, "Auth client updated successfully")
}

// Set Auth client status
func (h *ClientHandler) SetStatus(w http.ResponseWriter, r *http.Request) {
	// Get tenant from context
	tenant := middleware.AuthFromRequest(r).Tenant
	if tenant == nil {
		resp.Error(w, http.StatusUnauthorized, "Tenant not found in context")
		return
	}

	// Get authentication context
	user := middleware.AuthFromRequest(r).User
	if user == nil {
		resp.Error(w, http.StatusUnauthorized, "User not found in context")
		return
	}

	ClientUUID, err := uuid.Parse(chi.URLParam(r, "client_uuid"))
	if err != nil {
		resp.Error(w, http.StatusBadRequest, "Invalid auth client UUID")
		return
	}

	// Honour the status the caller asked for. This previously read the current
	// row and flipped it, ignoring the body — so the result depended on server
	// state at the moment of the request rather than on the operator's choice.
	var req ClientSetStatusRequestDTO
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		resp.BadRequestBody(w)
		return
	}
	if err := req.Validate(); err != nil {
		resp.ValidationError(w, err)
		return
	}
	newStatus := req.Status

	Client, err := h.ClientService.SetStatusByUUID(r.Context(), ClientUUID, tenant.TenantID, newStatus, user.UserUUID)
	if err != nil {
		resp.HandleServiceError(w, r, "Failed to update auth client status", err)
		return
	}

	dtoRes := toClientResponseDTO(*Client)

	actorUserIDStatus := &user.UserID
	changesJSONStatus, _ := json.Marshal(map[string]any{"update": map[string]any{"status": string(newStatus)}})
	h.logAudit(r, tenant.TenantID, actorUserIDStatus, "set_status", "client", ClientUUID.String(), &ClientUUID, string(changesJSONStatus), "success")

	resp.Success(w, dtoRes, "Auth client status updated successfully")
}

// RotateSecret generates a new client secret, optionally keeping the old one
// valid for a grace period, and returns the new plaintext secret exactly once.
func (h *ClientHandler) RotateSecret(w http.ResponseWriter, r *http.Request) {
	tenant := middleware.AuthFromRequest(r).Tenant
	if tenant == nil {
		resp.Error(w, http.StatusUnauthorized, "Tenant not found in context")
		return
	}
	user := middleware.AuthFromRequest(r).User
	if user == nil {
		resp.Error(w, http.StatusUnauthorized, "User not found in context")
		return
	}

	clientUUID, err := uuid.Parse(chi.URLParam(r, "client_uuid"))
	if err != nil {
		resp.Error(w, http.StatusBadRequest, "Invalid auth client UUID")
		return
	}

	var req RotateSecretRequestDTO
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		req = RotateSecretRequestDTO{GracePeriodHours: 24}
	}
	if err := req.Validate(); err != nil {
		resp.ValidationError(w, err)
		return
	}

	newSecret, err := h.ClientService.RotateSecret(r.Context(), clientUUID, tenant.TenantID, user.UserUUID, req.GracePeriodHours)
	if err != nil {
		resp.HandleServiceError(w, r, "Failed to rotate client secret", err)
		return
	}

	var expiresAt *string
	if req.GracePeriodHours > 0 {
		t := time.Now().Add(time.Duration(req.GracePeriodHours) * time.Hour).UTC().Format(time.RFC3339)
		expiresAt = &t
	}

	dtoRes := RotateSecretResponseDTO{
		ClientSecret:            newSecret,
		PreviousSecretExpiresAt: expiresAt,
	}

	actorUserIDRotate := &user.UserID
	changesJSONRotate, _ := json.Marshal(map[string]any{"update": map[string]any{"secret_rotated": true, "grace_period_hours": req.GracePeriodHours}})
	h.logAudit(r, tenant.TenantID, actorUserIDRotate, "rotate_secret", "client", clientUUID.String(), &clientUUID, string(changesJSONRotate), "success")

	resp.Success(w, dtoRes, "Client secret rotated successfully. Store the new secret now — it will not be shown again.")
}

// Delete Auth Client
func (h *ClientHandler) Delete(w http.ResponseWriter, r *http.Request) {
	// Get tenant from context
	tenant := middleware.AuthFromRequest(r).Tenant
	if tenant == nil {
		resp.Error(w, http.StatusUnauthorized, "Tenant not found in context")
		return
	}

	// Get authentication context
	user := middleware.AuthFromRequest(r).User
	if user == nil {
		resp.Error(w, http.StatusUnauthorized, "User not found in context")
		return
	}

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

	actorUserIDDelete := &user.UserID
	changesJSONDelete, _ := json.Marshal(map[string]any{"before": map[string]any{"id": ClientUUID.String()}})
	h.logAudit(r, tenant.TenantID, actorUserIDDelete, "delete", "client", ClientUUID.String(), &ClientUUID, string(changesJSONDelete), "success")

	resp.Success(w, dtoRes, "Auth client deleted successfully")
}

func (h *ClientHandler) GetURIs(w http.ResponseWriter, r *http.Request) {
	// Get tenant from context
	tenant := middleware.AuthFromRequest(r).Tenant
	if tenant == nil {
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
	var uris []ClientURIResponseDTO
	if Client.ClientURIs != nil {
		uris = make([]ClientURIResponseDTO, len(*Client.ClientURIs))
		for i, uri := range *Client.ClientURIs {
			uris[i] = ClientURIResponseDTO(uri)
		}
	}

	response := ClientURIsResponseDTO{
		URIs: uris,
	}

	resp.Success(w, response, "URIs retrieved successfully")
}

func (h *ClientHandler) CreateURI(w http.ResponseWriter, r *http.Request) {
	// Get tenant from context
	tenant := middleware.AuthFromRequest(r).Tenant
	if tenant == nil {
		resp.Error(w, http.StatusUnauthorized, "Tenant not found in context")
		return
	}

	// Get authentication context
	user := middleware.AuthFromRequest(r).User
	if user == nil {
		resp.Error(w, http.StatusUnauthorized, "User not found in context")
		return
	}

	ClientUUID, err := uuid.Parse(chi.URLParam(r, "client_uuid"))
	if err != nil {
		resp.Error(w, http.StatusBadRequest, "Invalid auth client UUID")
		return
	}

	var req ClientURICreateOrUpdateRequestDTO
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		resp.BadRequestBody(w)
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

	// The service returns the parent client, so index 0 is the client's FIRST URI
	// — not the one just created — and it panics outright when the slice is empty.
	// Select the URI that was actually requested.
	created := findCreatedURI(uri.ClientURIs, req.URI, req.Type)
	if created == nil {
		resp.Error(w, http.StatusInternalServerError, "URI was created but could not be read back")
		return
	}
	dtoRes := ClientURIResponseDTO{
		ClientURIUUID: created.ClientURIUUID,
		URI:           created.URI,
		Type:          created.Type,
		CreatedAt:     created.CreatedAt,
		UpdatedAt:     created.UpdatedAt,
	}

	actorUserIDCreateURI := &user.UserID
	createdURIUUID := (*uri.ClientURIs)[0].ClientURIUUID
	changesJSONCreateURI, _ := json.Marshal(map[string]any{"after": dtoRes})
	h.logAudit(r, tenant.TenantID, actorUserIDCreateURI, "create", "client_uri", createdURIUUID.String(), &createdURIUUID, string(changesJSONCreateURI), "success")

	resp.Created(w, dtoRes, "URI created successfully")
}

func (h *ClientHandler) UpdateURI(w http.ResponseWriter, r *http.Request) {
	// Get tenant from context
	tenant := middleware.AuthFromRequest(r).Tenant
	if tenant == nil {
		resp.Error(w, http.StatusUnauthorized, "Tenant not found in context")
		return
	}

	// Get authentication context
	user := middleware.AuthFromRequest(r).User
	if user == nil {
		resp.Error(w, http.StatusUnauthorized, "User not found in context")
		return
	}

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

	var req ClientURICreateOrUpdateRequestDTO
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		resp.BadRequestBody(w)
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
	var updatedURI *ClientURIServiceDataResult
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

	dtoRes := ClientURIResponseDTO{
		ClientURIUUID: updatedURI.ClientURIUUID,
		URI:           updatedURI.URI,
		Type:          updatedURI.Type,
		CreatedAt:     updatedURI.CreatedAt,
		UpdatedAt:     updatedURI.UpdatedAt,
	}

	actorUserIDUpdateURI := &user.UserID
	changesJSONUpdateURI, _ := json.Marshal(map[string]any{"update": req, "after": dtoRes})
	h.logAudit(r, tenant.TenantID, actorUserIDUpdateURI, "update", "client_uri", ClientURIUUID.String(), &ClientURIUUID, string(changesJSONUpdateURI), "success")

	resp.Success(w, dtoRes, "URI updated successfully")
}

func (h *ClientHandler) DeleteURI(w http.ResponseWriter, r *http.Request) {
	// Get tenant from context
	tenant := middleware.AuthFromRequest(r).Tenant
	if tenant == nil {
		resp.Error(w, http.StatusUnauthorized, "Tenant not found in context")
		return
	}

	// Get authentication context
	user := middleware.AuthFromRequest(r).User
	if user == nil {
		resp.Error(w, http.StatusUnauthorized, "User not found in context")
		return
	}

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

	actorUserIDDeleteURI := &user.UserID
	changesJSONDeleteURI, _ := json.Marshal(map[string]any{"before": map[string]any{"id": ClientURIUUID.String()}})
	h.logAudit(r, tenant.TenantID, actorUserIDDeleteURI, "delete", "client_uri", ClientURIUUID.String(), &ClientURIUUID, string(changesJSONDeleteURI), "success")

	resp.Success(w, dtoRes, "URI deleted successfully")
}

// GetAPIs retrieves APIs assigned to auth client.
func (h *ClientHandler) GetAPIs(w http.ResponseWriter, r *http.Request) {
	// Get authentication context
	// Get tenant from context
	tenant := middleware.AuthFromRequest(r).Tenant
	if tenant == nil {
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
	apis := make([]ClientAPIResponseDTO, len(ClientAPIs))
	for i, api := range ClientAPIs {
		// Convert API service data to DTO
		apiDTO := APIResponseDTO{
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
		permissions := make([]PermissionResponseDTO, len(api.Permissions))
		for j, perm := range api.Permissions {
			permissions[j] = PermissionResponseDTO{
				PermissionUUID: perm.PermissionUUID,
				Name:           perm.Name,
				Description:    perm.Description,
				Status:         perm.Status,
				IsSystem:       perm.IsSystem,
				CreatedAt:      perm.CreatedAt,
				UpdatedAt:      perm.UpdatedAt,
			}
		}

		apis[i] = ClientAPIResponseDTO{
			ClientAPIUUID: api.ClientAPIUUID,
			API:           apiDTO,
			Permissions:   permissions,
			CreatedAt:     api.CreatedAt,
		}
	}

	response := ClientAPIsResponseDTO{
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

	var req AddClientAPIsRequestDTO
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		resp.BadRequestBody(w)
		return
	}

	// The DTO's rules existed but were never run, so an empty list or a nil UUID
	// reached the service.
	if err := req.Validate(); err != nil {
		resp.ValidationError(w, err)
		return
	}

	// Get tenant from context
	tenant := middleware.AuthFromRequest(r).Tenant
	if tenant == nil {
		resp.Error(w, http.StatusUnauthorized, "Tenant not found in context")
		return
	}

	// The actor is the second trust boundary on a grant mutation: without it the
	// middleware-supplied tenant is the only one.
	user := middleware.AuthFromRequest(r).User
	if user == nil {
		resp.Error(w, http.StatusUnauthorized, "User not found in context")
		return
	}

	// Add APIs to auth client
	err = h.ClientService.AddClientAPIs(r.Context(), tenant.TenantID, ClientUUID, req.APIUUIDs, user.UserUUID)
	if err != nil {
		resp.HandleServiceError(w, r, "Failed to add APIs to auth client", err)
		return
	}

	authCtxAddAPIs := middleware.AuthFromRequest(r)
	var actorUserIDAddAPIs *int64
	if authCtxAddAPIs.User != nil {
		actorUserIDAddAPIs = &authCtxAddAPIs.User.UserID
	}
	changesJSONAddAPIs, _ := json.Marshal(map[string]any{"update": req})
	h.logAudit(r, tenant.TenantID, actorUserIDAddAPIs, "add_apis", "client", ClientUUID.String(), &ClientUUID, string(changesJSONAddAPIs), "success")

	response := SuccessResponseDTO{
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
	tenant := middleware.AuthFromRequest(r).Tenant
	if tenant == nil {
		resp.Error(w, http.StatusUnauthorized, "Tenant not found in context")
		return
	}

	// Remove API from auth client
	// The actor is the second trust boundary on a grant mutation: without it the
	// middleware-supplied tenant is the only one.
	user := middleware.AuthFromRequest(r).User
	if user == nil {
		resp.Error(w, http.StatusUnauthorized, "User not found in context")
		return
	}

	err = h.ClientService.RemoveClientAPI(r.Context(), tenant.TenantID, ClientUUID, apiUUID, user.UserUUID)
	if err != nil {
		resp.HandleServiceError(w, r, "Failed to remove API from auth client", err)
		return
	}

	authCtxRemoveAPI := middleware.AuthFromRequest(r)
	var actorUserIDRemoveAPI *int64
	if authCtxRemoveAPI.User != nil {
		actorUserIDRemoveAPI = &authCtxRemoveAPI.User.UserID
	}
	changesJSONRemoveAPI, _ := json.Marshal(map[string]any{"update": map[string]any{"api_uuid": apiUUID.String()}})
	h.logAudit(r, tenant.TenantID, actorUserIDRemoveAPI, "remove_api", "client", ClientUUID.String(), &ClientUUID, string(changesJSONRemoveAPI), "success")

	response := SuccessResponseDTO{
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
	tenant := middleware.AuthFromRequest(r).Tenant
	if tenant == nil {
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

	response := ClientAPIPermissionsResponseDTO{
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

	var req AddClientAPIPermissionsRequestDTO
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		resp.BadRequestBody(w)
		return
	}

	if err := req.Validate(); err != nil {
		resp.ValidationError(w, err)
		return
	}

	// Get tenant from context
	tenant := middleware.AuthFromRequest(r).Tenant
	if tenant == nil {
		resp.Error(w, http.StatusUnauthorized, "Tenant not found in context")
		return
	}

	// Add permissions to auth client API
	// The actor is the second trust boundary on a grant mutation: without it the
	// middleware-supplied tenant is the only one.
	user := middleware.AuthFromRequest(r).User
	if user == nil {
		resp.Error(w, http.StatusUnauthorized, "User not found in context")
		return
	}

	err = h.ClientService.AddClientAPIPermissions(r.Context(), tenant.TenantID, ClientUUID, apiUUID, req.PermissionUUIDs, user.UserUUID)
	if err != nil {
		resp.HandleServiceError(w, r, "Failed to add permissions to auth client API", err)
		return
	}

	authCtxAddPerms := middleware.AuthFromRequest(r)
	var actorUserIDAddPerms *int64
	if authCtxAddPerms.User != nil {
		actorUserIDAddPerms = &authCtxAddPerms.User.UserID
	}
	changesJSONAddPerms, _ := json.Marshal(map[string]any{"update": req})
	h.logAudit(r, tenant.TenantID, actorUserIDAddPerms, "add_api_permissions", "client", ClientUUID.String(), &ClientUUID, string(changesJSONAddPerms), "success")

	response := SuccessResponseDTO{
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
	tenant := middleware.AuthFromRequest(r).Tenant
	if tenant == nil {
		resp.Error(w, http.StatusUnauthorized, "Tenant not found in context")
		return
	}

	// Remove permission from auth client API
	// The actor is the second trust boundary on a grant mutation: without it the
	// middleware-supplied tenant is the only one.
	user := middleware.AuthFromRequest(r).User
	if user == nil {
		resp.Error(w, http.StatusUnauthorized, "User not found in context")
		return
	}

	err = h.ClientService.RemoveClientAPIPermission(r.Context(), tenant.TenantID, ClientUUID, apiUUID, permissionUUID, user.UserUUID)
	if err != nil {
		resp.HandleServiceError(w, r, "Failed to remove permission from auth client API", err)
		return
	}

	authCtxRemovePerm := middleware.AuthFromRequest(r)
	var actorUserIDRemovePerm *int64
	if authCtxRemovePerm.User != nil {
		actorUserIDRemovePerm = &authCtxRemovePerm.User.UserID
	}
	changesJSONRemovePerm, _ := json.Marshal(map[string]any{"update": map[string]any{"api_uuid": apiUUID.String(), "permission_uuid": permissionUUID.String()}})
	h.logAudit(r, tenant.TenantID, actorUserIDRemovePerm, "remove_api_permission", "client", ClientUUID.String(), &ClientUUID, string(changesJSONRemovePerm), "success")

	response := SuccessResponseDTO{
		Message: "Permission removed from auth client API successfully",
	}

	resp.Success(w, response, "Permission removed from auth client API successfully")
}

// Convert result to DTO
func toClientResponseDTO(r ClientServiceDataResult) ClientResponseDTO {
	result := ClientResponseDTO{
		ClientUUID:                       r.ClientUUID,
		Identifier:                       r.Identifier,
		ServiceUUID:                      r.ServiceUUID,
		Name:                             r.Name,
		DisplayName:                      r.DisplayName,
		ClientType:                       r.ClientType,
		Domain:                           r.Domain,
		Status:                           r.Status,
		IsDefault:                        r.IsDefault,
		IsSystem:                         r.IsSystem,
		BrandingUUID:                     brandingUUIDToStringPtr(r.BrandingUUID),
		AllowRegistration:                r.AllowRegistration,
		AllowMagicLink:                   r.AllowMagicLink,
		BackchannelLogoutURI:             r.BackchannelLogoutURI,
		FrontchannelLogoutURI:            r.FrontchannelLogoutURI,
		BackchannelLogoutSessionRequired: r.BackchannelLogoutSessionRequired,
		DPoPRequired:                     r.DPoPRequired,
		TokenEndpointAuthMethod:          r.TokenEndpointAuthMethod,
		GrantTypes:                       r.GrantTypes,
		ResponseTypes:                    r.ResponseTypes,
		AllowedScopes:                    nonNilStrings(r.AllowedScopes),
		RequireConsent:                   r.RequireConsent,
		AccessTokenTTL:                   r.AccessTokenTTL,
		RefreshTokenTTL:                  r.RefreshTokenTTL,
		RequirePKCE:                      r.RequirePKCE,
		RequiredACR:                      r.RequiredACR,
		SessionIdleTimeout:               r.SessionIdleTimeout,
		SessionAbsoluteTimeout:           r.SessionAbsoluteTimeout,
		CreatedAt:                        r.CreatedAt,
		UpdatedAt:                        r.UpdatedAt,
	}

	if r.IdentityProvider != nil {
		result.IdentityProvider = &IdentityProviderResponseDTO{
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
	if r.Connections != nil && len(*r.Connections) > 0 {
		result.Connections = toClientIdentityProviderDTOs(*r.Connections)
	}

	if r.ClientURIs != nil && len(*r.ClientURIs) > 0 {
		result.URIs = make([]ClientURIResponseDTO, len(*r.ClientURIs))
		for i, uri := range *r.ClientURIs {
			result.URIs[i] = ClientURIResponseDTO(uri)
		}
	}

	// Map Permissions if present
	if r.Permissions != nil {
		permissions := make([]PermissionResponseDTO, len(*r.Permissions))
		for i, permission := range *r.Permissions {
			permissions[i] = PermissionResponseDTO{
				PermissionUUID: permission.PermissionUUID,
				Name:           permission.Name,
				Description:    permission.Description,
				Status:         permission.Status,
				IsSystem:       permission.IsSystem,
				CreatedAt:      permission.CreatedAt,
				UpdatedAt:      permission.UpdatedAt,
			}
		}
		result.Permissions = &permissions
	}

	return result
}

// toClientIdentityProviderDTOs maps identity provider connection service results
// to their wire DTOs. Shared by the full client response and the connection list
// endpoint so the mapping lives in one place.
func toClientIdentityProviderDTOs(connections []ClientIdentityProviderServiceDataResult) []ClientIdentityProviderDTO {
	result := make([]ClientIdentityProviderDTO, 0, len(connections))
	for _, connection := range connections {
		result = append(result, ClientIdentityProviderDTO{
			ClientIdentityProviderUUID: connection.ClientIdentityProviderUUID,
			IdentityProvider: IdentityProviderResponseDTO{
				IdentityProviderUUID: connection.IdentityProvider.IdentityProviderUUID,
				Name:                 connection.IdentityProvider.Name,
				DisplayName:          connection.IdentityProvider.DisplayName,
				Provider:             connection.IdentityProvider.Provider,
				ProviderType:         connection.IdentityProvider.ProviderType,
				Identifier:           connection.IdentityProvider.Identifier,
				Status:               connection.IdentityProvider.Status,
				IsDefault:            connection.IdentityProvider.IsDefault,
				IsSystem:             connection.IdentityProvider.IsSystem,
				CreatedAt:            connection.IdentityProvider.CreatedAt,
				UpdatedAt:            connection.IdentityProvider.UpdatedAt,
			},
			IsDefault:    connection.IsDefault,
			Enabled:      connection.Enabled,
			DisplayOrder: connection.DisplayOrder,
			CreatedAt:    connection.CreatedAt,
			UpdatedAt:    connection.UpdatedAt,
		})
	}
	return result
}

// GetConnections retrieves the identity provider connections enabled on a client.
func (h *ClientHandler) GetConnections(w http.ResponseWriter, r *http.Request) {
	tenant := middleware.AuthFromRequest(r).Tenant
	if tenant == nil {
		resp.Error(w, http.StatusUnauthorized, "Tenant not found in context")
		return
	}

	ClientUUID, err := uuid.Parse(chi.URLParam(r, "client_uuid"))
	if err != nil {
		resp.Error(w, http.StatusBadRequest, "Invalid auth client UUID")
		return
	}

	connections, err := h.ClientService.GetConnections(r.Context(), ClientUUID, tenant.TenantID)
	if err != nil {
		resp.HandleServiceError(w, r, "Failed to get identity provider connections", err)
		return
	}

	response := ClientIdentityProvidersResponseDTO{
		Connections: toClientIdentityProviderDTOs(connections),
	}

	resp.Success(w, response, "Identity provider connections retrieved successfully")
}

// AddConnection connects an identity provider to a client.
func (h *ClientHandler) AddConnection(w http.ResponseWriter, r *http.Request) {
	tenant := middleware.AuthFromRequest(r).Tenant
	if tenant == nil {
		resp.Error(w, http.StatusUnauthorized, "Tenant not found in context")
		return
	}
	user := middleware.AuthFromRequest(r).User
	if user == nil {
		resp.Error(w, http.StatusUnauthorized, "User not found in context")
		return
	}

	ClientUUID, err := uuid.Parse(chi.URLParam(r, "client_uuid"))
	if err != nil {
		resp.Error(w, http.StatusBadRequest, "Invalid auth client UUID")
		return
	}

	var req AddClientIdentityProviderRequestDTO
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		resp.BadRequestBody(w)
		return
	}
	if err := req.Validate(); err != nil {
		resp.ValidationError(w, err)
		return
	}

	identityProviderUUID, err := uuid.Parse(req.IdentityProviderUUID)
	if err != nil {
		resp.Error(w, http.StatusBadRequest, "Invalid identity provider UUID")
		return
	}

	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}

	Client, err := h.ClientService.AddConnection(r.Context(), ClientUUID, tenant.TenantID, identityProviderUUID, req.IsDefault, enabled, req.DisplayOrder, user.UserUUID)
	if err != nil {
		resp.HandleServiceError(w, r, "Failed to connect identity provider", err)
		return
	}

	dtoRes := toClientResponseDTO(*Client)

	actorUserIDAddConn := &user.UserID
	changesJSONAddConn, _ := json.Marshal(map[string]any{"update": req})
	h.logAudit(r, tenant.TenantID, actorUserIDAddConn, "add_connection", "client", ClientUUID.String(), &ClientUUID, string(changesJSONAddConn), "success")

	resp.Created(w, dtoRes, "Identity provider connected successfully")
}

// UpdateConnection updates an identity provider connection on a client.
func (h *ClientHandler) UpdateConnection(w http.ResponseWriter, r *http.Request) {
	tenant := middleware.AuthFromRequest(r).Tenant
	if tenant == nil {
		resp.Error(w, http.StatusUnauthorized, "Tenant not found in context")
		return
	}
	user := middleware.AuthFromRequest(r).User
	if user == nil {
		resp.Error(w, http.StatusUnauthorized, "User not found in context")
		return
	}

	ClientUUID, err := uuid.Parse(chi.URLParam(r, "client_uuid"))
	if err != nil {
		resp.Error(w, http.StatusBadRequest, "Invalid auth client UUID")
		return
	}
	connectionUUID, err := uuid.Parse(chi.URLParam(r, "client_identity_provider_uuid"))
	if err != nil {
		resp.Error(w, http.StatusBadRequest, "Invalid identity provider connection UUID")
		return
	}

	var req UpdateClientIdentityProviderRequestDTO
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		resp.BadRequestBody(w)
		return
	}
	if err := req.Validate(); err != nil {
		resp.ValidationError(w, err)
		return
	}

	// Pass the pointers straight through: the service leaves a nil field at its
	// current stored value rather than resetting it.
	Client, err := h.ClientService.UpdateConnection(r.Context(), ClientUUID, tenant.TenantID, connectionUUID, req.IsDefault, req.Enabled, req.DisplayOrder, user.UserUUID)
	if err != nil {
		resp.HandleServiceError(w, r, "Failed to update identity provider connection", err)
		return
	}

	dtoRes := toClientResponseDTO(*Client)

	actorUserIDUpdateConn := &user.UserID
	changesJSONUpdateConn, _ := json.Marshal(map[string]any{"update": req})
	h.logAudit(r, tenant.TenantID, actorUserIDUpdateConn, "update_connection", "client", ClientUUID.String(), &ClientUUID, string(changesJSONUpdateConn), "success")

	resp.Success(w, dtoRes, "Identity provider connection updated successfully")
}

// RemoveConnection detaches an identity provider from a client.
func (h *ClientHandler) RemoveConnection(w http.ResponseWriter, r *http.Request) {
	tenant := middleware.AuthFromRequest(r).Tenant
	if tenant == nil {
		resp.Error(w, http.StatusUnauthorized, "Tenant not found in context")
		return
	}
	user := middleware.AuthFromRequest(r).User
	if user == nil {
		resp.Error(w, http.StatusUnauthorized, "User not found in context")
		return
	}

	ClientUUID, err := uuid.Parse(chi.URLParam(r, "client_uuid"))
	if err != nil {
		resp.Error(w, http.StatusBadRequest, "Invalid auth client UUID")
		return
	}
	connectionUUID, err := uuid.Parse(chi.URLParam(r, "client_identity_provider_uuid"))
	if err != nil {
		resp.Error(w, http.StatusBadRequest, "Invalid identity provider connection UUID")
		return
	}

	Client, err := h.ClientService.RemoveConnection(r.Context(), ClientUUID, tenant.TenantID, connectionUUID, user.UserUUID)
	if err != nil {
		resp.HandleServiceError(w, r, "Failed to remove identity provider connection", err)
		return
	}

	dtoRes := toClientResponseDTO(*Client)

	actorUserIDRemoveConn := &user.UserID
	changesJSONRemoveConn, _ := json.Marshal(map[string]any{"update": map[string]any{"connection_uuid": connectionUUID.String()}})
	h.logAudit(r, tenant.TenantID, actorUserIDRemoveConn, "remove_connection", "client", ClientUUID.String(), &ClientUUID, string(changesJSONRemoveConn), "success")

	resp.Success(w, dtoRes, "Identity provider connection removed successfully")
}

func parseOptionalUUID(s *string) *uuid.UUID {
	if s == nil || *s == "" {
		return nil
	}
	parsed, err := uuid.Parse(*s)
	if err != nil {
		return nil
	}
	return &parsed
}

func boolValue(p *bool, def bool) bool {
	if p == nil {
		return def
	}
	return *p
}

func brandingUUIDToStringPtr(b *uuid.UUID) *string {
	if b == nil {
		return nil
	}
	s := b.String()
	return &s
}

// nonNilStrings keeps a JSON array from serializing as null, which forces every
// consumer to null-check a list that is conceptually always present.
func nonNilStrings(v []string) []string {
	if v == nil {
		return []string{}
	}
	return v
}

// findCreatedURI picks the URI matching the requested value and type out of the
// client's full URI list. Matching on the pair rather than taking a position keeps
// the response correct regardless of ordering, and returns nil instead of
// panicking when nothing matches.
func findCreatedURI(uris *[]ClientURIServiceDataResult, uri, uriType string) *ClientURIServiceDataResult {
	if uris == nil {
		return nil
	}
	for i := range *uris {
		candidate := &(*uris)[i]
		if candidate.URI == uri && candidate.Type == uriType {
			return candidate
		}
	}
	return nil
}
