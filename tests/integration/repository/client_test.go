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

type testClient struct {
	ClientID           int64     `gorm:"column:client_id;primaryKey"`
	ClientUUID         uuid.UUID `gorm:"column:client_uuid"`
	TenantID           int64     `gorm:"column:tenant_id"`
	Name               string    `gorm:"column:name"`
	ClientType         string    `gorm:"column:client_type"`
	Status             string    `gorm:"column:status"`
	IsDefault          bool      `gorm:"column:is_default"`
	IsSystem           bool      `gorm:"column:is_system"`
	IdentityProviderID int64     `gorm:"column:identity_provider_id"`
	CreatedAt          time.Time `gorm:"column:created_at"`
	UpdatedAt          time.Time `gorm:"column:updated_at"`
}

func (testClient) TableName() string { return "clients" }

func TestIntegration_Client_FindByNameAndIdentityProvider(t *testing.T) {
	db, mock := newMockDB(t)
	now := time.Now()

	mock.ExpectQuery(`SELECT \* FROM "clients" WHERE.*ORDER BY.*LIMIT`).
		WillReturnRows(sqlmock.NewRows([]string{"client_id", "client_uuid", "tenant_id", "name", "client_type", "status", "is_default", "is_system", "identity_provider_id", "created_at", "updated_at"}).
			AddRow(1, uuid.New(), 1, "my-app", "spa", "active", false, false, 1, now, now))

	var client testClient
	err := db.Where("name = ? AND identity_provider_id = ?", "my-app", 1).First(&client).Error
	require.NoError(t, err)
	assert.Equal(t, "my-app", client.Name)
	assert.Equal(t, "active", client.Status)
}

func TestIntegration_Client_Create(t *testing.T) {
	db, mock := newMockDB(t)

	mock.ExpectBegin()
	mock.ExpectQuery(`INSERT INTO "clients"`).
		WillReturnRows(sqlmock.NewRows([]string{"client_id"}).AddRow(5))
	mock.ExpectCommit()

	client := &testClient{
		ClientUUID:         uuid.New(),
		TenantID:           1,
		Name:               "new-app",
		ClientType:         "spa",
		Status:             "active",
		IdentityProviderID: 1,
	}
	err := db.Create(client).Error
	require.NoError(t, err)
	assert.Equal(t, int64(5), client.ClientID)
}

func TestIntegration_Client_FindSystem(t *testing.T) {
	db, mock := newMockDB(t)
	now := time.Now()

	mock.ExpectQuery(`SELECT \* FROM "clients" WHERE.*ORDER BY.*LIMIT`).
		WillReturnRows(sqlmock.NewRows([]string{"client_id", "client_uuid", "tenant_id", "name", "client_type", "status", "is_default", "is_system", "identity_provider_id", "created_at", "updated_at"}).
			AddRow(1, uuid.New(), 1, "system", "server", "active", true, true, 1, now, now))

	var client testClient
	err := db.Where("is_system = ?", true).First(&client).Error
	require.NoError(t, err)
	assert.True(t, client.IsSystem)
}

func TestIntegration_Client_SetStatus(t *testing.T) {
	db, mock := newMockDB(t)
	clientUUID := uuid.New()

	mock.ExpectBegin()
	mock.ExpectExec(`UPDATE "clients"`).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	err := db.Model(&testClient{}).Where("client_uuid = ?", clientUUID).
		Update("status", "inactive").Error
	require.NoError(t, err)
}

func TestIntegration_Client_Delete(t *testing.T) {
	db, mock := newMockDB(t)
	clientUUID := uuid.New()

	mock.ExpectBegin()
	mock.ExpectExec(`DELETE FROM "clients"`).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	result := db.Where("client_uuid = ?", clientUUID).Delete(&testClient{})
	require.NoError(t, result.Error)
}
