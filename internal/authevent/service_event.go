package authevent

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/maintainerd/maintainerd-auth/internal/platform/apperror"
	"github.com/maintainerd/maintainerd-auth/internal/platform/logging"
	"github.com/maintainerd/maintainerd-auth/internal/platform/telemetry"
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

	// Shutdown is retained for lifecycle symmetry; auth-event logging is now
	// synchronous and durable (persisted before return), so there is nothing to
	// drain. Auth events are deliberately NOT fanned out to webhooks — they are
	// not integration events (see the event catalog).
	Shutdown()
}

type authEventService struct {
	authEventRepo     AuthEventRepository
	auditConfigReader AuditConfigReader
	auditConfigCache  map[int64]auditConfigCacheEntry
	auditConfigMu     sync.RWMutex
}

// NewAuthEventService creates a new AuthEventService.
func NewAuthEventService(authEventRepo AuthEventRepository, readers ...AuditConfigReader) AuthEventService {
	var reader AuditConfigReader
	if len(readers) > 0 {
		reader = readers[0]
	}
	return &authEventService{
		authEventRepo:     authEventRepo,
		auditConfigReader: reader,
		auditConfigCache:  map[int64]auditConfigCacheEntry{},
	}
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

	// Meter every auth event (login/token/lockout/oauth/…) regardless of whether
	// audit_config persists it, so operational dashboards see the true rates.
	telemetry.RecordAuthEvent(ctx, input.Category, input.EventType, input.Result)

	cfg := s.getAuditConfig(ctx, input.TenantID)
	if !cfg.Enabled || !cfg.allowsEvent(input.EventType) || !cfg.allowsSeverity(input.Severity) {
		span.SetStatus(codes.Ok, "auth event skipped by audit_config")
		return
	}

	var traceID *string
	if sc := trace.SpanContextFromContext(ctx); sc.HasTraceID() {
		tid := sc.TraceID().String()
		traceID = &tid
	}

	description := input.Description
	errorReason := input.ErrorReason
	metadata := input.Metadata
	if cfg.masksPII() {
		description = logging.RedactString(input.Description)
		errorReason = logging.RedactString(input.ErrorReason)
		metadata = datatypes.JSON(logging.RedactJSON(input.Metadata))
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
		Description:  description,
		ErrorReason:  errorReason,
		TraceID:      traceID,
		Metadata:     metadata,
	}

	if _, err := s.authEventRepo.Create(event); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to persist auth event")
		return
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

// DeleteExpiredByAuditConfig removes events using each tenant's audit_config
// retention_days value.
func (s *authEventService) DeleteExpiredByAuditConfig(ctx context.Context, now time.Time) (int64, error) {
	_, span := otel.Tracer("service").Start(ctx, "auth_event.delete_expired_by_audit_config")
	defer span.End()

	count, err := s.authEventRepo.DeleteExpiredByAuditConfig(now, defaultAuditRetention)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "delete expired by audit config failed")
		return 0, apperror.NewInternal("failed to delete expired auth events", err)
	}

	span.SetAttributes(attribute.Int64("deleted_count", count))
	span.SetStatus(codes.Ok, "")
	return count, nil
}

type AuthEventExport struct {
	Format      string
	ContentType string
	Filename    string
	Data        []byte
}

// Export renders auth events as JSON or CSV. Empty format defaults to JSON.
func (s *authEventService) Export(ctx context.Context, filter AuthEventRepositoryGetFilter, format string) (*AuthEventExport, error) {
	_, span := otel.Tracer("service").Start(ctx, "auth_event.export")
	defer span.End()

	normalizedFormat, err := normalizeExportFormat(format)
	if err != nil {
		return nil, apperror.NewValidation(err.Error())
	}

	filter.Page = 1
	filter.Limit = maxAuthEventExportLimit
	result, err := s.FindPaginated(ctx, filter)
	if err != nil {
		return nil, err
	}

	switch normalizedFormat {
	case "csv":
		return s.exportCSV(result.Data)
	default:
		return s.exportJSON(result.Data)
	}
}

// Shutdown is a no-op: auth-event logging is synchronous and durable, so there
// is no in-flight async work to drain.
func (s *authEventService) Shutdown() {}

func (s *authEventService) getAuditConfig(ctx context.Context, tenantID int64) auditConfig {
	if s.auditConfigReader == nil || tenantID == 0 {
		return legacyAuditConfig()
	}

	now := time.Now()
	s.auditConfigMu.RLock()
	entry, ok := s.auditConfigCache[tenantID]
	s.auditConfigMu.RUnlock()
	if ok && now.Before(entry.expiresAt) {
		return entry.config
	}

	raw, err := s.auditConfigReader.GetAuditConfig(ctx, tenantID)
	if err != nil {
		return legacyAuditConfig()
	}
	cfg := parseAuditConfig(raw)

	s.auditConfigMu.Lock()
	s.auditConfigCache[tenantID] = auditConfigCacheEntry{
		config:    cfg,
		expiresAt: now.Add(auditConfigCacheTTL),
	}
	s.auditConfigMu.Unlock()
	return cfg
}

func (s *authEventService) exportJSON(events []AuthEventServiceDataResult) (*AuthEventExport, error) {
	data, err := json.MarshalIndent(toAuthEventResponseDTOList(events), "", "  ")
	if err != nil {
		return nil, apperror.NewInternal("failed to export auth events", err)
	}
	return &AuthEventExport{
		Format:      "json",
		ContentType: "application/json",
		Filename:    "auth-events.json",
		Data:        data,
	}, nil
}

func (s *authEventService) exportCSV(events []AuthEventServiceDataResult) (*AuthEventExport, error) {
	var buffer csvBuffer
	writer := csv.NewWriter(&buffer)
	if err := writer.Write([]string{
		"auth_event_id",
		"ip_address",
		"user_agent",
		"category",
		"event_type",
		"severity",
		"result",
		"description",
		"error_reason",
		"trace_id",
		"metadata",
		"created_at",
	}); err != nil {
		return nil, apperror.NewInternal("failed to export auth events", err)
	}
	for _, event := range events {
		if err := writer.Write(authEventCSVRow(event)); err != nil {
			return nil, apperror.NewInternal("failed to export auth events", err)
		}
	}
	writer.Flush()
	if err := writer.Error(); err != nil {
		return nil, apperror.NewInternal("failed to export auth events", err)
	}
	return &AuthEventExport{
		Format:      "csv",
		ContentType: "text/csv",
		Filename:    "auth-events.csv",
		Data:        buffer.Bytes(),
	}, nil
}

type csvBuffer []byte

func (b *csvBuffer) Write(p []byte) (int, error) {
	*b = append(*b, p...)
	return len(p), nil
}

func (b *csvBuffer) Bytes() []byte {
	return []byte(*b)
}

func authEventCSVRow(event AuthEventServiceDataResult) []string {
	return []string{
		event.AuthEventUUID.String(),
		event.IPAddress,
		stringPtrValue(event.UserAgent),
		event.Category,
		event.EventType,
		event.Severity,
		event.Result,
		stringPtrValue(event.Description),
		stringPtrValue(event.ErrorReason),
		stringPtrValue(event.TraceID),
		string(event.Metadata),
		event.CreatedAt.Format(time.RFC3339),
	}
}

func stringPtrValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
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
