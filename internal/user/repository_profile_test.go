package user

import (
	"errors"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewProfileRepository(t *testing.T) {
	db, _ := newMockGormDB(t)
	repo := NewProfileRepository(db)
	require.NotNil(t, repo)
}

func TestProfileRepository_WithTx(t *testing.T) {
	db, _ := newMockGormDB(t)
	repo := NewProfileRepository(db)
	txDB, _ := newMockGormDB(t)
	txRepo := repo.WithTx(txDB)
	require.NotNil(t, txRepo)
	assert.NotSame(t, repo, txRepo)
}

func TestProfileRepository_FindByUserID(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		repo := NewProfileRepository(db)

		rows := sqlmock.NewRows([]string{"profile_id", "profile_uuid", "user_id", "first_name", "is_default"}).
			AddRow(1, testResourceUUID, 42, "John", true)
		mock.ExpectQuery(`SELECT \* FROM "profiles" WHERE user_id = \$1 AND "profiles"\."deleted_at" IS NULL ORDER BY "profiles"\."profile_id" LIMIT \$2`).
			WithArgs(int64(42), 1).
			WillReturnRows(rows)

		result, err := repo.FindByUserID(42)
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, int64(42), result.UserID)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("not found", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		repo := NewProfileRepository(db)

		mock.ExpectQuery(`SELECT \* FROM "profiles" WHERE user_id = \$1`).
			WillReturnRows(sqlmock.NewRows([]string{"profile_id", "profile_uuid", "user_id", "first_name"}))

		result, err := repo.FindByUserID(99)
		require.NoError(t, err)
		assert.Nil(t, result)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("db error", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		repo := NewProfileRepository(db)

		mock.ExpectQuery(`SELECT \* FROM "profiles" WHERE`).
			WillReturnError(errors.New("db error"))

		result, err := repo.FindByUserID(42)
		require.Error(t, err)
		assert.Nil(t, result)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestProfileRepository_FindDefaultByUserID(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		repo := NewProfileRepository(db)

		rows := sqlmock.NewRows([]string{"profile_id", "profile_uuid", "user_id", "is_default"}).
			AddRow(1, testResourceUUID, 42, true)
		mock.ExpectQuery(`SELECT \* FROM "profiles" WHERE \(user_id = \$1 AND is_default = \$2\) AND "profiles"\."deleted_at" IS NULL ORDER BY "profiles"\."profile_id" LIMIT \$3`).
			WithArgs(int64(42), true, 1).
			WillReturnRows(rows)

		result, err := repo.FindDefaultByUserID(42)
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.True(t, result.IsDefault)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("not found", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		repo := NewProfileRepository(db)

		mock.ExpectQuery(`SELECT \* FROM "profiles" WHERE`).
			WillReturnRows(sqlmock.NewRows([]string{"profile_id", "profile_uuid", "user_id"}))

		result, err := repo.FindDefaultByUserID(99)
		require.NoError(t, err)
		assert.Nil(t, result)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("db error", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		repo := NewProfileRepository(db)

		mock.ExpectQuery(`SELECT \* FROM "profiles" WHERE`).
			WillReturnError(errors.New("db error"))

		_, err := repo.FindDefaultByUserID(42)
		require.Error(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestProfileRepository_FindAllByUserID(t *testing.T) {
	t.Run("success with results", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		repo := NewProfileRepository(db)

		mock.ExpectQuery(`SELECT count\(\*\) FROM "profiles" WHERE`).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(2))
		mock.ExpectQuery(`SELECT \* FROM "profiles" WHERE .+ LIMIT \$[0-9]+`).
			WillReturnRows(sqlmock.NewRows([]string{"profile_id", "profile_uuid", "user_id", "first_name"}).
				AddRow(1, testResourceUUID, 42, "John").
				AddRow(2, testUserUUID, 42, "Jane"))

		result, err := repo.FindAllByUserID(ProfileRepositoryGetFilter{
			UserID: 42, Page: 1, Limit: 10, SortBy: "created_at", SortOrder: SortOrderDesc,
		})
		require.NoError(t, err)
		assert.Equal(t, int64(2), result.Total)
		assert.Len(t, result.Data, 2)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("empty results", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		repo := NewProfileRepository(db)

		mock.ExpectQuery(`SELECT count\(\*\) FROM "profiles" WHERE`).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))
		mock.ExpectQuery(`SELECT \* FROM "profiles" WHERE`).
			WillReturnRows(sqlmock.NewRows([]string{"profile_id", "profile_uuid", "user_id"}))

		result, err := repo.FindAllByUserID(ProfileRepositoryGetFilter{
			UserID: 42, Page: 1, Limit: 10, SortBy: "created_at", SortOrder: SortOrderDesc,
		})
		require.NoError(t, err)
		assert.Equal(t, int64(0), result.Total)
		assert.Len(t, result.Data, 0)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("count error", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		repo := NewProfileRepository(db)

		mock.ExpectQuery(`SELECT count\(\*\) FROM "profiles" WHERE`).
			WillReturnError(errors.New("db error"))

		_, err := repo.FindAllByUserID(ProfileRepositoryGetFilter{
			UserID: 42, Page: 1, Limit: 10, SortBy: "created_at", SortOrder: SortOrderDesc,
		})
		require.Error(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("data error", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		repo := NewProfileRepository(db)

		mock.ExpectQuery(`SELECT count\(\*\) FROM "profiles" WHERE`).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
		mock.ExpectQuery(`SELECT \* FROM "profiles" WHERE`).
			WillReturnError(errors.New("db error"))

		_, err := repo.FindAllByUserID(ProfileRepositoryGetFilter{
			UserID: 42, Page: 1, Limit: 10, SortBy: "created_at", SortOrder: SortOrderDesc,
		})
		require.Error(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("with first name filter", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		repo := NewProfileRepository(db)
		fn := "John"

		mock.ExpectQuery(`SELECT count\(\*\) FROM "profiles" WHERE .+`).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
		mock.ExpectQuery(`SELECT \* FROM "profiles" WHERE .+`).
			WillReturnRows(sqlmock.NewRows([]string{"profile_id", "profile_uuid", "user_id", "first_name"}).
				AddRow(1, testResourceUUID, 42, "John"))

		result, err := repo.FindAllByUserID(ProfileRepositoryGetFilter{
			UserID: 42, FirstName: &fn, Page: 1, Limit: 10, SortBy: "created_at", SortOrder: SortOrderDesc,
		})
		require.NoError(t, err)
		assert.Equal(t, int64(1), result.Total)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("with email filter", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		repo := NewProfileRepository(db)
		e := "test@test.com"

		mock.ExpectQuery(`SELECT count\(\*\) FROM "profiles"`).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
		mock.ExpectQuery(`SELECT .+ FROM "profiles" JOIN users ON users.user_id = profiles.user_id WHERE`).
			WillReturnRows(sqlmock.NewRows([]string{"profile_id", "profile_uuid", "user_id"}).
				AddRow(1, testResourceUUID, 42))

		result, err := repo.FindAllByUserID(ProfileRepositoryGetFilter{
			UserID: 42, Email: &e, Page: 1, Limit: 10, SortBy: "created_at", SortOrder: SortOrderDesc,
		})
		require.NoError(t, err)
		assert.Equal(t, int64(1), result.Total)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("with phone filter", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		repo := NewProfileRepository(db)
		p := "1234567890"

		mock.ExpectQuery(`SELECT count\(\*\) FROM "profiles" WHERE .+`).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
		mock.ExpectQuery(`SELECT \* FROM "profiles" WHERE .+`).
			WillReturnRows(sqlmock.NewRows([]string{"profile_id", "profile_uuid", "user_id"}).
				AddRow(1, testResourceUUID, 42))

		result, err := repo.FindAllByUserID(ProfileRepositoryGetFilter{
			UserID: 42, Phone: &p, Page: 1, Limit: 10, SortBy: "created_at", SortOrder: SortOrderDesc,
		})
		require.NoError(t, err)
		assert.Equal(t, int64(1), result.Total)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("with is default filter", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		repo := NewProfileRepository(db)
		d := true

		mock.ExpectQuery(`SELECT count\(\*\) FROM "profiles" WHERE .+is_default = \$[0-9]+`).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
		mock.ExpectQuery(`SELECT \* FROM "profiles" WHERE .+is_default = \$[0-9]+`).
			WillReturnRows(sqlmock.NewRows([]string{"profile_id", "profile_uuid", "user_id"}).
				AddRow(1, testResourceUUID, 42))

		result, err := repo.FindAllByUserID(ProfileRepositoryGetFilter{
			UserID: 42, IsDefault: &d, Page: 1, Limit: 10, SortBy: "created_at", SortOrder: SortOrderDesc,
		})
		require.NoError(t, err)
		assert.Equal(t, int64(1), result.Total)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("with last name filter", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		repo := NewProfileRepository(db)
		ln := "Doe"

		mock.ExpectQuery(`SELECT count\(\*\) FROM "profiles" WHERE .+`).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
		mock.ExpectQuery(`SELECT \* FROM "profiles" WHERE .+`).
			WillReturnRows(sqlmock.NewRows([]string{"profile_id", "profile_uuid", "user_id", "last_name"}).
				AddRow(1, testResourceUUID, 42, "Doe"))

		result, err := repo.FindAllByUserID(ProfileRepositoryGetFilter{
			UserID: 42, LastName: &ln, Page: 1, Limit: 10, SortBy: "created_at", SortOrder: SortOrderDesc,
		})
		require.NoError(t, err)
		assert.Equal(t, int64(1), result.Total)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestProfileRepository_UpdateByUserID(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		repo := NewProfileRepository(db)

		mock.ExpectBegin()
		mock.ExpectExec(`UPDATE "profiles" SET .+ WHERE user_id = \$[0-9]+ AND "profiles"\."deleted_at" IS NULL`).
			WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectCommit()

		err := repo.UpdateByUserID(42, &Profile{FirstName: "Jane"})
		require.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("db error", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		repo := NewProfileRepository(db)

		mock.ExpectBegin()
		mock.ExpectExec(`UPDATE "profiles" SET .+ WHERE user_id = \$[0-9]+`).
			WillReturnError(errors.New("db error"))
		mock.ExpectRollback()

		err := repo.UpdateByUserID(42, &Profile{})
		require.Error(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestProfileRepository_DeleteByUserID(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		repo := NewProfileRepository(db)

		mock.ExpectBegin()
		mock.ExpectExec(`UPDATE "profiles" SET "deleted_at"=\$1 WHERE user_id = \$2 AND "profiles"\."deleted_at" IS NULL`).
			WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectCommit()

		err := repo.DeleteByUserID(42)
		require.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("db error", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		repo := NewProfileRepository(db)

		mock.ExpectBegin()
		mock.ExpectExec(`UPDATE "profiles" SET .+ WHERE user_id = \$[0-9]+`).
			WillReturnError(errors.New("db error"))
		mock.ExpectRollback()

		err := repo.DeleteByUserID(42)
		require.Error(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestProfileRepository_UnsetDefaultProfiles(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		repo := NewProfileRepository(db)

		mock.ExpectBegin()
		mock.ExpectExec(`UPDATE "profiles" SET "is_default"=\$1,"updated_at"=\$2 WHERE \(user_id = \$3 AND is_default = \$4\) AND "profiles"\."deleted_at" IS NULL`).
			WillReturnResult(sqlmock.NewResult(0, 2))
		mock.ExpectCommit()

		err := repo.UnsetDefaultProfiles(42)
		require.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("db error", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		repo := NewProfileRepository(db)

		mock.ExpectBegin()
		mock.ExpectExec(`UPDATE "profiles" SET .+ WHERE \(user_id = \$[0-9]+`).
			WillReturnError(errors.New("db error"))
		mock.ExpectRollback()

		err := repo.UnsetDefaultProfiles(42)
		require.Error(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}
