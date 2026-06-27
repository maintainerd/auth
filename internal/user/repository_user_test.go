package user

import (
	"database/sql/driver"
	"errors"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/shared"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewUserRepository(t *testing.T) {
	db, _ := newMockGormDB(t)
	repo := NewUserRepository(db)
	require.NotNil(t, repo)
}

func TestUserRepository_WithTx(t *testing.T) {
	db, _ := newMockGormDB(t)
	repo := NewUserRepository(db)
	txDB, _ := newMockGormDB(t)
	txRepo := repo.WithTx(txDB)
	require.NotNil(t, txRepo)
	assert.NotSame(t, repo, txRepo)
}

func TestUserRepository_FindByUsernameAndTenantID(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		repo := NewUserRepository(db)

		rows := sqlmock.NewRows([]string{"user_id", "user_uuid", "username", "email"}).
			AddRow(42, testUserUUID, "testuser", "test@test.com")
		mock.ExpectQuery(`SELECT .+ FROM "users"`).
			WithArgs("testuser", int64(1), 1).
			WillReturnRows(rows)

		result, err := repo.FindByUsernameAndTenantID("testuser", 1)
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, "testuser", result.Username)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("not found", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		repo := NewUserRepository(db)

		mock.ExpectQuery(`SELECT .+ FROM "users" WHERE`).
			WillReturnRows(sqlmock.NewRows([]string{"user_id"}))

		result, err := repo.FindByUsernameAndTenantID("missing", 1)
		require.NoError(t, err)
		assert.Nil(t, result)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("db error", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		repo := NewUserRepository(db)

		mock.ExpectQuery(`SELECT .+ FROM "users" WHERE`).
			WillReturnError(errors.New("db error"))

		_, err := repo.FindByUsernameAndTenantID("testuser", 1)
		require.Error(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestUserRepository_FindByEmailAndTenantID_AdditionalCases(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		repo := NewUserRepository(db)

		rows := sqlmock.NewRows([]string{"user_id", "user_uuid", "email"}).
			AddRow(42, testUserUUID, "test@test.com")
		mock.ExpectQuery(`SELECT .+ FROM "users"`).
			WithArgs("test@test.com", int64(1), 1).
			WillReturnRows(rows)

		result, err := repo.FindByEmailAndTenantID("test@test.com", 1)
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, "test@test.com", result.Email)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("not found", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		repo := NewUserRepository(db)

		mock.ExpectQuery(`SELECT .+ FROM "users" WHERE`).
			WillReturnRows(sqlmock.NewRows([]string{"user_id"}))

		result, err := repo.FindByEmailAndTenantID("missing@test.com", 1)
		require.NoError(t, err)
		assert.Nil(t, result)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("db error", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		repo := NewUserRepository(db)

		mock.ExpectQuery(`SELECT .+ FROM "users" WHERE`).
			WillReturnError(errors.New("db error"))

		_, err := repo.FindByEmailAndTenantID("test@test.com", 1)
		require.Error(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestUserRepository_FindByEmailAndTenantID(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		repo := NewUserRepository(db)

		rows := sqlmock.NewRows([]string{"user_id", "user_uuid", "email"}).
			AddRow(42, testUserUUID, "test@test.com")
		mock.ExpectQuery(`SELECT .+ FROM "users" WHERE \(email = \$1 AND tenant_id = \$2\) AND "users"\."deleted_at" IS NULL ORDER BY "users"\."user_id" LIMIT \$3`).
			WithArgs("test@test.com", int64(1), 1).
			WillReturnRows(rows)

		result, err := repo.FindByEmailAndTenantID("test@test.com", 1)
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, "test@test.com", result.Email)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("not found", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		repo := NewUserRepository(db)

		mock.ExpectQuery(`SELECT .+ FROM "users"`).
			WillReturnRows(sqlmock.NewRows([]string{"user_id"}))

		result, err := repo.FindByEmailAndTenantID("missing@test.com", 1)
		require.NoError(t, err)
		assert.Nil(t, result)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("db error", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		repo := NewUserRepository(db)

		mock.ExpectQuery(`SELECT .+ FROM "users"`).
			WillReturnError(errors.New("db error"))

		_, err := repo.FindByEmailAndTenantID("test@test.com", 1)
		require.Error(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestUserRepository_FindByPhoneAndTenantID(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		repo := NewUserRepository(db)

		rows := sqlmock.NewRows([]string{"user_id", "user_uuid", "phone"}).
			AddRow(42, testUserUUID, "1234567890")
		mock.ExpectQuery(`SELECT .+ FROM "users"`).
			WithArgs("1234567890", int64(1), 1).
			WillReturnRows(rows)

		result, err := repo.FindByPhoneAndTenantID("1234567890", 1)
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, "1234567890", result.Phone)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("not found", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		repo := NewUserRepository(db)

		mock.ExpectQuery(`SELECT .+ FROM "users" WHERE`).
			WillReturnRows(sqlmock.NewRows([]string{"user_id"}))

		result, err := repo.FindByPhoneAndTenantID("000", 1)
		require.NoError(t, err)
		assert.Nil(t, result)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("db error", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		repo := NewUserRepository(db)

		mock.ExpectQuery(`SELECT .+ FROM "users" WHERE`).
			WillReturnError(errors.New("db error"))

		_, err := repo.FindByPhoneAndTenantID("1234567890", 1)
		require.Error(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestUserRepository_FindSuperAdmin(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		repo := NewUserRepository(db)

		rows := sqlmock.NewRows([]string{"user_id", "user_uuid", "username", "email"}).
			AddRow(1, testUserUUID, "admin", "admin@test.com")
		mock.ExpectQuery(`SELECT .+ FROM "users"`).
			WillReturnRows(rows)

		result, err := repo.FindSuperAdmin()
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, "admin", result.Username)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("not found", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		repo := NewUserRepository(db)

		mock.ExpectQuery(`SELECT .+ FROM "users"`).
			WillReturnRows(sqlmock.NewRows([]string{"user_id"}))

		result, err := repo.FindSuperAdmin()
		require.NoError(t, err)
		assert.Nil(t, result)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("db error", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		repo := NewUserRepository(db)

		mock.ExpectQuery(`SELECT .+ FROM "users"`).
			WillReturnError(errors.New("db error"))

		_, err := repo.FindSuperAdmin()
		require.Error(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestUserRepository_FindRoles(t *testing.T) {
	t.Run("success with results", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		repo := NewUserRepository(db)

		rows := sqlmock.NewRows([]string{"role_id", "role_uuid", "name"}).
			AddRow(10, testResourceUUID, shared.RoleRegistered)
		mock.ExpectQuery(`SELECT roles\.\* FROM "roles" JOIN user_roles ur ON ur.role_id = roles.role_id WHERE ur.user_id = \$1`).
			WithArgs(int64(42)).
			WillReturnRows(rows)

		result, err := repo.FindRoles(42)
		require.NoError(t, err)
		assert.Len(t, result, 1)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("empty results", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		repo := NewUserRepository(db)

		mock.ExpectQuery(`SELECT .+ FROM "roles"`).
			WillReturnRows(sqlmock.NewRows([]string{"role_id"}))

		result, err := repo.FindRoles(99)
		require.NoError(t, err)
		assert.Len(t, result, 0)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("db error", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		repo := NewUserRepository(db)

		mock.ExpectQuery(`SELECT .+ FROM "roles"`).
			WillReturnError(errors.New("db error"))

		_, err := repo.FindRoles(42)
		require.Error(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestUserRepository_FindRolesPaginated(t *testing.T) {
	t.Run("success with results", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		repo := NewUserRepository(db)

		mock.ExpectQuery(`SELECT count\(.+\) FROM "roles"`).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(2))
		mock.ExpectQuery(`SELECT .+ FROM "roles" WHERE roles.role_id IN .+ LIMIT \$[0-9]+`).
			WillReturnRows(sqlmock.NewRows([]string{"role_id", "role_uuid", "name"}).
				AddRow(10, testResourceUUID, shared.RoleRegistered).
				AddRow(20, testUserUUID, "admin"))

		result, err := repo.FindRolesPaginated(GetUserRolesFilter{
			UserID: 42, Page: 1, Limit: 10, SortBy: "created_at", SortOrder: SortOrderDesc,
		})
		require.NoError(t, err)
		assert.Equal(t, int64(2), result.Total)
		assert.Len(t, result.Data, 2)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("empty results", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		repo := NewUserRepository(db)

		mock.ExpectQuery(`SELECT count\(.+\) FROM "roles"`).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))
		mock.ExpectQuery(`SELECT .+ FROM "roles"`).
			WillReturnRows(sqlmock.NewRows([]string{"role_id"}))

		result, err := repo.FindRolesPaginated(GetUserRolesFilter{
			UserID: 42, Page: 1, Limit: 10, SortBy: "created_at", SortOrder: SortOrderDesc,
		})
		require.NoError(t, err)
		assert.Equal(t, int64(0), result.Total)
		assert.Len(t, result.Data, 0)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("count error", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		repo := NewUserRepository(db)

		mock.ExpectQuery(`SELECT count\(.+\) FROM "roles"`).
			WillReturnError(errors.New("db error"))

		_, err := repo.FindRolesPaginated(GetUserRolesFilter{
			UserID: 42, Page: 1, Limit: 10, SortBy: "created_at", SortOrder: SortOrderDesc,
		})
		require.Error(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("with status filter", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		repo := NewUserRepository(db)
		active := "active"

		mock.ExpectQuery(`SELECT count\(.+\) FROM "roles"`).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
		mock.ExpectQuery(`SELECT .+ FROM "roles"`).
			WillReturnRows(sqlmock.NewRows([]string{"role_id", "role_uuid", "name"}).
				AddRow(10, testResourceUUID, shared.RoleRegistered))

		result, err := repo.FindRolesPaginated(GetUserRolesFilter{
			UserID: 42, Page: 1, Limit: 10, SortBy: "created_at", SortOrder: SortOrderDesc,
			Status: &active,
		})
		require.NoError(t, err)
		assert.Equal(t, int64(1), result.Total)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestUserRepository_FindBySubAndClientID(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.MatchExpectationsInOrder(false)
		repo := NewUserRepository(db)

		rows := sqlmock.NewRows([]string{"user_id", "user_uuid", "username", "email"}).
			AddRow(42, testUserUUID, "testuser", "test@test.com")
		mock.ExpectQuery(`SELECT .+ FROM "users"`).
			WillReturnRows(rows)
		for i := 0; i < 20; i++ {
			mock.ExpectQuery(`SELECT`).WillReturnRows(sqlmock.NewRows([]string{"id"}))
		}

		result, err := repo.FindBySubAndClientID("sub-1", "client-1")
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, "testuser", result.Username)
	})

	t.Run("not found", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		repo := NewUserRepository(db)

		mock.ExpectQuery(`SELECT .+ FROM "users"`).
			WillReturnRows(sqlmock.NewRows([]string{"user_id"}))

		result, err := repo.FindBySubAndClientID("unknown-sub", "unknown-client")
		require.NoError(t, err)
		assert.Nil(t, result)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("db error", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		repo := NewUserRepository(db)

		mock.ExpectQuery(`SELECT .+ FROM "users"`).
			WillReturnError(errors.New("db error"))

		_, err := repo.FindBySubAndClientID("sub-1", "client-1")
		require.Error(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestUserRepository_FindPaginated(t *testing.T) {
	t.Run("success with results", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		repo := NewUserRepository(db)

		mock.ExpectQuery(`SELECT count\(\*\) FROM "users"`).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(2))
		mock.ExpectQuery(`SELECT .+ FROM "users" WHERE "users"\."deleted_at" IS NULL .+ LIMIT \$[0-9]+`).
			WillReturnRows(sqlmock.NewRows([]string{"user_id", "user_uuid", "username"}).
				AddRow(1, testResourceUUID, "user1").
				AddRow(2, testUserUUID, "user2"))
		expectProfilePreloads(mock, int64(1), int64(2))

		result, err := repo.FindPaginated(UserRepositoryGetFilter{
			Page: 1, Limit: 10, SortBy: "created_at", SortOrder: SortOrderDesc,
		})
		require.NoError(t, err)
		assert.Equal(t, int64(2), result.Total)
		assert.Len(t, result.Data, 2)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("empty results", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		repo := NewUserRepository(db)

		mock.ExpectQuery(`SELECT count\(\*\) FROM "users"`).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))
		mock.ExpectQuery(`SELECT .+ FROM "users"`).
			WillReturnRows(sqlmock.NewRows([]string{"user_id"}))

		result, err := repo.FindPaginated(UserRepositoryGetFilter{
			Page: 1, Limit: 10, SortBy: "created_at", SortOrder: SortOrderDesc,
		})
		require.NoError(t, err)
		assert.Equal(t, int64(0), result.Total)
		assert.Len(t, result.Data, 0)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("with tenant filter", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		repo := NewUserRepository(db)
		tid := int64(1)

		mock.ExpectQuery(`SELECT count\(\*\) FROM "users"`).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
		mock.ExpectQuery(`SELECT .+ FROM "users"`).
			WillReturnRows(sqlmock.NewRows([]string{"user_id", "user_uuid", "username"}).
				AddRow(1, testResourceUUID, "user1"))
		expectProfilePreloads(mock, int64(1))

		result, err := repo.FindPaginated(UserRepositoryGetFilter{
			TenantID: &tid, Page: 1, Limit: 10, SortBy: "created_at", SortOrder: SortOrderDesc,
		})
		require.NoError(t, err)
		assert.Equal(t, int64(1), result.Total)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("count error", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		repo := NewUserRepository(db)

		mock.ExpectQuery(`SELECT count\(\*\) FROM "users"`).
			WillReturnError(errors.New("db error"))

		_, err := repo.FindPaginated(UserRepositoryGetFilter{
			Page: 1, Limit: 10, SortBy: "created_at", SortOrder: SortOrderDesc,
		})
		require.Error(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("with client filter", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		repo := NewUserRepository(db)
		cid := int64(5)

		mock.ExpectQuery(`SELECT count\(\*\) FROM "users"`).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
		mock.ExpectQuery(`SELECT .+ FROM "users"`).
			WillReturnRows(sqlmock.NewRows([]string{"user_id", "user_uuid", "username"}).
				AddRow(1, testResourceUUID, "user1"))
		expectProfilePreloads(mock, int64(1))

		result, err := repo.FindPaginated(UserRepositoryGetFilter{
			ClientID: &cid, Page: 1, Limit: 10, SortBy: "created_at", SortOrder: SortOrderDesc,
		})
		require.NoError(t, err)
		assert.Equal(t, int64(1), result.Total)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("with status filter", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		repo := NewUserRepository(db)

		mock.ExpectQuery(`SELECT count\(\*\) FROM "users" WHERE users.status IN \(\$1\) AND "users"\."deleted_at" IS NULL`).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
		mock.ExpectQuery(`SELECT .+ FROM "users" WHERE users.status IN \(\$1\) AND "users"\."deleted_at" IS NULL .+ LIMIT \$[0-9]+`).
			WillReturnRows(sqlmock.NewRows([]string{"user_id", "user_uuid", "username"}).
				AddRow(1, testResourceUUID, "user1"))
		expectProfilePreloads(mock, int64(1))

		result, err := repo.FindPaginated(UserRepositoryGetFilter{
			Status: []string{"active"}, Page: 1, Limit: 10, SortBy: "created_at", SortOrder: SortOrderDesc,
		})
		require.NoError(t, err)
		assert.Equal(t, int64(1), result.Total)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("with role filter", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		repo := NewUserRepository(db)
		rid := int64(10)

		mock.ExpectQuery(`SELECT count\(\*\) FROM "users"`).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
		mock.ExpectQuery(`SELECT .+ FROM "users"`).
			WillReturnRows(sqlmock.NewRows([]string{"user_id", "user_uuid", "username"}).
				AddRow(1, testResourceUUID, "user1"))
		expectProfilePreloads(mock, int64(1))

		result, err := repo.FindPaginated(UserRepositoryGetFilter{
			RoleID: &rid, Page: 1, Limit: 10, SortBy: "created_at", SortOrder: SortOrderDesc,
		})
		require.NoError(t, err)
		assert.Equal(t, int64(1), result.Total)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestUserRepository_SetEmailVerified(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		repo := NewUserRepository(db)
		userUUID := uuid.New()

		mock.ExpectBegin()
		mock.ExpectExec(`UPDATE "users" SET "is_email_verified"=\$1,"updated_at"=\$2 WHERE user_uuid = \$3 AND "users"\."deleted_at" IS NULL`).
			WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectCommit()

		err := repo.SetEmailVerified(userUUID, true)
		require.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func expectProfilePreloads(mock sqlmock.Sqlmock, userIDs ...int64) {
	switch len(userIDs) {
	case 0:
		return
	case 1:
		mock.ExpectQuery(`SELECT \* FROM "profiles" WHERE`).
			WithArgs(userIDs[0], true).
			WillReturnRows(sqlmock.NewRows([]string{"profile_id", "user_id", "is_default", "display_name"}).
				AddRow(1, userIDs[0], true, "Test User"))
	default:
		args := make([]driver.Value, len(userIDs)+1)
		for i, uid := range userIDs {
			args[i] = uid
		}
		args[len(userIDs)] = true
		mock.ExpectQuery(`SELECT \* FROM "profiles" WHERE`).
			WithArgs(args...).
			WillReturnRows(sqlmock.NewRows([]string{"profile_id", "user_id", "is_default", "display_name"}).
				AddRow(1, userIDs[0], true, "Test User").
				AddRow(2, userIDs[1], true, "Test User 2"))
	}
}

func TestUserRepository_SetStatus(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		repo := NewUserRepository(db)
		userUUID := uuid.New()

		mock.ExpectBegin()
		mock.ExpectExec(`UPDATE "users" SET "status"=\$1,"updated_at"=\$2 WHERE user_uuid = \$3 AND "users"\."deleted_at" IS NULL`).
			WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectCommit()

		err := repo.SetStatus(userUUID, shared.StatusActive)
		require.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestUserRepository_SetForcePasswordChange(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		repo := NewUserRepository(db)
		userUUID := uuid.New()

		mock.ExpectBegin()
		mock.ExpectExec(`UPDATE "users" SET "force_password_change"=\$1,"updated_at"=\$2 WHERE user_uuid = \$3 AND "users"\."deleted_at" IS NULL`).
			WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectCommit()

		err := repo.SetForcePasswordChange(userUUID, true)
		require.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestUserRepository_SetPendingEmail(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		repo := NewUserRepository(db)
		userUUID := uuid.New()
		expires := time.Now().Add(24 * time.Hour)

		mock.ExpectBegin()
		mock.ExpectExec(`UPDATE "users" SET .+ WHERE user_uuid = \$[0-9]+ AND "users"\."deleted_at" IS NULL`).
			WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectCommit()

		err := repo.SetPendingEmail(userUUID, "new@test.com", "otp-123", expires)
		require.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestUserRepository_ClearEmailChange(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		repo := NewUserRepository(db)
		userUUID := uuid.New()

		mock.ExpectBegin()
		mock.ExpectExec(`UPDATE "users" SET .+ WHERE user_uuid = \$[0-9]+ AND "users"\."deleted_at" IS NULL`).
			WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectCommit()

		err := repo.ClearEmailChange(userUUID)
		require.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestUserRepository_UpdateEmail(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		repo := NewUserRepository(db)
		userUUID := uuid.New()

		mock.ExpectBegin()
		mock.ExpectExec(`UPDATE "users" SET "email"=\$1,"updated_at"=\$2 WHERE user_uuid = \$3 AND "users"\."deleted_at" IS NULL`).
			WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectCommit()

		err := repo.UpdateEmail(userUUID, "new@test.com")
		require.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestUserRepository_UpdateUsername(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		repo := NewUserRepository(db)
		userUUID := uuid.New()

		mock.ExpectBegin()
		mock.ExpectExec(`UPDATE "users" SET "username"=\$1,"updated_at"=\$2 WHERE user_uuid = \$3 AND "users"\."deleted_at" IS NULL`).
			WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectCommit()

		err := repo.UpdateUsername(userUUID, "newuser")
		require.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestUserRepository_FindByPendingEmailAndTenantID(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		repo := NewUserRepository(db)

		rows := sqlmock.NewRows([]string{"user_id", "user_uuid", "pending_email"}).
			AddRow(42, testUserUUID, "pending@test.com")
		mock.ExpectQuery(`SELECT .+ FROM "users"`).
			WithArgs("pending@test.com", int64(1), 1).
			WillReturnRows(rows)

		result, err := repo.FindByPendingEmailAndTenantID("pending@test.com", 1)
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, "pending@test.com", *result.PendingEmail)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("not found", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		repo := NewUserRepository(db)

		mock.ExpectQuery(`SELECT .+ FROM "users" WHERE`).
			WillReturnRows(sqlmock.NewRows([]string{"user_id"}))

		result, err := repo.FindByPendingEmailAndTenantID("missing@test.com", 1)
		require.NoError(t, err)
		assert.Nil(t, result)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("db error", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		repo := NewUserRepository(db)

		mock.ExpectQuery(`SELECT .+ FROM "users" WHERE`).
			WillReturnError(errors.New("db error"))

		_, err := repo.FindByPendingEmailAndTenantID("pending@test.com", 1)
		require.Error(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}
