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

type UserTOTPSecretRepository interface {
	BaseRepositoryMethods[UserTOTPSecret]
	WithTx(tx *gorm.DB) UserTOTPSecretRepository
	FindByUserID(userID int64) (*UserTOTPSecret, error)
	Upsert(secret *UserTOTPSecret) error
	Enable(userID int64) error
	Disable(userID int64) error
	UpdateLastUsed(userID int64) error
	DeleteByUserID(userID int64) error
}

type userTOTPSecretRepository struct {
	*BaseRepository[UserTOTPSecret]
}

func NewUserTOTPSecretRepository(db *gorm.DB) UserTOTPSecretRepository {
	return &userTOTPSecretRepository{
		BaseRepository: NewBaseRepository[UserTOTPSecret](db, "totp_secret_uuid", "totp_secret_id"),
	}
}

func (r *userTOTPSecretRepository) WithTx(tx *gorm.DB) UserTOTPSecretRepository {
	return &userTOTPSecretRepository{BaseRepository: r.BaseRepository.WithTx(tx)}
}

func (r *userTOTPSecretRepository) FindByUserID(userID int64) (*UserTOTPSecret, error) {
	var s UserTOTPSecret
	err := r.DB().Where("user_id = ?", userID).First(&s).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &s, nil
}

func (r *userTOTPSecretRepository) Upsert(secret *UserTOTPSecret) error {
	var existing UserTOTPSecret
	err := r.DB().Where("user_id = ?", secret.UserID).First(&existing).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return r.DB().Create(secret).Error
		}
		return err
	}
	return r.DB().Model(&existing).Updates(map[string]any{
		"secret":     secret.Secret,
		"is_enabled": secret.IsEnabled,
		"updated_at": time.Now(),
	}).Error
}

func (r *userTOTPSecretRepository) Enable(userID int64) error {
	now := time.Now()
	return r.DB().Model(&UserTOTPSecret{}).
		Where("user_id = ?", userID).
		Updates(map[string]any{
			"is_enabled":  true,
			"enrolled_at": now,
			"updated_at":  now,
		}).Error
}

func (r *userTOTPSecretRepository) Disable(userID int64) error {
	return r.DB().Model(&UserTOTPSecret{}).
		Where("user_id = ?", userID).
		Updates(map[string]any{
			"is_enabled": false,
			"updated_at": time.Now(),
		}).Error
}

func (r *userTOTPSecretRepository) UpdateLastUsed(userID int64) error {
	now := time.Now()
	return r.DB().Model(&UserTOTPSecret{}).
		Where("user_id = ?", userID).
		Updates(map[string]any{
			"last_used_at": now,
			"updated_at":   now,
		}).Error
}

func (r *userTOTPSecretRepository) DeleteByUserID(userID int64) error {
	return r.DB().Where("user_id = ?", userID).Delete(&UserTOTPSecret{}).Error
}

type UserWebAuthnCredentialRepository interface {
	BaseRepositoryMethods[UserWebAuthnCredential]
	WithTx(tx *gorm.DB) UserWebAuthnCredentialRepository
	FindByUserID(userID int64) ([]UserWebAuthnCredential, error)
	FindByCredentialKeyID(credentialKeyID string) (*UserWebAuthnCredential, error)
	// CreateCredential persists a new credential without shadowing the base Create method.
	CreateCredential(cred *UserWebAuthnCredential) error
	UpdateSignCount(credentialID int64, signCount int64) error
	UpdateLastUsed(credentialID int64) error
	// DeleteCredentialByID removes the credential matching both IDs for ownership safety.
	DeleteCredentialByID(credentialID int64, userID int64) error
	DeleteAllByUserID(userID int64) error
}

type userWebAuthnCredentialRepository struct {
	*BaseRepository[UserWebAuthnCredential]
}

func NewUserWebAuthnCredentialRepository(db *gorm.DB) UserWebAuthnCredentialRepository {
	return &userWebAuthnCredentialRepository{
		BaseRepository: NewBaseRepository[UserWebAuthnCredential](db, "credential_uuid", "credential_id"),
	}
}

func (r *userWebAuthnCredentialRepository) WithTx(tx *gorm.DB) UserWebAuthnCredentialRepository {
	return &userWebAuthnCredentialRepository{BaseRepository: r.BaseRepository.WithTx(tx)}
}

func (r *userWebAuthnCredentialRepository) FindByUserID(userID int64) ([]UserWebAuthnCredential, error) {
	var creds []UserWebAuthnCredential
	err := r.DB().Where("user_id = ?", userID).Find(&creds).Error
	return creds, err
}

func (r *userWebAuthnCredentialRepository) FindByCredentialKeyID(credentialKeyID string) (*UserWebAuthnCredential, error) {
	var c UserWebAuthnCredential
	err := r.DB().Where("credential_key_id = ?", credentialKeyID).First(&c).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &c, nil
}

func (r *userWebAuthnCredentialRepository) CreateCredential(cred *UserWebAuthnCredential) error {
	return r.DB().Create(cred).Error
}

func (r *userWebAuthnCredentialRepository) UpdateSignCount(credentialID int64, signCount int64) error {
	return r.DB().Model(&UserWebAuthnCredential{}).
		Where("credential_id = ?", credentialID).
		Updates(map[string]any{
			"sign_count": signCount,
			"updated_at": time.Now(),
		}).Error
}

func (r *userWebAuthnCredentialRepository) UpdateLastUsed(credentialID int64) error {
	now := time.Now()
	return r.DB().Model(&UserWebAuthnCredential{}).
		Where("credential_id = ?", credentialID).
		Updates(map[string]any{
			"last_used_at": now,
			"updated_at":   now,
		}).Error
}

func (r *userWebAuthnCredentialRepository) DeleteCredentialByID(credentialID int64, userID int64) error {
	return r.DB().
		Where("credential_id = ? AND user_id = ?", credentialID, userID).
		Delete(&UserWebAuthnCredential{}).Error
}

func (r *userWebAuthnCredentialRepository) DeleteAllByUserID(userID int64) error {
	return r.DB().Where("user_id = ?", userID).Delete(&UserWebAuthnCredential{}).Error
}
