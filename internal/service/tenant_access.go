package service

import (
	"errors"

	"github.com/maintainerd/auth/internal/model"
)

// ValidateTenantAccess validates if a user can access the target tenant
// Rules:
// - Users from default tenant can access any tenant
// - Users from non-default tenant can only access their own tenant
// - User must have at least one identity to validate access
func ValidateTenantAccess(actorUser *model.User, targetTenant *model.Tenant) error {
	// User must have at least one identity
	if len(actorUser.UserIdentities) == 0 {
		return errors.New("actor user has no identities")
	}

	// Check if user has access to the target tenant through any of their identities
	hasAccessToTargetTenant := false
	hasDefaultTenantAccess := false

	for _, identity := range actorUser.UserIdentities {
		// If user has identity in a default tenant, they can access any tenant
		if identity.Tenant.IsDefault {
			hasDefaultTenantAccess = true
			break
		}

		// Check if user has identity in the target tenant
		if identity.TenantID == targetTenant.TenantID {
			hasAccessToTargetTenant = true
		}
	}

	// If user has default tenant access, allow
	if hasDefaultTenantAccess {
		return nil
	}

	// If user has access to target tenant, allow
	if hasAccessToTargetTenant {
		return nil
	}

	return errors.New("access denied: user does not have access to this tenant")
}

// ValidateTenantAccessByID validates tenant access using tenant ID
// Rules:
// - Users from default tenant can access any tenant
// - Users from non-default tenant can only access their own tenant
// - User must have at least one identity to validate access
func ValidateTenantAccessByID(actorUser *model.User, targetTenantID int64) error {
	// User must have at least one identity
	if len(actorUser.UserIdentities) == 0 {
		return errors.New("actor user has no identities")
	}

	// Check if user has access to the target tenant through any of their identities
	hasAccessToTargetTenant := false
	hasDefaultTenantAccess := false

	for _, identity := range actorUser.UserIdentities {
		// If user has identity in a default tenant, they can access any tenant
		if identity.Tenant.IsDefault {
			hasDefaultTenantAccess = true
			break
		}

		// Check if user has identity in the target tenant
		if identity.TenantID == targetTenantID {
			hasAccessToTargetTenant = true
		}
	}

	// If user has default tenant access, allow
	if hasDefaultTenantAccess {
		return nil
	}

	// If user has access to target tenant, allow
	if hasAccessToTargetTenant {
		return nil
	}

	return errors.New("access denied: user does not have access to this tenant")
}
