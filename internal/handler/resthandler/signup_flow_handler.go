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
		util.Error(w, http.StatusUnauthorized, "Tenant not found in context")
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
	filter := dto.SignupFlowFilterDto{
		Name:           util.PtrOrNil(q.Get("name")),
		Identifier:     util.PtrOrNil(q.Get("identifier")),
		Status:         status,
		AuthClientUUID: util.PtrOrNil(q.Get("client_id")),
		PaginationRequestDto: dto.PaginationRequestDto{
			Page:      page,
			Limit:     limit,
			SortBy:    q.Get("sort_by"),
			SortOrder: q.Get("sort_order"),
		},
	}

	// Validate filter parameters
	if err := filter.Validate(); err != nil {
		util.ValidationError(w, err)
		return
	}

	// Parse and validate auth client UUID if provided
	var authClientUUIDPtr *uuid.UUID
	if filter.AuthClientUUID != nil {
		authClientUUID, err := uuid.Parse(*filter.AuthClientUUID)
		if err != nil {
			util.Error(w, http.StatusBadRequest, "Invalid auth client UUID")
			return
		}
		authClientUUIDPtr = &authClientUUID
	}

	// Fetch signup flows from service layer
	result, err := h.signupFlowService.GetAll(tenant.TenantID, filter.Name, filter.Identifier, filter.Status, authClientUUIDPtr, filter.Page, filter.Limit, filter.SortBy, filter.SortOrder)
	if err != nil {
		util.Error(w, http.StatusInternalServerError, "Failed to get signup flows", err.Error())
		return
	}

	// Build paginated response
	response := dto.PaginatedResponseDto[dto.SignupFlowResponseDto]{
		Rows:       toSignupFlowResponseDtoList(result.Data),
		Total:      result.Total,
		Page:       result.Page,
		Limit:      result.Limit,
		TotalPages: result.TotalPages,
	}

	util.Success(w, response, "Signup flows retrieved successfully")
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
		util.Error(w, http.StatusUnauthorized, "Tenant not found in context")
		return
	}

	// Parse and validate signup flow UUID from URL parameter
	signupFlowUUIDStr := chi.URLParam(r, "signup_flow_uuid")
	signupFlowUUID, err := uuid.Parse(signupFlowUUIDStr)
	if err != nil {
		util.Error(w, http.StatusBadRequest, "Invalid signup flow UUID")
		return
	}

	// Fetch signup flow (service validates tenant ownership)
	signupFlow, err := h.signupFlowService.GetByUUID(signupFlowUUID, tenant.TenantID)
	if err != nil {
		util.Error(w, http.StatusNotFound, "Signup flow not found")
		return
	}

	util.Success(w, toSignupFlowResponseDto(*signupFlow), "Signup flow retrieved successfully")
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
		util.Error(w, http.StatusUnauthorized, "Tenant not found in context")
		return
	}

	// Decode and validate request body
	var req dto.SignupFlowCreateRequestDto
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		util.Error(w, http.StatusBadRequest, "Invalid request", err.Error())
		return
	}

	if err := req.Validate(); err != nil {
		util.ValidationError(w, err)
		return
	}

	// Parse and validate auth client UUID
	authClientUUID, err := uuid.Parse(req.AuthClientUUID)
	if err != nil {
		util.Error(w, http.StatusBadRequest, "Invalid auth client UUID")
		return
	}

	// Set default status if not provided
	status := "active"
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
		authClientUUID,
	)
	if err != nil {
		util.Error(w, http.StatusBadRequest, "Failed to create signup flow", err.Error())
		return
	}

	util.Created(w, toSignupFlowResponseDto(*signupFlow), "Signup flow created successfully")
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
		util.Error(w, http.StatusUnauthorized, "Tenant not found in context")
		return
	}

	// Parse and validate signup flow UUID from URL parameter
	signupFlowUUIDStr := chi.URLParam(r, "signup_flow_uuid")
	signupFlowUUID, err := uuid.Parse(signupFlowUUIDStr)
	if err != nil {
		util.Error(w, http.StatusBadRequest, "Invalid signup flow UUID")
		return
	}

	// Decode and validate request body
	var req dto.SignupFlowUpdateRequestDto
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		util.Error(w, http.StatusBadRequest, "Invalid request", err.Error())
		return
	}

	if err := req.Validate(); err != nil {
		util.ValidationError(w, err)
		return
	}

	// Set default status if not provided
	status := "active"
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
		util.Error(w, http.StatusBadRequest, "Failed to update signup flow", err.Error())
		return
	}

	util.Success(w, toSignupFlowResponseDto(*signupFlow), "Signup flow updated successfully")
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
		util.Error(w, http.StatusUnauthorized, "Tenant not found in context")
		return
	}

	// Parse and validate signup flow UUID from URL parameter
	signupFlowUUIDStr := chi.URLParam(r, "signup_flow_uuid")
	signupFlowUUID, err := uuid.Parse(signupFlowUUIDStr)
	if err != nil {
		util.Error(w, http.StatusBadRequest, "Invalid signup flow UUID")
		return
	}

	// Delete signup flow (service validates tenant ownership)
	signupFlow, err := h.signupFlowService.Delete(signupFlowUUID, tenant.TenantID)
	if err != nil {
		util.Error(w, http.StatusBadRequest, "Failed to delete signup flow", err.Error())
		return
	}

	util.Success(w, toSignupFlowResponseDto(*signupFlow), "Signup flow deleted successfully")
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
		util.Error(w, http.StatusUnauthorized, "Tenant not found in context")
		return
	}

	// Parse and validate signup flow UUID from URL parameter
	signupFlowUUIDStr := chi.URLParam(r, "signup_flow_uuid")
	signupFlowUUID, err := uuid.Parse(signupFlowUUIDStr)
	if err != nil {
		util.Error(w, http.StatusBadRequest, "Invalid signup flow UUID")
		return
	}

	// Decode and validate request body
	var req dto.SignupFlowUpdateStatusRequestDto
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		util.Error(w, http.StatusBadRequest, "Invalid request body", err.Error())
		return
	}

	if err := req.Validate(); err != nil {
		util.ValidationError(w, err)
		return
	}

	// Update status (service validates tenant ownership)
	signupFlow, err := h.signupFlowService.UpdateStatus(signupFlowUUID, tenant.TenantID, req.Status)
	if err != nil {
		util.Error(w, http.StatusBadRequest, "Failed to update signup flow status", err.Error())
		return
	}

	util.Success(w, toSignupFlowResponseDto(*signupFlow), "Signup flow status updated successfully")
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
		util.Error(w, http.StatusUnauthorized, "Tenant not found in context")
		return
	}

	// Parse and validate signup flow UUID from URL parameter
	signupFlowUUIDStr := chi.URLParam(r, "signup_flow_uuid")
	if signupFlowUUIDStr == "" {
		util.Error(w, http.StatusBadRequest, "Invalid signup flow UUID", "UUID parameter is required")
		return
	}

	signupFlowUUID, err := uuid.Parse(signupFlowUUIDStr)
	if err != nil {
		util.Error(w, http.StatusBadRequest, "Invalid UUID format", err.Error())
		return
	}

	// Decode and validate request body
	var req dto.SignupFlowAssignRolesRequestDto
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		util.Error(w, http.StatusBadRequest, "Invalid request body", err.Error())
		return
	}

	if err := req.Validate(); err != nil {
		util.ValidationError(w, err)
		return
	}

	// Parse and validate role UUIDs from request
	roleUUIDs := make([]uuid.UUID, len(req.RoleUUIDs))
	for i, roleUUIDStr := range req.RoleUUIDs {
		roleUUID, err := uuid.Parse(roleUUIDStr)
		if err != nil {
			util.Error(w, http.StatusBadRequest, "Invalid role UUID format", err.Error())
			return
		}
		roleUUIDs[i] = roleUUID
	}

	// Assign roles to signup flow (service validates tenant ownership)
	roles, err := h.signupFlowService.AssignRoles(signupFlowUUID, tenant.TenantID, roleUUIDs)
	if err != nil {
		util.Error(w, http.StatusBadRequest, "Failed to assign roles", err.Error())
		return
	}

	// Map service results to DTOs
	response := make([]dto.RoleResponseDto, len(roles))
	for i, role := range roles {
		response[i] = dto.RoleResponseDto{
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

	util.Success(w, response, "Roles assigned successfully")
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
		util.Error(w, http.StatusUnauthorized, "Tenant not found in context")
		return
	}

	// Parse and validate signup flow UUID from URL parameter
	signupFlowUUIDStr := chi.URLParam(r, "signup_flow_uuid")
	if signupFlowUUIDStr == "" {
		util.Error(w, http.StatusBadRequest, "Invalid signup flow UUID", "UUID parameter is required")
		return
	}

	signupFlowUUID, err := uuid.Parse(signupFlowUUIDStr)
	if err != nil {
		util.Error(w, http.StatusBadRequest, "Invalid UUID format", err.Error())
		return
	}

	// Parse query parameters
	q := r.URL.Query()

	// Parse pagination parameters
	page, _ := strconv.Atoi(q.Get("page"))
	limit, _ := strconv.Atoi(q.Get("limit"))

	// Build pagination DTO for validation
	reqParams := dto.PaginationRequestDto{
		Page:      page,
		Limit:     limit,
		SortBy:    q.Get("sort_by"),
		SortOrder: q.Get("sort_order"),
	}

	if err := reqParams.Validate(); err != nil {
		util.ValidationError(w, err)
		return
	}

	// Fetch roles for the signup flow (service validates tenant ownership)
	result, err := h.signupFlowService.GetRoles(signupFlowUUID, tenant.TenantID, reqParams.Page, reqParams.Limit)
	if err != nil {
		util.Error(w, http.StatusBadRequest, "Failed to retrieve roles", err.Error())
		return
	}

	// Map service results to DTOs
	rows := make([]dto.RoleResponseDto, len(result.Data))
	for i, role := range result.Data {
		rows[i] = dto.RoleResponseDto{
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
	response := dto.PaginatedResponseDto[dto.RoleResponseDto]{
		Rows:       rows,
		Total:      result.Total,
		Page:       result.Page,
		Limit:      result.Limit,
		TotalPages: result.TotalPages,
	}

	util.Success(w, response, "Roles retrieved successfully")
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
		util.Error(w, http.StatusUnauthorized, "Tenant not found in context")
		return
	}

	// Parse and validate UUIDs from URL parameters
	signupFlowUUIDStr := chi.URLParam(r, "signup_flow_uuid")
	roleUUIDStr := chi.URLParam(r, "role_uuid")

	if signupFlowUUIDStr == "" || roleUUIDStr == "" {
		util.Error(w, http.StatusBadRequest, "Invalid parameters", "Both signup flow UUID and role UUID are required")
		return
	}

	signupFlowUUID, err := uuid.Parse(signupFlowUUIDStr)
	if err != nil {
		util.Error(w, http.StatusBadRequest, "Invalid signup flow UUID format", err.Error())
		return
	}

	roleUUID, err := uuid.Parse(roleUUIDStr)
	if err != nil {
		util.Error(w, http.StatusBadRequest, "Invalid role UUID format", err.Error())
		return
	}

	// Remove role from signup flow (service validates tenant ownership)
	if err := h.signupFlowService.RemoveRole(signupFlowUUID, tenant.TenantID, roleUUID); err != nil {
		util.Error(w, http.StatusBadRequest, "Failed to remove role", err.Error())
		return
	}

	util.Success(w, nil, "Role removed successfully")
}

// Helper functions for converting service data to response DTOs

// toSignupFlowResponseDto converts a service result to a signup flow response DTO.
func toSignupFlowResponseDto(sf service.SignupFlowServiceDataResult) dto.SignupFlowResponseDto {
	return dto.SignupFlowResponseDto{
		SignupFlowUUID: sf.SignupFlowUUID.String(),
		Name:           sf.Name,
		Description:    sf.Description,
		Identifier:     sf.Identifier,
		Config:         sf.Config,
		Status:         sf.Status,
		AuthClientUUID: sf.AuthClientUUID.String(),
		CreatedAt:      sf.CreatedAt,
		UpdatedAt:      sf.UpdatedAt,
	}
}

// toSignupFlowResponseDtoList converts a list of service results to signup flow response DTOs.
func toSignupFlowResponseDtoList(sfList []service.SignupFlowServiceDataResult) []dto.SignupFlowResponseDto {
	result := make([]dto.SignupFlowResponseDto, len(sfList))
	for i, sf := range sfList {
		result[i] = toSignupFlowResponseDto(sf)
	}
	return result
}

// toSignupFlowRoleResponseDtoList converts a list of signup flow role results to DTOs.
func toSignupFlowRoleResponseDtoList(roles []service.SignupFlowRoleServiceDataResult) []dto.SignupFlowRoleResponseDto {
	result := make([]dto.SignupFlowRoleResponseDto, len(roles))
	for i, role := range roles {
		result[i] = dto.SignupFlowRoleResponseDto{
			SignupFlowRoleUUID: role.SignupFlowRoleUUID.String(),
			SignupFlowUUID:     role.SignupFlowUUID.String(),
			RoleUUID:           role.RoleUUID.String(),
			RoleName:           role.RoleName,
			CreatedAt:          role.CreatedAt,
		}
	}
	return result
}
