package event

import (
	"context"

	"github.com/maintainerd/auth/internal/platform/apperror"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"gorm.io/gorm"
)

// EventTypeService is the business-layer service for event_types (catalog).
type EventTypeService interface {
	ListActive(ctx context.Context) ([]EventTypeServiceDataResult, error)
	ListAll(ctx context.Context) ([]EventTypeServiceDataResult, error)
	ListByCategory(ctx context.Context, category string) ([]EventTypeServiceDataResult, error)
}

type eventTypeServiceImpl struct {
	eventTypeRepo EventTypeRepository
}

func NewEventTypeServiceImpl(eventTypeRepo EventTypeRepository) EventTypeService {
	return &eventTypeServiceImpl{eventTypeRepo: eventTypeRepo}
}

func (s *eventTypeServiceImpl) ListActive(ctx context.Context) ([]EventTypeServiceDataResult, error) {
	_, span := otel.Tracer("service").Start(ctx, "eventType.listActive")
	defer span.End()

	types, err := s.eventTypeRepo.FindAllActive()
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "list event types failed")
		return nil, err
	}

	result := make([]EventTypeServiceDataResult, len(types))
	for i, t := range types {
		result[i] = toServiceResult(&t)
	}
	span.SetStatus(codes.Ok, "")
	return result, nil
}

func (s *eventTypeServiceImpl) ListAll(ctx context.Context) ([]EventTypeServiceDataResult, error) {
	_, span := otel.Tracer("service").Start(ctx, "eventType.listAll")
	defer span.End()

	types, err := s.eventTypeRepo.FindAll()
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "list all event types failed")
		return nil, err
	}

	result := make([]EventTypeServiceDataResult, len(types))
	for i, t := range types {
		result[i] = toServiceResult(&t)
	}
	span.SetStatus(codes.Ok, "")
	return result, nil
}

func (s *eventTypeServiceImpl) ListByCategory(ctx context.Context, category string) ([]EventTypeServiceDataResult, error) {
	_, span := otel.Tracer("service").Start(ctx, "eventType.listByCategory")
	defer span.End()
	span.SetAttributes(attribute.String("category", category))

	types, err := s.eventTypeRepo.FindByCategory(category)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "list by category failed")
		return nil, err
	}

	result := make([]EventTypeServiceDataResult, len(types))
	for i, t := range types {
		result[i] = toServiceResult(&t)
	}
	span.SetStatus(codes.Ok, "")
	return result, nil
}

// TenantEventTypeConfigService manages per-tenant event type toggles (master switch).
type TenantEventTypeConfigService interface {
	GetByTenant(ctx context.Context, tenantID int64, tenantUUID string) ([]TenantEventTypeConfigResult, error)
	SetEnabled(ctx context.Context, tenantID int64, tenantUUID string, eventTypeUUID string, enabled bool) (*TenantEventTypeConfigResult, error)
}

type TenantEventTypeConfigResult struct {
	TenantEventTypeUUID string `json:"tenant_event_type_uuid"`
	TenantUUID          string `json:"tenant_uuid"`
	EventTypeUUID       string `json:"event_type_uuid"`
	EventTypeKey        string `json:"event_type_key"`
	Enabled             bool   `json:"enabled"`
	TenantID            int64  `json:"-"`
}

type tenantEventTypeConfigService struct {
	tenantEventTypeRepo TenantEventTypeRepository
	eventTypeRepo       EventTypeRepository
	writeGate           *WriteGate
	db                  *gorm.DB
}

func NewTenantEventTypeConfigService(
	db *gorm.DB,
	tenantEventTypeRepo TenantEventTypeRepository,
	eventTypeRepo EventTypeRepository,
	writeGate *WriteGate,
) TenantEventTypeConfigService {
	return &tenantEventTypeConfigService{
		tenantEventTypeRepo: tenantEventTypeRepo,
		eventTypeRepo:       eventTypeRepo,
		writeGate:           writeGate,
		db:                  db,
	}
}

func (s *tenantEventTypeConfigService) GetByTenant(ctx context.Context, tenantID int64, tenantUUID string) ([]TenantEventTypeConfigResult, error) {
	_, span := otel.Tracer("service").Start(ctx, "tenantEventTypeConfig.list")
	defer span.End()

	types, err := s.tenantEventTypeRepo.FindByTenantID(tenantID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "get tenant config failed")
		return nil, err
	}

	result := make([]TenantEventTypeConfigResult, len(types))
	for i, t := range types {
		key := ""
		eventTypeUUID := ""
		if et, _ := s.eventTypeRepo.FindByID(t.EventTypeID); et != nil {
			key = et.Key
			eventTypeUUID = et.EventTypeUUID.String()
		}
		result[i] = TenantEventTypeConfigResult{
			TenantEventTypeUUID: t.TenantEventTypeUUID.String(),
			TenantUUID:          tenantUUID,
			TenantID:            t.TenantID,
			EventTypeUUID:       eventTypeUUID,
			EventTypeKey:        key,
			Enabled:             t.Enabled,
		}
	}
	span.SetStatus(codes.Ok, "")
	return result, nil
}

func (s *tenantEventTypeConfigService) SetEnabled(ctx context.Context, tenantID int64, tenantUUID string, eventTypeUUID string, enabled bool) (*TenantEventTypeConfigResult, error) {
	_, span := otel.Tracer("service").Start(ctx, "tenantEventTypeConfig.set")
	defer span.End()
	span.SetAttributes(
		attribute.Int64("tenant.id", tenantID),
		attribute.String("event_type.uuid", eventTypeUUID),
		attribute.Bool("enabled", enabled),
	)

	et, err := s.eventTypeRepo.FindByUUID(eventTypeUUID)
	if err != nil || et == nil {
		span.SetStatus(codes.Error, "event type not found")
		return nil, apperror.NewNotFound("event type")
	}
	eventTypeID := et.EventTypeID

	existing, err := s.tenantEventTypeRepo.FindByTenantIDAndEventTypeID(tenantID, eventTypeID)
	if err != nil && err != gorm.ErrRecordNotFound {
		span.RecordError(err)
		span.SetStatus(codes.Error, "find tenant config failed")
		return nil, err
	}

	var result *TenantEventType

	if existing != nil {
		updated, err := s.tenantEventTypeRepo.UpdateByID(existing.TenantEventTypeID, map[string]any{"enabled": enabled})
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, "update tenant config failed")
			return nil, err
		}
		result = updated
	} else {
		tet := &TenantEventType{
			TenantID:    tenantID,
			EventTypeID: eventTypeID,
			Enabled:     enabled,
		}
		created, err := s.tenantEventTypeRepo.Create(tet)
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, "create tenant config failed")
			return nil, err
		}
		result = created
	}

	s.writeGate.InvalidateTenant(ctx, tenantID)
	span.SetStatus(codes.Ok, "")
	return &TenantEventTypeConfigResult{
		TenantEventTypeUUID: result.TenantEventTypeUUID.String(),
		TenantUUID:          tenantUUID,
		TenantID:            result.TenantID,
		EventTypeUUID:       et.EventTypeUUID.String(),
		EventTypeKey:        et.Key,
		Enabled:             result.Enabled,
	}, nil
}

func toServiceResult(et *EventType) EventTypeServiceDataResult {
	return EventTypeServiceDataResult{
		EventTypeUUID: et.EventTypeUUID,
		Key:           et.Key,
		Category:      et.Category,
		Description:   et.Description,
		Version:       et.Version,
		IsActive:      et.IsActive,
		CreatedAt:     et.CreatedAt,
		UpdatedAt:     et.UpdatedAt,
	}
}

// NoopTenantEventTypeConfigService returns a no-op implementation.
type noopTenantEventTypeConfigService struct{}

func (noopTenantEventTypeConfigService) GetByTenant(_ context.Context, _ int64, _ string) ([]TenantEventTypeConfigResult, error) {
	return nil, nil
}
func (noopTenantEventTypeConfigService) SetEnabled(_ context.Context, _ int64, _ string, _ string, _ bool) (*TenantEventTypeConfigResult, error) {
	return nil, nil
}

var _ TenantEventTypeConfigService = noopTenantEventTypeConfigService{}
