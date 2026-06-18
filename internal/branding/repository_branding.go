package branding

import (
	"errors"

	"github.com/maintainerd/auth/internal/platform/database"
	"gorm.io/gorm"
)

// BrandingRepository defines persistence operations for the branding entity.
type BrandingRepository interface {
	BaseRepositoryMethods[Branding]
	FindByUUID(uuid any, preloads ...string) (*Branding, error)
	DeleteByUUID(uuid any) error
	WithTx(tx *gorm.DB) BrandingRepository
	FindByTenantID(tenantID int64) (*Branding, error)
	FindAllByTenantID(tenantID int64) ([]Branding, error)
	FindActive(tenantID int64) (*Branding, error)
	FindSystem(tenantID int64) (*Branding, error)
	DeactivateAll(tenantID int64) error
}

type brandingRepository struct {
	*BaseRepository[Branding]
}

// NewBrandingRepository creates a new BrandingRepository backed by the given
// database connection.
func NewBrandingRepository(db *gorm.DB) BrandingRepository {
	return &brandingRepository{
		BaseRepository: database.NewBaseRepository[Branding](db, "branding_uuid", "branding_id"),
	}
}

// WithTx returns a copy of the repository bound to the supplied transaction.
func (r *brandingRepository) WithTx(tx *gorm.DB) BrandingRepository {
	return &brandingRepository{
		BaseRepository: r.BaseRepository.WithTx(tx),
	}
}

// FindByTenantID retrieves the FIRST branding record for a tenant. Returns
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

// FindAllByTenantID returns every branding record for a tenant, system records
// first, then newest. Used by the admin branding list.
func (r *brandingRepository) FindAllByTenantID(tenantID int64) ([]Branding, error) {
	var brandings []Branding
	err := r.DB().
		Where("tenant_id = ? AND deleted_at IS NULL", tenantID).
		Order("is_system DESC, created_at ASC").
		Find(&brandings).Error
	return brandings, err
}

// FindActive returns the active branding for a tenant, or falls back to the
// system branding if no custom active record exists.
func (r *brandingRepository) FindActive(tenantID int64) (*Branding, error) {
	var b Branding
	err := r.DB().
		Where("tenant_id = ? AND is_active = ? AND deleted_at IS NULL", tenantID, true).
		First(&b).Error
	if err == nil {
		return &b, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	return r.FindSystem(tenantID)
}

// FindSystem returns the immutable system branding record.
func (r *brandingRepository) FindSystem(tenantID int64) (*Branding, error) {
	var b Branding
	err := r.DB().
		Where("tenant_id = ? AND is_system = ? AND deleted_at IS NULL", tenantID, true).
		First(&b).Error
	if err != nil {
		return nil, err
	}
	return &b, nil
}

// DeactivateAll sets is_active=false for ALL branding records of the given
// tenant (including system records) so a different one can become active —
// system themes (e.g. light/dark) are switchable, just not deletable.
func (r *brandingRepository) DeactivateAll(tenantID int64) error {
	return r.DB().
		Model(&Branding{}).
		Where("tenant_id = ?", tenantID).
		Update("is_active", false).Error
}
