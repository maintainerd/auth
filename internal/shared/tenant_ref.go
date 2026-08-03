package shared

import (
	"context"

	"github.com/google/uuid"
)

// TenantRefResolver maps between a tenant's INTERNAL id (bigint PK) and its
// EXTERNAL uuid. Tokens carry the opaque uuid in the `tenant_id` claim (never the
// internal PK — least-disclosure, RFC 9068), and the auth layer resolves it back
// to the internal id for scoping. Injected once at startup via
// SetTenantRefResolver so the token-mint and JWT-parse layers never need to
// import the tenant repository. The id<->uuid mapping is immutable, so
// implementations should cache aggressively.
type TenantRefResolver interface {
	TenantUUIDByID(ctx context.Context, id int64) (uuid.UUID, bool)
	TenantIDByUUID(ctx context.Context, u uuid.UUID) (int64, bool)
}

var tenantRefResolver TenantRefResolver

// SetTenantRefResolver installs the tenant id<->uuid resolver. Call once during
// app startup before serving requests.
func SetTenantRefResolver(r TenantRefResolver) { tenantRefResolver = r }

// TenantUUIDByID returns the external uuid for an internal tenant id, or uuid.Nil
// when unresolved (no resolver wired, or the id is unknown/non-positive). Used at
// token mint to stamp the `tenant_id` claim.
func TenantUUIDByID(ctx context.Context, id int64) uuid.UUID {
	if tenantRefResolver == nil || id <= 0 {
		return uuid.Nil
	}
	u, _ := tenantRefResolver.TenantUUIDByID(ctx, id)
	return u
}

// TenantUUIDStringByID returns the tenant's external uuid as a string for the
// `tenant_id` claim, or "" when unresolved (so the mint layer can omit a bogus
// claim rather than stamp a zero uuid).
func TenantUUIDStringByID(ctx context.Context, id int64) string {
	u := TenantUUIDByID(ctx, id)
	if u == uuid.Nil {
		return ""
	}
	return u.String()
}

// TenantIDByUUID returns the internal tenant id for an external uuid, or 0 when
// unresolved. Used at JWT parse to resolve the `tenant_id` claim back to the id
// every scoping check expects.
func TenantIDByUUID(ctx context.Context, u uuid.UUID) int64 {
	if tenantRefResolver == nil || u == uuid.Nil {
		return 0
	}
	id, _ := tenantRefResolver.TenantIDByUUID(ctx, u)
	return id
}

// TenantIDByUUIDString resolves a uuid string claim to the internal tenant id, or
// 0 when the string is empty/invalid or unresolved.
func TenantIDByUUIDString(ctx context.Context, s string) int64 {
	if s == "" {
		return 0
	}
	u, err := uuid.Parse(s)
	if err != nil {
		return 0
	}
	return TenantIDByUUID(ctx, u)
}
