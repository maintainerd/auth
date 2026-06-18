package event

import (
	"context"
	"log/slog"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
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
	outboxRepo OutboxRepository
	writeGate  *WriteGate
	relay      *Relay
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

// Shutdown stops the outbox relay and background workers.
func (s *eventService) Shutdown() {
	if s.relay != nil {
		s.relay.Shutdown()
	}
}
