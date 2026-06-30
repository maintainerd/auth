package oauth

import (
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/platform/database"
	"gorm.io/gorm"
)

// OAuthConsentChallengeRepository defines data access operations for pending
// consent challenges.
type OAuthConsentChallengeRepository interface {
	BaseRepositoryMethods[OAuthConsentChallenge]
	WithTx(tx *gorm.DB) OAuthConsentChallengeRepository
	FindChallengeByUUID(challengeUUID uuid.UUID) (*OAuthConsentChallenge, error)
	DeleteChallengeByUUID(challengeUUID uuid.UUID) error
	DeleteExpired(before time.Time) (int64, error)
}

type oauthConsentChallengeRepository struct {
	*BaseRepository[OAuthConsentChallenge]
}

// NewOAuthConsentChallengeRepository creates a new OAuthConsentChallengeRepository.
func NewOAuthConsentChallengeRepository(db *gorm.DB) OAuthConsentChallengeRepository {
	return &oauthConsentChallengeRepository{
		BaseRepository: database.NewBaseRepository[OAuthConsentChallenge](db, "oauth_consent_challenge_uuid", "oauth_consent_challenge_id"),
	}
}

func (r *oauthConsentChallengeRepository) WithTx(tx *gorm.DB) OAuthConsentChallengeRepository {
	return &oauthConsentChallengeRepository{
		BaseRepository: r.BaseRepository.WithTx(tx),
	}
}

// FindChallengeByUUID looks up a consent challenge by its UUID. Returns nil, nil when
// no matching challenge exists.
func (r *oauthConsentChallengeRepository) FindChallengeByUUID(challengeUUID uuid.UUID) (*OAuthConsentChallenge, error) {
	var challenge OAuthConsentChallenge
	err := r.DB().
		Preload("Client").
		Preload("Client.Tenant").
		Where("oauth_consent_challenge_uuid = ?", challengeUUID).
		First(&challenge).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &challenge, nil
}

// DeleteChallengeByUUID removes a consent challenge after it has been resolved.
func (r *oauthConsentChallengeRepository) DeleteChallengeByUUID(challengeUUID uuid.UUID) error {
	return r.DB().
		Where("oauth_consent_challenge_uuid = ?", challengeUUID).
		Delete(&OAuthConsentChallenge{}).Error
}

// DeleteExpired removes consent challenges that expired before the given
// cutoff time. Returns the number of rows deleted.
func (r *oauthConsentChallengeRepository) DeleteExpired(before time.Time) (int64, error) {
	var total int64
	for {
		result := r.DB().
			Where("expires_at < ?", before).
			Limit(10000).
			Delete(&OAuthConsentChallenge{})
		if result.Error != nil {
			return total, result.Error
		}
		if result.RowsAffected == 0 {
			break
		}
		total += result.RowsAffected
	}
	return total, nil
}
