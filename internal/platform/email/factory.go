package email

import (
	"context"
	"fmt"

	"github.com/maintainerd/maintainerd-auth/internal/platform/crypto"
	"github.com/maintainerd/maintainerd-auth/internal/shared"
	"gorm.io/gorm"
)

// NewProviderFromDB reads the tenant-scoped email_config row, decrypts the
// stored password, and returns a Provider. When no config exists for the
// given tenant, it falls back to the system tenant.
func NewProviderFromDB(ctx context.Context, db *gorm.DB, tenantID int64) (Provider, error) {
	var cfg struct {
		Provider          string
		Host              string
		Port              int
		Username          string
		PasswordEncrypted string
		FromAddress       string
		FromName          string
		APIKey            string
		Domain            string
		Region            string
		Status            string
	}

	err := db.WithContext(ctx).
		Table("email_config").
		Select("provider, host, port, username, password_encrypted, from_address, from_name, "+
			"'' AS api_key, '' AS domain, '' AS region, status").
		Where("tenant_id = ? AND status = ? AND deleted_at IS NULL", tenantID, shared.StatusActive).
		First(&cfg).Error

	if err != nil {
		if err != gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("email: lookup config for tenant %d: %w", tenantID, err)
		}
		err = db.WithContext(ctx).
			Table("email_config ec").
			Select("ec.provider, ec.host, ec.port, ec.username, ec.password_encrypted, "+
				"ec.from_address, ec.from_name, '' AS api_key, '' AS domain, '' AS region, ec.status").
			Joins("JOIN tenants t ON ec.tenant_id = t.tenant_id").
			Where("t.is_system = true AND ec.status = ? AND ec.deleted_at IS NULL", shared.StatusActive).
			First(&cfg).Error
		if err != nil {
			return nil, fmt.Errorf("email: no active email_config for tenant %d or system tenant", tenantID)
		}
	}

	password := cfg.PasswordEncrypted
	if password != "" {
		// DecryptAtRest, not DecryptString. notifier.EmailConfigService stores this with
		// EncryptAtRest, which wraps the ciphertext in a "k1:<key-id>:" envelope
		// so a rotated key can still decrypt its own rows. DecryptString expects
		// bare base64, so it choked on the envelope's ':' — every send failed with
		// "invalid base64: illegal base64 data at input byte 2", and no mail or
		// SMS could ever go out.
		decrypted, decErr := crypto.DecryptAtRest(password)
		if decErr != nil {
			return nil, fmt.Errorf("email: decrypt password: %w", decErr)
		}
		password = decrypted
	}

	pc := ProviderConfig{
		Provider:    cfg.Provider,
		FromAddress: cfg.FromAddress,
		FromName:    cfg.FromName,
		Host:        cfg.Host,
		Port:        cfg.Port,
		Username:    cfg.Username,
		Password:    password,
		APIKey:      cfg.APIKey,
		Domain:      cfg.Domain,
		Region:      cfg.Region,
	}
	return NewProvider(ctx, pc)
}

// NewProvider returns the Provider implementation for cfg.Provider.
func NewProvider(ctx context.Context, cfg ProviderConfig) (Provider, error) {
	switch cfg.Provider {
	case "smtp", "":
		return newSMTPProvider(cfg), nil
	case "ses":
		return newSESProvider(ctx, cfg)
	case "sendgrid":
		return newSendGridProvider(cfg), nil
	case "postmark":
		return newPostmarkProvider(cfg), nil
	case "mailgun":
		return newMailgunProvider(cfg), nil
	case "resend":
		return newResendProvider(cfg), nil
	default:
		return nil, fmt.Errorf("email: unknown provider %q", cfg.Provider)
	}
}
