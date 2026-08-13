package app

import (
	"context"
	"sync"

	"github.com/google/uuid"
	"github.com/maintainerd/maintainerd-auth/internal/user"
)

// eventPublicIDResolver converts internal primary keys into public UUIDs before
// integration events leave Auth over webhooks or the broker.
type eventPublicIDResolver struct {
	tenantRefs *tenantRefResolverAdapter
	userRepo   user.UserRepository
	userCache  sync.Map // int64 -> uuid.UUID
}

func newEventPublicIDResolver(tenantRefs *tenantRefResolverAdapter, userRepo user.UserRepository) *eventPublicIDResolver {
	return &eventPublicIDResolver{
		tenantRefs: tenantRefs,
		userRepo:   userRepo,
	}
}

func (r *eventPublicIDResolver) TenantUUIDByID(ctx context.Context, id int64) (uuid.UUID, bool) {
	if r == nil || r.tenantRefs == nil {
		return uuid.Nil, false
	}
	return r.tenantRefs.TenantUUIDByID(ctx, id)
}

func (r *eventPublicIDResolver) UserUUIDByID(_ context.Context, id int64) (uuid.UUID, bool) {
	if r == nil || r.userRepo == nil || id <= 0 {
		return uuid.Nil, false
	}
	if v, ok := r.userCache.Load(id); ok {
		return v.(uuid.UUID), true
	}
	u, err := r.userRepo.FindByID(id)
	if err != nil || u == nil || u.UserUUID == uuid.Nil {
		return uuid.Nil, false
	}
	r.userCache.Store(id, u.UserUUID)
	return u.UserUUID, true
}
