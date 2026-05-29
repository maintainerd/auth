package app

import (
	"github.com/maintainerd/auth/internal/authn"
	"github.com/maintainerd/auth/internal/client"
	"github.com/maintainerd/auth/internal/iam"
	"github.com/maintainerd/auth/internal/idp"
	"github.com/maintainerd/auth/internal/invite"
	"github.com/maintainerd/auth/internal/user"
)

func toAuthnUser(u *user.User) *authn.User {
	if u == nil {
		return nil
	}
	return &authn.User{
		UserID:              u.UserID,
		UserUUID:            u.UserUUID,
		Username:            u.Username,
		Fullname:            u.Fullname,
		Email:               u.Email,
		Phone:               u.Phone,
		Password:            u.Password,
		IsEmailVerified:     u.IsEmailVerified,
		IsPhoneVerified:     u.IsPhoneVerified,
		IsProfileCompleted:  u.IsProfileCompleted,
		IsAccountCompleted:  u.IsAccountCompleted,
		Status:              u.Status,
		ForcePasswordChange: u.ForcePasswordChange,
		PasswordChangedAt:   u.PasswordChangedAt,
		CreatedAt:           u.CreatedAt,
		UpdatedAt:           u.UpdatedAt,
	}
}

func toUserUser(u *authn.User) *user.User {
	if u == nil {
		return nil
	}
	return &user.User{
		UserID:              u.UserID,
		UserUUID:            u.UserUUID,
		Username:            u.Username,
		Fullname:            u.Fullname,
		Email:               u.Email,
		Phone:               u.Phone,
		Password:            u.Password,
		IsEmailVerified:     u.IsEmailVerified,
		IsPhoneVerified:     u.IsPhoneVerified,
		IsProfileCompleted:  u.IsProfileCompleted,
		IsAccountCompleted:  u.IsAccountCompleted,
		Status:              u.Status,
		ForcePasswordChange: u.ForcePasswordChange,
		PasswordChangedAt:   u.PasswordChangedAt,
		CreatedAt:           u.CreatedAt,
		UpdatedAt:           u.UpdatedAt,
	}
}

func mapAuthnUsers(items []user.User) []authn.User {
	out := make([]authn.User, len(items))
	for i := range items {
		out[i] = *toAuthnUser(&items[i])
	}
	return out
}

func toAuthnTenantFromClient(t *client.Tenant) *authn.Tenant {
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

func toAuthnUserIdentity(u *user.UserIdentity) *authn.UserIdentity {
	if u == nil {
		return nil
	}
	return &authn.UserIdentity{
		UserIdentityID: u.UserIdentityID, UserIdentityUUID: u.UserIdentityUUID,
		TenantID: u.TenantID, UserID: u.UserID, ClientID: u.ClientID,
		Provider: u.Provider, Sub: u.Sub, Metadata: u.Metadata,
		CreatedAt: u.CreatedAt, UpdatedAt: u.UpdatedAt,
	}
}

func toUserUserIdentity(u *authn.UserIdentity) *user.UserIdentity {
	if u == nil {
		return nil
	}
	return &user.UserIdentity{
		UserIdentityID: u.UserIdentityID, UserIdentityUUID: u.UserIdentityUUID,
		TenantID: u.TenantID, UserID: u.UserID, ClientID: u.ClientID,
		Provider: u.Provider, Sub: u.Sub, Metadata: u.Metadata,
		CreatedAt: u.CreatedAt, UpdatedAt: u.UpdatedAt,
	}
}

func mapAuthnUserIdentities(items []user.UserIdentity) []authn.UserIdentity {
	out := make([]authn.UserIdentity, len(items))
	for i := range items {
		out[i] = *toAuthnUserIdentity(&items[i])
	}
	return out
}

func toAuthnUserRole(u *user.UserRole) *authn.UserRole {
	if u == nil {
		return nil
	}
	return &authn.UserRole{UserRoleID: u.UserRoleID, UserID: u.UserID, RoleID: u.RoleID, CreatedAt: u.CreatedAt}
}

func toUserUserRole(u *authn.UserRole) *user.UserRole {
	if u == nil {
		return nil
	}
	return &user.UserRole{UserRoleID: u.UserRoleID, UserID: u.UserID, RoleID: u.RoleID, CreatedAt: u.CreatedAt}
}

func mapAuthnUserRoles(items []user.UserRole) []authn.UserRole {
	out := make([]authn.UserRole, len(items))
	for i := range items {
		out[i] = *toAuthnUserRole(&items[i])
	}
	return out
}

func toAuthnRole(r *iam.Role) *authn.Role {
	if r == nil {
		return nil
	}
	return &authn.Role{
		RoleID: r.RoleID, RoleUUID: r.RoleUUID, TenantID: r.TenantID, Name: r.Name,
		Description: r.Description, Status: r.Status, IsDefault: r.IsDefault, IsSystem: r.IsSystem,
		CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt,
	}
}

func toIamRole(r *authn.Role) *iam.Role {
	if r == nil {
		return nil
	}
	return &iam.Role{
		RoleID: r.RoleID, RoleUUID: r.RoleUUID, TenantID: r.TenantID, Name: r.Name,
		Description: r.Description, Status: r.Status, IsDefault: r.IsDefault, IsSystem: r.IsSystem,
		CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt,
	}
}

func mapAuthnRoles(items []iam.Role) []authn.Role {
	out := make([]authn.Role, len(items))
	for i := range items {
		out[i] = *toAuthnRole(&items[i])
	}
	return out
}

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

func toAuthnInvite(i *invite.Invite) *authn.Invite {
	if i == nil {
		return nil
	}
	return &authn.Invite{
		InviteID: i.InviteID, InviteUUID: i.InviteUUID, TenantID: i.TenantID,
		InvitedEmail: i.InvitedEmail, Status: i.Status, ExpiresAt: i.ExpiresAt,
		CreatedAt: i.CreatedAt, UpdatedAt: i.UpdatedAt,
	}
}

func toInviteInvite(i *authn.Invite) *invite.Invite {
	if i == nil {
		return nil
	}
	return &invite.Invite{
		InviteID: i.InviteID, InviteUUID: i.InviteUUID, TenantID: i.TenantID,
		InvitedEmail: i.InvitedEmail, Status: i.Status, ExpiresAt: i.ExpiresAt,
		CreatedAt: i.CreatedAt, UpdatedAt: i.UpdatedAt,
	}
}

func mapAuthnInvites(items []invite.Invite) []authn.Invite {
	out := make([]authn.Invite, len(items))
	for i := range items {
		out[i] = *toAuthnInvite(&items[i])
	}
	return out
}
