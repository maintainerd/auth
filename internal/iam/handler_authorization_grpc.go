package iam

import (
	"context"

	authv1 "github.com/maintainerd/maintainerd-auth/internal/platform/gen/go/maintainerd/auth"
)

type AuthorizationGRPCHandler struct {
	authv1.UnimplementedAuthorizationServiceServer
	authorizationService ServiceAuthorizationService
}

func NewAuthorizationGRPCHandler(authorizationService ServiceAuthorizationService) *AuthorizationGRPCHandler {
	return &AuthorizationGRPCHandler{authorizationService: authorizationService}
}

func (h *AuthorizationGRPCHandler) Authorize(ctx context.Context, req *authv1.AuthorizeRequest) (*authv1.AuthorizeResponse, error) {
	decision := h.authorizationService.Authorize(ctx, AuthzRequest{Principal: req.GetPrincipal(), Action: req.GetAction(), Resource: req.GetResource()})
	return &authv1.AuthorizeResponse{Allowed: decision.Allowed, Reason: decision.Reason}, nil
}
