package mfa

import (
	"errors"
	"time"

	"github.com/maintainerd/auth/internal/platform/database"
	"gorm.io/gorm"
)

type UserMFABackupCodeRepository interface {
	BaseRepositoryMethods[UserMFABackupCode]
	WithTx(tx *gorm.DB) UserMFABackupCodeRepository
	CreateBulk(codes []*UserMFABackupCode) error
	FindUnusedByUserID(userID int64) ([]UserMFABackupCode, error)
	FindByUserIDAndCodeHash(userID int64, codeHash string) (*UserMFABackupCode, error)
	MarkUsed(id int64) error
	DeleteAllByUserID(userID int64) error
}

type userMFABackupCodeRepository struct {
	*BaseRepository[UserMFABackupCode]
}

func NewUserMFABackupCodeRepository(db *gorm.DB) UserMFABackupCodeRepository {
	return &userMFABackupCodeRepository{
		BaseRepository: database.NewBaseRepository[UserMFABackupCode](db, "backup_code_uuid", "backup_code_id"),
	}
}

func (r *userMFABackupCodeRepository) WithTx(tx *gorm.DB) UserMFABackupCodeRepository {
	return &userMFABackupCodeRepository{
		BaseRepository: r.BaseRepository.WithTx(tx),
	}
}

func (r *userMFABackupCodeRepository) CreateBulk(codes []*UserMFABackupCode) error {
	return r.DB().Create(&codes).Error
}

func (r *userMFABackupCodeRepository) FindUnusedByUserID(userID int64) ([]UserMFABackupCode, error) {
	var codes []UserMFABackupCode
	err := r.DB().
		Where("user_id = ? AND used = false", userID).
		Find(&codes).Error
	return codes, err
}

func (r *userMFABackupCodeRepository) FindByUserIDAndCodeHash(userID int64, codeHash string) (*UserMFABackupCode, error) {
	var code UserMFABackupCode
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

func (r *userMFABackupCodeRepository) MarkUsed(id int64) error {
	now := time.Now()
	return r.DB().Model(&UserMFABackupCode{}).
		Where("backup_code_id = ?", id).
		Updates(map[string]interface{}{
			"used":    true,
			"used_at": now,
		}).Error
}

func (r *userMFABackupCodeRepository) DeleteAllByUserID(userID int64) error {
	return r.DB().
		Where("user_id = ?", userID).
		Delete(&UserMFABackupCode{}).Error
}
