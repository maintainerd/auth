package tenant

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"
	"github.com/maintainerd/maintainerd-auth/internal/platform/apperror"
	authv1 "github.com/maintainerd/maintainerd-auth/internal/platform/gen/go/maintainerd/auth"
	"github.com/maintainerd/maintainerd-auth/internal/platform/pagination"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type TenantGRPCHandler struct {
	authv1.UnimplementedTenantServiceServer
	tenantService       TenantService
	tenantMemberService TenantMemberService
}

func NewTenantGRPCHandler(tenantService TenantService, tenantMemberService TenantMemberService) *TenantGRPCHandler {
	return &TenantGRPCHandler{tenantService: tenantService, tenantMemberService: tenantMemberService}
}

func (h *TenantGRPCHandler) GetDefaultTenant(ctx context.Context, _ *authv1.GetDefaultTenantRequest) (*authv1.GetDefaultTenantResponse, error) {
	tenant, err := h.tenantService.GetSystem(ctx)
	if err != nil {
		return nil, apperror.ToGRPCError(err)
	}
	return &authv1.GetDefaultTenantResponse{Tenant: tenantProto(tenant)}, nil
}

func (h *TenantGRPCHandler) GetTenantByIdentifier(ctx context.Context, req *authv1.GetTenantByIdentifierRequest) (*authv1.GetTenantByIdentifierResponse, error) {
	if req.GetIdentifier() == "" {
		return nil, apperror.ToGRPCError(apperror.NewValidation("Identifier is required"))
	}
	tenant, err := h.tenantService.GetByIdentifier(ctx, req.GetIdentifier())
	if err != nil {
		return nil, apperror.ToGRPCError(err)
	}
	return &authv1.GetTenantByIdentifierResponse{Tenant: tenantProto(tenant)}, nil
}

func (h *TenantGRPCHandler) ListTenants(ctx context.Context, req *authv1.ListTenantsRequest) (*authv1.ListTenantsResponse, error) {
	dto := TenantFilterDTO{
		Name:                 optionalString(req.GetName()),
		DisplayName:          optionalString(req.GetDisplayName()),
		Description:          optionalString(req.GetDescription()),
		Identifier:           optionalString(req.GetIdentifier()),
		Status:               req.GetStatus(),
		IsSystem:             optionalBool(req.IsSystem),
		PaginationRequestDTO: paginationDTO(req.GetPagination()),
	}
	if err := dto.Validate(); err != nil {
		return nil, apperror.ToGRPCError(apperror.NewValidation(err.Error()))
	}

	result, err := h.tenantService.Get(ctx, TenantServiceGetFilter{
		Name:        dto.Name,
		DisplayName: dto.DisplayName,
		Description: dto.Description,
		Identifier:  dto.Identifier,
		Status:      dto.Status,
		IsSystem:    dto.IsSystem,
		Page:        dto.Page,
		Limit:       dto.Limit,
		SortBy:      dto.SortBy,
		SortOrder:   dto.SortOrder,
	})
	if err != nil {
		return nil, apperror.ToGRPCError(err)
	}

	rows := make([]*authv1.Tenant, len(result.Data))
	for i := range result.Data {
		rows[i] = tenantProto(&result.Data[i])
	}
	return &authv1.ListTenantsResponse{Tenants: rows, Page: tenantPageProto(result.Total, result.Page, result.Limit, result.TotalPages)}, nil
}

func (h *TenantGRPCHandler) GetTenant(ctx context.Context, req *authv1.GetTenantRequest) (*authv1.GetTenantResponse, error) {
	tenantUUID, err := parseGRPCUUID(req.GetTenantUuid(), "Tenant UUID")
	if err != nil {
		return nil, err
	}
	tenant, err := h.tenantService.GetByUUID(ctx, tenantUUID)
	if err != nil {
		return nil, apperror.ToGRPCError(err)
	}
	return &authv1.GetTenantResponse{Tenant: tenantProto(tenant)}, nil
}

func (h *TenantGRPCHandler) CreateTenant(ctx context.Context, req *authv1.TenantServiceCreateTenantRequest) (*authv1.TenantServiceCreateTenantResponse, error) {
	dto := TenantCreateRequestDTO{
		Name:        req.GetName(),
		DisplayName: req.GetDisplayName(),
		Description: req.GetDescription(),
		Status:      req.GetStatus(),
	}
	if err := dto.Validate(); err != nil {
		return nil, apperror.ToGRPCError(apperror.NewValidation(err.Error()))
	}
	tenant, err := h.tenantService.Create(ctx, dto.Name, dto.DisplayName, dto.Description, dto.Status)
	if err != nil {
		return nil, apperror.ToGRPCError(err)
	}
	return &authv1.TenantServiceCreateTenantResponse{Tenant: tenantProto(tenant)}, nil
}

func (h *TenantGRPCHandler) UpdateTenant(ctx context.Context, req *authv1.TenantServiceUpdateTenantRequest) (*authv1.TenantServiceUpdateTenantResponse, error) {
	tenantUUID, err := parseGRPCUUID(req.GetTenantUuid(), "Tenant UUID")
	if err != nil {
		return nil, err
	}
	dto := TenantUpdateRequestDTO{
		Name:        req.GetName(),
		DisplayName: req.GetDisplayName(),
		Description: req.GetDescription(),
		Status:      req.GetStatus(),
	}
	if err := dto.Validate(); err != nil {
		return nil, apperror.ToGRPCError(apperror.NewValidation(err.Error()))
	}
	tenant, err := h.tenantService.Update(ctx, tenantUUID, dto.Name, dto.DisplayName, dto.Description, dto.Status)
	if err != nil {
		return nil, apperror.ToGRPCError(err)
	}
	return &authv1.TenantServiceUpdateTenantResponse{Tenant: tenantProto(tenant)}, nil
}

func (h *TenantGRPCHandler) SetTenantStatus(ctx context.Context, req *authv1.SetTenantStatusRequest) (*authv1.SetTenantStatusResponse, error) {
	tenantUUID, err := parseGRPCUUID(req.GetTenantUuid(), "Tenant UUID")
	if err != nil {
		return nil, err
	}
	dto := TenantSetStatusRequestDTO{Status: req.GetStatus()}
	if err := dto.Validate(); err != nil {
		return nil, apperror.ToGRPCError(apperror.NewValidation(err.Error()))
	}
	tenant, err := h.tenantService.SetStatusByUUID(ctx, tenantUUID, dto.Status)
	if err != nil {
		return nil, apperror.ToGRPCError(err)
	}
	return &authv1.SetTenantStatusResponse{Tenant: tenantProto(tenant)}, nil
}

func (h *TenantGRPCHandler) DeleteTenant(ctx context.Context, req *authv1.DeleteTenantRequest) (*authv1.DeleteTenantResponse, error) {
	tenantUUID, err := parseGRPCUUID(req.GetTenantUuid(), "Tenant UUID")
	if err != nil {
		return nil, err
	}
	actorUserID, err := h.resolveActorUserID(ctx, req.GetActorUserUuid())
	if err != nil {
		return nil, err
	}
	tenant, err := h.tenantService.DeleteByUUID(ctx, tenantUUID, actorUserID)
	if err != nil {
		return nil, apperror.ToGRPCError(err)
	}
	return &authv1.DeleteTenantResponse{Tenant: tenantProto(tenant)}, nil
}

func (h *TenantGRPCHandler) ListTenantMembers(ctx context.Context, req *authv1.ListTenantMembersRequest) (*authv1.ListTenantMembersResponse, error) {
	tenant, err := h.resolveTenant(ctx, req.GetTenantUuid())
	if err != nil {
		return nil, err
	}
	dto := TenantMemberFilterDTO{
		Role:                 optionalString(req.GetRole()),
		PaginationRequestDTO: paginationDTO(req.GetPagination()),
	}
	if err := dto.Validate(); err != nil {
		return nil, apperror.ToGRPCError(apperror.NewValidation(err.Error()))
	}
	members, err := h.tenantMemberService.ListByTenant(ctx, TenantMemberServiceListFilter{
		TenantID:  tenant.TenantID,
		Role:      dto.Role,
		Page:      dto.Page,
		Limit:     dto.Limit,
		SortBy:    dto.SortBy,
		SortOrder: dto.SortOrder,
	})
	if err != nil {
		return nil, apperror.ToGRPCError(err)
	}
	rows := make([]*authv1.TenantMember, len(members.Data))
	for i := range members.Data {
		rows[i] = tenantMemberProto(&members.Data[i])
	}
	return &authv1.ListTenantMembersResponse{Members: rows, Page: tenantPageProto(members.Total, members.Page, members.Limit, members.TotalPages)}, nil
}

func (h *TenantGRPCHandler) AddTenantMember(ctx context.Context, req *authv1.AddTenantMemberRequest) (*authv1.AddTenantMemberResponse, error) {
	tenant, err := h.resolveTenant(ctx, req.GetTenantUuid())
	if err != nil {
		return nil, err
	}
	userUUID, err := parseGRPCUUID(req.GetUserUuid(), "User UUID")
	if err != nil {
		return nil, err
	}
	dto := TenantMemberAddMemberRequestDTO{UserUUID: userUUID, Role: req.GetRole()}
	if err := dto.Validate(); err != nil {
		return nil, apperror.ToGRPCError(apperror.NewValidation(err.Error()))
	}
	actorUserID, err := h.resolveActorUserID(ctx, req.GetActorUserUuid())
	if err != nil {
		return nil, err
	}
	member, err := h.tenantMemberService.CreateByUserUUID(ctx, tenant.TenantID, dto.UserUUID, dto.Role, actorUserID)
	if err != nil {
		return nil, apperror.ToGRPCError(err)
	}
	return &authv1.AddTenantMemberResponse{Member: tenantMemberProto(member)}, nil
}

func (h *TenantGRPCHandler) UpdateTenantMemberRole(ctx context.Context, req *authv1.UpdateTenantMemberRoleRequest) (*authv1.UpdateTenantMemberRoleResponse, error) {
	tenant, err := h.resolveTenant(ctx, req.GetTenantUuid())
	if err != nil {
		return nil, err
	}
	memberUUID, err := parseGRPCUUID(req.GetTenantMemberUuid(), "Tenant member UUID")
	if err != nil {
		return nil, err
	}
	dto := TenantMemberUpdateRoleRequestDTO{Role: req.GetRole()}
	if err := dto.Validate(); err != nil {
		return nil, apperror.ToGRPCError(apperror.NewValidation(err.Error()))
	}
	actorUserID, err := h.resolveActorUserID(ctx, req.GetActorUserUuid())
	if err != nil {
		return nil, err
	}
	member, err := h.tenantMemberService.UpdateRole(ctx, tenant.TenantID, memberUUID, dto.Role, actorUserID)
	if err != nil {
		return nil, apperror.ToGRPCError(err)
	}
	return &authv1.UpdateTenantMemberRoleResponse{Member: tenantMemberProto(member)}, nil
}

func (h *TenantGRPCHandler) RemoveTenantMember(ctx context.Context, req *authv1.RemoveTenantMemberRequest) (*authv1.RemoveTenantMemberResponse, error) {
	tenant, err := h.resolveTenant(ctx, req.GetTenantUuid())
	if err != nil {
		return nil, err
	}
	memberUUID, err := parseGRPCUUID(req.GetTenantMemberUuid(), "Tenant member UUID")
	if err != nil {
		return nil, err
	}
	actorUserID, err := h.resolveActorUserID(ctx, req.GetActorUserUuid())
	if err != nil {
		return nil, err
	}
	if err := h.tenantMemberService.DeleteByUUID(ctx, tenant.TenantID, memberUUID, actorUserID); err != nil {
		return nil, apperror.ToGRPCError(err)
	}
	return &authv1.RemoveTenantMemberResponse{Removed: true}, nil
}

func (h *TenantGRPCHandler) resolveActorUserID(ctx context.Context, raw string) (int64, error) {
	actorUUID, err := parseGRPCUUID(raw, "Actor user UUID")
	if err != nil {
		return 0, err
	}
	if h.tenantMemberService == nil {
		return 0, apperror.ToGRPCError(apperror.NewInternal("tenant member service is unavailable", nil))
	}
	actorUserID, err := h.tenantMemberService.ResolveUserID(ctx, actorUUID)
	if err != nil {
		return 0, apperror.ToGRPCError(err)
	}
	return actorUserID, nil
}

func (h *TenantGRPCHandler) resolveTenant(ctx context.Context, tenantUUID string) (*TenantServiceDataResult, error) {
	parsed, err := parseGRPCUUID(tenantUUID, "Tenant UUID")
	if err != nil {
		return nil, err
	}
	tenant, err := h.tenantService.GetByUUID(ctx, parsed)
	if err != nil {
		return nil, apperror.ToGRPCError(err)
	}
	return tenant, nil
}

func parseGRPCUUID(value string, label string) (uuid.UUID, error) {
	if value == "" {
		return uuid.Nil, apperror.ToGRPCError(apperror.NewValidation(label + " is required"))
	}
	parsed, err := uuid.Parse(value)
	if err != nil {
		return uuid.Nil, apperror.ToGRPCError(apperror.NewValidation("Invalid " + label))
	}
	return parsed, nil
}

func paginationDTO(req *authv1.Pagination) PaginationRequestDTO {
	if req == nil {
		return PaginationRequestDTO{Page: 1, Limit: pagination.DefaultPageSize}
	}
	page := int(req.GetPage())
	limit := int(req.GetLimit())
	if page == 0 {
		page = 1
	}
	if limit == 0 {
		limit = pagination.DefaultPageSize
	}
	return PaginationRequestDTO{Page: page, Limit: limit, SortBy: req.GetSortBy(), SortOrder: req.GetSortOrder()}
}

func tenantPageProto(total int64, page int, limit int, totalPages int) *authv1.PageMetadata {
	return &authv1.PageMetadata{Total: total, Page: int32(page), Limit: int32(limit), TotalPages: int32(totalPages)}
}

func tenantProto(tenant *TenantServiceDataResult) *authv1.Tenant {
	if tenant == nil {
		return nil
	}
	return &authv1.Tenant{
		TenantUuid:  tenant.TenantUUID.String(),
		Name:        tenant.Name,
		DisplayName: tenant.DisplayName,
		Description: tenant.Description,
		Identifier:  tenant.Identifier,
		Status:      tenant.Status,
		IsSystem:    tenant.IsSystem,
		Metadata:    jsonStruct(tenant.Metadata),
		CreatedAt:   timestamppb.New(tenant.CreatedAt),
		UpdatedAt:   timestamppb.New(tenant.UpdatedAt),
	}
}

func tenantMemberProto(member *TenantMemberServiceDataResult) *authv1.TenantMember {
	if member == nil {
		return nil
	}
	return &authv1.TenantMember{
		TenantMemberUuid: member.TenantMemberUUID.String(),
		Role:             member.Role,
		User:             tenantMemberUserProto(member.User),
		CreatedAt:        timestamppb.New(member.CreatedAt),
		UpdatedAt:        timestamppb.New(member.UpdatedAt),
	}
}

func tenantMemberUserProto(user *MemberUser) *authv1.TenantMemberUser {
	if user == nil {
		return nil
	}
	return &authv1.TenantMemberUser{
		UserUuid:           user.UserUUID.String(),
		Username:           user.Username,
		Fullname:           user.Fullname,
		Email:              user.Email,
		Phone:              user.Phone,
		IsEmailVerified:    user.IsEmailVerified,
		IsPhoneVerified:    user.IsPhoneVerified,
		Status:             user.Status,
		Metadata:           jsonStruct(user.Metadata),
		CreatedAt:          timestamppb.New(user.CreatedAt),
		UpdatedAt:          timestamppb.New(user.UpdatedAt),
	}
}

func jsonStruct(raw []byte) *structpb.Struct {
	if len(raw) == 0 {
		return nil
	}
	var values map[string]any
	if err := json.Unmarshal(raw, &values); err != nil {
		return nil
	}
	result, _ := structpb.NewStruct(values)
	return result
}

func optionalString(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}

func optionalBool(value *bool) *bool {
	return value
}
