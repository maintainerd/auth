package repository

import (
	"time"

	"github.com/maintainerd/auth/internal/model"
	"gorm.io/gorm"
)

type AuthLogRepository interface {
	BaseRepositoryMethods[model.AuthLog]
	FindByUserID(userID int64, limit int, offset int) ([]model.AuthLog, error)
	FindByEventType(eventType string, authContainerID int64, limit int, offset int) ([]model.AuthLog, error)
	FindByDateRange(authContainerID int64, from, to time.Time) ([]model.AuthLog, error)
	DeleteOlderThan(cutoff time.Time) error
	CountByEventType(eventType string, authContainerID int64) (int64, error)
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

func (r *authLogRepository) FindByEventType(eventType string, authContainerID int64, limit int, offset int) ([]model.AuthLog, error) {
	var logs []model.AuthLog
	err := r.db.
		Where("event_type = ? AND auth_container_id = ?", eventType, authContainerID).
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&logs).Error
	return logs, err
}

func (r *authLogRepository) FindByDateRange(authContainerID int64, from, to time.Time) ([]model.AuthLog, error) {
	var logs []model.AuthLog
	err := r.db.
		Where("auth_container_id = ? AND created_at BETWEEN ? AND ?", authContainerID, from, to).
		Order("created_at DESC").
		Find(&logs).Error
	return logs, err
}

func (r *authLogRepository) DeleteOlderThan(cutoff time.Time) error {
	return r.db.
		Where("created_at < ?", cutoff).
		Delete(&model.AuthLog{}).Error
}

func (r *authLogRepository) CountByEventType(eventType string, authContainerID int64) (int64, error) {
	var count int64
	err := r.db.
		Model(&model.AuthLog{}).
		Where("event_type = ? AND auth_container_id = ?", eventType, authContainerID).
		Count(&count).Error
	return count, err
}
