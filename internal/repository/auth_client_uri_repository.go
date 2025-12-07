package repository

import (
	"errors"

	"github.com/maintainerd/auth/internal/model"
	"gorm.io/gorm"
)

type AuthClientURIRepository interface {
	BaseRepositoryMethods[model.AuthClientURI]
	WithTx(tx *gorm.DB) AuthClientURIRepository
	FindByURIAndType(uri string, uriType string, authClientID int64) (*model.AuthClientURI, error)
	FindByAuthClientIDAndType(authClientID int64, uriType string) ([]model.AuthClientURI, error)
}

type authClientURIRepository struct {
	*BaseRepository[model.AuthClientURI]
	db *gorm.DB
}

func NewAuthClientURIRepository(db *gorm.DB) AuthClientURIRepository {
	return &authClientURIRepository{
		BaseRepository: NewBaseRepository[model.AuthClientURI](db, "auth_client_uri_uuid", "auth_client_uri_id"),
		db:             db,
	}
}

func (r *authClientURIRepository) WithTx(tx *gorm.DB) AuthClientURIRepository {
	return &authClientURIRepository{
		BaseRepository: NewBaseRepository[model.AuthClientURI](tx, "auth_client_uri_uuid", "auth_client_uri_id"),
		db:             tx,
	}
}

func (r *authClientURIRepository) FindByURIAndType(uri string, uriType string, authClientID int64) (*model.AuthClientURI, error) {
	var authClientURI model.AuthClientURI
	err := r.db.Where("uri = ? AND type = ? AND auth_client_id = ?", uri, uriType, authClientID).First(&authClientURI).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}

	return &authClientURI, nil
}

func (r *authClientURIRepository) FindByAuthClientIDAndType(authClientID int64, uriType string) ([]model.AuthClientURI, error) {
	var authClientURIs []model.AuthClientURI
	err := r.db.Where("auth_client_id = ? AND type = ?", authClientID, uriType).Find(&authClientURIs).Error

	if err != nil {
		return nil, err
	}

	return authClientURIs, nil
}
