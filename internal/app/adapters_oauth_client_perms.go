package app

import (
	"context"

	"github.com/maintainerd/maintainerd-auth/internal/oauth"
	"github.com/maintainerd/maintainerd-auth/internal/shared"
	"gorm.io/gorm"
)

type clientPermissionResolver struct {
	db *gorm.DB
}

func newClientPermissionResolver(db *gorm.DB) oauth.ClientPermissionResolver {
	return &clientPermissionResolver{db: db}
}

func (r *clientPermissionResolver) ResolvePermissions(ctx context.Context, clientID int64) ([]string, error) {
	var names []string

	// Direct permissions via client_permissions → permissions
	rows, err := r.db.WithContext(ctx).
		Table("client_permissions").
		Select("DISTINCT p.name").
		Joins("JOIN permissions p ON client_permissions.permission_id = p.permission_id").
		Joins("JOIN client_apis ca ON client_permissions.client_api_id = ca.client_api_id").
		// The permission-status filters are REQUIRED: this is a raw Table() join,
		// so GORM's soft-delete scope does not apply. Without them a deleted or
		// deactivated permission kept being written into every machine token —
		// deleting or deactivating a permission did not revoke it. The sibling
		// resolvers in adapters_idp.go and adapters_authn_invite.go already
		// filter; this path did not.
		Where("ca.client_id = ? AND p.deleted_at IS NULL AND p.status = ?", clientID, shared.StatusActive).
		Rows()
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		names = append(names, name)
	}

	// Role-inherited permissions via client_roles → role_permissions → permissions.
	//
	// The roles table has to be joined even though role_id is already on
	// client_roles: without it nothing filters roles.deleted_at / roles.status,
	// so a soft-deleted or deactivated role kept granting every permission it
	// carried in every client_credentials token. client_roles has no deleted_at
	// of its own, so deleting the role IS the only way to revoke the grant —
	// which made this the whole revocation path for machine clients. The user
	// path (iamUserRepo.EffectivePermissionNames) filters exactly these columns;
	// the machine path must match it or the two disagree about what a role means.
	roleRows, err := r.db.WithContext(ctx).
		Table("client_roles").
		Select("DISTINCT p.name").
		Joins("JOIN roles r ON client_roles.role_id = r.role_id").
		Joins("JOIN role_permissions rp ON r.role_id = rp.role_id").
		Joins("JOIN permissions p ON rp.permission_id = p.permission_id").
		Where("client_roles.client_id = ?", clientID).
		Where("r.deleted_at IS NULL AND r.status = ?", shared.StatusActive).
		Where("p.deleted_at IS NULL AND p.status = ?", shared.StatusActive).
		Rows()
	if err != nil {
		return nil, err
	}
	defer func() { _ = roleRows.Close() }()
	seen := make(map[string]struct{}, len(names))
	for _, n := range names {
		seen[n] = struct{}{}
	}
	for roleRows.Next() {
		var name string
		if err := roleRows.Scan(&name); err != nil {
			return nil, err
		}
		if _, ok := seen[name]; !ok {
			seen[name] = struct{}{}
			names = append(names, name)
		}
	}

	return names, nil
}
