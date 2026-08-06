package server

import (
	"context"

	"github.com/maintainerd/maintainerd-auth/internal/authctx"
	"github.com/maintainerd/maintainerd-auth/internal/shared"
	"github.com/maintainerd/maintainerd-auth/internal/user"
)

type middlewareUserContextProvider struct {
	userService user.UserService
}

func newMiddlewareUserContextProvider(userService user.UserService) *middlewareUserContextProvider {
	return &middlewareUserContextProvider{userService: userService}
}

func (p *middlewareUserContextProvider) FindBySubAndClientID(ctx context.Context, sub string, clientID string) (*authctx.UserContext, error) {
	u, err := p.userService.FindBySubAndClientID(ctx, sub, clientID)
	if err != nil || u == nil {
		return nil, err
	}
	// The repository already proved this client may use the identity's provider
	// (client_identity_providers). Resolving the client here is purely to
	// populate the context.
	c, err := p.userService.FindClientByIdentifier(ctx, clientID)
	if err != nil {
		return nil, err
	}
	return toUserContext(u, sub, c), nil
}

// toUserContext builds the auth context for a resolved user.
//
// The identity is selected by SUB, which is unique per (tenant, provider, sub)
// — it is exactly the row the lookup matched. Tenant and identity provider come
// from that identity; the CLIENT comes from the request, because an identity
// belongs to a provider and is usable from every client connected to it.
//
// This previously matched on identity.Client.Identifier, which required every
// identity to be bound to one client — the thing that made the same person a
// different subject on every application.
func toUserContext(u *user.User, sub string, requestClient *user.Client) *authctx.UserContext {
	var tenantID int64
	var tenant *authctx.AuthTenant
	var provider *authctx.AuthProvider
	var client *authctx.AuthClient

	for i := range u.UserIdentities {
		identity := &u.UserIdentities[i]
		if identity.Sub != sub {
			continue
		}

		if identity.Tenant != nil {
			tenantID = identity.Tenant.TenantID
			tenant = &authctx.AuthTenant{
				TenantID:    identity.Tenant.TenantID,
				TenantUUID:  identity.Tenant.TenantUUID,
				Name:        identity.Tenant.Name,
				DisplayName: identity.Tenant.DisplayName,
				Identifier:  identity.Tenant.Name,
			}
		} else if identity.TenantID != 0 {
			tenantID = identity.TenantID
			tenant = &authctx.AuthTenant{
				TenantID: identity.TenantID,
			}
		}

		if identity.IdentityProvider != nil {
			provider = &authctx.AuthProvider{
				IdentityProviderID:   identity.IdentityProvider.IdentityProviderID,
				IdentityProviderUUID: identity.IdentityProvider.IdentityProviderUUID,
			}
		}
		break
	}

	if requestClient != nil {
		client = &authctx.AuthClient{
			ClientID:   requestClient.ClientID,
			ClientUUID: requestClient.ClientUUID,
			Identifier: requestClient.Identifier,
		}
	}

	return &authctx.UserContext{
		User:     toAuthUser(u, tenantID),
		Tenant:   tenant,
		Provider: provider,
		Client:   client,
	}
}

func toUserContextByTenant(u *user.User, tenantID int64) *authctx.UserContext {
	var tenant *authctx.AuthTenant
	for i := range u.UserIdentities {
		identity := &u.UserIdentities[i]
		if identity.TenantID == tenantID || (identity.Tenant != nil && identity.Tenant.TenantID == tenantID) {
			if identity.Tenant != nil {
				tenant = &authctx.AuthTenant{
					TenantID:    identity.Tenant.TenantID,
					TenantUUID:  identity.Tenant.TenantUUID,
					Name:        identity.Tenant.Name,
					DisplayName: identity.Tenant.DisplayName,
					Identifier:  identity.Tenant.Name,
				}
			} else {
				tenant = &authctx.AuthTenant{TenantID: tenantID}
			}
			break
		}
	}
	if tenant == nil {
		tenant = &authctx.AuthTenant{TenantID: tenantID}
	}

	return &authctx.UserContext{
		User:   toAuthUser(u, tenantID),
		Tenant: tenant,
	}
}

func toAuthUser(u *user.User, tenantID int64) *authctx.AuthUser {
	if u == nil {
		return nil
	}

	roles := make([]authctx.AuthRole, 0, len(u.UserRoles))
	for _, ur := range u.UserRoles {
		if ur.Role == nil {
			continue
		}
		role := ur.Role
		if tenantID != 0 && role.TenantID != tenantID {
			continue
		}
		// Deactivating a role or permission is a revocation — SetStatus even
		// revokes the affected sessions to force a reload. Without this filter the
		// reload handed the permission straight back, so "inactive" only ever
		// changed how the row rendered in the console.
		if !statusGrants(role.Status) {
			continue
		}
		perms := make([]authctx.AuthPermission, 0, len(role.RolePermissions))
		for _, rp := range role.RolePermissions {
			if !statusGrants(rp.Permission.Status) {
				continue
			}
			perms = append(perms, authctx.AuthPermission{
				PermissionID:   rp.Permission.PermissionID,
				PermissionUUID: rp.Permission.PermissionUUID,
				Name:           rp.Permission.Name,
			})
		}
		roles = append(roles, authctx.AuthRole{
			RoleID:      role.RoleID,
			RoleUUID:    role.RoleUUID,
			Name:        role.Name,
			Permissions: perms,
		})
	}

	var profile *authctx.AuthProfile
	if u.Profile != nil {
		profile = &authctx.AuthProfile{
			DisplayName: u.Profile.DisplayName,
			FirstName:   u.Profile.FirstName,
			LastName:    u.Profile.LastName,
			ProfileURL:  u.Profile.ProfileURL,
		}
	}

	return &authctx.AuthUser{
		UserID:          u.UserID,
		UserUUID:        u.UserUUID,
		Status:          u.Status,
		Roles:           roles,
		Email:           u.Email,
		IsEmailVerified: u.IsEmailVerified,
		Phone:           u.Phone,
		IsPhoneVerified: u.IsPhoneVerified,
		Fullname:        u.Fullname,
		UpdatedAt:       u.UpdatedAt,
		Profile:         profile,
	}
}

// statusGrants reports whether a role or permission in this state still confers
// access. An empty status is treated as granting: several projections select a
// subset of columns and legitimately leave it blank, and failing closed there
// would lock every user out rather than deny one permission.
func statusGrants(status string) bool {
	return status == "" || status == shared.StatusActive
}
