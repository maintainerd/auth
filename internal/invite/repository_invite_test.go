package invite

import (
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/shared"
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

func TestInviteRepository_FindByToken(t *testing.T) {
	now := time.Now()

	t.Run("success", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		// Remove the gorm:preload callback to bypass schema validation for "Roles"
		// (Invite struct does not have a Roles field yet Preload("Roles") is called)
		origCb := db.Callback().Query().Get("gorm:preload")
		_ = db.Callback().Query().Remove("gorm:preload")
		defer func() { _ = db.Callback().Query().Register("gorm:preload", origCb) }()

		testUUID := uuid.New()
		mock.ExpectQuery(`SELECT \* FROM "invites" WHERE invite_token = \$1 AND "invites"\."deleted_at" IS NULL ORDER BY "invites"\."invite_id" LIMIT \$2`).
			WithArgs("token123", 1).
			WillReturnRows(sqlmock.NewRows([]string{"invite_id", "invite_uuid", "tenant_id", "client_id", "invited_email", "invite_token", "status", "created_at", "updated_at", "deleted_at"}).
				AddRow(1, testUUID, 10, 5, "user@example.com", "token123", shared.StatusPending, now, now, nil))

		result, err := NewInviteRepository(db).FindByToken("token123")

		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, "token123", result.InviteToken)
		assert.Equal(t, testUUID, result.InviteUUID)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("not found returns nil", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectQuery(`SELECT \* FROM "invites" WHERE invite_token = \$1 AND "invites"\."deleted_at" IS NULL ORDER BY "invites"\."invite_id" LIMIT \$2`).
			WithArgs("token123", 1).
			WillReturnRows(sqlmock.NewRows([]string{"invite_id", "invite_uuid"}))

		result, err := NewInviteRepository(db).FindByToken("token123")

		require.NoError(t, err)
		assert.Nil(t, result)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("repo error", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectQuery(`SELECT \* FROM "invites" WHERE invite_token = \$1 AND "invites"\."deleted_at" IS NULL`).
			WithArgs("token123", 1).
			WillReturnError(assert.AnError)

		result, err := NewInviteRepository(db).FindByToken("token123")

		require.Error(t, err)
		assert.Nil(t, result)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("role preload error on missing field", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectQuery(`SELECT \* FROM "invites" WHERE invite_token = \$1 AND "invites"\."deleted_at" IS NULL ORDER BY "invites"\."invite_id" LIMIT \$2`).
			WithArgs("token123", 1).
			WillReturnRows(sqlmock.NewRows([]string{"invite_id", "invite_uuid", "tenant_id", "client_id", "invited_email", "invite_token", "status", "created_at", "updated_at", "deleted_at"}).
				AddRow(1, uuid.New(), 10, 5, "user@example.com", "token123", shared.StatusPending, now, now, nil))

		result, err := NewInviteRepository(db).FindByToken("token123")

		require.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "Roles")
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
		mock.ExpectExec(`UPDATE "invites" SET .+ WHERE invite_uuid = \$3 AND "invites"\."deleted_at" IS NULL`).
			WithArgs(shared.StatusAccepted, sqlmock.AnyArg(), testUUID).
			WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectCommit()

		err := NewInviteRepository(db).MarkAsUsed(testUUID)

		require.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("repo error", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectExec(`UPDATE "invites" SET .+ WHERE invite_uuid = \$3 AND "invites"\."deleted_at" IS NULL`).
			WithArgs(shared.StatusAccepted, sqlmock.AnyArg(), testUUID).
			WillReturnError(assert.AnError)
		mock.ExpectRollback()

		err := NewInviteRepository(db).MarkAsUsed(testUUID)

		require.Error(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestInviteRepository_RevokeByUUID(t *testing.T) {
	testUUID := uuid.New()

	t.Run("success", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectExec(`UPDATE "invites" SET .+ WHERE invite_uuid = \$3 AND "invites"\."deleted_at" IS NULL`).
			WithArgs(shared.StatusRevoked, sqlmock.AnyArg(), testUUID).
			WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectCommit()

		err := NewInviteRepository(db).RevokeByUUID(testUUID)

		require.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("repo error", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectExec(`UPDATE "invites" SET .+ WHERE invite_uuid = \$3 AND "invites"\."deleted_at" IS NULL`).
			WithArgs(shared.StatusRevoked, sqlmock.AnyArg(), testUUID).
			WillReturnError(assert.AnError)
		mock.ExpectRollback()

		err := NewInviteRepository(db).RevokeByUUID(testUUID)

		require.Error(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}
