package authctx

import (
	"time"

	"github.com/google/uuid"
)

// UserContext is the cached/request auth context for an authenticated subject.
type UserContext struct {
	User     *AuthUser     `json:"user"`
	Tenant   *AuthTenant   `json:"tenant"`
	Provider *AuthProvider `json:"provider"`
	Client   *AuthClient   `json:"client"`
}

// AuthContext holds the authenticated principal and their associated tenant,
// identity provider, and client for request handling.
type AuthContext struct {
	User     *AuthUser
	Tenant   *AuthTenant
	Provider *AuthProvider
	Client   *AuthClient
}

// AuthProfile holds optional profile fields cached alongside the user context.
type AuthProfile struct {
	DisplayName *string `json:"display_name,omitempty"`
	FirstName   string  `json:"first_name,omitempty"`
	LastName    *string `json:"last_name,omitempty"`
	ProfileURL  *string `json:"profile_url,omitempty"`
}

type AuthUser struct {
	UserID   int64     `json:"user_id"`
	UserUUID uuid.UUID `json:"user_uuid"`
	// Status is carried so every authenticated request can refuse a user who is
	// no longer active. Deactivating, suspending or soft-deleting an account has
	// to take effect on the next request — without this it only took effect when
	// the access token happened to expire, leaving a disabled account fully
	// usable for the remainder of its token lifetime.
	Status          string       `json:"status,omitempty"`
	Roles           []AuthRole   `json:"roles,omitempty"`
	Email           string       `json:"email,omitempty"`
	IsEmailVerified bool         `json:"is_email_verified,omitempty"`
	Phone           string       `json:"phone,omitempty"`
	IsPhoneVerified bool         `json:"is_phone_verified,omitempty"`
	Fullname        string       `json:"fullname,omitempty"`
	UpdatedAt       time.Time    `json:"updated_at,omitempty"`
	Profile         *AuthProfile `json:"profile,omitempty"`
}

type AuthRole struct {
	RoleID      int64            `json:"role_id"`
	RoleUUID    uuid.UUID        `json:"role_uuid"`
	Name        string           `json:"name"`
	Permissions []AuthPermission `json:"permissions,omitempty"`
}

type AuthPermission struct {
	PermissionID   int64     `json:"permission_id"`
	PermissionUUID uuid.UUID `json:"permission_uuid"`
	Name           string    `json:"name"`
}

type AuthTenant struct {
	TenantID    int64     `json:"tenant_id"`
	TenantUUID  uuid.UUID `json:"tenant_uuid"`
	Name        string    `json:"name"`
	DisplayName string    `json:"display_name"`
	Identifier  string    `json:"identifier"`
}

type AuthProvider struct {
	IdentityProviderID   int64     `json:"identity_provider_id"`
	IdentityProviderUUID uuid.UUID `json:"identity_provider_uuid"`
}

type AuthClient struct {
	ClientID   int64     `json:"client_id"`
	ClientUUID uuid.UUID `json:"client_uuid"`
	Identifier *string   `json:"identifier,omitempty"`
}
