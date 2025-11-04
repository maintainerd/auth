package service

import (
	"errors"

	"github.com/maintainerd/auth/internal/model"
)

// ValidateTenantAccess validates if a user can access the target tenant
// Rules:
// - Users from default tenant can access any tenant
// - Users from non-default tenant can only access their own tenant
func ValidateTenantAccess(actorUser *model.User, targetTenant *model.Tenant) error {
	if actorUser.Tenant == nil {
		return errors.New("actor user has no tenant")
	}

	// If actor is from default tenant, they can access any tenant
	if actorUser.Tenant.IsDefault {
		return nil
	}

	// If actor is from non-default tenant, they can only access their own tenant
	if actorUser.TenantID == targetTenant.TenantID {
		return nil
	}

	return errors.New("access denied: non-default tenant users can only access their own tenant")
}

// ValidateTenantAccessByID validates tenant access using tenant ID
func ValidateTenantAccessByID(actorUser *model.User, targetTenantID int64) error {
	if actorUser.Tenant == nil {
		return errors.New("actor user has no tenant")
	}

	// If actor is from default tenant, they can access any tenant
	if actorUser.Tenant.IsDefault {
		return nil
	}

	// If actor is from non-default tenant, they can only access their own tenant
	if actorUser.TenantID == targetTenantID {
		return nil
	}

	return errors.New("access denied: non-default tenant users can only access their own tenant")
}
