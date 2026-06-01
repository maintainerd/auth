//go:build integration

package repository_test

import (
	"testing"
	"time"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type testOAuthToken struct {
	OAuthRefreshTokenID int64     `gorm:"column:oauth_refresh_token_id;primaryKey"`
	TokenHash           string    `gorm:"column:token_hash"`
	FamilyUUID          uuid.UUID `gorm:"column:family_uuid"`
	UserID              int64     `gorm:"column:user_id"`
	ClientID            int64     `gorm:"column:client_id"`
	IsRevoked           bool      `gorm:"column:is_revoked"`
	ExpiresAt           time.Time `gorm:"column:expires_at"`
	CreatedAt           time.Time `gorm:"column:created_at"`
}

func (testOAuthToken) TableName() string { return "oauth_refresh_tokens" }

func TestIntegration_OAuth_FindActiveToken(t *testing.T) {
	db, mock := newMockDB(t)
	familyUUID := uuid.New()
	now := time.Now()

	mock.ExpectQuery(`SELECT \* FROM "oauth_refresh_tokens" WHERE.*ORDER BY.*LIMIT`).
		WillReturnRows(sqlmock.NewRows([]string{"oauth_refresh_token_id", "token_hash", "family_uuid", "user_id", "client_id", "is_revoked", "expires_at", "created_at"}).
			AddRow(1, "hash-123", familyUUID, 42, 1, false, now.Add(time.Hour), now))

	var token testOAuthToken
	err := db.Where("token_hash = ? AND is_revoked = ?", "hash-123", false).First(&token).Error
	require.NoError(t, err)
	assert.Equal(t, int64(42), token.UserID)
	assert.False(t, token.IsRevoked)
}

func TestIntegration_OAuth_RevokeToken(t *testing.T) {
	db, mock := newMockDB(t)

	mock.ExpectBegin()
	mock.ExpectExec(`UPDATE "oauth_refresh_tokens"`).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	err := db.Model(&testOAuthToken{}).Where("oauth_refresh_token_id = ?", 1).
		Update("is_revoked", true).Error
	require.NoError(t, err)
}

func TestIntegration_OAuth_RevokeByFamily(t *testing.T) {
	db, mock := newMockDB(t)

	mock.ExpectBegin()
	mock.ExpectExec(`UPDATE "oauth_refresh_tokens"`).
		WillReturnResult(sqlmock.NewResult(3, 3))
	mock.ExpectCommit()

	result := db.Model(&testOAuthToken{}).Where("family_uuid = ?", uuid.New()).
		Update("is_revoked", true)
	require.NoError(t, result.Error)
	assert.Equal(t, int64(3), result.RowsAffected)
}

func TestIntegration_OAuth_CleanupExpired(t *testing.T) {
	db, mock := newMockDB(t)

	mock.ExpectBegin()
	mock.ExpectExec(`DELETE FROM "oauth_refresh_tokens"`).
		WillReturnResult(sqlmock.NewResult(5, 5))
	mock.ExpectCommit()

	result := db.Where("expires_at < ?", time.Now()).Delete(&testOAuthToken{})
	require.NoError(t, result.Error)
	assert.Equal(t, int64(5), result.RowsAffected)
}
