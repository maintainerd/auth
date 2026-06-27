package idp

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/platform/middleware"
	"github.com/maintainerd/auth/internal/platform/pagination"
	"github.com/maintainerd/auth/internal/platform/ptr"
	resp "github.com/maintainerd/auth/internal/platform/response"
	"github.com/maintainerd/auth/internal/shared"
)

// AuthFlowHandler handles signup flow management operations.
//
// This handler manages tenant-scoped signup flows (user registration processes).
// Signup flows define the registration experience for users, including which roles
// are automatically assigned upon signup. All operations are tenant-isolated -
// middleware validates tenant access and stores it in the request context.
type AuthFlowHandler struct {
	authFlowService AuthFlowService
}

// NewAuthFlowHandler creates a new signup flow handler instance.
func NewAuthFlowHandler(authFlowService AuthFlowService) *AuthFlowHandler {
	return &AuthFlowHandler{
		authFlowService: authFlowService,
	}
}

// GetAll retrieves all signup flows for the tenant with pagination and filters.
//
// GET /auth-flows
//
// Returns a paginated list of signup flows belonging to the authenticated tenant.
// Supports filtering by name, identifier, status, and auth client UUID.
func (h *AuthFlowHandler) GetAll(w http.ResponseWriter, r *http.Request) {
	// Get tenant from context (middleware already validated access)
	tenant := middleware.AuthFromRequest(r).Tenant
	if tenant == nil {
		resp.Error(w, http.StatusUnauthorized, "Tenant not found in context")
		return
	}

	// Parse query parameters
	q := r.URL.Query()

	// Parse pagination parameters

	// Parse status filter
	var status []string
	if v := q.Get("status"); v != "" {
		status = append(status, v)
	}

	// Build filter DTO for validation
	filter := AuthFlowFilterDTO{
		Name:                 ptr.PtrOrNil(q.Get("name")),
		Identifier:           ptr.PtrOrNil(q.Get("identifier")),
		Status:               status,
		ClientUUID:           ptr.PtrOrNil(q.Get("client_id")),
		PaginationRequestDTO: pagination.ParseQuery(r),
	}

	// Validate filter parameters
	if err := filter.Validate(); err != nil {
		resp.ValidationError(w, err)
		return
	}

	// Parse and validate auth client UUID if provided
	var ClientUUIDPtr *uuid.UUID
	if filter.ClientUUID != nil {
		// Already validated as UUID by DTO
		ClientUUID, _ := uuid.Parse(*filter.ClientUUID)
		ClientUUIDPtr = &ClientUUID
	}

	// Fetch signup flows from service layer
	result, err := h.authFlowService.GetAll(r.Context(), tenant.TenantID, filter.Name, filter.Identifier, filter.Status, ClientUUIDPtr, filter.Page, filter.Limit, filter.SortBy, filter.SortOrder)
	if err != nil {
		resp.HandleServiceError(w, r, "Failed to get signup flows", err)
		return
	}

	// Build paginated response
	response := PaginatedResponseDTO[AuthFlowResponseDTO]{
		Rows:       toAuthFlowResponseDtoList(result.Data),
		Total:      result.Total,
		Page:       result.Page,
		Limit:      result.Limit,
		TotalPages: result.TotalPages,
	}

	resp.Success(w, response, "Signup flows retrieved successfully")
}

// Get retrieves a specific signup flow by UUID.
//
// GET /auth-flows/{auth_flow_uuid}
//
// Returns detailed information about a single signup flow. The service layer
// validates that the signup flow belongs to the tenant.
func (h *AuthFlowHandler) Get(w http.ResponseWriter, r *http.Request) {
	// Get tenant from context (middleware already validated access)
	tenant := middleware.AuthFromRequest(r).Tenant
	if tenant == nil {
		resp.Error(w, http.StatusUnauthorized, "Tenant not found in context")
		return
	}

	// Parse and validate signup flow UUID from URL parameter
	authFlowUUIDStr := chi.URLParam(r, "auth_flow_uuid")
	authFlowUUID, err := uuid.Parse(authFlowUUIDStr)
	if err != nil {
		resp.Error(w, http.StatusBadRequest, "Invalid signup flow UUID")
		return
	}

	// Fetch signup flow (service validates tenant ownership)
	authFlow, err := h.authFlowService.GetByUUID(r.Context(), authFlowUUID, tenant.TenantID)
	if err != nil {
		resp.HandleServiceError(w, r, "Signup flow not found", err)
		return
	}

	resp.Success(w, toAuthFlowResponseDTO(*authFlow), "Signup flow retrieved successfully")
}

// Create creates a new signup flow for the tenant.
//
// POST /auth-flows
//
// Creates a new signup flow defining the user registration process. The flow
// includes configuration for the signup experience and is linked to an auth client.
func (h *AuthFlowHandler) Create(w http.ResponseWriter, r *http.Request) {
	// Get tenant from context (middleware already validated access)
	tenant := middleware.AuthFromRequest(r).Tenant
	if tenant == nil {
		resp.Error(w, http.StatusUnauthorized, "Tenant not found in context")
		return
	}

	// Decode and validate request body
	var req AuthFlowCreateRequestDTO
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		resp.BadRequestBody(w)
		return
	}

	if err := req.Validate(); err != nil {
		resp.ValidationError(w, err)
		return
	}

	// Parse auth client UUID (already validated as UUID by DTO)
	ClientUUID, _ := uuid.Parse(req.ClientUUID)

	// Parse optional branding UUID
	brandingUUID := parseOptionalUUID(req.BrandingUUID)

	// Set default status if not provided
	status := shared.StatusActive
	if req.Status != nil {
		status = *req.Status
	}

	destination := req.Destination
	if destination == "" {
		destination = shared.DestinationIdentity
	}

	// Create signup flow
	authFlow, err := h.authFlowService.Create(
		r.Context(),
		tenant.TenantID,
		req.Name,
		req.Description,
		status,
		destination,
		ClientUUID,
		brandingUUID,
		parseUUIDList(req.RoleIDs),
		parseUUIDList(req.ClientURIIDs),
		boolValue(req.AllowRegistration, true),
		boolValue(req.VerificationRequired, false),
		strValue(req.RequiredFields, "[]"),
	)
	if err != nil {
		resp.HandleServiceError(w, r, "Failed to create signup flow", err)
		return
	}

	resp.Created(w, toAuthFlowResponseDTO(*authFlow), "Signup flow created successfully")
}

// Update updates an existing signup flow.
//
// PUT /auth-flows/{auth_flow_uuid}
//
// Updates the configuration and settings of an existing signup flow.
// The service layer validates that the signup flow belongs to the tenant.
func (h *AuthFlowHandler) Update(w http.ResponseWriter, r *http.Request) {
	// Get tenant from context (middleware already validated access)
	tenant := middleware.AuthFromRequest(r).Tenant
	if tenant == nil {
		resp.Error(w, http.StatusUnauthorized, "Tenant not found in context")
		return
	}

	// Parse and validate signup flow UUID from URL parameter
	authFlowUUIDStr := chi.URLParam(r, "auth_flow_uuid")
	authFlowUUID, err := uuid.Parse(authFlowUUIDStr)
	if err != nil {
		resp.Error(w, http.StatusBadRequest, "Invalid signup flow UUID")
		return
	}

	// Decode and validate request body
	var req AuthFlowUpdateRequestDTO
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
	if req.Status != nil {
		status = *req.Status
	}

	// Parse optional branding UUID (nil clears it)
	brandingUUID := parseOptionalUUID(req.BrandingUUID)

	// Update signup flow (service validates tenant ownership)
	authFlow, err := h.authFlowService.Update(
		r.Context(),
		authFlowUUID,
		tenant.TenantID,
		req.Name,
		req.Description,
		status,
		brandingUUID,
		parseUUIDList(req.RoleIDs),
		parseUUIDList(req.ClientURIIDs),
		boolValue(req.AllowRegistration, true),
		boolValue(req.VerificationRequired, false),
		strValue(req.RequiredFields, "[]"),
	)
	if err != nil {
		resp.HandleServiceError(w, r, "Failed to update signup flow", err)
		return
	}

	resp.Success(w, toAuthFlowResponseDTO(*authFlow), "Signup flow updated successfully")
}

// Delete deletes a signup flow.
//
// DELETE /auth-flows/{auth_flow_uuid}
//
// Permanently deletes a signup flow from the tenant. This will also remove
// any associated role assignments for the flow.
func (h *AuthFlowHandler) Delete(w http.ResponseWriter, r *http.Request) {
	// Get tenant from context (middleware already validated access)
	tenant := middleware.AuthFromRequest(r).Tenant
	if tenant == nil {
		resp.Error(w, http.StatusUnauthorized, "Tenant not found in context")
		return
	}

	// Parse and validate signup flow UUID from URL parameter
	authFlowUUIDStr := chi.URLParam(r, "auth_flow_uuid")
	authFlowUUID, err := uuid.Parse(authFlowUUIDStr)
	if err != nil {
		resp.Error(w, http.StatusBadRequest, "Invalid signup flow UUID")
		return
	}

	// Delete signup flow (service validates tenant ownership)
	authFlow, err := h.authFlowService.Delete(r.Context(), authFlowUUID, tenant.TenantID)
	if err != nil {
		resp.HandleServiceError(w, r, "Failed to delete signup flow", err)
		return
	}

	resp.Success(w, toAuthFlowResponseDTO(*authFlow), "Signup flow deleted successfully")
}

// UpdateStatus updates the status of a signup flow.
//
// PATCH /auth-flows/{auth_flow_uuid}/status
//
// Updates only the status field of a signup flow (e.g., active, inactive).
// This is a convenience endpoint for status-only updates.
func (h *AuthFlowHandler) UpdateStatus(w http.ResponseWriter, r *http.Request) {
	// Get tenant from context (middleware already validated access)
	tenant := middleware.AuthFromRequest(r).Tenant
	if tenant == nil {
		resp.Error(w, http.StatusUnauthorized, "Tenant not found in context")
		return
	}

	// Parse and validate signup flow UUID from URL parameter
	authFlowUUIDStr := chi.URLParam(r, "auth_flow_uuid")
	authFlowUUID, err := uuid.Parse(authFlowUUIDStr)
	if err != nil {
		resp.Error(w, http.StatusBadRequest, "Invalid signup flow UUID")
		return
	}

	// Decode and validate request body
	var req AuthFlowUpdateStatusRequestDTO
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		resp.BadRequestBody(w)
		return
	}

	if err := req.Validate(); err != nil {
		resp.ValidationError(w, err)
		return
	}

	// Update status (service validates tenant ownership)
	authFlow, err := h.authFlowService.UpdateStatus(r.Context(), authFlowUUID, tenant.TenantID, req.Status)
	if err != nil {
		resp.HandleServiceError(w, r, "Failed to update signup flow status", err)
		return
	}

	resp.Success(w, toAuthFlowResponseDTO(*authFlow), "Signup flow status updated successfully")
}

// AssignRoles assigns roles to a signup flow.
//
// POST /auth-flows/{auth_flow_uuid}/roles
//
// Associates one or more roles with a signup flow. Users who complete registration
// through this flow will automatically be assigned these roles.
func (h *AuthFlowHandler) AssignRoles(w http.ResponseWriter, r *http.Request) {
	// Get tenant from context (middleware already validated access)
	tenant := middleware.AuthFromRequest(r).Tenant
	if tenant == nil {
		resp.Error(w, http.StatusUnauthorized, "Tenant not found in context")
		return
	}

	// Parse and validate signup flow UUID from URL parameter
	authFlowUUIDStr := chi.URLParam(r, "auth_flow_uuid")
	if authFlowUUIDStr == "" {
		resp.Error(w, http.StatusBadRequest, "Invalid signup flow UUID", "UUID parameter is required")
		return
	}

	authFlowUUID, err := uuid.Parse(authFlowUUIDStr)
	if err != nil {
		resp.Error(w, http.StatusBadRequest, "Invalid UUID format")
		return
	}

	// Decode and validate request body
	var req AuthFlowAssignRolesRequestDTO
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		resp.BadRequestBody(w)
		return
	}

	if err := req.Validate(); err != nil {
		resp.ValidationError(w, err)
		return
	}

	// Parse role UUIDs (each already validated as UUID by DTO)
	roleUUIDs := make([]uuid.UUID, len(req.RoleUUIDs))
	for i, roleUUIDStr := range req.RoleUUIDs {
		roleUUIDs[i], _ = uuid.Parse(roleUUIDStr)
	}

	// Assign roles to signup flow (service validates tenant ownership)
	roles, err := h.authFlowService.AssignRoles(r.Context(), authFlowUUID, tenant.TenantID, roleUUIDs)
	if err != nil {
		resp.HandleServiceError(w, r, "Failed to assign roles", err)
		return
	}

	// Map service results to DTOs
	response := make([]RoleResponseDTO, len(roles))
	for i, role := range roles {
		response[i] = RoleResponseDTO{
			RoleUUID:    role.RoleUUID,
			Name:        role.RoleName,
			Description: role.RoleDescription,
			IsDefault:   role.RoleIsDefault,
			IsSystem:    role.RoleIsSystem,
			Status:      role.RoleStatus,
			CreatedAt:   role.CreatedAt,
			UpdatedAt:   role.UpdatedAt,
		}
	}

	resp.Success(w, response, "Roles assigned successfully")
}

// GetRoles retrieves all roles assigned to a signup flow.
//
// GET /auth-flows/{auth_flow_uuid}/roles
//
// Returns a paginated list of roles that are automatically assigned to users
// who complete registration through this signup flow.
func (h *AuthFlowHandler) GetRoles(w http.ResponseWriter, r *http.Request) {
	// Get tenant from context (middleware already validated access)
	tenant := middleware.AuthFromRequest(r).Tenant
	if tenant == nil {
		resp.Error(w, http.StatusUnauthorized, "Tenant not found in context")
		return
	}

	// Parse and validate signup flow UUID from URL parameter
	authFlowUUIDStr := chi.URLParam(r, "auth_flow_uuid")
	if authFlowUUIDStr == "" {
		resp.Error(w, http.StatusBadRequest, "Invalid signup flow UUID", "UUID parameter is required")
		return
	}

	authFlowUUID, err := uuid.Parse(authFlowUUIDStr)
	if err != nil {
		resp.Error(w, http.StatusBadRequest, "Invalid UUID format")
		return
	}

	// Build pagination DTO for validation
	reqParams := pagination.ParseQuery(r)

	if err := reqParams.Validate(); err != nil {
		resp.ValidationError(w, err)
		return
	}

	// Fetch roles for the signup flow (service validates tenant ownership)
	result, err := h.authFlowService.GetRoles(r.Context(), authFlowUUID, tenant.TenantID, reqParams.Page, reqParams.Limit)
	if err != nil {
		resp.HandleServiceError(w, r, "Failed to retrieve roles", err)
		return
	}

	// Map service results to DTOs
	rows := make([]RoleResponseDTO, len(result.Data))
	for i, role := range result.Data {
		rows[i] = RoleResponseDTO{
			RoleUUID:    role.RoleUUID,
			Name:        role.RoleName,
			Description: role.RoleDescription,
			IsDefault:   role.RoleIsDefault,
			IsSystem:    role.RoleIsSystem,
			Status:      role.RoleStatus,
			CreatedAt:   role.CreatedAt,
			UpdatedAt:   role.UpdatedAt,
		}
	}

	// Build paginated response
	response := PaginatedResponseDTO[RoleResponseDTO]{
		Rows:       rows,
		Total:      result.Total,
		Page:       result.Page,
		Limit:      result.Limit,
		TotalPages: result.TotalPages,
	}

	resp.Success(w, response, "Roles retrieved successfully")
}

// RemoveRole removes a role from a signup flow.
//
// DELETE /auth-flows/{auth_flow_uuid}/roles/{role_uuid}
//
// Removes the association between a role and a signup flow. Users who register
// through this flow will no longer automatically receive this role.
func (h *AuthFlowHandler) RemoveRole(w http.ResponseWriter, r *http.Request) {
	// Get tenant from context (middleware already validated access)
	tenant := middleware.AuthFromRequest(r).Tenant
	if tenant == nil {
		resp.Error(w, http.StatusUnauthorized, "Tenant not found in context")
		return
	}

	// Parse and validate UUIDs from URL parameters
	authFlowUUIDStr := chi.URLParam(r, "auth_flow_uuid")
	roleUUIDStr := chi.URLParam(r, "role_uuid")

	if authFlowUUIDStr == "" || roleUUIDStr == "" {
		resp.Error(w, http.StatusBadRequest, "Invalid parameters", "Both signup flow UUID and role UUID are required")
		return
	}

	authFlowUUID, err := uuid.Parse(authFlowUUIDStr)
	if err != nil {
		resp.Error(w, http.StatusBadRequest, "Invalid signup flow UUID format")
		return
	}

	roleUUID, err := uuid.Parse(roleUUIDStr)
	if err != nil {
		resp.Error(w, http.StatusBadRequest, "Invalid role UUID format")
		return
	}

	// Remove role from signup flow (service validates tenant ownership)
	if err := h.authFlowService.RemoveRole(r.Context(), authFlowUUID, tenant.TenantID, roleUUID); err != nil {
		resp.HandleServiceError(w, r, "Failed to remove role", err)
		return
	}

	resp.Success(w, nil, "Role removed successfully")
}

// Helper functions for converting service data to response DTOs

// toAuthFlowResponseDTO converts a service result to a signup flow response DTO.
func toAuthFlowResponseDTO(sf AuthFlowServiceDataResult) AuthFlowResponseDTO {
	dto := AuthFlowResponseDTO{
		AuthFlowUUID: sf.AuthFlowUUID.String(),
		Name:         sf.Name,
		Description:  sf.Description,
		Identifier:   sf.Identifier,
		Destination: sf.Destination,
		Status:       sf.Status,
		ClientUUID:   sf.ClientUUID.String(),
		CreatedAt:    sf.CreatedAt,
		UpdatedAt:    sf.UpdatedAt,
	}
	if sf.BrandingUUID != nil {
		dto.BrandingUUID = sf.BrandingUUID.String()
	}
	dto.AllowRegistration = sf.AllowRegistration
	dto.VerificationRequired = sf.VerificationRequired
	dto.RequiredFields = sf.RequiredFields
	return dto
}

// parseOptionalUUID parses an optional UUID string pointer into a *uuid.UUID,
// returning nil when absent, empty, or unparseable.
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

// parseUUIDList parses a slice of UUID strings, preserving the nil-vs-empty
// distinction (nil input → nil output, so "field omitted" stays distinguishable
// from "field set to empty" for replace-semantics on update).
func parseUUIDList(ss []string) []uuid.UUID {
	if ss == nil {
		return nil
	}
	out := make([]uuid.UUID, 0, len(ss))
	for _, s := range ss {
		if parsed, err := uuid.Parse(s); err == nil {
			out = append(out, parsed)
		}
	}
	return out
}

// toAuthFlowResponseDtoList converts a list of service results to signup flow response DTOs.
func toAuthFlowResponseDtoList(sfList []AuthFlowServiceDataResult) []AuthFlowResponseDTO {
	result := make([]AuthFlowResponseDTO, len(sfList))
	for i, sf := range sfList {
		result[i] = toAuthFlowResponseDTO(sf)
	}
	return result
}

func toAuthFlowCallbackURIResponseDTO(c AuthFlowCallbackURIServiceDataResult) AuthFlowCallbackURIResponseDTO {
	return AuthFlowCallbackURIResponseDTO{
		AuthFlowCallbackURIUUID: c.AuthFlowCallbackURIUUID.String(),
		AuthFlowUUID:            c.AuthFlowUUID.String(),
		ClientURIUUID:           c.ClientURIUUID.String(),
		URI:                     c.URI,
		CreatedAt:               c.CreatedAt,
	}
}

// AssignCallbackURIs attaches one or more of the client's registered URIs to the
// auth flow as callback URIs.
//
// POST /auth_flows/{auth_flow_uuid}/callback_uris
func (h *AuthFlowHandler) AssignCallbackURIs(w http.ResponseWriter, r *http.Request) {
	tenant := middleware.AuthFromRequest(r).Tenant
	if tenant == nil {
		resp.Error(w, http.StatusUnauthorized, "Tenant not found in context")
		return
	}

	authFlowUUID, err := uuid.Parse(chi.URLParam(r, "auth_flow_uuid"))
	if err != nil {
		resp.Error(w, http.StatusBadRequest, "Invalid auth flow UUID")
		return
	}

	var req AuthFlowAssignCallbackURIsRequestDTO
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		resp.BadRequestBody(w)
		return
	}
	if err := req.Validate(); err != nil {
		resp.ValidationError(w, err)
		return
	}

	clientURIUUIDs := make([]uuid.UUID, len(req.ClientURIUUIDs))
	for i, s := range req.ClientURIUUIDs {
		clientURIUUIDs[i], _ = uuid.Parse(s)
	}

	result, err := h.authFlowService.AssignCallbackURIs(r.Context(), authFlowUUID, tenant.TenantID, clientURIUUIDs)
	if err != nil {
		resp.HandleServiceError(w, r, "Failed to assign callback URIs", err)
		return
	}

	response := make([]AuthFlowCallbackURIResponseDTO, len(result))
	for i, c := range result {
		response[i] = toAuthFlowCallbackURIResponseDTO(c)
	}

	resp.Success(w, response, "Callback URIs assigned successfully")
}

// GetCallbackURIs returns the callback URIs attached to an auth flow.
//
// GET /auth_flows/{auth_flow_uuid}/callback_uris
func (h *AuthFlowHandler) GetCallbackURIs(w http.ResponseWriter, r *http.Request) {
	tenant := middleware.AuthFromRequest(r).Tenant
	if tenant == nil {
		resp.Error(w, http.StatusUnauthorized, "Tenant not found in context")
		return
	}

	authFlowUUID, err := uuid.Parse(chi.URLParam(r, "auth_flow_uuid"))
	if err != nil {
		resp.Error(w, http.StatusBadRequest, "Invalid auth flow UUID")
		return
	}

	reqParams := pagination.ParseQuery(r)
	if err := reqParams.Validate(); err != nil {
		resp.ValidationError(w, err)
		return
	}

	result, err := h.authFlowService.GetCallbackURIs(r.Context(), authFlowUUID, tenant.TenantID, reqParams.Page, reqParams.Limit)
	if err != nil {
		resp.HandleServiceError(w, r, "Failed to retrieve callback URIs", err)
		return
	}

	rows := make([]AuthFlowCallbackURIResponseDTO, len(result.Data))
	for i, c := range result.Data {
		rows[i] = toAuthFlowCallbackURIResponseDTO(c)
	}

	response := PaginatedResponseDTO[AuthFlowCallbackURIResponseDTO]{
		Rows:       rows,
		Total:      result.Total,
		Page:       result.Page,
		Limit:      result.Limit,
		TotalPages: result.TotalPages,
	}

	resp.Success(w, response, "Callback URIs retrieved successfully")
}

// RemoveCallbackURI detaches a callback URI from an auth flow.
//
// DELETE /auth_flows/{auth_flow_uuid}/callback_uris/{client_uri_uuid}
func (h *AuthFlowHandler) RemoveCallbackURI(w http.ResponseWriter, r *http.Request) {
	tenant := middleware.AuthFromRequest(r).Tenant
	if tenant == nil {
		resp.Error(w, http.StatusUnauthorized, "Tenant not found in context")
		return
	}

	authFlowUUID, err := uuid.Parse(chi.URLParam(r, "auth_flow_uuid"))
	if err != nil {
		resp.Error(w, http.StatusBadRequest, "Invalid auth flow UUID")
		return
	}

	clientURIUUID, err := uuid.Parse(chi.URLParam(r, "client_uri_uuid"))
	if err != nil {
		resp.Error(w, http.StatusBadRequest, "Invalid client URI UUID")
		return
	}

	if err := h.authFlowService.RemoveCallbackURI(r.Context(), authFlowUUID, tenant.TenantID, clientURIUUID); err != nil {
		resp.HandleServiceError(w, r, "Failed to remove callback URI", err)
		return
	}

	resp.Success(w, nil, "Callback URI removed successfully")
}

func boolValue(p *bool, def bool) bool {
	if p == nil { return def }
	return *p
}

func strValue(p *string, def string) string {
	if p == nil { return def }
	return *p
}
