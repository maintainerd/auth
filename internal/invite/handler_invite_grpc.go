package invite

import (
	"context"

	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/platform/apperror"
	authv1 "github.com/maintainerd/auth/internal/platform/gen/go/maintainerd/auth"
)

type TenantResolver interface {
	GetTenantIDByUUID(ctx context.Context, tenantUUID uuid.UUID) (int64, error)
}

type InviteGRPCHandler struct {
	authv1.UnimplementedInviteServiceServer
	tenantResolver TenantResolver
	inviteService  InviteService
}

func NewInviteGRPCHandler(tenantResolver TenantResolver, inviteService InviteService) *InviteGRPCHandler {
	return &InviteGRPCHandler{tenantResolver: tenantResolver, inviteService: inviteService}
}

func (h *InviteGRPCHandler) SendInvite(ctx context.Context, req *authv1.SendInviteRequest) (*authv1.SendInviteResponse, error) {
	tenantUUID, err := parseUUID(req.GetTenantUuid(), "Tenant UUID")
	if err != nil {
		return nil, err
	}
	tenantID, err := h.tenantResolver.GetTenantIDByUUID(ctx, tenantUUID)
	if err != nil {
		return nil, apperror.ToGRPCError(err)
	}
	_, err = h.inviteService.SendInvite(ctx, tenantID, req.GetEmail(), 0, nil, nil)
	if err != nil {
		return nil, apperror.ToGRPCError(err)
	}
	return &authv1.SendInviteResponse{Message: "Invite sent successfully"}, nil
}

func parseUUID(value string, label string) (uuid.UUID, error) {
	if value == "" {
		return uuid.Nil, apperror.ToGRPCError(apperror.NewValidation(label + " is required"))
	}
	parsed, err := uuid.Parse(value)
	if err != nil {
		return uuid.Nil, apperror.ToGRPCError(apperror.NewValidation("Invalid " + label))
	}
	return parsed, nil
}
