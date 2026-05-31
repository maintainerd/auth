package app

import (
	"github.com/maintainerd/auth/internal/authn"
	"github.com/maintainerd/auth/internal/idp"
)

func toAuthnTenantFromIdp(t *idp.Tenant) *authn.Tenant {
	if t == nil {
		return nil
	}
	return &authn.Tenant{
		TenantID: t.TenantID, TenantUUID: t.TenantUUID, Name: t.Name,
		DisplayName: t.DisplayName, Description: t.Description, Identifier: t.Identifier,
		Status: t.Status, IsPublic: t.IsPublic, IsSystem: t.IsSystem,
		CreatedAt: t.CreatedAt, UpdatedAt: t.UpdatedAt,
	}
}

func toAuthnIDPFromIdp(p *idp.IdentityProvider) *authn.IdentityProvider {
	if p == nil {
		return nil
	}
	return &authn.IdentityProvider{
		IdentityProviderID: p.IdentityProviderID, IdentityProviderUUID: p.IdentityProviderUUID,
		TenantID: p.TenantID, Name: p.Name, Provider: p.Provider, ProviderType: p.ProviderType,
		Identifier: p.Identifier, Status: p.Status, IsDefault: p.IsDefault, IsSystem: p.IsSystem,
		Tenant:    toAuthnTenantFromIdp(p.Tenant),
		CreatedAt: p.CreatedAt, UpdatedAt: p.UpdatedAt,
	}
}

func toIdpIDP(p *authn.IdentityProvider) *idp.IdentityProvider {
	if p == nil {
		return nil
	}
	return &idp.IdentityProvider{
		IdentityProviderID: p.IdentityProviderID, IdentityProviderUUID: p.IdentityProviderUUID,
		TenantID: p.TenantID, Name: p.Name, Provider: p.Provider, ProviderType: p.ProviderType,
		Identifier: p.Identifier, Status: p.Status, IsDefault: p.IsDefault, IsSystem: p.IsSystem,
		CreatedAt: p.CreatedAt, UpdatedAt: p.UpdatedAt,
	}
}

func mapAuthnIDPs(items []idp.IdentityProvider) []authn.IdentityProvider {
	out := make([]authn.IdentityProvider, len(items))
	for i := range items {
		out[i] = *toAuthnIDPFromIdp(&items[i])
	}
	return out
}
