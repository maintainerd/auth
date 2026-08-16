package tenant

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"
	"github.com/maintainerd/maintainerd-auth/internal/platform/apperror"
	authv1 "github.com/maintainerd/maintainerd-auth/internal/platform/gen/go/maintainerd/auth"
	"github.com/maintainerd/maintainerd-auth/internal/platform/middleware"
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

// requireSystemTenantActor authorizes an operation that only the system tenant
// may perform (creating tenants). The caller's tenant binding comes from the
// issuer-stamped, unforgeable `tenant_id` claim, so a regular-tenant principal —
// user OR service (e.g. the control plane bound to a regular tenant) — is
// rejected. System-tenant service principals (the provisioning control plane)
// are allowed, mirroring rule #10.
func (h *TenantGRPCHandler) requireSystemTenantActor(ctx context.Context) error {
	return grpcRequireSystemTenantActor(ctx, h.tenantService)
}

func (h *TenantGRPCHandler) authorizeTenantManagement(ctx context.Context, tenantUUID uuid.UUID) error {
	return grpcAuthorizeTenantManagement(ctx, h.tenantService, h.tenantMemberService, tenantUUID)
}

// grpcRequireSystemTenantActor authorizes an operation only the system tenant may
// perform (creating tenants). The caller's tenant binding comes from the
// issuer-stamped, unforgeable `tenant_id` claim, so a regular-tenant principal —
// user OR service (e.g. the control plane bound to a regular tenant) — is
// rejected. System-tenant service principals (the provisioning control plane)
// are allowed.
func grpcRequireSystemTenantActor(ctx context.Context, ts TenantService) error {
	claims := middleware.JWTClaimsFromContext(ctx)
	if claims == nil || claims.TenantID == 0 {
		return apperror.ToGRPCError(apperror.NewUnauthorized("authenticated actor is required"))
	}
	systemTenant, err := ts.GetSystem(ctx)
	if err != nil {
		return apperror.ToGRPCError(err)
	}
	if systemTenant == nil || claims.TenantID != systemTenant.TenantID {
		return apperror.ToGRPCError(apperror.NewForbidden("only members of the system tenant can create tenants"))
	}
	return nil
}

// grpcAuthorizeTenantManagement authorizes a tenant-management operation (info,
// status, or settings) on the target tenant. A system-tenant principal (user or
// the control-plane service) may manage any tenant; otherwise the caller must be
// a user who is a member of the target tenant (CanManageTenant). Enforces that a
// principal bound to tenant A cannot mutate tenant B. Shared by the tenant and
// tenant-setting gRPC handlers so the boundary check lives in exactly one place.
func grpcAuthorizeTenantManagement(ctx context.Context, ts TenantService, ms TenantMemberService, tenantUUID uuid.UUID) error {
	claims := middleware.JWTClaimsFromContext(ctx)
	if claims == nil || claims.TenantID == 0 {
		return apperror.ToGRPCError(apperror.NewUnauthorized("authenticated actor is required"))
	}
	systemTenant, err := ts.GetSystem(ctx)
	if err != nil {
		return apperror.ToGRPCError(err)
	}
	if systemTenant != nil && claims.TenantID == systemTenant.TenantID {
		return nil
	}
	if claims.UserUUID == uuid.Nil {
		return apperror.ToGRPCError(apperror.NewForbidden("only system-tenant principals or tenant members can manage this tenant"))
	}
	if ms == nil {
		return apperror.ToGRPCError(apperror.NewInternal("tenant member service is unavailable", nil))
	}
	actorUserID, err := ms.ResolveUserID(ctx, claims.UserUUID)
	if err != nil {
		return apperror.ToGRPCError(err)
	}
	canManage, err := ms.CanManageTenant(ctx, actorUserID, tenantUUID)
	if err != nil {
		return apperror.ToGRPCError(err)
	}
	if !canManage {
		return apperror.ToGRPCError(apperror.NewForbidden("you do not have access to manage this tenant"))
	}
	return nil
}

// grpcCallerTenantScope resolves the caller's tenant binding from the
// issuer-stamped, unforgeable `tenant_id` claim and reports whether that tenant
// is the system tenant (the only one allowed to read across tenants). Read RPCs
// use it for the scoping the HTTP handlers already do; without it every
// tenant-bound token could enumerate every tenant over gRPC.
func grpcCallerTenantScope(ctx context.Context, ts TenantService) (callerTenantID int64, isSystem bool, err error) {
	claims := middleware.JWTClaimsFromContext(ctx)
	if claims == nil || claims.TenantID == 0 {
		return 0, false, apperror.ToGRPCError(apperror.NewUnauthorized("authenticated actor is required"))
	}
	systemTenant, err := ts.GetSystem(ctx)
	if err != nil {
		return 0, false, apperror.ToGRPCError(err)
	}
	return claims.TenantID, systemTenant != nil && claims.TenantID == systemTenant.TenantID, nil
}

// grpcActorUserID returns the acting user taken from the VERIFIED token, never
// from the request body. A body-supplied actor_user_uuid let a caller name any
// user and have the membership/escalation guards (service_member.go
// authorizeManager) evaluated against THAT user's standing — a tenant-B token
// naming a tenant-A owner was granted tenant-A management.
//
// This gRPC surface authenticates SERVICE principals, which carry no user
// identity, so these operations fail closed rather than running as an
// unattributable actor. Re-enabling them for services means deciding what bounds
// a service principal's grants, not dropping this check.
// grpcActorUserID resolves the acting user for a mutating tenant RPC.
//
// Delegates to the one shared definition so this surface cannot drift from the
// client and iam surfaces again.
func grpcActorUserID(ctx context.Context, operation string) (int64, error) {
	actor, err := middleware.GRPCActor(ctx, operation)
	if err != nil {
		return 0, err
	}
	return actor.UserID, nil
}

func (h *TenantGRPCHandler) GetDefaultTenant(ctx context.Context, _ *authv1.GetDefaultTenantRequest) (*authv1.GetDefaultTenantResponse, error) {
	tenant, err := h.tenantService.GetSystem(ctx)
	if err != nil {
		return nil, apperror.ToGRPCError(err)
	}
	return &authv1.GetDefaultTenantResponse{Tenant: tenantProto(tenant)}, nil
}

func (h *TenantGRPCHandler) ListTenants(ctx context.Context, req *authv1.ListTenantsRequest) (*authv1.ListTenantsResponse, error) {
	dto := TenantFilterDTO{
		Name:                 optionalString(req.GetName()),
		DisplayName:          optionalString(req.GetDisplayName()),
		Description:          optionalString(req.GetDescription()),
		Status:               req.GetStatus(),
		IsSystem:             optionalBool(req.IsSystem),
		PaginationRequestDTO: paginationDTO(req.GetPagination()),
	}
	if err := dto.Validate(); err != nil {
		return nil, apperror.ToGRPCError(apperror.NewValidation(err.Error()))
	}

	callerTenantID, isSystem, err := grpcCallerTenantScope(ctx, h.tenantService)
	if err != nil {
		return nil, err
	}

	filter := TenantServiceGetFilter{
		Name:        dto.Name,
		DisplayName: dto.DisplayName,
		Description: dto.Description,
		Status:      dto.Status,
		IsSystem:    dto.IsSystem,
		Page:        dto.Page,
		Limit:       dto.Limit,
		SortBy:      dto.SortBy,
		SortOrder:   dto.SortOrder,
	}
	// Scope parity with the HTTP handler (handler_tenant.go Get): only
	// system-tenant principals may enumerate every tenant. Unscoped, any
	// tenant-bound token read every tenant record over gRPC.
	if !isSystem {
		filter.TenantIDs = []int64{callerTenantID}
	}

	result, err := h.tenantService.Get(ctx, filter)
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
	tenantUUID, err := parseGRPCUUID(req.GetTenantId(), "Tenant UUID")
	if err != nil {
		return nil, err
	}
	callerTenantID, isSystem, err := grpcCallerTenantScope(ctx, h.tenantService)
	if err != nil {
		return nil, err
	}
	tenant, err := h.tenantService.GetByUUID(ctx, tenantUUID)
	if err != nil {
		return nil, apperror.ToGRPCError(err)
	}
	if tenant == nil {
		return nil, apperror.ToGRPCError(apperror.NewNotFound("tenant"))
	}
	// Scope parity with the HTTP handler (handler_tenant.go GetByUUID): a
	// tenant-bound principal may read only its own tenant record.
	if !isSystem && tenant.TenantID != callerTenantID {
		return nil, apperror.ToGRPCError(apperror.NewForbidden("you can only view your own tenant"))
	}
	return &authv1.GetTenantResponse{Tenant: tenantProto(tenant)}, nil
}

func (h *TenantGRPCHandler) CreateTenant(ctx context.Context, req *authv1.TenantServiceCreateTenantRequest) (*authv1.TenantServiceCreateTenantResponse, error) {
	// Boundary parity with the HTTP handler: only system-tenant principals may
	// create tenants. Without this the gRPC surface let any tenant's principal
	// (every tenant's super-admin is seeded with tenant:create) create tenants,
	// bypassing the rule enforced on HTTP.
	if err := h.requireSystemTenantActor(ctx); err != nil {
		return nil, err
	}
	// Authorization runs BEFORE the ledger claim so a caller that may not create
	// tenants cannot consume — or occupy — a key it has no right to spend.
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
	tenantUUID, err := parseGRPCUUID(req.GetTenantId(), "Tenant UUID")
	if err != nil {
		return nil, err
	}
	// Boundary parity with the HTTP handler: the target tenant must be one the
	// caller may manage. Without this a principal from tenant A could rewrite
	// tenant B's name (the DNS subdomain slug) or status over gRPC.
	if err := h.authorizeTenantManagement(ctx, tenantUUID); err != nil {
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
	tenantUUID, err := parseGRPCUUID(req.GetTenantId(), "Tenant UUID")
	if err != nil {
		return nil, err
	}
	if err := h.authorizeTenantManagement(ctx, tenantUUID); err != nil {
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
	tenantUUID, err := parseGRPCUUID(req.GetTenantId(), "Tenant UUID")
	if err != nil {
		return nil, err
	}
	// Boundary parity with the HTTP handler (handler_tenant.go Delete): only
	// system-tenant principals may delete a tenant. Without this check any
	// tenant's token could delete any other tenant over gRPC.
	if err := h.requireSystemTenantActor(ctx); err != nil {
		return nil, err
	}
	actorUserID, err := grpcActorUserID(ctx, "tenant deletion")
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
	tenant, err := h.resolveManagedTenant(ctx, req.GetTenantId())
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
	tenant, err := h.resolveManagedTenant(ctx, req.GetTenantId())
	if err != nil {
		return nil, err
	}
	userUUID, err := parseGRPCUUID(req.GetUserId(), "User UUID")
	if err != nil {
		return nil, err
	}
	dto := TenantMemberAddMemberRequestDTO{UserUUID: userUUID, Role: req.GetRole()}
	if err := dto.Validate(); err != nil {
		return nil, apperror.ToGRPCError(apperror.NewValidation(err.Error()))
	}
	actorUserID, err := grpcActorUserID(ctx, "tenant member management")
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
	tenant, err := h.resolveManagedTenant(ctx, req.GetTenantId())
	if err != nil {
		return nil, err
	}
	memberUUID, err := parseGRPCUUID(req.GetTenantMemberId(), "Tenant member UUID")
	if err != nil {
		return nil, err
	}
	dto := TenantMemberUpdateRoleRequestDTO{Role: req.GetRole()}
	if err := dto.Validate(); err != nil {
		return nil, apperror.ToGRPCError(apperror.NewValidation(err.Error()))
	}
	actorUserID, err := grpcActorUserID(ctx, "tenant member management")
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
	tenant, err := h.resolveManagedTenant(ctx, req.GetTenantId())
	if err != nil {
		return nil, err
	}
	memberUUID, err := parseGRPCUUID(req.GetTenantMemberId(), "Tenant member UUID")
	if err != nil {
		return nil, err
	}
	actorUserID, err := grpcActorUserID(ctx, "tenant member management")
	if err != nil {
		return nil, err
	}
	if err := h.tenantMemberService.DeleteByUUID(ctx, tenant.TenantID, memberUUID, actorUserID); err != nil {
		return nil, apperror.ToGRPCError(err)
	}
	return &authv1.RemoveTenantMemberResponse{Removed: true}, nil
}

// resolveManagedTenant parses the target tenant UUID, enforces the
// tenant-management boundary, and returns the tenant. Every member RPC goes
// through it: the previous bare lookup did no boundary check at all, so a
// principal bound to tenant A could list, add, re-role, and remove tenant B's
// members — the same gate authorizeTenantManagement already applied to
// UpdateTenant/SetTenantStatus.
func (h *TenantGRPCHandler) resolveManagedTenant(ctx context.Context, tenantUUID string) (*TenantServiceDataResult, error) {
	parsed, err := parseGRPCUUID(tenantUUID, "Tenant UUID")
	if err != nil {
		return nil, err
	}
	if err := h.authorizeTenantManagement(ctx, parsed); err != nil {
		return nil, err
	}
	tenant, err := h.tenantService.GetByUUID(ctx, parsed)
	if err != nil {
		return nil, apperror.ToGRPCError(err)
	}
	if tenant == nil {
		return nil, apperror.ToGRPCError(apperror.NewNotFound("tenant"))
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
		TenantId:    tenant.TenantUUID.String(),
		Name:        tenant.Name,
		DisplayName: tenant.DisplayName,
		Description: tenant.Description,
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
		TenantMemberId: member.TenantMemberUUID.String(),
		Role:           member.Role,
		User:           tenantMemberUserProto(member.User),
		CreatedAt:      timestamppb.New(member.CreatedAt),
		UpdatedAt:      timestamppb.New(member.UpdatedAt),
	}
}

func tenantMemberUserProto(user *MemberUser) *authv1.TenantMemberUser {
	if user == nil {
		return nil
	}
	return &authv1.TenantMemberUser{
		UserId:          user.UserUUID.String(),
		Username:        user.Username,
		Fullname:        user.Fullname,
		Email:           user.Email,
		Phone:           user.Phone,
		IsEmailVerified: user.IsEmailVerified,
		IsPhoneVerified: user.IsPhoneVerified,
		Status:          user.Status,
		Metadata:        jsonStruct(user.Metadata),
		CreatedAt:       timestamppb.New(user.CreatedAt),
		UpdatedAt:       timestamppb.New(user.UpdatedAt),
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
