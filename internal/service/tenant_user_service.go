package service

import (
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/model"
	"github.com/maintainerd/auth/internal/repository"
	"gorm.io/gorm"
)

type TenantUserServiceDataResult struct {
	TenantUserUUID uuid.UUID
	TenantID       int64
	UserID         int64
	Role           string
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

type TenantUserService interface {
	Create(tenantID int64, userID int64, role string) (*TenantUserServiceDataResult, error)
	GetByUUID(tenantUserUUID uuid.UUID) (*TenantUserServiceDataResult, error)
	GetByTenantAndUser(tenantID int64, userID int64) (*TenantUserServiceDataResult, error)
	ListByTenant(tenantID int64) ([]TenantUserServiceDataResult, error)
	ListByUser(userID int64) ([]TenantUserServiceDataResult, error)
	UpdateRole(tenantUserUUID uuid.UUID, role string) (*TenantUserServiceDataResult, error)
	DeleteByUUID(tenantUserUUID uuid.UUID) error
}

type tenantUserService struct {
	db             *gorm.DB
	tenantUserRepo repository.TenantUserRepository
}

func NewTenantUserService(db *gorm.DB, tenantUserRepo repository.TenantUserRepository) TenantUserService {
	return &tenantUserService{
		db:             db,
		tenantUserRepo: tenantUserRepo,
	}
}

func (s *tenantUserService) Create(tenantID int64, userID int64, role string) (*TenantUserServiceDataResult, error) {
	var created *model.TenantUser
	err := s.db.Transaction(func(tx *gorm.DB) error {
		repo := s.tenantUserRepo.WithTx(tx)
		tu := &model.TenantUser{
			TenantID: tenantID,
			UserID:   userID,
			Role:     role,
		}
		var err error
		created, err = repo.Create(tu)
		return err
	})
	if err != nil {
		return nil, err
	}
	return toTenantUserServiceDataResult(created), nil
}

func (s *tenantUserService) GetByUUID(tenantUserUUID uuid.UUID) (*TenantUserServiceDataResult, error) {
	tu, err := s.tenantUserRepo.FindByTenantUserUUID(tenantUserUUID)
	if err != nil || tu == nil {
		return nil, errors.New("tenant user not found")
	}
	return toTenantUserServiceDataResult(tu), nil
}

func (s *tenantUserService) GetByTenantAndUser(tenantID int64, userID int64) (*TenantUserServiceDataResult, error) {
	tu, err := s.tenantUserRepo.FindByTenantAndUser(tenantID, userID)
	if err != nil || tu == nil {
		return nil, errors.New("tenant user not found")
	}
	return toTenantUserServiceDataResult(tu), nil
}

func (s *tenantUserService) ListByTenant(tenantID int64) ([]TenantUserServiceDataResult, error) {
	tus, err := s.tenantUserRepo.FindAllByTenant(tenantID)
	if err != nil {
		return nil, err
	}

	result := make([]TenantUserServiceDataResult, len(tus))
	for i, tu := range tus {
		result[i] = *toTenantUserServiceDataResult(&tu)
	}
	return result, nil
}

func (s *tenantUserService) ListByUser(userID int64) ([]TenantUserServiceDataResult, error) {
	tus, err := s.tenantUserRepo.FindAllByUser(userID)
	if err != nil {
		return nil, err
	}

	result := make([]TenantUserServiceDataResult, len(tus))
	for i, tu := range tus {
		result[i] = *toTenantUserServiceDataResult(&tu)
	}
	return result, nil
}

func (s *tenantUserService) UpdateRole(tenantUserUUID uuid.UUID, role string) (*TenantUserServiceDataResult, error) {
	var updated *model.TenantUser
	err := s.db.Transaction(func(tx *gorm.DB) error {
		repo := s.tenantUserRepo.WithTx(tx)
		tu, err := repo.FindByTenantUserUUID(tenantUserUUID)
		if err != nil {
			return err
		}
		if tu == nil {
			return errors.New("tenant user not found")
		}
		tu.Role = role
		updated, err = repo.CreateOrUpdate(tu)
		return err
	})
	if err != nil {
		return nil, err
	}
	return toTenantUserServiceDataResult(updated), nil
}

func (s *tenantUserService) DeleteByUUID(tenantUserUUID uuid.UUID) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		repo := s.tenantUserRepo.WithTx(tx)
		tu, err := repo.FindByTenantUserUUID(tenantUserUUID)
		if err != nil {
			return err
		}
		if tu == nil {
			return errors.New("tenant user not found")
		}
		return repo.DeleteByUUID(tenantUserUUID)
	})
}

func toTenantUserServiceDataResult(tu *model.TenantUser) *TenantUserServiceDataResult {
	return &TenantUserServiceDataResult{
		TenantUserUUID: tu.TenantUserUUID,
		TenantID:       tu.TenantID,
		UserID:         tu.UserID,
		Role:           tu.Role,
		CreatedAt:      tu.CreatedAt,
		UpdatedAt:      tu.UpdatedAt,
	}
}
