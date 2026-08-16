package idp

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/maintainerd/maintainerd-auth/internal/auditlog"
	"github.com/maintainerd/maintainerd-auth/internal/platform/middleware"
	"github.com/maintainerd/maintainerd-auth/internal/platform/pagination"
	"github.com/maintainerd/maintainerd-auth/internal/platform/ptr"
	resp "github.com/maintainerd/maintainerd-auth/internal/platform/response"
	"github.com/maintainerd/maintainerd-auth/internal/shared"
)

// RegistrationFlowHandler handles registration flow management operations.
//
// This handler manages tenant-scoped registration flows (user registration processes).
// Registration flows define the registration experience for users, including which roles
// are automatically assigned upon signup. All operations are tenant-isolated -
// middleware validates tenant access and stores it in the request context.
type RegistrationFlowHandler struct {
	registrationFlowService RegistrationFlowService
	auditLogger             auditlog.ManagementAuditLogger
}

// NewRegistrationFlowHandler creates a new registration flow handler instance.
func NewRegistrationFlowHandler(registrationFlowService RegistrationFlowService) *RegistrationFlowHandler {
	return &RegistrationFlowHandler{
		registrationFlowService: registrationFlowService,
	}
}

func (h *RegistrationFlowHandler) SetAuditLogger(l auditlog.ManagementAuditLogger) { h.auditLogger = l }

func (h *RegistrationFlowHandler) logAudit(r *http.Request, tenantID int64, actorUserID *int64, action, resourceType, resourceID string, resourceUUID *uuid.UUID, changes, outcome string) {
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

// Get retrieves all registration flows for the tenant with pagination and filters.
//
// GET /registration_flows
//
// Returns a paginated list of registration flows belonging to the authenticated tenant.
// Supports free-text search plus filtering by name, status, client and is_system.
func (h *RegistrationFlowHandler) Get(w http.ResponseWriter, r *http.Request) {
	// Get tenant from context (middleware already validated access)
	tenant := middleware.AuthFromRequest(r).Tenant
	if tenant == nil {
		resp.Error(w, http.StatusUnauthorized, "Tenant not found in context")
		return
	}

	q := r.URL.Query()

	// Parse status filter (comma-separated multi-select)
	var status []string
	if v := q.Get("status"); v != "" {
		status = strings.Split(strings.ReplaceAll(v, " ", ""), ",")
	}

	var isSystem *bool
	if v := q.Get("is_system"); v != "" {
		parsed := v == "true" || v == "1"
		isSystem = &parsed
	}

	// Build filter DTO for validation
	filter := RegistrationFlowFilterDTO{
		Name:                 ptr.PtrOrNil(q.Get("name")),
		Search:               ptr.PtrOrNil(q.Get("search")),
		Status:               status,
		ClientUUID:           ptr.PtrOrNil(q.Get("client_id")),
		IsSystem:             isSystem,
		PaginationRequestDTO: pagination.ParseQuery(r),
	}

	// Validate filter parameters
	if err := filter.Validate(); err != nil {
		resp.ValidationError(w, err)
		return
	}

	// Parse auth client UUID if provided (already validated as UUID by DTO)
	var clientUUIDPtr *uuid.UUID
	if filter.ClientUUID != nil {
		clientUUID, _ := uuid.Parse(*filter.ClientUUID)
		clientUUIDPtr = &clientUUID
	}

	result, err := h.registrationFlowService.Get(r.Context(), RegistrationFlowServiceGetFilter{
		TenantID:   tenant.TenantID,
		Name:       filter.Name,
		Search:     filter.Search,
		Status:     filter.Status,
		ClientUUID: clientUUIDPtr,
		IsSystem:   filter.IsSystem,
		Page:       filter.Page,
		Limit:      filter.Limit,
		SortBy:     filter.SortBy,
		SortOrder:  filter.SortOrder,
	})
	if err != nil {
		resp.HandleServiceError(w, r, "Failed to get registration flows", err)
		return
	}

	// Build paginated response
	response := PaginatedResponseDTO[RegistrationFlowResponseDTO]{
		Rows:       toRegistrationFlowListResponseDTOList(result.Data),
		Total:      result.Total,
		Page:       result.Page,
		Limit:      result.Limit,
		TotalPages: result.TotalPages,
	}

	resp.Success(w, response, "Registration flows retrieved successfully")
}

// GetByUUID retrieves a specific registration flow by UUID.
//
// GET /registration_flows/{registration_flow_uuid}
//
// Returns detailed information about a single registration flow. The service layer
// validates that the registration flow belongs to the tenant.
func (h *RegistrationFlowHandler) GetByUUID(w http.ResponseWriter, r *http.Request) {
	tenant := middleware.AuthFromRequest(r).Tenant
	if tenant == nil {
		resp.Error(w, http.StatusUnauthorized, "Tenant not found in context")
		return
	}

	registrationFlowUUID, err := uuid.Parse(chi.URLParam(r, "registration_flow_uuid"))
	if err != nil {
		resp.Error(w, http.StatusBadRequest, "Invalid registration flow UUID")
		return
	}

	registrationFlow, err := h.registrationFlowService.GetByUUID(r.Context(), registrationFlowUUID, tenant.TenantID)
	if err != nil {
		resp.HandleServiceError(w, r, "Registration flow not found", err)
		return
	}

	resp.Success(w, toRegistrationFlowDetailResponseDTO(*registrationFlow), "Registration flow retrieved successfully")
}

// Create creates a new registration flow for the tenant.
//
// POST /registration_flows
//
// Creates a new registration flow defining the user registration process. The
// flow is linked to an auth client, and its name is the public selector used in
// the flow's registration link.
func (h *RegistrationFlowHandler) Create(w http.ResponseWriter, r *http.Request) {
	auth := middleware.AuthFromRequest(r)
	tenant := auth.Tenant
	if tenant == nil {
		resp.Error(w, http.StatusUnauthorized, "Tenant not found in context")
		return
	}
	user := auth.User
	if user == nil {
		resp.Error(w, http.StatusUnauthorized, "User not found in context")
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
	clientUUID, _ := uuid.Parse(req.ClientUUID)

	// Set default status if not provided
	status := shared.StatusActive
	if req.Status != nil {
		status = *req.Status
	}

	registrationFlow, err := h.registrationFlowService.Create(r.Context(), RegistrationFlowCreateInput{
		TenantID:             tenant.TenantID,
		ActorUserUUID:        user.UserUUID,
		Name:                 req.Name,
		Description:          req.Description,
		Status:               status,
		ClientUUID:           clientUUID,
		RoleUUIDs:            parseUUIDList(req.RoleIDs),
		VerificationRequired: req.VerificationRequired != nil && *req.VerificationRequired,
		RequiredFields:       req.RequiredFields,
	})
	if err != nil {
		resp.HandleServiceError(w, r, "Failed to create registration flow", err)
		return
	}

	dtoRes := toRegistrationFlowDetailResponseDTO(*registrationFlow)
	flowUUID := registrationFlow.RegistrationFlowUUID
	changesJSON, _ := json.Marshal(map[string]any{"after": dtoRes})
	h.logAudit(r, tenant.TenantID, &user.UserID, "registration_flow.create", "registration_flow", flowUUID.String(), &flowUUID, string(changesJSON), "success")

	resp.Created(w, dtoRes, "Registration flow created successfully")
}

// Update updates an existing registration flow.
//
// PUT /registration_flows/{registration_flow_uuid}
//
// Omitted fields are left unchanged. Renaming a flow changes its public
// registration link, so links an external app already published stop resolving.
func (h *RegistrationFlowHandler) Update(w http.ResponseWriter, r *http.Request) {
	auth := middleware.AuthFromRequest(r)
	tenant := auth.Tenant
	if tenant == nil {
		resp.Error(w, http.StatusUnauthorized, "Tenant not found in context")
		return
	}
	user := auth.User
	if user == nil {
		resp.Error(w, http.StatusUnauthorized, "User not found in context")
		return
	}

	registrationFlowUUID, err := uuid.Parse(chi.URLParam(r, "registration_flow_uuid"))
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

	registrationFlow, err := h.registrationFlowService.Update(r.Context(), RegistrationFlowUpdateInput{
		RegistrationFlowUUID: registrationFlowUUID,
		TenantID:             tenant.TenantID,
		ActorUserUUID:        user.UserUUID,
		Name:                 req.Name,
		Description:          req.Description,
		Status:               req.Status,
		RoleUUIDs:            parseUUIDList(req.RoleIDs),
		VerificationRequired: req.VerificationRequired,
		RequiredFields:       req.RequiredFields,
	})
	if err != nil {
		resp.HandleServiceError(w, r, "Failed to update registration flow", err)
		return
	}

	dtoRes := toRegistrationFlowDetailResponseDTO(*registrationFlow)
	changesJSON, _ := json.Marshal(map[string]any{"update": req, "after": dtoRes})
	h.logAudit(r, tenant.TenantID, &user.UserID, "registration_flow.update", "registration_flow", registrationFlowUUID.String(), &registrationFlowUUID, string(changesJSON), "success")

	resp.Success(w, dtoRes, "Registration flow updated successfully")
}

// Delete deletes a registration flow.
//
// DELETE /registration_flows/{registration_flow_uuid}
//
// Soft-deletes a registration flow and clears its role membership. Blocked when
// the flow is still referenced by pending invites, or when it is system-managed.
func (h *RegistrationFlowHandler) Delete(w http.ResponseWriter, r *http.Request) {
	auth := middleware.AuthFromRequest(r)
	tenant := auth.Tenant
	if tenant == nil {
		resp.Error(w, http.StatusUnauthorized, "Tenant not found in context")
		return
	}
	user := auth.User
	if user == nil {
		resp.Error(w, http.StatusUnauthorized, "User not found in context")
		return
	}

	registrationFlowUUID, err := uuid.Parse(chi.URLParam(r, "registration_flow_uuid"))
	if err != nil {
		resp.Error(w, http.StatusBadRequest, "Invalid registration flow UUID")
		return
	}

	registrationFlow, err := h.registrationFlowService.Delete(r.Context(), registrationFlowUUID, tenant.TenantID, user.UserUUID)
	if err != nil {
		resp.HandleServiceError(w, r, "Failed to delete registration flow", err)
		return
	}

	changesJSON, _ := json.Marshal(map[string]any{"before": map[string]any{"id": registrationFlowUUID.String()}})
	h.logAudit(r, tenant.TenantID, &user.UserID, "registration_flow.delete", "registration_flow", registrationFlowUUID.String(), &registrationFlowUUID, string(changesJSON), "success")

	resp.Success(w, toRegistrationFlowDetailResponseDTO(*registrationFlow), "Registration flow deleted successfully")
}

// SetStatus updates the status of a registration flow.
//
// PATCH /registration_flows/{registration_flow_uuid}/status
//
// Status is the kill switch for a published registration link, so this is a
// dedicated endpoint rather than a general update.
func (h *RegistrationFlowHandler) SetStatus(w http.ResponseWriter, r *http.Request) {
	auth := middleware.AuthFromRequest(r)
	tenant := auth.Tenant
	if tenant == nil {
		resp.Error(w, http.StatusUnauthorized, "Tenant not found in context")
		return
	}
	user := auth.User
	if user == nil {
		resp.Error(w, http.StatusUnauthorized, "User not found in context")
		return
	}

	registrationFlowUUID, err := uuid.Parse(chi.URLParam(r, "registration_flow_uuid"))
	if err != nil {
		resp.Error(w, http.StatusBadRequest, "Invalid registration flow UUID")
		return
	}

	var req RegistrationFlowUpdateStatusRequestDTO
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		resp.BadRequestBody(w)
		return
	}

	if err := req.Validate(); err != nil {
		resp.ValidationError(w, err)
		return
	}

	registrationFlow, err := h.registrationFlowService.SetStatus(r.Context(), registrationFlowUUID, tenant.TenantID, user.UserUUID, req.Status)
	if err != nil {
		resp.HandleServiceError(w, r, "Failed to update registration flow status", err)
		return
	}

	changesJSON, _ := json.Marshal(map[string]any{"update": map[string]any{"status": req.Status}})
	h.logAudit(r, tenant.TenantID, &user.UserID, "registration_flow.set_status", "registration_flow", registrationFlowUUID.String(), &registrationFlowUUID, string(changesJSON), "success")

	resp.Success(w, toRegistrationFlowDetailResponseDTO(*registrationFlow), "Registration flow status updated successfully")
}

// AssignRoles assigns roles to a registration flow.
//
// POST /registration_flows/{registration_flow_uuid}/roles
//
// Users who register through this flow are automatically granted these roles, so
// the service caps the set: no system roles, and only roles the actor possesses.
func (h *RegistrationFlowHandler) AssignRoles(w http.ResponseWriter, r *http.Request) {
	auth := middleware.AuthFromRequest(r)
	tenant := auth.Tenant
	if tenant == nil {
		resp.Error(w, http.StatusUnauthorized, "Tenant not found in context")
		return
	}
	user := auth.User
	if user == nil {
		resp.Error(w, http.StatusUnauthorized, "User not found in context")
		return
	}

	registrationFlowUUID, err := uuid.Parse(chi.URLParam(r, "registration_flow_uuid"))
	if err != nil {
		resp.Error(w, http.StatusBadRequest, "Invalid registration flow UUID")
		return
	}

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

	roles, err := h.registrationFlowService.AssignRoles(r.Context(), registrationFlowUUID, tenant.TenantID, user.UserUUID, roleUUIDs)
	if err != nil {
		resp.HandleServiceError(w, r, "Failed to assign roles", err)
		return
	}

	changesJSON, _ := json.Marshal(map[string]any{"update": map[string]any{"role_uuids": req.RoleUUIDs}})
	h.logAudit(r, tenant.TenantID, &user.UserID, "registration_flow.assign_roles", "registration_flow", registrationFlowUUID.String(), &registrationFlowUUID, string(changesJSON), "success")

	resp.Success(w, toRegistrationFlowRoleResponseDTOList(roles), "Roles assigned successfully")
}

// GetRoles retrieves all roles assigned to a registration flow.
//
// GET /registration_flows/{registration_flow_uuid}/roles
func (h *RegistrationFlowHandler) GetRoles(w http.ResponseWriter, r *http.Request) {
	tenant := middleware.AuthFromRequest(r).Tenant
	if tenant == nil {
		resp.Error(w, http.StatusUnauthorized, "Tenant not found in context")
		return
	}

	registrationFlowUUID, err := uuid.Parse(chi.URLParam(r, "registration_flow_uuid"))
	if err != nil {
		resp.Error(w, http.StatusBadRequest, "Invalid registration flow UUID")
		return
	}

	reqParams := pagination.ParseQuery(r)
	if err := reqParams.Validate(); err != nil {
		resp.ValidationError(w, err)
		return
	}

	result, err := h.registrationFlowService.GetRoles(r.Context(), registrationFlowUUID, tenant.TenantID, reqParams.Page, reqParams.Limit)
	if err != nil {
		resp.HandleServiceError(w, r, "Failed to retrieve roles", err)
		return
	}

	response := PaginatedResponseDTO[RoleResponseDTO]{
		Rows:       toRegistrationFlowRoleResponseDTOList(result.Data),
		Total:      result.Total,
		Page:       result.Page,
		Limit:      result.Limit,
		TotalPages: result.TotalPages,
	}

	resp.Success(w, response, "Roles retrieved successfully")
}

// RemoveRole removes a role from a registration flow.
//
// DELETE /registration_flows/{registration_flow_uuid}/roles/{role_uuid}
func (h *RegistrationFlowHandler) RemoveRole(w http.ResponseWriter, r *http.Request) {
	auth := middleware.AuthFromRequest(r)
	tenant := auth.Tenant
	if tenant == nil {
		resp.Error(w, http.StatusUnauthorized, "Tenant not found in context")
		return
	}
	user := auth.User
	if user == nil {
		resp.Error(w, http.StatusUnauthorized, "User not found in context")
		return
	}

	registrationFlowUUID, err := uuid.Parse(chi.URLParam(r, "registration_flow_uuid"))
	if err != nil {
		resp.Error(w, http.StatusBadRequest, "Invalid registration flow UUID")
		return
	}

	roleUUID, err := uuid.Parse(chi.URLParam(r, "role_uuid"))
	if err != nil {
		resp.Error(w, http.StatusBadRequest, "Invalid role UUID")
		return
	}

	registrationFlow, err := h.registrationFlowService.RemoveRole(r.Context(), registrationFlowUUID, tenant.TenantID, user.UserUUID, roleUUID)
	if err != nil {
		resp.HandleServiceError(w, r, "Failed to remove role", err)
		return
	}

	changesJSON, _ := json.Marshal(map[string]any{"before": map[string]any{"role_id": roleUUID.String()}})
	h.logAudit(r, tenant.TenantID, &user.UserID, "registration_flow.remove_role", "registration_flow", registrationFlowUUID.String(), &registrationFlowUUID, string(changesJSON), "success")

	resp.Success(w, toRegistrationFlowDetailResponseDTO(*registrationFlow), "Role removed successfully")
}

// Helper functions for converting service data to response DTOs

// toRegistrationFlowListResponseDTO builds the lean list projection.
// required_fields and the resolved client are detail-only.
func toRegistrationFlowListResponseDTO(sf RegistrationFlowServiceDataResult) RegistrationFlowResponseDTO {
	return RegistrationFlowResponseDTO{
		RegistrationFlowUUID: sf.RegistrationFlowUUID.String(),
		Name:                 sf.Name,
		Description:          sf.Description,
		Status:               sf.Status,
		ClientUUID:           clientUUIDString(sf.ClientUUID),
		VerificationRequired: sf.VerificationRequired,
		IsSystem:             sf.IsSystem,
		CreatedAt:            sf.CreatedAt,
		UpdatedAt:            sf.UpdatedAt,
	}
}

func toRegistrationFlowListResponseDTOList(sfList []RegistrationFlowServiceDataResult) []RegistrationFlowResponseDTO {
	result := make([]RegistrationFlowResponseDTO, len(sfList))
	for i, sf := range sfList {
		result[i] = toRegistrationFlowListResponseDTO(sf)
	}
	return result
}

// toRegistrationFlowDetailResponseDTO builds the full detail projection.
func toRegistrationFlowDetailResponseDTO(sf RegistrationFlowServiceDataResult) RegistrationFlowDetailResponseDTO {
	dto := RegistrationFlowDetailResponseDTO{
		RegistrationFlowUUID: sf.RegistrationFlowUUID.String(),
		Name:                 sf.Name,
		Description:          sf.Description,
		Status:               sf.Status,
		ClientUUID:           clientUUIDString(sf.ClientUUID),
		VerificationRequired: sf.VerificationRequired,
		RequiredFields:       sf.RequiredFields,
		IsSystem:             sf.IsSystem,
		CreatedAt:            sf.CreatedAt,
		UpdatedAt:            sf.UpdatedAt,
	}
	if sf.ClientUUID != nil {
		dto.Client = &RegistrationFlowClientSummaryDTO{
			ClientUUID:  sf.ClientUUID.String(),
			Name:        sf.ClientName,
			DisplayName: sf.ClientDisplayName,
			Identifier:  sf.ClientIdentifier,
			Status:      sf.ClientStatus,
		}
	}
	return dto
}

func toRegistrationFlowRoleResponseDTOList(roles []RegistrationFlowRoleServiceDataResult) []RoleResponseDTO {
	out := make([]RoleResponseDTO, len(roles))
	for i, role := range roles {
		out[i] = RoleResponseDTO{
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
	return out
}

func clientUUIDString(id *uuid.UUID) *string {
	if id == nil {
		return nil
	}
	s := id.String()
	return &s
}

// parseUUIDList parses a slice of UUID strings, preserving the nil-vs-empty
// distinction (nil input → nil output, so "field omitted" stays distinguishable
// from "field set to empty" for replace-semantics on update). Entries are
// already validated as UUIDs by the request DTO.
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
