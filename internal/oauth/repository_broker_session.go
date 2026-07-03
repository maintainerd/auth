package oauth

import (
	"errors"
	"time"

	"github.com/maintainerd/maintainerd-auth/internal/platform/database"
	"gorm.io/gorm"
)

// OAuthBrokerSessionRepository defines data access for brokered-login sessions
// that correlate the app↔maintainerd (OAuth #1) and maintainerd↔provider
// (OAuth #2) legs of a brokered login.
type OAuthBrokerSessionRepository interface {
	BaseRepositoryMethods[OAuthBrokerSession]
	WithTx(tx *gorm.DB) OAuthBrokerSessionRepository
	FindByIdpState(idpState string) (*OAuthBrokerSession, error)
	Consume(id int64, at time.Time) error
	DeleteExpired(before time.Time) (int64, error)
}

type oauthBrokerSessionRepository struct {
	*BaseRepository[OAuthBrokerSession]
}

// NewOAuthBrokerSessionRepository creates a new OAuthBrokerSessionRepository.
func NewOAuthBrokerSessionRepository(db *gorm.DB) OAuthBrokerSessionRepository {
	return &oauthBrokerSessionRepository{
		BaseRepository: database.NewBaseRepository[OAuthBrokerSession](db, "oauth_broker_session_uuid", "oauth_broker_session_id"),
	}
}

func (r *oauthBrokerSessionRepository) WithTx(tx *gorm.DB) OAuthBrokerSessionRepository {
	return &oauthBrokerSessionRepository{
		BaseRepository: r.BaseRepository.WithTx(tx),
	}
}

// FindByIdpState looks up an unconsumed broker session by the state value
// maintainerd sent to the upstream provider. Returns nil, nil when none match.
func (r *oauthBrokerSessionRepository) FindByIdpState(idpState string) (*OAuthBrokerSession, error) {
	var session OAuthBrokerSession
	err := r.DB().
		Where("idp_state = ? AND consumed_at IS NULL", idpState).
		First(&session).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &session, nil
}

// Consume atomically marks the session as used so concurrent callbacks cannot
// replay the same idp_state. It fails with ErrAlreadyConsumed when the row
// was already consumed (RowsAffected == 0).
func (r *oauthBrokerSessionRepository) Consume(id int64, at time.Time) error {
	result := r.DB().Model(&OAuthBrokerSession{}).
		Where("oauth_broker_session_id = ? AND consumed_at IS NULL", id).
		Update("consumed_at", at)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrAlreadyConsumed
	}
	return nil
}

// ErrAlreadyConsumed is returned by Consume when the broker session has
// already been consumed by another callback.
var ErrAlreadyConsumed = errors.New("broker session already consumed")

// DeleteExpired removes broker sessions that expired before the given cutoff.
func (r *oauthBrokerSessionRepository) DeleteExpired(before time.Time) (int64, error) {
	result := r.DB().
		Where("expires_at < ?", before).
		Delete(&OAuthBrokerSession{})
	return result.RowsAffected, result.Error
}
