package app

import (
	"github.com/maintainerd/maintainerd-auth/internal/authn"
	"github.com/maintainerd/maintainerd-auth/internal/client"
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
		AllowRegistration: p.AllowRegistration,
		Tenant:            toAuthnTenantFromClient(p.Tenant),
		CreatedAt:         p.CreatedAt, UpdatedAt: p.UpdatedAt,
	}
}

func defaultConnectedIDPFromClient(c *client.Client) *client.IdentityProvider {
	if c == nil {
		return nil
	}
	if c.IdentityProvider != nil {
		return c.IdentityProvider
	}
	if c.ConnectedProviders == nil {
		return nil
	}
	var first *client.IdentityProvider
	for i := range *c.ConnectedProviders {
		conn := &(*c.ConnectedProviders)[i]
		if !conn.Enabled || conn.IdentityProvider == nil {
			continue
		}
		if first == nil {
			first = conn.IdentityProvider
		}
		if conn.IsDefault {
			return conn.IdentityProvider
		}
	}
	return first
}

func toAuthnClientIdentityProviders(c *client.Client) *[]authn.ClientIdentityProvider {
	if c == nil || c.ConnectedProviders == nil {
		return nil
	}
	out := make([]authn.ClientIdentityProvider, 0, len(*c.ConnectedProviders))
	for i := range *c.ConnectedProviders {
		conn := (*c.ConnectedProviders)[i]
		out = append(out, authn.ClientIdentityProvider{
			ClientIdentityProviderID:   conn.ClientIdentityProviderID,
			ClientIdentityProviderUUID: conn.ClientIdentityProviderUUID,
			TenantID:                   conn.TenantID,
			ClientID:                   conn.ClientID,
			IdentityProviderID:         conn.IdentityProviderID,
			Enabled:                    conn.Enabled,
			IsDefault:                  conn.IsDefault,
			IdentityProvider:           toAuthnIDPFromClient(conn.IdentityProvider),
		})
	}
	return &out
}

func toAuthnClient(c *client.Client) *authn.Client {
	if c == nil {
		return nil
	}
	idp := defaultConnectedIDPFromClient(c)
	var idpID int64
	if idp != nil {
		idpID = idp.IdentityProviderID
	}
	return &authn.Client{
		ClientID: c.ClientID, ClientUUID: c.ClientUUID, TenantID: c.TenantID,
		IdentityProviderID: idpID, Name: c.Name, DisplayName: c.DisplayName,
		ClientType: c.ClientType, Domain: c.Domain, Identifier: c.Identifier,
		Status: c.Status, IsDefault: c.IsDefault, IsSystem: c.IsSystem,
		AccessTokenTTL: c.AccessTokenTTL, RefreshTokenTTL: c.RefreshTokenTTL,
		RequiredACR: c.RequiredACR, RequirePKCE: c.RequirePKCE,
		SessionIdleTimeout: c.SessionIdleTimeout, SessionAbsoluteTimeout: c.SessionAbsoluteTimeout,
		BrandingID:         c.BrandingID,
		AllowRegistration:  c.AllowRegistration,
		IdentityProvider:   toAuthnIDPFromClient(idp),
		ConnectedProviders: toAuthnClientIdentityProviders(c),
		CreatedAt:          c.CreatedAt, UpdatedAt: c.UpdatedAt,
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
		BrandingID:        c.BrandingID,
		AllowRegistration: c.AllowRegistration,
		CreatedAt:         c.CreatedAt, UpdatedAt: c.UpdatedAt,
	}
}

func mapAuthnClients(items []client.Client) []authn.Client {
	out := make([]authn.Client, len(items))
	for i := range items {
		out[i] = *toAuthnClient(&items[i])
	}
	return out
}
