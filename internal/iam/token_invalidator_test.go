package iam

import (
	"context"
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
