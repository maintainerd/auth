package user

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/platform/middleware"
	"github.com/maintainerd/auth/internal/platform/pagination"
	"github.com/maintainerd/auth/internal/platform/ptr"
	resp "github.com/maintainerd/auth/internal/platform/response"
)

// UserHandler handles user management operations.
//
// This handler manages tenant-scoped user accounts. In the multi-tenant architecture,
// users are associated with tenants through the user_identities table. All operations
// are tenant-isolated - middleware validates tenant access and stores it in the request
// context. The handler supports CRUD operations, role management, identity management,
// and account verification workflows.
type UserHandler struct {
	userService UserService
}

// NewUserHandler creates a new user handler instance.
func NewUserHandler(userService UserService) *UserHandler {
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

	// Parse role UUID filter
	var roleUUID *string
	if v := q.Get("role_id"); v != "" {
		roleUUID = &v
	}

	// Parse user pool UUID filter
	var userPoolUUID *string
	if v := q.Get("user_pool_id"); v != "" {
		userPoolUUID = &v
	}

	// Parse client UUID filter
	var clientUUID *string
	if v := q.Get("client_id"); v != "" {
		clientUUID = &v
	}

	// Build filter DTO for validation
	reqParams := UserFilterDTO{
		Username:             ptr.PtrOrNil(q.Get("username")),
		Email:                ptr.PtrOrNil(q.Get("email")),
		Phone:                ptr.PtrOrNil(q.Get("phone")),
		Status:               status,
		RoleUUID:             roleUUID,
		UserPoolUUID:         userPoolUUID,
		ClientUUID:           clientUUID,
		PaginationRequestDTO: pagination.ParseQuery(r),
	}

	// Validate filter parameters
	if err := reqParams.Validate(); err != nil {
		resp.ValidationError(w, err)
		return
	}

	// Build service filter with tenant context
	filter := UserServiceGetFilter{
		Username:     reqParams.Username,
		Email:        reqParams.Email,
		Phone:        reqParams.Phone,
		Status:       reqParams.Status,
		TenantID:     tenant.TenantID,
		RoleUUID:     reqParams.RoleUUID,
		UserPoolUUID: reqParams.UserPoolUUID,
		ClientUUID:   reqParams.ClientUUID,
		Page:         reqParams.PaginationRequestDTO.Page,
		Limit:        reqParams.PaginationRequestDTO.Limit,
		SortBy:       reqParams.PaginationRequestDTO.SortBy,
		SortOrder:    reqParams.PaginationRequestDTO.SortOrder,
	}

	// Fetch users from service layer
	result, err := h.userService.Get(r.Context(), filter)
	if err != nil {
		resp.HandleServiceError(w, r, "Failed to fetch users", err)
		return
	}

	// Map service results to DTOs
	rows := make([]UserResponseDTO, len(result.Data))
	for i, r := range result.Data {
		rows[i] = toUserResponseDTO(r)
	}

	// Build paginated response
	response := PaginatedResponseDTO[UserResponseDTO]{
		Rows:       rows,
		Total:      result.Total,
		Page:       result.Page,
		Limit:      result.Limit,
		TotalPages: result.TotalPages,
	}

	resp.Success(w, response, "Users fetched successfully")
}

// GetUser retrieves a specific user by UUID.
//
// GET /users/{user_uuid}
//
// Returns detailed information about a single user. The service layer
// validates that the user belongs to the tenant.
func (h *UserHandler) GetUser(w http.ResponseWriter, r *http.Request) {
	// Get tenant from context (middleware already validated access)
	tenant := middleware.AuthFromRequest(r).Tenant
	if tenant == nil {
		resp.Error(w, http.StatusUnauthorized, "Tenant not found in context")
		return
	}

	// Parse and validate user UUID from URL parameter
	userUUIDStr := chi.URLParam(r, "user_uuid")
	userUUID, err := uuid.Parse(userUUIDStr)
	if err != nil {
		resp.Error(w, http.StatusBadRequest, "Invalid user UUID")
		return
	}

	// Fetch user (service validates tenant ownership)
	user, err := h.userService.GetByUUID(r.Context(), userUUID, tenant.TenantID)
	if err != nil {
		resp.HandleServiceError(w, r, "User not found", err)
		return
	}

	// Map to response DTO
	dtoRes := toUserResponseDTO(*user)

	resp.Success(w, dtoRes, "User fetched successfully")
}

// CreateUser creates a new user for the tenant.
//
// POST /users
//
// Creates a new user account within the authenticated tenant. The creator's
// context is used for audit tracking.
func (h *UserHandler) CreateUser(w http.ResponseWriter, r *http.Request) {
	// Get tenant from context (middleware already validated access)
	tenant := middleware.AuthFromRequest(r).Tenant
	if tenant == nil {
		resp.Error(w, http.StatusUnauthorized, "Tenant not found in context")
		return
	}

	// Get creator user from context (needed for audit trail)
	creatorUser := middleware.AuthFromRequest(r).User

	// Decode and validate request body
	var req UserCreateRequestDTO
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		resp.BadRequestBody(w)
		return
	}

	if err := req.Validate(); err != nil {
		resp.ValidationError(w, err)
		return
	}

	// Create user (includes creator context for audit trail)
	user, err := h.userService.Create(r.Context(), req.Username, req.Fullname, req.Email, req.Phone, req.Password, req.Status, req.Metadata, tenant.TenantUUID.String(), creatorUser.UserUUID)
	if err != nil {
		resp.HandleServiceError(w, r, "Failed to create user", err)
		return
	}

	// Map to response DTO
	dtoRes := toUserResponseDTO(*user)

	resp.Created(w, dtoRes, "User created successfully")
}

// UpdateUser updates an existing user.
//
// PUT /users/{user_uuid}
//
// Updates user account information. The service layer validates that the user
// belongs to the tenant. The updater's context is used for audit tracking.
func (h *UserHandler) UpdateUser(w http.ResponseWriter, r *http.Request) {
	// Get tenant from context (middleware already validated access)
	tenant := middleware.AuthFromRequest(r).Tenant
	if tenant == nil {
		resp.Error(w, http.StatusUnauthorized, "Tenant not found in context")
		return
	}

	// Get updater user from context (needed for audit trail)
	updaterUser := middleware.AuthFromRequest(r).User

	// Parse and validate user UUID from URL parameter
	userUUIDStr := chi.URLParam(r, "user_uuid")
	userUUID, err := uuid.Parse(userUUIDStr)
	if err != nil {
		resp.Error(w, http.StatusBadRequest, "Invalid user UUID")
		return
	}

	// Decode and validate request body
	var req UserUpdateRequestDTO
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		resp.BadRequestBody(w)
		return
	}

	if err := req.Validate(); err != nil {
		resp.ValidationError(w, err)
		return
	}

	// Update user (service validates tenant ownership, includes updater context for audit)
	user, err := h.userService.Update(r.Context(), userUUID, tenant.TenantID, req.Username, req.Fullname, req.Email, req.Phone, req.Status, req.Metadata, updaterUser.UserUUID)
	if err != nil {
		resp.HandleServiceError(w, r, "Failed to update user", err)
		return
	}

	// Map to response DTO
	dtoRes := toUserResponseDTO(*user)

	resp.Success(w, dtoRes, "User updated successfully")
}

// SetUserStatus updates the status of a user.
//
// PATCH /users/{user_uuid}/status
//
// Updates only the status field of a user (e.g., active, inactive, suspended).
// This is a convenience endpoint for status-only updates.
func (h *UserHandler) SetUserStatus(w http.ResponseWriter, r *http.Request) {
	// Get tenant from context (middleware already validated access)
	tenant := middleware.AuthFromRequest(r).Tenant
	if tenant == nil {
		resp.Error(w, http.StatusUnauthorized, "Tenant not found in context")
		return
	}

	// Get updater user from context (needed for audit trail)
	updaterUser := middleware.AuthFromRequest(r).User

	// Parse and validate user UUID from URL parameter
	userUUIDStr := chi.URLParam(r, "user_uuid")
	userUUID, err := uuid.Parse(userUUIDStr)
	if err != nil {
		resp.Error(w, http.StatusBadRequest, "Invalid user UUID")
		return
	}

	// Decode and validate request body
	var req UserSetStatusRequestDTO
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		resp.BadRequestBody(w)
		return
	}

	if err := req.Validate(); err != nil {
		resp.ValidationError(w, err)
		return
	}

	// Update user status (service validates tenant ownership, includes updater context for audit)
	user, err := h.userService.SetStatus(r.Context(), userUUID, tenant.TenantID, req.Status, updaterUser.UserUUID)
	if err != nil {
		resp.HandleServiceError(w, r, "Failed to update user status", err)
		return
	}

	// Map to response DTO
	dtoRes := toUserResponseDTO(*user)

	resp.Success(w, dtoRes, "User status updated successfully")
}

// VerifyEmail marks a user's email as verified.
//
// POST /users/{user_uuid}/verify-email
//
// Verifies the user's email address and may mark the account as completed
// if all required verification steps are done.
func (h *UserHandler) VerifyEmail(w http.ResponseWriter, r *http.Request) {
	// Get tenant from context (middleware already validated access)
	tenant := middleware.AuthFromRequest(r).Tenant
	if tenant == nil {
		resp.Error(w, http.StatusUnauthorized, "Tenant not found in context")
		return
	}

	// Parse and validate user UUID from URL parameter
	userUUIDStr := chi.URLParam(r, "user_uuid")
	userUUID, err := uuid.Parse(userUUIDStr)
	if err != nil {
		resp.Error(w, http.StatusBadRequest, "Invalid user UUID")
		return
	}

	// Verify email (service validates tenant ownership)
	user, err := h.userService.VerifyEmail(r.Context(), userUUID, tenant.TenantID)
	if err != nil {
		resp.HandleServiceError(w, r, "Failed to verify email", err)
		return
	}

	dtoRes := toUserResponseDTO(*user)
	resp.Success(w, dtoRes, "Email verified and account completed successfully")
}

// VerifyPhone marks a user's phone number as verified.
//
// POST /users/{user_uuid}/verify-phone
//
// Verifies the user's phone number for two-factor authentication
// or account recovery purposes.
func (h *UserHandler) VerifyPhone(w http.ResponseWriter, r *http.Request) {
	// Get tenant from context (middleware already validated access)
	tenant := middleware.AuthFromRequest(r).Tenant
	if tenant == nil {
		resp.Error(w, http.StatusUnauthorized, "Tenant not found in context")
		return
	}

	// Parse and validate user UUID from URL parameter
	userUUIDStr := chi.URLParam(r, "user_uuid")
	userUUID, err := uuid.Parse(userUUIDStr)
	if err != nil {
		resp.Error(w, http.StatusBadRequest, "Invalid user UUID")
		return
	}

	// Verify phone (service validates tenant ownership)
	user, err := h.userService.VerifyPhone(r.Context(), userUUID, tenant.TenantID)
	if err != nil {
		resp.HandleServiceError(w, r, "Failed to verify phone", err)
		return
	}

	dtoRes := toUserResponseDTO(*user)
	resp.Success(w, dtoRes, "Phone verified successfully")
}

// CompleteAccount marks a user's account as completed.
//
// POST /users/{user_uuid}/complete-account
//
// Manually marks an account as completed, typically after all required
// profile information and verifications are done.
func (h *UserHandler) CompleteAccount(w http.ResponseWriter, r *http.Request) {
	// Get tenant from context (middleware already validated access)
	tenant := middleware.AuthFromRequest(r).Tenant
	if tenant == nil {
		resp.Error(w, http.StatusUnauthorized, "Tenant not found in context")
		return
	}

	// Parse and validate user UUID from URL parameter
	userUUIDStr := chi.URLParam(r, "user_uuid")
	userUUID, err := uuid.Parse(userUUIDStr)
	if err != nil {
		resp.Error(w, http.StatusBadRequest, "Invalid user UUID")
		return
	}

	// Mark account as completed (service validates tenant ownership)
	user, err := h.userService.CompleteAccount(r.Context(), userUUID, tenant.TenantID)
	if err != nil {
		resp.HandleServiceError(w, r, "Failed to complete account", err)
		return
	}

	dtoRes := toUserResponseDTO(*user)
	resp.Success(w, dtoRes, "Account marked as completed successfully")
}

// DeleteUser deletes a user.
//
// DELETE /users/{user_uuid}
//
// Permanently deletes a user account from the tenant. The service layer
// validates tenant ownership. The deleter's context is used for audit tracking.
func (h *UserHandler) DeleteUser(w http.ResponseWriter, r *http.Request) {
	// Get tenant from context (middleware already validated access)
	tenant := middleware.AuthFromRequest(r).Tenant
	if tenant == nil {
		resp.Error(w, http.StatusUnauthorized, "Tenant not found in context")
		return
	}

	// Get deleter user from context (needed for audit trail)
	deleterUser := middleware.AuthFromRequest(r).User

	// Parse and validate user UUID from URL parameter
	userUUIDStr := chi.URLParam(r, "user_uuid")
	userUUID, err := uuid.Parse(userUUIDStr)
	if err != nil {
		resp.Error(w, http.StatusBadRequest, "Invalid user UUID")
		return
	}

	// Delete user (service validates tenant ownership, includes deleter context for audit)
	user, err := h.userService.DeleteByUUID(r.Context(), userUUID, tenant.TenantID, deleterUser.UserUUID)
	if err != nil {
		resp.HandleServiceError(w, r, "Failed to delete user", err)
		return
	}

	// Map to response DTO
	dtoRes := toUserResponseDTO(*user)

	resp.Success(w, dtoRes, "User deleted successfully")
}

// AssignRoles assigns roles to a user.
//
// POST /users/{user_uuid}/roles
//
// Associates one or more roles with a user, granting them the permissions
// defined by those roles.
func (h *UserHandler) AssignRoles(w http.ResponseWriter, r *http.Request) {
	// Get tenant from context (middleware already validated access)
	tenant := middleware.AuthFromRequest(r).Tenant
	if tenant == nil {
		resp.Error(w, http.StatusUnauthorized, "Tenant not found in context")
		return
	}

	// Parse and validate user UUID from URL parameter
	userUUIDStr := chi.URLParam(r, "user_uuid")
	userUUID, err := uuid.Parse(userUUIDStr)
	if err != nil {
		resp.Error(w, http.StatusBadRequest, "Invalid user UUID")
		return
	}

	// Decode and validate request body
	var req UserAssignRolesRequestDTO
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		resp.BadRequestBody(w)
		return
	}

	if err := req.Validate(); err != nil {
		resp.ValidationError(w, err)
		return
	}

	// Assign roles to user (service validates tenant ownership)
	user, err := h.userService.AssignUserRoles(r.Context(), userUUID, req.RoleUUIDs, tenant.TenantID)
	if err != nil {
		resp.HandleServiceError(w, r, "Failed to assign roles to user", err)
		return
	}

	// Map to response DTO
	dtoRes := toUserResponseDTO(*user)

	resp.Success(w, dtoRes, "Roles assigned to user successfully")
}

// RemoveRole removes a role from a user.
//
// DELETE /users/{user_uuid}/roles/{role_uuid}
//
// Removes the association between a role and a user, revoking the permissions
// granted by that role.
func (h *UserHandler) RemoveRole(w http.ResponseWriter, r *http.Request) {
	// Get tenant from context (middleware already validated access)
	tenant := middleware.AuthFromRequest(r).Tenant
	if tenant == nil {
		resp.Error(w, http.StatusUnauthorized, "Tenant not found in context")
		return
	}

	// Parse and validate user UUID from URL parameter
	userUUIDStr := chi.URLParam(r, "user_uuid")
	userUUID, err := uuid.Parse(userUUIDStr)
	if err != nil {
		resp.Error(w, http.StatusBadRequest, "Invalid user UUID")
		return
	}

	// Parse and validate role UUID from URL parameter
	roleUUIDStr := chi.URLParam(r, "role_uuid")
	roleUUID, err := uuid.Parse(roleUUIDStr)
	if err != nil {
		resp.Error(w, http.StatusBadRequest, "Invalid role UUID")
		return
	}

	// Remove role from user (service validates tenant ownership)
	user, err := h.userService.RemoveUserRole(r.Context(), userUUID, roleUUID, tenant.TenantID)
	if err != nil {
		resp.HandleServiceError(w, r, "Failed to remove role from user", err)
		return
	}

	// Map to response DTO
	dtoRes := toUserResponseDTO(*user)

	resp.Success(w, dtoRes, "Role removed from user successfully")
}

// ForcePasswordChange sets the force_password_change flag on a user.
//
// PUT /users/{user_uuid}/force-password-change
//
// Marks a user account so that they must change their password on next login.
func (h *UserHandler) ForcePasswordChange(w http.ResponseWriter, r *http.Request) {
	userUUIDStr := chi.URLParam(r, "user_uuid")
	userUUID, err := uuid.Parse(userUUIDStr)
	if err != nil {
		resp.Error(w, http.StatusBadRequest, "Invalid user UUID")
		return
	}

	if err := h.userService.ForcePasswordChange(r.Context(), userUUID, true); err != nil {
		resp.HandleServiceError(w, r, "Failed to set force password change", err)
		return
	}

	resp.Success(w, nil, "User will be required to change password on next login")
}

// Helper functions for converting service data to response DTOs

// toUserResponseDTO converts a service result to a user response DTO.
func toUserResponseDTO(u UserServiceDataResult) UserResponseDTO {
	result := UserResponseDTO{
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
		result.Tenant = &TenantResponseDTO{
			TenantUUID:  u.Tenant.TenantUUID,
			Name:        u.Tenant.Name,
			Description: u.Tenant.Description,
			Identifier:  u.Tenant.Identifier,
			Status:      u.Tenant.Status,
			IsPublic:    u.Tenant.IsPublic,

			IsSystem:  u.Tenant.IsSystem,
			CreatedAt: u.Tenant.CreatedAt,
			UpdatedAt: u.Tenant.UpdatedAt,
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
		resp.Error(w, http.StatusBadRequest, "Invalid user UUID")
		return
	}

	// Parse query parameters
	q := r.URL.Query()

	// Build filter DTO for validation
	reqParams := UserRoleFilterDTO{
		Name:                 ptr.PtrOrNil(q.Get("name")),
		Description:          ptr.PtrOrNil(q.Get("description")),
		Status:               ptr.PtrOrNil(q.Get("status")),
		PaginationRequestDTO: pagination.ParseQuery(r),
	}

	// Validate filter parameters
	if err := reqParams.Validate(); err != nil {
		resp.ValidationError(w, err)
		return
	}

	// Get tenant from context (middleware already validated access)
	tenant := middleware.AuthFromRequest(r).Tenant
	if tenant == nil {
		resp.Error(w, http.StatusUnauthorized, "Tenant not found in context")
		return
	}

	// Build service filter with pagination and sorting params
	filter := GetUserRolesFilter{
		Name:      reqParams.Name,
		Description: reqParams.Description,
		Status:    reqParams.Status,
		Page:      reqParams.Page,
		Limit:     reqParams.Limit,
		SortBy:    reqParams.SortBy,
		SortOrder: reqParams.SortOrder,
	}

	// Fetch roles for the user (service validates tenant ownership internally)
	roles, total, err := h.userService.GetUserRoles(r.Context(), userUUID, tenant.TenantID, filter)
	if err != nil {
		resp.HandleServiceError(w, r, "Failed to fetch user roles", err)
		return
	}

	// Map to DTOs
	rows := make([]RoleResponseDTO, len(roles))
	for i, role := range roles {
		rows[i] = RoleResponseDTO{
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
	response := PaginatedResponseDTO[RoleResponseDTO]{
		Rows:       rows,
		Total:      total,
		Page:       reqParams.Page,
		Limit:      reqParams.Limit,
		TotalPages: totalPages,
	}

	resp.Success(w, response, "User roles fetched successfully")
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
		resp.Error(w, http.StatusBadRequest, "Invalid user UUID")
		return
	}

	// Parse query parameters
	q := r.URL.Query()

	// Build filter DTO for validation
	reqParams := UserIdentityFilterDTO{
		Provider:             ptr.PtrOrNil(q.Get("provider")),
		PaginationRequestDTO: pagination.ParseQuery(r),
	}

	// Validate filter parameters
	if err := reqParams.Validate(); err != nil {
		resp.ValidationError(w, err)
		return
	}

	// Get tenant from context (middleware already validated access)
	tenant := middleware.AuthFromRequest(r).Tenant
	if tenant == nil {
		resp.Error(w, http.StatusUnauthorized, "Tenant not found in context")
		return
	}

	// Build service filter with pagination and sorting params
	filter := GetUserIdentitiesFilter{
		Provider:  reqParams.Provider,
		Page:      reqParams.Page,
		Limit:     reqParams.Limit,
		SortBy:    reqParams.SortBy,
		SortOrder: reqParams.SortOrder,
	}

	// Fetch identities for the user (service validates tenant ownership internally)
	identities, total, err := h.userService.GetUserIdentities(r.Context(), userUUID, tenant.TenantID, filter)
	if err != nil {
		resp.HandleServiceError(w, r, "Failed to fetch user identities", err)
		return
	}

	// Map to DTOs
	rows := make([]UserIdentityResponseDTO, len(identities))
	for i, identity := range identities {
		rows[i] = UserIdentityResponseDTO{
			UserIdentityUUID: identity.UserIdentityUUID,
			Provider:         identity.Provider,
			Sub:              identity.Sub,
			Metadata:         identity.Metadata,
			CreatedAt:        identity.CreatedAt,
			UpdatedAt:        identity.UpdatedAt,
		}
		if identity.Client != nil {
			rows[i].Client = &ClientResponseDTO{
				ClientUUID:  identity.Client.ClientUUID,
				Name:        identity.Client.Name,
				DisplayName: identity.Client.DisplayName,
				ClientType:  identity.Client.ClientType,
				Domain:      identity.Client.Domain,
				Status:      identity.Client.Status,
				IsDefault:   identity.Client.IsDefault,
				IsSystem:    identity.Client.IsSystem,
				CreatedAt:   identity.Client.CreatedAt,
				UpdatedAt:   identity.Client.UpdatedAt,
			}
		}
	}

	totalPages := int((total + int64(reqParams.Limit) - 1) / int64(reqParams.Limit))
	response := PaginatedResponseDTO[UserIdentityResponseDTO]{
		Rows:       rows,
		Total:      total,
		Page:       reqParams.Page,
		Limit:      reqParams.Limit,
		TotalPages: totalPages,
	}

	resp.Success(w, response, "User identities fetched successfully")
}


