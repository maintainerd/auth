package repository

import (
	"github.com/maintainerd/auth/internal/model"
	"gorm.io/gorm"
)

type UserSettingRepository interface {
	BaseRepositoryMethods[model.UserSetting]
	WithTx(tx *gorm.DB) UserSettingRepository
	FindByUserID(userID int64) (*model.UserSetting, error)
	UpdateByUserID(userID int64, updatedUserSetting *model.UserSetting) error
	DeleteByUserID(userID int64) error
}

type userSettingRepository struct {
	*BaseRepository[model.UserSetting]
	db *gorm.DB
}

func NewUserSettingRepository(db *gorm.DB) UserSettingRepository {
	return &userSettingRepository{
		BaseRepository: NewBaseRepository[model.UserSetting](db, "user_setting_uuid", "user_setting_id"),
		db:             db,
	}
}

func (r *userSettingRepository) WithTx(tx *gorm.DB) UserSettingRepository {
	return &userSettingRepository{
		BaseRepository: NewBaseRepository[model.UserSetting](tx, "user_setting_uuid", "user_setting_id"),
		db:             tx,
	}
}

func (r *userSettingRepository) FindByUserID(userID int64) (*model.UserSetting, error) {
	var userSetting model.UserSetting
	err := r.db.Where("user_id = ?", userID).First(&userSetting).Error
	return &userSetting, err
}

func (r *userSettingRepository) UpdateByUserID(userID int64, updatedUserSetting *model.UserSetting) error {
	return r.db.Model(&model.UserSetting{}).
		Where("user_id = ?", userID).
		Updates(updatedUserSetting).Error
}

func (r *userSettingRepository) DeleteByUserID(userID int64) error {
	return r.db.Where("user_id = ?", userID).Delete(&model.UserSetting{}).Error
}
