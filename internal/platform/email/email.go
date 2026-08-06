package email

import (
	"context"

	"gorm.io/gorm"
)

type SendEmailParams struct {
	// TenantID scopes which tenant's email_config is used. It comes from the
	// request context (set by the auth middleware). A zero value falls back to
	// the system tenant's config.
	TenantID  int64
	To        string
	From      string
	Subject   string
	BodyHTML  string
	BodyPlain string
}

// SendEmail sends an email using the DB-backed provider. When db is nil
// it falls back to a no-op — useful for tests that don't need real delivery.
// The entire flow is swappable in tests by overriding SendEmail.
var SendEmail = sendEmail

func sendEmail(ctx context.Context, db *gorm.DB, params SendEmailParams) error {
	if db == nil {
		return nil
	}
	provider, err := NewProviderFromDB(ctx, db, params.TenantID)
	if err != nil {
		return err
	}
	return provider.Send(ctx, SendParams{
		To:        params.To,
		From:      params.From,
		Subject:   params.Subject,
		BodyHTML:  params.BodyHTML,
		BodyPlain: params.BodyPlain,
	})
}

// GetLogoURL returns the tenant's email logo, falling back to the system
// tenant's when the tenant has not set one.
//
// The tenant was hardcoded to 0 here, which matches no real tenant — so every
// outgoing email rendered with an empty logo regardless of what the tenant had
// configured.
func GetLogoURL(ctx context.Context, db *gorm.DB, tenantID int64) string {
	if db == nil {
		return ""
	}
	var logo string
	_ = db.WithContext(ctx).
		Table("email_config").
		Select("logo_url").
		Where("tenant_id = ? AND status = ? AND deleted_at IS NULL", tenantID, "active").
		Scan(&logo).Error
	if logo != "" {
		return logo
	}
	_ = db.WithContext(ctx).
		Table("email_config ec").
		Select("ec.logo_url").
		Joins("JOIN tenants t ON ec.tenant_id = t.tenant_id").
		Where("t.is_system = true AND ec.status = ? AND ec.deleted_at IS NULL", "active").
		Scan(&logo).Error
	return logo
}
