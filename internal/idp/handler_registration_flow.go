package idp

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/maintainerd/maintainerd-auth/internal/platform/middleware"
	"github.com/maintainerd/maintainerd-auth/internal/platform/pagination"
	"github.com/maintainerd/maintainerd-auth/internal/platform/ptr"
	resp "github.com/maintainerd/maintainerd-auth/internal/platform/response"
	"github.com/maintainerd/maintainerd-auth/internal/shared"
	"gorm.io/datatypes"
)

// RegistrationFlowHandler handles registration flow management operations.
//
// This handler manages tenant-scoped registration flows (user registration processes).
// Registration flows define the registration experience for users, including which roles
// are automatically assigned upon signup. All operations are tenant-isolated -
// middleware validates tenant access and stores it in the request context.
type RegistrationFlowHandler struct {
	registrationFlowService RegistrationFlowService
}

// NewRegistrationFlowHandler creates a new registration flow handler instance.
func NewRegistrationFlowHandler(registrationFlowService RegistrationFlowService) *RegistrationFlowHandler {
	return &RegistrationFlowHandler{
		registrationFlowService: registrationFlowService,
	}
}

// GetAll retrieves all registration flows for the tenant with pagination and filters.
//
// GET /registration-flows
//
// Returns a paginated list of registration flows belonging to the authenticated tenant.
// Supports filtering by name, identifier, status, and auth client UUID.
func (h *RegistrationFlowHandler) GetAll(w http.ResponseWriter, r *http.Request) {
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
	filter := RegistrationFlowFilterDTO{
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

	// Fetch registration flows from service layer
	result, err := h.registrationFlowService.GetAll(r.Context(), tenant.TenantID, filter.Name, filter.Identifier, filter.Status, ClientUUIDPtr, filter.Page, filter.Limit, filter.SortBy, filter.SortOrder)
	if err != nil {
		resp.HandleServiceError(w, r, "Failed to get registration flows", err)
		return
	}

	// Build paginated response
	response := PaginatedResponseDTO[RegistrationFlowResponseDTO]{
		Rows:       toRegistrationFlowResponseDtoList(result.Data),
		Total:      result.Total,
		Page:       result.Page,
		Limit:      result.Limit,
		TotalPages: result.TotalPages,
	}

	resp.Success(w, response, "Registration flows retrieved successfully")
}

// Get retrieves a specific registration flow by UUID.
//
// GET /registration-flows/{registration_flow_uuid}
//
// Returns detailed information about a single registration flow. The service layer
// validates that the registration flow belongs to the tenant.
func (h *RegistrationFlowHandler) Get(w http.ResponseWriter, r *http.Request) {
	// Get tenant from context (middleware already validated access)
	tenant := middleware.AuthFromRequest(r).Tenant
	if tenant == nil {
		resp.Error(w, http.StatusUnauthorized, "Tenant not found in context")
		return
	}

	// Parse and validate registration flow UUID from URL parameter
	registrationFlowUUIDStr := chi.URLParam(r, "registration_flow_uuid")
	registrationFlowUUID, err := uuid.Parse(registrationFlowUUIDStr)
	if err != nil {
		resp.Error(w, http.StatusBadRequest, "Invalid registration flow UUID")
		return
	}

	// Fetch registration flow (service validates tenant ownership)
	registrationFlow, err := h.registrationFlowService.GetByUUID(r.Context(), registrationFlowUUID, tenant.TenantID)
	if err != nil {
		resp.HandleServiceError(w, r, "Registration flow not found", err)
		return
	}

	resp.Success(w, toRegistrationFlowResponseDTO(*registrationFlow), "Registration flow retrieved successfully")
}

// Create creates a new registration flow for the tenant.
//
// POST /registration-flows
//
// Creates a new registration flow defining the user registration process. The flow
// includes configuration for the signup experience and is linked to an auth client.
func (h *RegistrationFlowHandler) Create(w http.ResponseWriter, r *http.Request) {
	// Get tenant from context (middleware already validated access)
	tenant := middleware.AuthFromRequest(r).Tenant
	if tenant == nil {
		resp.Error(w, http.StatusUnauthorized, "Tenant not found in context")
		return
	}

	// Decode and validate request body
	var req RegistrationFlowCreateRequestDTO
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

	// Set default status if not provided
	status := shared.StatusActive
	if req.Status != nil {
		status = *req.Status
	}

	// Create registration flow
	registrationFlow, err := h.registrationFlowService.Create(
		r.Context(),
		tenant.TenantID,
		req.Name,
		req.Description,
		status,
		ClientUUID,
		req.Identifier,
		parseUUIDList(req.RoleIDs),
		boolValue(req.VerificationRequired, false),
		requiredFieldsJSON(req.RequiredFields),
	)
	if err != nil {
		resp.HandleServiceError(w, r, "Failed to create registration flow", err)
		return
	}

	resp.Created(w, toRegistrationFlowResponseDTO(*registrationFlow), "Registration flow created successfully")
}

// Update updates an existing registration flow.
//
// PUT /registration-flows/{registration_flow_uuid}
//
// Updates the configuration and settings of an existing registration flow.
// The service layer validates that the registration flow belongs to the tenant.
func (h *RegistrationFlowHandler) Update(w http.ResponseWriter, r *http.Request) {
	tenant := middleware.AuthFromRequest(r).Tenant
	if tenant == nil {
		resp.Error(w, http.StatusUnauthorized, "Tenant not found in context")
		return
	}

	registrationFlowUUIDStr := chi.URLParam(r, "registration_flow_uuid")
	registrationFlowUUID, err := uuid.Parse(registrationFlowUUIDStr)
	if err != nil {
		resp.Error(w, http.StatusBadRequest, "Invalid registration flow UUID")
		return
	}

	var req RegistrationFlowUpdateRequestDTO
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		resp.BadRequestBody(w)
		return
	}

	if err := req.Validate(); err != nil {
		resp.ValidationError(w, err)
		return
	}

	status := shared.StatusActive
	if req.Status != nil {
		status = *req.Status
	}

	registrationFlow, err := h.registrationFlowService.Update(
		r.Context(),
		registrationFlowUUID,
		tenant.TenantID,
		req.Name,
		req.Description,
		status,
		parseUUIDList(req.RoleIDs),
		boolValue(req.VerificationRequired, false),
		requiredFieldsJSON(req.RequiredFields),
	)
	if err != nil {
		resp.HandleServiceError(w, r, "Failed to update registration flow", err)
		return
	}

	resp.Success(w, toRegistrationFlowResponseDTO(*registrationFlow), "Registration flow updated successfully")
}

// Delete deletes a registration flow.
//
// DELETE /registration-flows/{registration_flow_uuid}
//
// Permanently deletes a registration flow from the tenant. This will also remove
// any associated role assignments for the flow.
func (h *RegistrationFlowHandler) Delete(w http.ResponseWriter, r *http.Request) {
	// Get tenant from context (middleware already validated access)
	tenant := middleware.AuthFromRequest(r).Tenant
	if tenant == nil {
		resp.Error(w, http.StatusUnauthorized, "Tenant not found in context")
		return
	}

	// Parse and validate registration flow UUID from URL parameter
	registrationFlowUUIDStr := chi.URLParam(r, "registration_flow_uuid")
	registrationFlowUUID, err := uuid.Parse(registrationFlowUUIDStr)
	if err != nil {
		resp.Error(w, http.StatusBadRequest, "Invalid registration flow UUID")
		return
	}

	// Delete registration flow (service validates tenant ownership)
	registrationFlow, err := h.registrationFlowService.Delete(r.Context(), registrationFlowUUID, tenant.TenantID)
	if err != nil {
		resp.HandleServiceError(w, r, "Failed to delete registration flow", err)
		return
	}

	resp.Success(w, toRegistrationFlowResponseDTO(*registrationFlow), "Registration flow deleted successfully")
}

// UpdateStatus updates the status of a registration flow.
//
// PATCH /registration-flows/{registration_flow_uuid}/status
//
// Updates only the status field of a registration flow (e.g., active, inactive).
// This is a convenience endpoint for status-only updates.
func (h *RegistrationFlowHandler) UpdateStatus(w http.ResponseWriter, r *http.Request) {
	// Get tenant from context (middleware already validated access)
	tenant := middleware.AuthFromRequest(r).Tenant
	if tenant == nil {
		resp.Error(w, http.StatusUnauthorized, "Tenant not found in context")
		return
	}

	// Parse and validate registration flow UUID from URL parameter
	registrationFlowUUIDStr := chi.URLParam(r, "registration_flow_uuid")
	registrationFlowUUID, err := uuid.Parse(registrationFlowUUIDStr)
	if err != nil {
		resp.Error(w, http.StatusBadRequest, "Invalid registration flow UUID")
		return
	}

	// Decode and validate request body
	var req RegistrationFlowUpdateStatusRequestDTO
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		resp.BadRequestBody(w)
		return
	}

	if err := req.Validate(); err != nil {
		resp.ValidationError(w, err)
		return
	}

	// Update status (service validates tenant ownership)
	registrationFlow, err := h.registrationFlowService.UpdateStatus(r.Context(), registrationFlowUUID, tenant.TenantID, req.Status)
	if err != nil {
		resp.HandleServiceError(w, r, "Failed to update registration flow status", err)
		return
	}

	resp.Success(w, toRegistrationFlowResponseDTO(*registrationFlow), "Registration flow status updated successfully")
}

// AssignRoles assigns roles to a registration flow.
//
// POST /registration-flows/{registration_flow_uuid}/roles
//
// Associates one or more roles with a registration flow. Users who complete registration
// through this flow will automatically be assigned these roles.
func (h *RegistrationFlowHandler) AssignRoles(w http.ResponseWriter, r *http.Request) {
	// Get tenant from context (middleware already validated access)
	tenant := middleware.AuthFromRequest(r).Tenant
	if tenant == nil {
		resp.Error(w, http.StatusUnauthorized, "Tenant not found in context")
		return
	}

	// Parse and validate registration flow UUID from URL parameter
	registrationFlowUUIDStr := chi.URLParam(r, "registration_flow_uuid")
	if registrationFlowUUIDStr == "" {
		resp.Error(w, http.StatusBadRequest, "Invalid registration flow UUID", "UUID parameter is required")
		return
	}

	registrationFlowUUID, err := uuid.Parse(registrationFlowUUIDStr)
	if err != nil {
		resp.Error(w, http.StatusBadRequest, "Invalid UUID format")
		return
	}

	// Decode and validate request body
	var req RegistrationFlowAssignRolesRequestDTO
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

	// Assign roles to registration flow (service validates tenant ownership)
	roles, err := h.registrationFlowService.AssignRoles(r.Context(), registrationFlowUUID, tenant.TenantID, roleUUIDs)
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

// GetRoles retrieves all roles assigned to a registration flow.
//
// GET /registration-flows/{registration_flow_uuid}/roles
//
// Returns a paginated list of roles that are automatically assigned to users
// who complete registration through this registration flow.
func (h *RegistrationFlowHandler) GetRoles(w http.ResponseWriter, r *http.Request) {
	// Get tenant from context (middleware already validated access)
	tenant := middleware.AuthFromRequest(r).Tenant
	if tenant == nil {
		resp.Error(w, http.StatusUnauthorized, "Tenant not found in context")
		return
	}

	// Parse and validate registration flow UUID from URL parameter
	registrationFlowUUIDStr := chi.URLParam(r, "registration_flow_uuid")
	if registrationFlowUUIDStr == "" {
		resp.Error(w, http.StatusBadRequest, "Invalid registration flow UUID", "UUID parameter is required")
		return
	}

	registrationFlowUUID, err := uuid.Parse(registrationFlowUUIDStr)
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

	// Fetch roles for the registration flow (service validates tenant ownership)
	result, err := h.registrationFlowService.GetRoles(r.Context(), registrationFlowUUID, tenant.TenantID, reqParams.Page, reqParams.Limit)
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

// RemoveRole removes a role from a registration flow.
//
// DELETE /registration-flows/{registration_flow_uuid}/roles/{role_uuid}
//
// Removes the association between a role and a registration flow. Users who register
// through this flow will no longer automatically receive this role.
func (h *RegistrationFlowHandler) RemoveRole(w http.ResponseWriter, r *http.Request) {
	// Get tenant from context (middleware already validated access)
	tenant := middleware.AuthFromRequest(r).Tenant
	if tenant == nil {
		resp.Error(w, http.StatusUnauthorized, "Tenant not found in context")
		return
	}

	// Parse and validate UUIDs from URL parameters
	registrationFlowUUIDStr := chi.URLParam(r, "registration_flow_uuid")
	roleUUIDStr := chi.URLParam(r, "role_uuid")

	if registrationFlowUUIDStr == "" || roleUUIDStr == "" {
		resp.Error(w, http.StatusBadRequest, "Invalid parameters", "Both registration flow UUID and role UUID are required")
		return
	}

	registrationFlowUUID, err := uuid.Parse(registrationFlowUUIDStr)
	if err != nil {
		resp.Error(w, http.StatusBadRequest, "Invalid registration flow UUID format")
		return
	}

	roleUUID, err := uuid.Parse(roleUUIDStr)
	if err != nil {
		resp.Error(w, http.StatusBadRequest, "Invalid role UUID format")
		return
	}

	// Remove role from registration flow (service validates tenant ownership)
	if err := h.registrationFlowService.RemoveRole(r.Context(), registrationFlowUUID, tenant.TenantID, roleUUID); err != nil {
		resp.HandleServiceError(w, r, "Failed to remove role", err)
		return
	}

	resp.Success(w, nil, "Role removed successfully")
}

// Helper functions for converting service data to response DTOs

// toRegistrationFlowResponseDTO converts a service result to a registration flow response DTO.
func toRegistrationFlowResponseDTO(sf RegistrationFlowServiceDataResult) RegistrationFlowResponseDTO {
	dto := RegistrationFlowResponseDTO{
		RegistrationFlowUUID: sf.RegistrationFlowUUID.String(),
		Name:                 sf.Name,
		Description:          sf.Description,
		Identifier:           sf.Identifier,
		Status:               sf.Status,
		ClientUUID:           sf.ClientUUID.String(),
		CreatedAt:            sf.CreatedAt,
		UpdatedAt:            sf.UpdatedAt,
	}
	dto.VerificationRequired = sf.VerificationRequired
	dto.RequiredFields = sf.RequiredFields
	return dto
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

// toRegistrationFlowResponseDtoList converts a list of service results to registration flow response DTOs.
func toRegistrationFlowResponseDtoList(sfList []RegistrationFlowServiceDataResult) []RegistrationFlowResponseDTO {
	result := make([]RegistrationFlowResponseDTO, len(sfList))
	for i, sf := range sfList {
		result[i] = toRegistrationFlowResponseDTO(sf)
	}
	return result
}

func boolValue(p *bool, def bool) bool {
	if p == nil {
		return def
	}
	return *p
}

func requiredFieldsJSON(arr *[]string) datatypes.JSON {
	if arr == nil {
		return datatypes.JSON([]byte("[]"))
	}
	b, _ := json.Marshal(*arr)
	return datatypes.JSON(b)
}
