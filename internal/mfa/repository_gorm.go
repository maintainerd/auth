package mfa

import (
	"errors"
	"time"

	"gorm.io/gorm"
)

type UserBackupCodeRepository interface {
	BaseRepositoryMethods[UserBackupCode]
	WithTx(tx *gorm.DB) UserBackupCodeRepository
	CreateBulk(codes []*UserBackupCode) error
	FindUnusedByUserID(userID int64) ([]UserBackupCode, error)
	FindByUserIDAndCodeHash(userID int64, codeHash string) (*UserBackupCode, error)
	MarkUsed(id int64) error
	DeleteAllByUserID(userID int64) error
}

type userBackupCodeRepository struct {
	*BaseRepository[UserBackupCode]
}

func NewUserBackupCodeRepository(db *gorm.DB) UserBackupCodeRepository {
	return &userBackupCodeRepository{
		BaseRepository: NewBaseRepository[UserBackupCode](db, "backup_code_uuid", "backup_code_id"),
	}
}

func (r *userBackupCodeRepository) WithTx(tx *gorm.DB) UserBackupCodeRepository {
	return &userBackupCodeRepository{
		BaseRepository: r.BaseRepository.WithTx(tx),
	}
}

func (r *userBackupCodeRepository) CreateBulk(codes []*UserBackupCode) error {
	return r.DB().Create(&codes).Error
}

func (r *userBackupCodeRepository) FindUnusedByUserID(userID int64) ([]UserBackupCode, error) {
	var codes []UserBackupCode
	err := r.DB().
		Where("user_id = ? AND used = false", userID).
		Find(&codes).Error
	return codes, err
}

func (r *userBackupCodeRepository) FindByUserIDAndCodeHash(userID int64, codeHash string) (*UserBackupCode, error) {
	var code UserBackupCode
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
	return r.DB().Model(&UserBackupCode{}).
		Where("backup_code_id = ?", id).
		Updates(map[string]interface{}{
			"used":    true,
			"used_at": now,
		}).Error
}

func (r *userBackupCodeRepository) DeleteAllByUserID(userID int64) error {
	return r.DB().
		Where("user_id = ?", userID).
		Delete(&UserBackupCode{}).Error
}
