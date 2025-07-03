package service

import (
	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/model"
	"github.com/maintainerd/auth/internal/repository"
)

type AuthConfigService interface {
	Create(role *model.AuthConfig) error
	GetLatestConfig() (*model.AuthConfig, error)
}

type authConfigService struct {
	repo repository.AuthConfigRepository
}

func (s *authConfigService) Create(authConfig *model.AuthConfig) error {
	authConfig.AuthConfigUUID = uuid.New()
	return s.repo.Create(authConfig)
}

func NewAuthConfigService(repo repository.AuthConfigRepository) AuthConfigService {
	return &authConfigService{repo}
}

func (s *authConfigService) GetLatestConfig() (*model.AuthConfig, error) {
	return s.repo.FetchLatest()
}
