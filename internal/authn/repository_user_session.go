package authn

import (
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/maintainerd/maintainerd-auth/internal/platform/database"
	"gorm.io/gorm"
)

type UserSessionRepository interface {
	FindActiveByUserID(userID int64) ([]UserSession, error)
	FindActiveByUUID(userID int64, sessionUUID uuid.UUID) (*UserSession, error)
	CountActive(userID int64) (int64, error)
	Create(session *UserSession) error
	Touch(sessionID int64, now time.Time) error
	RevokeByUUID(userID int64, sessionUUID uuid.UUID, reason string) error
	RevokeAllByUserID(userID int64) error
	DeleteExpired() (int64, error)
}

type userSessionRepository struct {
	*BaseRepository[UserSession]
}

func NewUserSessionRepository(db *gorm.DB) UserSessionRepository {
	return &userSessionRepository{
		BaseRepository: database.NewBaseRepository[UserSession](db, "user_session_uuid", "user_session_id"),
	}
}

func (r *userSessionRepository) FindActiveByUserID(userID int64) ([]UserSession, error) {
	var sessions []UserSession
	err := r.DB().
		Where("user_id = ? AND revoked_at IS NULL AND expires_at > ?", userID, time.Now()).
		Order("created_at DESC").
		Find(&sessions).Error
	return sessions, err
}

func (r *userSessionRepository) FindActiveByUUID(userID int64, sessionUUID uuid.UUID) (*UserSession, error) {
	var session UserSession
	err := r.DB().
		Where("user_session_uuid = ? AND user_id = ? AND revoked_at IS NULL", sessionUUID, userID).
		First(&session).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &session, nil
}

func (r *userSessionRepository) CountActive(userID int64) (int64, error) {
	var count int64
	err := r.DB().Model(&UserSession{}).
		Where("user_id = ? AND revoked_at IS NULL AND expires_at > ?", userID, time.Now()).
		Count(&count).Error
	return count, err
}

func (r *userSessionRepository) Create(session *UserSession) error {
	return r.DB().Create(session).Error
}

func (r *userSessionRepository) Touch(sessionID int64, now time.Time) error {
	return r.DB().Model(&UserSession{}).
		Where("user_session_id = ?", sessionID).
		Update("last_active_at", now).Error
}

func (r *userSessionRepository) RevokeByUUID(userID int64, sessionUUID uuid.UUID, reason string) error {
	now := time.Now()
	return r.DB().Model(&UserSession{}).
		Where("user_session_uuid = ? AND user_id = ?", sessionUUID, userID).
		Updates(map[string]interface{}{
			"revoked_at":     &now,
			"revoked_reason": &reason,
		}).Error
}

func (r *userSessionRepository) RevokeAllByUserID(userID int64) error {
	now := time.Now()
	reason := "admin_revoke"
	return r.DB().Model(&UserSession{}).
		Where("user_id = ? AND revoked_at IS NULL", userID).
		Updates(map[string]interface{}{
			"revoked_at":     &now,
			"revoked_reason": &reason,
		}).Error
}

func (r *userSessionRepository) DeleteExpired() (int64, error) {
	result := r.DB().
		Where("expires_at < ? OR (revoked_at IS NOT NULL AND revoked_at < ?)", time.Now(), time.Now().AddDate(0, 0, -30)).
		Delete(&UserSession{})
	return result.RowsAffected, result.Error
}
