package repository

import (
	"errors"

	"github.com/maintainerd/auth/internal/model"
	"gorm.io/gorm"
)

type AuthClientURIRepository interface {
	BaseRepositoryMethods[model.AuthClientURI]
	WithTx(tx *gorm.DB) AuthClientURIRepository
	FindByUUIDAndTenantID(uuid string, tenantID int64) (*model.AuthClientURI, error)
	FindByURIAndType(uri string, uriType string, authClientID int64, tenantID int64) (*model.AuthClientURI, error)
	FindByAuthClientIDAndType(authClientID int64, uriType string, tenantID int64) ([]model.AuthClientURI, error)
	DeleteByUUIDAndTenantID(uuid string, tenantID int64) error
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

func (r *authClientURIRepository) FindByUUIDAndTenantID(uuid string, tenantID int64) (*model.AuthClientURI, error) {
	var authClientURI model.AuthClientURI
	err := r.db.Where("auth_client_uri_uuid = ? AND tenant_id = ?", uuid, tenantID).First(&authClientURI).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}

	return &authClientURI, nil
}

func (r *authClientURIRepository) FindByURIAndType(uri string, uriType string, authClientID int64, tenantID int64) (*model.AuthClientURI, error) {
	var authClientURI model.AuthClientURI
	err := r.db.Where("uri = ? AND type = ? AND auth_client_id = ? AND tenant_id = ?", uri, uriType, authClientID, tenantID).First(&authClientURI).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}

	return &authClientURI, nil
}

func (r *authClientURIRepository) FindByAuthClientIDAndType(authClientID int64, uriType string, tenantID int64) ([]model.AuthClientURI, error) {
	var authClientURIs []model.AuthClientURI
	err := r.db.Where("auth_client_id = ? AND type = ? AND tenant_id = ?", authClientID, uriType, tenantID).Find(&authClientURIs).Error

	if err != nil {
		return nil, err
	}

	return authClientURIs, nil
}

func (r *authClientURIRepository) DeleteByUUIDAndTenantID(uuid string, tenantID int64) error {
	result := r.db.Where("auth_client_uri_uuid = ? AND tenant_id = ?", uuid, tenantID).Delete(&model.AuthClientURI{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}
