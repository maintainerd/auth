package service

import (
	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/model"
	"github.com/maintainerd/auth/internal/repository"
)

type RoleService interface {
	Create(role *model.Role) error
	GetAll() ([]model.Role, error)
	GetByUUID(roleUUID uuid.UUID) (*model.Role, error)
	UpdateByUUID(roleUUID uuid.UUID, updatedRole *model.Role) error
	DeleteByUUID(roleUUID uuid.UUID) error
}

type roleService struct {
	repo repository.RoleRepository
}

func NewRoleService(repo repository.RoleRepository) RoleService {
	return &roleService{repo}
}

func (s *roleService) Create(role *model.Role) error {
	role.RoleUUID = uuid.New()
	return s.repo.Create(role)
}

func (s *roleService) GetAll() ([]model.Role, error) {
	return s.repo.FindAll()
}

func (s *roleService) GetByUUID(roleUUID uuid.UUID) (*model.Role, error) {
	return s.repo.FindByUUID(roleUUID)
}

func (s *roleService) UpdateByUUID(roleUUID uuid.UUID, updatedRole *model.Role) error {
	return s.repo.UpdateByUUID(roleUUID, updatedRole)
}

func (s *roleService) DeleteByUUID(roleUUID uuid.UUID) error {
	return s.repo.DeleteByUUID(roleUUID)
}
