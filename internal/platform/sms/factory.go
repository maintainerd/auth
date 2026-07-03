package sms

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/maintainerd/maintainerd-auth/internal/platform/config"
	"github.com/maintainerd/maintainerd-auth/internal/platform/crypto"
	"github.com/maintainerd/maintainerd-auth/internal/shared"
	"gorm.io/gorm"
)

func NewProviderFromDB(ctx context.Context, db *gorm.DB, tenantID int64) (Provider, error) {
	var cfg struct {
		Provider           string
		AccountSID         string
		AuthTokenEncrypted string
		FromNumber         string
		Status             string
	}

	err := db.WithContext(ctx).
		Table("sms_config").
		Select("provider, account_sid, auth_token_encrypted, from_number, status").
		Where("tenant_id = ? AND status = ? AND deleted_at IS NULL", tenantID, shared.StatusActive).
		First(&cfg).Error

	if err != nil {
		if err != gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("sms: lookup config for tenant %d: %w", tenantID, err)
		}
		err = db.WithContext(ctx).
			Table("sms_config sc").
			Select("sc.provider, sc.account_sid, sc.auth_token_encrypted, sc.from_number, sc.status").
			Joins("JOIN tenants t ON sc.tenant_id = t.tenant_id").
			Where("t.is_system = true AND sc.status = ? AND sc.deleted_at IS NULL", shared.StatusActive).
			First(&cfg).Error
		if err != nil {
			return nil, fmt.Errorf("sms: no active sms_config for tenant %d or system tenant", tenantID)
		}
	}

	token := cfg.AuthTokenEncrypted
	if token != "" {
		decrypted, decErr := crypto.DecryptString(token, config.AppEncryptionKey)
		if decErr != nil {
			return nil, fmt.Errorf("sms: decrypt auth token: %w", decErr)
		}
		token = decrypted
	}

	return NewProvider(ctx, ProviderConfig{
		Provider:    cfg.Provider,
		TwilioSID:   cfg.AccountSID,
		TwilioToken: token,
		TwilioFrom:  cfg.FromNumber,
	})
}

func NewProvider(ctx context.Context, cfg ProviderConfig) (Provider, error) {
	switch cfg.Provider {
	case "twilio":
		return newTwilioProvider(cfg), nil
	case "sns":
		return newSNSProvider(ctx, cfg)
	case "vonage":
		return newVonageProvider(cfg), nil
	case "log", "":
		return &logProvider{}, nil
	default:
		return nil, fmt.Errorf("sms: unknown provider %q", cfg.Provider)
	}
}

type logProvider struct{}

func (logProvider) Send(_ context.Context, to, body string) error {
	slog.Info("SMS (log provider)", "to", to, "body", body)
	return nil
}
