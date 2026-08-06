package user

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/maintainerd/maintainerd-auth/internal/shared"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Locks the request-path status gate.
//
// Deactivating, suspending, or un-completing an account only wrote users.status;
// no request path read it back, so an already-issued access token kept returning
// 200 until it expired. The gate lives in the two resolvers that build the
// request AuthContext — FindBySubAndClientID (first-party middleware) and
// FindByUserID (multi-issuer middleware) — because every authenticated request
// funnels through one of them. Deleting either gate must fail here.
//
// Non-active resolves to (nil, nil), not an error: the middleware turns a nil
// user into 401 without disclosing that the account exists.

// nonActiveStatuses is every value users.status can hold that must not be
// allowed to act on a request. The empty string stands in for an unset column
// and any status added later — the gate is an allowlist of "active", so a new
// status is refused until someone deliberately admits it.
var nonActiveStatuses = []string{
	shared.StatusInactive,
	shared.StatusPending,
	shared.StatusSuspended,
	"banned",
	"",
}

func TestUserService_FindBySubAndClientID_RefusesNonActiveUser(t *testing.T) {
	const sub, clientID = "sub-1", "client-1"

	for _, status := range nonActiveStatuses {
		t.Run("status="+statusLabel(status), func(t *testing.T) {
			ur, ui, urr, rr, tr, idp, cr := defaultMocks()
			ur.findBySubAndClientIDFn = func(_, _ string) (*User, error) {
				return &User{UserID: 1, UserUUID: uuid.New(), Status: status}, nil
			}
			_, svc := fullUserSvc(t, ur, ui, urr, rr, tr, idp, cr)

			got, err := svc.FindBySubAndClientID(context.Background(), sub, clientID)
			require.NoError(t, err)
			assert.Nil(t, got, "a %q user must not resolve into a request AuthContext", status)
		})
	}

	t.Run("active user still resolves", func(t *testing.T) {
		ur, ui, urr, rr, tr, idp, cr := defaultMocks()
		want := &User{UserID: 7, UserUUID: uuid.New(), Status: shared.StatusActive}
		ur.findBySubAndClientIDFn = func(gotSub, gotClient string) (*User, error) {
			assert.Equal(t, sub, gotSub)
			assert.Equal(t, clientID, gotClient)
			return want, nil
		}
		_, svc := fullUserSvc(t, ur, ui, urr, rr, tr, idp, cr)

		got, err := svc.FindBySubAndClientID(context.Background(), sub, clientID)
		require.NoError(t, err)
		require.NotNil(t, got)
		assert.Equal(t, want.UserID, got.UserID)
	})

	t.Run("unresolved user stays nil", func(t *testing.T) {
		ur, ui, urr, rr, tr, idp, cr := defaultMocks()
		ur.findBySubAndClientIDFn = func(_, _ string) (*User, error) { return nil, nil }
		_, svc := fullUserSvc(t, ur, ui, urr, rr, tr, idp, cr)

		got, err := svc.FindBySubAndClientID(context.Background(), sub, clientID)
		require.NoError(t, err)
		assert.Nil(t, got)
	})

	t.Run("repository error is not swallowed into a nil user", func(t *testing.T) {
		ur, ui, urr, rr, tr, idp, cr := defaultMocks()
		ur.findBySubAndClientIDFn = func(_, _ string) (*User, error) { return nil, errors.New("db down") }
		_, svc := fullUserSvc(t, ur, ui, urr, rr, tr, idp, cr)

		got, err := svc.FindBySubAndClientID(context.Background(), sub, clientID)
		require.Error(t, err)
		assert.Nil(t, got)
	})
}

func TestUserService_FindByUserID_RefusesNonActiveUser(t *testing.T) {
	for _, status := range nonActiveStatuses {
		t.Run("status="+statusLabel(status), func(t *testing.T) {
			ur, ui, urr, rr, tr, idp, cr := defaultMocks()
			ur.findByIDFn = func(_ any, _ ...string) (*User, error) {
				return &User{UserID: 1, UserUUID: uuid.New(), Status: status}, nil
			}
			_, svc := fullUserSvc(t, ur, ui, urr, rr, tr, idp, cr)

			got, err := svc.FindByUserID(context.Background(), 1)
			require.NoError(t, err)
			assert.Nil(t, got, "a %q user must not resolve through the multi-issuer path either", status)
		})
	}

	t.Run("active user still resolves", func(t *testing.T) {
		ur, ui, urr, rr, tr, idp, cr := defaultMocks()
		ur.findByIDFn = func(_ any, _ ...string) (*User, error) {
			return &User{UserID: 9, UserUUID: uuid.New(), Status: shared.StatusActive}, nil
		}
		_, svc := fullUserSvc(t, ur, ui, urr, rr, tr, idp, cr)

		got, err := svc.FindByUserID(context.Background(), 9)
		require.NoError(t, err)
		require.NotNil(t, got)
		assert.Equal(t, int64(9), got.UserID)
	})
}

// The resolver gate only runs on a cache MISS: UserContextMiddleware serves a
// cached UserContext without consulting the service at all. Disabling an account
// therefore has to drop the cached entry, or the old context keeps being served
// for the rest of the TTL and the gate above never gets a chance to refuse it.
func TestUserService_SetStatus_EvictsCachedUserContext(t *testing.T) {
	uid, updaterUUID := uuid.New(), uuid.New()
	const tenantID int64 = 1

	newSvc := func(t *testing.T, finalStatus string) (UserService, *recordingInvalidator, func()) {
		t.Helper()
		ur, ui, urr, rr, tr, idp, cr := defaultMocks()
		callCount := 0
		ur.findByUUIDFn = func(_ any, _ ...string) (*User, error) {
			callCount++
			switch callCount {
			case 1:
				return &User{UserID: 1, UserIdentities: []UserIdentity{{TenantID: tenantID, Tenant: &Tenant{TenantID: tenantID}}}}, nil
			case 2:
				return userWithAccess(2, tenantID), nil
			default:
				return &User{
					UserUUID:       uid,
					Status:         finalStatus,
					UserIdentities: []UserIdentity{{Sub: "sub-1"}, {Sub: "sub-2"}},
				}, nil
			}
		}
		db, mock := newMockGormDB(t)
		inv := &recordingInvalidator{}
		svc := NewUserService(db, ur, ui, urr, rr, tr, idp, cr, inv, nil, nil, nil, nil, nil)
		mock.ExpectBegin()
		if finalStatus != shared.StatusActive {
			expectCredentialRevocation(mock)
		}
		mock.ExpectCommit()
		return svc, inv, func() { require.NoError(t, mock.ExpectationsWereMet()) }
	}

	t.Run("suspending drops every cached sub for the user", func(t *testing.T) {
		svc, inv, assertSQL := newSvc(t, shared.StatusSuspended)
		_, err := svc.SetStatus(context.Background(), uid, tenantID, shared.StatusSuspended, updaterUUID)
		require.NoError(t, err)
		assert.ElementsMatch(t, []string{"sub-1", "sub-2"}, inv.subs)
		assertSQL()
	})

	t.Run("reactivating also drops the cache", func(t *testing.T) {
		svc, inv, assertSQL := newSvc(t, shared.StatusActive)
		_, err := svc.SetStatus(context.Background(), uid, tenantID, shared.StatusActive, updaterUUID)
		require.NoError(t, err)
		assert.ElementsMatch(t, []string{"sub-1", "sub-2"}, inv.subs)
		assertSQL()
	})
}

// The general update endpoint can change status too. Routing round PATCH /status
// must not be a way to leave a disabled user's context cached and live.
func TestUserService_Update_StatusChangeEvictsCachedUserContext(t *testing.T) {
	uid, updaterUUID := uuid.New(), uuid.New()
	const tenantID int64 = 1

	ur, ui, urr, rr, tr, idp, cr := defaultMocks()
	callCount := 0
	ur.findByUUIDFn = func(_ any, _ ...string) (*User, error) {
		callCount++
		switch callCount {
		case 1:
			return &User{
				UserID:         1,
				UserUUID:       uid,
				Username:       "alice",
				Email:          "alice@example.com",
				Status:         shared.StatusActive,
				UserIdentities: []UserIdentity{{TenantID: tenantID, Tenant: &Tenant{TenantID: tenantID}}},
			}, nil
		case 2:
			return userWithAccess(2, tenantID), nil
		default:
			return &User{
				UserUUID:       uid,
				Status:         shared.StatusSuspended,
				UserIdentities: []UserIdentity{{Sub: "sub-1"}},
			}, nil
		}
	}
	db, mock := newMockGormDB(t)
	inv := &recordingInvalidator{}
	svc := NewUserService(db, ur, ui, urr, rr, tr, idp, cr, inv, nil, nil, nil, nil, nil)

	mock.ExpectBegin()
	expectCredentialRevocation(mock)
	mock.ExpectCommit()

	email := "alice@example.com"
	_, err := svc.Update(context.Background(), uid, tenantID, "alice", &email, nil, shared.StatusSuspended, nil, updaterUUID)
	require.NoError(t, err)
	assert.Equal(t, []string{"sub-1"}, inv.subs)
	require.NoError(t, mock.ExpectationsWereMet())
}

// statusLabel keeps subtest names readable when the status under test is the
// empty string.
func statusLabel(status string) string {
	if status == "" {
		return "unset"
	}
	return status
}

// A nil user must not be authenticatable, so a caller that forgets its own nil
// check cannot accidentally admit a missing user.
func TestIsAuthenticatable_NilUserIsRefused(t *testing.T) {
	assert.False(t, isAuthenticatable(nil))
}
