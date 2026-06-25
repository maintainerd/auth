package idp

import (
	"errors"
	"strings"

	"gorm.io/gorm"
)

// IdentityProviderEmailDomainRepository manages the identity_provider_email_domains
// child table that backs home-realm discovery. A domain maps to exactly one IdP
// per tenant (uq_idp_email_domain), so FindByTenantAndDomain is a single indexed
// lookup replacing the former full-scan-and-JSON-parse over every provider.
type IdentityProviderEmailDomainRepository interface {
	WithTx(tx *gorm.DB) IdentityProviderEmailDomainRepository
	FindByTenantAndDomain(tenantID int64, domain string) (*IdentityProviderEmailDomain, error)
	FindByProviderID(idpID int64) ([]IdentityProviderEmailDomain, error)
	// ReplaceForProvider sets a provider's email-domain membership to exactly the
	// provided set (deletes existing, inserts the new normalized/deduped list).
	ReplaceForProvider(tenantID, idpID int64, domains []string) error
}

type identityProviderEmailDomainRepository struct {
	db *gorm.DB
}

func NewIdentityProviderEmailDomainRepository(db *gorm.DB) IdentityProviderEmailDomainRepository {
	return &identityProviderEmailDomainRepository{db: db}
}

func (r *identityProviderEmailDomainRepository) WithTx(tx *gorm.DB) IdentityProviderEmailDomainRepository {
	return &identityProviderEmailDomainRepository{db: tx}
}

func (r *identityProviderEmailDomainRepository) FindByTenantAndDomain(tenantID int64, domain string) (*IdentityProviderEmailDomain, error) {
	var d IdentityProviderEmailDomain
	err := r.db.
		Where("tenant_id = ? AND domain = ? AND deleted_at IS NULL", tenantID, strings.ToLower(strings.TrimSpace(domain))).
		First(&d).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &d, nil
}

func (r *identityProviderEmailDomainRepository) FindByProviderID(idpID int64) ([]IdentityProviderEmailDomain, error) {
	var domains []IdentityProviderEmailDomain
	err := r.db.
		Where("identity_provider_id = ? AND deleted_at IS NULL", idpID).
		Find(&domains).Error
	return domains, err
}

func (r *identityProviderEmailDomainRepository) ReplaceForProvider(tenantID, idpID int64, domains []string) error {
	// Hard-delete the current set, then insert the new one. Hard delete keeps the
	// table free of soft-deleted rows that would otherwise accumulate on every
	// edit (they are already excluded from the unique index via deleted_at).
	if err := r.db.Unscoped().
		Where("identity_provider_id = ?", idpID).
		Delete(&IdentityProviderEmailDomain{}).Error; err != nil {
		return err
	}

	rows := make([]IdentityProviderEmailDomain, 0, len(domains))
	seen := make(map[string]struct{}, len(domains))
	for _, d := range domains {
		d = strings.ToLower(strings.TrimSpace(d))
		if d == "" {
			continue
		}
		if _, ok := seen[d]; ok {
			continue
		}
		seen[d] = struct{}{}
		rows = append(rows, IdentityProviderEmailDomain{
			TenantID:           tenantID,
			IdentityProviderID: idpID,
			Domain:             d,
		})
	}
	if len(rows) == 0 {
		return nil
	}
	return r.db.Create(&rows).Error
}
