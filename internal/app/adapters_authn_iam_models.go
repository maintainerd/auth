package app

import (
	"github.com/maintainerd/auth/internal/authn"
	"github.com/maintainerd/auth/internal/iam"
)

func toAuthnRole(r *iam.Role) *authn.Role {
	if r == nil {
		return nil
	}
	return &authn.Role{
		RoleID: r.RoleID, RoleUUID: r.RoleUUID, TenantID: r.TenantID, Name: r.Name,
		Description: r.Description, Status: r.Status, IsDefault: r.IsDefault, IsSystem: r.IsSystem,
		CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt,
	}
}

func toIamRole(r *authn.Role) *iam.Role {
	if r == nil {
		return nil
	}
	return &iam.Role{
		RoleID: r.RoleID, RoleUUID: r.RoleUUID, TenantID: r.TenantID, Name: r.Name,
		Description: r.Description, Status: r.Status, IsDefault: r.IsDefault, IsSystem: r.IsSystem,
		CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt,
	}
}

func mapAuthnRoles(items []iam.Role) []authn.Role {
	out := make([]authn.Role, len(items))
	for i := range items {
		out[i] = *toAuthnRole(&items[i])
	}
	return out
}
