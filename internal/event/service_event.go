package event

import (
	"context"
	"log/slog"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// EventService emits integration events to the transactional outbox.
type EventService interface {
	// Emit writes a new integration event to the outbox inside the given transaction.
	// The write gate is checked before writing. Returns the event if emitted, nil if gated.
	// Callers must pass a tx that is part of their mutation transaction.
	Emit(ctx context.Context, tx *gorm.DB, event *IntegrationEvent) (*IntegrationEvent, error)

	// Shutdown stops the outbox relay and any background workers.
	Shutdown()
}

type eventService struct {
	outboxRepo  OutboxRepository
	writeGate   *WriteGate
	relay       *Relay
}

// NewEventService creates a new EventService.
func NewEventService(
	outboxRepo OutboxRepository,
	writeGate *WriteGate,
	relay *Relay,
) EventService {
	return &eventService{
		outboxRepo: outboxRepo,
		writeGate:  writeGate,
		relay:      relay,
	}
}

// Emit writes a new integration event to the outbox inside the given transaction.
// The write gate is checked first — if the gate is closed, nothing is written.
func (s *eventService) Emit(ctx context.Context, tx *gorm.DB, event *IntegrationEvent) (*IntegrationEvent, error) {
	_, span := otel.Tracer("service").Start(ctx, "event.emit")
	defer span.End()
	span.SetAttributes(
		attribute.String("event.type", event.EventType),
		attribute.Int64("tenant.id", event.TenantID),
	)

	if !s.writeGate.ShouldEmit(ctx, event.EventType, event.TenantID) {
		span.SetAttributes(attribute.Bool("event.gated", true))
		slog.Debug("event: gated, skipping emit",
			"event_type", event.EventType,
			"tenant_id", event.TenantID,
		)
		return nil, nil
	}

	if sc := trace.SpanContextFromContext(ctx); sc.HasTraceID() {
		event.TraceID = sc.TraceID().String()
	}

	outbox := event.ToOutbox()

	repo := s.outboxRepo
	if tx != nil {
		repo = repo.WithTx(tx)
	}

	created, err := repo.Create(outbox)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "outbox write failed")
		return nil, err
	}

	span.SetAttributes(
		attribute.String("event.id", event.EventID.String()),
		attribute.Int64("outbox.id", created.OutboxID),
	)
	span.SetStatus(codes.Ok, "")

	return event, nil
}

// EmitWithPayload is a convenience helper for service layers.
func EmitWithPayload(
	svc EventService,
	ctx context.Context,
	tx *gorm.DB,
	eventType string,
	version int,
	tenantID int64,
	actorUserID *int64,
	subjectUUID *uuid.UUID,
	subjectType string,
	changedFields []string,
	payload datatypes.JSON,
) {
	event := NewIntegrationEvent(eventType, version, tenantID).
		SetActor(actorUserID).
		SetSubject(subjectUUID, subjectType).
		SetChangedFields(changedFields...).
		SetPayload(payload)

	_, err := svc.Emit(ctx, tx, event)
	if err != nil {
		slog.Error("event: emit failed",
			"event_type", eventType,
			"tenant_id", tenantID,
			"err", err,
		)
	}
}

// Shutdown stops the outbox relay and background workers.
func (s *eventService) Shutdown() {
	if s.relay != nil {
		s.relay.Shutdown()
	}
}
