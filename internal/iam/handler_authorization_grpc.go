package iam

import (
	"context"

	authv1 "github.com/maintainerd/maintainerd-auth/internal/platform/gen/go/maintainerd/auth"
	"github.com/maintainerd/maintainerd-auth/internal/platform/middleware"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type AuthorizationGRPCHandler struct {
	authv1.UnimplementedAuthorizationServiceServer
	authorizationService ServiceAuthorizationService
}

func NewAuthorizationGRPCHandler(authorizationService ServiceAuthorizationService) *AuthorizationGRPCHandler {
	return &AuthorizationGRPCHandler{authorizationService: authorizationService}
}

func (h *AuthorizationGRPCHandler) Authorize(ctx context.Context, req *authv1.AuthorizeRequest) (*authv1.AuthorizeResponse, error) {
	// The caller supplies only the QUESTION (action + resource). Principal and
	// tenant come from the signed token, matching the REST twin — a body-supplied
	// principal lets any valid token probe allow/deny against any principal.
	//
	// The request message carries no tenant at all, so TenantID stayed 0 and
	// PolicyBundle short-circuited on its tenant guard: every call, legitimate or
	// not, answered "principal bundle unavailable".
	claims := middleware.JWTClaimsFromContext(ctx)
	if claims == nil {
		return nil, status.Error(codes.Unauthenticated, "authentication required")
	}
	principal := authorizationPrincipal(claims)
	if principal == "" {
		return nil, status.Error(codes.PermissionDenied, "this token has no principal to authorize")
	}
	tenantID := claims.TenantID
	if auth := middleware.AuthFromContext(ctx); auth != nil && auth.Tenant != nil {
		tenantID = auth.Tenant.TenantID
	}
	if tenantID == 0 {
		// A tenant-less decision would be evaluated against whichever tenant's
		// policies happened to be found first.
		return nil, status.Error(codes.PermissionDenied, "this token is not bound to a tenant")
	}

	decision := h.authorizationService.Authorize(ctx, AuthzRequest{
		Principal: principal,
		Action:    req.GetAction(),
		Resource:  req.GetResource(),
		TenantID:  tenantID,
	})
	return &authv1.AuthorizeResponse{Allowed: decision.Allowed, Reason: decision.Reason}, nil
}
