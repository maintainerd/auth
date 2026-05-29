package tenant

import (
	"testing"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func newMockGormDB(t *testing.T) (*gorm.DB, sqlmock.Sqlmock) {
	t.Helper()
	sqlDB, mock, err := sqlmock.New()
	require.NoError(t, err)
	gormDB, err := gorm.Open(postgres.New(postgres.Config{Conn: sqlDB}), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	require.NoError(t, err)
	return gormDB, mock
}

// mockUserRepo satisfies the tenant.UserReader consumer interface.
type mockUserRepo struct {
	findByUUIDFn func(uuid.UUID) (*MemberUser, error)
	findByIDFn   func(int64) (*MemberUser, error)
}

func (m *mockUserRepo) FindByUUID(id uuid.UUID) (*MemberUser, error) {
	if m.findByUUIDFn != nil {
		return m.findByUUIDFn(id)
	}
	return nil, nil
}

func (m *mockUserRepo) FindByID(id int64) (*MemberUser, error) {
	if m.findByIDFn != nil {
		return m.findByIDFn(id)
	}
	return nil, nil
}

// validPagination returns a valid pagination request for filter-validation tests.
func validPagination() PaginationRequestDTO {
	return PaginationRequestDTO{Page: 1, Limit: 10}
}

// testCascadeModels returns a 30-element cascade model list (matching the
// production cascade length) so DeleteByUUID issues the expected number of
// cascade DELETEs in tests.
func testCascadeModels() []any {
	m := make([]any, 30)
	for i := range m {
		m[i] = &Tenant{}
	}
	return m
}
