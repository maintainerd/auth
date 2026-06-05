package iam

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/platform/apperror"
	authv1 "github.com/maintainerd/auth/internal/platform/gen/go/maintainerd/auth"
	"github.com/maintainerd/auth/internal/platform/pagination"
	"github.com/maintainerd/auth/internal/tenant"
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

func resolveIAMTenant(ctx context.Context, tenantService TenantResolver, tenantUUID string) (*tenantScope, error) {
	parsed, err := iamParseUUID(tenantUUID, "Tenant UUID")
	if err != nil {
		return nil, err
	}
	result, err := tenantService.GetByUUID(ctx, parsed)
	if err != nil {
		return nil, apperror.ToGRPCError(err)
	}
	return &tenantScope{TenantID: result.TenantID, TenantUUID: result.TenantUUID}, nil
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
		ApiType:     result.APIType,
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
		IsDefault:      result.IsDefault,
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
