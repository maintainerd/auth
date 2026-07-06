package oauth

import (
	"time"

	"gorm.io/gorm"
)

type oauthDPoPNonceRepository struct {
	db *gorm.DB
}

// NewOAuthDPoPNonceRepository creates a new OAuthDPoPNonceRepository.
func NewOAuthDPoPNonceRepository(db *gorm.DB) OAuthDPoPNonceRepository {
	return &oauthDPoPNonceRepository{db: db}
}

func (r *oauthDPoPNonceRepository) SaveNonce(tenantID, clientID int64, nonce string, expiresAt time.Time) error {
	return r.db.Create(&OAuthDPoPNonce{
		TenantID:  tenantID,
		ClientID:  clientID,
		Nonce:     nonce,
		ExpiresAt: expiresAt,
	}).Error
}

// ConsumeNonce atomically marks a still-valid nonce as used. The single UPDATE
// with used_at IS NULL AND expires_at > now() guarantees single-use even under
// concurrent requests (only one UPDATE affects the row).
func (r *oauthDPoPNonceRepository) ConsumeNonce(nonce string) (bool, error) {
	result := r.db.Model(&OAuthDPoPNonce{}).
		Where("nonce = ? AND used_at IS NULL AND expires_at > ?", nonce, time.Now()).
		Update("used_at", time.Now())
	if result.Error != nil {
		return false, result.Error
	}
	return result.RowsAffected > 0, nil
}

func (r *oauthDPoPNonceRepository) DeleteExpired() (int64, error) {
	result := r.db.Where("expires_at < ?", time.Now()).Delete(&OAuthDPoPNonce{})
	return result.RowsAffected, result.Error
}
