package repository

import (
	"errors"

	"github.com/maintainerd/auth/internal/model"
	"gorm.io/gorm"
)

// OAuthConsentGrantRepository defines data access operations for user consent
// grants per client.
type OAuthConsentGrantRepository interface {
	BaseRepositoryMethods[model.OAuthConsentGrant]
	WithTx(tx *gorm.DB) OAuthConsentGrantRepository
	FindByUserAndClient(userID, clientID int64) (*model.OAuthConsentGrant, error)
	Upsert(grant *model.OAuthConsentGrant) (*model.OAuthConsentGrant, error)
	DeleteByUserAndClient(userID, clientID int64) error
	FindByUserID(userID int64) ([]model.OAuthConsentGrant, error)
}

type oauthConsentGrantRepository struct {
	*BaseRepository[model.OAuthConsentGrant]
}

// NewOAuthConsentGrantRepository creates a new OAuthConsentGrantRepository.
func NewOAuthConsentGrantRepository(db *gorm.DB) OAuthConsentGrantRepository {
	return &oauthConsentGrantRepository{
		BaseRepository: NewBaseRepository[model.OAuthConsentGrant](db, "oauth_consent_grant_uuid", "oauth_consent_grant_id"),
	}
}

func (r *oauthConsentGrantRepository) WithTx(tx *gorm.DB) OAuthConsentGrantRepository {
	return &oauthConsentGrantRepository{
		BaseRepository: r.BaseRepository.WithTx(tx),
	}
}

// FindByUserAndClient looks up the consent grant for a user-client pair.
// Returns nil, nil when no consent exists.
func (r *oauthConsentGrantRepository) FindByUserAndClient(userID, clientID int64) (*model.OAuthConsentGrant, error) {
	var grant model.OAuthConsentGrant
	err := r.DB().
		Where("user_id = ? AND client_id = ?", userID, clientID).
		First(&grant).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &grant, nil
}

// Upsert creates a new consent grant or updates the scopes if one already
// exists for the user-client pair.
func (r *oauthConsentGrantRepository) Upsert(grant *model.OAuthConsentGrant) (*model.OAuthConsentGrant, error) {
	existing, err := r.FindByUserAndClient(grant.UserID, grant.ClientID)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		existing.Scopes = grant.Scopes
		if err := r.DB().Save(existing).Error; err != nil {
			return nil, err
		}
		return existing, nil
	}
	return r.Create(grant)
}

// DeleteByUserAndClient removes the consent grant for a user-client pair.
func (r *oauthConsentGrantRepository) DeleteByUserAndClient(userID, clientID int64) error {
	return r.DB().
		Where("user_id = ? AND client_id = ?", userID, clientID).
		Delete(&model.OAuthConsentGrant{}).Error
}

// FindByUserID returns all consent grants for a user.
func (r *oauthConsentGrantRepository) FindByUserID(userID int64) ([]model.OAuthConsentGrant, error) {
	var grants []model.OAuthConsentGrant
	err := r.DB().
		Preload("Client").
		Where("user_id = ?", userID).
		Find(&grants).Error
	return grants, err
}
