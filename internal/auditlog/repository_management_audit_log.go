package auditlog

import (
	"github.com/maintainerd/maintainerd-auth/internal/platform/database"
	"gorm.io/gorm"
)

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
