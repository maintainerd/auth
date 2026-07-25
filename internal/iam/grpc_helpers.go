package iam

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"
	"github.com/maintainerd/maintainerd-auth/internal/platform/apperror"
	authv1 "github.com/maintainerd/maintainerd-auth/internal/platform/gen/go/maintainerd/auth"
	"github.com/maintainerd/maintainerd-auth/internal/platform/middleware"
	"github.com/maintainerd/maintainerd-auth/internal/platform/pagination"
	"github.com/maintainerd/maintainerd-auth/internal/tenant"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"
	"gorm.io/datatypes"
)

type TenantResolver interface {
	GetByUUID(ctx context.Context, tenantUUID uuid.UUID) (*tenant.TenantServiceDataResult, error)
}

type tenantScope struct {
	TenantID   int64
	TenantUUID uuid.UUID
}

func iamParseUUID(value string, label string) (uuid.UUID, error) {
	if value == "" {
		return uuid.Nil, apperror.ToGRPCError(apperror.NewValidation(label + " is required"))
	}
	parsed, err := uuid.Parse(value)
	if err != nil {
		return uuid.Nil, apperror.ToGRPCError(apperror.NewValidation("Invalid " + label))
	}
	return parsed, nil
}

func iamPaginationDTO(req *authv1.Pagination) PaginationRequestDTO {
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

func iamPageProto(total int64, page int, limit int, totalPages int) *authv1.PageMetadata {
	return &authv1.PageMetadata{Total: total, Page: int32(page), Limit: int32(limit), TotalPages: int32(totalPages)}
}

func iamOptionalString(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}

// resolveIAMTenant resolves the target tenant from the request AND checks the caller
// is allowed to act on it.
//
// The gRPC surface takes the target tenant from the request body, and the interceptor
// authorizes an ACTION only — it never compares the requested tenant against the
// token. Existence was therefore the only check: any principal holding, say,
// `service:update` in its own tenant could pass another tenant's UUID and mutate
// that tenant's services.
//
// The rule: a caller may act on its own tenant, and a caller whose token is bound to
// the SYSTEM tenant may act on any tenant. The latter is what lets the control plane
// configure a tenant remotely without a human in this app's frontend; it is not a
// blanket grant, because a tenant principal is now pinned to its own tenant.
func resolveIAMTenant(ctx context.Context, tenantService TenantResolver, tenantUUID string) (*tenantScope, error) {
	parsed, err := iamParseUUID(tenantUUID, "Tenant UUID")
	if err != nil {
		return nil, err
	}
	result, err := tenantService.GetByUUID(ctx, parsed)
	if err != nil {
		return nil, apperror.ToGRPCError(err)
	}
	if err := assertCallerMayActOnTenant(ctx, tenantService, result.TenantID); err != nil {
		return nil, err
	}
	return &tenantScope{TenantID: result.TenantID, TenantUUID: result.TenantUUID}, nil
}

// assertCallerMayActOnTenant enforces the tenant boundary described above.
func assertCallerMayActOnTenant(ctx context.Context, tenantService TenantResolver, targetTenantID int64) error {
	claims := middleware.JWTClaimsFromContext(ctx)
	if claims == nil || claims.TenantID == 0 {
		// A token with no tenant cannot prove it may act anywhere. The interceptor
		// already refuses these for permission-gated methods; this is defence in depth.
		return status.Error(codes.PermissionDenied, "this token is not bound to a tenant")
	}
	if claims.TenantID == targetTenantID {
		return nil
	}
	// Cross-tenant is reserved for the control plane, which lives in the system tenant.
	callerIsSystem, err := callerTenantIsSystem(ctx, tenantService, claims.TenantID)
	if err != nil {
		return err
	}
	if !callerIsSystem {
		return status.Error(codes.PermissionDenied,
			"this token may only act on its own tenant")
	}
	return nil
}

// callerTenantIsSystem reports whether the caller's own tenant is the system tenant.
//
// Resolved through GetSystem rather than by loading the caller's tenant, so the
// check needs no new repository method and compares ids directly.
func callerTenantIsSystem(ctx context.Context, tenantService TenantResolver, callerTenantID int64) (bool, error) {
	resolver, ok := tenantService.(interface {
		GetSystem(ctx context.Context) (*tenant.TenantServiceDataResult, error)
	})
	if !ok {
		// Fail closed: without a way to identify the system tenant, cross-tenant
		// access cannot be justified.
		return false, status.Error(codes.PermissionDenied, "cross-tenant access cannot be verified")
	}
	systemTenant, err := resolver.GetSystem(ctx)
	if err != nil || systemTenant == nil {
		return false, status.Error(codes.PermissionDenied, "cross-tenant access cannot be verified")
	}
	return systemTenant.TenantID == callerTenantID, nil
}

func resolveIAMTenantAndUUID(ctx context.Context, tenantService TenantResolver, tenantUUID string, value string, label string) (*tenantScope, uuid.UUID, error) {
	scope, err := resolveIAMTenant(ctx, tenantService, tenantUUID)
	if err != nil {
		return nil, uuid.Nil, err
	}
	parsed, err := iamParseUUID(value, label)
	if err != nil {
		return nil, uuid.Nil, err
	}
	return scope, parsed, nil
}

func resolveIAMTenantAndActor(ctx context.Context, tenantService TenantResolver, tenantUUID string, actorValue string) (*tenantScope, uuid.UUID, error) {
	scope, err := resolveIAMTenant(ctx, tenantService, tenantUUID)
	if err != nil {
		return nil, uuid.Nil, err
	}
	actor, err := iamParseUUID(actorValue, "Actor user UUID")
	if err != nil {
		return nil, uuid.Nil, err
	}
	return scope, actor, nil
}

func resolveIAMTenantRoleActor(ctx context.Context, tenantService TenantResolver, tenantUUID string, roleValue string, actorValue string) (*tenantScope, uuid.UUID, uuid.UUID, error) {
	scope, roleUUID, err := resolveIAMTenantAndUUID(ctx, tenantService, tenantUUID, roleValue, "Role UUID")
	if err != nil {
		return nil, uuid.Nil, uuid.Nil, err
	}
	actor, err := iamParseUUID(actorValue, "Actor user UUID")
	if err != nil {
		return nil, uuid.Nil, uuid.Nil, err
	}
	return scope, roleUUID, actor, nil
}

func parseIAMUUIDs(values []string, label string) ([]uuid.UUID, error) {
	result := make([]uuid.UUID, len(values))
	for i, value := range values {
		parsed, err := iamParseUUID(value, label)
		if err != nil {
			return nil, err
		}
		result[i] = parsed
	}
	return result, nil
}

func serviceProto(result *ServiceServiceDataResult) *authv1.Service {
	if result == nil {
		return nil
	}
	return &authv1.Service{
		ServiceUuid: result.ServiceUUID.String(),
		Name:        result.Name,
		DisplayName: result.DisplayName,
		Description: result.Description,
		Version:     result.Version,
		Status:      result.Status,
		IsSystem:    result.IsSystem,
		ApiCount:    result.APICount,
		PolicyCount: result.PolicyCount,
		CreatedAt:   timestamppb.New(result.CreatedAt),
		UpdatedAt:   timestamppb.New(result.UpdatedAt),
	}
}

func apiProto(result *APIServiceDataResult) *authv1.API {
	if result == nil {
		return nil
	}
	return &authv1.API{
		ApiUuid:     result.APIUUID.String(),
		Name:        result.Name,
		DisplayName: result.DisplayName,
		Description: result.Description,
		Identifier:  result.Identifier,
		Status:      result.Status,
		IsSystem:    result.IsSystem,
		Service:     serviceProto(result.Service),
		CreatedAt:   timestamppb.New(result.CreatedAt),
		UpdatedAt:   timestamppb.New(result.UpdatedAt),
	}
}

func permissionProto(result *PermissionServiceDataResult) *authv1.Permission {
	if result == nil {
		return nil
	}
	return &authv1.Permission{
		PermissionUuid: result.PermissionUUID.String(),
		Name:           result.Name,
		Description:    result.Description,
		Api:            apiProto(result.API),
		Status:         result.Status,
		IsSystem:       result.IsSystem,
		CreatedAt:      timestamppb.New(result.CreatedAt),
		UpdatedAt:      timestamppb.New(result.UpdatedAt),
	}
}

func policyProto(result *PolicyServiceDataResult) *authv1.Policy {
	if result == nil {
		return nil
	}
	return &authv1.Policy{
		PolicyUuid:  result.PolicyUUID.String(),
		Name:        result.Name,
		Description: result.Description,
		Document:    policyDocumentProto(result.Document),
		Version:     result.Version,
		Status:      result.Status,
		IsSystem:    result.IsSystem,
		CreatedAt:   timestamppb.New(result.CreatedAt),
		UpdatedAt:   timestamppb.New(result.UpdatedAt),
	}
}

func policyServiceProto(result *PolicyServiceServiceDataResult) *authv1.Service {
	if result == nil {
		return nil
	}
	return &authv1.Service{
		ServiceUuid: result.ServiceUUID.String(),
		Name:        result.Name,
		DisplayName: result.DisplayName,
		Description: result.Description,
		Version:     result.Version,
		Status:      result.Status,
		IsSystem:    result.IsSystem,
		ApiCount:    result.APICount,
		PolicyCount: result.PolicyCount,
		CreatedAt:   timestamppb.New(result.CreatedAt),
		UpdatedAt:   timestamppb.New(result.UpdatedAt),
	}
}

func roleProto(result *RoleServiceDataResult) *authv1.Role {
	if result == nil {
		return nil
	}
	permissions := make([]*authv1.Permission, 0)
	if result.Permissions != nil {
		permissions = make([]*authv1.Permission, len(*result.Permissions))
		for i := range *result.Permissions {
			permissions[i] = permissionProto(&(*result.Permissions)[i])
		}
	}
	return &authv1.Role{
		RoleUuid:    result.RoleUUID.String(),
		Name:        result.Name,
		Description: result.Description,
		Permissions: permissions,
		IsDefault:   result.IsDefault,
		IsSystem:    result.IsSystem,
		Status:      result.Status,
		CreatedAt:   timestamppb.New(result.CreatedAt),
		UpdatedAt:   timestamppb.New(result.UpdatedAt),
	}
}

func policyDocumentJSON(document *structpb.Struct) (datatypes.JSON, error) {
	if document == nil {
		return nil, apperror.ToGRPCError(apperror.NewValidation("Policy document is required"))
	}
	payload, _ := json.Marshal(document.AsMap())
	return datatypes.JSON(payload), nil
}

func policyDocumentProto(document datatypes.JSON) *structpb.Struct {
	if len(document) == 0 {
		return &structpb.Struct{Fields: map[string]*structpb.Value{}}
	}
	var raw map[string]any
	if err := json.Unmarshal(document, &raw); err != nil {
		return &structpb.Struct{Fields: map[string]*structpb.Value{}}
	}
	result, _ := structpb.NewStruct(raw)
	return result
}
