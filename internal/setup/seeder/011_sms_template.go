package seeder

import (
	"log/slog"

	model "github.com/maintainerd/auth/internal/branding"
	"gorm.io/gorm"
)

func SeedSMSTemplates(db *gorm.DB, tenantID int64) error {
	templates := []model.SMSTemplate{
		newSMSTemplate(tenantID, "sms:login:otp",
			"SMS Login OTP",
			"Your verification code is: {{.OTP}}",
			`| Parameter | Description |
|-----------|-------------|
| `+"`{{.OTP}}`"+` | One-time verification code for SMS login |`,
		),
		newSMSTemplate(tenantID, "sms:mfa:stepup",
			"MFA Step-Up Code",
			"Your step-up code is: {{.OTP}}",
			`| Parameter | Description |
|-----------|-------------|
| `+"`{{.OTP}}`"+` | One-time code for MFA step-up verification |`,
		),
		newSMSTemplate(tenantID, "sms:mfa:enroll",
			"MFA Enrollment Code",
			"Your MFA verification code is: {{.OTP}}",
			`| Parameter | Description |
|-----------|-------------|
| `+"`{{.OTP}}`"+` | One-time code to verify phone during MFA enrollment |`,
		),
	}

	for _, t := range templates {
		var existing model.SMSTemplate
		err := db.Where("name = ? AND tenant_id = ?", t.Name, tenantID).First(&existing).Error
		if err == nil {
			if existing.ParametersDoc == nil && t.ParametersDoc != nil {
				db.Model(&existing).Update("parameters_doc", *t.ParametersDoc)
				slog.Info("SMS template parameters_doc updated", "name", t.Name)
			} else {
				slog.Info("SMS template already exists, skipping", "name", t.Name)
			}
			continue
		}

		if err := db.Create(&t).Error; err != nil {
			return err
		}

		slog.Info("SMS template seeded", "name", t.Name)
	}

	return nil
}

func newSMSTemplate(tenantID int64, name, description, message, parametersDoc string) model.SMSTemplate {
	return model.SMSTemplate{
		TenantID:      tenantID,
		Name:          name,
		Description:   &description,
		Message:       message,
		ParametersDoc: &parametersDoc,
		Status:        "active",
	}
}
