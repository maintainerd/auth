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

type SignupFlowHandler struct {
	signupFlowService service.SignupFlowService
}

func NewSignupFlowHandler(signupFlowService service.SignupFlowService) *SignupFlowHandler {
	return &SignupFlowHandler{
		signupFlowService: signupFlowService,
	}
}

func (h *SignupFlowHandler) GetAll(w http.ResponseWriter, r *http.Request) {
	// Get tenant from context
	tenant, ok := r.Context().Value(middleware.TenantContextKey).(*model.Tenant)
	if !ok || tenant == nil {
		util.Error(w, http.StatusUnauthorized, "Tenant not found in context")
		return
	}

	q := r.URL.Query()

	// Parse pagination
	page, _ := strconv.Atoi(q.Get("page"))
	limit, _ := strconv.Atoi(q.Get("limit"))

	// Parse status filter (can be multiple)
	var status []string
	if v := q.Get("status"); v != "" {
		status = append(status, v)
	}

	// Build filter DTO
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

	// Validate filter
	if err := filter.Validate(); err != nil {
		util.ValidationError(w, err)
		return
	}

	// Parse auth_client_uuid if provided
	var authClientUUIDPtr *uuid.UUID
	if filter.AuthClientUUID != nil {
		authClientUUID, err := uuid.Parse(*filter.AuthClientUUID)
		if err != nil {
			util.Error(w, http.StatusBadRequest, "Invalid auth client UUID")
			return
		}
		authClientUUIDPtr = &authClientUUID
	}

	result, err := h.signupFlowService.GetAll(tenant.TenantID, filter.Name, filter.Identifier, filter.Status, authClientUUIDPtr, filter.Page, filter.Limit, filter.SortBy, filter.SortOrder)
	if err != nil {
		util.Error(w, http.StatusInternalServerError, "Failed to get signup flows", err.Error())
		return
	}

	response := dto.PaginatedResponseDto[dto.SignupFlowResponseDto]{
		Rows:       toSignupFlowResponseDtoList(result.Data),
		Total:      result.Total,
		Page:       result.Page,
		Limit:      result.Limit,
		TotalPages: result.TotalPages,
	}

	util.Success(w, response, "Signup flows retrieved successfully")
}

func (h *SignupFlowHandler) Get(w http.ResponseWriter, r *http.Request) {
	// Get tenant from context
	tenant, ok := r.Context().Value(middleware.TenantContextKey).(*model.Tenant)
	if !ok || tenant == nil {
		util.Error(w, http.StatusUnauthorized, "Tenant not found in context")
		return
	}

	// Get user from context
	user, ok := r.Context().Value(middleware.UserContextKey).(*model.User)
	if !ok || user == nil {
		util.Error(w, http.StatusUnauthorized, "User not found in context")
		return
	}

	// Validate user belongs to tenant
	if user.TenantID != tenant.TenantID {
		util.Error(w, http.StatusForbidden, "User does not belong to this tenant")
		return
	}

	signupFlowUUIDStr := chi.URLParam(r, "signup_flow_uuid")
	signupFlowUUID, err := uuid.Parse(signupFlowUUIDStr)
	if err != nil {
		util.Error(w, http.StatusBadRequest, "Invalid signup flow UUID")
		return
	}

	signupFlow, err := h.signupFlowService.GetByUUID(signupFlowUUID, tenant.TenantID)
	if err != nil {
		util.Error(w, http.StatusNotFound, "Signup flow not found")
		return
	}

	util.Success(w, toSignupFlowResponseDto(*signupFlow), "Signup flow retrieved successfully")
}

func (h *SignupFlowHandler) Create(w http.ResponseWriter, r *http.Request) {
	// Get tenant from context
	tenant, ok := r.Context().Value(middleware.TenantContextKey).(*model.Tenant)
	if !ok || tenant == nil {
		util.Error(w, http.StatusUnauthorized, "Tenant not found in context")
		return
	}

	// Get user from context
	user, ok := r.Context().Value(middleware.UserContextKey).(*model.User)
	if !ok || user == nil {
		util.Error(w, http.StatusUnauthorized, "User not found in context")
		return
	}

	// Validate user belongs to tenant
	if user.TenantID != tenant.TenantID {
		util.Error(w, http.StatusForbidden, "User does not belong to this tenant")
		return
	}

	var req dto.SignupFlowCreateRequestDto
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		util.Error(w, http.StatusBadRequest, "Invalid request", err.Error())
		return
	}

	if err := req.Validate(); err != nil {
		util.ValidationError(w, err)
		return
	}

	authClientUUID, err := uuid.Parse(req.AuthClientUUID)
	if err != nil {
		util.Error(w, http.StatusBadRequest, "Invalid auth client UUID")
		return
	}

	status := "active"
	if req.Status != nil {
		status = *req.Status
	}

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

func (h *SignupFlowHandler) Update(w http.ResponseWriter, r *http.Request) {
	// Get tenant from context
	tenant, ok := r.Context().Value(middleware.TenantContextKey).(*model.Tenant)
	if !ok || tenant == nil {
		util.Error(w, http.StatusUnauthorized, "Tenant not found in context")
		return
	}

	// Get user from context
	user, ok := r.Context().Value(middleware.UserContextKey).(*model.User)
	if !ok || user == nil {
		util.Error(w, http.StatusUnauthorized, "User not found in context")
		return
	}

	// Validate user belongs to tenant
	if user.TenantID != tenant.TenantID {
		util.Error(w, http.StatusForbidden, "User does not belong to this tenant")
		return
	}

	signupFlowUUIDStr := chi.URLParam(r, "signup_flow_uuid")
	signupFlowUUID, err := uuid.Parse(signupFlowUUIDStr)
	if err != nil {
		util.Error(w, http.StatusBadRequest, "Invalid signup flow UUID")
		return
	}

	var req dto.SignupFlowUpdateRequestDto
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		util.Error(w, http.StatusBadRequest, "Invalid request", err.Error())
		return
	}

	if err := req.Validate(); err != nil {
		util.ValidationError(w, err)
		return
	}

	status := "active"
	if req.Status != nil {
		status = *req.Status
	}

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

func (h *SignupFlowHandler) Delete(w http.ResponseWriter, r *http.Request) {
	// Get tenant from context
	tenant, ok := r.Context().Value(middleware.TenantContextKey).(*model.Tenant)
	if !ok || tenant == nil {
		util.Error(w, http.StatusUnauthorized, "Tenant not found in context")
		return
	}

	// Get user from context
	user, ok := r.Context().Value(middleware.UserContextKey).(*model.User)
	if !ok || user == nil {
		util.Error(w, http.StatusUnauthorized, "User not found in context")
		return
	}

	// Validate user belongs to tenant
	if user.TenantID != tenant.TenantID {
		util.Error(w, http.StatusForbidden, "User does not belong to this tenant")
		return
	}

	signupFlowUUIDStr := chi.URLParam(r, "signup_flow_uuid")
	signupFlowUUID, err := uuid.Parse(signupFlowUUIDStr)
	if err != nil {
		util.Error(w, http.StatusBadRequest, "Invalid signup flow UUID")
		return
	}

	signupFlow, err := h.signupFlowService.Delete(signupFlowUUID, tenant.TenantID)
	if err != nil {
		util.Error(w, http.StatusBadRequest, "Failed to delete signup flow", err.Error())
		return
	}

	util.Success(w, toSignupFlowResponseDto(*signupFlow), "Signup flow deleted successfully")
}

func (h *SignupFlowHandler) UpdateStatus(w http.ResponseWriter, r *http.Request) {
	// Get tenant from context
	tenant, ok := r.Context().Value(middleware.TenantContextKey).(*model.Tenant)
	if !ok || tenant == nil {
		util.Error(w, http.StatusUnauthorized, "Tenant not found in context")
		return
	}

	// Get user from context
	user, ok := r.Context().Value(middleware.UserContextKey).(*model.User)
	if !ok || user == nil {
		util.Error(w, http.StatusUnauthorized, "User not found in context")
		return
	}

	// Validate user belongs to tenant
	if user.TenantID != tenant.TenantID {
		util.Error(w, http.StatusForbidden, "User does not belong to this tenant")
		return
	}

	signupFlowUUIDStr := chi.URLParam(r, "signup_flow_uuid")
	signupFlowUUID, err := uuid.Parse(signupFlowUUIDStr)
	if err != nil {
		util.Error(w, http.StatusBadRequest, "Invalid signup flow UUID")
		return
	}

	var req dto.SignupFlowUpdateStatusRequestDto
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		util.Error(w, http.StatusBadRequest, "Invalid request body", err.Error())
		return
	}

	if err := req.Validate(); err != nil {
		util.ValidationError(w, err)
		return
	}

	signupFlow, err := h.signupFlowService.UpdateStatus(signupFlowUUID, tenant.TenantID, req.Status)
	if err != nil {
		util.Error(w, http.StatusBadRequest, "Failed to update signup flow status", err.Error())
		return
	}

	util.Success(w, toSignupFlowResponseDto(*signupFlow), "Signup flow status updated successfully")
}

// AssignRoles assigns roles to a signup flow
func (h *SignupFlowHandler) AssignRoles(w http.ResponseWriter, r *http.Request) {
	// Get tenant from context
	tenant, ok := r.Context().Value(middleware.TenantContextKey).(*model.Tenant)
	if !ok || tenant == nil {
		util.Error(w, http.StatusUnauthorized, "Tenant not found in context")
		return
	}

	// Get user from context
	user, ok := r.Context().Value(middleware.UserContextKey).(*model.User)
	if !ok || user == nil {
		util.Error(w, http.StatusUnauthorized, "User not found in context")
		return
	}

	// Validate user belongs to tenant
	if user.TenantID != tenant.TenantID {
		util.Error(w, http.StatusForbidden, "User does not belong to this tenant")
		return
	}

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

	var req dto.SignupFlowAssignRolesRequestDto
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		util.Error(w, http.StatusBadRequest, "Invalid request body", err.Error())
		return
	}

	if err := req.Validate(); err != nil {
		util.ValidationError(w, err)
		return
	}

	roleUUIDs := make([]uuid.UUID, len(req.RoleUUIDs))
	for i, roleUUIDStr := range req.RoleUUIDs {
		roleUUID, err := uuid.Parse(roleUUIDStr)
		if err != nil {
			util.Error(w, http.StatusBadRequest, "Invalid role UUID format", err.Error())
			return
		}
		roleUUIDs[i] = roleUUID
	}

	roles, err := h.signupFlowService.AssignRoles(signupFlowUUID, tenant.TenantID, roleUUIDs)
	if err != nil {
		util.Error(w, http.StatusBadRequest, "Failed to assign roles", err.Error())
		return
	}

	// Map to RoleResponseDto
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

// GetRoles retrieves all roles assigned to a signup flow
func (h *SignupFlowHandler) GetRoles(w http.ResponseWriter, r *http.Request) {
	// Get tenant from context
	tenant, ok := r.Context().Value(middleware.TenantContextKey).(*model.Tenant)
	if !ok || tenant == nil {
		util.Error(w, http.StatusUnauthorized, "Tenant not found in context")
		return
	}

	// Get user from context
	user, ok := r.Context().Value(middleware.UserContextKey).(*model.User)
	if !ok || user == nil {
		util.Error(w, http.StatusUnauthorized, "User not found in context")
		return
	}

	// Validate user belongs to tenant
	if user.TenantID != tenant.TenantID {
		util.Error(w, http.StatusForbidden, "User does not belong to this tenant")
		return
	}

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

	q := r.URL.Query()

	// Parse pagination
	page, _ := strconv.Atoi(q.Get("page"))
	limit, _ := strconv.Atoi(q.Get("limit"))

	// Build request DTO
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

	result, err := h.signupFlowService.GetRoles(signupFlowUUID, tenant.TenantID, reqParams.Page, reqParams.Limit)
	if err != nil {
		util.Error(w, http.StatusBadRequest, "Failed to retrieve roles", err.Error())
		return
	}

	// Map to RoleResponseDto
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

	response := dto.PaginatedResponseDto[dto.RoleResponseDto]{
		Rows:       rows,
		Total:      result.Total,
		Page:       result.Page,
		Limit:      result.Limit,
		TotalPages: result.TotalPages,
	}

	util.Success(w, response, "Roles retrieved successfully")
}

// RemoveRole removes a role from a signup flow
func (h *SignupFlowHandler) RemoveRole(w http.ResponseWriter, r *http.Request) {
	// Get tenant from context
	tenant, ok := r.Context().Value(middleware.TenantContextKey).(*model.Tenant)
	if !ok || tenant == nil {
		util.Error(w, http.StatusUnauthorized, "Tenant not found in context")
		return
	}

	// Get user from context
	user, ok := r.Context().Value(middleware.UserContextKey).(*model.User)
	if !ok || user == nil {
		util.Error(w, http.StatusUnauthorized, "User not found in context")
		return
	}

	// Validate user belongs to tenant
	if user.TenantID != tenant.TenantID {
		util.Error(w, http.StatusForbidden, "User does not belong to this tenant")
		return
	}

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

	if err := h.signupFlowService.RemoveRole(signupFlowUUID, tenant.TenantID, roleUUID); err != nil {
		util.Error(w, http.StatusBadRequest, "Failed to remove role", err.Error())
		return
	}

	util.Success(w, nil, "Role removed successfully")
}

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

func toSignupFlowResponseDtoList(sfList []service.SignupFlowServiceDataResult) []dto.SignupFlowResponseDto {
	result := make([]dto.SignupFlowResponseDto, len(sfList))
	for i, sf := range sfList {
		result[i] = toSignupFlowResponseDto(sf)
	}
	return result
}

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
