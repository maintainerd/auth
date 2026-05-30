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

// stubActor is a test AccessActor backed by a fixed identity list. Shared by the
// tenant-access tests in service_tenant_test.go and the isolation tests.
type stubActor struct{ identities []AccessIdentity }

func (s stubActor) AccessIdentities() []AccessIdentity { return s.identities }

// buildUserWithIdentities creates an AccessActor with the provided identities.
func buildUserWithIdentities(identities []AccessIdentity) AccessActor {
	return stubActor{identities: identities}
}

// buildTenant creates a minimal tenant for access/isolation tests.
func buildTenant(id int64, isSystem bool) *Tenant {
	return &Tenant{
		TenantID: id,
		IsSystem: isSystem,
	}
}

// buildIdentity creates an AccessIdentity linked to the given tenant.
func buildIdentity(tenantID int64, isSystem bool) AccessIdentity {
	return AccessIdentity{
		TenantID:       tenantID,
		TenantIsSystem: isSystem,
	}
}
