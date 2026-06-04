package user

import (
	"errors"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewUserSettingRepository(t *testing.T) {
	db, _ := newMockGormDB(t)
	repo := NewUserSettingRepository(db)
	require.NotNil(t, repo)
}

func TestUserSettingRepository_WithTx(t *testing.T) {
	db, _ := newMockGormDB(t)
	repo := NewUserSettingRepository(db)
	txDB, _ := newMockGormDB(t)
	txRepo := repo.WithTx(txDB)
	require.NotNil(t, txRepo)
	assert.NotSame(t, repo, txRepo)
}

func TestUserSettingRepository_FindByUserID(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		repo := NewUserSettingRepository(db)

		rows := sqlmock.NewRows([]string{"user_setting_id", "user_setting_uuid", "user_id"}).
			AddRow(1, testResourceUUID, 42)
		mock.ExpectQuery(`SELECT \* FROM "user_settings" WHERE user_id = \$1 ORDER BY "user_settings"\."user_setting_id" LIMIT \$2`).
			WithArgs(int64(42), 1).
			WillReturnRows(rows)

		result, err := repo.FindByUserID(42)
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, int64(1), result.UserSettingID)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("not found", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		repo := NewUserSettingRepository(db)

		mock.ExpectQuery(`SELECT \* FROM "user_settings" WHERE user_id = \$1 ORDER BY "user_settings"\."user_setting_id" LIMIT \$2`).
			WithArgs(int64(99), 1).
			WillReturnRows(sqlmock.NewRows([]string{"user_setting_id", "user_setting_uuid", "user_id"}))

		result, err := repo.FindByUserID(99)
		require.Error(t, err)
		assert.NotNil(t, result)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("db error", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		repo := NewUserSettingRepository(db)

		mock.ExpectQuery(`SELECT \* FROM "user_settings" WHERE user_id = \$1 ORDER BY "user_settings"\."user_setting_id" LIMIT \$2`).
			WithArgs(int64(42), 1).
			WillReturnError(errors.New("db error"))

		result, err := repo.FindByUserID(42)
		require.Error(t, err)
		assert.NotNil(t, result)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestUserSettingRepository_UpdateByUserID(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		repo := NewUserSettingRepository(db)

		mock.ExpectBegin()
		mock.ExpectExec(`UPDATE "user_settings" SET .+ WHERE user_id = \$[0-9]+`).
			WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), int64(42)).
			WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectCommit()

		err := repo.UpdateByUserID(42, &UserSetting{Timezone: strPtr("UTC")})
		require.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("db error", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		repo := NewUserSettingRepository(db)

		mock.ExpectBegin()
		mock.ExpectExec(`UPDATE "user_settings" SET .+ WHERE user_id = \$[0-9]+`).
			WithArgs(sqlmock.AnyArg(), int64(42)).
			WillReturnError(errors.New("db error"))
		mock.ExpectRollback()

		err := repo.UpdateByUserID(42, &UserSetting{})
		require.Error(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestUserSettingRepository_DeleteByUserID(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		repo := NewUserSettingRepository(db)

		mock.ExpectBegin()
		mock.ExpectExec(`DELETE FROM "user_settings" WHERE user_id = \$1`).
			WithArgs(int64(42)).
			WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectCommit()

		err := repo.DeleteByUserID(42)
		require.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("db error", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		repo := NewUserSettingRepository(db)

		mock.ExpectBegin()
		mock.ExpectExec(`DELETE FROM "user_settings" WHERE user_id = \$1`).
			WithArgs(int64(42)).
			WillReturnError(errors.New("db error"))
		mock.ExpectRollback()

		err := repo.DeleteByUserID(42)
		require.Error(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}
