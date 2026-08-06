package user

import (
	"context"
	"errors"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/maintainerd/maintainerd-auth/internal/platform/cache"
	"github.com/maintainerd/maintainerd-auth/internal/platform/security"
	"github.com/maintainerd/maintainerd-auth/internal/shared"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// adminSetPlaintext is long and unusual enough to clear the default policy
// (min_length 12, min_strength_score 2, common-password blocklist).
const adminSetPlaintext = "lantern-otter-basalt-77"

// adminUserSvc builds a userService with the repos the administrative paths
// touch, plus the sqlmock the raw table writes/reads run against.
func adminUserSvc(
	t *testing.T,
	userRepo *mockUserRepo,
	uiRepo *mockUserIdentityRepo,
	history UserPasswordHistoryRepository,
) (sqlmock.Sqlmock, UserService) {
	t.Helper()
	db, mock := newMockGormDB(t)
	svc := NewUserService(db, userRepo, uiRepo, &mockUserRoleRepo{}, &mockRoleRepo{}, &mockTenantRepo{},
		&mockIdentityProviderRepo{}, &mockClientRepo{}, cache.NopInvalidator{}, nil, nil, history, nil, nil)
	return mock, svc
}

// adminTargetAndActor wires FindByUUID so the first call resolves the target
// user and every later call resolves the acting admin, both in tenant 1.
func adminTargetAndActor(repo *mockUserRepo, target *User) {
	calls := 0
	repo.findByUUIDFn = func(any, ...string) (*User, error) {
		calls++
		if calls == 1 {
			return target, nil
		}
		return userWithAccess(99, 1), nil
	}
}

func adminTargetUser() *User {
	return &User{
		UserID:         1,
		UserUUID:       uuid.New(),
		TenantID:       1,
		Username:       "target",
		Email:          "target@example.com",
		Status:         shared.StatusActive,
		UserIdentities: []UserIdentity{{TenantID: 1, Tenant: &Tenant{TenantID: 1, IsSystem: true}}},
	}
}

// ---------------------------------------------------------------------------
// SetPassword — the administrative password set
// ---------------------------------------------------------------------------

func TestUserService_SetPassword(t *testing.T) {
	targetUUID := uuid.New()
	actorUUID := uuid.New()

	t.Run("user not found", func(t *testing.T) {
		ur := &mockUserRepo{findByUUIDFn: func(any, ...string) (*User, error) { return nil, nil }}
		_, svc := adminUserSvc(t, ur, &mockUserIdentityRepo{}, &stubPasswordHistoryRepo{})
		err := svc.SetPassword(context.Background(), targetUUID, 1, adminSetPlaintext, false, actorUUID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "user not found")
	})

	t.Run("cross-tenant target is refused", func(t *testing.T) {
		other := adminTargetUser()
		other.UserIdentities = []UserIdentity{{TenantID: 99, Tenant: &Tenant{TenantID: 99}}}
		ur := &mockUserRepo{}
		adminTargetAndActor(ur, other)
		_, svc := adminUserSvc(t, ur, &mockUserIdentityRepo{}, &stubPasswordHistoryRepo{})
		err := svc.SetPassword(context.Background(), targetUUID, 1, adminSetPlaintext, false, actorUUID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "access denied")
	})

	t.Run("password below tenant policy is rejected", func(t *testing.T) {
		ur := &mockUserRepo{}
		adminTargetAndActor(ur, adminTargetUser())
		_, svc := adminUserSvc(t, ur, &mockUserIdentityRepo{}, &stubPasswordHistoryRepo{})
		err := svc.SetPassword(context.Background(), targetUUID, 1, "short", false, actorUUID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "at least")
	})

	t.Run("a password that restates the account's own email is rejected", func(t *testing.T) {
		ur := &mockUserRepo{}
		adminTargetAndActor(ur, adminTargetUser())
		_, svc := adminUserSvc(t, ur, &mockUserIdentityRepo{}, &stubPasswordHistoryRepo{})
		err := svc.SetPassword(context.Background(), targetUUID, 1, "target@example.com-x", false, actorUUID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "must not contain")
	})

	// Fail CLOSED: the shipped default policy keeps 5 passwords of history, so a
	// missing history repo is a wiring fault, never licence to skip the check.
	t.Run("policy requires history but none is configured", func(t *testing.T) {
		ur := &mockUserRepo{}
		adminTargetAndActor(ur, adminTargetUser())
		_, svc := adminUserSvc(t, ur, &mockUserIdentityRepo{}, nil)
		err := svc.SetPassword(context.Background(), targetUUID, 1, adminSetPlaintext, false, actorUUID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "password history is required by policy")
	})

	t.Run("an unreadable history is not an empty history", func(t *testing.T) {
		ur := &mockUserRepo{}
		adminTargetAndActor(ur, adminTargetUser())
		_, svc := adminUserSvc(t, ur, &mockUserIdentityRepo{}, &stubPasswordHistoryRepo{findErr: errors.New("db down")})
		err := svc.SetPassword(context.Background(), targetUUID, 1, adminSetPlaintext, false, actorUUID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to read password history")
	})

	t.Run("a recently used password is refused", func(t *testing.T) {
		prior, hErr := security.HashPasswordWithPolicy(context.Background(), []byte(adminSetPlaintext),
			security.PasswordPolicy{HashAlgorithm: "bcrypt"})
		require.NoError(t, hErr)

		ur := &mockUserRepo{}
		adminTargetAndActor(ur, adminTargetUser())
		_, svc := adminUserSvc(t, ur, &mockUserIdentityRepo{}, &stubPasswordHistoryRepo{recent: []string{string(prior)}})
		err := svc.SetPassword(context.Background(), targetUUID, 1, adminSetPlaintext, false, actorUUID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "used recently")
	})

	t.Run("permanent set writes the password and evicts every credential", func(t *testing.T) {
		ur := &mockUserRepo{}
		adminTargetAndActor(ur, adminTargetUser())
		var written map[string]any
		ur.updateByIDFn = func(_ any, data any) (*User, error) {
			written, _ = data.(map[string]any)
			return &User{}, nil
		}
		history := &stubPasswordHistoryRepo{}
		mock, svc := adminUserSvc(t, ur, &mockUserIdentityRepo{}, history)
		mock.ExpectBegin()
		// An admin set is the remedy for a suspected compromise, so unlike
		// self-service rotation it spares nothing.
		expectCredentialRevocation(mock)
		mock.ExpectCommit()

		require.NoError(t, svc.SetPassword(context.Background(), targetUUID, 1, adminSetPlaintext, false, actorUUID))
		require.NoError(t, mock.ExpectationsWereMet())

		require.NotNil(t, written)
		assert.NotEmpty(t, written["password"])
		assert.Equal(t, false, written["force_password_change"])
		assert.Nil(t, written["temporary_password_expires_at"],
			"a permanent password must stop being subject to temp-password expiry")
		assert.Len(t, history.added, 1, "the new hash must enter history or reuse detection has a hole")
	})

	t.Run("temporary set forces a change and starts the expiry clock", func(t *testing.T) {
		ur := &mockUserRepo{}
		adminTargetAndActor(ur, adminTargetUser())
		var written map[string]any
		ur.updateByIDFn = func(_ any, data any) (*User, error) {
			written, _ = data.(map[string]any)
			return &User{}, nil
		}
		mock, svc := adminUserSvc(t, ur, &mockUserIdentityRepo{}, &stubPasswordHistoryRepo{})
		mock.ExpectBegin()
		expectCredentialRevocation(mock)
		mock.ExpectCommit()

		require.NoError(t, svc.SetPassword(context.Background(), targetUUID, 1, adminSetPlaintext, true, actorUUID))
		assert.Equal(t, true, written["force_password_change"])
		assert.NotNil(t, written["temporary_password_expires_at"])
	})

	t.Run("a failed revoke rolls the password back", func(t *testing.T) {
		ur := &mockUserRepo{}
		adminTargetAndActor(ur, adminTargetUser())
		ur.updateByIDFn = func(any, any) (*User, error) { return &User{}, nil }
		mock, svc := adminUserSvc(t, ur, &mockUserIdentityRepo{}, &stubPasswordHistoryRepo{})
		mock.ExpectBegin()
		mock.ExpectExec(`UPDATE "user_sessions"`).WillReturnError(errors.New("revoke boom"))
		mock.ExpectRollback()

		// Reporting a reset that left the attacker's session spendable is worse
		// than reporting a failure, so the write does not survive.
		err := svc.SetPassword(context.Background(), targetUUID, 1, adminSetPlaintext, false, actorUUID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to revoke sessions")
	})
}

// ---------------------------------------------------------------------------
// AdminLinkIdentity — the operator's identity link
// ---------------------------------------------------------------------------

// expectProviderLookup answers the tenant-scoped, non-deleted provider query.
func expectProviderLookup(mock sqlmock.Sqlmock, idpID int64, idpUUID uuid.UUID, status string) {
	mock.ExpectQuery(`SELECT \* FROM "identity_providers"`).
		WillReturnRows(sqlmock.NewRows([]string{
			"identity_provider_id", "identity_provider_uuid", "tenant_id", "name", "provider", "status",
		}).AddRow(idpID, idpUUID, int64(1), "Google", "google", status))
}

func TestUserService_AdminLinkIdentity(t *testing.T) {
	targetUUID := uuid.New()
	actorUUID := uuid.New()
	idpUUID := uuid.New()

	t.Run("blank sub is rejected", func(t *testing.T) {
		_, svc := adminUserSvc(t, &mockUserRepo{}, &mockUserIdentityRepo{}, nil)
		_, err := svc.AdminLinkIdentity(context.Background(), targetUUID, 1, idpUUID, "   ", actorUUID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "sub is required")
	})

	t.Run("user not found", func(t *testing.T) {
		ur := &mockUserRepo{findByUUIDFn: func(any, ...string) (*User, error) { return nil, nil }}
		_, svc := adminUserSvc(t, ur, &mockUserIdentityRepo{}, nil)
		_, err := svc.AdminLinkIdentity(context.Background(), targetUUID, 1, idpUUID, "sub-1", actorUUID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "user not found")
	})

	// A provider belonging to another tenant reads as missing, not forbidden, so
	// the endpoint cannot be used to enumerate other tenants' providers.
	t.Run("unknown or cross-tenant provider reads as not found", func(t *testing.T) {
		ur := &mockUserRepo{}
		adminTargetAndActor(ur, adminTargetUser())
		mock, svc := adminUserSvc(t, ur, &mockUserIdentityRepo{}, nil)
		mock.ExpectQuery(`SELECT \* FROM "identity_providers"`).WillReturnError(gorm.ErrRecordNotFound)
		_, err := svc.AdminLinkIdentity(context.Background(), targetUUID, 1, idpUUID, "sub-1", actorUUID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "identity provider not found")
	})

	t.Run("a deactivated provider cannot be linked", func(t *testing.T) {
		ur := &mockUserRepo{}
		adminTargetAndActor(ur, adminTargetUser())
		mock, svc := adminUserSvc(t, ur, &mockUserIdentityRepo{}, nil)
		expectProviderLookup(mock, 5, idpUUID, shared.StatusInactive)
		_, err := svc.AdminLinkIdentity(context.Background(), targetUUID, 1, idpUUID, "sub-1", actorUUID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not active")
	})

	// The uniqueness constraint is (tenant_id, sub), so a sub already linked
	// anywhere in the tenant is a conflict — even under a different provider slug.
	t.Run("a sub already linked to another user is a conflict, never a move", func(t *testing.T) {
		ur := &mockUserRepo{}
		adminTargetAndActor(ur, adminTargetUser())
		mock, svc := adminUserSvc(t, ur, &mockUserIdentityRepo{}, nil)
		expectProviderLookup(mock, 5, idpUUID, shared.StatusActive)
		mock.ExpectQuery(`SELECT \* FROM "user_identities"`).
			WillReturnRows(sqlmock.NewRows([]string{"user_identity_id", "user_id", "tenant_id", "sub"}).
				AddRow(int64(7), int64(4242), int64(1), "sub-1"))
		_, err := svc.AdminLinkIdentity(context.Background(), targetUUID, 1, idpUUID, "sub-1", actorUUID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "already linked to another user")
	})

	t.Run("an unreadable existing-link check fails closed", func(t *testing.T) {
		ur := &mockUserRepo{}
		adminTargetAndActor(ur, adminTargetUser())
		mock, svc := adminUserSvc(t, ur, &mockUserIdentityRepo{}, nil)
		expectProviderLookup(mock, 5, idpUUID, shared.StatusActive)
		mock.ExpectQuery(`SELECT \* FROM "user_identities"`).WillReturnError(errors.New("db down"))
		_, err := svc.AdminLinkIdentity(context.Background(), targetUUID, 1, idpUUID, "sub-1", actorUUID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to check existing identity links")
	})

	t.Run("success links the identity to the target user", func(t *testing.T) {
		ur := &mockUserRepo{}
		target := adminTargetUser()
		adminTargetAndActor(ur, target)
		var created *UserIdentity
		ui := &mockUserIdentityRepo{createFn: func(e *UserIdentity) (*UserIdentity, error) {
			created = e
			e.UserIdentityUUID = uuid.New()
			return e, nil
		}}
		mock, svc := adminUserSvc(t, ur, ui, nil)
		expectProviderLookup(mock, 5, idpUUID, shared.StatusActive)
		mock.ExpectQuery(`SELECT \* FROM "user_identities"`).WillReturnError(gorm.ErrRecordNotFound)

		res, err := svc.AdminLinkIdentity(context.Background(), targetUUID, 1, idpUUID, " sub-1 ", actorUUID)
		require.NoError(t, err)
		require.NotNil(t, res)
		require.NotNil(t, created)
		assert.Equal(t, target.UserID, created.UserID)
		assert.Equal(t, int64(5), created.IdentityProviderID)
		assert.Equal(t, "google", created.Provider)
		assert.Equal(t, "sub-1", created.Sub, "the sub must be trimmed before it reaches the unique index")
		assert.Equal(t, "google", res.Provider)
	})
}
