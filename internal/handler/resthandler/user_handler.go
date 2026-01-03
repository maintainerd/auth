package resthandler

import (
	"encoding/json"
	"net/http"
	"sort"
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

// UserHandler handles user management operations.
//
// This handler manages tenant-scoped user accounts. In the multi-tenant architecture,
// users are associated with tenants through the user_identities table. All operations
// are tenant-isolated - middleware validates tenant access and stores it in the request
// context. The handler supports CRUD operations, role management, identity management,
// and account verification workflows.
type UserHandler struct {
	userService service.UserService
}

// NewUserHandler creates a new user handler instance.
func NewUserHandler(userService service.UserService) *UserHandler {
	return &UserHandler{
		userService: userService,
	}
}

// GetUsers retrieves all users for the tenant with pagination and filters.
//
// GET /users
//
// Returns a paginated list of users belonging to the authenticated tenant.
// Supports filtering by username, email, phone, status, and role UUID.
func (h *UserHandler) GetUsers(w http.ResponseWriter, r *http.Request) {
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

	// Parse role UUID filter
	var roleUUID *string
	if v := q.Get("role_id"); v != "" {
		roleUUID = &v
	}

	// Build filter DTO for validation
	reqParams := dto.UserFilterDto{
		Username: util.PtrOrNil(q.Get("username")),
		Email:    util.PtrOrNil(q.Get("email")),
		Phone:    util.PtrOrNil(q.Get("phone")),
		Status:   status,
		RoleUUID: roleUUID,
		PaginationRequestDto: dto.PaginationRequestDto{
			Page:      page,
			Limit:     limit,
			SortBy:    q.Get("sort_by"),
			SortOrder: q.Get("sort_order"),
		},
	}

	// Validate filter parameters
	if err := reqParams.Validate(); err != nil {
		util.ValidationError(w, err)
		return
	}

	// Build service filter with tenant context
	filter := service.UserServiceGetFilter{
		Username:  reqParams.Username,
		Email:     reqParams.Email,
		Phone:     reqParams.Phone,
		Status:    reqParams.Status,
		TenantID:  tenant.TenantID,
		RoleUUID:  reqParams.RoleUUID,
		Page:      reqParams.PaginationRequestDto.Page,
		Limit:     reqParams.PaginationRequestDto.Limit,
		SortBy:    reqParams.PaginationRequestDto.SortBy,
		SortOrder: reqParams.PaginationRequestDto.SortOrder,
	}

	// Fetch users from service layer
	result, err := h.userService.Get(filter)
	if err != nil {
		util.Error(w, http.StatusInternalServerError, "Failed to fetch users", err.Error())
		return
	}

	// Map service results to DTOs
	rows := make([]dto.UserResponseDto, len(result.Data))
	for i, r := range result.Data {
		rows[i] = toUserResponseDto(r)
	}

	// Build paginated response
	response := dto.PaginatedResponseDto[dto.UserResponseDto]{
		Rows:       rows,
		Total:      result.Total,
		Page:       result.Page,
		Limit:      result.Limit,
		TotalPages: result.TotalPages,
	}

	util.Success(w, response, "Users fetched successfully")
}

// GetUser retrieves a specific user by UUID.
//
// GET /users/{user_uuid}
//
// Returns detailed information about a single user. The service layer
// validates that the user belongs to the tenant.
func (h *UserHandler) GetUser(w http.ResponseWriter, r *http.Request) {
	// Get tenant from context (middleware already validated access)
	tenant, ok := r.Context().Value(middleware.TenantContextKey).(*model.Tenant)
	if !ok || tenant == nil {
		util.Error(w, http.StatusUnauthorized, "Tenant not found in context")
		return
	}

	// Parse and validate user UUID from URL parameter
	userUUIDStr := chi.URLParam(r, "user_uuid")
	userUUID, err := uuid.Parse(userUUIDStr)
	if err != nil {
		util.Error(w, http.StatusBadRequest, "Invalid user UUID")
		return
	}

	// Fetch user (service validates tenant ownership)
	user, err := h.userService.GetByUUID(userUUID, tenant.TenantID)
	if err != nil {
		util.Error(w, http.StatusNotFound, "User not found")
		return
	}

	// Map to response DTO
	dtoRes := toUserResponseDto(*user)

	util.Success(w, dtoRes, "User fetched successfully")
}

// CreateUser creates a new user for the tenant.
//
// POST /users
//
// Creates a new user account within the authenticated tenant. The creator's
// context is used for audit tracking.
func (h *UserHandler) CreateUser(w http.ResponseWriter, r *http.Request) {
	// Get tenant from context (middleware already validated access)
	tenant, ok := r.Context().Value(middleware.TenantContextKey).(*model.Tenant)
	if !ok || tenant == nil {
		util.Error(w, http.StatusUnauthorized, "Tenant not found in context")
		return
	}

	// Get creator user from context (needed for audit trail)
	creatorUser := r.Context().Value(middleware.UserContextKey).(*model.User)

	// Decode and validate request body
	var req dto.UserCreateRequestDto
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		util.Error(w, http.StatusBadRequest, "Invalid JSON format")
		return
	}

	if err := req.Validate(); err != nil {
		util.Error(w, http.StatusBadRequest, "Validation failed", err.Error())
		return
	}

	// Create user (includes creator context for audit trail)
	user, err := h.userService.Create(req.Username, req.Fullname, req.Email, req.Phone, req.Password, req.Status, req.Metadata, tenant.TenantUUID.String(), creatorUser.UserUUID)
	if err != nil {
		util.Error(w, http.StatusInternalServerError, "Failed to create user", err.Error())
		return
	}

	// Map to response DTO
	dtoRes := toUserResponseDto(*user)

	util.Created(w, dtoRes, "User created successfully")
}

// UpdateUser updates an existing user.
//
// PUT /users/{user_uuid}
//
// Updates user account information. The service layer validates that the user
// belongs to the tenant. The updater's context is used for audit tracking.
func (h *UserHandler) UpdateUser(w http.ResponseWriter, r *http.Request) {
	// Get tenant from context (middleware already validated access)
	tenant, ok := r.Context().Value(middleware.TenantContextKey).(*model.Tenant)
	if !ok || tenant == nil {
		util.Error(w, http.StatusUnauthorized, "Tenant not found in context")
		return
	}

	// Get updater user from context (needed for audit trail)
	updaterUser := r.Context().Value(middleware.UserContextKey).(*model.User)

	// Parse and validate user UUID from URL parameter
	userUUIDStr := chi.URLParam(r, "user_uuid")
	userUUID, err := uuid.Parse(userUUIDStr)
	if err != nil {
		util.Error(w, http.StatusBadRequest, "Invalid user UUID")
		return
	}

	// Decode and validate request body
	var req dto.UserUpdateRequestDto
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		util.Error(w, http.StatusBadRequest, "Invalid JSON format")
		return
	}

	if err := req.Validate(); err != nil {
		util.Error(w, http.StatusBadRequest, "Validation failed", err.Error())
		return
	}

	// Update user (service validates tenant ownership, includes updater context for audit)
	user, err := h.userService.Update(userUUID, tenant.TenantID, req.Username, req.Fullname, req.Email, req.Phone, req.Status, req.Metadata, updaterUser.UserUUID)
	if err != nil {
		util.Error(w, http.StatusInternalServerError, "Failed to update user", err.Error())
		return
	}

	// Map to response DTO
	dtoRes := toUserResponseDto(*user)

	util.Success(w, dtoRes, "User updated successfully")
}

// SetUserStatus updates the status of a user.
//
// PATCH /users/{user_uuid}/status
//
// Updates only the status field of a user (e.g., active, inactive, suspended).
// This is a convenience endpoint for status-only updates.
func (h *UserHandler) SetUserStatus(w http.ResponseWriter, r *http.Request) {
	// Get tenant from context (middleware already validated access)
	tenant, ok := r.Context().Value(middleware.TenantContextKey).(*model.Tenant)
	if !ok || tenant == nil {
		util.Error(w, http.StatusUnauthorized, "Tenant not found in context")
		return
	}

	// Get updater user from context (needed for audit trail)
	updaterUser := r.Context().Value(middleware.UserContextKey).(*model.User)

	// Parse and validate user UUID from URL parameter
	userUUIDStr := chi.URLParam(r, "user_uuid")
	userUUID, err := uuid.Parse(userUUIDStr)
	if err != nil {
		util.Error(w, http.StatusBadRequest, "Invalid user UUID")
		return
	}

	// Decode and validate request body
	var req dto.UserSetStatusRequestDto
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		util.Error(w, http.StatusBadRequest, "Invalid JSON format")
		return
	}

	if err := req.Validate(); err != nil {
		util.Error(w, http.StatusBadRequest, "Validation failed", err.Error())
		return
	}

	// Update user status (service validates tenant ownership, includes updater context for audit)
	user, err := h.userService.SetStatus(userUUID, tenant.TenantID, req.Status, updaterUser.UserUUID)
	if err != nil {
		util.Error(w, http.StatusInternalServerError, "Failed to update user status", err.Error())
		return
	}

	// Map to response DTO
	dtoRes := toUserResponseDto(*user)

	util.Success(w, dtoRes, "User status updated successfully")
}

// VerifyEmail marks a user's email as verified.
//
// POST /users/{user_uuid}/verify-email
//
// Verifies the user's email address and may mark the account as completed
// if all required verification steps are done.
func (h *UserHandler) VerifyEmail(w http.ResponseWriter, r *http.Request) {
	// Get tenant from context (middleware already validated access)
	tenant, ok := r.Context().Value(middleware.TenantContextKey).(*model.Tenant)
	if !ok || tenant == nil {
		util.Error(w, http.StatusUnauthorized, "Tenant not found in context")
		return
	}

	// Parse and validate user UUID from URL parameter
	userUUIDStr := chi.URLParam(r, "user_uuid")
	userUUID, err := uuid.Parse(userUUIDStr)
	if err != nil {
		util.Error(w, http.StatusBadRequest, "Invalid user UUID")
		return
	}

	// Verify email (service validates tenant ownership)
	user, err := h.userService.VerifyEmail(userUUID, tenant.TenantID)
	if err != nil {
		util.Error(w, http.StatusInternalServerError, "Failed to verify email", err.Error())
		return
	}

	dtoRes := toUserResponseDto(*user)
	util.Success(w, dtoRes, "Email verified and account completed successfully")
}

// VerifyPhone marks a user's phone number as verified.
//
// POST /users/{user_uuid}/verify-phone
//
// Verifies the user's phone number for two-factor authentication
// or account recovery purposes.
func (h *UserHandler) VerifyPhone(w http.ResponseWriter, r *http.Request) {
	// Get tenant from context (middleware already validated access)
	tenant, ok := r.Context().Value(middleware.TenantContextKey).(*model.Tenant)
	if !ok || tenant == nil {
		util.Error(w, http.StatusUnauthorized, "Tenant not found in context")
		return
	}

	// Parse and validate user UUID from URL parameter
	userUUIDStr := chi.URLParam(r, "user_uuid")
	userUUID, err := uuid.Parse(userUUIDStr)
	if err != nil {
		util.Error(w, http.StatusBadRequest, "Invalid user UUID")
		return
	}

	// Verify phone (service validates tenant ownership)
	user, err := h.userService.VerifyPhone(userUUID, tenant.TenantID)
	if err != nil {
		util.Error(w, http.StatusInternalServerError, "Failed to verify phone", err.Error())
		return
	}

	dtoRes := toUserResponseDto(*user)
	util.Success(w, dtoRes, "Phone verified successfully")
}

// CompleteAccount marks a user's account as completed.
//
// POST /users/{user_uuid}/complete-account
//
// Manually marks an account as completed, typically after all required
// profile information and verifications are done.
func (h *UserHandler) CompleteAccount(w http.ResponseWriter, r *http.Request) {
	// Get tenant from context (middleware already validated access)
	tenant, ok := r.Context().Value(middleware.TenantContextKey).(*model.Tenant)
	if !ok || tenant == nil {
		util.Error(w, http.StatusUnauthorized, "Tenant not found in context")
		return
	}

	// Parse and validate user UUID from URL parameter
	userUUIDStr := chi.URLParam(r, "user_uuid")
	userUUID, err := uuid.Parse(userUUIDStr)
	if err != nil {
		util.Error(w, http.StatusBadRequest, "Invalid user UUID")
		return
	}

	// Mark account as completed (service validates tenant ownership)
	user, err := h.userService.CompleteAccount(userUUID, tenant.TenantID)
	if err != nil {
		util.Error(w, http.StatusInternalServerError, "Failed to complete account", err.Error())
		return
	}

	dtoRes := toUserResponseDto(*user)
	util.Success(w, dtoRes, "Account marked as completed successfully")
}

// DeleteUser deletes a user.
//
// DELETE /users/{user_uuid}
//
// Permanently deletes a user account from the tenant. The service layer
// validates tenant ownership. The deleter's context is used for audit tracking.
func (h *UserHandler) DeleteUser(w http.ResponseWriter, r *http.Request) {
	// Get tenant from context (middleware already validated access)
	tenant, ok := r.Context().Value(middleware.TenantContextKey).(*model.Tenant)
	if !ok || tenant == nil {
		util.Error(w, http.StatusUnauthorized, "Tenant not found in context")
		return
	}

	// Get deleter user from context (needed for audit trail)
	deleterUser := r.Context().Value(middleware.UserContextKey).(*model.User)

	// Parse and validate user UUID from URL parameter
	userUUIDStr := chi.URLParam(r, "user_uuid")
	userUUID, err := uuid.Parse(userUUIDStr)
	if err != nil {
		util.Error(w, http.StatusBadRequest, "Invalid user UUID")
		return
	}

	// Delete user (service validates tenant ownership, includes deleter context for audit)
	user, err := h.userService.DeleteByUUID(userUUID, tenant.TenantID, deleterUser.UserUUID)
	if err != nil {
		util.Error(w, http.StatusInternalServerError, "Failed to delete user", err.Error())
		return
	}

	// Map to response DTO
	dtoRes := toUserResponseDto(*user)

	util.Success(w, dtoRes, "User deleted successfully")
}

// AssignRoles assigns roles to a user.
//
// POST /users/{user_uuid}/roles
//
// Associates one or more roles with a user, granting them the permissions
// defined by those roles.
func (h *UserHandler) AssignRoles(w http.ResponseWriter, r *http.Request) {
	// Get tenant from context (middleware already validated access)
	tenant, ok := r.Context().Value(middleware.TenantContextKey).(*model.Tenant)
	if !ok || tenant == nil {
		util.Error(w, http.StatusUnauthorized, "Tenant not found in context")
		return
	}

	// Parse and validate user UUID from URL parameter
	userUUIDStr := chi.URLParam(r, "user_uuid")
	userUUID, err := uuid.Parse(userUUIDStr)
	if err != nil {
		util.Error(w, http.StatusBadRequest, "Invalid user UUID")
		return
	}

	// Decode and validate request body
	var req dto.UserAssignRolesRequestDto
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		util.Error(w, http.StatusBadRequest, "Invalid JSON format")
		return
	}

	if err := req.Validate(); err != nil {
		util.Error(w, http.StatusBadRequest, "Validation failed", err.Error())
		return
	}

	// Assign roles to user (service validates tenant ownership)
	user, err := h.userService.AssignUserRoles(userUUID, req.RoleUUIDs, tenant.TenantID)
	if err != nil {
		util.Error(w, http.StatusInternalServerError, "Failed to assign roles to user", err.Error())
		return
	}

	// Map to response DTO
	dtoRes := toUserResponseDto(*user)

	util.Success(w, dtoRes, "Roles assigned to user successfully")
}

// RemoveRole removes a role from a user.
//
// DELETE /users/{user_uuid}/roles/{role_uuid}
//
// Removes the association between a role and a user, revoking the permissions
// granted by that role.
func (h *UserHandler) RemoveRole(w http.ResponseWriter, r *http.Request) {
	// Get tenant from context (middleware already validated access)
	tenant, ok := r.Context().Value(middleware.TenantContextKey).(*model.Tenant)
	if !ok || tenant == nil {
		util.Error(w, http.StatusUnauthorized, "Tenant not found in context")
		return
	}

	// Parse and validate user UUID from URL parameter
	userUUIDStr := chi.URLParam(r, "user_uuid")
	userUUID, err := uuid.Parse(userUUIDStr)
	if err != nil {
		util.Error(w, http.StatusBadRequest, "Invalid user UUID")
		return
	}

	// Parse and validate role UUID from URL parameter
	roleUUIDStr := chi.URLParam(r, "role_uuid")
	roleUUID, err := uuid.Parse(roleUUIDStr)
	if err != nil {
		util.Error(w, http.StatusBadRequest, "Invalid role UUID")
		return
	}

	// Remove role from user (service validates tenant ownership)
	user, err := h.userService.RemoveUserRole(userUUID, roleUUID, tenant.TenantID)
	if err != nil {
		util.Error(w, http.StatusInternalServerError, "Failed to remove role from user", err.Error())
		return
	}

	// Map to response DTO
	dtoRes := toUserResponseDto(*user)

	util.Success(w, dtoRes, "Role removed from user successfully")
}

// Helper functions for converting service data to response DTOs

// toUserResponseDto converts a service result to a user response DTO.
func toUserResponseDto(u service.UserServiceDataResult) dto.UserResponseDto {
	result := dto.UserResponseDto{
		UserUUID:           u.UserUUID,
		Username:           u.Username,
		Fullname:           u.Fullname,
		Email:              u.Email,
		Phone:              u.Phone,
		IsEmailVerified:    u.IsEmailVerified,
		IsPhoneVerified:    u.IsPhoneVerified,
		IsProfileCompleted: u.IsProfileCompleted,
		IsAccountCompleted: u.IsAccountCompleted,
		Status:             u.Status,
		Metadata:           u.Metadata,
		CreatedAt:          u.CreatedAt,
		UpdatedAt:          u.UpdatedAt,
	}

	// Map Tenant if present
	if u.Tenant != nil {
		result.Tenant = &dto.TenantResponseDto{
			TenantUUID:  u.Tenant.TenantUUID,
			Name:        u.Tenant.Name,
			Description: u.Tenant.Description,
			Identifier:  u.Tenant.Identifier,
			Status:      u.Tenant.Status,
			IsPublic:    u.Tenant.IsPublic,
			IsDefault:   u.Tenant.IsDefault,
			IsSystem:    u.Tenant.IsSystem,
			CreatedAt:   u.Tenant.CreatedAt,
			UpdatedAt:   u.Tenant.UpdatedAt,
		}
	}

	return result
}

// GetUserRoles retrieves all roles assigned to a user with pagination and filters.
//
// GET /users/{user_uuid}/roles
//
// Returns a paginated list of roles assigned to the user. Supports filtering
// by role name, description, and status.
func (h *UserHandler) GetUserRoles(w http.ResponseWriter, r *http.Request) {
	// Parse and validate user UUID from URL parameter
	userUUIDStr := chi.URLParam(r, "user_uuid")
	userUUID, err := uuid.Parse(userUUIDStr)
	if err != nil {
		util.Error(w, http.StatusBadRequest, "Invalid user UUID")
		return
	}

	// Parse query parameters
	q := r.URL.Query()

	// Parse pagination parameters
	page, _ := strconv.Atoi(q.Get("page"))
	limit, _ := strconv.Atoi(q.Get("limit"))

	// Build filter DTO for validation
	reqParams := dto.UserRoleFilterDto{
		Name:        util.PtrOrNil(q.Get("name")),
		Description: util.PtrOrNil(q.Get("description")),
		Status:      util.PtrOrNil(q.Get("status")),
		PaginationRequestDto: dto.PaginationRequestDto{
			Page:      page,
			Limit:     limit,
			SortBy:    q.Get("sort_by"),
			SortOrder: q.Get("sort_order"),
		},
	}

	// Validate filter parameters
	if err := reqParams.Validate(); err != nil {
		util.ValidationError(w, err)
		return
	}

	// Get tenant from context (middleware already validated access)
	tenant, ok := r.Context().Value(middleware.TenantContextKey).(*model.Tenant)
	if !ok || tenant == nil {
		util.Error(w, http.StatusUnauthorized, "Tenant not found in context")
		return
	}

	// Verify user exists and belongs to tenant
	user, err := h.userService.GetByUUID(userUUID, tenant.TenantID)
	if err != nil {
		util.Error(w, http.StatusNotFound, "User not found")
		return
	}

	// Fetch roles for the user
	roles, err := h.userService.GetUserRoles(user.UserUUID)
	if err != nil {
		util.Error(w, http.StatusInternalServerError, "Failed to fetch user roles", err.Error())
		return
	}

	// Apply filters
	filteredRoles := []service.RoleServiceDataResult{}
	for _, role := range roles {
		// Filter by name
		if reqParams.Name != nil && !containsIgnoreCase(role.Name, *reqParams.Name) {
			continue
		}
		// Filter by description
		if reqParams.Description != nil && !containsIgnoreCase(role.Description, *reqParams.Description) {
			continue
		}
		// Filter by status
		if reqParams.Status != nil && role.Status != *reqParams.Status {
			continue
		}
		filteredRoles = append(filteredRoles, role)
	}

	// Apply sorting
	if reqParams.SortBy != "" {
		sortRoles(filteredRoles, reqParams.SortBy, reqParams.SortOrder)
	}

	// Apply pagination
	total := int64(len(filteredRoles))
	offset := (reqParams.Page - 1) * reqParams.Limit
	end := offset + reqParams.Limit
	if end > len(filteredRoles) {
		end = len(filteredRoles)
	}
	if offset > len(filteredRoles) {
		offset = len(filteredRoles)
	}
	paginatedRoles := filteredRoles[offset:end]

	// Map to DTOs
	rows := make([]dto.RoleResponseDto, len(paginatedRoles))
	for i, role := range paginatedRoles {
		rows[i] = dto.RoleResponseDto{
			RoleUUID:    role.RoleUUID,
			Name:        role.Name,
			Description: role.Description,
			IsDefault:   role.IsDefault,
			IsSystem:    role.IsSystem,
			CreatedAt:   role.CreatedAt,
			UpdatedAt:   role.UpdatedAt,
		}
	}

	totalPages := int((total + int64(reqParams.Limit) - 1) / int64(reqParams.Limit))
	response := dto.PaginatedResponseDto[dto.RoleResponseDto]{
		Rows:       rows,
		Total:      total,
		Page:       reqParams.Page,
		Limit:      reqParams.Limit,
		TotalPages: totalPages,
	}

	util.Success(w, response, "User roles fetched successfully")
}

// GetUserIdentities retrieves all identities for a user with pagination and filters.
//
// GET /users/{user_uuid}/identities
//
// Returns a paginated list of identity providers linked to the user (e.g., Google, GitHub).
// Supports filtering by provider type.
func (h *UserHandler) GetUserIdentities(w http.ResponseWriter, r *http.Request) {
	// Parse and validate user UUID from URL parameter
	userUUIDStr := chi.URLParam(r, "user_uuid")
	userUUID, err := uuid.Parse(userUUIDStr)
	if err != nil {
		util.Error(w, http.StatusBadRequest, "Invalid user UUID")
		return
	}

	// Parse query parameters
	q := r.URL.Query()

	// Parse pagination parameters
	page, _ := strconv.Atoi(q.Get("page"))
	limit, _ := strconv.Atoi(q.Get("limit"))

	// Build filter DTO for validation
	reqParams := dto.UserIdentityFilterDto{
		Provider: util.PtrOrNil(q.Get("provider")),
		PaginationRequestDto: dto.PaginationRequestDto{
			Page:      page,
			Limit:     limit,
			SortBy:    q.Get("sort_by"),
			SortOrder: q.Get("sort_order"),
		},
	}

	// Validate filter parameters
	if err := reqParams.Validate(); err != nil {
		util.ValidationError(w, err)
		return
	}

	// Get tenant from context (middleware already validated access)
	tenant, ok := r.Context().Value(middleware.TenantContextKey).(*model.Tenant)
	if !ok || tenant == nil {
		util.Error(w, http.StatusUnauthorized, "Tenant not found in context")
		return
	}

	// Verify user exists and belongs to tenant
	user, err := h.userService.GetByUUID(userUUID, tenant.TenantID)
	if err != nil {
		util.Error(w, http.StatusNotFound, "User not found")
		return
	}

	// Fetch identities for the user
	identities, err := h.userService.GetUserIdentities(user.UserUUID)
	if err != nil {
		util.Error(w, http.StatusInternalServerError, "Failed to fetch user identities", err.Error())
		return
	}

	// Apply filters
	filteredIdentities := []service.UserIdentityServiceDataResult{}
	for _, identity := range identities {
		// Filter by provider
		if reqParams.Provider != nil && !containsIgnoreCase(identity.Provider, *reqParams.Provider) {
			continue
		}
		filteredIdentities = append(filteredIdentities, identity)
	}

	// Apply sorting
	if reqParams.SortBy != "" {
		sortIdentities(filteredIdentities, reqParams.SortBy, reqParams.SortOrder)
	}

	// Apply pagination
	total := int64(len(filteredIdentities))
	offset := (reqParams.Page - 1) * reqParams.Limit
	end := offset + reqParams.Limit
	if end > len(filteredIdentities) {
		end = len(filteredIdentities)
	}
	if offset > len(filteredIdentities) {
		offset = len(filteredIdentities)
	}
	paginatedIdentities := filteredIdentities[offset:end]

	// Map to DTOs
	rows := make([]dto.UserIdentityResponseDto, len(paginatedIdentities))
	for i, identity := range paginatedIdentities {
		rows[i] = dto.UserIdentityResponseDto{
			UserIdentityUUID: identity.UserIdentityUUID,
			Provider:         identity.Provider,
			Sub:              identity.Sub,
			Metadata:         identity.Metadata,
			CreatedAt:        identity.CreatedAt,
			UpdatedAt:        identity.UpdatedAt,
		}
		if identity.AuthClient != nil {
			rows[i].AuthClient = &dto.AuthClientResponseDto{
				AuthClientUUID: identity.AuthClient.AuthClientUUID,
				Name:           identity.AuthClient.Name,
				DisplayName:    identity.AuthClient.DisplayName,
				ClientType:     identity.AuthClient.ClientType,
				Domain:         identity.AuthClient.Domain,
				Status:         identity.AuthClient.Status,
				IsDefault:      identity.AuthClient.IsDefault,
				IsSystem:       identity.AuthClient.IsSystem,
				CreatedAt:      identity.AuthClient.CreatedAt,
				UpdatedAt:      identity.AuthClient.UpdatedAt,
			}
		}
	}

	totalPages := int((total + int64(reqParams.Limit) - 1) / int64(reqParams.Limit))
	response := dto.PaginatedResponseDto[dto.UserIdentityResponseDto]{
		Rows:       rows,
		Total:      total,
		Page:       reqParams.Page,
		Limit:      reqParams.Limit,
		TotalPages: totalPages,
	}

	util.Success(w, response, "User identities fetched successfully")
}

// Helper function for case-insensitive contains check
func containsIgnoreCase(str, substr string) bool {
	return strings.Contains(strings.ToLower(str), strings.ToLower(substr))
}

// Helper function to sort roles
func sortRoles(roles []service.RoleServiceDataResult, sortBy, sortOrder string) {
	sort.Slice(roles, func(i, j int) bool {
		var result bool
		switch sortBy {
		case "name":
			result = roles[i].Name < roles[j].Name
		case "description":
			result = roles[i].Description < roles[j].Description
		case "status":
			result = roles[i].Status < roles[j].Status
		case "created_at":
			result = roles[i].CreatedAt.Before(roles[j].CreatedAt)
		case "updated_at":
			result = roles[i].UpdatedAt.Before(roles[j].UpdatedAt)
		default:
			result = roles[i].CreatedAt.After(roles[j].CreatedAt) // Default sort by created_at DESC
		}
		if sortOrder == "desc" {
			return !result
		}
		return result
	})
}

// Helper function to sort identities
func sortIdentities(identities []service.UserIdentityServiceDataResult, sortBy, sortOrder string) {
	sort.Slice(identities, func(i, j int) bool {
		var result bool
		switch sortBy {
		case "provider":
			result = identities[i].Provider < identities[j].Provider
		case "sub":
			result = identities[i].Sub < identities[j].Sub
		case "created_at":
			result = identities[i].CreatedAt.Before(identities[j].CreatedAt)
		case "updated_at":
			result = identities[i].UpdatedAt.Before(identities[j].UpdatedAt)
		default:
			result = identities[i].CreatedAt.After(identities[j].CreatedAt) // Default sort by created_at DESC
		}
		if sortOrder == "desc" {
			return !result
		}
		return result
	})
}
