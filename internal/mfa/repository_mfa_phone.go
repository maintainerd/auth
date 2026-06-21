package mfa

import (
	"errors"

	"github.com/maintainerd/auth/internal/platform/database"
	"gorm.io/gorm"
)

type UserMFAPhoneRepository interface {
	BaseRepositoryMethods[UserMFAPhone]
	WithTx(tx *gorm.DB) UserMFAPhoneRepository
	FindByUserID(userID int64) (*UserMFAPhone, error)
	DeleteByUserID(userID int64) error
}

type userMFAPhoneRepository struct {
	*BaseRepository[UserMFAPhone]
}

func NewUserMFAPhoneRepository(db *gorm.DB) UserMFAPhoneRepository {
	return &userMFAPhoneRepository{
		BaseRepository: database.NewBaseRepository[UserMFAPhone](db, "mfa_phone_uuid", "mfa_phone_id"),
	}
}

func (r *userMFAPhoneRepository) WithTx(tx *gorm.DB) UserMFAPhoneRepository {
	return &userMFAPhoneRepository{
		BaseRepository: r.BaseRepository.WithTx(tx),
	}
}

func (r *userMFAPhoneRepository) FindByUserID(userID int64) (*UserMFAPhone, error) {
	var phone UserMFAPhone
	err := r.DB().Where("user_id = ?", userID).First(&phone).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &phone, nil
}

func (r *userMFAPhoneRepository) DeleteByUserID(userID int64) error {
	return r.DB().Where("user_id = ?", userID).Delete(&UserMFAPhone{}).Error
}
