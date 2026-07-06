package user

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"
	"github.com/maintainerd/maintainerd-auth/internal/platform/apperror"
	authv1 "github.com/maintainerd/maintainerd-auth/internal/platform/gen/go/maintainerd/auth"
	"github.com/maintainerd/maintainerd-auth/internal/platform/pagination"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"
	"gorm.io/datatypes"
)

type TenantResolver interface {
	GetByUUID(ctx context.Context, tenantUUID uuid.UUID) (*TenantServiceDataResult, error)
}

type UserGRPCHandler struct {
	authv1.UnimplementedUserServiceServer
	tenantResolver TenantResolver
	userService    UserService
}

func NewUserGRPCHandler(tenantResolver TenantResolver, userService UserService) *UserGRPCHandler {
	return &UserGRPCHandler{tenantResolver: tenantResolver, userService: userService}
}

func (h *UserGRPCHandler) ListUsers(ctx context.Context, req *authv1.ListUsersRequest) (*authv1.ListUsersResponse, error) {
	tenant, err := h.resolveTenant(ctx, req.GetTenantUuid())
	if err != nil {
		return nil, err
	}
	dto := grpcPagination(req.GetPagination())
	if err := dto.Validate(); err != nil {
		return nil, apperror.ToGRPCError(apperror.NewValidation(err.Error()))
	}
	result, err := h.userService.Get(ctx, UserServiceGetFilter{
		TenantID:  tenant.TenantID,
		Username:  optionalStr(req.GetUsername()),
		Email:     optionalEmail(req.GetEmail()),
		Phone:     optionalPhone(req.GetPhone()),
		Status:    req.GetStatus(),
		Page:      dto.Page,
		Limit:     dto.Limit,
		SortBy:    dto.SortBy,
		SortOrder: dto.SortOrder,
	})
	if err != nil {
		return nil, apperror.ToGRPCError(err)
	}
	rows := make([]*authv1.User, len(result.Data))
	for i := range result.Data {
		rows[i] = toUserProto(&result.Data[i])
	}
	return &authv1.ListUsersResponse{Users: rows, Page: grpcPageProto(result.Total, result.Page, result.Limit, result.TotalPages)}, nil
}

func (h *UserGRPCHandler) GetUser(ctx context.Context, req *authv1.GetUserRequest) (*authv1.GetUserResponse, error) {
	tenant, err := h.resolveTenant(ctx, req.GetTenantUuid())
	if err != nil {
		return nil, err
	}
	userUUID, err := grpcUUID(req.GetUserUuid(), "User UUID")
	if err != nil {
		return nil, err
	}
	result, err := h.userService.GetByUUID(ctx, userUUID, tenant.TenantID)
	if err != nil {
		return nil, apperror.ToGRPCError(err)
	}
	return &authv1.GetUserResponse{User: toUserProto(result)}, nil
}

func (h *UserGRPCHandler) CreateUser(ctx context.Context, req *authv1.CreateUserRequest) (*authv1.CreateUserResponse, error) {
	tenant, err := h.resolveTenant(ctx, req.GetTenantUuid())
	if err != nil {
		return nil, err
	}
	actorUUID := uuid.Nil
	if req.GetActorUserUuid() != "" {
		actorUUID, err = grpcUUID(req.GetActorUserUuid(), "Actor user UUID")
		if err != nil {
			return nil, err
		}
	}
	metadata := structToJSON(req.GetMetadata())
	result, err := h.userService.Create(ctx, req.GetUsername(), optionalEmail(req.GetEmail()), optionalPhone(req.GetPhone()), req.GetPassword(), req.GetStatus(), metadata, tenant.TenantUUID.String(), actorUUID)
	if err != nil {
		return nil, apperror.ToGRPCError(err)
	}
	return &authv1.CreateUserResponse{User: toUserProto(result)}, nil
}

func (h *UserGRPCHandler) UpdateUser(ctx context.Context, req *authv1.UpdateUserRequest) (*authv1.UpdateUserResponse, error) {
	tenant, err := h.resolveTenant(ctx, req.GetTenantUuid())
	if err != nil {
		return nil, err
	}
	userUUID, err := grpcUUID(req.GetUserUuid(), "User UUID")
	if err != nil {
		return nil, err
	}
	actorUUID := uuid.Nil
	if req.GetActorUserUuid() != "" {
		actorUUID, err = grpcUUID(req.GetActorUserUuid(), "Actor user UUID")
		if err != nil {
			return nil, err
		}
	}
	metadata := structToJSON(req.GetMetadata())
	result, err := h.userService.Update(ctx, userUUID, tenant.TenantID, req.GetUsername(), optionalEmail(req.GetEmail()), optionalPhone(req.GetPhone()), req.GetStatus(), metadata, actorUUID)
	if err != nil {
		return nil, apperror.ToGRPCError(err)
	}
	return &authv1.UpdateUserResponse{User: toUserProto(result)}, nil
}

func (h *UserGRPCHandler) SetUserStatus(ctx context.Context, req *authv1.SetUserStatusRequest) (*authv1.SetUserStatusResponse, error) {
	tenant, err := h.resolveTenant(ctx, req.GetTenantUuid())
	if err != nil {
		return nil, err
	}
	userUUID, err := grpcUUID(req.GetUserUuid(), "User UUID")
	if err != nil {
		return nil, err
	}
	actorUUID := uuid.Nil
	if req.GetActorUserUuid() != "" {
		actorUUID, err = grpcUUID(req.GetActorUserUuid(), "Actor user UUID")
		if err != nil {
			return nil, err
		}
	}
	result, err := h.userService.SetStatus(ctx, userUUID, tenant.TenantID, req.GetStatus(), actorUUID)
	if err != nil {
		return nil, apperror.ToGRPCError(err)
	}
	return &authv1.SetUserStatusResponse{User: toUserProto(result)}, nil
}

func (h *UserGRPCHandler) VerifyUserEmail(ctx context.Context, req *authv1.VerifyUserEmailRequest) (*authv1.VerifyUserEmailResponse, error) {
	tenant, err := h.resolveTenant(ctx, req.GetTenantUuid())
	if err != nil {
		return nil, err
	}
	userUUID, err := grpcUUID(req.GetUserUuid(), "User UUID")
	if err != nil {
		return nil, err
	}
	result, err := h.userService.VerifyEmail(ctx, userUUID, tenant.TenantID)
	if err != nil {
		return nil, apperror.ToGRPCError(err)
	}
	return &authv1.VerifyUserEmailResponse{User: toUserProto(result)}, nil
}

func (h *UserGRPCHandler) VerifyUserPhone(ctx context.Context, req *authv1.VerifyUserPhoneRequest) (*authv1.VerifyUserPhoneResponse, error) {
	tenant, err := h.resolveTenant(ctx, req.GetTenantUuid())
	if err != nil {
		return nil, err
	}
	userUUID, err := grpcUUID(req.GetUserUuid(), "User UUID")
	if err != nil {
		return nil, err
	}
	result, err := h.userService.VerifyPhone(ctx, userUUID, tenant.TenantID)
	if err != nil {
		return nil, apperror.ToGRPCError(err)
	}
	return &authv1.VerifyUserPhoneResponse{User: toUserProto(result)}, nil
}

func (h *UserGRPCHandler) CompleteUserAccount(ctx context.Context, req *authv1.CompleteUserAccountRequest) (*authv1.CompleteUserAccountResponse, error) {
	tenant, err := h.resolveTenant(ctx, req.GetTenantUuid())
	if err != nil {
		return nil, err
	}
	userUUID, err := grpcUUID(req.GetUserUuid(), "User UUID")
	if err != nil {
		return nil, err
	}
	result, err := h.userService.CompleteAccount(ctx, userUUID, tenant.TenantID)
	if err != nil {
		return nil, apperror.ToGRPCError(err)
	}
	return &authv1.CompleteUserAccountResponse{User: toUserProto(result)}, nil
}

func (h *UserGRPCHandler) DeleteUser(ctx context.Context, req *authv1.DeleteUserRequest) (*authv1.DeleteUserResponse, error) {
	tenant, err := h.resolveTenant(ctx, req.GetTenantUuid())
	if err != nil {
		return nil, err
	}
	userUUID, err := grpcUUID(req.GetUserUuid(), "User UUID")
	if err != nil {
		return nil, err
	}
	actorUUID := uuid.Nil
	if req.GetActorUserUuid() != "" {
		actorUUID, err = grpcUUID(req.GetActorUserUuid(), "Actor user UUID")
		if err != nil {
			return nil, err
		}
	}
	result, err := h.userService.DeleteByUUID(ctx, userUUID, tenant.TenantID, actorUUID)
	if err != nil {
		return nil, apperror.ToGRPCError(err)
	}
	return &authv1.DeleteUserResponse{User: toUserProto(result)}, nil
}

func (h *UserGRPCHandler) ForceUserPasswordChange(ctx context.Context, req *authv1.ForceUserPasswordChangeRequest) (*authv1.ForceUserPasswordChangeResponse, error) {
	tenant, err := h.resolveTenant(ctx, req.GetTenantUuid())
	if err != nil {
		return nil, err
	}
	userUUID, err := grpcUUID(req.GetUserUuid(), "User UUID")
	if err != nil {
		return nil, err
	}
	if err := h.userService.ForcePasswordChange(ctx, userUUID, tenant.TenantID, req.GetForce()); err != nil {
		return nil, apperror.ToGRPCError(err)
	}
	return &authv1.ForceUserPasswordChangeResponse{Success: true}, nil
}

func (h *UserGRPCHandler) ListUserRoles(ctx context.Context, req *authv1.ListUserRolesRequest) (*authv1.ListUserRolesResponse, error) {
	tenant, err := h.resolveTenant(ctx, req.GetTenantUuid())
	if err != nil {
		return nil, err
	}
	userUUID, err := grpcUUID(req.GetUserUuid(), "User UUID")
	if err != nil {
		return nil, err
	}
	dto := grpcPagination(req.GetPagination())
	roles, total, err := h.userService.GetUserRoles(ctx, userUUID, tenant.TenantID, GetUserRolesFilter{Page: dto.Page, Limit: dto.Limit, SortBy: dto.SortBy, SortOrder: dto.SortOrder})
	if err != nil {
		return nil, apperror.ToGRPCError(err)
	}
	rows := make([]*authv1.UserRole, len(roles))
	for i := range roles {
		rows[i] = toUserRoleProto(&roles[i])
	}
	totalPages := 0
	if dto.Limit > 0 {
		totalPages = int(total) / dto.Limit
		if int(total)%dto.Limit > 0 {
			totalPages++
		}
	}
	return &authv1.ListUserRolesResponse{Roles: rows, Page: grpcPageProto(total, dto.Page, dto.Limit, totalPages)}, nil
}

func (h *UserGRPCHandler) ListUserIdentities(ctx context.Context, req *authv1.ListUserIdentitiesRequest) (*authv1.ListUserIdentitiesResponse, error) {
	tenant, err := h.resolveTenant(ctx, req.GetTenantUuid())
	if err != nil {
		return nil, err
	}
	userUUID, err := grpcUUID(req.GetUserUuid(), "User UUID")
	if err != nil {
		return nil, err
	}
	dto := grpcPagination(req.GetPagination())
	identities, total, err := h.userService.GetUserIdentities(ctx, userUUID, tenant.TenantID, GetUserIdentitiesFilter{Page: dto.Page, Limit: dto.Limit, SortBy: dto.SortBy, SortOrder: dto.SortOrder})
	if err != nil {
		return nil, apperror.ToGRPCError(err)
	}
	rows := make([]*authv1.UserIdentity, len(identities))
	for i := range identities {
		rows[i] = toUserIdentityProto(&identities[i])
	}
	totalPages := 0
	if dto.Limit > 0 {
		totalPages = int(total) / dto.Limit
		if int(total)%dto.Limit > 0 {
			totalPages++
		}
	}
	return &authv1.ListUserIdentitiesResponse{Identities: rows, Page: grpcPageProto(total, dto.Page, dto.Limit, totalPages)}, nil
}

func (h *UserGRPCHandler) AssignUserRoles(ctx context.Context, req *authv1.AssignUserRolesRequest) (*authv1.AssignUserRolesResponse, error) {
	tenant, err := h.resolveTenant(ctx, req.GetTenantUuid())
	if err != nil {
		return nil, err
	}
	userUUID, err := grpcUUID(req.GetUserUuid(), "User UUID")
	if err != nil {
		return nil, err
	}
	roleUUIDs := make([]uuid.UUID, len(req.GetRoleUuids()))
	for i, r := range req.GetRoleUuids() {
		parsed, err := grpcUUID(r, "Role UUID")
		if err != nil {
			return nil, err
		}
		roleUUIDs[i] = parsed
	}
	result, err := h.userService.AssignUserRoles(ctx, userUUID, roleUUIDs, tenant.TenantID)
	if err != nil {
		return nil, apperror.ToGRPCError(err)
	}
	return &authv1.AssignUserRolesResponse{User: toUserProto(result)}, nil
}

func (h *UserGRPCHandler) RemoveUserRole(ctx context.Context, req *authv1.RemoveUserRoleRequest) (*authv1.RemoveUserRoleResponse, error) {
	tenant, err := h.resolveTenant(ctx, req.GetTenantUuid())
	if err != nil {
		return nil, err
	}
	userUUID, err := grpcUUID(req.GetUserUuid(), "User UUID")
	if err != nil {
		return nil, err
	}
	roleUUID, err := grpcUUID(req.GetRoleUuid(), "Role UUID")
	if err != nil {
		return nil, err
	}
	result, err := h.userService.RemoveUserRole(ctx, userUUID, roleUUID, tenant.TenantID)
	if err != nil {
		return nil, apperror.ToGRPCError(err)
	}
	return &authv1.RemoveUserRoleResponse{User: toUserProto(result)}, nil
}

func (h *UserGRPCHandler) resolveTenant(ctx context.Context, tenantUUID string) (*TenantServiceDataResult, error) {
	parsed, err := grpcUUID(tenantUUID, "Tenant UUID")
	if err != nil {
		return nil, err
	}
	result, err := h.tenantResolver.GetByUUID(ctx, parsed)
	if err != nil {
		return nil, apperror.ToGRPCError(err)
	}
	return result, nil
}

func grpcUUID(value string, label string) (uuid.UUID, error) {
	if value == "" {
		return uuid.Nil, apperror.ToGRPCError(apperror.NewValidation(label + " is required"))
	}
	parsed, err := uuid.Parse(value)
	if err != nil {
		return uuid.Nil, apperror.ToGRPCError(apperror.NewValidation("Invalid " + label))
	}
	return parsed, nil
}

func optionalStr(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}

func optionalEmail(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}

func optionalPhone(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}

func grpcPagination(req *authv1.Pagination) PaginationRequestDTO {
	if req == nil {
		return PaginationRequestDTO{Page: 1, Limit: pagination.DefaultPageSize}
	}
	page := int(req.GetPage())
	limit := int(req.GetLimit())
	if page == 0 {
		page = 1
	}
	if limit == 0 {
		limit = int(pagination.DefaultPageSize)
	}
	return PaginationRequestDTO{Page: page, Limit: limit, SortBy: req.GetSortBy(), SortOrder: req.GetSortOrder()}
}

func grpcPageProto(total int64, page int, limit int, totalPages int) *authv1.PageMetadata {
	return &authv1.PageMetadata{Total: total, Page: int32(page), Limit: int32(limit), TotalPages: int32(totalPages)}
}

func structToJSON(s *structpb.Struct) datatypes.JSON {
	if s == nil {
		return nil
	}
	payload, _ := json.Marshal(s.AsMap())
	return datatypes.JSON(payload)
}

func toUserProto(result *UserServiceDataResult) *authv1.User {
	if result == nil {
		return nil
	}
	return &authv1.User{
		UserUuid:        result.UserUUID.String(),
		Username:        result.Username,
		Fullname:        result.Fullname,
		Email:           result.Email,
		Phone:           result.Phone,
		IsEmailVerified: result.IsEmailVerified,
		IsPhoneVerified: result.IsPhoneVerified,
		Status:          result.Status,
		CreatedAt:       timestamppb.New(result.CreatedAt),
		UpdatedAt:       timestamppb.New(result.UpdatedAt),
	}
}

func toUserRoleProto(result *RoleServiceDataResult) *authv1.UserRole {
	if result == nil {
		return nil
	}
	return &authv1.UserRole{
		RoleUuid:    result.RoleUUID.String(),
		Name:        result.Name,
		Description: result.Description,
		IsDefault:   result.IsDefault,
		IsSystem:    result.IsSystem,
		Status:      result.Status,
		CreatedAt:   timestamppb.New(result.CreatedAt),
		UpdatedAt:   timestamppb.New(result.UpdatedAt),
	}
}

func toUserIdentityProto(result *UserIdentityServiceDataResult) *authv1.UserIdentity {
	if result == nil {
		return nil
	}
	return &authv1.UserIdentity{
		UserIdentityUuid: result.UserIdentityUUID.String(),
		Provider:         result.Provider,
		Sub:              result.Sub,
		CreatedAt:        timestamppb.New(result.CreatedAt),
		UpdatedAt:        timestamppb.New(result.UpdatedAt),
	}
}
