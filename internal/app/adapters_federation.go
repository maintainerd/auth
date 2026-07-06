package app

import (
	"context"

	"github.com/maintainerd/maintainerd-auth/internal/federation"
	"github.com/maintainerd/maintainerd-auth/internal/oauth"
	"github.com/maintainerd/maintainerd-auth/internal/platform/apperror"
)

// ---------------------------------------------------------------------------
// federation.ExchangeAuditor ← oauth.OAuthTokenExchangeRepository
//
// The federation package records every successful workload token exchange in
// oauth_token_exchanges (section 3.20) via this adapter, so it never imports
// the oauth domain.
// ---------------------------------------------------------------------------

type federationExchangeAuditor struct {
	repo oauth.OAuthTokenExchangeRepository
}

func newFederationExchangeAuditor(repo oauth.OAuthTokenExchangeRepository) *federationExchangeAuditor {
	return &federationExchangeAuditor{repo: repo}
}

func (a *federationExchangeAuditor) RecordExchange(_ context.Context, entry federation.ExchangeAuditEntry) error {
	if a.repo == nil {
		return nil
	}
	return a.repo.Record(&oauth.OAuthTokenExchange{
		TenantID:           entry.TenantID,
		ActorClientID:      entry.ActorClientID,
		SubjectTokenType:   entry.SubjectTokenType,
		RequestedTokenType: entry.RequestedTokenType,
		ExchangeType:       entry.ExchangeType,
		Scope:              entry.Scopes,
		IssuedJTI:          entry.IssuedJTI,
		IPAddress:          entry.IPAddress,
	})
}

// ---------------------------------------------------------------------------
// oauth.WorkloadIdentityExchanger ← federation.WorkloadIdentityFederationService
//
// The oauth token endpoint delegates workload identity federation exchanges to
// the federation service through this adapter, so the oauth domain never
// imports the federation domain.
// ---------------------------------------------------------------------------

type oauthWorkloadIdentityExchanger struct {
	service federation.WorkloadIdentityFederationService
}

func newOAuthWorkloadIdentityExchanger(service federation.WorkloadIdentityFederationService) *oauthWorkloadIdentityExchanger {
	return &oauthWorkloadIdentityExchanger{service: service}
}

func (a *oauthWorkloadIdentityExchanger) ExchangeWorkloadToken(ctx context.Context, in oauth.WorkloadTokenExchangeInput) (*oauth.WorkloadTokenExchangeResult, *apperror.OAuthError) {
	result, oerr := a.service.ExchangeWorkloadToken(ctx, federation.WorkloadExchangeInput{
		SubjectToken: in.SubjectToken,
		Scope:        in.Scope,
		Audience:     in.Audience,
		Resource:     in.Resource,
		IPAddress:    in.IPAddress,
	})
	if oerr != nil {
		return nil, oerr
	}
	if result == nil {
		return nil, nil
	}
	return &oauth.WorkloadTokenExchangeResult{
		AccessToken:     result.AccessToken,
		IssuedTokenType: result.IssuedTokenType,
		TokenType:       result.TokenType,
		ExpiresIn:       result.ExpiresIn,
		Scope:           result.Scope,
	}, nil
}
