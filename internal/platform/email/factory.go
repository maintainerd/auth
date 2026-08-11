package email

import (
	"context"
	"fmt"

	"github.com/maintainerd/maintainerd-auth/internal/platform/crypto"
	"github.com/maintainerd/maintainerd-auth/internal/shared"
	"gorm.io/gorm"
)

// NewProviderFromDB reads the tenant-scoped email_config row, decrypts the
// stored password, and returns an SMTP Provider. When no config exists for the
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
		Status            string
	}

	err := db.WithContext(ctx).
		Table("email_config").
		Select("provider, host, port, username, password_encrypted, from_address, from_name, status").
		Where("tenant_id = ? AND status = ? AND deleted_at IS NULL", tenantID, shared.StatusActive).
		First(&cfg).Error

	if err != nil {
		if err != gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("email: lookup config for tenant %d: %w", tenantID, err)
		}
		err = db.WithContext(ctx).
			Table("email_config ec").
			Select("ec.provider, ec.host, ec.port, ec.username, ec.password_encrypted, "+
				"ec.from_address, ec.from_name, ec.status").
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
	}
	return NewProvider(ctx, pc)
}

// NewProvider returns the SMTP Provider. maintainerd delivers email over SMTP
// only; any provider (SES, Mailgun, SendGrid, …) is used via its SMTP relay. An
// empty provider is treated as smtp for backward compatibility.
func NewProvider(_ context.Context, cfg ProviderConfig) (Provider, error) {
	switch cfg.Provider {
	case "smtp", "":
		return newSMTPProvider(cfg), nil
	default:
		return nil, fmt.Errorf("email: unsupported provider %q (only smtp is supported)", cfg.Provider)
	}
}
