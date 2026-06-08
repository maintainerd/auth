package mfa

import (
	"errors"

	"github.com/maintainerd/auth/internal/platform/database"
	"gorm.io/gorm"
)

type UserSMSPhoneRepository interface {
	BaseRepositoryMethods[UserSMSPhone]
	WithTx(tx *gorm.DB) UserSMSPhoneRepository
	FindByUserID(userID int64) (*UserSMSPhone, error)
	DeleteByUserID(userID int64) error
}

type userSMSPhoneRepository struct {
	*BaseRepository[UserSMSPhone]
}

func NewUserSMSPhoneRepository(db *gorm.DB) UserSMSPhoneRepository {
	return &userSMSPhoneRepository{
		BaseRepository: database.NewBaseRepository[UserSMSPhone](db, "mfa_phone_uuid", "mfa_phone_id"),
	}
}

func (r *userSMSPhoneRepository) WithTx(tx *gorm.DB) UserSMSPhoneRepository {
	return &userSMSPhoneRepository{
		BaseRepository: r.BaseRepository.WithTx(tx),
	}
}

func (r *userSMSPhoneRepository) FindByUserID(userID int64) (*UserSMSPhone, error) {
	var phone UserSMSPhone
	err := r.DB().Where("user_id = ?", userID).First(&phone).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &phone, nil
}

func (r *userSMSPhoneRepository) DeleteByUserID(userID int64) error {
	return r.DB().Where("user_id = ?", userID).Delete(&UserSMSPhone{}).Error
}
