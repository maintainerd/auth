package app

import (
	"context"
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/maintainerd/maintainerd-auth/internal/shared"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// The client_credentials permission resolver is the only thing that decides what
// a machine token may do, and client_roles / client_permissions carry no
// deleted_at or status of their own. So if the resolver does not filter the
// roles and permissions it joins through, deactivating or soft-deleting a role
// or a permission does not revoke anything: the grant keeps landing in every
// token the client mints, forever.
func TestClientPermissionResolver_FiltersDeletedAndInactiveRolesAndPermissions(t *testing.T) {
	var executed []string

	sqlDB, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherFunc(
		func(_, actualSQL string) error {
			executed = append(executed, actualSQL)
			return nil
		})))
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer func() { _ = sqlDB.Close() }()

	db, err := gorm.Open(postgres.New(postgres.Config{Conn: sqlDB}), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("gorm.Open: %v", err)
	}

	// Direct grants: client_permissions → permissions, scoped to the client's APIs.
	mock.ExpectQuery("").
		WithArgs(int64(3), shared.StatusActive).
		WillReturnRows(sqlmock.NewRows([]string{"name"}).AddRow("api:read"))

	// Role-inherited grants: client_roles → roles → role_permissions → permissions.
	mock.ExpectQuery("").
		WithArgs(int64(3), shared.StatusActive, shared.StatusActive).
		WillReturnRows(sqlmock.NewRows([]string{"name"}).AddRow("api:write"))

	names, err := newClientPermissionResolver(db).ResolvePermissions(context.Background(), 3)
	if err != nil {
		t.Fatalf("ResolvePermissions error: %v", err)
	}
	if len(names) != 2 || names[0] != "api:read" || names[1] != "api:write" {
		t.Fatalf("names = %#v, want [api:read api:write]", names)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sql expectations: %v", err)
	}
	if len(executed) != 2 {
		t.Fatalf("executed %d queries, want 2: %#v", len(executed), executed)
	}

	direct := executed[0]
	for _, want := range []string{"p.deleted_at IS NULL", "p.status ="} {
		if !strings.Contains(direct, want) {
			t.Errorf("direct-permission query is missing %q:\n%s", want, direct)
		}
	}

	// The roles join is the load-bearing part: role_id is already on client_roles,
	// so without joining roles there is nothing to filter deleted_at/status on.
	inherited := executed[1]
	for _, want := range []string{
		"JOIN roles",
		"r.deleted_at IS NULL",
		"r.status =",
		"p.deleted_at IS NULL",
		"p.status =",
	} {
		if !strings.Contains(inherited, want) {
			t.Errorf("role-inherited query is missing %q:\n%s", want, inherited)
		}
	}
}
