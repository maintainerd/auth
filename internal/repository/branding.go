package repository

import (
	"errors"

	"github.com/maintainerd/auth/internal/model"
	"gorm.io/gorm"
)

// BrandingRepository defines persistence operations for the branding entity.
type BrandingRepository interface {
	BaseRepositoryMethods[model.Branding]
	WithTx(tx *gorm.DB) BrandingRepository
	FindByTenantID(tenantID int64) (*model.Branding, error)
}

type brandingRepository struct {
	*BaseRepository[model.Branding]
}

// NewBrandingRepository creates a new BrandingRepository backed by the given
// database connection.
func NewBrandingRepository(db *gorm.DB) BrandingRepository {
	return &brandingRepository{
		BaseRepository: NewBaseRepository[model.Branding](db, "branding_uuid", "branding_id"),
	}
}

// WithTx returns a copy of the repository bound to the supplied transaction.
func (r *brandingRepository) WithTx(tx *gorm.DB) BrandingRepository {
	return &brandingRepository{
		BaseRepository: r.BaseRepository.WithTx(tx),
	}
}

// FindByTenantID retrieves the single branding record for a tenant. Returns
// nil, nil when no record exists.
func (r *brandingRepository) FindByTenantID(tenantID int64) (*model.Branding, error) {
	var branding model.Branding
	err := r.DB().Where("tenant_id = ?", tenantID).First(&branding).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &branding, nil
}
