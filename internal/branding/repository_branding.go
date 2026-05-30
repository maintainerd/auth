package branding

import (
	"errors"
	"gorm.io/gorm"
)

// BrandingRepository defines persistence operations for the branding entity.
type BrandingRepository interface {
	BaseRepositoryMethods[Branding]
	WithTx(tx *gorm.DB) BrandingRepository
	FindByTenantID(tenantID int64) (*Branding, error)
}

type brandingRepository struct {
	*BaseRepository[Branding]
}

// NewBrandingRepository creates a new BrandingRepository backed by the given
// database connection.
func NewBrandingRepository(db *gorm.DB) BrandingRepository {
	return &brandingRepository{
		BaseRepository: NewBaseRepository[Branding](db, "branding_uuid", "branding_id"),
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
func (r *brandingRepository) FindByTenantID(tenantID int64) (*Branding, error) {
	var branding Branding
	err := r.DB().Where("tenant_id = ?", tenantID).First(&branding).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &branding, nil
}
