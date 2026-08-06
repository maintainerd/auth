package mfa

import (
	"reflect"
	"strings"
	"testing"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUserMFABackupCodeRepository(t *testing.T) {
	t.Run("constructor and WithTx", func(t *testing.T) {
		db, _ := newMockGormDB(t)
		repo := NewUserMFABackupCodeRepository(db)
		require.NotNil(t, repo)
		assert.NotNil(t, repo.(*userMFABackupCodeRepository).WithTx(db))
	})

	t.Run("CreateBulk stores codes", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectQuery(`INSERT INTO "user_mfa_backup_codes"`).WillReturnRows(sqlmock.NewRows([]string{"backup_code_id"}).AddRow(1))
		mock.ExpectCommit()

		err := NewUserMFABackupCodeRepository(db).CreateBulk([]*UserMFABackupCode{{UserID: mfaTestUserID, CodeHash: "hash"}})

		require.NoError(t, err)
		assertExpectationsMet(t, mock)
	})

	t.Run("FindUnusedByUserID returns rows", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		expectMFASelect(mock, "user_mfa_backup_codes").WillReturnRows(userBackupCodeRows())

		got, err := NewUserMFABackupCodeRepository(db).FindUnusedByUserID(mfaTestUserID)

		require.NoError(t, err)
		assert.Len(t, got, 1)
		assertExpectationsMet(t, mock)
	})

	// Inverted from "FindByUserIDAndCodeHash success not found and error", which
	// asserted that a `code_hash = ?` equality lookup worked. Backup codes are
	// stored bcrypt-hashed, so that lookup can never match a row — the old test
	// was encoding the split-hash scheme that made recovery permanently
	// impossible. Redemption goes through FindUnusedByUserID +
	// bcrypt.CompareHashAndPassword, so the equality lookup must stay gone.
	t.Run("no code-hash equality lookup is exposed", func(t *testing.T) {
		repoType := reflect.TypeOf((*UserMFABackupCodeRepository)(nil)).Elem()
		for i := range repoType.NumMethod() {
			name := repoType.Method(i).Name
			assert.False(t, strings.Contains(name, "CodeHash"),
				"%s reintroduces hash-equality redemption; use FindUnusedByUserID + bcrypt compare", name)
		}
	})

	t.Run("MarkUsed and DeleteAllByUserID mutate rows", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		expectMFAUpdate(mock, "user_mfa_backup_codes").WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectCommit()
		mock.ExpectBegin()
		expectMFADelete(mock, "user_mfa_backup_codes").WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectCommit()
		repo := NewUserMFABackupCodeRepository(db)

		require.NoError(t, repo.MarkUsed(1))
		require.NoError(t, repo.DeleteAllByUserID(mfaTestUserID))
		assertExpectationsMet(t, mock)
	})
}
