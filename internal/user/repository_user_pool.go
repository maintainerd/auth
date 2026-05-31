package user

import (
	"errors"

	"github.com/maintainerd/auth/internal/platform/database"
	"gorm.io/gorm"
)

// UserPoolRepository provides read and write access to the user_pools table.
// A user pool is the isolation namespace for users, roles, and settings within
// a single tenant deployment.
type UserPoolRepository interface {
	BaseRepositoryMethods[UserPool]
	WithTx(tx *gorm.DB) UserPoolRepository
	FindByIdentifier(tenantID int64, identifier string) (*UserPool, error)
	FindDefault(tenantID int64) (*UserPool, error)
	FindSystem(tenantID int64) (*UserPool, error)
	FindAllByTenantID(tenantID int64) ([]UserPool, error)
}

type userPoolRepository struct {
	*BaseRepository[UserPool]
}

// NewUserPoolRepository returns a UserPoolRepository backed by the given gorm.DB.
func NewUserPoolRepository(db *gorm.DB) UserPoolRepository {
	return &userPoolRepository{
		BaseRepository: database.NewBaseRepository[UserPool](db, "user_pool_uuid", "user_pool_id"),
	}
}

// WithTx returns a new UserPoolRepository scoped to the given transaction.
func (r *userPoolRepository) WithTx(tx *gorm.DB) UserPoolRepository {
	return &userPoolRepository{
		BaseRepository: r.BaseRepository.WithTx(tx),
	}
}

// FindByIdentifier retrieves a user pool by its slug within a tenant.
func (r *userPoolRepository) FindByIdentifier(tenantID int64, identifier string) (*UserPool, error) {
	var pool UserPool
	err := r.DB().
		Where("tenant_id = ? AND identifier = ? AND deleted_at IS NULL", tenantID, identifier).
		First(&pool).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &pool, nil
}

// FindDefault retrieves the default user pool for the given tenant.
func (r *userPoolRepository) FindDefault(tenantID int64) (*UserPool, error) {
	var pool UserPool
	err := r.DB().
		Where("tenant_id = ? AND is_default = ? AND deleted_at IS NULL", tenantID, true).
		First(&pool).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &pool, nil
}

// FindSystem retrieves the system user pool for the given tenant.
func (r *userPoolRepository) FindSystem(tenantID int64) (*UserPool, error) {
	var pool UserPool
	err := r.DB().
		Where("tenant_id = ? AND is_system = ? AND deleted_at IS NULL", tenantID, true).
		First(&pool).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &pool, nil
}

// FindAllByTenantID retrieves all non-deleted user pools belonging to a tenant.
func (r *userPoolRepository) FindAllByTenantID(tenantID int64) ([]UserPool, error) {
	var pools []UserPool
	err := r.DB().
		Where("tenant_id = ? AND deleted_at IS NULL", tenantID).
		Find(&pools).Error
	return pools, err
}
