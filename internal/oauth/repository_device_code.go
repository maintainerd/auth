package oauth

import (
	"errors"
	"time"

	"github.com/maintainerd/auth/internal/platform/database"
	"gorm.io/gorm"
)

// OAuthDeviceCodeRepository defines data access operations for device
// authorization codes (RFC 8628).
type OAuthDeviceCodeRepository interface {
	BaseRepositoryMethods[OAuthDeviceCode]
	WithTx(tx *gorm.DB) OAuthDeviceCodeRepository
	FindByDeviceCodeHash(hash string) (*OAuthDeviceCode, error)
	FindByUserCode(userCode string) (*OAuthDeviceCode, error)
	UpdateStatus(id int64, status string, userID *int64) error
	UpdateLastPollAt(id int64) error
	DeleteExpired(before time.Time) (int64, error)
}

type oauthDeviceCodeRepository struct {
	*BaseRepository[OAuthDeviceCode]
}

// NewOAuthDeviceCodeRepository creates a new OAuthDeviceCodeRepository.
func NewOAuthDeviceCodeRepository(db *gorm.DB) OAuthDeviceCodeRepository {
	return &oauthDeviceCodeRepository{
		BaseRepository: database.NewBaseRepository[OAuthDeviceCode](db, "oauth_device_code_uuid", "oauth_device_code_id"),
	}
}

func (r *oauthDeviceCodeRepository) WithTx(tx *gorm.DB) OAuthDeviceCodeRepository {
	return &oauthDeviceCodeRepository{
		BaseRepository: r.BaseRepository.WithTx(tx),
	}
}

// FindByDeviceCodeHash looks up a device code record by the SHA-256 hash of the
// raw device_code. Returns nil, nil when not found.
func (r *oauthDeviceCodeRepository) FindByDeviceCodeHash(hash string) (*OAuthDeviceCode, error) {
	var code OAuthDeviceCode
	err := r.DB().
		Preload("Client").
		Preload("Client.IdentityProvider").
		Preload("User").
		Where("device_code_hash = ?", hash).
		First(&code).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &code, nil
}

// FindByUserCode looks up a pending device code by the human-readable user_code.
// Returns nil, nil when not found.
func (r *oauthDeviceCodeRepository) FindByUserCode(userCode string) (*OAuthDeviceCode, error) {
	var code OAuthDeviceCode
	err := r.DB().
		Preload("Client").
		Where("user_code = ? AND status = 'pending'", userCode).
		First(&code).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &code, nil
}

// UpdateStatus sets the status and optionally the approving user on a device code.
func (r *oauthDeviceCodeRepository) UpdateStatus(id int64, status string, userID *int64) error {
	updates := map[string]any{"status": status}
	if userID != nil {
		updates["user_id"] = *userID
	}
	return r.DB().Model(&OAuthDeviceCode{}).
		Where("oauth_device_code_id = ?", id).
		Updates(updates).Error
}

// UpdateLastPollAt records when the device last polled to enforce the minimum
// polling interval.
func (r *oauthDeviceCodeRepository) UpdateLastPollAt(id int64) error {
	now := time.Now()
	return r.DB().Model(&OAuthDeviceCode{}).
		Where("oauth_device_code_id = ?", id).
		Update("last_poll_at", now).Error
}

// DeleteExpired removes device codes that expired before the given cutoff.
func (r *oauthDeviceCodeRepository) DeleteExpired(before time.Time) (int64, error) {
	result := r.DB().
		Where("expires_at < ?", before).
		Delete(&OAuthDeviceCode{})
	return result.RowsAffected, result.Error
}
