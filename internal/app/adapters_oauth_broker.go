package app

import (
	"context"
	"time"

	"github.com/maintainerd/maintainerd-auth/internal/authn"
	"github.com/maintainerd/maintainerd-auth/internal/idp"
	"github.com/maintainerd/maintainerd-auth/internal/oauth"
)

// oauthBrokerProviderResolver bridges idp.FederationService.ResolveBrokerProvider
// to the oauth.BrokerProviderResolver consumer interface.
type oauthBrokerProviderResolver struct {
	federation idp.FederationService
}

func (r *oauthBrokerProviderResolver) ResolveBrokerProvider(ctx context.Context, idpIdentifier string) (*oauth.BrokerProvider, error) {
	info, err := r.federation.ResolveBrokerProvider(ctx, idpIdentifier)
	if err != nil {
		return nil, err
	}
	return &oauth.BrokerProvider{
		AuthorizationEndpoint: info.AuthorizationEndpoint,
		ClientID:              info.ClientID,
		Scopes:                info.Scopes,
	}, nil
}

// oauthBrokerCallbackResolver bridges idp.FederationService.ResolveBrokerUser
// to the oauth.BrokerCallbackResolver consumer interface.
type oauthBrokerCallbackResolver struct {
	federation idp.FederationService
}

func (r *oauthBrokerCallbackResolver) ResolveBrokerUser(ctx context.Context, idpID int64, code, pkceVerifier, nonce, redirectURI string, clientID int64) (*oauth.BrokerResolvedUser, error) {
	info, err := r.federation.ResolveBrokerUser(ctx, idpID, code, pkceVerifier, nonce, redirectURI, clientID)
	if err != nil {
		return nil, err
	}
	return &oauth.BrokerResolvedUser{
		UserID:              info.UserID,
		UserUUID:            info.UserUUID,
		IdentitySub:         info.IdentitySub,
		SessionID:           info.SessionID,
		AccountLinkToken:    info.AccountLinkToken,
		AccountLinkProvider: info.AccountLinkProvider,
		AccountLinkEmail:    info.AccountLinkEmail,
	}, nil
}

// accountLinkVerifierAdapter bridges authn.AccountLinkRequestRepository to the
// oauth.AccountLinkVerifier consumer interface.
type accountLinkVerifierAdapter struct {
	repo authn.AccountLinkRequestRepository
}

func (a *accountLinkVerifierAdapter) FindConfirmedLink(token string) (int64, bool, error) {
	req, err := a.repo.FindByToken(token)
	if err != nil || req == nil {
		return 0, false, err
	}
	// Must be confirmed AND still within its TTL. A confirmed-but-expired token
	// was previously usable indefinitely; treat an expired confirmation as not
	// found so a stale capability cannot be redeemed.
	if req.Status != "confirmed" || req.IsExpired() {
		return 0, false, nil
	}
	return req.ExistingUserID, true, nil
}

// ConsumeConfirmedLink marks a confirmed link token used so it is single-use at
// the broker-resume step. Returns true when THIS call performed the transition
// (RowsAffected 1); false means it was already used/expired — the caller treats
// that as a replay and refuses.
func (a *accountLinkVerifierAdapter) ConsumeConfirmedLink(token string) (bool, error) {
	req, err := a.repo.FindByToken(token)
	if err != nil || req == nil {
		return false, err
	}
	affected, err := a.repo.MarkUsed(req.AccountLinkRequestID, time.Now())
	if err != nil {
		return false, err
	}
	return affected == 1, nil
}
