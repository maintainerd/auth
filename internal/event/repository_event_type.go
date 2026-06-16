package event

import (
	"github.com/maintainerd/auth/internal/platform/database"
	"gorm.io/gorm"
)

// EventTypeRepository defines persistence operations for event_types.
type EventTypeRepository interface {
	BaseRepositoryMethods[EventType]
	FindAllActive() ([]EventType, error)
	FindByKey(key string) (*EventType, error)
	// FindByKeyAndTenantID scopes the lookup to a tenant. Event types are
	// tenant-scoped, so the same key exists in every tenant — use this whenever
	// a tenant is known to avoid resolving another tenant's event type.
	FindByKeyAndTenantID(key string, tenantID int64) (*EventType, error)
	FindByKeys(keys []string) ([]EventType, error)
	FindByCategory(category string) ([]EventType, error)
	WithTx(tx *gorm.DB) EventTypeRepository
}

type eventTypeRepository struct {
	*BaseRepository[EventType]
}

func NewEventTypeRepository(db *gorm.DB) EventTypeRepository {
	return &eventTypeRepository{
		BaseRepository: database.NewBaseRepository[EventType](db, "event_type_uuid", "event_type_id"),
	}
}

func (r *eventTypeRepository) WithTx(tx *gorm.DB) EventTypeRepository {
	return &eventTypeRepository{
		BaseRepository: r.BaseRepository.WithTx(tx),
	}
}

func (r *eventTypeRepository) FindAllActive() ([]EventType, error) {
	var types []EventType
	err := r.DB().Where("is_active = ?", true).Order("key ASC").Find(&types).Error
	return types, err
}

func (r *eventTypeRepository) FindByKey(key string) (*EventType, error) {
	var et EventType
	err := r.DB().Where("key = ?", key).First(&et).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &et, nil
}

func (r *eventTypeRepository) FindByKeyAndTenantID(key string, tenantID int64) (*EventType, error) {
	var et EventType
	err := r.DB().Where("key = ? AND tenant_id = ?", key, tenantID).First(&et).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &et, nil
}

func (r *eventTypeRepository) FindByKeys(keys []string) ([]EventType, error) {
	var types []EventType
	err := r.DB().Where("key IN ?", keys).Find(&types).Error
	return types, err
}

func (r *eventTypeRepository) FindByCategory(category string) ([]EventType, error) {
	var types []EventType
	err := r.DB().Where("category = ? AND is_active = ?", category, true).Order("key ASC").Find(&types).Error
	return types, err
}
