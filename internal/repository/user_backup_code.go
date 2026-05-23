package repository

import (
	"errors"
	"time"

	"github.com/maintainerd/auth/internal/model"
	"gorm.io/gorm"
)

type UserBackupCodeRepository interface {
	BaseRepositoryMethods[model.UserBackupCode]
	WithTx(tx *gorm.DB) UserBackupCodeRepository
	CreateBulk(codes []*model.UserBackupCode) error
	FindUnusedByUserID(userID int64) ([]model.UserBackupCode, error)
	FindByUserIDAndCodeHash(userID int64, codeHash string) (*model.UserBackupCode, error)
	MarkUsed(id int64) error
	DeleteAllByUserID(userID int64) error
}

type userBackupCodeRepository struct {
	*BaseRepository[model.UserBackupCode]
}

func NewUserBackupCodeRepository(db *gorm.DB) UserBackupCodeRepository {
	return &userBackupCodeRepository{
		BaseRepository: NewBaseRepository[model.UserBackupCode](db, "backup_code_uuid", "backup_code_id"),
	}
}

func (r *userBackupCodeRepository) WithTx(tx *gorm.DB) UserBackupCodeRepository {
	return &userBackupCodeRepository{
		BaseRepository: r.BaseRepository.WithTx(tx),
	}
}

func (r *userBackupCodeRepository) CreateBulk(codes []*model.UserBackupCode) error {
	return r.DB().Create(&codes).Error
}

func (r *userBackupCodeRepository) FindUnusedByUserID(userID int64) ([]model.UserBackupCode, error) {
	var codes []model.UserBackupCode
	err := r.DB().
		Where("user_id = ? AND used = false", userID).
		Find(&codes).Error
	return codes, err
}

func (r *userBackupCodeRepository) FindByUserIDAndCodeHash(userID int64, codeHash string) (*model.UserBackupCode, error) {
	var code model.UserBackupCode
	err := r.DB().
		Where("user_id = ? AND code_hash = ? AND used = false", userID, codeHash).
		First(&code).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &code, nil
}

func (r *userBackupCodeRepository) MarkUsed(id int64) error {
	now := time.Now()
	return r.DB().Model(&model.UserBackupCode{}).
		Where("backup_code_id = ?", id).
		Updates(map[string]interface{}{
			"used":    true,
			"used_at": now,
		}).Error
}

func (r *userBackupCodeRepository) DeleteAllByUserID(userID int64) error {
	return r.DB().
		Where("user_id = ?", userID).
		Delete(&model.UserBackupCode{}).Error
}
