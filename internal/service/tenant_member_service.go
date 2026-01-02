package service

import (
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/model"
	"github.com/maintainerd/auth/internal/repository"
	"gorm.io/gorm"
)

type TenantMemberServiceDataResult struct {
	TenantMemberUUID uuid.UUID
	TenantID         int64
	UserID           int64
	Role             string
	User             *UserServiceDataResult
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

type TenantMemberService interface {
	Create(tenantID int64, userID int64, role string) (*TenantMemberServiceDataResult, error)
	CreateByUserUUID(tenantID int64, userUUID uuid.UUID, role string) (*TenantMemberServiceDataResult, error)
	GetByUUID(tenantMemberUUID uuid.UUID) (*TenantMemberServiceDataResult, error)
	GetByTenantAndUser(tenantID int64, userID int64) (*TenantMemberServiceDataResult, error)
	ListByTenant(tenantID int64) ([]TenantMemberServiceDataResult, error)
	ListByUser(userID int64) ([]TenantMemberServiceDataResult, error)
	UpdateRole(tenantMemberUUID uuid.UUID, role string) (*TenantMemberServiceDataResult, error)
	DeleteByUUID(tenantMemberUUID uuid.UUID) error
	IsUserInTenant(userID int64, tenantUUID uuid.UUID) (bool, error)
}

type tenantMemberService struct {
	db               *gorm.DB
	tenantMemberRepo repository.TenantMemberRepository
	userRepo         repository.UserRepository
	tenantRepo       repository.TenantRepository
}

func NewTenantMemberService(db *gorm.DB, tenantMemberRepo repository.TenantMemberRepository, userRepo repository.UserRepository, tenantRepo repository.TenantRepository) TenantMemberService {
	return &tenantMemberService{
		db:               db,
		tenantMemberRepo: tenantMemberRepo,
		userRepo:         userRepo,
		tenantRepo:       tenantRepo,
	}
}

func (s *tenantMemberService) Create(tenantID int64, userID int64, role string) (*TenantMemberServiceDataResult, error) {
	var created *model.TenantMember
	err := s.db.Transaction(func(tx *gorm.DB) error {
		repo := s.tenantMemberRepo.WithTx(tx)
		tu := &model.TenantMember{
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
	return toTenantMemberServiceDataResult(created), nil
}

func (s *tenantMemberService) CreateByUserUUID(tenantID int64, userUUID uuid.UUID, role string) (*TenantMemberServiceDataResult, error) {
	// First get the user to retrieve the user_id
	user, err := s.userRepo.FindByUUID(userUUID)
	if err != nil || user == nil {
		return nil, errors.New("user not found")
	}

	// Check if user is already a member of this tenant
	existing, _ := s.tenantMemberRepo.FindByTenantAndUser(tenantID, user.UserID)
	if existing != nil {
		return nil, errors.New("user is already a member of this tenant")
	}

	result, err := s.Create(tenantID, user.UserID, role)
	if err != nil {
		return nil, err
	}

	// Populate user information in the result
	result.User = toUserServiceDataResult(user)

	return result, nil
}

func (s *tenantMemberService) GetByUUID(tenantMemberUUID uuid.UUID) (*TenantMemberServiceDataResult, error) {
	tu, err := s.tenantMemberRepo.FindByTenantMemberUUID(tenantMemberUUID)
	if err != nil || tu == nil {
		return nil, errors.New("tenant member not found")
	}
	return toTenantMemberServiceDataResult(tu), nil
}

func (s *tenantMemberService) GetByTenantAndUser(tenantID int64, userID int64) (*TenantMemberServiceDataResult, error) {
	tu, err := s.tenantMemberRepo.FindByTenantAndUser(tenantID, userID)
	if err != nil || tu == nil {
		return nil, errors.New("tenant member not found")
	}
	return toTenantMemberServiceDataResult(tu), nil
}

func (s *tenantMemberService) ListByTenant(tenantID int64) ([]TenantMemberServiceDataResult, error) {
	tus, err := s.tenantMemberRepo.FindAllByTenant(tenantID)
	if err != nil {
		return nil, err
	}

	result := make([]TenantMemberServiceDataResult, len(tus))
	for i, tu := range tus {
		dr := toTenantMemberServiceDataResult(&tu)

		// Fetch user information
		user, err := s.userRepo.FindByID(tu.UserID)
		if err == nil && user != nil {
			dr.User = toUserServiceDataResult(user)
		}

		result[i] = *dr
	}
	return result, nil
}

func (s *tenantMemberService) ListByUser(userID int64) ([]TenantMemberServiceDataResult, error) {
	tus, err := s.tenantMemberRepo.FindAllByUser(userID)
	if err != nil {
		return nil, err
	}

	result := make([]TenantMemberServiceDataResult, len(tus))
	for i, tu := range tus {
		result[i] = *toTenantMemberServiceDataResult(&tu)
	}
	return result, nil
}

func (s *tenantMemberService) UpdateRole(tenantMemberUUID uuid.UUID, role string) (*TenantMemberServiceDataResult, error) {
	var updated *model.TenantMember
	err := s.db.Transaction(func(tx *gorm.DB) error {
		repo := s.tenantMemberRepo.WithTx(tx)
		tu, err := repo.FindByTenantMemberUUID(tenantMemberUUID)
		if err != nil {
			return err
		}
		if tu == nil {
			return errors.New("tenant member not found")
		}
		tu.Role = role
		updated, err = repo.CreateOrUpdate(tu)
		return err
	})
	if err != nil {
		return nil, err
	}

	result := toTenantMemberServiceDataResult(updated)

	// Fetch and populate user information
	user, err := s.userRepo.FindByID(updated.UserID)
	if err == nil && user != nil {
		result.User = toUserServiceDataResult(user)
	}

	return result, nil
}

func (s *tenantMemberService) DeleteByUUID(tenantMemberUUID uuid.UUID) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		repo := s.tenantMemberRepo.WithTx(tx)
		tu, err := repo.FindByTenantMemberUUID(tenantMemberUUID)
		if err != nil {
			return err
		}
		if tu == nil {
			return errors.New("tenant member not found")
		}
		return repo.DeleteByUUID(tenantMemberUUID)
	})
}

// IsUserInTenant checks if a user is a member of the specified tenant
func (s *tenantMemberService) IsUserInTenant(userID int64, tenantUUID uuid.UUID) (bool, error) {
	// First get the tenant to retrieve tenant_id
	tenant, err := s.tenantRepo.FindByUUID(tenantUUID)
	if err != nil || tenant == nil {
		return false, errors.New("tenant not found")
	}

	// Check if user is in tenant_members
	tenantMember, err := s.tenantMemberRepo.FindByTenantAndUser(tenant.TenantID, userID)
	if err != nil {
		return false, err
	}

	return tenantMember != nil, nil
}

func toTenantMemberServiceDataResult(tu *model.TenantMember) *TenantMemberServiceDataResult {
	return &TenantMemberServiceDataResult{
		TenantMemberUUID: tu.TenantMemberUUID,
		TenantID:         tu.TenantID,
		UserID:           tu.UserID,
		Role:             tu.Role,
		CreatedAt:        tu.CreatedAt,
		UpdatedAt:        tu.UpdatedAt,
	}
}
