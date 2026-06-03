package iam

import (
	"context"
	"errors"
	"testing"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type recordingInvalidator struct {
	subs []string
}

func (r *recordingInvalidator) InvalidateUserAll(_ context.Context, sub string) {
	r.subs = append(r.subs, sub)
}

func (r *recordingInvalidator) InvalidateUser(context.Context, string, string) {}
func (r *recordingInvalidator) InvalidateAllUsers(context.Context)             {}

func TestDBAuthorizationTokenInvalidator_InvalidateRoleChange(t *testing.T) {
	db, mock := newMockGormDB(t)
	cacheInvalidator := &recordingInvalidator{}
	invalidator := NewDBAuthorizationTokenInvalidator(db, cacheInvalidator)

	mock.ExpectQuery(`SELECT DISTINCT "user_id" FROM "user_roles" WHERE role_id IN \(\$1\)`).
		WithArgs(int64(7)).
		WillReturnRows(sqlmock.NewRows([]string{"user_id"}).AddRow(int64(11)).AddRow(int64(12)))
	mock.ExpectBegin()
	mock.ExpectExec(`UPDATE "user_tokens" SET "is_revoked"=\$1 WHERE user_id IN \(\$2,\$3\)`).
		WithArgs(true, int64(11), int64(12)).
		WillReturnResult(sqlmock.NewResult(0, 2))
	mock.ExpectCommit()
	mock.ExpectQuery(`SELECT DISTINCT "sub" FROM "user_identities" WHERE user_id IN \(\$1,\$2\) AND sub <> ''`).
		WithArgs(int64(11), int64(12)).
		WillReturnRows(sqlmock.NewRows([]string{"sub"}).AddRow("sub-11").AddRow("sub-12"))

	require.NoError(t, invalidator.InvalidateRoleChange(context.Background(), 7))
	assert.ElementsMatch(t, []string{"sub-11", "sub-12"}, cacheInvalidator.subs)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestDBAuthorizationTokenInvalidator_EdgeCases(t *testing.T) {
	t.Run("constructor uses nop invalidator when nil", func(t *testing.T) {
		db, _ := newMockGormDB(t)
		got := NewDBAuthorizationTokenInvalidator(db, nil)
		assert.NotNil(t, got)
	})

	t.Run("nil receiver db and empty role IDs are noops", func(t *testing.T) {
		var invalidator *dbAuthorizationTokenInvalidator
		require.NoError(t, invalidator.InvalidateRoleChange(context.Background(), 1))
		require.NoError(t, invalidator.InvalidatePermissionChange(context.Background(), 1))

		invalidator = &dbAuthorizationTokenInvalidator{}
		require.NoError(t, invalidator.InvalidateRoleChange(context.Background(), 1))
		require.NoError(t, invalidator.InvalidatePermissionChange(context.Background(), 1))
		db, _ := newMockGormDB(t)
		invalidator = &dbAuthorizationTokenInvalidator{db: db}
		require.NoError(t, invalidator.InvalidateRoleChange(context.Background()))
	})

	t.Run("role lookup error is returned", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		invalidator := NewDBAuthorizationTokenInvalidator(db, nil)
		mock.ExpectQuery(`SELECT DISTINCT "user_id" FROM "user_roles".*`).
			WillReturnError(errors.New("pluck error"))

		err := invalidator.InvalidateRoleChange(context.Background(), 7)

		require.Error(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("permission lookup error is returned", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		invalidator := NewDBAuthorizationTokenInvalidator(db, nil)
		mock.ExpectQuery(`SELECT DISTINCT "role_id" FROM "role_permissions".*`).
			WillReturnError(errors.New("pluck error"))

		err := invalidator.InvalidatePermissionChange(context.Background(), 9)

		require.Error(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("permission change with no roles is noop after lookup", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		invalidator := NewDBAuthorizationTokenInvalidator(db, nil)
		mock.ExpectQuery(`SELECT DISTINCT "role_id" FROM "role_permissions".*`).
			WillReturnRows(sqlmock.NewRows([]string{"role_id"}))

		require.NoError(t, invalidator.InvalidatePermissionChange(context.Background(), 9))
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("invalidate users update error is returned", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		invalidator := &dbAuthorizationTokenInvalidator{db: db, cacheInvalidator: &recordingInvalidator{}}
		mock.ExpectBegin()
		mock.ExpectExec(`UPDATE "user_tokens".*`).WillReturnError(errors.New("update error"))
		mock.ExpectRollback()

		err := invalidator.invalidateUsers(context.Background(), []int64{11})

		require.Error(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("invalidate users with empty users is noop", func(t *testing.T) {
		db, _ := newMockGormDB(t)
		invalidator := &dbAuthorizationTokenInvalidator{db: db, cacheInvalidator: &recordingInvalidator{}}

		require.NoError(t, invalidator.invalidateUsers(context.Background(), nil))
	})

	t.Run("invalidate users sub lookup error is returned", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		invalidator := &dbAuthorizationTokenInvalidator{db: db, cacheInvalidator: &recordingInvalidator{}}
		mock.ExpectBegin()
		mock.ExpectExec(`UPDATE "user_tokens".*`).WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectCommit()
		mock.ExpectQuery(`SELECT DISTINCT "sub" FROM "user_identities".*`).WillReturnError(errors.New("sub error"))

		err := invalidator.invalidateUsers(context.Background(), []int64{11})

		require.Error(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}
