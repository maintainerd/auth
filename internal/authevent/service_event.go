package authevent

import (
	"context"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/platform/apperror"
	"github.com/maintainerd/auth/internal/platform/logging"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"gorm.io/datatypes"
)

// AuthEventInput groups the parameters for recording a single auth event.
type AuthEventInput struct {
	TenantID     int64
	ActorUserID  *int64
	TargetUserID *int64
	IPAddress    string
	UserAgent    *string
	Category     string
	EventType    string
	Severity     string
	Result       string
	Description  *string
	ErrorReason  *string
	Metadata     datatypes.JSON
}

// AuthEventServiceDataResult is the service-layer representation of an auth
// event, decoupled from the persistence
type AuthEventServiceDataResult struct {
	AuthEventUUID uuid.UUID
	TenantID      int64
	ActorUserID   *int64
	TargetUserID  *int64
	IPAddress     string
	UserAgent     *string
	Category      string
	EventType     string
	Severity      string
	Result        string
	Description   *string
	ErrorReason   *string
	TraceID       *string
	Metadata      datatypes.JSON
	CreatedAt     time.Time
}

// WebhookDispatcher delivers a persisted auth event to subscribed webhook endpoints.
type WebhookDispatcher interface {
	Dispatch(ctx context.Context, event *AuthEvent)
	Shutdown()
}

// AuthEventService defines business operations on security auth events.
type AuthEventService interface {
	// Log records a new auth event. The trace ID is extracted from the context
	// automatically. Errors are logged but never propagated — callers should
	// fire-and-forget so event logging cannot break business flows.
	Log(ctx context.Context, input AuthEventInput)

	// FindPaginated returns a page of events filtered by the supplied criteria.
	FindPaginated(ctx context.Context, filter AuthEventRepositoryGetFilter) (*PaginationResult[AuthEventServiceDataResult], error)

	// FindByUUID returns a single event by UUID scoped to a tenant.
	FindByUUID(ctx context.Context, tenantID int64, eventUUID uuid.UUID) (*AuthEventServiceDataResult, error)

	// CountByEventType returns the count of events matching the type within a tenant.
	CountByEventType(ctx context.Context, eventType string, tenantID int64) (int64, error)

	// DeleteOlderThan removes events older than the cutoff. Returns the number
	// of rows deleted. Used by the retention background job.
	DeleteOlderThan(ctx context.Context, cutoff time.Time) (int64, error)

	// Shutdown waits for all in-flight webhook dispatches to complete.
	Shutdown()
}

type authEventService struct {
	authEventRepo AuthEventRepository
	dispatcher    WebhookDispatcher
	wg            sync.WaitGroup
}

// NewAuthEventService creates a new AuthEventService.
// Pass nil for dispatcher to disable webhook delivery (e.g. in tests).
func NewAuthEventService(authEventRepo AuthEventRepository, dispatcher WebhookDispatcher) AuthEventService {
	return &authEventService{authEventRepo: authEventRepo, dispatcher: dispatcher}
}

// noopAuthEventService is a silent implementation used when no real service is
// wired (e.g. in unit tests that pass nil to service constructors).
type noopAuthEventService struct{}

func (noopAuthEventService) Log(_ context.Context, _ AuthEventInput) {}
func (noopAuthEventService) FindPaginated(_ context.Context, _ AuthEventRepositoryGetFilter) (*PaginationResult[AuthEventServiceDataResult], error) {
	return &PaginationResult[AuthEventServiceDataResult]{}, nil
}
func (noopAuthEventService) FindByUUID(_ context.Context, _ int64, _ uuid.UUID) (*AuthEventServiceDataResult, error) {
	return nil, nil
}
func (noopAuthEventService) CountByEventType(_ context.Context, _ string, _ int64) (int64, error) {
	return 0, nil
}
func (noopAuthEventService) DeleteOlderThan(_ context.Context, _ time.Time) (int64, error) {
	return 0, nil
}
func (noopAuthEventService) Shutdown() {}

// NoopService returns a silent AuthEventService that discards all events.
func NoopService() AuthEventService {
	return noopAuthEventService{}
}

// coalesceAuthEventService returns svc if non-nil, otherwise a no-op.
func coalesceAuthEventService(svc AuthEventService) AuthEventService {
	if svc != nil {
		return svc
	}
	return noopAuthEventService{}
}

// Log records a new auth event. The trace ID is extracted from the span
// context automatically so it appears in both the DB and OTel.
func (s *authEventService) Log(ctx context.Context, input AuthEventInput) {
	_, span := otel.Tracer("service").Start(ctx, "auth_event.log")
	defer span.End()
	span.SetAttributes(
		attribute.String("auth_event.category", input.Category),
		attribute.String("auth_event.event_type", input.EventType),
		attribute.String("auth_event.result", input.Result),
	)

	var traceID *string
	if sc := trace.SpanContextFromContext(ctx); sc.HasTraceID() {
		tid := sc.TraceID().String()
		traceID = &tid
	}

	event := &AuthEvent{
		TenantID:     input.TenantID,
		ActorUserID:  input.ActorUserID,
		TargetUserID: input.TargetUserID,
		IPAddress:    input.IPAddress,
		UserAgent:    input.UserAgent,
		Category:     input.Category,
		EventType:    input.EventType,
		Severity:     input.Severity,
		Result:       input.Result,
		Description:  input.Description,
		ErrorReason:  input.ErrorReason,
		TraceID:      traceID,
		Metadata:     datatypes.JSON(logging.RedactJSON(input.Metadata)),
	}

	created, err := s.authEventRepo.Create(event)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to persist auth event")
		return
	}

	if s.dispatcher != nil {
		s.wg.Add(1)
		go func() {
			defer s.wg.Done()
			dCtx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
			defer cancel()
			s.dispatcher.Dispatch(dCtx, created)
		}()
	}
}

// FindPaginated returns a page of auth events filtered by the supplied criteria.
func (s *authEventService) FindPaginated(ctx context.Context, filter AuthEventRepositoryGetFilter) (*PaginationResult[AuthEventServiceDataResult], error) {
	_, span := otel.Tracer("service").Start(ctx, "auth_event.find_paginated")
	defer span.End()

	result, err := s.authEventRepo.FindPaginated(filter)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "find paginated auth events failed")
		return nil, apperror.NewInternal("failed to query auth events", err)
	}

	mapped := make([]AuthEventServiceDataResult, len(result.Data))
	for i, e := range result.Data {
		mapped[i] = toAuthEventServiceDataResult(&e)
	}

	span.SetStatus(codes.Ok, "")
	return &PaginationResult[AuthEventServiceDataResult]{
		Data:       mapped,
		Total:      result.Total,
		Page:       result.Page,
		Limit:      result.Limit,
		TotalPages: result.TotalPages,
	}, nil
}

// FindByUUID returns a single auth event by UUID scoped to a tenant.
func (s *authEventService) FindByUUID(ctx context.Context, tenantID int64, eventUUID uuid.UUID) (*AuthEventServiceDataResult, error) {
	_, span := otel.Tracer("service").Start(ctx, "auth_event.find_by_uuid")
	defer span.End()
	span.SetAttributes(attribute.String("auth_event.uuid", eventUUID.String()))

	event, err := s.authEventRepo.FindByUUIDAndTenantID(eventUUID.String(), tenantID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "find auth event by uuid failed")
		return nil, apperror.NewInternal("failed to find auth event", err)
	}
	if event == nil {
		return nil, apperror.NewNotFound("auth event")
	}

	result := toAuthEventServiceDataResult(event)
	span.SetStatus(codes.Ok, "")
	return &result, nil
}

// CountByEventType returns the count of events matching the type within a tenant.
func (s *authEventService) CountByEventType(ctx context.Context, eventType string, tenantID int64) (int64, error) {
	_, span := otel.Tracer("service").Start(ctx, "auth_event.count_by_event_type")
	defer span.End()
	span.SetAttributes(
		attribute.String("auth_event.event_type", eventType),
		attribute.Int64("tenant.id", tenantID),
	)

	count, err := s.authEventRepo.CountByEventType(eventType, tenantID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "count by event type failed")
		return 0, apperror.NewInternal("failed to count auth events by type", err)
	}

	span.SetStatus(codes.Ok, "")
	return count, nil
}

// DeleteOlderThan removes events older than the cutoff and returns the count.
func (s *authEventService) DeleteOlderThan(ctx context.Context, cutoff time.Time) (int64, error) {
	_, span := otel.Tracer("service").Start(ctx, "auth_event.delete_older_than")
	defer span.End()
	span.SetAttributes(attribute.String("cutoff", cutoff.Format(time.RFC3339)))

	count, err := s.authEventRepo.DeleteOlderThan(cutoff)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "delete older than failed")
		return 0, apperror.NewInternal("failed to delete old auth events", err)
	}

	span.SetAttributes(attribute.Int64("deleted_count", count))
	span.SetStatus(codes.Ok, "")
	return count, nil
}

// Shutdown waits for all in-flight webhook dispatches to complete.
func (s *authEventService) Shutdown() {
	s.wg.Wait()
	if s.dispatcher != nil {
		s.dispatcher.Shutdown()
	}
}

func toAuthEventServiceDataResult(e *AuthEvent) AuthEventServiceDataResult {
	return AuthEventServiceDataResult{
		AuthEventUUID: e.AuthEventUUID,
		TenantID:      e.TenantID,
		ActorUserID:   e.ActorUserID,
		TargetUserID:  e.TargetUserID,
		IPAddress:     e.IPAddress,
		UserAgent:     e.UserAgent,
		Category:      e.Category,
		EventType:     e.EventType,
		Severity:      e.Severity,
		Result:        e.Result,
		Description:   e.Description,
		ErrorReason:   e.ErrorReason,
		TraceID:       e.TraceID,
		Metadata:      e.Metadata,
		CreatedAt:     e.CreatedAt,
	}
}
