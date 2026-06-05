package branding

import (
	"context"

	"github.com/maintainerd/auth/internal/platform/apperror"
	authv1 "github.com/maintainerd/auth/internal/platform/gen/go/maintainerd/auth"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type EmailTemplateGRPCHandler struct {
	authv1.UnimplementedEmailTemplateServiceServer
	tenantResolver TenantResolver
	svc            EmailTemplateService
}

func NewEmailTemplateGRPCHandler(tenantResolver TenantResolver, svc EmailTemplateService) *EmailTemplateGRPCHandler {
	return &EmailTemplateGRPCHandler{tenantResolver: tenantResolver, svc: svc}
}

func (h *EmailTemplateGRPCHandler) ListEmailTemplates(ctx context.Context, req *authv1.ListEmailTemplatesRequest) (*authv1.ListEmailTemplatesResponse, error) {
	tenant, err := h.resolveTenant(ctx, req.GetTenantUuid())
	if err != nil { return nil, err }
	pg := grpcPagination(req.GetPagination())
	result, err := h.svc.GetAll(ctx, tenant.TenantID, optionalStr(req.GetName()), req.GetStatus(), req.IsDefault, req.IsSystem, pg.Page, pg.Limit, pg.SortBy, pg.SortOrder)
	if err != nil { return nil, apperror.ToGRPCError(err) }
	rows := make([]*authv1.EmailTemplate, len(result.Data))
	for i := range result.Data { rows[i] = emailTemplateProto(&result.Data[i]) }
	return &authv1.ListEmailTemplatesResponse{EmailTemplates: rows, Page: pageProto(result.Total, result.Page, result.Limit, result.TotalPages)}, nil
}

func (h *EmailTemplateGRPCHandler) GetEmailTemplate(ctx context.Context, req *authv1.GetEmailTemplateRequest) (*authv1.GetEmailTemplateResponse, error) {
	tenant, err := h.resolveTenant(ctx, req.GetTenantUuid())
	if err != nil { return nil, err }
	etUUID, err := parseUUID(req.GetEmailTemplateUuid(), "Email template UUID")
	if err != nil { return nil, err }
	result, err := h.svc.GetByUUID(ctx, etUUID, tenant.TenantID)
	if err != nil { return nil, apperror.ToGRPCError(err) }
	return &authv1.GetEmailTemplateResponse{EmailTemplate: emailTemplateProto(result)}, nil
}

func (h *EmailTemplateGRPCHandler) CreateEmailTemplate(ctx context.Context, req *authv1.CreateEmailTemplateRequest) (*authv1.CreateEmailTemplateResponse, error) {
	tenant, err := h.resolveTenant(ctx, req.GetTenantUuid())
	if err != nil { return nil, err }
	result, err := h.svc.Create(ctx, tenant.TenantID, req.GetName(), req.GetSubject(), req.GetBodyHtml(), req.BodyPlain, req.GetStatus(), req.GetIsDefault())
	if err != nil { return nil, apperror.ToGRPCError(err) }
	return &authv1.CreateEmailTemplateResponse{EmailTemplate: emailTemplateProto(result)}, nil
}

func (h *EmailTemplateGRPCHandler) UpdateEmailTemplate(ctx context.Context, req *authv1.UpdateEmailTemplateRequest) (*authv1.UpdateEmailTemplateResponse, error) {
	tenant, err := h.resolveTenant(ctx, req.GetTenantUuid())
	if err != nil { return nil, err }
	etUUID, err := parseUUID(req.GetEmailTemplateUuid(), "Email template UUID")
	if err != nil { return nil, err }
	result, err := h.svc.Update(ctx, etUUID, tenant.TenantID, req.GetName(), req.GetSubject(), req.GetBodyHtml(), req.BodyPlain, req.GetStatus())
	if err != nil { return nil, apperror.ToGRPCError(err) }
	return &authv1.UpdateEmailTemplateResponse{EmailTemplate: emailTemplateProto(result)}, nil
}

func (h *EmailTemplateGRPCHandler) SetEmailTemplateStatus(ctx context.Context, req *authv1.SetEmailTemplateStatusRequest) (*authv1.SetEmailTemplateStatusResponse, error) {
	tenant, err := h.resolveTenant(ctx, req.GetTenantUuid())
	if err != nil { return nil, err }
	etUUID, err := parseUUID(req.GetEmailTemplateUuid(), "Email template UUID")
	if err != nil { return nil, err }
	result, err := h.svc.UpdateStatus(ctx, etUUID, tenant.TenantID, req.GetStatus())
	if err != nil { return nil, apperror.ToGRPCError(err) }
	return &authv1.SetEmailTemplateStatusResponse{EmailTemplate: emailTemplateProto(result)}, nil
}

func (h *EmailTemplateGRPCHandler) DeleteEmailTemplate(ctx context.Context, req *authv1.DeleteEmailTemplateRequest) (*authv1.DeleteEmailTemplateResponse, error) {
	tenant, err := h.resolveTenant(ctx, req.GetTenantUuid())
	if err != nil { return nil, err }
	etUUID, err := parseUUID(req.GetEmailTemplateUuid(), "Email template UUID")
	if err != nil { return nil, err }
	result, err := h.svc.Delete(ctx, etUUID, tenant.TenantID)
	if err != nil { return nil, apperror.ToGRPCError(err) }
	return &authv1.DeleteEmailTemplateResponse{EmailTemplate: emailTemplateProto(result)}, nil
}

func (h *EmailTemplateGRPCHandler) resolveTenant(ctx context.Context, tenantUUID string) (*TenantServiceDataResult, error) {
	parsed, err := parseUUID(tenantUUID, "Tenant UUID")
	if err != nil { return nil, err }
	result, err := h.tenantResolver.GetByUUID(ctx, parsed)
	if err != nil { return nil, apperror.ToGRPCError(err) }
	return result, nil
}

func emailTemplateProto(r *EmailTemplateServiceDataResult) *authv1.EmailTemplate {
	if r == nil { return nil }
	return &authv1.EmailTemplate{
		EmailTemplateUuid: r.EmailTemplateUUID.String(), Name: r.Name, Subject: r.Subject,
		BodyHtml: r.BodyHTML, BodyPlain: r.BodyPlain, Status: r.Status,
		IsDefault: r.IsDefault, IsSystem: r.IsSystem,
		CreatedAt: timestamppb.New(r.CreatedAt), UpdatedAt: timestamppb.New(r.UpdatedAt),
	}
}
