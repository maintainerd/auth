package notifier

import (
	"context"

	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/platform/apperror"
	authv1 "github.com/maintainerd/auth/internal/platform/gen/go/maintainerd/auth"
)

type TenantResolver interface {
	GetByUUID(ctx context.Context, tenantUUID uuid.UUID) (*TenantServiceDataResult, error)
}

type EmailConfigGRPCHandler struct {
	authv1.UnimplementedEmailConfigServiceServer
	tenantResolver TenantResolver
	svc            EmailConfigService
}

func NewEmailConfigGRPCHandler(r TenantResolver, svc EmailConfigService) *EmailConfigGRPCHandler {
	return &EmailConfigGRPCHandler{tenantResolver: r, svc: svc}
}

func (h *EmailConfigGRPCHandler) GetEmailConfig(ctx context.Context, req *authv1.GetEmailConfigRequest) (*authv1.GetEmailConfigResponse, error) {
	t, err := h.tenant(ctx, req.GetTenantUuid())
	if err != nil {
		return nil, err
	}
	r, err := h.svc.Get(ctx, t.TenantID)
	if err != nil {
		return nil, apperror.ToGRPCError(err)
	}
	return &authv1.GetEmailConfigResponse{Config: &authv1.EmailConfig{
		EmailConfigUuid: r.EmailConfigUUID.String(), Provider: r.Provider, Host: r.Host,
		Port: int32(r.Port), Username: r.Username, FromAddress: r.FromAddress, FromName: r.FromName,
		ReplyTo: r.ReplyTo, Encryption: r.Encryption, TestMode: r.TestMode, Status: r.Status,
	}}, nil
}

func (h *EmailConfigGRPCHandler) UpdateEmailConfig(ctx context.Context, req *authv1.UpdateEmailConfigRequest) (*authv1.UpdateEmailConfigResponse, error) {
	t, err := h.tenant(ctx, req.GetTenantUuid())
	if err != nil {
		return nil, err
	}
	r, err := h.svc.Update(ctx, t.TenantID, req.GetProvider(), req.GetHost(), int(req.GetPort()),
		req.GetUsername(), req.GetPassword(), req.GetFromAddress(), req.GetFromName(),
		req.GetReplyTo(), req.GetEncryption(), req.TestMode)
	if err != nil {
		return nil, apperror.ToGRPCError(err)
	}
	return &authv1.UpdateEmailConfigResponse{Config: &authv1.EmailConfig{
		EmailConfigUuid: r.EmailConfigUUID.String(), Provider: r.Provider, Host: r.Host,
		Port: int32(r.Port), Username: r.Username, FromAddress: r.FromAddress, FromName: r.FromName,
		ReplyTo: r.ReplyTo, Encryption: r.Encryption, TestMode: r.TestMode, Status: r.Status,
	}}, nil
}

func (h *EmailConfigGRPCHandler) tenant(ctx context.Context, tuuid string) (*TenantServiceDataResult, error) {
	p, err := uuid.Parse(tuuid)
	if err != nil {
		return nil, apperror.ToGRPCError(apperror.NewValidation("Invalid tenant UUID"))
	}
	r, err := h.tenantResolver.GetByUUID(ctx, p)
	if err != nil {
		return nil, apperror.ToGRPCError(err)
	}
	return r, nil
}
