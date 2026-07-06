package oauth

import (
	"github.com/maintainerd/maintainerd-auth/internal/platform/database"
	"gorm.io/gorm"
)

type SigningKeyRepository interface {
	FindActiveByTenantID(tenantID int64) ([]SigningKey, error)
	FindByKID(kid string) (*SigningKey, error)
	Create(key *SigningKey) error
	RetireByKID(kid string) error
	MarkCompromised(kid string) error
}

type signingKeyRepository struct {
	*BaseRepository[SigningKey]
}

func NewSigningKeyRepository(db *gorm.DB) SigningKeyRepository {
	return &signingKeyRepository{
		BaseRepository: database.NewBaseRepository[SigningKey](db, "signing_key_uuid", "signing_key_id"),
	}
}

func (r *signingKeyRepository) FindActiveByTenantID(tenantID int64) ([]SigningKey, error) {
	var keys []SigningKey
	err := r.DB().
		Where("tenant_id = ? AND status = ?", tenantID, "active").
		Find(&keys).Error
	return keys, err
}

func (r *signingKeyRepository) FindByKID(kid string) (*SigningKey, error) {
	var key SigningKey
	err := r.DB().Where("kid = ?", kid).First(&key).Error
	if err != nil {
		return nil, err
	}
	return &key, nil
}

func (r *signingKeyRepository) Create(key *SigningKey) error {
	return r.DB().Create(key).Error
}

func (r *signingKeyRepository) RetireByKID(kid string) error {
	return r.DB().Model(&SigningKey{}).Where("kid = ?", kid).Update("status", "retired").Error
}

func (r *signingKeyRepository) MarkCompromised(kid string) error {
	return r.DB().Model(&SigningKey{}).Where("kid = ?", kid).Update("status", "compromised").Error
}
