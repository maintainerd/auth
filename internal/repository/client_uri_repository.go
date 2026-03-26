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
	FindByURIAndType(uri string, uriType string, ClientID int64, tenantID int64) (*model.ClientURI, error)
	FindByClientIDAndType(ClientID int64, uriType string, tenantID int64) ([]model.ClientURI, error)
	DeleteByUUIDAndTenantID(uuid string, tenantID int64) error
}

type clientURIRepository struct {
	*BaseRepository[model.ClientURI]
	db *gorm.DB
}

func NewClientURIRepository(db *gorm.DB) ClientURIRepository {
	return &clientURIRepository{
		BaseRepository: NewBaseRepository[model.ClientURI](db, "client_uri_uuid", "client_uri_id"),
		db:             db,
	}
}

func (r *clientURIRepository) WithTx(tx *gorm.DB) ClientURIRepository {
	return &clientURIRepository{
		BaseRepository: NewBaseRepository[model.ClientURI](tx, "client_uri_uuid", "client_uri_id"),
		db:             tx,
	}
}

func (r *clientURIRepository) FindByUUIDAndTenantID(uuid string, tenantID int64) (*model.ClientURI, error) {
	var ClientURI model.ClientURI
	err := r.db.Where("client_uri_uuid = ? AND tenant_id = ?", uuid, tenantID).First(&ClientURI).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}

	return &ClientURI, nil
}

func (r *clientURIRepository) FindByURIAndType(uri string, uriType string, ClientID int64, tenantID int64) (*model.ClientURI, error) {
	var ClientURI model.ClientURI
	err := r.db.Where("uri = ? AND type = ? AND client_id = ? AND tenant_id = ?", uri, uriType, ClientID, tenantID).First(&ClientURI).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}

	return &ClientURI, nil
}

func (r *clientURIRepository) FindByClientIDAndType(ClientID int64, uriType string, tenantID int64) ([]model.ClientURI, error) {
	var ClientURIs []model.ClientURI
	err := r.db.Where("client_id = ? AND type = ? AND tenant_id = ?", ClientID, uriType, tenantID).Find(&ClientURIs).Error

	if err != nil {
		return nil, err
	}

	return ClientURIs, nil
}

func (r *clientURIRepository) DeleteByUUIDAndTenantID(uuid string, tenantID int64) error {
	result := r.db.Where("client_uri_uuid = ? AND tenant_id = ?", uuid, tenantID).Delete(&model.ClientURI{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}
