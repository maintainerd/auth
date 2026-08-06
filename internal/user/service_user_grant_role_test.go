package user

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/maintainerd/maintainerd-auth/internal/authevent"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// recordingInvalidator captures the subs whose cached user context was dropped.
type recordingInvalidator struct{ subs []string }

func (r *recordingInvalidator) InvalidateUser(context.Context, string, string) {}
func (r *recordingInvalidator) InvalidateUserAll(_ context.Context, sub string) {
	r.subs = append(r.subs, sub)
}
func (r *recordingInvalidator) InvalidateAllUsers(context.Context) {}

// recordingTokenRepo captures the users whose sessions were revoked.
type recordingTokenRepo struct {
	mockUserTokenRepo
	revoked []int64
}

func (r *recordingTokenRepo) RevokeAllSessionsByUserID(userID int64) error {
	r.revoked = append(r.revoked, userID)
	return nil
}

// GrantRoleByName is reached from tenant ownership transfer, where it grants
// super-admin. It used to skip the cache invalidation and session revocation
// that AssignUserRoles and RemoveUserRole both perform, so the new role sat
// behind the cached user context and existing access tokens — the transfer took
// up to the cache TTL (~10 minutes) to actually apply.
func TestUserService_GrantRoleByName_PropagatesTheGrant(t *testing.T) {
	const tenantID = int64(1)
	const targetUserID = int64(7)
	sourceUUID := uuid.New()
	targetUUID := uuid.New()

	newSvc := func(t *testing.T, roleAlreadyHeld bool) (*userService, *recordingInvalidator, *recordingTokenRepo) {
		t.Helper()
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectCommit()

		inv := &recordingInvalidator{}
		tokens := &recordingTokenRepo{}

		var existing *UserRole
		if roleAlreadyHeld {
			existing = &UserRole{UserID: targetUserID, RoleID: 3}
		}

		return &userService{
			db: db,
			userRepo: &mockUserRepo{
				findByUUIDFn: func(any, ...string) (*User, error) {
					return &User{
						UserID: targetUserID, UserUUID: targetUUID, Email: "owner@example.com",
						UserIdentities: []UserIdentity{{Sub: "sub-1"}, {Sub: "sub-1"}, {Sub: "sub-2"}},
					}, nil
				},
				findByEmailFn: func(string) (*User, error) {
					return &User{UserID: targetUserID, UserUUID: targetUUID, Email: "owner@example.com"}, nil
				},
			},
			roleRepo: &mockRoleRepo{
				findByNameAndTenantIDFn: func(string, int64) (*Role, error) {
					return &Role{RoleID: 3, TenantID: tenantID, Name: "super-admin"}, nil
				},
			},
			userRoleRepo: &mockUserRoleRepo{
				findByUserIDAndRoleIDFn: func(int64, int64) (*UserRole, error) { return existing, nil },
			},
			cacheInvalidator: inv,
			userTokenRepo:    tokens,
			authEventService: authevent.NoopService(),
		}, inv, tokens
	}

	t.Run("a new grant invalidates the cache and revokes sessions", func(t *testing.T) {
		svc, inv, tokens := newSvc(t, false)

		require.NoError(t, svc.GrantRoleByName(context.Background(), sourceUUID, tenantID, "super-admin"))

		// Deduplicated per sub, exactly as invalidateUserCache does elsewhere.
		assert.Equal(t, []string{"sub-1", "sub-2"}, inv.subs)
		assert.Equal(t, []int64{targetUserID}, tokens.revoked)
	})

	t.Run("an already-held role stays a no-op", func(t *testing.T) {
		// The early return must not sign the user out for nothing.
		svc, inv, tokens := newSvc(t, true)

		require.NoError(t, svc.GrantRoleByName(context.Background(), sourceUUID, tenantID, "super-admin"))

		assert.Empty(t, inv.subs)
		assert.Empty(t, tokens.revoked)
	})
}
