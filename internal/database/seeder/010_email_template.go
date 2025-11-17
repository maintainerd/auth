package seeder

import (
	"log"

	"github.com/maintainerd/auth/internal/model"
	emailtemplate "github.com/maintainerd/auth/internal/templates/email"
	"gorm.io/gorm"
)

func SeedEmailTemplates(db *gorm.DB) error {
	templates := []model.EmailTemplate{
		newEmailTemplate(
			"internal:user:invite",
			"You're Invited to Join Our Organization!",
			emailtemplate.InviteEmailHTML,
			`You're invited to join our organization. Accept the invite: {{.InviteURL}}`,
		),
		newEmailTemplate(
			"internal:user:password:reset",
			"Password Reset Request",
			emailtemplate.ForgotPasswordEmailHTML,
			emailtemplate.ForgotPasswordEmailPlain,
		),
	}

	for _, t := range templates {
		var existing model.EmailTemplate
		err := db.Where("name = ?", t.Name).First(&existing).Error
		if err == nil {
			log.Printf("⚠️ Email template '%s' already exists, skipping", t.Name)
			continue
		}

		if err := db.Create(&t).Error; err != nil {
			return err
		}

		log.Printf("✅ Email template '%s' seeded", t.Name)
	}

	return nil
}

func newEmailTemplate(name, subject, bodyHTML, bodyPlain string) model.EmailTemplate {
	return model.EmailTemplate{
		Name:      name,
		Subject:   subject,
		BodyHTML:  bodyHTML,
		BodyPlain: &bodyPlain,
		IsActive:  true,
	}
}
