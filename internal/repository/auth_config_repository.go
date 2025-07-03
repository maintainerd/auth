package repository

import (
	"github.com/maintainerd/auth/internal/model"
	"gorm.io/gorm"
)

type AuthConfigRepository interface {
	Create(role *model.AuthConfig) error
	FetchLatest() (*model.AuthConfig, error)
}

type authConfigRepository struct {
	db *gorm.DB
}

func NewAuthConfigRepository(db *gorm.DB) AuthConfigRepository {
	return &authConfigRepository{db}
}

func (r *authConfigRepository) Create(authConfig *model.AuthConfig) error {
	return r.db.Create(authConfig).Error
}

func (r *authConfigRepository) FetchLatest() (*model.AuthConfig, error) {
	var authConfig model.AuthConfig
	err := r.db.Order("created_at DESC").First(&authConfig).Error
	if err != nil {
		return nil, err
	}
	return &authConfig, nil
}
