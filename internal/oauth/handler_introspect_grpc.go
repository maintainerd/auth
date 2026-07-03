package oauth

import (
	"context"

	authv1 "github.com/maintainerd/maintainerd-auth/internal/platform/gen/go/maintainerd/auth"
)

type OAuthIntrospectionGRPCHandler struct {
	authv1.UnimplementedOAuthIntrospectionServiceServer
	svc OAuthTokenService
}

func NewOAuthIntrospectionGRPCHandler(svc OAuthTokenService) *OAuthIntrospectionGRPCHandler {
	return &OAuthIntrospectionGRPCHandler{svc: svc}
}

func (h *OAuthIntrospectionGRPCHandler) Introspect(ctx context.Context, req *authv1.IntrospectRequest) (*authv1.IntrospectResponse, error) {
	result, oerr := h.svc.Introspect(ctx, OAuthIntrospectRequestDTO{
		Token:         req.GetToken(),
		TokenTypeHint: req.GetTokenTypeHint(),
	}, OAuthClientCredentials{})
	if oerr != nil {
		return &authv1.IntrospectResponse{Active: false}, nil
	}
	return &authv1.IntrospectResponse{
		Active:    result.Active,
		Scope:     result.Scope,
		ClientId:  result.ClientID,
		Username:  result.Username,
		TokenType: result.TokenType,
		Exp:       result.Exp,
		Iat:       result.Iat,
		Sub:       result.Sub,
		Aud:       result.Aud,
		Iss:       result.Iss,
		Jti:       result.Jti,
	}, nil
}
