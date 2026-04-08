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
	"github.com/maintainerd/auth/internal/service"
	"github.com/maintainerd/auth/internal/ptr"
	resp "github.com/maintainerd/auth/internal/rest/response"
)

// SignupFlowHandler handles signup flow management operations.
//
// This handler manages tenant-scoped signup flows (user registration processes).
// Signup flows define the registration experience for users, including which roles
// are automatically assigned upon signup. All operations are tenant-isolated -
// middleware validates tenant access and stores it in the request context.
type SignupFlowHandler struct {
	signupFlowService service.SignupFlowService
}

// NewSignupFlowHandler creates a new signup flow handler instance.
func NewSignupFlowHandler(signupFlowService service.SignupFlowService) *SignupFlowHandler {
	return &SignupFlowHandler{
		signupFlowService: signupFlowService,
	}
}

// GetAll retrieves all signup flows for the tenant with pagination and filters.
//
// GET /signup-flows
//
// Returns a paginated list of signup flows belonging to the authenticated tenant.
// Supports filtering by name, identifier, status, and auth client UUID.
func (h *SignupFlowHandler) GetAll(w http.ResponseWriter, r *http.Request) {
	// Get tenant from context (middleware already validated access)
	tenant, ok := r.Context().Value(middleware.TenantContextKey).(*model.Tenant)
	if !ok || tenant == nil {
		resp.Error(w, http.StatusUnauthorized, "Tenant not found in context")
		return
	}

	// Parse query parameters
	q := r.URL.Query()

	// Parse pagination parameters
	page, _ := strconv.Atoi(q.Get("page"))
	limit, _ := strconv.Atoi(q.Get("limit"))

	// Parse status filter
	var status []string
	if v := q.Get("status"); v != "" {
		status = append(status, v)
	}

	// Build filter DTO for validation
	filter := dto.SignupFlowFilterDTO{
		Name:       ptr.PtrOrNil(q.Get("name")),
		Identifier: ptr.PtrOrNil(q.Get("identifier")),
		Status:     status,
		ClientUUID: ptr.PtrOrNil(q.Get("client_id")),
		PaginationRequestDTO: dto.PaginationRequestDTO{
			Page:      page,
			Limit:     limit,
			SortBy:    q.Get("sort_by"),
			SortOrder: q.Get("sort_order"),
		},
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
	result, err := h.signupFlowService.GetAll(tenant.TenantID, filter.Name, filter.Identifier, filter.Status, ClientUUIDPtr, filter.Page, filter.Limit, filter.SortBy, filter.SortOrder)
	if err != nil {
		resp.HandleServiceError(w, r, "Failed to get signup flows", err)
		return
	}

	// Build paginated response
	response := dto.PaginatedResponseDTO[dto.SignupFlowResponseDTO]{
		Rows:       toSignupFlowResponseDtoList(result.Data),
		Total:      result.Total,
		Page:       result.Page,
		Limit:      result.Limit,
		TotalPages: result.TotalPages,
	}

	resp.Success(w, response, "Signup flows retrieved successfully")
}

// Get retrieves a specific signup flow by UUID.
//
// GET /signup-flows/{signup_flow_uuid}
//
// Returns detailed information about a single signup flow. The service layer
// validates that the signup flow belongs to the tenant.
func (h *SignupFlowHandler) Get(w http.ResponseWriter, r *http.Request) {
	// Get tenant from context (middleware already validated access)
	tenant, ok := r.Context().Value(middleware.TenantContextKey).(*model.Tenant)
	if !ok || tenant == nil {
		resp.Error(w, http.StatusUnauthorized, "Tenant not found in context")
		return
	}

	// Parse and validate signup flow UUID from URL parameter
	signupFlowUUIDStr := chi.URLParam(r, "signup_flow_uuid")
	signupFlowUUID, err := uuid.Parse(signupFlowUUIDStr)
	if err != nil {
		resp.Error(w, http.StatusBadRequest, "Invalid signup flow UUID")
		return
	}

	// Fetch signup flow (service validates tenant ownership)
	signupFlow, err := h.signupFlowService.GetByUUID(signupFlowUUID, tenant.TenantID)
	if err != nil {
		resp.HandleServiceError(w, r, "Signup flow not found", err)
		return
	}

	resp.Success(w, toSignupFlowResponseDTO(*signupFlow), "Signup flow retrieved successfully")
}

// Create creates a new signup flow for the tenant.
//
// POST /signup-flows
//
// Creates a new signup flow defining the user registration process. The flow
// includes configuration for the signup experience and is linked to an auth client.
func (h *SignupFlowHandler) Create(w http.ResponseWriter, r *http.Request) {
	// Get tenant from context (middleware already validated access)
	tenant, ok := r.Context().Value(middleware.TenantContextKey).(*model.Tenant)
	if !ok || tenant == nil {
		resp.Error(w, http.StatusUnauthorized, "Tenant not found in context")
		return
	}

	// Decode and validate request body
	var req dto.SignupFlowCreateRequestDTO
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		resp.Error(w, http.StatusBadRequest, "Invalid request")
		return
	}

	if err := req.Validate(); err != nil {
		resp.ValidationError(w, err)
		return
	}

	// Parse auth client UUID (already validated as UUID by DTO)
	ClientUUID, _ := uuid.Parse(req.ClientUUID)

	// Set default status if not provided
	status := model.StatusActive
	if req.Status != nil {
		status = *req.Status
	}

	// Create signup flow
	signupFlow, err := h.signupFlowService.Create(
		tenant.TenantID,
		req.Name,
		req.Description,
		req.Config,
		status,
		ClientUUID,
	)
	if err != nil {
		resp.HandleServiceError(w, r, "Failed to create signup flow", err)
		return
	}

	resp.Created(w, toSignupFlowResponseDTO(*signupFlow), "Signup flow created successfully")
}

// Update updates an existing signup flow.
//
// PUT /signup-flows/{signup_flow_uuid}
//
// Updates the configuration and settings of an existing signup flow.
// The service layer validates that the signup flow belongs to the tenant.
func (h *SignupFlowHandler) Update(w http.ResponseWriter, r *http.Request) {
	// Get tenant from context (middleware already validated access)
	tenant, ok := r.Context().Value(middleware.TenantContextKey).(*model.Tenant)
	if !ok || tenant == nil {
		resp.Error(w, http.StatusUnauthorized, "Tenant not found in context")
		return
	}

	// Parse and validate signup flow UUID from URL parameter
	signupFlowUUIDStr := chi.URLParam(r, "signup_flow_uuid")
	signupFlowUUID, err := uuid.Parse(signupFlowUUIDStr)
	if err != nil {
		resp.Error(w, http.StatusBadRequest, "Invalid signup flow UUID")
		return
	}

	// Decode and validate request body
	var req dto.SignupFlowUpdateRequestDTO
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
	if req.Status != nil {
		status = *req.Status
	}

	// Update signup flow (service validates tenant ownership)
	signupFlow, err := h.signupFlowService.Update(
		signupFlowUUID,
		tenant.TenantID,
		req.Name,
		req.Description,
		req.Config,
		status,
	)
	if err != nil {
		resp.HandleServiceError(w, r, "Failed to update signup flow", err)
		return
	}

	resp.Success(w, toSignupFlowResponseDTO(*signupFlow), "Signup flow updated successfully")
}

// Delete deletes a signup flow.
//
// DELETE /signup-flows/{signup_flow_uuid}
//
// Permanently deletes a signup flow from the tenant. This will also remove
// any associated role assignments for the flow.
func (h *SignupFlowHandler) Delete(w http.ResponseWriter, r *http.Request) {
	// Get tenant from context (middleware already validated access)
	tenant, ok := r.Context().Value(middleware.TenantContextKey).(*model.Tenant)
	if !ok || tenant == nil {
		resp.Error(w, http.StatusUnauthorized, "Tenant not found in context")
		return
	}

	// Parse and validate signup flow UUID from URL parameter
	signupFlowUUIDStr := chi.URLParam(r, "signup_flow_uuid")
	signupFlowUUID, err := uuid.Parse(signupFlowUUIDStr)
	if err != nil {
		resp.Error(w, http.StatusBadRequest, "Invalid signup flow UUID")
		return
	}

	// Delete signup flow (service validates tenant ownership)
	signupFlow, err := h.signupFlowService.Delete(signupFlowUUID, tenant.TenantID)
	if err != nil {
		resp.HandleServiceError(w, r, "Failed to delete signup flow", err)
		return
	}

	resp.Success(w, toSignupFlowResponseDTO(*signupFlow), "Signup flow deleted successfully")
}

// UpdateStatus updates the status of a signup flow.
//
// PATCH /signup-flows/{signup_flow_uuid}/status
//
// Updates only the status field of a signup flow (e.g., active, inactive).
// This is a convenience endpoint for status-only updates.
func (h *SignupFlowHandler) UpdateStatus(w http.ResponseWriter, r *http.Request) {
	// Get tenant from context (middleware already validated access)
	tenant, ok := r.Context().Value(middleware.TenantContextKey).(*model.Tenant)
	if !ok || tenant == nil {
		resp.Error(w, http.StatusUnauthorized, "Tenant not found in context")
		return
	}

	// Parse and validate signup flow UUID from URL parameter
	signupFlowUUIDStr := chi.URLParam(r, "signup_flow_uuid")
	signupFlowUUID, err := uuid.Parse(signupFlowUUIDStr)
	if err != nil {
		resp.Error(w, http.StatusBadRequest, "Invalid signup flow UUID")
		return
	}

	// Decode and validate request body
	var req dto.SignupFlowUpdateStatusRequestDTO
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		resp.Error(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if err := req.Validate(); err != nil {
		resp.ValidationError(w, err)
		return
	}

	// Update status (service validates tenant ownership)
	signupFlow, err := h.signupFlowService.UpdateStatus(signupFlowUUID, tenant.TenantID, req.Status)
	if err != nil {
		resp.HandleServiceError(w, r, "Failed to update signup flow status", err)
		return
	}

	resp.Success(w, toSignupFlowResponseDTO(*signupFlow), "Signup flow status updated successfully")
}

// AssignRoles assigns roles to a signup flow.
//
// POST /signup-flows/{signup_flow_uuid}/roles
//
// Associates one or more roles with a signup flow. Users who complete registration
// through this flow will automatically be assigned these roles.
func (h *SignupFlowHandler) AssignRoles(w http.ResponseWriter, r *http.Request) {
	// Get tenant from context (middleware already validated access)
	tenant, ok := r.Context().Value(middleware.TenantContextKey).(*model.Tenant)
	if !ok || tenant == nil {
		resp.Error(w, http.StatusUnauthorized, "Tenant not found in context")
		return
	}

	// Parse and validate signup flow UUID from URL parameter
	signupFlowUUIDStr := chi.URLParam(r, "signup_flow_uuid")
	if signupFlowUUIDStr == "" {
		resp.Error(w, http.StatusBadRequest, "Invalid signup flow UUID", "UUID parameter is required")
		return
	}

	signupFlowUUID, err := uuid.Parse(signupFlowUUIDStr)
	if err != nil {
		resp.Error(w, http.StatusBadRequest, "Invalid UUID format")
		return
	}

	// Decode and validate request body
	var req dto.SignupFlowAssignRolesRequestDTO
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		resp.Error(w, http.StatusBadRequest, "Invalid request body")
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
	roles, err := h.signupFlowService.AssignRoles(signupFlowUUID, tenant.TenantID, roleUUIDs)
	if err != nil {
		resp.HandleServiceError(w, r, "Failed to assign roles", err)
		return
	}

	// Map service results to DTOs
	response := make([]dto.RoleResponseDTO, len(roles))
	for i, role := range roles {
		response[i] = dto.RoleResponseDTO{
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
// GET /signup-flows/{signup_flow_uuid}/roles
//
// Returns a paginated list of roles that are automatically assigned to users
// who complete registration through this signup flow.
func (h *SignupFlowHandler) GetRoles(w http.ResponseWriter, r *http.Request) {
	// Get tenant from context (middleware already validated access)
	tenant, ok := r.Context().Value(middleware.TenantContextKey).(*model.Tenant)
	if !ok || tenant == nil {
		resp.Error(w, http.StatusUnauthorized, "Tenant not found in context")
		return
	}

	// Parse and validate signup flow UUID from URL parameter
	signupFlowUUIDStr := chi.URLParam(r, "signup_flow_uuid")
	if signupFlowUUIDStr == "" {
		resp.Error(w, http.StatusBadRequest, "Invalid signup flow UUID", "UUID parameter is required")
		return
	}

	signupFlowUUID, err := uuid.Parse(signupFlowUUIDStr)
	if err != nil {
		resp.Error(w, http.StatusBadRequest, "Invalid UUID format")
		return
	}

	// Parse query parameters
	q := r.URL.Query()

	// Parse pagination parameters
	page, _ := strconv.Atoi(q.Get("page"))
	limit, _ := strconv.Atoi(q.Get("limit"))

	// Build pagination DTO for validation
	reqParams := dto.PaginationRequestDTO{
		Page:      page,
		Limit:     limit,
		SortBy:    q.Get("sort_by"),
		SortOrder: q.Get("sort_order"),
	}

	if err := reqParams.Validate(); err != nil {
		resp.ValidationError(w, err)
		return
	}

	// Fetch roles for the signup flow (service validates tenant ownership)
	result, err := h.signupFlowService.GetRoles(signupFlowUUID, tenant.TenantID, reqParams.Page, reqParams.Limit)
	if err != nil {
		resp.HandleServiceError(w, r, "Failed to retrieve roles", err)
		return
	}

	// Map service results to DTOs
	rows := make([]dto.RoleResponseDTO, len(result.Data))
	for i, role := range result.Data {
		rows[i] = dto.RoleResponseDTO{
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
	response := dto.PaginatedResponseDTO[dto.RoleResponseDTO]{
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
// DELETE /signup-flows/{signup_flow_uuid}/roles/{role_uuid}
//
// Removes the association between a role and a signup flow. Users who register
// through this flow will no longer automatically receive this role.
func (h *SignupFlowHandler) RemoveRole(w http.ResponseWriter, r *http.Request) {
	// Get tenant from context (middleware already validated access)
	tenant, ok := r.Context().Value(middleware.TenantContextKey).(*model.Tenant)
	if !ok || tenant == nil {
		resp.Error(w, http.StatusUnauthorized, "Tenant not found in context")
		return
	}

	// Parse and validate UUIDs from URL parameters
	signupFlowUUIDStr := chi.URLParam(r, "signup_flow_uuid")
	roleUUIDStr := chi.URLParam(r, "role_uuid")

	if signupFlowUUIDStr == "" || roleUUIDStr == "" {
		resp.Error(w, http.StatusBadRequest, "Invalid parameters", "Both signup flow UUID and role UUID are required")
		return
	}

	signupFlowUUID, err := uuid.Parse(signupFlowUUIDStr)
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
	if err := h.signupFlowService.RemoveRole(signupFlowUUID, tenant.TenantID, roleUUID); err != nil {
		resp.HandleServiceError(w, r, "Failed to remove role", err)
		return
	}

	resp.Success(w, nil, "Role removed successfully")
}

// Helper functions for converting service data to response DTOs

// toSignupFlowResponseDTO converts a service result to a signup flow response DTO.
func toSignupFlowResponseDTO(sf service.SignupFlowServiceDataResult) dto.SignupFlowResponseDTO {
	return dto.SignupFlowResponseDTO{
		SignupFlowUUID: sf.SignupFlowUUID.String(),
		Name:           sf.Name,
		Description:    sf.Description,
		Identifier:     sf.Identifier,
		Config:         sf.Config,
		Status:         sf.Status,
		ClientUUID:     sf.ClientUUID.String(),
		CreatedAt:      sf.CreatedAt,
		UpdatedAt:      sf.UpdatedAt,
	}
}

// toSignupFlowResponseDtoList converts a list of service results to signup flow response DTOs.
func toSignupFlowResponseDtoList(sfList []service.SignupFlowServiceDataResult) []dto.SignupFlowResponseDTO {
	result := make([]dto.SignupFlowResponseDTO, len(sfList))
	for i, sf := range sfList {
		result[i] = toSignupFlowResponseDTO(sf)
	}
	return result
}
