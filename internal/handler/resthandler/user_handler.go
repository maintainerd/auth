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

type UserHandler struct {
	userService service.UserService
}

func NewUserHandler(userService service.UserService) *UserHandler {
	return &UserHandler{
		userService: userService,
	}
}

// Get users with pagination and filtering
func (h *UserHandler) GetUsers(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()

	// Parse pagination
	page, _ := strconv.Atoi(q.Get("page"))
	limit, _ := strconv.Atoi(q.Get("limit"))

	// Parse status filter (can be multiple)
	var status []string
	if v := q.Get("status"); v != "" {
		status = append(status, v)
	}

	// Parse tenant UUID safely
	var tenantUUID *string
	if v := q.Get("tenant_id"); v != "" {
		tenantUUID = &v
	}

	// Parse role UUID safely
	var roleUUID *string
	if v := q.Get("role_id"); v != "" {
		roleUUID = &v
	}

	// Build request DTO (for validation)
	reqParams := dto.UserFilterDto{
		Username:   util.PtrOrNil(q.Get("username")),
		Email:      util.PtrOrNil(q.Get("email")),
		Phone:      util.PtrOrNil(q.Get("phone")),
		Status:     status,
		TenantUUID: tenantUUID,
		RoleUUID:   roleUUID,
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

	// Convert to service filter
	filter := service.UserServiceGetFilter{
		Username:   reqParams.Username,
		Email:      reqParams.Email,
		Phone:      reqParams.Phone,
		Status:     reqParams.Status,
		TenantUUID: reqParams.TenantUUID,
		RoleUUID:   reqParams.RoleUUID,
		Page:       reqParams.PaginationRequestDto.Page,
		Limit:      reqParams.PaginationRequestDto.Limit,
		SortBy:     reqParams.PaginationRequestDto.SortBy,
		SortOrder:  reqParams.PaginationRequestDto.SortOrder,
	}

	// Get users
	result, err := h.userService.Get(filter)
	if err != nil {
		util.Error(w, http.StatusInternalServerError, "Failed to fetch users", err.Error())
		return
	}

	// Map service result to dto
	rows := make([]dto.UserResponseDto, len(result.Data))
	for i, r := range result.Data {
		rows[i] = toUserResponseDto(r)
	}

	// Build response data
	response := dto.PaginatedResponseDto[dto.UserResponseDto]{
		Rows:       rows,
		Total:      result.Total,
		Page:       result.Page,
		Limit:      result.Limit,
		TotalPages: result.TotalPages,
	}

	util.Success(w, response, "Users fetched successfully")
}

// Get user by UUID
func (h *UserHandler) GetUser(w http.ResponseWriter, r *http.Request) {
	userUUIDStr := chi.URLParam(r, "user_uuid")
	userUUID, err := uuid.Parse(userUUIDStr)
	if err != nil {
		util.Error(w, http.StatusBadRequest, "Invalid user UUID")
		return
	}

	user, err := h.userService.GetByUUID(userUUID)
	if err != nil {
		util.Error(w, http.StatusNotFound, "User not found")
		return
	}

	// Build response data
	dtoRes := toUserResponseDto(*user)

	util.Success(w, dtoRes, "User fetched successfully")
}

// Create user
func (h *UserHandler) CreateUser(w http.ResponseWriter, r *http.Request) {
	// Get authentication context
	creatorUser := r.Context().Value(middleware.UserContextKey).(*model.User)

	var req dto.UserCreateRequestDto
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		util.Error(w, http.StatusBadRequest, "Invalid JSON format")
		return
	}

	// Validate request
	if err := req.Validate(); err != nil {
		util.Error(w, http.StatusBadRequest, "Validation failed", err.Error())
		return
	}

	// Create user with creator context
	user, err := h.userService.Create(req.Username, req.Fullname, req.Email, req.Phone, req.Password, req.Status, req.Metadata, req.TenantUUID, creatorUser.UserUUID)
	if err != nil {
		util.Error(w, http.StatusInternalServerError, "Failed to create user", err.Error())
		return
	}

	// Build response data
	dtoRes := toUserResponseDto(*user)

	util.Created(w, dtoRes, "User created successfully")
}

// Update user
func (h *UserHandler) UpdateUser(w http.ResponseWriter, r *http.Request) {
	// Get authentication context
	updaterUser := r.Context().Value(middleware.UserContextKey).(*model.User)

	userUUIDStr := chi.URLParam(r, "user_uuid")
	userUUID, err := uuid.Parse(userUUIDStr)
	if err != nil {
		util.Error(w, http.StatusBadRequest, "Invalid user UUID")
		return
	}

	var req dto.UserUpdateRequestDto
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		util.Error(w, http.StatusBadRequest, "Invalid JSON format")
		return
	}

	// Validate request
	if err := req.Validate(); err != nil {
		util.Error(w, http.StatusBadRequest, "Validation failed", err.Error())
		return
	}

	// Update user
	user, err := h.userService.Update(userUUID, req.Username, req.Fullname, req.Email, req.Phone, req.Status, req.Metadata, updaterUser.UserUUID)
	if err != nil {
		util.Error(w, http.StatusInternalServerError, "Failed to update user", err.Error())
		return
	}

	// Build response data
	dtoRes := toUserResponseDto(*user)

	util.Success(w, dtoRes, "User updated successfully")
}

// Set user active status
func (h *UserHandler) SetUserStatus(w http.ResponseWriter, r *http.Request) {
	// Get authentication context
	updaterUser := r.Context().Value(middleware.UserContextKey).(*model.User)

	userUUIDStr := chi.URLParam(r, "user_uuid")
	userUUID, err := uuid.Parse(userUUIDStr)
	if err != nil {
		util.Error(w, http.StatusBadRequest, "Invalid user UUID")
		return
	}

	var req dto.UserSetStatusRequestDto
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		util.Error(w, http.StatusBadRequest, "Invalid JSON format")
		return
	}

	// Validate request
	if err := req.Validate(); err != nil {
		util.Error(w, http.StatusBadRequest, "Validation failed", err.Error())
		return
	}

	// Update user status
	user, err := h.userService.SetStatus(userUUID, req.Status, updaterUser.UserUUID)
	if err != nil {
		util.Error(w, http.StatusInternalServerError, "Failed to update user status", err.Error())
		return
	}

	// Build response data
	dtoRes := toUserResponseDto(*user)

	util.Success(w, dtoRes, "User status updated successfully")
}

// Delete user
func (h *UserHandler) DeleteUser(w http.ResponseWriter, r *http.Request) {
	// Get authentication context
	deleterUser := r.Context().Value(middleware.UserContextKey).(*model.User)

	userUUIDStr := chi.URLParam(r, "user_uuid")
	userUUID, err := uuid.Parse(userUUIDStr)
	if err != nil {
		util.Error(w, http.StatusBadRequest, "Invalid user UUID")
		return
	}

	// Delete user
	user, err := h.userService.DeleteByUUID(userUUID, deleterUser.UserUUID)
	if err != nil {
		util.Error(w, http.StatusInternalServerError, "Failed to delete user", err.Error())
		return
	}

	// Build response data
	dtoRes := toUserResponseDto(*user)

	util.Success(w, dtoRes, "User deleted successfully")
}

// Assign roles to user
func (h *UserHandler) AssignRoles(w http.ResponseWriter, r *http.Request) {
	userUUIDStr := chi.URLParam(r, "user_uuid")
	userUUID, err := uuid.Parse(userUUIDStr)
	if err != nil {
		util.Error(w, http.StatusBadRequest, "Invalid user UUID")
		return
	}

	var req dto.UserAssignRolesRequestDto
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		util.Error(w, http.StatusBadRequest, "Invalid JSON format")
		return
	}

	// Validate request
	if err := req.Validate(); err != nil {
		util.Error(w, http.StatusBadRequest, "Validation failed", err.Error())
		return
	}

	// Assign roles to user
	user, err := h.userService.AssignUserRoles(userUUID, req.RoleUUIDs)
	if err != nil {
		util.Error(w, http.StatusInternalServerError, "Failed to assign roles to user", err.Error())
		return
	}

	// Build response data
	dtoRes := toUserResponseDto(*user)

	util.Success(w, dtoRes, "Roles assigned to user successfully")
}

// Remove role from user
func (h *UserHandler) RemoveRole(w http.ResponseWriter, r *http.Request) {
	userUUIDStr := chi.URLParam(r, "user_uuid")
	userUUID, err := uuid.Parse(userUUIDStr)
	if err != nil {
		util.Error(w, http.StatusBadRequest, "Invalid user UUID")
		return
	}

	roleUUIDStr := chi.URLParam(r, "role_uuid")
	roleUUID, err := uuid.Parse(roleUUIDStr)
	if err != nil {
		util.Error(w, http.StatusBadRequest, "Invalid role UUID")
		return
	}

	// Remove role from user
	user, err := h.userService.RemoveUserRole(userUUID, roleUUID)
	if err != nil {
		util.Error(w, http.StatusInternalServerError, "Failed to remove role from user", err.Error())
		return
	}

	// Build response data
	dtoRes := toUserResponseDto(*user)

	util.Success(w, dtoRes, "Role removed from user successfully")
}

// Convert service result to dto
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

// Get user roles with pagination
func (h *UserHandler) GetUserRoles(w http.ResponseWriter, r *http.Request) {
	userUUIDStr := chi.URLParam(r, "user_uuid")
	userUUID, err := uuid.Parse(userUUIDStr)
	if err != nil {
		util.Error(w, http.StatusBadRequest, "Invalid user UUID")
		return
	}

	q := r.URL.Query()

	// Parse pagination
	page, _ := strconv.Atoi(q.Get("page"))
	limit, _ := strconv.Atoi(q.Get("limit"))

	// Build request DTO
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

	if err := reqParams.Validate(); err != nil {
		util.ValidationError(w, err)
		return
	}

	// Get user to verify it exists
	user, err := h.userService.GetByUUID(userUUID)
	if err != nil {
		util.Error(w, http.StatusNotFound, "User not found")
		return
	}

	// Get roles for the user
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

// Get user identities with pagination
func (h *UserHandler) GetUserIdentities(w http.ResponseWriter, r *http.Request) {
	userUUIDStr := chi.URLParam(r, "user_uuid")
	userUUID, err := uuid.Parse(userUUIDStr)
	if err != nil {
		util.Error(w, http.StatusBadRequest, "Invalid user UUID")
		return
	}

	q := r.URL.Query()

	// Parse pagination
	page, _ := strconv.Atoi(q.Get("page"))
	limit, _ := strconv.Atoi(q.Get("limit"))

	// Build request DTO
	reqParams := dto.UserIdentityFilterDto{
		Provider: util.PtrOrNil(q.Get("provider")),
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

	// Get user to verify it exists
	user, err := h.userService.GetByUUID(userUUID)
	if err != nil {
		util.Error(w, http.StatusNotFound, "User not found")
		return
	}

	// Get identities for the user
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
