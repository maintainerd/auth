package auditlog

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/google/uuid"
	"github.com/maintainerd/maintainerd-auth/internal/platform/middleware"
	"github.com/maintainerd/maintainerd-auth/internal/platform/ptr"
	"github.com/maintainerd/maintainerd-auth/internal/platform/telemetry"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"gorm.io/datatypes"
)

// LogEntry holds the data needed to record a single management audit event.
type LogEntry struct {
	TenantID      int64
	ActorUserID   *int64
	ActorClientID *int64
	Action        string
	ResourceType  string
	ResourceID    string
	ResourceUUID  *uuid.UUID
	Changes       string
	Outcome       string
	ErrorMessage  *string
}

// ManagementAuditLogger is the service interface for writing audit log entries.
type ManagementAuditLogger interface {
	Log(ctx context.Context, entry LogEntry) error
}

type managementAuditLogger struct {
	repo ManagementAuditLogRepository
}

// NewManagementAuditLogger creates a ManagementAuditLogger backed by repo.
func NewManagementAuditLogger(repo ManagementAuditLogRepository) ManagementAuditLogger {
	return &managementAuditLogger{repo: repo}
}

func (l *managementAuditLogger) Log(ctx context.Context, entry LogEntry) error {
	_, span := otel.Tracer("service").Start(ctx, "auditlog.log")
	defer span.End()

	if entry.Outcome == "" {
		entry.Outcome = "success"
	}

	record := &ManagementAuditLog{
		TenantID:      entry.TenantID,
		ActorUserID:   entry.ActorUserID,
		ActorClientID: entry.ActorClientID,
		Action:        entry.Action,
		ResourceType:  entry.ResourceType,
		ResourceID:    entry.ResourceID,
		ResourceUUID:  entry.ResourceUUID,
		Outcome:       entry.Outcome,
		ErrorMessage:  entry.ErrorMessage,
		IPAddress:     ptr.PtrOrNil(middleware.ClientIPFromContext(ctx)),
		UserAgent:     ptr.PtrOrNil(middleware.UserAgentFromContext(ctx)),
	}

	// Populate TraceID from OTel span context.
	sc := trace.SpanFromContext(ctx).SpanContext()
	if sc.IsValid() {
		tid := sc.TraceID().String()
		record.TraceID = &tid
	}

	// Populate RequestID from security middleware context.
	rid, _ := ctx.Value(middleware.RequestIDKey).(string)
	record.RequestID = ptr.PtrOrNil(rid)

	// Validate and set Changes; default to empty object when omitted.
	if entry.Changes == "" {
		record.Changes = datatypes.JSON("{}")
	} else {
		if !json.Valid([]byte(entry.Changes)) {
			return fmt.Errorf("auditlog: Changes is not valid JSON")
		}
		record.Changes = datatypes.JSON(entry.Changes)
	}

	if err := l.repo.Create(record); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "audit log write failed")
		// Audit writes are best-effort (the business action already succeeded), so
		// this metric is the only reliable signal that the audit trail has a gap.
		telemetry.RecordAuditWriteFailure(ctx)
		slog.ErrorContext(ctx, "failed to write management audit log",
			"action", entry.Action,
			"resource_type", entry.ResourceType,
			"error", err,
		)
		return err
	}

	span.SetStatus(codes.Ok, "")
	return nil
}
