package seeder

import (
	"log/slog"

	"github.com/google/uuid"
	model "github.com/maintainerd/maintainerd-auth/internal/idp"
	"github.com/maintainerd/maintainerd-auth/internal/platform/crypto"
	"github.com/maintainerd/maintainerd-auth/internal/shared"
	"gorm.io/gorm"
)

const (
	// SystemRegistrationFlowOwner is the name of the system registration flow that grants the
	// registered + super-admin roles. Used when inviting a tenant owner.
	SystemRegistrationFlowOwner = "system:onboarding:owner"
)

// SeedRegistrationFlows seeds the one genuinely special system registration flow:
// registered + super-admin (owner invites). Normal registration is flow-free.
// All are marked IsSystem=true and cannot be deleted through normal API paths.
func SeedRegistrationFlows(db *gorm.DB, tenantID int64) error {
	var consoleClient model.Client
	err := db.Where("name = ? AND tenant_id = ? AND is_system = ?",
		shared.SystemClientNameAuthConsole, tenantID, true).First(&consoleClient).Error
	if err != nil {
		slog.Error("Console system client not found for registration flow seeding", "tenant_id", tenantID, "error", err)
		return err
	}

	if err := seedRegistrationFlowWithRoleNames(db, tenantID, SystemRegistrationFlowOwner,
		"Owner invitation — grants registered and super-admin roles",
		&consoleClient, "registered", "super-admin"); err != nil {
		return err
	}

	slog.Info("System registration flows seeded", "tenant_id", tenantID)
	return nil
}

func seedRegistrationFlowWithRoleNames(db *gorm.DB, tenantID int64, name, description string,
	client *model.Client, roleNames ...string) error {

	var existing model.RegistrationFlow
	err := db.Where("name = ? AND tenant_id = ?", name, tenantID).First(&existing).Error
	if err == nil {
		slog.Info("Registration flow already exists, skipping", "name", name, "id", existing.RegistrationFlowID)
		return nil
	}
	if err != gorm.ErrRecordNotFound {
		return err
	}

	identifier, err := crypto.GenerateIdentifier(16)
	if err != nil {
		return err
	}

	flow := &model.RegistrationFlow{
		RegistrationFlowUUID: uuid.New(),
		TenantID:             tenantID,
		Name:                 name,
		Description:          description,
		Identifier:           identifier,
		IsSystem:             true,
		Status:               shared.StatusActive,
		ClientID:             client.ClientID,
	}
	if err := db.Create(flow).Error; err != nil {
		return err
	}

	for _, roleName := range roleNames {
		var roleID int64
		if err := db.Table("roles").
			Select("role_id").
			Where("name = ? AND tenant_id = ?", roleName, tenantID).
			Scan(&roleID).Error; err != nil || roleID == 0 {
			slog.Error("Role not found for registration flow", "role", roleName, "registration_flow", name, "error", err)
			return err
		}
		afr := &model.RegistrationFlowRole{
			RegistrationFlowRoleUUID: uuid.New(),
			RegistrationFlowID:       flow.RegistrationFlowID,
			RoleID:                   roleID,
		}
		if err := db.Create(afr).Error; err != nil {
			return err
		}
	}

	slog.Info("Registration flow seeded", "name", name, "id", flow.RegistrationFlowID)
	return nil
}
