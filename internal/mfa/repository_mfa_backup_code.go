package mfa

import (
	"time"

	"github.com/maintainerd/maintainerd-auth/internal/platform/apperror"
	"github.com/maintainerd/maintainerd-auth/internal/platform/database"
	"gorm.io/gorm"
)

type UserMFABackupCodeRepository interface {
	BaseRepositoryMethods[UserMFABackupCode]
	WithTx(tx *gorm.DB) UserMFABackupCodeRepository
	CreateBulk(codes []*UserMFABackupCode) error
	FindUnusedByUserID(userID int64) ([]UserMFABackupCode, error)
	// FindByUserIDAndCodeHash is gone. Backup codes are stored bcrypt-hashed, so
	// an equality lookup on a digest can never match a row — the only correct
	// redemption path is FindUnusedByUserID + bcrypt.CompareHashAndPassword.
	// Keeping the method invited a caller to hash the submitted code with
	// SHA-256 and look it up, which is exactly the split-hash bug that made
	// account recovery permanently impossible.
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

func (r *userMFABackupCodeRepository) MarkUsed(id int64) error {
	now := time.Now()
	result := r.DB().Model(&UserMFABackupCode{}).
		Where("backup_code_id = ? AND used = false", id).
		Updates(map[string]interface{}{
			"used":    true,
			"used_at": now,
		})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return apperror.NewConflict("backup code already used")
	}
	return nil
}

func (r *userMFABackupCodeRepository) DeleteAllByUserID(userID int64) error {
	return r.DB().
		Where("user_id = ?", userID).
		Delete(&UserMFABackupCode{}).Error
}
