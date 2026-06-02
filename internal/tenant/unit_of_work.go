package tenant

import (
	"context"

	"gorm.io/gorm"
)

// Transaction exposes tenant repositories scoped to a single transaction.
type Transaction interface {
	TenantRepository() TenantRepository
	TenantMemberRepository() TenantMemberRepository
	DeleteTenantCascade(ctx context.Context, tenantID int64) error
}

// UnitOfWork wraps transaction management so services do not depend on GORM.
type UnitOfWork interface {
	Do(ctx context.Context, fn func(tx Transaction) error) error
}

type directUnitOfWork struct {
	tenantRepo       TenantRepository
	tenantMemberRepo TenantMemberRepository
}

func newDirectUnitOfWork(tenantRepo TenantRepository, tenantMemberRepo TenantMemberRepository) UnitOfWork {
	return &directUnitOfWork{
		tenantRepo:       tenantRepo,
		tenantMemberRepo: tenantMemberRepo,
	}
}

func (u *directUnitOfWork) Do(ctx context.Context, fn func(tx Transaction) error) error {
	return fn(&directTransaction{
		tenantRepo:       u.tenantRepo,
		tenantMemberRepo: u.tenantMemberRepo,
	})
}

type directTransaction struct {
	tenantRepo       TenantRepository
	tenantMemberRepo TenantMemberRepository
}

func (t *directTransaction) TenantRepository() TenantRepository {
	return t.tenantRepo
}

func (t *directTransaction) TenantMemberRepository() TenantMemberRepository {
	return t.tenantMemberRepo
}

func (t *directTransaction) DeleteTenantCascade(context.Context, int64) error {
	return nil
}

type gormUnitOfWork struct {
	db               *gorm.DB
	tenantRepo       TenantRepository
	tenantMemberRepo TenantMemberRepository
	cascadeModels    []any
}

func NewGormUnitOfWork(db *gorm.DB, tenantRepo TenantRepository, tenantMemberRepo TenantMemberRepository, cascadeModels []any) UnitOfWork {
	return &gormUnitOfWork{
		db:               db,
		tenantRepo:       tenantRepo,
		tenantMemberRepo: tenantMemberRepo,
		cascadeModels:    cascadeModels,
	}
}

func (u *gormUnitOfWork) Do(ctx context.Context, fn func(tx Transaction) error) error {
	if u == nil || u.db == nil {
		if u == nil {
			return fn(&directTransaction{})
		}
		return fn(&directTransaction{
			tenantRepo:       u.tenantRepo,
			tenantMemberRepo: u.tenantMemberRepo,
		})
	}
	return u.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return fn(&gormTransaction{
			tx:               tx,
			tenantRepo:       u.tenantRepo,
			tenantMemberRepo: u.tenantMemberRepo,
			cascadeModels:    u.cascadeModels,
		})
	})
}

type gormTransaction struct {
	tx               *gorm.DB
	tenantRepo       TenantRepository
	tenantMemberRepo TenantMemberRepository
	cascadeModels    []any
}

func (t *gormTransaction) TenantRepository() TenantRepository {
	return t.tenantRepo.WithTx(t.tx)
}

func (t *gormTransaction) TenantMemberRepository() TenantMemberRepository {
	return t.tenantMemberRepo.WithTx(t.tx)
}

func (t *gormTransaction) DeleteTenantCascade(ctx context.Context, tenantID int64) error {
	return t.tenantRepo.DeleteCascade(ctx, t.tx, tenantID, t.cascadeModels)
}
