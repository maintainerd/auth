package service

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/apperror"
	"github.com/maintainerd/auth/internal/model"
	"github.com/maintainerd/auth/internal/repository"
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
// event, decoupled from the persistence model.
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

// AuthEventService defines business operations on security auth events.
type AuthEventService interface {
	// Log records a new auth event. The trace ID is extracted from the context
	// automatically. Errors are logged but never propagated — callers should
	// fire-and-forget so event logging cannot break business flows.
	Log(ctx context.Context, input AuthEventInput)

	// FindPaginated returns a page of events filtered by the supplied criteria.
	FindPaginated(ctx context.Context, filter repository.AuthEventRepositoryGetFilter) (*repository.PaginationResult[AuthEventServiceDataResult], error)

	// FindByUUID returns a single event by UUID scoped to a tenant.
	FindByUUID(ctx context.Context, tenantID int64, eventUUID uuid.UUID) (*AuthEventServiceDataResult, error)

	// CountByEventType returns the count of events matching the type within a tenant.
	CountByEventType(ctx context.Context, eventType string, tenantID int64) (int64, error)

	// DeleteOlderThan removes events older than the cutoff. Returns the number
	// of rows deleted. Used by the retention background job.
	DeleteOlderThan(ctx context.Context, cutoff time.Time) (int64, error)
}

type authEventService struct {
	authEventRepo repository.AuthEventRepository
}

// NewAuthEventService creates a new AuthEventService.
func NewAuthEventService(authEventRepo repository.AuthEventRepository) AuthEventService {
	return &authEventService{authEventRepo: authEventRepo}
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

	event := &model.AuthEvent{
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
		Metadata:     input.Metadata,
	}

	if _, err := s.authEventRepo.Create(event); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to persist auth event")
	}
}

// FindPaginated returns a page of auth events filtered by the supplied criteria.
func (s *authEventService) FindPaginated(ctx context.Context, filter repository.AuthEventRepositoryGetFilter) (*repository.PaginationResult[AuthEventServiceDataResult], error) {
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
	return &repository.PaginationResult[AuthEventServiceDataResult]{
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

func toAuthEventServiceDataResult(e *model.AuthEvent) AuthEventServiceDataResult {
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
