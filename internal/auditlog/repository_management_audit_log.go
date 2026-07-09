package auditlog

import (
	"errors"

	"github.com/google/uuid"
	"github.com/maintainerd/maintainerd-auth/internal/platform/database"
	"gorm.io/gorm"
)

// ManagementAuditLogRepository is the persistence layer for management audit log entries.
type ManagementAuditLogRepository interface {
	Create(record *ManagementAuditLog) error
	FindPaginated(tenantID int64, filter ManagementAuditLogFilter) ([]ManagementAuditLog, int64, error)
	FindByUUIDAndTenantID(auditLogUUID uuid.UUID, tenantID int64) (*ManagementAuditLog, error)
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

func withActorLabels(query *gorm.DB) *gorm.DB {
	return query.
		Select(`
			management_audit_log.*,
			COALESCE(
				NULLIF(profiles.display_name, ''),
				NULLIF(CONCAT_WS(' ', profiles.first_name, profiles.last_name), ''),
				users.username,
				users.email
			) AS actor_user_name,
			COALESCE(NULLIF(clients.display_name, ''), clients.name) AS actor_client_name
		`).
		Joins(`
			LEFT JOIN users
				ON users.user_id = management_audit_log.actor_user_id
				AND users.tenant_id = management_audit_log.tenant_id
				AND users.deleted_at IS NULL
		`).
		Joins(`
			LEFT JOIN profiles
				ON profiles.user_id = users.user_id
				AND profiles.is_default = TRUE
				AND profiles.deleted_at IS NULL
		`).
		Joins(`
			LEFT JOIN clients
				ON clients.client_id = management_audit_log.actor_client_id
				AND clients.tenant_id = management_audit_log.tenant_id
				AND clients.deleted_at IS NULL
		`)
}

func (r *managementAuditLogRepository) FindPaginated(tenantID int64, filter ManagementAuditLogFilter) ([]ManagementAuditLog, int64, error) {
	query := r.DB().Where("management_audit_log.tenant_id = ?", tenantID)
	if filter.ResourceType != "" {
		query = query.Where("management_audit_log.resource_type = ?", filter.ResourceType)
	}
	if filter.Action != "" {
		query = query.Where("management_audit_log.action = ?", filter.Action)
	}
	if filter.ActorUserID != nil {
		query = query.Where("management_audit_log.actor_user_id = ?", *filter.ActorUserID)
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
	if err := withActorLabels(query).
		Order("management_audit_log.created_at DESC").
		Limit(limit).
		Offset((page - 1) * limit).
		Find(&logs).Error; err != nil {
		return nil, 0, err
	}

	return logs, total, nil
}

func (r *managementAuditLogRepository) FindByUUIDAndTenantID(auditLogUUID uuid.UUID, tenantID int64) (*ManagementAuditLog, error) {
	var log ManagementAuditLog
	err := withActorLabels(r.DB().Where(
		"management_audit_log_uuid = ? AND management_audit_log.tenant_id = ?",
		auditLogUUID,
		tenantID,
	)).First(&log).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &log, nil
}
