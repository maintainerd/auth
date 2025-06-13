package repository

import (
	"time"

	"github.com/maintainerd/auth/internal/model"
	"gorm.io/gorm"
)

type LoginAttemptRepository interface {
	BaseRepositoryMethods[model.LoginAttempt]
	FindAllByUserID(userID int64, limit int, offset int) ([]model.LoginAttempt, error)
	FindAllByEmail(email string, limit int, offset int) ([]model.LoginAttempt, error)
	FindRecentAttemptsByIP(ip string, withinMinutes int) ([]model.LoginAttempt, error)
	CountFailedAttemptsByUserID(userID int64, withinMinutes int) (int64, error)
	DeleteOlderThan(cutoff time.Time) error
}

type loginAttemptRepository struct {
	*BaseRepository[model.LoginAttempt]
	db *gorm.DB
}

func NewLoginAttemptRepository(db *gorm.DB) LoginAttemptRepository {
	return &loginAttemptRepository{
		BaseRepository: NewBaseRepository[model.LoginAttempt](db, "login_attempt_uuid", "login_attempt_id"),
		db:             db,
	}
}

func (r *loginAttemptRepository) FindAllByUserID(userID int64, limit int, offset int) ([]model.LoginAttempt, error) {
	var attempts []model.LoginAttempt
	err := r.db.
		Where("user_id = ?", userID).
		Order("attempted_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&attempts).Error
	return attempts, err
}

func (r *loginAttemptRepository) FindAllByEmail(email string, limit int, offset int) ([]model.LoginAttempt, error) {
	var attempts []model.LoginAttempt
	err := r.db.
		Where("email = ?", email).
		Order("attempted_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&attempts).Error
	return attempts, err
}

func (r *loginAttemptRepository) FindRecentAttemptsByIP(ip string, withinMinutes int) ([]model.LoginAttempt, error) {
	var attempts []model.LoginAttempt
	cutoff := time.Now().Add(-time.Duration(withinMinutes) * time.Minute)
	err := r.db.
		Where("ip_address = ? AND attempted_at >= ?", ip, cutoff).
		Order("attempted_at DESC").
		Find(&attempts).Error
	return attempts, err
}

func (r *loginAttemptRepository) CountFailedAttemptsByUserID(userID int64, withinMinutes int) (int64, error) {
	var count int64
	cutoff := time.Now().Add(-time.Duration(withinMinutes) * time.Minute)
	err := r.db.
		Model(&model.LoginAttempt{}).
		Where("user_id = ? AND is_success = false AND attempted_at >= ?", userID, cutoff).
		Count(&count).Error
	return count, err
}

func (r *loginAttemptRepository) DeleteOlderThan(cutoff time.Time) error {
	return r.db.
		Where("attempted_at < ?", cutoff).
		Delete(&model.LoginAttempt{}).Error
}
