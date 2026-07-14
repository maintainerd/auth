package idp

import (
	"time"

	"github.com/google/uuid"
	"github.com/maintainerd/maintainerd-auth/internal/platform/crypto"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type IdentityProvider struct {
	IdentityProviderID   int64     `gorm:"column:identity_provider_id;primaryKey"`
	IdentityProviderUUID uuid.UUID `gorm:"column:identity_provider_uuid"`
	TenantID             int64     `gorm:"column:tenant_id"`
	Name                 string    `gorm:"column:name"`
	DisplayName          string    `gorm:"column:display_name"`
	Provider             string    `gorm:"column:provider"`
	ProviderType         string    `gorm:"column:provider_type"`
	Identifier           string    `gorm:"column:identifier"`
	// Security-critical and queried fields promoted out of the config JSONB blob.
	// Issuer and ProviderClientID are columns (indexed; issuer is unique where set).
	// ProviderClientSecretEncrypted holds the upstream OAuth2 client secret
	// encrypted at rest; it is written but never returned by any read/list endpoint.
	Issuer                        *string `gorm:"column:issuer"`
	ProviderClientID              *string `gorm:"column:provider_client_id"`
	ProviderClientSecretEncrypted *string `gorm:"column:provider_client_secret_encrypted"`
	AllowJITProvisioning          bool    `gorm:"column:allow_jit_provisioning;default:false"`
	AllowRegistration             bool    `gorm:"column:allow_registration;default:true"`
	AllowTokenFederation          bool    `gorm:"column:allow_token_federation;default:false"`

	Config               datatypes.JSON `gorm:"column:config;type:jsonb;not null;default:'{}'"`
	CertificateExpiresAt *time.Time     `gorm:"column:certificate_expires_at"`
	Status               string         `gorm:"column:status;not null;default:'inactive'"`
	IsDefault            bool           `gorm:"column:is_default;not null;default:false"`
	IsSystem             bool           `gorm:"column:is_system;not null;default:false"`
	CreatedBy            *int64         `gorm:"column:created_by"`
	UpdatedBy            *int64         `gorm:"column:updated_by"`
	CreatedAt            time.Time      `gorm:"column:created_at;not null;autoCreateTime"`
	UpdatedAt            time.Time      `gorm:"column:updated_at;not null;autoUpdateTime"`
	DeletedAt            gorm.DeletedAt `gorm:"column:deleted_at;index"`

	// Relationships
	Tenant           *Tenant                           `gorm:"foreignKey:TenantID;references:TenantID"`
	EmailDomains     []IdentityProviderEmailDomain     `gorm:"foreignKey:IdentityProviderID;references:IdentityProviderID"`
	AllowedAudiences []IdentityProviderAllowedAudience `gorm:"foreignKey:IdentityProviderID;references:IdentityProviderID"`
}

func (IdentityProvider) TableName() string {
	return "identity_providers"
}

// DecryptedProviderClientSecret returns the upstream OAuth2 client secret in
// plaintext, decrypting the stored ProviderClientSecretEncrypted column. It
// returns "" when no secret is configured. Only federation/broker flows that
// actually call the upstream provider should use this; read/list endpoints must
// never load it.
func (ip *IdentityProvider) DecryptedProviderClientSecret() string {
	if ip == nil || ip.ProviderClientSecretEncrypted == nil || *ip.ProviderClientSecretEncrypted == "" {
		return ""
	}
	return crypto.SafeDecryptAtRest(*ip.ProviderClientSecretEncrypted)
}

// DecryptedProviderClientSecretStrict returns the upstream OAuth2 client secret in
// plaintext, failing closed on any decrypt error. Unlike DecryptedProviderClientSecret
// (which falls open via SafeDecryptAtRest and returns the raw ciphertext on failure),
// this variant surfaces the error so security-critical federation/broker exchange
// paths never POST an undecryptable ciphertext blob upstream. It returns ("", nil)
// when no secret is configured.
func (ip *IdentityProvider) DecryptedProviderClientSecretStrict() (string, error) {
	if ip == nil || ip.ProviderClientSecretEncrypted == nil || *ip.ProviderClientSecretEncrypted == "" {
		return "", nil
	}
	return crypto.DecryptAtRest(*ip.ProviderClientSecretEncrypted)
}

// IssuerOrEmpty / ProviderClientIDOrEmpty dereference the nullable column pointers.
func (ip *IdentityProvider) IssuerOrEmpty() string {
	if ip == nil || ip.Issuer == nil {
		return ""
	}
	return *ip.Issuer
}

func (ip *IdentityProvider) ProviderClientIDOrEmpty() string {
	if ip == nil || ip.ProviderClientID == nil {
		return ""
	}
	return *ip.ProviderClientID
}

// IdentityProviderEmailDomain maps an email domain to an identity provider for
// home-realm discovery. One domain belongs to exactly one IdP per tenant
// (enforced by uq_idp_email_domain). Replaces the former config.email_domains.
type IdentityProviderEmailDomain struct {
	IdentityProviderEmailDomainID   int64          `gorm:"column:identity_provider_email_domain_id;primaryKey"`
	IdentityProviderEmailDomainUUID uuid.UUID      `gorm:"column:identity_provider_email_domain_uuid"`
	TenantID                        int64          `gorm:"column:tenant_id"`
	IdentityProviderID              int64          `gorm:"column:identity_provider_id"`
	Domain                          string         `gorm:"column:domain"`
	CreatedAt                       time.Time      `gorm:"column:created_at;autoCreateTime"`
	DeletedAt                       gorm.DeletedAt `gorm:"column:deleted_at;index"`
}

func (IdentityProviderEmailDomain) TableName() string {
	return "identity_provider_email_domains"
}

func (d *IdentityProviderEmailDomain) BeforeCreate(tx *gorm.DB) (err error) {
	if d.IdentityProviderEmailDomainUUID == uuid.Nil {
		d.IdentityProviderEmailDomainUUID = uuid.New()
	}
	return
}

type IdentityProviderAllowedAudience struct {
	IdentityProviderAllowedAudienceID   int64          `gorm:"column:identity_provider_allowed_audience_id;primaryKey"`
	IdentityProviderAllowedAudienceUUID uuid.UUID      `gorm:"column:identity_provider_allowed_audience_uuid"`
	TenantID                            int64          `gorm:"column:tenant_id"`
	IdentityProviderID                  int64          `gorm:"column:identity_provider_id"`
	Audience                            string         `gorm:"column:audience"`
	CreatedAt                           time.Time      `gorm:"column:created_at;autoCreateTime"`
	DeletedAt                           gorm.DeletedAt `gorm:"column:deleted_at;index"`
}

func (IdentityProviderAllowedAudience) TableName() string {
	return "identity_provider_allowed_audiences"
}

func (a *IdentityProviderAllowedAudience) BeforeCreate(tx *gorm.DB) (err error) {
	if a.IdentityProviderAllowedAudienceUUID == uuid.Nil {
		a.IdentityProviderAllowedAudienceUUID = uuid.New()
	}
	return
}

func (ip *IdentityProvider) BeforeCreate(tx *gorm.DB) (err error) {
	if ip.IdentityProviderUUID == uuid.Nil {
		ip.IdentityProviderUUID = uuid.New()
	}
	return
}
