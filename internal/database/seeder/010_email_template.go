package seeder

import (
	"log/slog"

	"github.com/maintainerd/auth/internal/model"
	"github.com/maintainerd/auth/internal/templates/emailtemplate"
	"gorm.io/gorm"
)

func SeedEmailTemplates(db *gorm.DB, tenantID int64) error {
	templates := []model.EmailTemplate{
		newEmailTemplate(
			tenantID,
			"internal:user:invite",
			"You're Invited to Join Our Organization!",
			emailtemplate.InviteEmailHTML,
			`You're invited to join our organization. Accept the invite: {{.InviteURL}}`,
		),
		newEmailTemplate(
			tenantID,
			"internal:user:password:reset",
			"Password Reset Request",
			emailtemplate.ForgotPasswordEmailHTML,
			emailtemplate.ForgotPasswordEmailPlain,
		),
	}

	for _, t := range templates {
		var existing model.EmailTemplate
		err := db.Where("name = ? AND tenant_id = ?", t.Name, tenantID).First(&existing).Error
		if err == nil {
			slog.Info("Email template already exists, skipping", "name", t.Name)
			continue
		}

		if err := db.Create(&t).Error; err != nil {
			return err
		}

		slog.Info("Email template seeded", "name", t.Name)
	}

	return nil
}

func newEmailTemplate(tenantID int64, name, subject, bodyHTML, bodyPlain string) model.EmailTemplate {
	return model.EmailTemplate{
		TenantID:  tenantID,
		Name:      name,
		Subject:   subject,
		BodyHTML:  bodyHTML,
		BodyPlain: &bodyPlain,
		Status:    "active",
	}
}
