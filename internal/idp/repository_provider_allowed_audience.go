package idp

import (
	"strings"

	"gorm.io/gorm"
)

type IdentityProviderAllowedAudienceRepository interface {
	WithTx(tx *gorm.DB) IdentityProviderAllowedAudienceRepository
	FindByProviderID(idpID int64) ([]IdentityProviderAllowedAudience, error)
	ReplaceForProvider(tenantID, idpID int64, audiences []string) error
}

type identityProviderAllowedAudienceRepository struct {
	db *gorm.DB
}

func NewIdentityProviderAllowedAudienceRepository(db *gorm.DB) IdentityProviderAllowedAudienceRepository {
	return &identityProviderAllowedAudienceRepository{db: db}
}

func (r *identityProviderAllowedAudienceRepository) WithTx(tx *gorm.DB) IdentityProviderAllowedAudienceRepository {
	return &identityProviderAllowedAudienceRepository{db: tx}
}

func (r *identityProviderAllowedAudienceRepository) FindByProviderID(idpID int64) ([]IdentityProviderAllowedAudience, error) {
	var audiences []IdentityProviderAllowedAudience
	err := r.db.
		Where("identity_provider_id = ? AND deleted_at IS NULL", idpID).
		Find(&audiences).Error
	return audiences, err
}

func (r *identityProviderAllowedAudienceRepository) ReplaceForProvider(tenantID, idpID int64, audiences []string) error {
	if err := r.db.Unscoped().
		Where("identity_provider_id = ?", idpID).
		Delete(&IdentityProviderAllowedAudience{}).Error; err != nil {
		return err
	}

	rows := make([]IdentityProviderAllowedAudience, 0, len(audiences))
	seen := make(map[string]struct{}, len(audiences))
	for _, a := range audiences {
		a = strings.TrimSpace(a)
		if a == "" {
			continue
		}
		if _, ok := seen[a]; ok {
			continue
		}
		seen[a] = struct{}{}
		rows = append(rows, IdentityProviderAllowedAudience{
			TenantID:           tenantID,
			IdentityProviderID: idpID,
			Audience:           a,
		})
	}
	if len(rows) == 0 {
		return nil
	}
	return r.db.Create(&rows).Error
}
