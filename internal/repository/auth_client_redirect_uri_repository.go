package repository

import (
	"errors"

	"github.com/maintainerd/auth/internal/model"
	"gorm.io/gorm"
)

type AuthClientRedirectURIRepository interface {
	BaseRepositoryMethods[model.AuthClientRedirectURI]
	WithTx(tx *gorm.DB) AuthClientRedirectURIRepository
	FindByName(redirectUri string, authClientID int64) (*model.AuthClientRedirectURI, error)
}

type authClientRedirectURIRepository struct {
	*BaseRepository[model.AuthClientRedirectURI]
	db *gorm.DB
}

func NewAuthClientRedirectURIRepository(db *gorm.DB) AuthClientRedirectURIRepository {
	return &authClientRedirectURIRepository{
		BaseRepository: NewBaseRepository[model.AuthClientRedirectURI](db, "auth_client_redirect_uri_uuid", "auth_client_redirect_uri_id"),
		db:             db,
	}
}

func (r *authClientRedirectURIRepository) WithTx(tx *gorm.DB) AuthClientRedirectURIRepository {
	return &authClientRedirectURIRepository{
		BaseRepository: NewBaseRepository[model.AuthClientRedirectURI](tx, "auth_client_redirect_uri_uuid", "auth_client_redirect_uri_id"),
		db:             tx,
	}
}

func (r *authClientRedirectURIRepository) FindByName(redirectUri string, authClientID int64) (*model.AuthClientRedirectURI, error) {
	var authClientRedirectUri model.AuthClientRedirectURI
	err := r.db.Where("redirect_uri = ? AND auth_client_id = ?", redirectUri, authClientID).First(&authClientRedirectUri).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}

	return &authClientRedirectUri, nil
}
