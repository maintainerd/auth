package mfa

import (
	"errors"
	"time"

	"github.com/maintainerd/auth/internal/platform/database"
	"gorm.io/gorm"
)

type UserMFAWebAuthnCredentialRepository interface {
	BaseRepositoryMethods[UserMFAWebAuthnCredential]
	FindByUUID(uuid any, preloads ...string) (*UserMFAWebAuthnCredential, error)
	WithTx(tx *gorm.DB) UserMFAWebAuthnCredentialRepository
	FindByUserID(userID int64) ([]UserMFAWebAuthnCredential, error)
	FindByCredentialKeyID(credentialKeyID string) (*UserMFAWebAuthnCredential, error)
	// CreateCredential persists a new credential without shadowing the base Create method.
	CreateCredential(cred *UserMFAWebAuthnCredential) error
	UpdateSignCount(credentialID int64, signCount int64) error
	UpdateLastUsed(credentialID int64) error
	// DeleteCredentialByID removes the credential matching both IDs for ownership safety.
	DeleteCredentialByID(credentialID int64, userID int64) error
	DeleteAllByUserID(userID int64) error
}

type userMFAWebAuthnCredentialRepository struct {
	*BaseRepository[UserMFAWebAuthnCredential]
}

func NewUserMFAWebAuthnCredentialRepository(db *gorm.DB) UserMFAWebAuthnCredentialRepository {
	return &userMFAWebAuthnCredentialRepository{
		BaseRepository: database.NewBaseRepository[UserMFAWebAuthnCredential](db, "credential_uuid", "credential_id"),
	}
}

func (r *userMFAWebAuthnCredentialRepository) WithTx(tx *gorm.DB) UserMFAWebAuthnCredentialRepository {
	return &userMFAWebAuthnCredentialRepository{BaseRepository: r.BaseRepository.WithTx(tx)}
}

func (r *userMFAWebAuthnCredentialRepository) FindByUserID(userID int64) ([]UserMFAWebAuthnCredential, error) {
	var creds []UserMFAWebAuthnCredential
	err := r.DB().Where("user_id = ?", userID).Find(&creds).Error
	return creds, err
}

func (r *userMFAWebAuthnCredentialRepository) FindByCredentialKeyID(credentialKeyID string) (*UserMFAWebAuthnCredential, error) {
	var c UserMFAWebAuthnCredential
	err := r.DB().Where("credential_key_id = ?", credentialKeyID).First(&c).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &c, nil
}

func (r *userMFAWebAuthnCredentialRepository) CreateCredential(cred *UserMFAWebAuthnCredential) error {
	return r.DB().Create(cred).Error
}

func (r *userMFAWebAuthnCredentialRepository) UpdateSignCount(credentialID int64, signCount int64) error {
	return r.DB().Model(&UserMFAWebAuthnCredential{}).
		Where("credential_id = ?", credentialID).
		Updates(map[string]any{
			"sign_count": signCount,
			"updated_at": time.Now(),
		}).Error
}

func (r *userMFAWebAuthnCredentialRepository) UpdateLastUsed(credentialID int64) error {
	now := time.Now()
	return r.DB().Model(&UserMFAWebAuthnCredential{}).
		Where("credential_id = ?", credentialID).
		Updates(map[string]any{
			"last_used_at": now,
			"updated_at":   now,
		}).Error
}

func (r *userMFAWebAuthnCredentialRepository) DeleteCredentialByID(credentialID int64, userID int64) error {
	return r.DB().
		Where("credential_id = ? AND user_id = ?", credentialID, userID).
		Delete(&UserMFAWebAuthnCredential{}).Error
}

func (r *userMFAWebAuthnCredentialRepository) DeleteAllByUserID(userID int64) error {
	return r.DB().Where("user_id = ?", userID).Delete(&UserMFAWebAuthnCredential{}).Error
}
