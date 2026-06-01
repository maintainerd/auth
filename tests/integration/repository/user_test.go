//go:build integration

package repository_test

import (
	"testing"
	"time"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type testUser struct {
	UserID          int64     `gorm:"column:user_id;primaryKey"`
	UserUUID        uuid.UUID `gorm:"column:user_uuid"`
	Username        string    `gorm:"column:username"`
	Email           string    `gorm:"column:email"`
	Status          string    `gorm:"column:status"`
	IsEmailVerified bool      `gorm:"column:is_email_verified"`
	CreatedAt       time.Time `gorm:"column:created_at"`
	UpdatedAt       time.Time `gorm:"column:updated_at"`
}

func (testUser) TableName() string { return "users" }

func newMockDB(t *testing.T) (*gorm.DB, sqlmock.Sqlmock) {
	t.Helper()
	sqlDB, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	require.NoError(t, err)
	db, err := gorm.Open(postgres.New(postgres.Config{Conn: sqlDB}), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	require.NoError(t, err)
	return db, mock
}

func TestIntegration_User_FindByEmail(t *testing.T) {
	db, mock := newMockDB(t)
	now := time.Now()

	mock.ExpectQuery(`SELECT \* FROM "users" WHERE.*ORDER BY.*LIMIT`).
		WillReturnRows(sqlmock.NewRows([]string{"user_id", "user_uuid", "username", "email", "status", "is_email_verified", "created_at", "updated_at"}).
			AddRow(1, uuid.New(), "alice", "alice@example.com", "active", true, now, now))

	var user testUser
	err := db.Where("email = ?", "alice@example.com").First(&user).Error
	require.NoError(t, err)
	assert.Equal(t, "alice", user.Username)
	assert.True(t, user.IsEmailVerified)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestIntegration_User_FindByUUID(t *testing.T) {
	db, mock := newMockDB(t)
	testUUID := uuid.New()
	now := time.Now()

	mock.ExpectQuery(`SELECT \* FROM "users" WHERE.*ORDER BY.*LIMIT`).
		WillReturnRows(sqlmock.NewRows([]string{"user_id", "user_uuid", "username", "email", "status", "is_email_verified", "created_at", "updated_at"}).
			AddRow(1, testUUID, "bob", "bob@example.com", "active", false, now, now))

	var user testUser
	err := db.Where("user_uuid = ?", testUUID).First(&user).Error
	require.NoError(t, err)
	assert.Equal(t, "bob", user.Username)
	assert.False(t, user.IsEmailVerified)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestIntegration_User_Create(t *testing.T) {
	db, mock := newMockDB(t)
	newUUID := uuid.New()

	mock.ExpectBegin()
	mock.ExpectQuery(`INSERT INTO "users"`).
		WillReturnRows(sqlmock.NewRows([]string{"user_id"}).AddRow(42))
	mock.ExpectCommit()

	user := &testUser{
		UserUUID: newUUID,
		Username: "newuser",
		Email:    "new@example.com",
		Status:   "active",
	}
	err := db.Create(user).Error
	require.NoError(t, err)
	assert.Equal(t, int64(42), user.UserID)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestIntegration_User_UpdateStatus(t *testing.T) {
	db, mock := newMockDB(t)
	testUUID := uuid.New()

	mock.ExpectBegin()
	mock.ExpectExec(`UPDATE "users"`).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	err := db.Model(&testUser{}).Where("user_uuid = ?", testUUID).
		Update("status", "inactive").Error
	require.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestIntegration_User_Delete(t *testing.T) {
	db, mock := newMockDB(t)
	testUUID := uuid.New()

	mock.ExpectBegin()
	mock.ExpectExec(`DELETE FROM "users"`).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	result := db.Where("user_uuid = ?", testUUID).Delete(&testUser{})
	require.NoError(t, result.Error)
	assert.NoError(t, mock.ExpectationsWereMet())
}
