package app

import (
	"context"

	"github.com/maintainerd/maintainerd-auth/internal/oauth"
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
		Where("ca.client_id = ?", clientID).
		Rows()
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		names = append(names, name)
	}

	// Role-inherited permissions via client_roles → role_permissions → permissions
	roleRows, err := r.db.WithContext(ctx).
		Table("client_roles").
		Select("DISTINCT p.name").
		Joins("JOIN role_permissions rp ON client_roles.role_id = rp.role_id").
		Joins("JOIN permissions p ON rp.permission_id = p.permission_id").
		Where("client_roles.client_id = ?", clientID).
		Rows()
	if err != nil {
		return nil, err
	}
	defer roleRows.Close()
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
