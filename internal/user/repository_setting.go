package user

import (
	"github.com/maintainerd/auth/internal/platform/database"
	"gorm.io/gorm"
)

type UserSettingRepository interface {
	BaseRepositoryMethods[UserSetting]
	WithTx(tx *gorm.DB) UserSettingRepository
	FindByUserID(userID int64) (*UserSetting, error)
	UpdateByUserID(userID int64, updatedUserSetting *UserSetting) error
	DeleteByUserID(userID int64) error
}

type userSettingRepository struct {
	*BaseRepository[UserSetting]
}

func NewUserSettingRepository(db *gorm.DB) UserSettingRepository {
	return &userSettingRepository{
		BaseRepository: database.NewBaseRepository[UserSetting](db, "user_setting_uuid", "user_setting_id"),
	}
}

func (r *userSettingRepository) WithTx(tx *gorm.DB) UserSettingRepository {
	return &userSettingRepository{
		BaseRepository: r.BaseRepository.WithTx(tx),
	}
}

func (r *userSettingRepository) FindByUserID(userID int64) (*UserSetting, error) {
	var userSetting UserSetting
	err := r.DB().Where("user_id = ?", userID).First(&userSetting).Error
	return &userSetting, err
}

func (r *userSettingRepository) UpdateByUserID(userID int64, updatedUserSetting *UserSetting) error {
	return r.DB().Model(&UserSetting{}).
		Where("user_id = ?", userID).
		Updates(updatedUserSetting).Error
}

func (r *userSettingRepository) DeleteByUserID(userID int64) error {
	return r.DB().Where("user_id = ?", userID).Delete(&UserSetting{}).Error
}
