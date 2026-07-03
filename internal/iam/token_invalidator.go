package iam

import (
	"context"

	"github.com/maintainerd/maintainerd-auth/internal/platform/cache"
	"gorm.io/gorm"
)

// AuthorizationTokenInvalidator revokes sessions/tokens and clears cached
// authorization state for users affected by IAM changes.
type AuthorizationTokenInvalidator interface {
	InvalidateRoleChange(ctx context.Context, roleIDs ...int64) error
	InvalidatePermissionChange(ctx context.Context, permissionID int64) error
}

type noopAuthorizationTokenInvalidator struct{}

func (noopAuthorizationTokenInvalidator) InvalidateRoleChange(context.Context, ...int64) error {
	return nil
}

func (noopAuthorizationTokenInvalidator) InvalidatePermissionChange(context.Context, int64) error {
	return nil
}

type dbAuthorizationTokenInvalidator struct {
	db               *gorm.DB
	cacheInvalidator cache.Invalidator
}

func NewDBAuthorizationTokenInvalidator(db *gorm.DB, cacheInvalidator cache.Invalidator) AuthorizationTokenInvalidator {
	if cacheInvalidator == nil {
		cacheInvalidator = cache.NopInvalidator{}
	}
	return &dbAuthorizationTokenInvalidator{
		db:               db,
		cacheInvalidator: cacheInvalidator,
	}
}

func (i *dbAuthorizationTokenInvalidator) InvalidateRoleChange(ctx context.Context, roleIDs ...int64) error {
	if i == nil || i.db == nil || len(roleIDs) == 0 {
		return nil
	}

	var userIDs []int64
	if err := i.db.WithContext(ctx).
		Model(&UserRole{}).
		Where("role_id IN ?", roleIDs).
		Distinct().
		Pluck("user_id", &userIDs).Error; err != nil {
		return err
	}
	return i.invalidateUsers(ctx, userIDs)
}

func (i *dbAuthorizationTokenInvalidator) InvalidatePermissionChange(ctx context.Context, permissionID int64) error {
	if i == nil || i.db == nil {
		return nil
	}

	var roleIDs []int64
	if err := i.db.WithContext(ctx).
		Model(&RolePermission{}).
		Where("permission_id = ?", permissionID).
		Distinct().
		Pluck("role_id", &roleIDs).Error; err != nil {
		return err
	}
	return i.InvalidateRoleChange(ctx, roleIDs...)
}

func (i *dbAuthorizationTokenInvalidator) invalidateUsers(ctx context.Context, userIDs []int64) error {
	if len(userIDs) == 0 {
		return nil
	}

	if err := i.db.WithContext(ctx).
		Model(&UserToken{}).
		Where("user_id IN ?", userIDs).
		Update("is_revoked", true).Error; err != nil {
		return err
	}

	var subs []string
	if err := i.db.WithContext(ctx).
		Model(&UserIdentity{}).
		Where("user_id IN ? AND sub <> ''", userIDs).
		Distinct().
		Pluck("sub", &subs).Error; err != nil {
		return err
	}
	for _, sub := range subs {
		i.cacheInvalidator.InvalidateUserAll(ctx, sub)
	}
	return nil
}
