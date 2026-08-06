package app

import (
	"github.com/maintainerd/maintainerd-auth/internal/authn"
	"github.com/maintainerd/maintainerd-auth/internal/user"
)

func toAuthnUser(u *user.User) *authn.User {
	if u == nil {
		return nil
	}
	return &authn.User{
		UserID:                     u.UserID,
		UserUUID:                   u.UserUUID,
		TenantID:                   u.TenantID,
		Username:                   u.Username,
		Fullname:                   u.Fullname,
		Email:                      u.Email,
		Phone:                      u.Phone,
		Password:                   u.Password,
		IsEmailVerified:            u.IsEmailVerified,
		IsPhoneVerified:            u.IsPhoneVerified,
		Status:                     u.Status,
		ForcePasswordChange:        u.ForcePasswordChange,
		PasswordChangedAt:          u.PasswordChangedAt,
		TemporaryPasswordExpiresAt: u.TemporaryPasswordExpiresAt,
		IsTOTPEnabled:              u.IsTOTPEnabled,
		IsWebAuthnEnabled:          u.IsWebAuthnEnabled,
		FirstMFAEnrolledAt:         u.FirstMFAEnrolledAt,
		CreatedAt:                  u.CreatedAt,
		UpdatedAt:                  u.UpdatedAt,
	}
}

func toUserUser(u *authn.User) *user.User {
	if u == nil {
		return nil
	}
	return &user.User{
		UserID:                     u.UserID,
		UserUUID:                   u.UserUUID,
		TenantID:                   u.TenantID,
		Username:                   u.Username,
		Fullname:                   u.Fullname,
		Email:                      u.Email,
		Phone:                      u.Phone,
		Password:                   u.Password,
		IsEmailVerified:            u.IsEmailVerified,
		IsPhoneVerified:            u.IsPhoneVerified,
		Status:                     u.Status,
		ForcePasswordChange:        u.ForcePasswordChange,
		PasswordChangedAt:          u.PasswordChangedAt,
		TemporaryPasswordExpiresAt: u.TemporaryPasswordExpiresAt,
		IsTOTPEnabled:              u.IsTOTPEnabled,
		IsWebAuthnEnabled:          u.IsWebAuthnEnabled,
		FirstMFAEnrolledAt:         u.FirstMFAEnrolledAt,
		CreatedAt:                  u.CreatedAt,
		UpdatedAt:                  u.UpdatedAt,
	}
}

func mapAuthnUsers(items []user.User) []authn.User {
	out := make([]authn.User, len(items))
	for i := range items {
		out[i] = *toAuthnUser(&items[i])
	}
	return out
}

func toAuthnUserIdentity(u *user.UserIdentity) *authn.UserIdentity {
	if u == nil {
		return nil
	}
	return &authn.UserIdentity{
		UserIdentityID: u.UserIdentityID, UserIdentityUUID: u.UserIdentityUUID,
		TenantID: u.TenantID, UserID: u.UserID,
		IdentityProviderID: u.IdentityProviderID,
		Provider:           u.Provider, Sub: u.Sub, Metadata: u.Metadata,
		CreatedAt: u.CreatedAt, UpdatedAt: u.UpdatedAt,
	}
}

func toUserUserIdentity(u *authn.UserIdentity) *user.UserIdentity {
	if u == nil {
		return nil
	}
	return &user.UserIdentity{
		UserIdentityID: u.UserIdentityID, UserIdentityUUID: u.UserIdentityUUID,
		TenantID: u.TenantID, UserID: u.UserID,
		IdentityProviderID: u.IdentityProviderID,
		Provider:           u.Provider, Sub: u.Sub, Metadata: u.Metadata,
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
