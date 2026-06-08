package email

import (
	"context"

	"gorm.io/gorm"
)

type SendEmailParams struct {
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
	provider, err := NewProviderFromDB(ctx, db, 0)
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

func GetLogoURL(ctx context.Context, db *gorm.DB) string {
	if db == nil {
		return ""
	}
	var logo string
	_ = db.WithContext(ctx).
		Table("email_config").
		Select("logo_url").
		Where("tenant_id = ? AND status = ? AND deleted_at IS NULL", 0, "active").
		Scan(&logo).Error
	return logo
}
