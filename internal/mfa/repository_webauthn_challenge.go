package mfa

import (
	"time"

	"github.com/maintainerd/maintainerd-auth/internal/platform/database"
	"gorm.io/gorm"
)

type webAuthnChallengeRepository struct {
	*BaseRepository[WebAuthnChallenge]
}

func NewWebAuthnChallengeRepository(db *gorm.DB) WebAuthnChallengeRepository {
	return &webAuthnChallengeRepository{
		BaseRepository: database.NewBaseRepository[WebAuthnChallenge](db, "webauthn_challenge_uuid", "webauthn_challenge_id"),
	}
}

func (r *webAuthnChallengeRepository) Store(c *WebAuthnChallenge) error {
	return r.DB().Create(c).Error
}

func (r *webAuthnChallengeRepository) Consume(challenge string, operation string) error {
	now := time.Now()
	result := r.DB().Model(&WebAuthnChallenge{}).
		Where("challenge = ? AND operation = ? AND used_at IS NULL AND expires_at > ?", challenge, operation, now).
		Update("used_at", now)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func (r *webAuthnChallengeRepository) DeleteExpired() (int64, error) {
	result := r.DB().
		Where("expires_at < ?", time.Now().Add(-1*time.Hour)).
		Delete(&WebAuthnChallenge{})
	return result.RowsAffected, result.Error
}
