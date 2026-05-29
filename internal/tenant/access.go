package tenant

import (
	"github.com/maintainerd/auth/internal/platform/apperror"
)

// ValidateTenantAccess validates if an actor can access the target tenant.
// Rules:
//   - Actors with an identity in a system/default tenant can access any tenant
//   - Otherwise the actor may only access tenants they have an identity in
//   - The actor must have at least one identity
//
// The actor is supplied as a consumer-defined interface so this package does
// not depend on the user domain (see deps.go).
func ValidateTenantAccess(actor AccessActor, targetTenant *Tenant) error {
	if actor == nil {
		return apperror.NewValidation("actor user is nil")
	}
	if targetTenant == nil {
		return apperror.NewValidation("target tenant is nil")
	}
	return validateTenantAccessByID(actor, targetTenant.TenantID)
}

// ValidateTenantAccessByID validates tenant access using a tenant ID.
func ValidateTenantAccessByID(actor AccessActor, targetTenantID int64) error {
	if actor == nil {
		return apperror.NewValidation("actor user is nil")
	}
	return validateTenantAccessByID(actor, targetTenantID)
}

func validateTenantAccessByID(actor AccessActor, targetTenantID int64) error {
	identities := actor.AccessIdentities()

	// Actor must have at least one identity.
	if len(identities) == 0 {
		return apperror.NewValidation("actor user has no identities")
	}

	hasAccessToTargetTenant := false
	for _, identity := range identities {
		// An identity in a system/default tenant grants access to any tenant.
		if identity.TenantIsSystem {
			return nil
		}
		if identity.TenantID == targetTenantID {
			hasAccessToTargetTenant = true
		}
	}

	if hasAccessToTargetTenant {
		return nil
	}

	return apperror.NewForbidden("access denied: user does not have access to this tenant")
}
