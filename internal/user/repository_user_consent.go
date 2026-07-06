package user

import (
	"github.com/maintainerd/maintainerd-auth/internal/platform/database"
	"gorm.io/gorm"
)

type UserConsentRepository interface {
	BaseRepositoryMethods[UserConsent]
	WithTx(tx *gorm.DB) UserConsentRepository
	FindByUserID(userID int64) ([]UserConsent, error)
	FindByUserAndType(userID int64, consentType string) ([]UserConsent, error)
	FindLatestByUserAndType(userID int64, consentType string) (*UserConsent, error)
	CreateConsent(consent *UserConsent) error
}

type userConsentRepository struct {
	*BaseRepository[UserConsent]
}

func NewUserConsentRepository(db *gorm.DB) UserConsentRepository {
	return &userConsentRepository{
		BaseRepository: database.NewBaseRepository[UserConsent](db, "user_consent_uuid", "user_consent_id"),
	}
}

func (r *userConsentRepository) WithTx(tx *gorm.DB) UserConsentRepository {
	return &userConsentRepository{
		BaseRepository: r.BaseRepository.WithTx(tx),
	}
}

func (r *userConsentRepository) FindByUserID(userID int64) ([]UserConsent, error) {
	var consents []UserConsent
	err := r.DB().Where("user_id = ?", userID).Order("created_at DESC").Find(&consents).Error
	return consents, err
}

func (r *userConsentRepository) FindByUserAndType(userID int64, consentType string) ([]UserConsent, error) {
	var consents []UserConsent
	err := r.DB().
		Where("user_id = ? AND consent_type = ?", userID, consentType).
		Order("created_at DESC").
		Find(&consents).Error
	return consents, err
}

func (r *userConsentRepository) FindLatestByUserAndType(userID int64, consentType string) (*UserConsent, error) {
	var consent UserConsent
	err := r.DB().
		Where("user_id = ? AND consent_type = ?", userID, consentType).
		Order("created_at DESC").
		First(&consent).Error
	if err != nil {
		return nil, err
	}
	return &consent, nil
}

func (r *userConsentRepository) CreateConsent(consent *UserConsent) error {
	return r.DB().Create(consent).Error
}
