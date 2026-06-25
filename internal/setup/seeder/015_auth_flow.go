package seeder

import (
	"log/slog"

	"github.com/google/uuid"
	model "github.com/maintainerd/auth/internal/idp"
	"github.com/maintainerd/auth/internal/platform/crypto"
	"github.com/maintainerd/auth/internal/shared"
	"gorm.io/gorm"
)

const (
	// SystemAuthFlowRegistered is the name of the system auth flow that grants
	// only the registered role. It defaults to the console frontend for admin
	// invites. Used as the default when auth_flow UUID is not provided.
	SystemAuthFlowRegistered = "system:onboarding:registered"

	// SystemAuthFlowIdentity is the name of the system auth flow that grants
	// only the registered role and targets the identity frontend for public
	// onboarding flows.
	SystemAuthFlowIdentity = "system:onboarding:identity"

	// SystemAuthFlowOwner is the name of the system auth flow that grants the
	// registered + super-admin roles. It targets the console frontend and is
	// used when inviting a tenant owner.
	SystemAuthFlowOwner = "system:onboarding:owner"
)

// SeedAuthFlows seeds the three system auth flows for a tenant:
//   - registered role only, console destination (default for admin invites)
//   - registered role only, identity destination (public onboarding)
//   - registered + super-admin roles, console destination (owner invites)
//
// All are marked IsSystem=true and cannot be deleted through normal API paths.
func SeedAuthFlows(db *gorm.DB, tenantID int64) error {
	var consoleClient model.Client
	err := db.Where("name = ? AND tenant_id = ? AND is_system = ?",
		shared.SystemClientNameAuthConsole, tenantID, true).First(&consoleClient).Error
	if err != nil {
		slog.Error("Console system client not found for auth flow seeding", "tenant_id", tenantID, "error", err)
		return err
	}

	var identityClient model.Client
	err = db.Where("name = ? AND tenant_id = ? AND is_system = ?",
		shared.SystemClientNameAuthIdentity, tenantID, true).First(&identityClient).Error
	if err != nil {
		slog.Error("Identity system client not found for auth flow seeding", "tenant_id", tenantID, "error", err)
		return err
	}

	if err := seedAuthFlowWithRoleNames(db, tenantID, SystemAuthFlowRegistered,
		"Default tenant onboarding — grants the registered role (console)",
		shared.DestinationConsole, &consoleClient, "registered"); err != nil {
		return err
	}

	if err := seedAuthFlowWithRoleNames(db, tenantID, SystemAuthFlowIdentity,
		"Public onboarding — grants the registered role (identity)",
		shared.DestinationIdentity, &identityClient, "registered"); err != nil {
		return err
	}

	if err := seedAuthFlowWithRoleNames(db, tenantID, SystemAuthFlowOwner,
		"Owner invitation — grants registered and super-admin roles (console)",
		shared.DestinationConsole, &consoleClient, "registered", "super-admin"); err != nil {
		return err
	}

	slog.Info("System auth flows seeded", "tenant_id", tenantID)
	return nil
}

func seedAuthFlowWithRoleNames(db *gorm.DB, tenantID int64, name, description, destination string,
	client *model.Client, roleNames ...string) error {

	var existing model.AuthFlow
	err := db.Where("name = ? AND tenant_id = ?", name, tenantID).First(&existing).Error
	if err == nil {
		slog.Info("Auth flow already exists, skipping", "name", name, "id", existing.AuthFlowID)
		return nil
	}
	if err != gorm.ErrRecordNotFound {
		return err
	}

	identifier, err := crypto.GenerateIdentifier(16)
	if err != nil {
		return err
	}

	af := &model.AuthFlow{
		AuthFlowUUID: uuid.New(),
		TenantID:     tenantID,
		Name:         name,
		Description:  description,
		Identifier:   identifier,
		Destination:  destination,
		IsSystem:     true,
		Status:       shared.StatusActive,
		ClientID:     &client.ClientID,
	}
	if err := db.Create(af).Error; err != nil {
		return err
	}

	for _, roleName := range roleNames {
		var roleID int64
		if err := db.Table("roles").
			Select("role_id").
			Where("name = ? AND tenant_id = ?", roleName, tenantID).
			Scan(&roleID).Error; err != nil || roleID == 0 {
			slog.Error("Role not found for auth flow", "role", roleName, "auth_flow", name, "error", err)
			return err
		}
		afr := &model.AuthFlowRole{
			AuthFlowRoleUUID: uuid.New(),
			AuthFlowID:       af.AuthFlowID,
			RoleID:           roleID,
		}
		if err := db.Create(afr).Error; err != nil {
			return err
		}
	}

	slog.Info("Auth flow seeded", "name", name, "id", af.AuthFlowID)
	return nil
}
