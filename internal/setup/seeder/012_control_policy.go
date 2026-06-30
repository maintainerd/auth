package seeder

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	model "github.com/maintainerd/auth/internal/iam"
	"github.com/maintainerd/auth/internal/platform/ptr"
	"github.com/maintainerd/auth/internal/shared"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

const SystemControlPolicyName = "auth-control"

func SeedControlPolicy(db *gorm.DB, tenantID int64) error {
	var existing model.Policy
	err := db.Where("name = ? AND tenant_id = ? AND version = ?", SystemControlPolicyName, tenantID, "v1").
		First(&existing).Error
	if err == nil {
		slog.Info("Control policy already exists, skipping", "name", SystemControlPolicyName)
		return nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return fmt.Errorf("failed to check control policy: %w", err)
	}

	document := controlPolicyDocument()

	raw, _ := json.Marshal(document)

	policy := model.Policy{
		PolicyUUID:  uuid.New(),
		TenantID:    tenantID,
		Name:        SystemControlPolicyName,
		Description: ptr.Ptr("Default service-to-service control policy for auth management APIs."),
		Document:    datatypes.JSON(raw),
		Version:     "v1",
		Status:      shared.StatusActive,
		IsSystem:    true,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	if err := db.Create(&policy).Error; err != nil {
		return fmt.Errorf("failed to seed control policy: %w", err)
	}
	slog.Info("Control policy seeded", "name", policy.Name, "id", policy.PolicyID)
	return nil
}

func controlPolicyDocument() model.PolicyDocument {
	return model.PolicyDocument{
		Version: "v1",
		Statement: []model.PolicyStatement{
			{
				Effect: "allow",
				Action: []string{
					"tenant:*",
					"service:*",
					"api:*",
					"permission:*",
					"policy:*",
					"role:*",
					"idp:*",
					"client:*",
					"api_key:*",
					"user:*",
					"auth_event:*",
					"account:*:self",
					"profile:*",
					"registration-flow:*",
					"security-setting:*",
					"ip-restriction-rule:*",
					"email-template:*",
					"sms-template:*",
					"branding:*",
					"tenant-setting:*",
					"email-config:*",
					"sms-config:*",
					"webhook-endpoint:*",
					"security:*",
					"settings:*",
					"notification:*",
					"system:*",
					"audit:*",
					"root:*",
					"webhook:*",
				},
				Resource: []string{"*"},
			},
		},
	}
}
