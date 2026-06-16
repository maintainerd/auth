package app

import (
	"github.com/maintainerd/auth/internal/authn"
	"github.com/maintainerd/auth/internal/client"
)

func toAuthnTenantFromClient(t *client.Tenant) *authn.Tenant {
	if t == nil {
		return nil
	}
	return &authn.Tenant{
		TenantID: t.TenantID, TenantUUID: t.TenantUUID, Name: t.Name,
		DisplayName: t.DisplayName, Description: t.Description, Identifier: t.Identifier,
		Status: t.Status, IsSystem: t.IsSystem,
		CreatedAt: t.CreatedAt, UpdatedAt: t.UpdatedAt,
	}
}

func toAuthnIDPFromClient(p *client.IdentityProvider) *authn.IdentityProvider {
	if p == nil {
		return nil
	}
	return &authn.IdentityProvider{
		IdentityProviderID: p.IdentityProviderID, IdentityProviderUUID: p.IdentityProviderUUID,
		TenantID: p.TenantID, Name: p.Name, Provider: p.Provider, ProviderType: p.ProviderType,
		Identifier: p.Identifier, Status: p.Status, IsDefault: p.IsDefault, IsSystem: p.IsSystem,
		Tenant:    toAuthnTenantFromClient(p.Tenant),
		CreatedAt: p.CreatedAt, UpdatedAt: p.UpdatedAt,
	}
}

func toAuthnClient(c *client.Client) *authn.Client {
	if c == nil {
		return nil
	}
	return &authn.Client{
		ClientID: c.ClientID, ClientUUID: c.ClientUUID, TenantID: c.TenantID,
		IdentityProviderID: c.IdentityProviderID, Name: c.Name, DisplayName: c.DisplayName,
		ClientType: c.ClientType, Domain: c.Domain, Identifier: c.Identifier,
		Status: c.Status, IsDefault: c.IsDefault, IsSystem: c.IsSystem,
		AccessTokenTTL: c.AccessTokenTTL, RefreshTokenTTL: c.RefreshTokenTTL,
		RequiredACR: c.RequiredACR, RequirePKCE: c.RequirePKCE,
		SessionIdleTimeout: c.SessionIdleTimeout, SessionAbsoluteTimeout: c.SessionAbsoluteTimeout,
		IdentityProvider: toAuthnIDPFromClient(c.IdentityProvider),
		CreatedAt:        c.CreatedAt, UpdatedAt: c.UpdatedAt,
	}
}

func toClientClient(c *authn.Client) *client.Client {
	if c == nil {
		return nil
	}
	return &client.Client{
		ClientID: c.ClientID, ClientUUID: c.ClientUUID, TenantID: c.TenantID,
		IdentityProviderID: c.IdentityProviderID, Name: c.Name, DisplayName: c.DisplayName,
		ClientType: c.ClientType, Domain: c.Domain, Identifier: c.Identifier,
		Status: c.Status, IsDefault: c.IsDefault, IsSystem: c.IsSystem,
		AccessTokenTTL: c.AccessTokenTTL, RefreshTokenTTL: c.RefreshTokenTTL,
		RequiredACR: c.RequiredACR, RequirePKCE: c.RequirePKCE,
		SessionIdleTimeout: c.SessionIdleTimeout, SessionAbsoluteTimeout: c.SessionAbsoluteTimeout,
		CreatedAt: c.CreatedAt, UpdatedAt: c.UpdatedAt,
	}
}

func mapAuthnClients(items []client.Client) []authn.Client {
	out := make([]authn.Client, len(items))
	for i := range items {
		out[i] = *toAuthnClient(&items[i])
	}
	return out
}
