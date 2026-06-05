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
	GetByTenant(ctx context.Context, tenantID int64) ([]TenantEventTypeConfigResult, error)
	SetEnabled(ctx context.Context, tenantID int64, eventTypeID int64, enabled bool) (*TenantEventTypeConfigResult, error)
}

type TenantEventTypeConfigResult struct {
	TenantEventTypeUUID string
	TenantID            int64
	EventTypeID         int64
	EventTypeKey        string
	Enabled             bool
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

func (s *tenantEventTypeConfigService) GetByTenant(ctx context.Context, tenantID int64) ([]TenantEventTypeConfigResult, error) {
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
		if et, _ := s.eventTypeRepo.FindByID(t.EventTypeID); et != nil {
			key = et.Key
		}
		result[i] = TenantEventTypeConfigResult{
			TenantEventTypeUUID: t.TenantEventTypeUUID.String(),
			TenantID:            t.TenantID,
			EventTypeID:         t.EventTypeID,
			EventTypeKey:        key,
			Enabled:             t.Enabled,
		}
	}
	span.SetStatus(codes.Ok, "")
	return result, nil
}

func (s *tenantEventTypeConfigService) SetEnabled(ctx context.Context, tenantID int64, eventTypeID int64, enabled bool) (*TenantEventTypeConfigResult, error) {
	_, span := otel.Tracer("service").Start(ctx, "tenantEventTypeConfig.set")
	defer span.End()
	span.SetAttributes(
		attribute.Int64("tenant.id", tenantID),
		attribute.Int64("event_type.id", eventTypeID),
		attribute.Bool("enabled", enabled),
	)

	et, err := s.eventTypeRepo.FindByID(eventTypeID)
	if err != nil || et == nil {
		span.SetStatus(codes.Error, "event type not found")
		return nil, apperror.NewNotFound("event type")
	}

	existing, err := s.tenantEventTypeRepo.FindByTenantIDAndEventTypeID(tenantID, eventTypeID)
	if err != nil && err != gorm.ErrRecordNotFound {
		span.RecordError(err)
		span.SetStatus(codes.Error, "find tenant config failed")
		return nil, err
	}

	var result *TenantEventType

	if existing != nil {
		existing.Enabled = enabled
		updated, err := s.tenantEventTypeRepo.UpdateByID(existing.TenantEventTypeID, existing)
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
		TenantID:            result.TenantID,
		EventTypeID:         result.EventTypeID,
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

func (noopTenantEventTypeConfigService) GetByTenant(_ context.Context, _ int64) ([]TenantEventTypeConfigResult, error) {
	return nil, nil
}
func (noopTenantEventTypeConfigService) SetEnabled(_ context.Context, _ int64, _ int64, _ bool) (*TenantEventTypeConfigResult, error) {
	return nil, nil
}

var _ TenantEventTypeConfigService = noopTenantEventTypeConfigService{}
