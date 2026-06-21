package seeder

import (
	"log/slog"

	model "github.com/maintainerd/auth/internal/branding"
	"github.com/maintainerd/auth/internal/platform/templates/emailtemplate"
	"gorm.io/gorm"
)

func SeedEmailTemplates(db *gorm.DB, tenantID int64) error {
	templates := []model.EmailTemplate{
		newEmailTemplate(
			tenantID,
			"user:invite",
			"You're Invited to Join Our Organization!",
			emailtemplate.InviteEmailHTML,
			`You're invited to join our organization. Accept the invite: {{.InviteURL}}`,
			`| Parameter | Description |
|-----------|-------------|
| `+"`{{.InviteURL}}`"+` | The invitation acceptance link the recipient clicks to join |
| `+"`{{.LogoURL}}`"+` | Your organization's logo URL from the email delivery config |`,
		),
		newEmailTemplate(
			tenantID,
			"user:password:reset",
			"Password Reset Request",
			emailtemplate.ForgotPasswordEmailHTML,
			emailtemplate.ForgotPasswordEmailPlain,
			`| Parameter | Description |
|-----------|-------------|
| `+"`{{.ResetURL}}`"+` | Signed URL that directs the user to the password reset form |
| `+"`{{.LogoURL}}`"+` | Your organization's logo URL from the email delivery config |`,
		),
		newEmailTemplate(
			tenantID,
			"user:email:verification",
			"Verify Your Email Address",
			emailtemplate.EmailVerificationEmailHTML,
			emailtemplate.EmailVerificationEmailPlain,
			`| Parameter | Description |
|-----------|-------------|
| `+"`{{.OTP}}`"+` | One-time verification code the user enters to confirm their email |
| `+"`{{.LogoURL}}`"+` | Your organization's logo URL from the email delivery config |`,
		),
		newEmailTemplate(
			tenantID,
			"user:magic_link",
			"Sign in to your account",
			emailtemplate.MagicLinkEmailHTML,
			emailtemplate.MagicLinkEmailPlain,
			`| Parameter | Description |
|-----------|-------------|
| `+"`{{.MagicLinkURL}}`"+` | Signed URL that logs the user in without a password |
| `+"`{{.LogoURL}}`"+` | Your organization's logo URL from the email delivery config |`,
		),
		newEmailTemplate(
			tenantID,
			"user:ciba:notification",
			"Authentication Request",
			emailtemplate.CIBANotificationEmailHTML,
			emailtemplate.CIBANotificationEmailPlain,
			`| Parameter | Description |
|-----------|-------------|
| `+"`{{.ClientName}}`"+` | Display name of the application requesting access |
| `+"`{{.BindingMessage}}`"+` | Optional message from the requesting application (may be empty) |
| `+"`{{.LogoURL}}`"+` | Your organization's logo URL from the email delivery config |`,
		),
		newEmailTemplate(
			tenantID,
			"user:device:approved",
			"Device Authorization Approved",
			emailtemplate.DeviceApprovedEmailHTML,
			emailtemplate.DeviceApprovedEmailPlain,
			`| Parameter | Description |
|-----------|-------------|
| `+"`{{.ClientName}}`"+` | Display name of the application that was authorized |
| `+"`{{.LogoURL}}`"+` | Your organization's logo URL from the email delivery config |`,
		),
		newEmailTemplate(
			tenantID,
			"user:email:change",
			"Your email change verification code",
			emailtemplate.EmailChangeOTPEmailHTML,
			emailtemplate.EmailChangeOTPEmailPlain,
			`| Parameter | Description |
|-----------|-------------|
| `+"`{{.OTP}}`"+` | One-time verification code to confirm the email address change |
| `+"`{{.LogoURL}}`"+` | Your organization's logo URL from the email delivery config |`,
		),
		newEmailTemplate(
			tenantID,
			"user:mfa:enroll",
			"MFA Enrollment Verification",
			emailtemplate.MFAEnrollEmailHTML,
			emailtemplate.MFAEnrollEmailPlain,
			`| Parameter | Description |
|-----------|-------------|
| `+"`{{.OTP}}`"+` | One-time code to verify email during MFA enrollment |
| `+"`{{.LogoURL}}`"+` | Your organization's logo URL from the email delivery config |`,
		),
		newEmailTemplate(
			tenantID,
			"user:mfa:stepup",
			"Verification Code",
			emailtemplate.MFAStepUpEmailHTML,
			emailtemplate.MFAStepUpEmailPlain,
			`| Parameter | Description |
|-----------|-------------|
| `+"`{{.OTP}}`"+` | One-time code for identity verification (step-up / login MFA) |
| `+"`{{.LogoURL}}`"+` | Your organization's logo URL from the email delivery config |`,
		),
	}

	for _, t := range templates {
		var existing model.EmailTemplate
		err := db.Where("name = ? AND tenant_id = ?", t.Name, tenantID).First(&existing).Error
		if err == nil {
			if existing.ParametersDoc == nil && t.ParametersDoc != nil {
				db.Model(&existing).Update("parameters_doc", *t.ParametersDoc)
				slog.Info("Email template parameters_doc updated", "name", t.Name)
			} else {
				slog.Info("Email template already exists, skipping", "name", t.Name)
			}
			continue
		}

		if err := db.Create(&t).Error; err != nil {
			return err
		}

		slog.Info("Email template seeded", "name", t.Name)
	}

	return nil
}

func newEmailTemplate(tenantID int64, name, subject, bodyHTML, bodyPlain, parametersDoc string) model.EmailTemplate {
	return model.EmailTemplate{
		TenantID:      tenantID,
		Name:          name,
		Subject:       subject,
		BodyHTML:      bodyHTML,
		BodyPlain:     &bodyPlain,
		ParametersDoc: &parametersDoc,
		Status:        "active",
	}
}
