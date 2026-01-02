package repository

import (
	"time"

	"github.com/maintainerd/auth/internal/model"
	"gorm.io/gorm"
)

type AuthLogRepository interface {
	BaseRepositoryMethods[model.AuthLog]
	FindByUserID(userID int64, limit int, offset int) ([]model.AuthLog, error)
	FindByUUIDAndTenantID(uuid string, tenantID int64) (*model.AuthLog, error)
	FindByEventType(eventType string, tenantID int64, limit int, offset int) ([]model.AuthLog, error)
	FindByDateRange(tenantID int64, from, to time.Time) ([]model.AuthLog, error)
	DeleteOlderThan(cutoff time.Time) error
	CountByEventType(eventType string, tenantID int64) (int64, error)
}

type authLogRepository struct {
	*BaseRepository[model.AuthLog]
	db *gorm.DB
}

func NewAuthLogRepository(db *gorm.DB) AuthLogRepository {
	return &authLogRepository{
		BaseRepository: NewBaseRepository[model.AuthLog](db, "auth_log_uuid", "auth_log_id"),
		db:             db,
	}
}

func (r *authLogRepository) FindByUserID(userID int64, limit int, offset int) ([]model.AuthLog, error) {
	var logs []model.AuthLog
	err := r.db.
		Where("user_id = ?", userID).
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&logs).Error
	return logs, err
}

// FindByUUIDAndTenantID retrieves an auth log by UUID and tenant ID
func (r *authLogRepository) FindByUUIDAndTenantID(uuid string, tenantID int64) (*model.AuthLog, error) {
	var log model.AuthLog
	err := r.db.
		Where("auth_log_uuid = ? AND tenant_id = ?", uuid, tenantID).
		First(&log).Error
	if err != nil {
		return nil, err
	}
	return &log, nil
}

func (r *authLogRepository) FindByEventType(eventType string, tenantID int64, limit int, offset int) ([]model.AuthLog, error) {
	var logs []model.AuthLog
	err := r.db.
		Where("event_type = ? AND tenant_id = ?", eventType, tenantID).
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&logs).Error
	return logs, err
}

func (r *authLogRepository) FindByDateRange(tenantID int64, from, to time.Time) ([]model.AuthLog, error) {
	var logs []model.AuthLog
	err := r.db.
		Where("tenant_id = ? AND created_at BETWEEN ? AND ?", tenantID, from, to).
		Order("created_at DESC").
		Find(&logs).Error
	return logs, err
}

func (r *authLogRepository) DeleteOlderThan(cutoff time.Time) error {
	return r.db.
		Where("created_at < ?", cutoff).
		Delete(&model.AuthLog{}).Error
}

func (r *authLogRepository) CountByEventType(eventType string, tenantID int64) (int64, error) {
	var count int64
	err := r.db.
		Model(&model.AuthLog{}).
		Where("event_type = ? AND tenant_id = ?", eventType, tenantID).
		Count(&count).Error
	return count, err
}
