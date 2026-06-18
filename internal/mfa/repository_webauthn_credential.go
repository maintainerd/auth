package mfa

import (
	"errors"
	"time"

	"github.com/maintainerd/auth/internal/platform/database"
	"gorm.io/gorm"
)

type UserWebAuthnCredentialRepository interface {
	BaseRepositoryMethods[UserWebAuthnCredential]
	FindByUUID(uuid any, preloads ...string) (*UserWebAuthnCredential, error)
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
		BaseRepository: database.NewBaseRepository[UserWebAuthnCredential](db, "credential_uuid", "credential_id"),
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
