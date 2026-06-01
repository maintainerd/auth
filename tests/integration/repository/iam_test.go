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

type testRole struct {
	RoleID      int64     `gorm:"column:role_id;primaryKey"`
	RoleUUID    uuid.UUID `gorm:"column:role_uuid"`
	TenantID    int64     `gorm:"column:tenant_id"`
	Name        string    `gorm:"column:name"`
	Description string    `gorm:"column:description"`
	Status      string    `gorm:"column:status"`
	IsDefault   bool      `gorm:"column:is_default"`
	IsSystem    bool      `gorm:"column:is_system"`
	CreatedAt   time.Time `gorm:"column:created_at"`
	UpdatedAt   time.Time `gorm:"column:updated_at"`
}

func (testRole) TableName() string { return "roles" }

type testPermission struct {
	PermissionID   int64     `gorm:"column:permission_id;primaryKey"`
	PermissionUUID uuid.UUID `gorm:"column:permission_uuid"`
	Name           string    `gorm:"column:name"`
	Description    string    `gorm:"column:description"`
	CreatedAt      time.Time `gorm:"column:created_at"`
}

func (testPermission) TableName() string { return "permissions" }

func TestIntegration_IAM_FindRoleByNameAndTenant(t *testing.T) {
	db, mock := newMockDB(t)
	now := time.Now()

	mock.ExpectQuery(`SELECT \* FROM "roles" WHERE.*ORDER BY.*LIMIT`).
		WillReturnRows(sqlmock.NewRows([]string{"role_id", "role_uuid", "tenant_id", "name", "description", "status", "is_default", "is_system", "created_at", "updated_at"}).
			AddRow(1, uuid.New(), 1, "admin", "Administrator", "active", true, true, now, now))

	var role testRole
	err := db.Where("name = ? AND tenant_id = ?", "admin", 1).First(&role).Error
	require.NoError(t, err)
	assert.Equal(t, "admin", role.Name)
	assert.True(t, role.IsSystem)
}

func TestIntegration_IAM_CreateRole(t *testing.T) {
	db, mock := newMockDB(t)

	mock.ExpectBegin()
	mock.ExpectQuery(`INSERT INTO "roles"`).
		WillReturnRows(sqlmock.NewRows([]string{"role_id"}).AddRow(10))
	mock.ExpectCommit()

	role := &testRole{
		RoleUUID:    uuid.New(),
		TenantID:    1,
		Name:        "viewer",
		Description: "Viewer role",
		Status:      "active",
	}
	err := db.Create(role).Error
	require.NoError(t, err)
	assert.Equal(t, int64(10), role.RoleID)
}

func TestIntegration_IAM_FindPermissions(t *testing.T) {
	db, mock := newMockDB(t)
	now := time.Now()

	mock.ExpectQuery(`SELECT \* FROM "permissions"`).
		WillReturnRows(sqlmock.NewRows([]string{"permission_id", "permission_uuid", "name", "description", "created_at"}).
			AddRow(1, uuid.New(), "tenant:read", "Read tenant", now).
			AddRow(2, uuid.New(), "tenant:write", "Write tenant", now))

	var perms []testPermission
	err := db.Find(&perms).Error
	require.NoError(t, err)
	assert.Len(t, perms, 2)
	assert.Equal(t, "tenant:read", perms[0].Name)
}
