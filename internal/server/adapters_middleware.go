package server

import (
	"context"

	"github.com/maintainerd/auth/internal/authctx"
	"github.com/maintainerd/auth/internal/user"
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
	return toUserContext(u, clientID), nil
}

// toUserContext builds the full auth context for a resolved user. The tenant,
// identity provider, and client are taken from the user identity whose client
// identifier matches the clientID the subject authenticated with — the same
// identity the lookup query matched on.
func toUserContext(u *user.User, clientID string) *authctx.UserContext {
	var tenantID int64
	var tenant *authctx.AuthTenant
	var provider *authctx.AuthProvider
	var client *authctx.AuthClient

	for i := range u.UserIdentities {
		identity := &u.UserIdentities[i]
		if identity.Client == nil || identity.Client.Identifier == nil || *identity.Client.Identifier != clientID {
			continue
		}

		if identity.Tenant != nil {
			tenantID = identity.Tenant.TenantID
			tenant = &authctx.AuthTenant{
				TenantID:    identity.Tenant.TenantID,
				TenantUUID:  identity.Tenant.TenantUUID,
				Name:        identity.Tenant.Name,
				DisplayName: identity.Tenant.DisplayName,
				Identifier:  identity.Tenant.Identifier,
			}
		} else if identity.TenantID != 0 {
			tenantID = identity.TenantID
			tenant = &authctx.AuthTenant{
				TenantID: identity.TenantID,
			}
		}

		client = &authctx.AuthClient{
			ClientID:   identity.Client.ClientID,
			ClientUUID: identity.Client.ClientUUID,
			Identifier: identity.Client.Identifier,
		}

		if identity.Client.IdentityProvider != nil {
			provider = &authctx.AuthProvider{
				IdentityProviderID:   identity.Client.IdentityProvider.IdentityProviderID,
				IdentityProviderUUID: identity.Client.IdentityProvider.IdentityProviderUUID,
			}
		}
		break
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
					Identifier:  identity.Tenant.Identifier,
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
		perms := make([]authctx.AuthPermission, 0, len(role.RolePermissions))
		for _, rp := range role.RolePermissions {
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
