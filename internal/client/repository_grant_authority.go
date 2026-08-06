package client

import (
	"github.com/maintainerd/maintainerd-auth/internal/shared"
	"gorm.io/gorm"
)

// GrantAuthorityRepository answers the two questions the client-grant
// escalation guard asks: what the acting user actually holds right now, and
// what a role would confer on the client it is attached to.
//
// Like the identity-provider connection repo, it is constructed inside
// NewClientService over the same *gorm.DB rather than injected: these two
// queries exist only for the guard, and injecting them would ripple through
// every NewClientService call site for no added value.
type GrantAuthorityRepository interface {
	WithTx(tx *gorm.DB) GrantAuthorityRepository
	// ActorPermissionNames returns the permission names the acting user can
	// actually exercise in a tenant right now.
	ActorPermissionNames(userID, tenantID int64) ([]string, error)
	// RolePermissionNames returns the permission names a role would confer.
	RolePermissionNames(roleID int64) ([]string, error)
}

type grantAuthorityRepository struct {
	db *gorm.DB
}

func NewGrantAuthorityRepository(db *gorm.DB) GrantAuthorityRepository {
	return &grantAuthorityRepository{db: db}
}

func (r *grantAuthorityRepository) WithTx(tx *gorm.DB) GrantAuthorityRepository {
	if tx == nil {
		return r
	}
	return &grantAuthorityRepository{db: tx}
}

// ActorPermissionNames filters every hop exactly the way the request auth
// context is filtered, so a soft-deleted or deactivated role or permission
// cannot satisfy the escalation guard for an actor who can no longer exercise
// it.
func (r *grantAuthorityRepository) ActorPermissionNames(userID, tenantID int64) ([]string, error) {
	var names []string
	err := r.db.
		Table("user_roles").
		Joins("JOIN roles ON roles.role_id = user_roles.role_id").
		Joins("JOIN role_permissions ON role_permissions.role_id = roles.role_id").
		Joins("JOIN permissions ON permissions.permission_id = role_permissions.permission_id").
		Where("user_roles.user_id = ?", userID).
		Where("roles.tenant_id = ? AND roles.deleted_at IS NULL AND roles.status = ?", tenantID, shared.StatusActive).
		Where("permissions.deleted_at IS NULL AND permissions.status = ?", shared.StatusActive).
		Distinct().
		Pluck("permissions.name", &names).Error
	return names, err
}

// RolePermissionNames deliberately does NOT filter on permissions.status, which
// is the opposite of ActorPermissionNames and is the fail-closed direction for
// each side.
//
// An inactive permission confers nothing today, so it must not count towards
// what the ACTOR holds. But it is still attached to the role, and reactivating
// it is a status flip on a different endpoint — so ignoring it here would let an
// actor smuggle tenant:delete onto a client inside a role while the permission
// is parked inactive, then have someone reactivate it. Only a soft-deleted
// permission is genuinely gone.
func (r *grantAuthorityRepository) RolePermissionNames(roleID int64) ([]string, error) {
	var names []string
	err := r.db.
		Table("role_permissions").
		Joins("JOIN permissions ON permissions.permission_id = role_permissions.permission_id").
		Where("role_permissions.role_id = ?", roleID).
		Where("permissions.deleted_at IS NULL").
		Distinct().
		Pluck("permissions.name", &names).Error
	return names, err
}
