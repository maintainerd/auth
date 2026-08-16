package federation

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/maintainerd/maintainerd-auth/internal/platform/apperror"
	authv1 "github.com/maintainerd/maintainerd-auth/internal/platform/gen/go/maintainerd/auth"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// TenantResolver turns the request's tenant UUID into the internal id the
// service scopes by.
type TenantResolver interface {
	GetByUUID(ctx context.Context, tenantUUID uuid.UUID) (int64, error)
}

// WorkloadIdentityFederationGRPCHandler exposes federation management to machine
// callers.
//
// The REST twin of this surface is unreachable to them: its permission
// middleware requires a resolved USER principal, and an orchestrator has none.
// Without this handler an orchestrator can provision a workload but can never
// give it a platform-attested identity, so every workload it creates would need
// a secret injected — the outcome workload identity exists to avoid.
//
// Authorization is not repeated here. The interceptor gates each method on its
// workload-identity-federation:* permission before the handler runs, and the
// service scopes every query to the resolved tenant, so a caller cannot name a
// federation outside its own tenant.
type WorkloadIdentityFederationGRPCHandler struct {
	authv1.UnimplementedWorkloadIdentityFederationServiceServer
	tenantResolver TenantResolver
	service        WorkloadIdentityFederationService
}

func NewWorkloadIdentityFederationGRPCHandler(tenantResolver TenantResolver, service WorkloadIdentityFederationService) *WorkloadIdentityFederationGRPCHandler {
	return &WorkloadIdentityFederationGRPCHandler{tenantResolver: tenantResolver, service: service}
}

func (h *WorkloadIdentityFederationGRPCHandler) ListWorkloadIdentityFederations(ctx context.Context, req *authv1.ListWorkloadIdentityFederationsRequest) (*authv1.ListWorkloadIdentityFederationsResponse, error) {
	tenantID, err := h.resolveTenant(ctx, req.GetTenantId())
	if err != nil {
		return nil, err
	}
	filter := WorkloadIdentityFederationListFilter{IsActive: req.IsActive}
	if name := req.GetName(); name != "" {
		filter.Name = &name
	}
	if p := req.GetPagination(); p != nil {
		filter.Page = int(p.GetPage())
		filter.Limit = int(p.GetLimit())
		filter.SortBy = p.GetSortBy()
		filter.SortOrder = p.GetSortOrder()
	}

	result, err := h.service.GetAll(ctx, tenantID, filter)
	if err != nil {
		return nil, apperror.ToGRPCError(err)
	}
	out := make([]*authv1.WorkloadIdentityFederation, 0, len(result.Data))
	for i := range result.Data {
		out = append(out, federationProto(&result.Data[i]))
	}
	return &authv1.ListWorkloadIdentityFederationsResponse{
		Federations: out,
		Page: &authv1.PageMetadata{
			Total:      result.Total,
			Page:       int32(result.Page),
			Limit:      int32(result.Limit),
			TotalPages: int32(result.TotalPages),
		},
	}, nil
}

func (h *WorkloadIdentityFederationGRPCHandler) GetWorkloadIdentityFederation(ctx context.Context, req *authv1.GetWorkloadIdentityFederationRequest) (*authv1.GetWorkloadIdentityFederationResponse, error) {
	tenantID, err := h.resolveTenant(ctx, req.GetTenantId())
	if err != nil {
		return nil, err
	}
	federationUUID, err := parseFederationUUID(req.GetWorkloadIdentityFederationId())
	if err != nil {
		return nil, err
	}
	result, err := h.service.GetByUUID(ctx, tenantID, federationUUID)
	if err != nil {
		return nil, apperror.ToGRPCError(err)
	}
	return &authv1.GetWorkloadIdentityFederationResponse{Federation: federationProto(result)}, nil
}

func (h *WorkloadIdentityFederationGRPCHandler) CreateWorkloadIdentityFederation(ctx context.Context, req *authv1.CreateWorkloadIdentityFederationRequest) (*authv1.CreateWorkloadIdentityFederationResponse, error) {
	tenantID, err := h.resolveTenant(ctx, req.GetTenantId())
	if err != nil {
		return nil, err
	}
	clientUUID, err := uuid.Parse(req.GetClientId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "client_uuid is not a valid UUID")
	}

	// Validation — https issuer, anchored subject pattern, required audience —
	// lives in the service and is shared with the REST path. Duplicating it here
	// would give the two transports two chances to disagree about what a safe
	// trust rule is.
	result, err := h.service.Create(ctx, tenantID, WorkloadIdentityFederationCreateInput{
		ClientUUID:       clientUUID,
		Name:             req.GetName(),
		Description:      req.GetDescription(),
		IssuerURL:        req.GetIssuerUrl(),
		Audience:         req.GetAudience(),
		SubjectClaim:     req.GetSubjectClaim(),
		SubjectPattern:   req.GetSubjectPattern(),
		AllowedScopes:    req.GetAllowedScopes(),
		AttributeMapping: req.GetAttributeMapping(),
		IsActive:         req.IsActive,
	})
	if err != nil {
		return nil, apperror.ToGRPCError(err)
	}
	return &authv1.CreateWorkloadIdentityFederationResponse{Federation: federationProto(result)}, nil
}

func (h *WorkloadIdentityFederationGRPCHandler) UpdateWorkloadIdentityFederation(ctx context.Context, req *authv1.UpdateWorkloadIdentityFederationRequest) (*authv1.UpdateWorkloadIdentityFederationResponse, error) {
	tenantID, err := h.resolveTenant(ctx, req.GetTenantId())
	if err != nil {
		return nil, err
	}
	federationUUID, err := parseFederationUUID(req.GetWorkloadIdentityFederationId())
	if err != nil {
		return nil, err
	}
	result, err := h.service.Update(ctx, tenantID, federationUUID, WorkloadIdentityFederationUpdateInput{
		Name:             req.GetName(),
		Description:      req.GetDescription(),
		IssuerURL:        req.GetIssuerUrl(),
		Audience:         req.GetAudience(),
		SubjectClaim:     req.GetSubjectClaim(),
		SubjectPattern:   req.GetSubjectPattern(),
		AllowedScopes:    req.GetAllowedScopes(),
		AttributeMapping: req.GetAttributeMapping(),
		IsActive:         req.IsActive,
	})
	if err != nil {
		return nil, apperror.ToGRPCError(err)
	}
	return &authv1.UpdateWorkloadIdentityFederationResponse{Federation: federationProto(result)}, nil
}

func (h *WorkloadIdentityFederationGRPCHandler) DeleteWorkloadIdentityFederation(ctx context.Context, req *authv1.DeleteWorkloadIdentityFederationRequest) (*authv1.DeleteWorkloadIdentityFederationResponse, error) {
	tenantID, err := h.resolveTenant(ctx, req.GetTenantId())
	if err != nil {
		return nil, err
	}
	federationUUID, err := parseFederationUUID(req.GetWorkloadIdentityFederationId())
	if err != nil {
		return nil, err
	}
	result, err := h.service.Delete(ctx, tenantID, federationUUID)
	if err != nil {
		return nil, apperror.ToGRPCError(err)
	}
	return &authv1.DeleteWorkloadIdentityFederationResponse{Federation: federationProto(result)}, nil
}

func (h *WorkloadIdentityFederationGRPCHandler) resolveTenant(ctx context.Context, tenantUUID string) (int64, error) {
	parsed, err := uuid.Parse(tenantUUID)
	if err != nil {
		return 0, status.Error(codes.InvalidArgument, "tenant_uuid is not a valid UUID")
	}
	tenantID, err := h.tenantResolver.GetByUUID(ctx, parsed)
	if err != nil {
		return 0, apperror.ToGRPCError(err)
	}
	return tenantID, nil
}

func parseFederationUUID(value string) (uuid.UUID, error) {
	parsed, err := uuid.Parse(value)
	if err != nil {
		return uuid.Nil, status.Error(codes.InvalidArgument, "workload_identity_federation_uuid is not a valid UUID")
	}
	return parsed, nil
}

func federationProto(in *WorkloadIdentityFederationServiceDataResult) *authv1.WorkloadIdentityFederation {
	if in == nil {
		return nil
	}
	return &authv1.WorkloadIdentityFederation{
		WorkloadIdentityFederationId: in.WorkloadIdentityFederationUUID.String(),
		ClientId:                     in.ClientUUID.String(),
		Name:                         in.Name,
		Description:                  in.Description,
		IssuerUrl:                    in.IssuerURL,
		Audience:                     in.Audience,
		SubjectClaim:                 in.SubjectClaim,
		SubjectPattern:               in.SubjectPattern,
		AllowedScopes:                in.AllowedScopes,
		AttributeMapping:             in.AttributeMapping,
		IsActive:                     in.IsActive,
		CreatedAt:                    in.CreatedAt.UTC().Format(time.RFC3339),
		UpdatedAt:                    in.UpdatedAt.UTC().Format(time.RFC3339),
	}
}
