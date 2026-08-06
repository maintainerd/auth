package invite

import (
	"database/sql/driver"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/maintainerd/maintainerd-auth/internal/shared"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewInviteRepository(t *testing.T) {
	db, _ := newMockGormDB(t)
	repo := NewInviteRepository(db)
	assert.NotNil(t, repo)
}

func TestInviteRepository_WithTx(t *testing.T) {
	db, mock := newMockGormDB(t)
	mock.ExpectBegin()
	mock.ExpectRollback()
	tx := db.Begin()
	require.NoError(t, tx.Error)

	repo := NewInviteRepository(db).WithTx(tx)
	assert.NotNil(t, repo)
	require.NoError(t, tx.Rollback().Error)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestInviteRepository_FindByUUIDAndTenantID(t *testing.T) {
	testUUID := uuid.New()
	now := time.Now()

	t.Run("success", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectQuery(`SELECT \* FROM "invites" WHERE \(invite_uuid = \$1 AND tenant_id = \$2\) AND "invites"\."deleted_at" IS NULL ORDER BY "invites"\."invite_id" LIMIT \$3`).
			WithArgs(testUUID, int64(10), 1).
			WillReturnRows(sqlmock.NewRows([]string{"invite_id", "invite_uuid", "tenant_id", "client_id", "invited_email", "invite_token", "status", "created_at", "updated_at", "deleted_at"}).
				AddRow(1, testUUID, 10, 5, "user@example.com", "token123", shared.StatusPending, now, now, nil))

		result, err := NewInviteRepository(db).FindByUUIDAndTenantID(testUUID, 10)

		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, testUUID, result.InviteUUID)
		assert.Equal(t, int64(10), result.TenantID)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("not found returns nil", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectQuery(`SELECT \* FROM "invites" WHERE \(invite_uuid = \$1 AND tenant_id = \$2\) AND "invites"\."deleted_at" IS NULL ORDER BY "invites"\."invite_id" LIMIT \$3`).
			WithArgs(testUUID, int64(10), 1).
			WillReturnRows(sqlmock.NewRows([]string{"invite_id", "invite_uuid"}))

		result, err := NewInviteRepository(db).FindByUUIDAndTenantID(testUUID, 10)

		require.NoError(t, err)
		assert.Nil(t, result)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("repo error", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectQuery(`SELECT \* FROM "invites" WHERE \(invite_uuid = \$1 AND tenant_id = \$2\) AND "invites"\."deleted_at" IS NULL`).
			WithArgs(testUUID, int64(10), 1).
			WillReturnError(assert.AnError)

		result, err := NewInviteRepository(db).FindByUUIDAndTenantID(testUUID, 10)

		require.Error(t, err)
		assert.Nil(t, result)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("invalid preload name returns error", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mainUUID := testUUID
		mock.ExpectQuery(`SELECT \* FROM "invites" WHERE \(invite_uuid = \$1 AND tenant_id = \$2\) AND "invites"\."deleted_at" IS NULL ORDER BY "invites"\."invite_id" LIMIT \$3`).
			WithArgs(mainUUID, int64(10), 1).
			WillReturnRows(sqlmock.NewRows([]string{"invite_id", "invite_uuid", "tenant_id", "client_id", "invited_email", "invite_token", "status", "created_at", "updated_at", "deleted_at"}).
				AddRow(1, mainUUID, 10, 5, "user@example.com", "token123", shared.StatusPending, now, now, nil))

		result, err := NewInviteRepository(db).FindByUUIDAndTenantID(testUUID, 10, "NonExistentField")

		require.Error(t, err)
		assert.Nil(t, result)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

// These used to bind the raw "token123" as the WHERE parameter and assert it came
// back on the record — they encoded the plaintext-storage bug. The lookup is now
// keyed on the digest, so WithArgs asserts hashInviteToken("token123") reaches the
// database and the raw token never does.
func TestInviteRepository_FindByToken(t *testing.T) {
	now := time.Now()
	tokenHash := hashInviteToken("token123")

	t.Run("success", func(t *testing.T) {
		db, mock := newMockGormDB(t)

		testUUID := uuid.New()
		mock.ExpectQuery(`SELECT \* FROM "invites" WHERE invite_token = \$1 AND "invites"\."deleted_at" IS NULL ORDER BY "invites"\."invite_id" LIMIT \$2`).
			WithArgs(tokenHash, 1).
			WillReturnRows(sqlmock.NewRows([]string{"invite_id", "invite_uuid", "tenant_id", "client_id", "invited_email", "invite_token", "status", "created_at", "updated_at", "deleted_at"}).
				AddRow(1, testUUID, 10, 5, "user@example.com", tokenHash, shared.StatusPending, now, now, nil))

		result, err := NewInviteRepository(db).FindByToken("token123")

		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, tokenHash, result.InviteTokenHash)
		assert.NotEqual(t, "token123", result.InviteTokenHash)
		assert.Equal(t, testUUID, result.InviteUUID)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("surrounding whitespace still resolves the same row", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectQuery(`SELECT \* FROM "invites" WHERE invite_token = \$1 AND "invites"\."deleted_at" IS NULL ORDER BY "invites"\."invite_id" LIMIT \$2`).
			WithArgs(tokenHash, 1).
			WillReturnRows(sqlmock.NewRows([]string{"invite_id", "invite_uuid", "tenant_id", "client_id", "invited_email", "invite_token", "status", "created_at", "updated_at", "deleted_at"}).
				AddRow(1, uuid.New(), 10, 5, "user@example.com", tokenHash, shared.StatusPending, now, now, nil))

		result, err := NewInviteRepository(db).FindByToken("  token123\n")

		require.NoError(t, err)
		require.NotNil(t, result)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	// Fail closed: a blank token must not reach the database at all. Hashing "" is
	// a fixed, attacker-known digest, so a caller that dropped the parameter would
	// otherwise be running a real lookup.
	t.Run("blank token never queries and reports not found", func(t *testing.T) {
		db, mock := newMockGormDB(t)

		for _, blank := range []string{"", "   "} {
			result, err := NewInviteRepository(db).FindByToken(blank)
			require.NoError(t, err)
			assert.Nil(t, result)
		}
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("not found returns nil", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectQuery(`SELECT \* FROM "invites" WHERE invite_token = \$1 AND "invites"\."deleted_at" IS NULL ORDER BY "invites"\."invite_id" LIMIT \$2`).
			WithArgs(tokenHash, 1).
			WillReturnRows(sqlmock.NewRows([]string{"invite_id", "invite_uuid"}))

		result, err := NewInviteRepository(db).FindByToken("token123")

		require.NoError(t, err)
		assert.Nil(t, result)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("repo error", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectQuery(`SELECT \* FROM "invites" WHERE invite_token = \$1 AND "invites"\."deleted_at" IS NULL`).
			WithArgs(tokenHash, 1).
			WillReturnError(assert.AnError)

		result, err := NewInviteRepository(db).FindByToken("token123")

		require.Error(t, err)
		assert.Nil(t, result)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("success without preload", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectQuery(`SELECT \* FROM "invites" WHERE invite_token = \$1 AND "invites"\."deleted_at" IS NULL ORDER BY "invites"\."invite_id" LIMIT \$2`).
			WithArgs(tokenHash, 1).
			WillReturnRows(sqlmock.NewRows([]string{"invite_id", "invite_uuid", "tenant_id", "client_id", "invited_email", "invite_token", "status", "created_at", "updated_at", "deleted_at"}).
				AddRow(1, uuid.New(), 10, 5, "user@example.com", tokenHash, shared.StatusPending, now, now, nil))

		result, err := NewInviteRepository(db).FindByToken("token123")

		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, tokenHash, result.InviteTokenHash)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestInviteRepository_FindAllByClientID(t *testing.T) {
	now := time.Now()

	t.Run("success", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectQuery(`SELECT \* FROM "invites" WHERE client_id = \$1 AND "invites"\."deleted_at" IS NULL`).
			WithArgs(int64(5)).
			WillReturnRows(sqlmock.NewRows([]string{"invite_id", "invite_uuid", "tenant_id", "client_id", "invited_email", "invite_token", "status", "created_at", "updated_at", "deleted_at"}).
				AddRow(1, uuid.New(), 10, 5, "user@example.com", "token123", shared.StatusPending, now, now, nil))

		result, err := NewInviteRepository(db).FindAllByClientID(5)

		require.NoError(t, err)
		assert.Len(t, result, 1)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("repo error", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectQuery(`SELECT \* FROM "invites" WHERE client_id = \$1 AND "invites"\."deleted_at" IS NULL`).
			WithArgs(int64(5)).
			WillReturnError(assert.AnError)

		result, err := NewInviteRepository(db).FindAllByClientID(5)

		require.Error(t, err)
		assert.Nil(t, result)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestInviteRepository_FindAllByTenantID(t *testing.T) {
	now := time.Now()

	t.Run("success", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectQuery(`SELECT \* FROM "invites" WHERE tenant_id = \$1 AND "invites"\."deleted_at" IS NULL`).
			WithArgs(int64(10)).
			WillReturnRows(sqlmock.NewRows([]string{"invite_id", "invite_uuid", "tenant_id", "client_id", "invited_email", "invite_token", "status", "created_at", "updated_at", "deleted_at"}).
				AddRow(1, uuid.New(), 10, 5, "user@example.com", "token123", shared.StatusPending, now, now, nil))

		result, err := NewInviteRepository(db).FindAllByTenantID(10)

		require.NoError(t, err)
		assert.Len(t, result, 1)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("repo error", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectQuery(`SELECT \* FROM "invites" WHERE tenant_id = \$1 AND "invites"\."deleted_at" IS NULL`).
			WithArgs(int64(10)).
			WillReturnError(assert.AnError)

		result, err := NewInviteRepository(db).FindAllByTenantID(10)

		require.Error(t, err)
		assert.Nil(t, result)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestInviteRepository_MarkAsUsed(t *testing.T) {
	testUUID := uuid.New()

	t.Run("success", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectExec(`UPDATE "invites" SET .+ WHERE \(invite_uuid = \$3 AND status = \$4\) AND "invites"\."deleted_at" IS NULL`).
			WithArgs(shared.StatusAccepted, sqlmock.AnyArg(), testUUID, shared.StatusPending).
			WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectCommit()

		err := NewInviteRepository(db).MarkAsUsed(testUUID)

		require.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("repo error", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectExec(`UPDATE "invites" SET .+ WHERE \(invite_uuid = \$3 AND status = \$4\) AND "invites"\."deleted_at" IS NULL`).
			WithArgs(shared.StatusAccepted, sqlmock.AnyArg(), testUUID, shared.StatusPending).
			WillReturnError(assert.AnError)
		mock.ExpectRollback()

		err := NewInviteRepository(db).MarkAsUsed(testUUID)

		require.Error(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

// The `AND status = pending` predicate asserted below is the fix: this test
// previously asserted a bare `WHERE invite_uuid = $3`, which is exactly what let
// an already-accepted invite be flipped to revoked with its used_at left set.
func TestInviteRepository_RevokeByUUID(t *testing.T) {
	testUUID := uuid.New()
	const revokeSQL = `UPDATE "invites" SET .+ WHERE \(invite_uuid = \$3 AND status = \$4\) AND "invites"\."deleted_at" IS NULL`

	t.Run("success", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectExec(revokeSQL).
			WithArgs(shared.StatusRevoked, sqlmock.AnyArg(), testUUID, shared.StatusPending).
			WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectCommit()

		err := NewInviteRepository(db).RevokeByUUID(testUUID)

		require.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("no pending row matched", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectExec(revokeSQL).
			WithArgs(shared.StatusRevoked, sqlmock.AnyArg(), testUUID, shared.StatusPending).
			WillReturnResult(sqlmock.NewResult(0, 0))
		mock.ExpectCommit()

		err := NewInviteRepository(db).RevokeByUUID(testUUID)

		require.ErrorIs(t, err, ErrInviteNotPending)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("repo error", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectExec(revokeSQL).
			WithArgs(shared.StatusRevoked, sqlmock.AnyArg(), testUUID, shared.StatusPending).
			WillReturnError(assert.AnError)
		mock.ExpectRollback()

		err := NewInviteRepository(db).RevokeByUUID(testUUID)

		require.Error(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

// ResetForResend had no status predicate at all, so it resurrected revoked and
// accepted invites — minting a fresh emailed token and clearing the used_at that
// recorded the acceptance.
func TestInviteRepository_ResetForResend(t *testing.T) {
	testUUID := uuid.New()
	expiry := time.Now().Add(time.Hour)
	const resendSQL = `UPDATE "invites" SET .+ WHERE \(invite_uuid = \$6 AND status IN \(\$7,\$8\)\) AND "invites"\."deleted_at" IS NULL`
	// This used to assert the raw "new-token" was written into invite_token. That
	// encoded the plaintext-storage bug: the resend path persisted a live,
	// redeemable account-creation credential. It now asserts the digest.
	args := []driver.Value{
		expiry, hashInviteToken("new-token"), shared.StatusPending, nil, sqlmock.AnyArg(),
		testUUID, shared.StatusPending, statusExpired,
	}

	t.Run("success", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectExec(resendSQL).WithArgs(args...).WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectCommit()

		err := NewInviteRepository(db).ResetForResend(testUUID, "new-token", expiry)

		require.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("a settled invite matches no row", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectExec(resendSQL).WithArgs(args...).WillReturnResult(sqlmock.NewResult(0, 0))
		mock.ExpectCommit()

		err := NewInviteRepository(db).ResetForResend(testUUID, "new-token", expiry)

		require.ErrorIs(t, err, ErrInviteNotResendable)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("repo error", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectExec(resendSQL).WithArgs(args...).WillReturnError(assert.AnError)
		mock.ExpectRollback()

		err := NewInviteRepository(db).ResetForResend(testUUID, "new-token", expiry)

		require.Error(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

// FindByTokenForUpdate is the path registration consumes an invite through, so it
// must hash exactly like FindByToken and fail closed on a blank token — otherwise
// the row-locking read would be the one place a raw token still had to be stored.
func TestInviteRepository_FindByTokenForUpdate(t *testing.T) {
	now := time.Now()
	tokenHash := hashInviteToken("token123")
	const selectForUpdate = `SELECT \* FROM "invites" WHERE invite_token = \$1 AND "invites"\."deleted_at" IS NULL ORDER BY "invites"\."invite_id" LIMIT \$2 FOR UPDATE`

	t.Run("looks the row up by digest", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		testUUID := uuid.New()
		mock.ExpectQuery(selectForUpdate).
			WithArgs(tokenHash, 1).
			WillReturnRows(sqlmock.NewRows([]string{"invite_id", "invite_uuid", "tenant_id", "client_id", "invited_email", "invite_token", "status", "created_at", "updated_at", "deleted_at"}).
				AddRow(1, testUUID, 10, 5, "user@example.com", tokenHash, shared.StatusPending, now, now, nil))

		result, err := NewInviteRepository(db).FindByTokenForUpdate("token123")

		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, testUUID, result.InviteUUID)
		assert.Equal(t, tokenHash, result.InviteTokenHash)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("blank token never queries and reports not found", func(t *testing.T) {
		db, mock := newMockGormDB(t)

		result, err := NewInviteRepository(db).FindByTokenForUpdate("  ")

		require.NoError(t, err)
		assert.Nil(t, result)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("not found returns nil", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectQuery(selectForUpdate).
			WithArgs(tokenHash, 1).
			WillReturnRows(sqlmock.NewRows([]string{"invite_id", "invite_uuid"}))

		result, err := NewInviteRepository(db).FindByTokenForUpdate("token123")

		require.NoError(t, err)
		assert.Nil(t, result)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("repo error", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectQuery(selectForUpdate).
			WithArgs(tokenHash, 1).
			WillReturnError(assert.AnError)

		result, err := NewInviteRepository(db).FindByTokenForUpdate("token123")

		require.Error(t, err)
		assert.Nil(t, result)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

// hashInviteToken is the whole point of the invites table no longer holding live
// credentials: it must never be the identity function, must be stable, and must
// separate distinct tokens.
func TestHashInviteToken(t *testing.T) {
	const raw = "a-raw-invite-token"

	assert.NotEqual(t, raw, hashInviteToken(raw))
	assert.Equal(t, hashInviteToken(raw), hashInviteToken(raw))
	assert.Equal(t, hashInviteToken(raw), hashInviteToken(" \t"+raw+"\n"))
	assert.NotEqual(t, hashInviteToken(raw), hashInviteToken(raw+"x"))
	// 32 raw bytes of SHA-256 in unpadded base64url.
	assert.Len(t, hashInviteToken(raw), 43)
}
