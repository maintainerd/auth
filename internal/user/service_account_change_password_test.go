package user

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/maintainerd/maintainerd-auth/internal/authevent"
	"github.com/maintainerd/maintainerd-auth/internal/platform/security"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// ---------------------------------------------------------------------------
// Fixtures
// ---------------------------------------------------------------------------

const (
	currentPlaintext = "vault-crimson-ledger-92"
	nextPlaintext    = "harbor-tangent-willow-41"
)

type stubPasswordHistoryRepo struct {
	recent  []string
	addErr  error
	findErr error
	added   []string
	pruned  bool
}

func (s *stubPasswordHistoryRepo) WithTx(*gorm.DB) UserPasswordHistoryRepository { return s }
func (s *stubPasswordHistoryRepo) AddEntry(_ int64, hash string) error {
	if s.addErr != nil {
		return s.addErr
	}
	s.added = append(s.added, hash)
	return nil
}
func (s *stubPasswordHistoryRepo) FindRecentHashes(_ int64, _ int) ([]string, error) {
	return s.recent, s.findErr
}
func (s *stubPasswordHistoryRepo) PruneExcess(int64, int) error { s.pruned = true; return nil }

type stubSessionRevoker struct {
	revokeAllCalled    bool
	revokedExceptUUID  *uuid.UUID
	revokeAllErr       error
	revokeExceptErr    error
	revokeExceptCalled bool
}

func (s *stubSessionRevoker) RevokeAllByUserID(int64, string) error {
	s.revokeAllCalled = true
	return s.revokeAllErr
}
func (s *stubSessionRevoker) RevokeAllExceptUUID(_ int64, keep uuid.UUID, _ string) error {
	s.revokeExceptCalled = true
	s.revokedExceptUUID = &keep
	return s.revokeExceptErr
}

// newChangePasswordSvc builds an accountService whose stored password is
// currentPlaintext, with a sqlmock DB that expects one committed transaction.
func newChangePasswordSvc(
	t *testing.T,
	history UserPasswordHistoryRepository,
	sessions SessionRevoker,
	expectCommit bool,
) (*accountService, *mockUserRepo, map[string]any) {
	t.Helper()

	hashed, err := security.HashPasswordWithPolicy(context.Background(), []byte(currentPlaintext),
		security.PasswordPolicy{HashAlgorithm: "bcrypt"})
	require.NoError(t, err)
	stored := string(hashed)

	db, mock := newMockGormDB(t)
	mock.ExpectBegin()
	if expectCommit {
		mock.ExpectCommit()
	} else {
		mock.ExpectRollback()
	}

	captured := map[string]any{}
	userRepo := &mockUserRepo{
		findByIDFn: func(_ any, _ ...string) (*User, error) {
			return &User{
				UserID:   7,
				UserUUID: uuid.New(),
				TenantID: 1,
				Username: "jbarnes",
				Email:    "jbarnes@example.test",
				Password: &stored,
			}, nil
		},
		updateByIDFn: func(_ any, data any) (*User, error) {
			if m, ok := data.(map[string]any); ok {
				for k, v := range m {
					captured[k] = v
				}
			}
			return &User{UserID: 7}, nil
		},
	}

	svc := &accountService{
		db:                  db,
		userRepo:            userRepo,
		authEventService:    authevent.NoopService(),
		passwordHistoryRepo: history,
		sessionRepo:         sessions,
	}
	return svc, userRepo, captured
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

func TestAccountService_ChangePassword_Succeeds(t *testing.T) {
	history := &stubPasswordHistoryRepo{}
	sessions := &stubSessionRevoker{}
	caller := uuid.New()
	svc, _, captured := newChangePasswordSvc(t, history, sessions, true)

	result, err := svc.ChangePassword(context.Background(), 7, currentPlaintext, nextPlaintext, &caller)
	require.NoError(t, err)
	require.NotNil(t, result)

	// Mirrors the reset path exactly: the temp-password fields must be cleared
	// too, or a user who rotates away from a temporary password stays subject to
	// temp-password expiry.
	assert.Equal(t, false, captured["force_password_change"])
	assert.Nil(t, captured["temporary_password_expires_at"])
	assert.NotNil(t, captured["password_changed_at"])
	assert.NotEqual(t, currentPlaintext, captured["password"], "the stored value must be a hash")
	assert.Len(t, history.added, 1)
}

// ASVS V3 requires OTHER sessions to be invalidated; nothing requires logging
// the user out of the session they are using to rotate. Doing that punishes the
// user for rotating, which is how you stop people rotating.
func TestAccountService_ChangePassword_PreservesTheCallersOwnSession(t *testing.T) {
	sessions := &stubSessionRevoker{}
	caller := uuid.New()
	svc, _, _ := newChangePasswordSvc(t, &stubPasswordHistoryRepo{}, sessions, true)

	result, err := svc.ChangePassword(context.Background(), 7, currentPlaintext, nextPlaintext, &caller)
	require.NoError(t, err)

	assert.True(t, sessions.revokeExceptCalled, "other sessions must be revoked")
	assert.False(t, sessions.revokeAllCalled, "the caller's own session must not be revoked")
	require.NotNil(t, sessions.revokedExceptUUID)
	assert.Equal(t, caller, *sessions.revokedExceptUUID)
	assert.True(t, result.OtherSessionsRevoked)
	assert.False(t, result.ReauthenticationRequired)
}

// No identifiable caller session means we cannot preserve one. Revoke
// everything and say so — never silently skip revocation.
func TestAccountService_ChangePassword_RevokesEverythingWhenTheCallerIsUnknown(t *testing.T) {
	sessions := &stubSessionRevoker{}
	svc, _, _ := newChangePasswordSvc(t, &stubPasswordHistoryRepo{}, sessions, true)

	result, err := svc.ChangePassword(context.Background(), 7, currentPlaintext, nextPlaintext, nil)
	require.NoError(t, err)

	assert.True(t, sessions.revokeAllCalled)
	assert.False(t, sessions.revokeExceptCalled)
	assert.True(t, result.ReauthenticationRequired)
}

func TestAccountService_ChangePassword_RejectsWrongCurrentPassword(t *testing.T) {
	svc, _, _ := newChangePasswordSvc(t, &stubPasswordHistoryRepo{}, &stubSessionRevoker{}, false)
	// No transaction is opened at all on this path.
	svc.db = nil

	_, err := svc.ChangePassword(context.Background(), 7, "not-the-current-password", nextPlaintext, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid current password")
}

// A tenant with HistoryCount = 0 has no reuse check, so without this the
// endpoint would happily "change" a password to itself and report success.
func TestAccountService_ChangePassword_RejectsReusingTheCurrentPassword(t *testing.T) {
	svc, _, _ := newChangePasswordSvc(t, &stubPasswordHistoryRepo{}, &stubSessionRevoker{}, false)
	svc.db = nil

	_, err := svc.ChangePassword(context.Background(), 7, currentPlaintext, currentPlaintext, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "must be different")
}

// This is the check that had exactly one caller before this endpoint existed.
func TestAccountService_ChangePassword_RejectsARecentlyUsedPassword(t *testing.T) {
	previous, err := security.HashPasswordWithPolicy(context.Background(), []byte(nextPlaintext),
		security.PasswordPolicy{HashAlgorithm: "bcrypt"})
	require.NoError(t, err)

	svc, _, _ := newChangePasswordSvc(t,
		&stubPasswordHistoryRepo{recent: []string{string(previous)}}, &stubSessionRevoker{}, false)
	svc.db = nil

	_, err = svc.ChangePassword(context.Background(), 7, currentPlaintext, nextPlaintext, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "used recently")
}

// An unreadable history is not an empty history.
func TestAccountService_ChangePassword_FailsClosedWhenHistoryIsUnreadable(t *testing.T) {
	svc, _, _ := newChangePasswordSvc(t,
		&stubPasswordHistoryRepo{findErr: errors.New("db down")}, &stubSessionRevoker{}, false)
	svc.db = nil

	_, err := svc.ChangePassword(context.Background(), 7, currentPlaintext, nextPlaintext, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to read password history")
}

// A federated or passwordless account has nothing to rotate — 400, not a panic
// or a 500.
func TestAccountService_ChangePassword_RejectsAccountWithNoPassword(t *testing.T) {
	svc := &accountService{
		authEventService: authevent.NoopService(),
		userRepo: &mockUserRepo{
			findByIDFn: func(_ any, _ ...string) (*User, error) {
				return &User{UserID: 7, UserUUID: uuid.New(), Password: nil}, nil
			},
		},
	}

	_, err := svc.ChangePassword(context.Background(), 7, currentPlaintext, nextPlaintext, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no password set")
}

// The new password must not simply restate the account's own identity. This is
// the one password flow that always knows exactly whose password it is.
func TestAccountService_ChangePassword_RejectsAPasswordContainingTheUsername(t *testing.T) {
	svc, _, _ := newChangePasswordSvc(t, &stubPasswordHistoryRepo{}, &stubSessionRevoker{}, false)
	svc.db = nil

	_, err := svc.ChangePassword(context.Background(), 7, currentPlaintext, "jbarnes-jbarnes-99", nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "username")
}
