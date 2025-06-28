package services

import (
	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/models"
	"github.com/maintainerd/auth/internal/repositories"
)

type RoleService interface {
	Create(role *models.Role) error
	GetAll() ([]models.Role, error)
	GetByUUID(roleUUID uuid.UUID) (*models.Role, error)
	UpdateByUUID(roleUUID uuid.UUID, updatedRole *models.Role) error
	DeleteByUUID(roleUUID uuid.UUID) error
}

type roleService struct {
	repo repositories.RoleRepository
}

func NewRoleService(repo repositories.RoleRepository) RoleService {
	return &roleService{repo}
}

func (s *roleService) Create(role *models.Role) error {
	role.RoleUUID = uuid.New()
	return s.repo.Create(role)
}

func (s *roleService) GetAll() ([]models.Role, error) {
	return s.repo.FindAll()
}

func (s *roleService) GetByUUID(roleUUID uuid.UUID) (*models.Role, error) {
	return s.repo.FindByUUID(roleUUID)
}

func (s *roleService) UpdateByUUID(roleUUID uuid.UUID, updatedRole *models.Role) error {
	return s.repo.UpdateByUUID(roleUUID, updatedRole)
}

func (s *roleService) DeleteByUUID(roleUUID uuid.UUID) error {
	return s.repo.DeleteByUUID(roleUUID)
}
