package app

import (
	"context"
	"time"

	"github.com/maintainerd/maintainerd-auth/internal/idp"
	"github.com/maintainerd/maintainerd-auth/internal/oauth"
)

// identityLinkStoreAdapter satisfies idp.IdentityLinkRequestStore over the
// OAuth broker-session table.
//
// Account linking and brokered sign-in are the same shape of problem — hold a
// state, a PKCE verifier and a nonce across a redirect to a provider — so they
// share one table, discriminated by `purpose`. The adapter lives here because
// internal/idp must not import internal/oauth.
//
// Every read is filtered to purpose='link'. A login session presented to the
// link callback would otherwise attach an arbitrary provider identity to the
// caller's account.
type identityLinkStoreAdapter struct {
	sessions oauth.OAuthBrokerSessionRepository
}

func (a identityLinkStoreAdapter) Create(_ context.Context, req *idp.IdentityLinkRequest) error {
	userID := req.UserID
	created, err := a.sessions.Create(&oauth.OAuthBrokerSession{
		TenantID:                   req.TenantID,
		ClientID:                   req.ClientID,
		IdentityProviderID:         req.IdentityProviderID,
		IdentityProviderIdentifier: req.ProviderIdentifier,
		Purpose:                    oauth.BrokerPurposeLink,
		UserID:                     &userID,
		IdpState:                   req.State,
		IdpPKCEVerifier:            req.PKCEVerifier,
		IdpNonce:                   &req.Nonce,
		ExpiresAt:                  req.ExpiresAt,
	})
	if err != nil {
		return err
	}
	req.ID = created.OAuthBrokerSessionID
	return nil
}

func (a identityLinkStoreAdapter) FindByState(_ context.Context, state string) (*idp.IdentityLinkRequest, error) {
	row, err := a.sessions.FindByIdpState(state)
	if err != nil || row == nil {
		return nil, err
	}
	// A state issued for brokered sign-in is not a link request. Reported as
	// "not found" so probing cannot distinguish the two.
	if row.Purpose != oauth.BrokerPurposeLink || row.UserID == nil {
		return nil, nil
	}
	nonce := ""
	if row.IdpNonce != nil {
		nonce = *row.IdpNonce
	}
	return &idp.IdentityLinkRequest{
		ID:                 row.OAuthBrokerSessionID,
		UserID:             *row.UserID,
		TenantID:           row.TenantID,
		ClientID:           row.ClientID,
		IdentityProviderID: row.IdentityProviderID,
		ProviderIdentifier: row.IdentityProviderIdentifier,
		State:              row.IdpState,
		PKCEVerifier:       row.IdpPKCEVerifier,
		Nonce:              nonce,
		ExpiresAt:          row.ExpiresAt,
		Consumed:           row.IsConsumed(),
	}, nil
}

func (a identityLinkStoreAdapter) Consume(_ context.Context, id int64) error {
	return a.sessions.Consume(id, time.Now())
}
