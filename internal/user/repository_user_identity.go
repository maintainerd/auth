package user

import (
	"errors"

	"gorm.io/gorm"
)

type UserIdentityRepository interface {
	BaseRepositoryMethods[UserIdentity]
	WithTx(tx *gorm.DB) UserIdentityRepository
	FindByUserID(userID int64) ([]UserIdentity, error)
	FindByUserIDAndClientID(userID int64, clientID int64) (*UserIdentity, error)
	// FindByProviderAndSub looks up an identity by the provider slug and the external subject.
	// Used by federation to match an incoming OIDC token to a known user.
	FindByProviderAndSub(provider string, sub string) (*UserIdentity, error)
	// FindByUserIDAndProvider returns the first identity for a user with the given provider slug.
	FindByUserIDAndProvider(userID int64, provider string) (*UserIdentity, error)
	// FindByIdentityProviderID lists all identities linked to a configured IDP.
	FindByIdentityProviderID(idpID int64) ([]UserIdentity, error)
	DeleteByUserID(userID int64) error
}

type userIdentityRepository struct {
	*BaseRepository[UserIdentity]
}

func NewUserIdentityRepository(db *gorm.DB) UserIdentityRepository {
	return &userIdentityRepository{
		BaseRepository: NewBaseRepository[UserIdentity](db, "user_identity_uuid", "user_identity_id"),
	}
}

func (r *userIdentityRepository) WithTx(tx *gorm.DB) UserIdentityRepository {
	return &userIdentityRepository{
		BaseRepository: r.BaseRepository.WithTx(tx),
	}
}

func (r *userIdentityRepository) FindByUserID(userID int64) ([]UserIdentity, error) {
	var identities []UserIdentity
	err := r.DB().Where("user_id = ?", userID).Find(&identities).Error
	return identities, err
}

func (r *userIdentityRepository) FindByUserIDAndClientID(userID int64, clientID int64) (*UserIdentity, error) {
	var identity UserIdentity
	err := r.DB().Where("user_id = ? AND client_id = ?", userID, clientID).First(&identity).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &identity, nil
}

func (r *userIdentityRepository) FindByProviderAndSub(provider string, sub string) (*UserIdentity, error) {
	var identity UserIdentity
	err := r.DB().Where("provider = ? AND sub = ?", provider, sub).First(&identity).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &identity, nil
}

func (r *userIdentityRepository) FindByUserIDAndProvider(userID int64, provider string) (*UserIdentity, error) {
	var identity UserIdentity
	err := r.DB().Where("user_id = ? AND provider = ?", userID, provider).First(&identity).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &identity, nil
}

func (r *userIdentityRepository) FindByIdentityProviderID(idpID int64) ([]UserIdentity, error) {
	var identities []UserIdentity
	err := r.DB().Where("identity_provider_id = ?", idpID).Find(&identities).Error
	return identities, err
}

func (r *userIdentityRepository) DeleteByUserID(userID int64) error {
	return r.DB().Where("user_id = ?", userID).Delete(&UserIdentity{}).Error
}
