package tenant

import (
	"context"
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// TenantSeeder seeds the per-tenant baseline (roles, permissions, client, idp,
// branding, etc.) for a newly created tenant. It is implemented by an app-layer
// adapter over internal/setup/seeder; the tenant package cannot call the seeder
// directly because the seeder imports tenant (import cycle). The seed runs
// inside the tenant-creation transaction, so it receives the active *gorm.DB.
type TenantSeeder interface {
	SeedTenant(ctx context.Context, tx *gorm.DB, tenantID int64) error
}

// Consumer-defined interfaces and projection types for upstream domains.
// tenant declares the shape of data it needs from the user domain so it does
// not import the user package directly (avoiding an import cycle). The user
// domain provides adapters that satisfy these interfaces; wiring injects them.

// MemberUser is tenant's projection of a user, used when listing tenant members.
type MemberUser struct {
	UserID          int64
	UserUUID        uuid.UUID
	Username        string
	Fullname        string
	Email           string
	Phone           string
	IsEmailVerified bool
	IsPhoneVerified bool
	Status          string
	Metadata        datatypes.JSON
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

// UserReader is the subset of the user domain that tenant needs to resolve
// member user details.
type UserReader interface {
	FindByUUID(userUUID uuid.UUID) (*MemberUser, error)
	FindByID(userID int64) (*MemberUser, error)
}

// UserProvisioner copies a user record into a target tenant with the same
// credentials. It is used when adding a member to a tenant: if the user does
// not already have a record in the target tenant, a copy is created so the
// user can log in there with the same credentials.
type UserProvisioner interface {
	EnsureUserInTenant(ctx context.Context, userUUID uuid.UUID, targetTenantID int64) (userID int64, err error)
	// GrantRoleByName and RevokeRoleByName participate in the tenant membership
	// transaction so ownership and its matching IAM role cannot drift apart.
	GrantRoleByName(ctx context.Context, tx *gorm.DB, userID, tenantID int64, roleName string) error
	RevokeRoleByName(ctx context.Context, tx *gorm.DB, userID, tenantID int64, roleName string) error
}

// SurfaceClient is tenant's projection of a client's public fields, used by the
// domain-bootstrap endpoint to advertise the seeded system client for a surface.
type SurfaceClient struct {
	ClientID    string
	Name        string
	DisplayName string
	ClientType  string
}

// SurfaceClientReader resolves the seeded system client for a tenant on a given
// frontend surface ("identity" or "console"). It is implemented by an app-layer
// adapter over the client service so the tenant package does not import client
// (avoiding a cross-domain dependency). tenantName is the DNS slug; surface is
// one of shared.FrontendSurfaceIdentity / shared.FrontendSurfaceConsole.
type SurfaceClientReader interface {
	GetSurfaceClient(ctx context.Context, tenantName, surface string) (*SurfaceClient, error)
}

// AccessActor exposes the tenant-relevant identity data needed for access
// control checks. The user domain implements this on its User.
type AccessActor interface {
	AccessIdentities() []AccessIdentity
}

// AccessIdentity is one of an actor's identities with the tenant facts needed
// for access control.
type AccessIdentity struct {
	TenantID       int64
	TenantIsSystem bool
}
