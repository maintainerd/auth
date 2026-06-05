package branding

import (
	"context"

	"github.com/maintainerd/auth/internal/platform/apperror"
	authv1 "github.com/maintainerd/auth/internal/platform/gen/go/maintainerd/auth"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type LoginTemplateGRPCHandler struct {
	authv1.UnimplementedLoginTemplateServiceServer
	tenantResolver TenantResolver
	svc            LoginTemplateService
}

func NewLoginTemplateGRPCHandler(tenantResolver TenantResolver, svc LoginTemplateService) *LoginTemplateGRPCHandler {
	return &LoginTemplateGRPCHandler{tenantResolver: tenantResolver, svc: svc}
}

func (h *LoginTemplateGRPCHandler) ListLoginTemplates(ctx context.Context, req *authv1.ListLoginTemplatesRequest) (*authv1.ListLoginTemplatesResponse, error) {
	tenant, err := h.resolveTenant(ctx, req.GetTenantUuid())
	if err != nil { return nil, err }
	pg := grpcPagination(req.GetPagination())
	result, err := h.svc.GetAll(ctx, tenant.TenantID, optionalStr(req.GetName()), req.GetStatus(), optionalStr(req.GetTemplate()), req.IsDefault, req.IsSystem, pg.Page, pg.Limit, pg.SortBy, pg.SortOrder)
	if err != nil { return nil, apperror.ToGRPCError(err) }
	rows := make([]*authv1.LoginTemplate, len(result.Data))
	for i := range result.Data { rows[i] = loginTemplateProto(&result.Data[i]) }
	return &authv1.ListLoginTemplatesResponse{LoginTemplates: rows, Page: pageProto(result.Total, result.Page, result.Limit, result.TotalPages)}, nil
}

func (h *LoginTemplateGRPCHandler) GetLoginTemplate(ctx context.Context, req *authv1.GetLoginTemplateRequest) (*authv1.GetLoginTemplateResponse, error) {
	tenant, err := h.resolveTenant(ctx, req.GetTenantUuid())
	if err != nil { return nil, err }
	ltUUID, err := parseUUID(req.GetLoginTemplateUuid(), "Login template UUID")
	if err != nil { return nil, err }
	result, err := h.svc.GetByUUID(ctx, ltUUID, tenant.TenantID)
	if err != nil { return nil, apperror.ToGRPCError(err) }
	return &authv1.GetLoginTemplateResponse{LoginTemplate: loginTemplateProto(result)}, nil
}

func (h *LoginTemplateGRPCHandler) CreateLoginTemplate(ctx context.Context, req *authv1.CreateLoginTemplateRequest) (*authv1.CreateLoginTemplateResponse, error) {
	tenant, err := h.resolveTenant(ctx, req.GetTenantUuid())
	if err != nil { return nil, err }
	meta := structMap(req.GetMetadata())
	result, err := h.svc.Create(ctx, tenant.TenantID, req.GetName(), req.Description, req.GetTemplate(), meta, req.GetStatus())
	if err != nil { return nil, apperror.ToGRPCError(err) }
	return &authv1.CreateLoginTemplateResponse{LoginTemplate: loginTemplateProto(result)}, nil
}

func (h *LoginTemplateGRPCHandler) UpdateLoginTemplate(ctx context.Context, req *authv1.UpdateLoginTemplateRequest) (*authv1.UpdateLoginTemplateResponse, error) {
	tenant, err := h.resolveTenant(ctx, req.GetTenantUuid())
	if err != nil { return nil, err }
	ltUUID, err := parseUUID(req.GetLoginTemplateUuid(), "Login template UUID")
	if err != nil { return nil, err }
	meta := structMap(req.GetMetadata())
	result, err := h.svc.Update(ctx, ltUUID, tenant.TenantID, req.GetName(), req.Description, req.GetTemplate(), meta, req.GetStatus())
	if err != nil { return nil, apperror.ToGRPCError(err) }
	return &authv1.UpdateLoginTemplateResponse{LoginTemplate: loginTemplateProto(result)}, nil
}

func (h *LoginTemplateGRPCHandler) SetLoginTemplateStatus(ctx context.Context, req *authv1.SetLoginTemplateStatusRequest) (*authv1.SetLoginTemplateStatusResponse, error) {
	tenant, err := h.resolveTenant(ctx, req.GetTenantUuid())
	if err != nil { return nil, err }
	ltUUID, err := parseUUID(req.GetLoginTemplateUuid(), "Login template UUID")
	if err != nil { return nil, err }
	result, err := h.svc.UpdateStatus(ctx, ltUUID, tenant.TenantID, req.GetStatus())
	if err != nil { return nil, apperror.ToGRPCError(err) }
	return &authv1.SetLoginTemplateStatusResponse{LoginTemplate: loginTemplateProto(result)}, nil
}

func (h *LoginTemplateGRPCHandler) DeleteLoginTemplate(ctx context.Context, req *authv1.DeleteLoginTemplateRequest) (*authv1.DeleteLoginTemplateResponse, error) {
	tenant, err := h.resolveTenant(ctx, req.GetTenantUuid())
	if err != nil { return nil, err }
	ltUUID, err := parseUUID(req.GetLoginTemplateUuid(), "Login template UUID")
	if err != nil { return nil, err }
	result, err := h.svc.Delete(ctx, ltUUID, tenant.TenantID)
	if err != nil { return nil, apperror.ToGRPCError(err) }
	return &authv1.DeleteLoginTemplateResponse{LoginTemplate: loginTemplateProto(result)}, nil
}

func (h *LoginTemplateGRPCHandler) resolveTenant(ctx context.Context, tenantUUID string) (*TenantServiceDataResult, error) {
	parsed, err := parseUUID(tenantUUID, "Tenant UUID")
	if err != nil { return nil, err }
	result, err := h.tenantResolver.GetByUUID(ctx, parsed)
	if err != nil { return nil, apperror.ToGRPCError(err) }
	return result, nil
}

func structMap(s *structpb.Struct) map[string]any {
	if s == nil { return nil }
	return s.AsMap()
}

func loginTemplateProto(r *LoginTemplateServiceDataResult) *authv1.LoginTemplate {
	if r == nil { return nil }
	var meta *structpb.Struct
	if r.Metadata != nil {
		meta, _ = structpb.NewStruct(r.Metadata)
	}
	return &authv1.LoginTemplate{
		LoginTemplateUuid: r.LoginTemplateUUID.String(), Name: r.Name, Description: r.Description,
		Template: r.Template, Metadata: meta, Status: r.Status,
		IsDefault: r.IsDefault, IsSystem: r.IsSystem,
		CreatedAt: timestamppb.New(r.CreatedAt), UpdatedAt: timestamppb.New(r.UpdatedAt),
	}
}
