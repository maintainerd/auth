package app

import (
	"context"

	"github.com/maintainerd/auth/internal/idp"
	"github.com/maintainerd/auth/internal/oauth"
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
		UserID:      info.UserID,
		UserUUID:    info.UserUUID,
		IdentitySub: info.IdentitySub,
		SessionID:   info.SessionID,
	}, nil
}
