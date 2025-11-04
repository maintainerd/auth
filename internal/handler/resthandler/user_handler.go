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

	// Parse bools safely
	var isActive *bool
	if v := q.Get("is_active"); v != "" {
		if parsed, err := strconv.ParseBool(v); err == nil {
			isActive = &parsed
		}
	}

	// Parse auth container UUID safely
	var authContainerUUID *string
	if v := q.Get("auth_container_uuid"); v != "" {
		authContainerUUID = &v
	}

	// Build request DTO (for validation)
	reqParams := dto.UserFilterDto{
		Username:   util.PtrOrNil(q.Get("username")),
		Email:      util.PtrOrNil(q.Get("email")),
		Phone:      util.PtrOrNil(q.Get("phone")),
		IsActive:   isActive,
		TenantUUID: authContainerUUID,
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
		IsActive:   reqParams.IsActive,
		TenantUUID: reqParams.TenantUUID,
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
	user, err := h.userService.Create(req.Username, req.Email, req.Phone, req.Password, req.TenantUUID, creatorUser.UserUUID)
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
	user, err := h.userService.Update(userUUID, req.Username, req.Email, req.Phone, updaterUser.UserUUID)
	if err != nil {
		util.Error(w, http.StatusInternalServerError, "Failed to update user", err.Error())
		return
	}

	// Build response data
	dtoRes := toUserResponseDto(*user)

	util.Success(w, dtoRes, "User updated successfully")
}

// Set user active status
func (h *UserHandler) SetUserActiveStatus(w http.ResponseWriter, r *http.Request) {
	// Get authentication context
	updaterUser := r.Context().Value(middleware.UserContextKey).(*model.User)

	userUUIDStr := chi.URLParam(r, "user_uuid")
	userUUID, err := uuid.Parse(userUUIDStr)
	if err != nil {
		util.Error(w, http.StatusBadRequest, "Invalid user UUID")
		return
	}

	var req dto.UserSetActiveStatusRequestDto
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
	user, err := h.userService.SetActiveStatus(userUUID, req.IsActive, updaterUser.UserUUID)
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
		Email:              u.Email,
		Phone:              u.Phone,
		IsEmailVerified:    u.IsEmailVerified,
		IsPhoneVerified:    u.IsPhoneVerified,
		IsProfileCompleted: u.IsProfileCompleted,
		IsAccountCompleted: u.IsAccountCompleted,
		IsActive:           u.IsActive,
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
			IsActive:    u.Tenant.IsActive,
			IsPublic:    u.Tenant.IsPublic,
			IsDefault:   u.Tenant.IsDefault,
			CreatedAt:   u.Tenant.CreatedAt,
			UpdatedAt:   u.Tenant.UpdatedAt,
		}
	}

	// Map UserIdentities if present
	if u.UserIdentities != nil {
		userIdentities := make([]dto.UserIdentityResponseDto, len(*u.UserIdentities))
		for i, ui := range *u.UserIdentities {
			userIdentities[i] = dto.UserIdentityResponseDto{
				UserIdentityUUID: ui.UserIdentityUUID,
				Provider:         ui.Provider,
				Sub:              ui.Sub,
				Metadata:         ui.Metadata,
				CreatedAt:        ui.CreatedAt,
				UpdatedAt:        ui.UpdatedAt,
			}
			// Map AuthClient if present
			if ui.AuthClient != nil {
				userIdentities[i].AuthClient = &dto.AuthClientResponseDto{
					AuthClientUUID: ui.AuthClient.AuthClientUUID,
					Name:           ui.AuthClient.Name,
					DisplayName:    ui.AuthClient.DisplayName,
					ClientType:     ui.AuthClient.ClientType,
					Domain:         ui.AuthClient.Domain,
					IsActive:       ui.AuthClient.IsActive,
					IsDefault:      ui.AuthClient.IsDefault,
					CreatedAt:      ui.AuthClient.CreatedAt,
					UpdatedAt:      ui.AuthClient.UpdatedAt,
				}
			}
		}
		result.UserIdentities = &userIdentities
	}

	// Map Roles if present
	if u.Roles != nil {
		roles := make([]dto.RoleResponseDto, len(*u.Roles))
		for i, role := range *u.Roles {
			roles[i] = dto.RoleResponseDto{
				RoleUUID:    role.RoleUUID,
				Name:        role.Name,
				Description: role.Description,
				IsDefault:   role.IsDefault,
				IsActive:    role.IsActive,
				CreatedAt:   role.CreatedAt,
				UpdatedAt:   role.UpdatedAt,
			}
			// Map Permissions if present
			if role.Permissions != nil {
				permissions := make([]dto.PermissionResponseDto, len(*role.Permissions))
				for j, permission := range *role.Permissions {
					permissions[j] = dto.PermissionResponseDto{
						PermissionUUID: permission.PermissionUUID,
						Name:           permission.Name,
						Description:    permission.Description,
						IsActive:       permission.IsActive,
						IsDefault:      permission.IsDefault,
						CreatedAt:      permission.CreatedAt,
						UpdatedAt:      permission.UpdatedAt,
					}
				}
				roles[i].Permissions = &permissions
			}
		}
		result.Roles = &roles
	}

	return result
}
