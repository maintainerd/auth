package app

import (
	"context"
	"sync"

	"github.com/google/uuid"
	"github.com/maintainerd/maintainerd-auth/internal/tenant"
)

// tenantRefResolverAdapter implements shared.TenantRefResolver over the tenant
// repository with an in-memory cache. The internal-id <-> external-uuid mapping
// is immutable for a tenant's lifetime, so cached entries never need to expire
// (a purged tenant simply stops being looked up). This keeps the token-mint and
// JWT-parse hot paths off the database after the first resolution per tenant.
type tenantRefResolverAdapter struct {
	repo     tenant.TenantRepository
	idToUUID sync.Map // int64 -> uuid.UUID
	uuidToID sync.Map // uuid.UUID -> int64
}

func newTenantRefResolver(repo tenant.TenantRepository) *tenantRefResolverAdapter {
	return &tenantRefResolverAdapter{repo: repo}
}

func (r *tenantRefResolverAdapter) TenantUUIDByID(_ context.Context, id int64) (uuid.UUID, bool) {
	if id <= 0 {
		return uuid.Nil, false
	}
	if v, ok := r.idToUUID.Load(id); ok {
		return v.(uuid.UUID), true
	}
	t, err := r.repo.FindByID(id)
	if err != nil || t == nil {
		return uuid.Nil, false
	}
	r.cache(t.TenantID, t.TenantUUID)
	return t.TenantUUID, true
}

func (r *tenantRefResolverAdapter) TenantIDByUUID(_ context.Context, u uuid.UUID) (int64, bool) {
	if u == uuid.Nil {
		return 0, false
	}
	if v, ok := r.uuidToID.Load(u); ok {
		return v.(int64), true
	}
	t, err := r.repo.FindByUUID(u)
	if err != nil || t == nil {
		return 0, false
	}
	r.cache(t.TenantID, t.TenantUUID)
	return t.TenantID, true
}

func (r *tenantRefResolverAdapter) cache(id int64, u uuid.UUID) {
	r.idToUUID.Store(id, u)
	r.uuidToID.Store(u, id)
}
