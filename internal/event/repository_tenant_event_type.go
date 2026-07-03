package event

import (
	"github.com/maintainerd/maintainerd-auth/internal/platform/database"
	"gorm.io/gorm"
)

// TenantEventTypeRepository defines persistence operations for tenant_event_types.
type TenantEventTypeRepository interface {
	BaseRepositoryMethods[TenantEventType]
	UpdateByID(id any, updatedData any) (*TenantEventType, error)
	FindByTenantID(tenantID int64) ([]TenantEventType, error)
	FindByTenantIDAndEventTypeID(tenantID, eventTypeID int64) (*TenantEventType, error)
	FindDisabledByTenantID(tenantID int64) ([]TenantEventType, error)
	FindDisabledKeysByTenantID(tenantID int64) ([]string, error)
	WithTx(tx *gorm.DB) TenantEventTypeRepository
}

type tenantEventTypeRepository struct {
	*BaseRepository[TenantEventType]
}

func NewTenantEventTypeRepository(db *gorm.DB) TenantEventTypeRepository {
	return &tenantEventTypeRepository{
		BaseRepository: database.NewBaseRepository[TenantEventType](db, "tenant_event_type_uuid", "tenant_event_type_id"),
	}
}

func (r *tenantEventTypeRepository) WithTx(tx *gorm.DB) TenantEventTypeRepository {
	return &tenantEventTypeRepository{
		BaseRepository: r.BaseRepository.WithTx(tx),
	}
}

func (r *tenantEventTypeRepository) FindByTenantID(tenantID int64) ([]TenantEventType, error) {
	var types []TenantEventType
	err := r.DB().Where("tenant_id = ?", tenantID).Find(&types).Error
	return types, err
}

func (r *tenantEventTypeRepository) FindByTenantIDAndEventTypeID(tenantID, eventTypeID int64) (*TenantEventType, error) {
	var tet TenantEventType
	err := r.DB().Where("tenant_id = ? AND event_type_id = ?", tenantID, eventTypeID).First(&tet).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &tet, nil
}

func (r *tenantEventTypeRepository) FindDisabledByTenantID(tenantID int64) ([]TenantEventType, error) {
	var types []TenantEventType
	err := r.DB().Where("tenant_id = ? AND enabled = ?", tenantID, false).Find(&types).Error
	return types, err
}

func (r *tenantEventTypeRepository) FindDisabledKeysByTenantID(tenantID int64) ([]string, error) {
	var keys []string
	err := r.DB().Table("tenant_event_types").
		Select("event_types.key").
		Joins("JOIN event_types ON event_types.event_type_id = tenant_event_types.event_type_id").
		Where("tenant_event_types.tenant_id = ? AND tenant_event_types.enabled = ?", tenantID, false).
		Pluck("event_types.key", &keys).Error
	return keys, err
}
