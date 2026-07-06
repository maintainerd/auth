package auditlog

import (
	"context"
	"log/slog"

	"github.com/maintainerd/maintainerd-auth/internal/platform/database"
	"github.com/maintainerd/maintainerd-auth/internal/platform/middleware"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	"gorm.io/gorm"
)

type LogEntry struct {
	TenantID      int64
	ActorUserID   *int64
	ActorClientID *int64
	Action        string
	ResourceType  string
	ResourceID    string
	Changes       string
	Outcome       string
	ErrorMessage  *string
}

type ManagementAuditLogger interface {
	Log(ctx context.Context, entry LogEntry) error
}

type managementAuditLogger struct {
	repo ManagementAuditLogRepository
}

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
		Outcome:       entry.Outcome,
		ErrorMessage:  entry.ErrorMessage,
		IPAddress:     strPtr(middleware.ClientIPFromContext(ctx)),
		UserAgent:     strPtr(middleware.UserAgentFromContext(ctx)),
	}

	if entry.Changes != "" {
		record.Changes = []byte(entry.Changes)
	}

	if err := l.repo.Create(record); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "audit log write failed")
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

// ManagementAuditLogRepository is the persistence layer for management audit log entries.
type ManagementAuditLogRepository interface {
	Create(record *ManagementAuditLog) error
	FindPaginated(tenantID int64, filter ManagementAuditLogFilter) ([]ManagementAuditLog, int64, error)
}

type managementAuditLogRepository struct {
	*BaseRepository[ManagementAuditLog]
}

func NewManagementAuditLogRepository(db *gorm.DB) ManagementAuditLogRepository {
	return &managementAuditLogRepository{
		BaseRepository: database.NewBaseRepository[ManagementAuditLog](db, "management_audit_log_uuid", "management_audit_log_id"),
	}
}

func (r *managementAuditLogRepository) Create(record *ManagementAuditLog) error {
	return r.DB().Create(record).Error
}

func (r *managementAuditLogRepository) FindPaginated(tenantID int64, filter ManagementAuditLogFilter) ([]ManagementAuditLog, int64, error) {
	query := r.DB().Where("tenant_id = ?", tenantID)
	if filter.ResourceType != "" {
		query = query.Where("resource_type = ?", filter.ResourceType)
	}
	if filter.Action != "" {
		query = query.Where("action = ?", filter.Action)
	}
	if filter.ActorUserID != nil {
		query = query.Where("actor_user_id = ?", *filter.ActorUserID)
	}

	var total int64
	if err := query.Model(&ManagementAuditLog{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var logs []ManagementAuditLog
	limit := filter.Limit
	if limit <= 0 {
		limit = 20
	}
	page := filter.Page
	if page <= 0 {
		page = 1
	}
	if err := query.Order("created_at DESC").Limit(limit).Offset((page - 1) * limit).Find(&logs).Error; err != nil {
		return nil, 0, err
	}

	return logs, total, nil
}

type ManagementAuditLogFilter struct {
	ResourceType string
	Action       string
	ActorUserID  *int64
	Page         int
	Limit        int
}

func strPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
