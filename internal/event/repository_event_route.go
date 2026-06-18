package event

import (
	"github.com/maintainerd/auth/internal/platform/database"
	"gorm.io/gorm"
)

// EventRouteRepository defines persistence operations for event_routes.
type EventRouteRepository interface {
	BaseRepositoryMethods[EventRoute]
	FindByUUID(uuid any, preloads ...string) (*EventRoute, error)
	UpdateByUUID(uuid any, updatedData any) (*EventRoute, error)
	DeleteByUUID(uuid any) error
	FindByTenantID(tenantID int64) ([]EventRoute, error)
	FindEnabledByTenantID(tenantID int64) ([]EventRoute, error)
	FindByTenantIDAndEventTypeID(tenantID, eventTypeID int64) (*EventRoute, error)
	WithTx(tx *gorm.DB) EventRouteRepository
}

type eventRouteRepository struct {
	*BaseRepository[EventRoute]
}

func NewEventRouteRepository(db *gorm.DB) EventRouteRepository {
	return &eventRouteRepository{
		BaseRepository: database.NewBaseRepository[EventRoute](db, "event_route_uuid", "event_route_id"),
	}
}

func (r *eventRouteRepository) WithTx(tx *gorm.DB) EventRouteRepository {
	return &eventRouteRepository{
		BaseRepository: r.BaseRepository.WithTx(tx),
	}
}

func (r *eventRouteRepository) FindByTenantID(tenantID int64) ([]EventRoute, error) {
	var routes []EventRoute
	err := r.DB().Where("tenant_id = ?", tenantID).Order("created_at ASC").Find(&routes).Error
	return routes, err
}

func (r *eventRouteRepository) FindEnabledByTenantID(tenantID int64) ([]EventRoute, error) {
	var routes []EventRoute
	err := r.DB().Where("tenant_id = ? AND enabled = ?", tenantID, true).Find(&routes).Error
	return routes, err
}

func (r *eventRouteRepository) FindByTenantIDAndEventTypeID(tenantID, eventTypeID int64) (*EventRoute, error) {
	var route EventRoute
	err := r.DB().Where("tenant_id = ? AND event_type_id = ?", tenantID, eventTypeID).First(&route).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &route, nil
}
