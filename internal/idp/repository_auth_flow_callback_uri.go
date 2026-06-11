package idp

import (
	"errors"

	"github.com/maintainerd/auth/internal/platform/database"
	"gorm.io/gorm"
)

type AuthFlowCallbackURIRepository interface {
	BaseRepositoryMethods[AuthFlowCallbackURI]
	WithTx(tx *gorm.DB) AuthFlowCallbackURIRepository
	FindByAuthFlowID(authFlowID int64) ([]AuthFlowCallbackURI, error)
	FindByAuthFlowIDPaginated(authFlowID int64, page, limit int) ([]AuthFlowCallbackURI, int64, error)
	FindByAuthFlowIDAndClientURIID(authFlowID, clientURIID int64) (*AuthFlowCallbackURI, error)
	DeleteByAuthFlowIDAndClientURIID(authFlowID, clientURIID int64) error
}

type authFlowCallbackURIRepository struct {
	*BaseRepository[AuthFlowCallbackURI]
}

func NewAuthFlowCallbackURIRepository(db *gorm.DB) AuthFlowCallbackURIRepository {
	return &authFlowCallbackURIRepository{
		BaseRepository: database.NewBaseRepository[AuthFlowCallbackURI](db, "auth_flow_callback_uri_uuid", "auth_flow_callback_uri_id"),
	}
}

func (r *authFlowCallbackURIRepository) WithTx(tx *gorm.DB) AuthFlowCallbackURIRepository {
	return &authFlowCallbackURIRepository{
		BaseRepository: r.BaseRepository.WithTx(tx),
	}
}

func (r *authFlowCallbackURIRepository) FindByAuthFlowID(authFlowID int64) ([]AuthFlowCallbackURI, error) {
	var rows []AuthFlowCallbackURI
	err := r.DB().Where("auth_flow_id = ?", authFlowID).Preload("ClientURI").Find(&rows).Error
	if err != nil {
		return nil, err
	}
	return rows, nil
}

func (r *authFlowCallbackURIRepository) FindByAuthFlowIDPaginated(authFlowID int64, page, limit int) ([]AuthFlowCallbackURI, int64, error) {
	var rows []AuthFlowCallbackURI
	var total int64

	query := r.DB().Where("auth_flow_id = ?", authFlowID)

	if err := query.Model(&AuthFlowCallbackURI{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * limit
	err := query.Preload("ClientURI").Offset(offset).Limit(limit).Find(&rows).Error
	if err != nil {
		return nil, 0, err
	}

	return rows, total, nil
}

func (r *authFlowCallbackURIRepository) FindByAuthFlowIDAndClientURIID(authFlowID, clientURIID int64) (*AuthFlowCallbackURI, error) {
	var row AuthFlowCallbackURI
	err := r.DB().Where("auth_flow_id = ? AND client_uri_id = ?", authFlowID, clientURIID).First(&row).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &row, nil
}

func (r *authFlowCallbackURIRepository) DeleteByAuthFlowIDAndClientURIID(authFlowID, clientURIID int64) error {
	return r.DB().Where("auth_flow_id = ? AND client_uri_id = ?", authFlowID, clientURIID).Delete(&AuthFlowCallbackURI{}).Error
}
