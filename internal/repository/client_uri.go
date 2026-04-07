package repository

import (
	"errors"

	"github.com/maintainerd/auth/internal/model"
	"gorm.io/gorm"
)

type ClientURIRepository interface {
	BaseRepositoryMethods[model.ClientURI]
	WithTx(tx *gorm.DB) ClientURIRepository
	FindByUUIDAndTenantID(uuid string, tenantID int64) (*model.ClientURI, error)
	FindByURIAndType(uri string, uriType string, clientID int64, tenantID int64) (*model.ClientURI, error)
	FindByClientIDAndType(clientID int64, uriType string, tenantID int64) ([]model.ClientURI, error)
	DeleteByUUIDAndTenantID(uuid string, tenantID int64) error
}

type clientURIRepository struct {
	*BaseRepository[model.ClientURI]
}

func NewClientURIRepository(db *gorm.DB) ClientURIRepository {
	return &clientURIRepository{
		BaseRepository: NewBaseRepository[model.ClientURI](db, "client_uri_uuid", "client_uri_id"),
	}
}

func (r *clientURIRepository) WithTx(tx *gorm.DB) ClientURIRepository {
	return &clientURIRepository{
		BaseRepository: r.BaseRepository.WithTx(tx),
	}
}

func (r *clientURIRepository) FindByUUIDAndTenantID(uuid string, tenantID int64) (*model.ClientURI, error) {
	var clientURI model.ClientURI
	err := r.DB().Where("client_uri_uuid = ? AND tenant_id = ?", uuid, tenantID).First(&clientURI).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}

	return &clientURI, nil
}

func (r *clientURIRepository) FindByURIAndType(uri string, uriType string, clientID int64, tenantID int64) (*model.ClientURI, error) {
	var clientURI model.ClientURI
	err := r.DB().Where("uri = ? AND type = ? AND client_id = ? AND tenant_id = ?", uri, uriType, clientID, tenantID).First(&clientURI).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}

	return &clientURI, nil
}

func (r *clientURIRepository) FindByClientIDAndType(clientID int64, uriType string, tenantID int64) ([]model.ClientURI, error) {
	var clientURIs []model.ClientURI
	err := r.DB().Where("client_id = ? AND type = ? AND tenant_id = ?", clientID, uriType, tenantID).Find(&clientURIs).Error

	if err != nil {
		return nil, err
	}

	return clientURIs, nil
}

func (r *clientURIRepository) DeleteByUUIDAndTenantID(uuid string, tenantID int64) error {
	result := r.DB().Where("client_uri_uuid = ? AND tenant_id = ?", uuid, tenantID).Delete(&model.ClientURI{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}
