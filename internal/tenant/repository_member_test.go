package tenant

import (
	"testing"
	"time"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTenantMemberRepository_FindByTenantMemberUUID(t *testing.T) {
	memberUUID := uuid.New()
	now := time.Now()

	t.Run("success", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectQuery(`SELECT \* FROM "tenant_members" WHERE tenant_member_uuid = \$1 AND "tenant_members"."deleted_at" IS NULL ORDER BY "tenant_members"."tenant_member_id" LIMIT \$2`).
			WithArgs(memberUUID, 1).
			WillReturnRows(sqlmock.NewRows([]string{"tenant_member_id", "tenant_member_uuid", "tenant_id", "user_id", "role", "created_at", "updated_at"}).
				AddRow(1, memberUUID, 10, 20, "owner", now, now))

		result, err := NewTenantMemberRepository(db).FindByTenantMemberUUID(memberUUID)

		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, "owner", result.Role)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("not found returns nil", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectQuery(`SELECT \* FROM "tenant_members" WHERE tenant_member_uuid = \$1 AND "tenant_members"."deleted_at" IS NULL ORDER BY "tenant_members"."tenant_member_id" LIMIT \$2`).
			WithArgs(memberUUID, 1).
			WillReturnRows(sqlmock.NewRows([]string{"tenant_member_id", "tenant_member_uuid"}))

		result, err := NewTenantMemberRepository(db).FindByTenantMemberUUID(memberUUID)

		require.NoError(t, err)
		assert.Nil(t, result)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("repo error", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectQuery(`SELECT \* FROM "tenant_members" WHERE tenant_member_uuid = \$1`).
			WithArgs(memberUUID, 1).
			WillReturnError(assert.AnError)

		result, err := NewTenantMemberRepository(db).FindByTenantMemberUUID(memberUUID)

		require.Error(t, err)
		assert.Nil(t, result)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestTenantMemberRepository_FindByTenantAndUser(t *testing.T) {
	now := time.Now()

	t.Run("success", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectQuery(`SELECT \* FROM "tenant_members" WHERE \(tenant_id = \$1 AND user_id = \$2\) AND "tenant_members"."deleted_at" IS NULL ORDER BY "tenant_members"."tenant_member_id" LIMIT \$3`).
			WithArgs(int64(10), int64(20), 1).
			WillReturnRows(sqlmock.NewRows([]string{"tenant_member_id", "tenant_member_uuid", "tenant_id", "user_id", "role", "created_at", "updated_at"}).
				AddRow(1, uuid.New(), 10, 20, "member", now, now))

		result, err := NewTenantMemberRepository(db).FindByTenantAndUser(10, 20)

		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, int64(10), result.TenantID)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("not found returns nil", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectQuery(`SELECT \* FROM "tenant_members" WHERE \(tenant_id = \$1 AND user_id = \$2\) AND "tenant_members"."deleted_at" IS NULL ORDER BY "tenant_members"."tenant_member_id" LIMIT \$3`).
			WithArgs(int64(10), int64(20), 1).
			WillReturnRows(sqlmock.NewRows([]string{"tenant_member_id", "tenant_member_uuid"}))

		result, err := NewTenantMemberRepository(db).FindByTenantAndUser(10, 20)

		require.NoError(t, err)
		assert.Nil(t, result)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("repo error", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectQuery(`SELECT \* FROM "tenant_members" WHERE \(tenant_id = \$1 AND user_id = \$2\)`).
			WithArgs(int64(10), int64(20), 1).
			WillReturnError(assert.AnError)

		result, err := NewTenantMemberRepository(db).FindByTenantAndUser(10, 20)

		require.Error(t, err)
		assert.Nil(t, result)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestTenantMemberRepository_FindByTenant(t *testing.T) {
	db, mock := newMockGormDB(t)
	now := time.Now()
	role := "member"

	mock.ExpectQuery(`SELECT count\(\*\) FROM "tenant_members" WHERE tenant_id = \$1 AND role = \$2 AND "tenant_members"."deleted_at" IS NULL`).
		WithArgs(int64(10), role).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
	mock.ExpectQuery(`SELECT \* FROM "tenant_members" WHERE tenant_id = \$1 AND role = \$2 AND "tenant_members"."deleted_at" IS NULL ORDER BY created_at DESC LIMIT \$3`).
		WithArgs(int64(10), role, 10).
		WillReturnRows(sqlmock.NewRows([]string{"tenant_member_id", "tenant_member_uuid", "tenant_id", "user_id", "role", "created_at", "updated_at"}).
			AddRow(1, uuid.New(), 10, 20, "member", now, now))

	result, err := NewTenantMemberRepository(db).FindByTenant(TenantMemberRepositoryListFilter{
		TenantID: 10,
		Role:     &role,
		Page:     1,
		Limit:    10,
	})

	require.NoError(t, err)
	require.Len(t, result.Data, 1)
	assert.Equal(t, "member", result.Data[0].Role)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestTenantMemberRepository_FindAllByUser(t *testing.T) {
	now := time.Now()

	t.Run("success", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectQuery(`SELECT \* FROM "tenant_members" WHERE user_id = \$1 AND "tenant_members"."deleted_at" IS NULL`).
			WithArgs(int64(20)).
			WillReturnRows(sqlmock.NewRows([]string{"tenant_member_id", "tenant_member_uuid", "tenant_id", "user_id", "role", "created_at", "updated_at"}).
				AddRow(1, uuid.New(), 10, 20, "member", now, now).
				AddRow(2, uuid.New(), 11, 20, "owner", now, now))

		result, err := NewTenantMemberRepository(db).FindAllByUser(20)

		require.NoError(t, err)
		require.Len(t, result, 2)
		assert.Equal(t, "owner", result[1].Role)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("repo error", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectQuery(`SELECT \* FROM "tenant_members" WHERE user_id = \$1`).
			WithArgs(int64(20)).
			WillReturnError(assert.AnError)

		result, err := NewTenantMemberRepository(db).FindAllByUser(20)

		require.Error(t, err)
		assert.Nil(t, result)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestTenantMemberRepository_WithTx(t *testing.T) {
	db, mock := newMockGormDB(t)
	mock.ExpectBegin()
	mock.ExpectRollback()
	tx := db.Begin()
	require.NoError(t, tx.Error)

	repo := NewTenantMemberRepository(db).WithTx(tx)

	assert.NotNil(t, repo)
	require.NoError(t, tx.Rollback().Error)
	assert.NoError(t, mock.ExpectationsWereMet())
}
