package idp

import (
	"context"
	"errors"
	"testing"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/maintainerd/maintainerd-auth/internal/shared"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

func buildRegistrationFlow() *RegistrationFlow {
	return &RegistrationFlow{
		RegistrationFlowID:   77,
		RegistrationFlowUUID: uuid.New(),
		TenantID:             tenantID,
		Name:                 "partner-signup",
		Description:          "desc",
		Status:               shared.StatusActive,
		ClientID:             1,
		Client:               &Client{ClientUUID: uuid.New()},
	}
}

func defaultCR() *mockClientRepo {
	return &mockClientRepo{
		findSystemFn:                        func() (*Client, error) { return nil, nil },
		findByClientIDAndIdentityProviderFn: func(_, _ string) (*Client, error) { return nil, nil },
	}
}

// testActor is the acting admin: a real user holding an identity in the target
// tenant, which is what loadActorForTenant demands of every mutation.
func testActor() *User {
	return &User{
		UserID:         testActorUserID,
		UserUUID:       testUserUUID,
		TenantID:       tenantID,
		UserIdentities: []UserIdentity{{TenantID: tenantID}},
	}
}

// rfSvcMocks bundles the eight dependencies of NewRegistrationFlowService,
// pre-wired for the happy path so each sub-test only overrides what it exercises.
type rfSvcMocks struct {
	flowRepo       *mockRegistrationFlowRepo
	flowRoleRepo   *mockRegistrationFlowRoleRepo
	roleRepo       *mockRoleRepo
	clientRepo     *mockClientRepo
	userRepo       *mockUserRepo
	userRoleRepo   *mockUserRoleRepo
	inviteCounter  *mockRegistrationFlowInviteCounter
	permNameReader *mockRolePermissionNameReader
}

func newRFSvcMocks() *rfSvcMocks {
	return &rfSvcMocks{
		flowRepo:     &mockRegistrationFlowRepo{},
		flowRoleRepo: &mockRegistrationFlowRoleRepo{},
		roleRepo:     &mockRoleRepo{},
		clientRepo:   defaultCR(),
		userRepo: &mockUserRepo{
			findByUUIDFn: func(_ any, _ ...string) (*User, error) { return testActor(), nil },
		},
		userRoleRepo:   &mockUserRoleRepo{},
		inviteCounter:  &mockRegistrationFlowInviteCounter{},
		permNameReader: &mockRolePermissionNameReader{},
	}
}

func (m *rfSvcMocks) svc(db *gorm.DB) RegistrationFlowService {
	return NewRegistrationFlowService(db, m.flowRepo, m.flowRoleRepo, m.roleRepo,
		m.clientRepo, m.userRepo, m.userRoleRepo, m.inviteCounter, m.permNameReader)
}

// grantsRole makes the acting user a holder of every role, satisfying the
// "you cannot grant roles you do not possess" cap.
func (m *rfSvcMocks) grantsRole() {
	m.userRoleRepo.findByUserIDAndRoleIDFn = func(userID, roleID int64) (*UserRole, error) {
		return &UserRole{UserRoleID: 1, UserID: userID, RoleID: roleID}, nil
	}
}

// expectTenantSelect mocks the tenant read inside loadActorForTenant. Every
// mutation performs it, so it sits between ExpectBegin and the outcome.
func expectTenantSelect(mock sqlmock.Sqlmock, tid int64) {
	mock.ExpectQuery(`SELECT \* FROM "tenants" WHERE tenant_id = \$1`).
		WithArgs(tid, 1).
		WillReturnRows(sqlmock.NewRows([]string{"tenant_id", "tenant_uuid", "name", "status"}).
			AddRow(tid, testTenantUUID, "acme", shared.StatusActive))
}

func expectTenantSelectError(mock sqlmock.Sqlmock, tid int64) {
	mock.ExpectQuery(`SELECT \* FROM "tenants" WHERE tenant_id = \$1`).
		WithArgs(tid, 1).
		WillReturnError(errors.New("no rows"))
}

func TestRequiredFieldsToJSON(t *testing.T) {
	t.Run("nil becomes an empty array", func(t *testing.T) {
		assert.JSONEq(t, `[]`, string(requiredFieldsToJSON(nil)))
	})

	t.Run("empty slice becomes an empty array", func(t *testing.T) {
		fields := []string{}
		assert.JSONEq(t, `[]`, string(requiredFieldsToJSON(&fields)))
	})

	t.Run("values are lowercased and trimmed", func(t *testing.T) {
		fields := []string{"  Email ", "FULLNAME"}
		assert.JSONEq(t, `["email","fullname"]`, string(requiredFieldsToJSON(&fields)))
	})
}

func TestToRegistrationFlowServiceDataResult(t *testing.T) {
	t.Run("nil input returns nil", func(t *testing.T) {
		assert.Nil(t, toRegistrationFlowServiceDataResult(nil))
	})

	t.Run("empty required_fields defaults to an empty array", func(t *testing.T) {
		res := toRegistrationFlowServiceDataResult(&RegistrationFlow{RegistrationFlowUUID: uuid.New()})
		require.NotNil(t, res)
		assert.JSONEq(t, `[]`, string(res.RequiredFields))
	})

	t.Run("without a client the client uuid stays nil", func(t *testing.T) {
		res := toRegistrationFlowServiceDataResult(&RegistrationFlow{RegistrationFlowUUID: uuid.New()})
		require.NotNil(t, res)
		assert.Nil(t, res.ClientUUID)
		assert.Empty(t, res.ClientName)
	})

	t.Run("with a client every client field is projected", func(t *testing.T) {
		clientUUID := uuid.New()
		identifier := "partner-portal-client"
		res := toRegistrationFlowServiceDataResult(&RegistrationFlow{
			RegistrationFlowUUID: uuid.New(),
			IsSystem:             true,
			Client: &Client{
				ClientUUID:  clientUUID,
				Name:        "partner-portal",
				DisplayName: "Partner Portal",
				Identifier:  &identifier,
				Status:      shared.StatusActive,
			},
		})
		require.NotNil(t, res)
		require.NotNil(t, res.ClientUUID)
		assert.Equal(t, clientUUID, *res.ClientUUID)
		assert.Equal(t, "partner-portal", res.ClientName)
		assert.Equal(t, "Partner Portal", res.ClientDisplayName)
		assert.Equal(t, identifier, res.ClientIdentifier)
		assert.Equal(t, shared.StatusActive, res.ClientStatus)
		assert.True(t, res.IsSystem)
	})

	t.Run("client with a nil identifier leaves the identifier empty", func(t *testing.T) {
		res := toRegistrationFlowServiceDataResult(&RegistrationFlow{
			RegistrationFlowUUID: uuid.New(),
			Client:               &Client{ClientUUID: uuid.New(), Identifier: nil},
		})
		require.NotNil(t, res)
		assert.Empty(t, res.ClientIdentifier)
	})
}

// ---------------------------------------------------------------------------
// Get (list)
// ---------------------------------------------------------------------------

func TestRegistrationFlowService_Get(t *testing.T) {
	flow := buildRegistrationFlow()

	t.Run("success without a client filter", func(t *testing.T) {
		db, _ := newMockGormDB(t)
		m := newRFSvcMocks()
		m.flowRepo.findPaginatedFn = func(f RegistrationFlowRepositoryGetFilter) (*PaginationResult[RegistrationFlow], error) {
			require.NotNil(t, f.TenantID)
			assert.Equal(t, tenantID, *f.TenantID)
			assert.Nil(t, f.ClientID)
			return &PaginationResult[RegistrationFlow]{
				Data: []RegistrationFlow{*flow}, Total: 1, Page: 1, Limit: 10, TotalPages: 1,
			}, nil
		}
		res, err := m.svc(db).Get(context.Background(), RegistrationFlowServiceGetFilter{TenantID: tenantID, Page: 1, Limit: 10})
		require.NoError(t, err)
		assert.Equal(t, int64(1), res.Total)
		require.Len(t, res.Data, 1)
		assert.Equal(t, flow.Name, res.Data[0].Name)
		assert.Equal(t, flow.Name, res.Data[0].Name)
	})

	t.Run("every filter is forwarded to the repository", func(t *testing.T) {
		db, _ := newMockGormDB(t)
		clientUUID := uuid.New()
		name, search := "partner", "part"
		isSystem := true
		m := newRFSvcMocks()
		m.clientRepo.findByUUIDFn = func(_ any, _ ...string) (*Client, error) {
			return &Client{ClientID: 5, TenantID: tenantID, Status: shared.StatusActive}, nil
		}
		m.flowRepo.findPaginatedFn = func(f RegistrationFlowRepositoryGetFilter) (*PaginationResult[RegistrationFlow], error) {
			assert.Equal(t, &name, f.Name)
			assert.Equal(t, &search, f.Search)
			assert.Equal(t, []string{shared.StatusActive}, f.Status)
			require.NotNil(t, f.IsSystem)
			assert.True(t, *f.IsSystem)
			require.NotNil(t, f.ClientID)
			assert.Equal(t, int64(5), *f.ClientID)
			assert.Equal(t, "name", f.SortBy)
			assert.Equal(t, "asc", f.SortOrder)
			return &PaginationResult[RegistrationFlow]{}, nil
		}
		_, err := m.svc(db).Get(context.Background(), RegistrationFlowServiceGetFilter{
			TenantID:   tenantID,
			Name:       &name,
			Search:     &search,
			Status:     []string{shared.StatusActive},
			IsSystem:   &isSystem,
			ClientUUID: &clientUUID,
			Page:       1, Limit: 10, SortBy: "name", SortOrder: "asc",
		})
		require.NoError(t, err)
	})

	t.Run("client filter not found", func(t *testing.T) {
		db, _ := newMockGormDB(t)
		clientUUID := uuid.New()
		m := newRFSvcMocks()
		m.clientRepo.findByUUIDFn = func(_ any, _ ...string) (*Client, error) { return nil, nil }
		_, err := m.svc(db).Get(context.Background(), RegistrationFlowServiceGetFilter{TenantID: tenantID, ClientUUID: &clientUUID})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "auth client not found")
	})

	t.Run("client filter repo error", func(t *testing.T) {
		db, _ := newMockGormDB(t)
		clientUUID := uuid.New()
		m := newRFSvcMocks()
		m.clientRepo.findByUUIDFn = func(_ any, _ ...string) (*Client, error) { return nil, errors.New("db") }
		_, err := m.svc(db).Get(context.Background(), RegistrationFlowServiceGetFilter{TenantID: tenantID, ClientUUID: &clientUUID})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "auth client not found")
	})

	t.Run("client from another tenant is forbidden", func(t *testing.T) {
		db, _ := newMockGormDB(t)
		clientUUID := uuid.New()
		m := newRFSvcMocks()
		m.clientRepo.findByUUIDFn = func(_ any, _ ...string) (*Client, error) {
			return &Client{ClientID: 5, TenantID: 999}, nil
		}
		_, err := m.svc(db).Get(context.Background(), RegistrationFlowServiceGetFilter{TenantID: tenantID, ClientUUID: &clientUUID})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "does not belong to your tenant")
	})

	t.Run("repository error is propagated", func(t *testing.T) {
		db, _ := newMockGormDB(t)
		m := newRFSvcMocks()
		m.flowRepo.findPaginatedFn = func(RegistrationFlowRepositoryGetFilter) (*PaginationResult[RegistrationFlow], error) {
			return nil, errors.New("paginate error")
		}
		_, err := m.svc(db).Get(context.Background(), RegistrationFlowServiceGetFilter{TenantID: tenantID})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "paginate error")
	})

	t.Run("empty result set", func(t *testing.T) {
		db, _ := newMockGormDB(t)
		m := newRFSvcMocks()
		res, err := m.svc(db).Get(context.Background(), RegistrationFlowServiceGetFilter{TenantID: tenantID})
		require.NoError(t, err)
		assert.Empty(t, res.Data)
		assert.Equal(t, int64(0), res.Total)
	})
}

// ---------------------------------------------------------------------------
// GetByUUID
// ---------------------------------------------------------------------------

func TestRegistrationFlowService_GetByUUID(t *testing.T) {
	flow := buildRegistrationFlow()

	t.Run("not found", func(t *testing.T) {
		db, _ := newMockGormDB(t)
		m := newRFSvcMocks()
		m.flowRepo.findByUUIDAndTenantIDFn = func(uuid.UUID, int64, ...string) (*RegistrationFlow, error) { return nil, nil }
		_, err := m.svc(db).GetByUUID(context.Background(), flow.RegistrationFlowUUID, tenantID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "registration flow not found or access denied")
	})

	t.Run("repo error is reported as not found (no existence oracle)", func(t *testing.T) {
		db, _ := newMockGormDB(t)
		m := newRFSvcMocks()
		m.flowRepo.findByUUIDAndTenantIDFn = func(uuid.UUID, int64, ...string) (*RegistrationFlow, error) {
			return nil, errors.New("db error")
		}
		_, err := m.svc(db).GetByUUID(context.Background(), flow.RegistrationFlowUUID, tenantID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "registration flow not found or access denied")
	})

	t.Run("success preloads the client and is tenant-scoped", func(t *testing.T) {
		db, _ := newMockGormDB(t)
		m := newRFSvcMocks()
		m.flowRepo.findByUUIDAndTenantIDFn = func(id uuid.UUID, tid int64, preloads ...string) (*RegistrationFlow, error) {
			assert.Equal(t, flow.RegistrationFlowUUID, id)
			assert.Equal(t, tenantID, tid)
			assert.Equal(t, []string{"Client"}, preloads)
			return flow, nil
		}
		res, err := m.svc(db).GetByUUID(context.Background(), flow.RegistrationFlowUUID, tenantID)
		require.NoError(t, err)
		assert.Equal(t, flow.Name, res.Name)
		assert.Equal(t, flow.Name, res.Name)
		require.NotNil(t, res.ClientUUID)
		assert.Equal(t, flow.Client.ClientUUID, *res.ClientUUID)
	})
}

// ---------------------------------------------------------------------------
// Create
// ---------------------------------------------------------------------------

func validCreateInput() RegistrationFlowCreateInput {
	return RegistrationFlowCreateInput{
		TenantID:      tenantID,
		ActorUserUUID: testUserUUID,
		Name:          "partner-signup",
		Description:   "desc",
		Status:        shared.StatusActive,
		ClientUUID:    uuid.New(),
	}
}

func activeClient() *Client {
	return &Client{ClientID: 1, TenantID: tenantID, Status: shared.StatusActive}
}

func TestRegistrationFlowService_Create(t *testing.T) {
	// 1. actor validation runs first — a request whose actor cannot be resolved
	// never reaches the client or name lookups.
	t.Run("actor not found", func(t *testing.T) {
		db, mock := newMockGormDBRegex(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		m := newRFSvcMocks()
		m.userRepo.findByUUIDFn = func(_ any, _ ...string) (*User, error) { return nil, nil }
		clientCalled := false
		m.clientRepo.findByUUIDFn = func(_ any, _ ...string) (*Client, error) {
			clientCalled = true
			return activeClient(), nil
		}
		_, err := m.svc(db).Create(context.Background(), validCreateInput())
		require.Error(t, err)
		assert.Contains(t, err.Error(), "actor user not found")
		assert.False(t, clientCalled, "must short-circuit before the client lookup")
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("actor lookup error", func(t *testing.T) {
		db, mock := newMockGormDBRegex(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		m := newRFSvcMocks()
		m.userRepo.findByUUIDFn = func(_ any, _ ...string) (*User, error) { return nil, errors.New("db error") }
		_, err := m.svc(db).Create(context.Background(), validCreateInput())
		require.Error(t, err)
		assert.Contains(t, err.Error(), "actor user not found")
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("actor is loaded with its tenant identities", func(t *testing.T) {
		db, mock := newMockGormDBRegex(t)
		mock.ExpectBegin()
		expectTenantSelect(mock, tenantID)
		mock.ExpectRollback()
		m := newRFSvcMocks()
		var gotUUID any
		var gotPreloads []string
		m.userRepo.findByUUIDFn = func(id any, p ...string) (*User, error) {
			gotUUID, gotPreloads = id, p
			return testActor(), nil
		}
		m.clientRepo.findByUUIDFn = func(_ any, _ ...string) (*Client, error) { return nil, nil }
		_, err := m.svc(db).Create(context.Background(), validCreateInput())
		require.Error(t, err)
		assert.Equal(t, testUserUUID, gotUUID)
		assert.Equal(t, []string{"UserIdentities.Tenant"}, gotPreloads)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("tenant not found", func(t *testing.T) {
		db, mock := newMockGormDBRegex(t)
		mock.ExpectBegin()
		expectTenantSelectError(mock, tenantID)
		mock.ExpectRollback()
		m := newRFSvcMocks()
		_, err := m.svc(db).Create(context.Background(), validCreateInput())
		require.Error(t, err)
		assert.Contains(t, err.Error(), "tenant not found")
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	// Defence in depth: the middleware-supplied tenant proves the request was
	// routed for a tenant, not that this user belongs to it.
	t.Run("actor without an identity in the target tenant is forbidden", func(t *testing.T) {
		db, mock := newMockGormDBRegex(t)
		mock.ExpectBegin()
		expectTenantSelect(mock, tenantID)
		mock.ExpectRollback()
		m := newRFSvcMocks()
		m.userRepo.findByUUIDFn = func(_ any, _ ...string) (*User, error) {
			return &User{UserID: testActorUserID, UserUUID: testUserUUID, UserIdentities: []UserIdentity{{TenantID: 999}}}, nil
		}
		_, err := m.svc(db).Create(context.Background(), validCreateInput())
		require.Error(t, err)
		assert.Contains(t, err.Error(), "tenant access denied")
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("actor with no identities is forbidden", func(t *testing.T) {
		db, mock := newMockGormDBRegex(t)
		mock.ExpectBegin()
		expectTenantSelect(mock, tenantID)
		mock.ExpectRollback()
		m := newRFSvcMocks()
		m.userRepo.findByUUIDFn = func(_ any, _ ...string) (*User, error) {
			return &User{UserID: testActorUserID, UserUUID: testUserUUID}, nil
		}
		_, err := m.svc(db).Create(context.Background(), validCreateInput())
		require.Error(t, err)
		assert.Contains(t, err.Error(), "no identities")
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("client not found", func(t *testing.T) {
		db, mock := newMockGormDBRegex(t)
		mock.ExpectBegin()
		expectTenantSelect(mock, tenantID)
		mock.ExpectRollback()
		m := newRFSvcMocks()
		m.clientRepo.findByUUIDFn = func(_ any, _ ...string) (*Client, error) { return nil, nil }
		_, err := m.svc(db).Create(context.Background(), validCreateInput())
		require.Error(t, err)
		assert.Contains(t, err.Error(), "auth client not found")
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("client from another tenant is forbidden", func(t *testing.T) {
		db, mock := newMockGormDBRegex(t)
		mock.ExpectBegin()
		expectTenantSelect(mock, tenantID)
		mock.ExpectRollback()
		m := newRFSvcMocks()
		m.clientRepo.findByUUIDFn = func(_ any, _ ...string) (*Client, error) {
			return &Client{ClientID: 1, TenantID: 999, Status: shared.StatusActive}, nil
		}
		_, err := m.svc(db).Create(context.Background(), validCreateInput())
		require.Error(t, err)
		assert.Contains(t, err.Error(), "does not belong to your tenant")
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("inactive client is rejected", func(t *testing.T) {
		db, mock := newMockGormDBRegex(t)
		mock.ExpectBegin()
		expectTenantSelect(mock, tenantID)
		mock.ExpectRollback()
		m := newRFSvcMocks()
		m.clientRepo.findByUUIDFn = func(_ any, _ ...string) (*Client, error) {
			return &Client{ClientID: 1, TenantID: tenantID, Status: shared.StatusInactive}, nil
		}
		_, err := m.svc(db).Create(context.Background(), validCreateInput())
		require.Error(t, err)
		assert.Contains(t, err.Error(), "auth client is inactive or deleted")
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("name lookup error", func(t *testing.T) {
		db, mock := newMockGormDBRegex(t)
		mock.ExpectBegin()
		expectTenantSelect(mock, tenantID)
		mock.ExpectRollback()
		m := newRFSvcMocks()
		m.clientRepo.findByUUIDFn = func(_ any, _ ...string) (*Client, error) { return activeClient(), nil }
		m.flowRepo.findByNameAndTenantIDFn = func(string, int64) (*RegistrationFlow, error) {
			return nil, errors.New("name err")
		}
		_, err := m.svc(db).Create(context.Background(), validCreateInput())
		require.Error(t, err)
		assert.Contains(t, err.Error(), "name err")
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("duplicate name is a conflict", func(t *testing.T) {
		db, mock := newMockGormDBRegex(t)
		mock.ExpectBegin()
		expectTenantSelect(mock, tenantID)
		mock.ExpectRollback()
		m := newRFSvcMocks()
		m.clientRepo.findByUUIDFn = func(_ any, _ ...string) (*Client, error) { return activeClient(), nil }
		m.flowRepo.findByNameAndTenantIDFn = func(name string, tid int64) (*RegistrationFlow, error) {
			assert.Equal(t, tenantID, tid, "the name check must be tenant-scoped")
			return &RegistrationFlow{RegistrationFlowID: 5, Name: name}, nil
		}
		_, err := m.svc(db).Create(context.Background(), validCreateInput())
		require.Error(t, err)
		assert.Contains(t, err.Error(), "name already exists")
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("name uniqueness pre-check error", func(t *testing.T) {
		db, mock := newMockGormDBRegex(t)
		mock.ExpectBegin()
		expectTenantSelect(mock, tenantID)
		mock.ExpectRollback()
		m := newRFSvcMocks()
		m.clientRepo.findByUUIDFn = func(_ any, _ ...string) (*Client, error) { return activeClient(), nil }
		m.flowRepo.findByNameAndTenantIDFn = func(string, int64) (*RegistrationFlow, error) {
			return nil, errors.New("name err")
		}
		_, err := m.svc(db).Create(context.Background(), validCreateInput())
		require.Error(t, err)
		assert.Contains(t, err.Error(), "name err")
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	// The pre-check is tenant-scoped so it matches uq_registration_flows_tenant_name.
	t.Run("name uniqueness pre-check is tenant-scoped", func(t *testing.T) {
		db, mock := newMockGormDBRegex(t)
		mock.ExpectBegin()
		expectTenantSelect(mock, tenantID)
		mock.ExpectCommit()
		m := newRFSvcMocks()
		m.clientRepo.findByUUIDFn = func(_ any, _ ...string) (*Client, error) { return activeClient(), nil }
		var gotName string
		var gotTenant int64
		m.flowRepo.findByNameAndTenantIDFn = func(name string, tid int64) (*RegistrationFlow, error) {
			gotName, gotTenant = name, tid
			return nil, nil
		}
		flow := buildRegistrationFlow()
		m.flowRepo.createFn = func(*RegistrationFlow) (*RegistrationFlow, error) { return flow, nil }
		m.flowRepo.findByUUIDAndTenantIDFn = func(uuid.UUID, int64, ...string) (*RegistrationFlow, error) { return flow, nil }
		_, err := m.svc(db).Create(context.Background(), validCreateInput())
		require.NoError(t, err)
		assert.Equal(t, "partner-signup", gotName)
		assert.Equal(t, tenantID, gotTenant)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("create repo error", func(t *testing.T) {
		db, mock := newMockGormDBRegex(t)
		mock.ExpectBegin()
		expectTenantSelect(mock, tenantID)
		mock.ExpectRollback()
		m := newRFSvcMocks()
		m.clientRepo.findByUUIDFn = func(_ any, _ ...string) (*Client, error) { return activeClient(), nil }
		m.flowRepo.createFn = func(*RegistrationFlow) (*RegistrationFlow, error) { return nil, errors.New("create err") }
		_, err := m.svc(db).Create(context.Background(), validCreateInput())
		require.Error(t, err)
		assert.Contains(t, err.Error(), "create err")
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("success builds the row from the input and the actor", func(t *testing.T) {
		db, mock := newMockGormDBRegex(t)
		mock.ExpectBegin()
		expectTenantSelect(mock, tenantID)
		mock.ExpectCommit()

		m := newRFSvcMocks()
		m.clientRepo.findByUUIDFn = func(_ any, _ ...string) (*Client, error) { return activeClient(), nil }
		var created *RegistrationFlow
		flow := buildRegistrationFlow()
		m.flowRepo.createFn = func(e *RegistrationFlow) (*RegistrationFlow, error) {
			created = e
			return flow, nil
		}
		m.flowRepo.findByUUIDAndTenantIDFn = func(uuid.UUID, int64, ...string) (*RegistrationFlow, error) { return flow, nil }

		in := validCreateInput()
		fields := []string{" Email ", "FULLNAME"}
		in.RequiredFields = &fields
		in.VerificationRequired = true
		res, err := m.svc(db).Create(context.Background(), in)
		require.NoError(t, err)
		require.NotNil(t, res)

		require.NotNil(t, created)
		assert.Equal(t, tenantID, created.TenantID)
		assert.Equal(t, "partner-signup", created.Name)
		assert.Equal(t, "desc", created.Description)
		// The name IS the public selector; it is persisted verbatim.
		assert.Equal(t, "partner-signup", created.Name)
		assert.Equal(t, shared.StatusActive, created.Status)
		assert.Equal(t, int64(1), created.ClientID)
		assert.True(t, created.VerificationRequired)
		assert.JSONEq(t, `["email","fullname"]`, string(created.RequiredFields))
		// A flow created through the API is never system-managed.
		assert.False(t, created.IsSystem)
		require.NotNil(t, created.CreatedBy)
		require.NotNil(t, created.UpdatedBy)
		assert.Equal(t, testActorUserID, *created.CreatedBy)
		assert.Equal(t, testActorUserID, *created.UpdatedBy)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("nil required_fields becomes an empty array", func(t *testing.T) {
		db, mock := newMockGormDBRegex(t)
		mock.ExpectBegin()
		expectTenantSelect(mock, tenantID)
		mock.ExpectCommit()
		m := newRFSvcMocks()
		m.clientRepo.findByUUIDFn = func(_ any, _ ...string) (*Client, error) { return activeClient(), nil }
		var created *RegistrationFlow
		flow := buildRegistrationFlow()
		m.flowRepo.createFn = func(e *RegistrationFlow) (*RegistrationFlow, error) { created = e; return flow, nil }
		m.flowRepo.findByUUIDAndTenantIDFn = func(uuid.UUID, int64, ...string) (*RegistrationFlow, error) { return flow, nil }
		_, err := m.svc(db).Create(context.Background(), validCreateInput())
		require.NoError(t, err)
		require.NotNil(t, created)
		assert.JSONEq(t, `[]`, string(created.RequiredFields))
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("no roles requested skips role sync entirely", func(t *testing.T) {
		db, mock := newMockGormDBRegex(t)
		mock.ExpectBegin()
		expectTenantSelect(mock, tenantID)
		mock.ExpectCommit()
		m := newRFSvcMocks()
		m.clientRepo.findByUUIDFn = func(_ any, _ ...string) (*Client, error) { return activeClient(), nil }
		flow := buildRegistrationFlow()
		m.flowRepo.createFn = func(*RegistrationFlow) (*RegistrationFlow, error) { return flow, nil }
		m.flowRepo.findByUUIDAndTenantIDFn = func(uuid.UUID, int64, ...string) (*RegistrationFlow, error) { return flow, nil }
		roleLookups := 0
		m.roleRepo.findByUUIDFn = func(_ any, _ ...string) (*Role, error) { roleLookups++; return nil, nil }
		_, err := m.svc(db).Create(context.Background(), validCreateInput())
		require.NoError(t, err)
		assert.Zero(t, roleLookups)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("role sync failure rolls the whole create back", func(t *testing.T) {
		db, mock := newMockGormDBRegex(t)
		mock.ExpectBegin()
		expectTenantSelect(mock, tenantID)
		mock.ExpectRollback()
		m := newRFSvcMocks()
		m.clientRepo.findByUUIDFn = func(_ any, _ ...string) (*Client, error) { return activeClient(), nil }
		flow := buildRegistrationFlow()
		m.flowRepo.createFn = func(*RegistrationFlow) (*RegistrationFlow, error) { return flow, nil }
		m.roleRepo.findByUUIDFn = func(_ any, _ ...string) (*Role, error) { return nil, nil } // role not found

		in := validCreateInput()
		in.RoleUUIDs = []uuid.UUID{uuid.New()}
		_, err := m.svc(db).Create(context.Background(), in)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "role not found")
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("success with roles attaches the membership", func(t *testing.T) {
		db, mock := newMockGormDBRegex(t)
		mock.ExpectBegin()
		expectTenantSelect(mock, tenantID)
		mock.ExpectCommit()
		m := newRFSvcMocks()
		m.grantsRole()
		m.clientRepo.findByUUIDFn = func(_ any, _ ...string) (*Client, error) { return activeClient(), nil }
		flow := buildRegistrationFlow()
		m.flowRepo.createFn = func(*RegistrationFlow) (*RegistrationFlow, error) { return flow, nil }
		m.flowRepo.findByUUIDAndTenantIDFn = func(uuid.UUID, int64, ...string) (*RegistrationFlow, error) { return flow, nil }
		roleUUID := uuid.New()
		m.roleRepo.findByUUIDFn = func(_ any, _ ...string) (*Role, error) {
			return &Role{RoleID: 10, RoleUUID: roleUUID, TenantID: tenantID, Name: "partner-user", Status: shared.StatusActive}, nil
		}
		var attached []int64
		m.flowRoleRepo.createFn = func(e *RegistrationFlowRole) (*RegistrationFlowRole, error) {
			attached = append(attached, e.RoleID)
			assert.Equal(t, flow.RegistrationFlowID, e.RegistrationFlowID)
			return e, nil
		}

		in := validCreateInput()
		in.RoleUUIDs = []uuid.UUID{roleUUID}
		_, err := m.svc(db).Create(context.Background(), in)
		require.NoError(t, err)
		assert.Equal(t, []int64{10}, attached)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

// ---------------------------------------------------------------------------
// Update
// ---------------------------------------------------------------------------

func strp(s string) *string { return &s }
func boolp(b bool) *bool    { return &b }

func validUpdateInput(flowUUID uuid.UUID) RegistrationFlowUpdateInput {
	return RegistrationFlowUpdateInput{
		RegistrationFlowUUID: flowUUID,
		TenantID:             tenantID,
		ActorUserUUID:        testUserUUID,
	}
}

func TestRegistrationFlowService_Update(t *testing.T) {
	t.Run("not found", func(t *testing.T) {
		db, mock := newMockGormDBRegex(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		m := newRFSvcMocks()
		m.flowRepo.findByUUIDAndTenantIDFn = func(uuid.UUID, int64, ...string) (*RegistrationFlow, error) { return nil, nil }
		_, err := m.svc(db).Update(context.Background(), validUpdateInput(uuid.New()))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "registration flow not found or access denied")
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("flow lookup error", func(t *testing.T) {
		db, mock := newMockGormDBRegex(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		m := newRFSvcMocks()
		m.flowRepo.findByUUIDAndTenantIDFn = func(uuid.UUID, int64, ...string) (*RegistrationFlow, error) {
			return nil, errors.New("db error")
		}
		_, err := m.svc(db).Update(context.Background(), validUpdateInput(uuid.New()))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "registration flow not found or access denied")
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	// Guard order is: ownership → actor → is_system. An actor with no tenant
	// access must not learn that a system flow exists.
	t.Run("actor is validated before the system-flow guard", func(t *testing.T) {
		db, mock := newMockGormDBRegex(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		m := newRFSvcMocks()
		flow := buildRegistrationFlow()
		flow.IsSystem = true
		m.flowRepo.findByUUIDAndTenantIDFn = func(uuid.UUID, int64, ...string) (*RegistrationFlow, error) { return flow, nil }
		m.userRepo.findByUUIDFn = func(_ any, _ ...string) (*User, error) { return nil, nil }
		_, err := m.svc(db).Update(context.Background(), validUpdateInput(flow.RegistrationFlowUUID))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "actor user not found")
		assert.NotContains(t, err.Error(), "system registration flow")
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("system flow cannot be updated", func(t *testing.T) {
		db, mock := newMockGormDBRegex(t)
		mock.ExpectBegin()
		expectTenantSelect(mock, tenantID)
		mock.ExpectRollback()
		m := newRFSvcMocks()
		flow := buildRegistrationFlow()
		flow.IsSystem = true
		m.flowRepo.findByUUIDAndTenantIDFn = func(uuid.UUID, int64, ...string) (*RegistrationFlow, error) { return flow, nil }
		saved := false
		m.flowRepo.createOrUpdateFn = func(e *RegistrationFlow) (*RegistrationFlow, error) { saved = true; return e, nil }

		in := validUpdateInput(flow.RegistrationFlowUUID)
		in.Name = strp("hijacked")
		_, err := m.svc(db).Update(context.Background(), in)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "system registration flow is not allowed to be updated")
		assert.False(t, saved, "the row must never be written")
		assert.Equal(t, "partner-signup", flow.Name, "the in-memory row must be untouched")
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("name conflict with another flow", func(t *testing.T) {
		db, mock := newMockGormDBRegex(t)
		mock.ExpectBegin()
		expectTenantSelect(mock, tenantID)
		mock.ExpectRollback()
		m := newRFSvcMocks()
		flow := buildRegistrationFlow()
		m.flowRepo.findByUUIDAndTenantIDFn = func(uuid.UUID, int64, ...string) (*RegistrationFlow, error) { return flow, nil }
		m.flowRepo.findByNameAndTenantIDFn = func(string, int64) (*RegistrationFlow, error) {
			return &RegistrationFlow{RegistrationFlowID: 999}, nil
		}
		in := validUpdateInput(flow.RegistrationFlowUUID)
		in.Name = strp("someone-elses-name")
		_, err := m.svc(db).Update(context.Background(), in)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "name already exists")
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("name lookup error", func(t *testing.T) {
		db, mock := newMockGormDBRegex(t)
		mock.ExpectBegin()
		expectTenantSelect(mock, tenantID)
		mock.ExpectRollback()
		m := newRFSvcMocks()
		flow := buildRegistrationFlow()
		m.flowRepo.findByUUIDAndTenantIDFn = func(uuid.UUID, int64, ...string) (*RegistrationFlow, error) { return flow, nil }
		m.flowRepo.findByNameAndTenantIDFn = func(string, int64) (*RegistrationFlow, error) { return nil, errors.New("name err") }
		in := validUpdateInput(flow.RegistrationFlowUUID)
		in.Name = strp("new-name")
		_, err := m.svc(db).Update(context.Background(), in)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "name err")
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("name matching the same flow is not a conflict", func(t *testing.T) {
		db, mock := newMockGormDBRegex(t)
		mock.ExpectBegin()
		expectTenantSelect(mock, tenantID)
		mock.ExpectCommit()
		m := newRFSvcMocks()
		flow := buildRegistrationFlow()
		m.flowRepo.findByUUIDAndTenantIDFn = func(uuid.UUID, int64, ...string) (*RegistrationFlow, error) { return flow, nil }
		m.flowRepo.findByNameAndTenantIDFn = func(string, int64) (*RegistrationFlow, error) {
			return &RegistrationFlow{RegistrationFlowID: flow.RegistrationFlowID}, nil
		}
		in := validUpdateInput(flow.RegistrationFlowUUID)
		in.Name = strp("new-name")
		res, err := m.svc(db).Update(context.Background(), in)
		require.NoError(t, err)
		assert.NotNil(t, res)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("unchanged name skips the conflict check", func(t *testing.T) {
		db, mock := newMockGormDBRegex(t)
		mock.ExpectBegin()
		expectTenantSelect(mock, tenantID)
		mock.ExpectCommit()
		m := newRFSvcMocks()
		flow := buildRegistrationFlow()
		m.flowRepo.findByUUIDAndTenantIDFn = func(uuid.UUID, int64, ...string) (*RegistrationFlow, error) { return flow, nil }
		nameLookups := 0
		m.flowRepo.findByNameAndTenantIDFn = func(string, int64) (*RegistrationFlow, error) { nameLookups++; return nil, nil }
		in := validUpdateInput(flow.RegistrationFlowUUID)
		in.Name = strp(flow.Name)
		_, err := m.svc(db).Update(context.Background(), in)
		require.NoError(t, err)
		assert.Zero(t, nameLookups)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	// The data-loss regression this change fixed: a partial update must not
	// silently re-activate a disabled flow, turn off verification, or wipe the
	// required-field set.
	t.Run("omitted fields are left unchanged", func(t *testing.T) {
		db, mock := newMockGormDBRegex(t)
		mock.ExpectBegin()
		expectTenantSelect(mock, tenantID)
		mock.ExpectCommit()
		m := newRFSvcMocks()
		flow := buildRegistrationFlow()
		flow.Status = shared.StatusInactive
		flow.VerificationRequired = true
		flow.Description = "original description"
		flow.RequiredFields = []byte(`["email"]`)
		m.flowRepo.findByUUIDAndTenantIDFn = func(uuid.UUID, int64, ...string) (*RegistrationFlow, error) { return flow, nil }
		var saved *RegistrationFlow
		m.flowRepo.createOrUpdateFn = func(e *RegistrationFlow) (*RegistrationFlow, error) { saved = e; return e, nil }
		syncCalls := 0
		m.flowRoleRepo.findByRegistrationFlowIDFn = func(int64) ([]RegistrationFlowRole, error) {
			syncCalls++
			return nil, nil
		}

		in := validUpdateInput(flow.RegistrationFlowUUID)
		in.Name = strp("renamed") // the only field present
		_, err := m.svc(db).Update(context.Background(), in)
		require.NoError(t, err)

		require.NotNil(t, saved)
		assert.Equal(t, "renamed", saved.Name)
		assert.Equal(t, shared.StatusInactive, saved.Status, "status must not be re-activated")
		assert.True(t, saved.VerificationRequired, "verification must not be downgraded")
		assert.Equal(t, "original description", saved.Description)
		assert.JSONEq(t, `["email"]`, string(saved.RequiredFields), "required_fields must not be wiped")
		assert.Zero(t, syncCalls, "nil role_ids must leave membership untouched")
		require.NotNil(t, saved.UpdatedBy)
		assert.Equal(t, testActorUserID, *saved.UpdatedBy)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("every present field is applied", func(t *testing.T) {
		db, mock := newMockGormDBRegex(t)
		mock.ExpectBegin()
		expectTenantSelect(mock, tenantID)
		mock.ExpectCommit()
		m := newRFSvcMocks()
		flow := buildRegistrationFlow()
		m.flowRepo.findByUUIDAndTenantIDFn = func(uuid.UUID, int64, ...string) (*RegistrationFlow, error) { return flow, nil }
		var saved *RegistrationFlow
		m.flowRepo.createOrUpdateFn = func(e *RegistrationFlow) (*RegistrationFlow, error) { saved = e; return e, nil }

		fields := []string{" Phone "}
		in := validUpdateInput(flow.RegistrationFlowUUID)
		in.Name = strp("renamed")
		in.Description = strp("new description")
		in.Status = strp(shared.StatusInactive)
		in.VerificationRequired = boolp(true)
		in.RequiredFields = &fields
		_, err := m.svc(db).Update(context.Background(), in)
		require.NoError(t, err)

		require.NotNil(t, saved)
		assert.Equal(t, "renamed", saved.Name)
		assert.Equal(t, "new description", saved.Description)
		assert.Equal(t, shared.StatusInactive, saved.Status)
		assert.True(t, saved.VerificationRequired)
		assert.JSONEq(t, `["phone"]`, string(saved.RequiredFields))
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("explicit false verification_required is applied", func(t *testing.T) {
		db, mock := newMockGormDBRegex(t)
		mock.ExpectBegin()
		expectTenantSelect(mock, tenantID)
		mock.ExpectCommit()
		m := newRFSvcMocks()
		flow := buildRegistrationFlow()
		flow.VerificationRequired = true
		m.flowRepo.findByUUIDAndTenantIDFn = func(uuid.UUID, int64, ...string) (*RegistrationFlow, error) { return flow, nil }
		var saved *RegistrationFlow
		m.flowRepo.createOrUpdateFn = func(e *RegistrationFlow) (*RegistrationFlow, error) { saved = e; return e, nil }

		in := validUpdateInput(flow.RegistrationFlowUUID)
		in.VerificationRequired = boolp(false)
		_, err := m.svc(db).Update(context.Background(), in)
		require.NoError(t, err)
		require.NotNil(t, saved)
		assert.False(t, saved.VerificationRequired)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("empty required_fields clears the set", func(t *testing.T) {
		db, mock := newMockGormDBRegex(t)
		mock.ExpectBegin()
		expectTenantSelect(mock, tenantID)
		mock.ExpectCommit()
		m := newRFSvcMocks()
		flow := buildRegistrationFlow()
		flow.RequiredFields = []byte(`["email"]`)
		m.flowRepo.findByUUIDAndTenantIDFn = func(uuid.UUID, int64, ...string) (*RegistrationFlow, error) { return flow, nil }
		var saved *RegistrationFlow
		m.flowRepo.createOrUpdateFn = func(e *RegistrationFlow) (*RegistrationFlow, error) { saved = e; return e, nil }

		fields := []string{}
		in := validUpdateInput(flow.RegistrationFlowUUID)
		in.RequiredFields = &fields
		_, err := m.svc(db).Update(context.Background(), in)
		require.NoError(t, err)
		require.NotNil(t, saved)
		assert.JSONEq(t, `[]`, string(saved.RequiredFields))
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("save error", func(t *testing.T) {
		db, mock := newMockGormDBRegex(t)
		mock.ExpectBegin()
		expectTenantSelect(mock, tenantID)
		mock.ExpectRollback()
		m := newRFSvcMocks()
		flow := buildRegistrationFlow()
		m.flowRepo.findByUUIDAndTenantIDFn = func(uuid.UUID, int64, ...string) (*RegistrationFlow, error) { return flow, nil }
		m.flowRepo.createOrUpdateFn = func(*RegistrationFlow) (*RegistrationFlow, error) { return nil, errors.New("update err") }
		_, err := m.svc(db).Update(context.Background(), validUpdateInput(flow.RegistrationFlowUUID))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "update err")
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	// role_ids: nil = untouched, non-nil (incl. empty) = replace.
	t.Run("empty non-nil role_ids removes all membership", func(t *testing.T) {
		db, mock := newMockGormDBRegex(t)
		mock.ExpectBegin()
		expectTenantSelect(mock, tenantID)
		mock.ExpectCommit()
		m := newRFSvcMocks()
		flow := buildRegistrationFlow()
		m.flowRepo.findByUUIDAndTenantIDFn = func(uuid.UUID, int64, ...string) (*RegistrationFlow, error) { return flow, nil }
		m.flowRoleRepo.findByRegistrationFlowIDFn = func(int64) ([]RegistrationFlowRole, error) {
			return []RegistrationFlowRole{{RoleID: 10}, {RoleID: 20}}, nil
		}
		var removed []int64
		m.flowRoleRepo.deleteByRegistrationFlowIDAndRoleIDFn = func(_, roleID int64) error {
			removed = append(removed, roleID)
			return nil
		}
		in := validUpdateInput(flow.RegistrationFlowUUID)
		in.RoleUUIDs = []uuid.UUID{}
		_, err := m.svc(db).Update(context.Background(), in)
		require.NoError(t, err)
		assert.Equal(t, []int64{10, 20}, removed)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("role_ids replaces membership (adds missing, removes extra, keeps shared)", func(t *testing.T) {
		db, mock := newMockGormDBRegex(t)
		mock.ExpectBegin()
		expectTenantSelect(mock, tenantID)
		mock.ExpectCommit()
		m := newRFSvcMocks()
		m.grantsRole()
		flow := buildRegistrationFlow()
		m.flowRepo.findByUUIDAndTenantIDFn = func(uuid.UUID, int64, ...string) (*RegistrationFlow, error) { return flow, nil }

		keepUUID, addUUID := uuid.New(), uuid.New()
		m.roleRepo.findByUUIDFn = func(id any, _ ...string) (*Role, error) {
			switch id {
			case keepUUID:
				return &Role{RoleID: 10, RoleUUID: keepUUID, TenantID: tenantID, Status: shared.StatusActive}, nil
			case addUUID:
				return &Role{RoleID: 30, RoleUUID: addUUID, TenantID: tenantID, Status: shared.StatusActive}, nil
			}
			return nil, nil
		}
		m.flowRoleRepo.findByRegistrationFlowIDFn = func(int64) ([]RegistrationFlowRole, error) {
			return []RegistrationFlowRole{{RoleID: 10}, {RoleID: 20}}, nil
		}
		var removed, added []int64
		m.flowRoleRepo.deleteByRegistrationFlowIDAndRoleIDFn = func(_, roleID int64) error {
			removed = append(removed, roleID)
			return nil
		}
		m.flowRoleRepo.createFn = func(e *RegistrationFlowRole) (*RegistrationFlowRole, error) {
			added = append(added, e.RoleID)
			return e, nil
		}

		in := validUpdateInput(flow.RegistrationFlowUUID)
		in.RoleUUIDs = []uuid.UUID{keepUUID, addUUID}
		_, err := m.svc(db).Update(context.Background(), in)
		require.NoError(t, err)
		assert.Equal(t, []int64{20}, removed, "only the role no longer desired is removed")
		assert.Equal(t, []int64{30}, added, "only the newly desired role is added")
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("membership lookup error rolls back", func(t *testing.T) {
		db, mock := newMockGormDBRegex(t)
		mock.ExpectBegin()
		expectTenantSelect(mock, tenantID)
		mock.ExpectRollback()
		m := newRFSvcMocks()
		m.grantsRole()
		flow := buildRegistrationFlow()
		m.flowRepo.findByUUIDAndTenantIDFn = func(uuid.UUID, int64, ...string) (*RegistrationFlow, error) { return flow, nil }
		roleUUID := uuid.New()
		m.roleRepo.findByUUIDFn = func(_ any, _ ...string) (*Role, error) {
			return &Role{RoleID: 10, RoleUUID: roleUUID, TenantID: tenantID, Status: shared.StatusActive}, nil
		}
		m.flowRoleRepo.findByRegistrationFlowIDFn = func(int64) ([]RegistrationFlowRole, error) {
			return nil, errors.New("membership err")
		}
		in := validUpdateInput(flow.RegistrationFlowUUID)
		in.RoleUUIDs = []uuid.UUID{roleUUID}
		_, err := m.svc(db).Update(context.Background(), in)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "membership err")
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

// ---------------------------------------------------------------------------
// the grantable-role cap (shared by Create/Update via syncRoles and AssignRoles)
// ---------------------------------------------------------------------------

func TestRegistrationFlowService_GrantableRoleCap(t *testing.T) {
	roleUUID := uuid.New()

	// runUpdateWithRole drives the cap through the update path.
	runUpdateWithRole := func(t *testing.T, configure func(*rfSvcMocks)) error {
		t.Helper()
		db, mock := newMockGormDBRegex(t)
		mock.ExpectBegin()
		expectTenantSelect(mock, tenantID)
		mock.ExpectRollback()
		m := newRFSvcMocks()
		flow := buildRegistrationFlow()
		m.flowRepo.findByUUIDAndTenantIDFn = func(uuid.UUID, int64, ...string) (*RegistrationFlow, error) { return flow, nil }
		configure(m)
		in := validUpdateInput(flow.RegistrationFlowUUID)
		in.RoleUUIDs = []uuid.UUID{roleUUID}
		_, err := m.svc(db).Update(context.Background(), in)
		assert.NoError(t, mock.ExpectationsWereMet())
		return err
	}

	// runUpdateWithRoleCommit is runUpdateWithRole for the cases that succeed, so
	// the sqlmock expectation is a commit rather than a rollback.
	runUpdateWithRoleCommit := func(t *testing.T, configure func(*rfSvcMocks)) error {
		t.Helper()
		db, mock := newMockGormDBRegex(t)
		mock.ExpectBegin()
		expectTenantSelect(mock, tenantID)
		mock.ExpectCommit()
		m := newRFSvcMocks()
		flow := buildRegistrationFlow()
		m.flowRepo.findByUUIDAndTenantIDFn = func(uuid.UUID, int64, ...string) (*RegistrationFlow, error) { return flow, nil }
		m.flowRepo.createOrUpdateFn = func(e *RegistrationFlow) (*RegistrationFlow, error) { return e, nil }
		configure(m)
		in := validUpdateInput(flow.RegistrationFlowUUID)
		in.RoleUUIDs = []uuid.UUID{roleUUID}
		_, err := m.svc(db).Update(context.Background(), in)
		assert.NoError(t, mock.ExpectationsWereMet())
		return err
	}

	// The flow name is the public selector and is deliberately guessable, so the
	// cap on what a flow may grant is the control that makes that safe. A role
	// carrying any management-plane permission must be refused outright.
	t.Run("role carrying an administrative permission is refused", func(t *testing.T) {
		err := runUpdateWithRole(t, func(m *rfSvcMocks) {
			m.roleRepo.findByUUIDFn = func(_ any, _ ...string) (*Role, error) {
				return &Role{RoleID: 10, RoleUUID: roleUUID, TenantID: tenantID, Name: "ops", Status: shared.StatusActive}, nil
			}
			m.permNameReader.findPermissionNamesByRoleIDFn = func(int64) ([]string, error) {
				return []string{"public:login", "account:profile:read:self", "tenant:delete"}, nil
			}
			m.grantsRole()
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "administrative permission")
		assert.Contains(t, err.Error(), "tenant:delete", "names the offending permission")
		assert.Contains(t, err.Error(), "use an invite instead")
	})

	t.Run("role with only public and own-account permissions is allowed", func(t *testing.T) {
		err := runUpdateWithRoleCommit(t, func(m *rfSvcMocks) {
			m.roleRepo.findByUUIDFn = func(_ any, _ ...string) (*Role, error) {
				return &Role{RoleID: 10, RoleUUID: roleUUID, TenantID: tenantID, Name: "customer", Status: shared.StatusActive}, nil
			}
			m.permNameReader.findPermissionNamesByRoleIDFn = func(int64) ([]string, error) {
				return []string{"public:register", "account:profile:update:self"}, nil
			}
			m.grantsRole()
		})
		require.NoError(t, err)
	})

	t.Run("inactive role is refused", func(t *testing.T) {
		err := runUpdateWithRole(t, func(m *rfSvcMocks) {
			m.roleRepo.findByUUIDFn = func(_ any, _ ...string) (*Role, error) {
				return &Role{RoleID: 10, RoleUUID: roleUUID, TenantID: tenantID, Name: "retired", Status: shared.StatusInactive}, nil
			}
			m.grantsRole()
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "inactive")
	})

	t.Run("permission lookup error is propagated", func(t *testing.T) {
		err := runUpdateWithRole(t, func(m *rfSvcMocks) {
			m.roleRepo.findByUUIDFn = func(_ any, _ ...string) (*Role, error) {
				return &Role{RoleID: 10, RoleUUID: roleUUID, TenantID: tenantID, Status: shared.StatusActive}, nil
			}
			m.permNameReader.findPermissionNamesByRoleIDFn = func(int64) ([]string, error) {
				return nil, errors.New("perm err")
			}
			m.grantsRole()
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "perm err")
	})

	t.Run("unknown role is not found", func(t *testing.T) {
		err := runUpdateWithRole(t, func(m *rfSvcMocks) {
			m.roleRepo.findByUUIDFn = func(_ any, _ ...string) (*Role, error) { return nil, nil }
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "role not found: "+roleUUID.String())
	})

	t.Run("role lookup error is reported as not found", func(t *testing.T) {
		err := runUpdateWithRole(t, func(m *rfSvcMocks) {
			m.roleRepo.findByUUIDFn = func(_ any, _ ...string) (*Role, error) { return nil, errors.New("db error") }
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "role not found")
	})

	t.Run("role from another tenant is forbidden", func(t *testing.T) {
		err := runUpdateWithRole(t, func(m *rfSvcMocks) {
			m.roleRepo.findByUUIDFn = func(_ any, _ ...string) (*Role, error) {
				return &Role{RoleID: 10, RoleUUID: roleUUID, TenantID: 999, Status: shared.StatusActive}, nil
			}
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "role does not belong to the same tenant")
	})

	// A seeded system role carries privileged grants; a public self-service flow
	// must never be able to hand one out.
	t.Run("system role can never be attached", func(t *testing.T) {
		err := runUpdateWithRole(t, func(m *rfSvcMocks) {
			m.grantsRole() // even if the actor holds it
			m.roleRepo.findByUUIDFn = func(_ any, _ ...string) (*Role, error) {
				return &Role{RoleID: 10, RoleUUID: roleUUID, TenantID: tenantID, IsSystem: true, Status: shared.StatusActive}, nil
			}
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "system roles cannot be assigned to a registration flow")
	})

	// Without this, registration-flow:update would confer strictly more power
	// than user:invite — the actual privilege-escalation path.
	t.Run("actor cannot grant a role it does not possess", func(t *testing.T) {
		err := runUpdateWithRole(t, func(m *rfSvcMocks) {
			m.roleRepo.findByUUIDFn = func(_ any, _ ...string) (*Role, error) {
				return &Role{RoleID: 10, RoleUUID: roleUUID, TenantID: tenantID, Status: shared.StatusActive}, nil
			}
			m.userRoleRepo.findByUserIDAndRoleIDFn = func(int64, int64) (*UserRole, error) { return nil, nil }
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "you cannot grant roles you do not possess")
	})

	t.Run("possession lookup error is propagated", func(t *testing.T) {
		err := runUpdateWithRole(t, func(m *rfSvcMocks) {
			m.roleRepo.findByUUIDFn = func(_ any, _ ...string) (*Role, error) {
				return &Role{RoleID: 10, RoleUUID: roleUUID, TenantID: tenantID, Status: shared.StatusActive}, nil
			}
			m.userRoleRepo.findByUserIDAndRoleIDFn = func(int64, int64) (*UserRole, error) {
				return nil, errors.New("user role err")
			}
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "user role err")
	})

	t.Run("possession is checked for the acting user and the exact role", func(t *testing.T) {
		var gotUserID, gotRoleID int64
		err := runUpdateWithRole(t, func(m *rfSvcMocks) {
			m.roleRepo.findByUUIDFn = func(_ any, _ ...string) (*Role, error) {
				return &Role{RoleID: 10, RoleUUID: roleUUID, TenantID: tenantID, Status: shared.StatusActive}, nil
			}
			m.userRoleRepo.findByUserIDAndRoleIDFn = func(userID, roleID int64) (*UserRole, error) {
				gotUserID, gotRoleID = userID, roleID
				return nil, nil
			}
		})
		require.Error(t, err)
		assert.Equal(t, testActorUserID, gotUserID)
		assert.Equal(t, int64(10), gotRoleID)
	})
}

// ---------------------------------------------------------------------------
// SetStatus
// ---------------------------------------------------------------------------

func TestRegistrationFlowService_SetStatus(t *testing.T) {
	t.Run("not found", func(t *testing.T) {
		db, mock := newMockGormDBRegex(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		m := newRFSvcMocks()
		m.flowRepo.findByUUIDAndTenantIDFn = func(uuid.UUID, int64, ...string) (*RegistrationFlow, error) { return nil, nil }
		_, err := m.svc(db).SetStatus(context.Background(), uuid.New(), tenantID, testUserUUID, shared.StatusInactive)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "registration flow not found or access denied")
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("actor is validated before the system-flow guard", func(t *testing.T) {
		db, mock := newMockGormDBRegex(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		m := newRFSvcMocks()
		flow := buildRegistrationFlow()
		flow.IsSystem = true
		m.flowRepo.findByUUIDAndTenantIDFn = func(uuid.UUID, int64, ...string) (*RegistrationFlow, error) { return flow, nil }
		m.userRepo.findByUUIDFn = func(_ any, _ ...string) (*User, error) { return nil, nil }
		_, err := m.svc(db).SetStatus(context.Background(), flow.RegistrationFlowUUID, tenantID, testUserUUID, shared.StatusInactive)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "actor user not found")
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	// Status is the kill switch on a published link: a system flow must not be
	// re-activated through the ordinary admin API.
	t.Run("system flow status cannot be changed", func(t *testing.T) {
		db, mock := newMockGormDBRegex(t)
		mock.ExpectBegin()
		expectTenantSelect(mock, tenantID)
		mock.ExpectRollback()
		m := newRFSvcMocks()
		flow := buildRegistrationFlow()
		flow.IsSystem = true
		flow.Status = shared.StatusInactive
		m.flowRepo.findByUUIDAndTenantIDFn = func(uuid.UUID, int64, ...string) (*RegistrationFlow, error) { return flow, nil }
		saved := false
		m.flowRepo.createOrUpdateFn = func(e *RegistrationFlow) (*RegistrationFlow, error) { saved = true; return e, nil }
		_, err := m.svc(db).SetStatus(context.Background(), flow.RegistrationFlowUUID, tenantID, testUserUUID, shared.StatusActive)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "system registration flow is not allowed to be updated")
		assert.False(t, saved)
		assert.Equal(t, shared.StatusInactive, flow.Status)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("save error", func(t *testing.T) {
		db, mock := newMockGormDBRegex(t)
		mock.ExpectBegin()
		expectTenantSelect(mock, tenantID)
		mock.ExpectRollback()
		m := newRFSvcMocks()
		flow := buildRegistrationFlow()
		m.flowRepo.findByUUIDAndTenantIDFn = func(uuid.UUID, int64, ...string) (*RegistrationFlow, error) { return flow, nil }
		m.flowRepo.createOrUpdateFn = func(*RegistrationFlow) (*RegistrationFlow, error) { return nil, errors.New("save err") }
		_, err := m.svc(db).SetStatus(context.Background(), flow.RegistrationFlowUUID, tenantID, testUserUUID, shared.StatusInactive)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "save err")
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("success sets the status and stamps the actor", func(t *testing.T) {
		db, mock := newMockGormDBRegex(t)
		mock.ExpectBegin()
		expectTenantSelect(mock, tenantID)
		mock.ExpectCommit()
		m := newRFSvcMocks()
		flow := buildRegistrationFlow()
		m.flowRepo.findByUUIDAndTenantIDFn = func(uuid.UUID, int64, ...string) (*RegistrationFlow, error) { return flow, nil }
		var saved *RegistrationFlow
		m.flowRepo.createOrUpdateFn = func(e *RegistrationFlow) (*RegistrationFlow, error) { saved = e; return e, nil }
		res, err := m.svc(db).SetStatus(context.Background(), flow.RegistrationFlowUUID, tenantID, testUserUUID, shared.StatusInactive)
		require.NoError(t, err)
		require.NotNil(t, res)
		require.NotNil(t, saved)
		assert.Equal(t, shared.StatusInactive, saved.Status)
		require.NotNil(t, saved.UpdatedBy)
		assert.Equal(t, testActorUserID, *saved.UpdatedBy)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

// ---------------------------------------------------------------------------
// Delete
// ---------------------------------------------------------------------------

func TestRegistrationFlowService_Delete(t *testing.T) {
	t.Run("not found", func(t *testing.T) {
		db, mock := newMockGormDBRegex(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		m := newRFSvcMocks()
		m.flowRepo.findByUUIDAndTenantIDFn = func(uuid.UUID, int64, ...string) (*RegistrationFlow, error) { return nil, nil }
		_, err := m.svc(db).Delete(context.Background(), uuid.New(), tenantID, testUserUUID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "registration flow not found or access denied")
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("actor is validated before the system-flow guard", func(t *testing.T) {
		db, mock := newMockGormDBRegex(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		m := newRFSvcMocks()
		flow := buildRegistrationFlow()
		flow.IsSystem = true
		m.flowRepo.findByUUIDAndTenantIDFn = func(uuid.UUID, int64, ...string) (*RegistrationFlow, error) { return flow, nil }
		m.userRepo.findByUUIDFn = func(_ any, _ ...string) (*User, error) { return nil, nil }
		_, err := m.svc(db).Delete(context.Background(), flow.RegistrationFlowUUID, tenantID, testUserUUID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "actor user not found")
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("system flow cannot be deleted", func(t *testing.T) {
		db, mock := newMockGormDBRegex(t)
		mock.ExpectBegin()
		expectTenantSelect(mock, tenantID)
		mock.ExpectRollback()
		m := newRFSvcMocks()
		flow := buildRegistrationFlow()
		flow.IsSystem = true
		m.flowRepo.findByUUIDAndTenantIDFn = func(uuid.UUID, int64, ...string) (*RegistrationFlow, error) { return flow, nil }
		deleted := false
		m.flowRepo.deleteByUUIDFn = func(any) error { deleted = true; return nil }
		countCalls := 0
		m.inviteCounter.countPendingByRegistrationFlowIDFn = func(int64) (int64, error) { countCalls++; return 0, nil }
		_, err := m.svc(db).Delete(context.Background(), flow.RegistrationFlowUUID, tenantID, testUserUUID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "system registration flow is not allowed to be deleted")
		assert.False(t, deleted)
		assert.Zero(t, countCalls, "must short-circuit before the invite count")
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	// A flow still referenced by pending invites cannot be deleted out from
	// under them; the count runs inside the tx so it cannot go stale.
	t.Run("pending invites block the delete", func(t *testing.T) {
		db, mock := newMockGormDBRegex(t)
		mock.ExpectBegin()
		expectTenantSelect(mock, tenantID)
		mock.ExpectRollback()
		m := newRFSvcMocks()
		flow := buildRegistrationFlow()
		m.flowRepo.findByUUIDAndTenantIDFn = func(uuid.UUID, int64, ...string) (*RegistrationFlow, error) { return flow, nil }
		var countedFlowID int64
		m.inviteCounter.countPendingByRegistrationFlowIDFn = func(id int64) (int64, error) {
			countedFlowID = id
			return 2, nil
		}
		deleted := false
		m.flowRepo.deleteByUUIDFn = func(any) error { deleted = true; return nil }
		_, err := m.svc(db).Delete(context.Background(), flow.RegistrationFlowUUID, tenantID, testUserUUID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "pending invites")
		assert.Equal(t, flow.RegistrationFlowID, countedFlowID)
		assert.False(t, deleted)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("invite count error", func(t *testing.T) {
		db, mock := newMockGormDBRegex(t)
		mock.ExpectBegin()
		expectTenantSelect(mock, tenantID)
		mock.ExpectRollback()
		m := newRFSvcMocks()
		flow := buildRegistrationFlow()
		m.flowRepo.findByUUIDAndTenantIDFn = func(uuid.UUID, int64, ...string) (*RegistrationFlow, error) { return flow, nil }
		m.inviteCounter.countPendingByRegistrationFlowIDFn = func(int64) (int64, error) {
			return 0, errors.New("count err")
		}
		_, err := m.svc(db).Delete(context.Background(), flow.RegistrationFlowUUID, tenantID, testUserUUID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "count err")
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	// registration_flows is soft-deleted, which does NOT fire the FK cascade, so
	// the membership must be cleared explicitly or it outlives the flow.
	t.Run("role membership is cleared before the soft delete", func(t *testing.T) {
		db, mock := newMockGormDBRegex(t)
		mock.ExpectBegin()
		expectTenantSelect(mock, tenantID)
		mock.ExpectCommit()
		m := newRFSvcMocks()
		flow := buildRegistrationFlow()
		m.flowRepo.findByUUIDAndTenantIDFn = func(uuid.UUID, int64, ...string) (*RegistrationFlow, error) { return flow, nil }
		var order []string
		var clearedFlowID int64
		m.flowRoleRepo.deleteByRegistrationFlowIDFn = func(id int64) error {
			clearedFlowID = id
			order = append(order, "clear-roles")
			return nil
		}
		m.flowRepo.deleteByUUIDFn = func(any) error {
			order = append(order, "delete-flow")
			return nil
		}
		res, err := m.svc(db).Delete(context.Background(), flow.RegistrationFlowUUID, tenantID, testUserUUID)
		require.NoError(t, err)
		require.NotNil(t, res)
		assert.Equal(t, []string{"clear-roles", "delete-flow"}, order)
		assert.Equal(t, flow.RegistrationFlowID, clearedFlowID)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("membership clear error rolls back before the delete", func(t *testing.T) {
		db, mock := newMockGormDBRegex(t)
		mock.ExpectBegin()
		expectTenantSelect(mock, tenantID)
		mock.ExpectRollback()
		m := newRFSvcMocks()
		flow := buildRegistrationFlow()
		m.flowRepo.findByUUIDAndTenantIDFn = func(uuid.UUID, int64, ...string) (*RegistrationFlow, error) { return flow, nil }
		m.flowRoleRepo.deleteByRegistrationFlowIDFn = func(int64) error { return errors.New("clear err") }
		deleted := false
		m.flowRepo.deleteByUUIDFn = func(any) error { deleted = true; return nil }
		_, err := m.svc(db).Delete(context.Background(), flow.RegistrationFlowUUID, tenantID, testUserUUID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "clear err")
		assert.False(t, deleted, "the flow must not be deleted when its membership could not be cleared")
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("delete error", func(t *testing.T) {
		db, mock := newMockGormDBRegex(t)
		mock.ExpectBegin()
		expectTenantSelect(mock, tenantID)
		mock.ExpectRollback()
		m := newRFSvcMocks()
		flow := buildRegistrationFlow()
		m.flowRepo.findByUUIDAndTenantIDFn = func(uuid.UUID, int64, ...string) (*RegistrationFlow, error) { return flow, nil }
		m.flowRepo.deleteByUUIDFn = func(any) error { return errors.New("delete err") }
		_, err := m.svc(db).Delete(context.Background(), flow.RegistrationFlowUUID, tenantID, testUserUUID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "delete err")
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("success returns the pre-delete snapshot", func(t *testing.T) {
		db, mock := newMockGormDBRegex(t)
		mock.ExpectBegin()
		expectTenantSelect(mock, tenantID)
		mock.ExpectCommit()
		m := newRFSvcMocks()
		flow := buildRegistrationFlow()
		var gotPreloads []string
		m.flowRepo.findByUUIDAndTenantIDFn = func(_ uuid.UUID, _ int64, preloads ...string) (*RegistrationFlow, error) {
			gotPreloads = preloads
			return flow, nil
		}
		res, err := m.svc(db).Delete(context.Background(), flow.RegistrationFlowUUID, tenantID, testUserUUID)
		require.NoError(t, err)
		require.NotNil(t, res)
		assert.Equal(t, flow.Name, res.Name)
		assert.Equal(t, flow.Name, res.Name)
		require.NotNil(t, res.ClientUUID)
		assert.Equal(t, flow.Client.ClientUUID, *res.ClientUUID)
		assert.Equal(t, []string{"Client"}, gotPreloads)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

// ---------------------------------------------------------------------------
// AssignRoles
// ---------------------------------------------------------------------------

func TestRegistrationFlowService_AssignRoles(t *testing.T) {
	roleUUID := uuid.New()
	activeRole := func() *Role {
		return &Role{
			RoleID: 10, RoleUUID: roleUUID, TenantID: tenantID,
			Name: "partner-user", Description: "Partner user",
			Status: shared.StatusActive, IsDefault: true,
		}
	}

	t.Run("flow not found", func(t *testing.T) {
		db, mock := newMockGormDBRegex(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		m := newRFSvcMocks()
		m.flowRepo.findByUUIDAndTenantIDFn = func(uuid.UUID, int64, ...string) (*RegistrationFlow, error) { return nil, nil }
		_, err := m.svc(db).AssignRoles(context.Background(), uuid.New(), tenantID, testUserUUID, []uuid.UUID{roleUUID})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "registration flow not found or access denied")
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("actor is validated before the system-flow guard", func(t *testing.T) {
		db, mock := newMockGormDBRegex(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		m := newRFSvcMocks()
		flow := buildRegistrationFlow()
		flow.IsSystem = true
		m.flowRepo.findByUUIDAndTenantIDFn = func(uuid.UUID, int64, ...string) (*RegistrationFlow, error) { return flow, nil }
		m.userRepo.findByUUIDFn = func(_ any, _ ...string) (*User, error) { return nil, nil }
		_, err := m.svc(db).AssignRoles(context.Background(), flow.RegistrationFlowUUID, tenantID, testUserUUID, []uuid.UUID{roleUUID})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "actor user not found")
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("system flow cannot receive roles", func(t *testing.T) {
		db, mock := newMockGormDBRegex(t)
		mock.ExpectBegin()
		expectTenantSelect(mock, tenantID)
		mock.ExpectRollback()
		m := newRFSvcMocks()
		flow := buildRegistrationFlow()
		flow.IsSystem = true
		m.flowRepo.findByUUIDAndTenantIDFn = func(uuid.UUID, int64, ...string) (*RegistrationFlow, error) { return flow, nil }
		created := false
		m.flowRoleRepo.createFn = func(e *RegistrationFlowRole) (*RegistrationFlowRole, error) { created = true; return e, nil }
		_, err := m.svc(db).AssignRoles(context.Background(), flow.RegistrationFlowUUID, tenantID, testUserUUID, []uuid.UUID{roleUUID})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "system registration flow is not allowed to be modified")
		assert.False(t, created)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("system role is forbidden", func(t *testing.T) {
		db, mock := newMockGormDBRegex(t)
		mock.ExpectBegin()
		expectTenantSelect(mock, tenantID)
		mock.ExpectRollback()
		m := newRFSvcMocks()
		m.grantsRole()
		flow := buildRegistrationFlow()
		m.flowRepo.findByUUIDAndTenantIDFn = func(uuid.UUID, int64, ...string) (*RegistrationFlow, error) { return flow, nil }
		m.roleRepo.findByUUIDFn = func(_ any, _ ...string) (*Role, error) {
			r := activeRole()
			r.IsSystem = true
			return r, nil
		}
		_, err := m.svc(db).AssignRoles(context.Background(), flow.RegistrationFlowUUID, tenantID, testUserUUID, []uuid.UUID{roleUUID})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "system roles cannot be assigned to a registration flow")
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("actor cannot grant a role it does not possess", func(t *testing.T) {
		db, mock := newMockGormDBRegex(t)
		mock.ExpectBegin()
		expectTenantSelect(mock, tenantID)
		mock.ExpectRollback()
		m := newRFSvcMocks()
		flow := buildRegistrationFlow()
		m.flowRepo.findByUUIDAndTenantIDFn = func(uuid.UUID, int64, ...string) (*RegistrationFlow, error) { return flow, nil }
		m.roleRepo.findByUUIDFn = func(_ any, _ ...string) (*Role, error) { return activeRole(), nil }
		m.userRoleRepo.findByUserIDAndRoleIDFn = func(int64, int64) (*UserRole, error) { return nil, nil }
		_, err := m.svc(db).AssignRoles(context.Background(), flow.RegistrationFlowUUID, tenantID, testUserUUID, []uuid.UUID{roleUUID})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "you cannot grant roles you do not possess")
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("role from another tenant is forbidden", func(t *testing.T) {
		db, mock := newMockGormDBRegex(t)
		mock.ExpectBegin()
		expectTenantSelect(mock, tenantID)
		mock.ExpectRollback()
		m := newRFSvcMocks()
		m.grantsRole()
		flow := buildRegistrationFlow()
		m.flowRepo.findByUUIDAndTenantIDFn = func(uuid.UUID, int64, ...string) (*RegistrationFlow, error) { return flow, nil }
		m.roleRepo.findByUUIDFn = func(_ any, _ ...string) (*Role, error) {
			r := activeRole()
			r.TenantID = 999
			return r, nil
		}
		_, err := m.svc(db).AssignRoles(context.Background(), flow.RegistrationFlowUUID, tenantID, testUserUUID, []uuid.UUID{roleUUID})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "role does not belong to the same tenant")
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("existing-assignment lookup error", func(t *testing.T) {
		db, mock := newMockGormDBRegex(t)
		mock.ExpectBegin()
		expectTenantSelect(mock, tenantID)
		mock.ExpectRollback()
		m := newRFSvcMocks()
		m.grantsRole()
		flow := buildRegistrationFlow()
		m.flowRepo.findByUUIDAndTenantIDFn = func(uuid.UUID, int64, ...string) (*RegistrationFlow, error) { return flow, nil }
		m.roleRepo.findByUUIDFn = func(_ any, _ ...string) (*Role, error) { return activeRole(), nil }
		m.flowRoleRepo.findByRegistrationFlowIDAndRoleIDFn = func(int64, int64) (*RegistrationFlowRole, error) {
			return nil, errors.New("lookup err")
		}
		_, err := m.svc(db).AssignRoles(context.Background(), flow.RegistrationFlowUUID, tenantID, testUserUUID, []uuid.UUID{roleUUID})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "lookup err")
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("already assigned role is skipped and an empty non-nil slice is returned", func(t *testing.T) {
		db, mock := newMockGormDBRegex(t)
		mock.ExpectBegin()
		expectTenantSelect(mock, tenantID)
		mock.ExpectCommit()
		m := newRFSvcMocks()
		m.grantsRole()
		flow := buildRegistrationFlow()
		m.flowRepo.findByUUIDAndTenantIDFn = func(uuid.UUID, int64, ...string) (*RegistrationFlow, error) { return flow, nil }
		m.roleRepo.findByUUIDFn = func(_ any, _ ...string) (*Role, error) { return activeRole(), nil }
		m.flowRoleRepo.findByRegistrationFlowIDAndRoleIDFn = func(int64, int64) (*RegistrationFlowRole, error) {
			return &RegistrationFlowRole{RegistrationFlowRoleID: 99}, nil
		}
		created := false
		m.flowRoleRepo.createFn = func(e *RegistrationFlowRole) (*RegistrationFlowRole, error) { created = true; return e, nil }
		res, err := m.svc(db).AssignRoles(context.Background(), flow.RegistrationFlowUUID, tenantID, testUserUUID, []uuid.UUID{roleUUID})
		require.NoError(t, err)
		require.NotNil(t, res, "an empty result must be a non-nil slice so it serialises as []")
		assert.Empty(t, res)
		assert.False(t, created)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("create error", func(t *testing.T) {
		db, mock := newMockGormDBRegex(t)
		mock.ExpectBegin()
		expectTenantSelect(mock, tenantID)
		mock.ExpectRollback()
		m := newRFSvcMocks()
		m.grantsRole()
		flow := buildRegistrationFlow()
		m.flowRepo.findByUUIDAndTenantIDFn = func(uuid.UUID, int64, ...string) (*RegistrationFlow, error) { return flow, nil }
		m.roleRepo.findByUUIDFn = func(_ any, _ ...string) (*Role, error) { return activeRole(), nil }
		m.flowRoleRepo.createFn = func(*RegistrationFlowRole) (*RegistrationFlowRole, error) {
			return nil, errors.New("create sfr err")
		}
		_, err := m.svc(db).AssignRoles(context.Background(), flow.RegistrationFlowUUID, tenantID, testUserUUID, []uuid.UUID{roleUUID})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "create sfr err")
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("success returns the newly assigned roles", func(t *testing.T) {
		db, mock := newMockGormDBRegex(t)
		mock.ExpectBegin()
		expectTenantSelect(mock, tenantID)
		mock.ExpectCommit()
		m := newRFSvcMocks()
		m.grantsRole()
		flow := buildRegistrationFlow()
		m.flowRepo.findByUUIDAndTenantIDFn = func(uuid.UUID, int64, ...string) (*RegistrationFlow, error) { return flow, nil }
		m.roleRepo.findByUUIDFn = func(_ any, _ ...string) (*Role, error) { return activeRole(), nil }
		sfrUUID := uuid.New()
		m.flowRoleRepo.createFn = func(e *RegistrationFlowRole) (*RegistrationFlowRole, error) {
			assert.Equal(t, flow.RegistrationFlowID, e.RegistrationFlowID)
			assert.Equal(t, int64(10), e.RoleID)
			e.RegistrationFlowRoleUUID = sfrUUID
			return e, nil
		}
		res, err := m.svc(db).AssignRoles(context.Background(), flow.RegistrationFlowUUID, tenantID, testUserUUID, []uuid.UUID{roleUUID})
		require.NoError(t, err)
		require.Len(t, res, 1)
		assert.Equal(t, sfrUUID, res[0].RegistrationFlowRoleUUID)
		assert.Equal(t, flow.RegistrationFlowUUID, res[0].RegistrationFlowUUID)
		assert.Equal(t, roleUUID, res[0].RoleUUID)
		assert.Equal(t, "partner-user", res[0].RoleName)
		assert.Equal(t, "Partner user", res[0].RoleDescription)
		assert.Equal(t, shared.StatusActive, res[0].RoleStatus)
		assert.True(t, res[0].RoleIsDefault)
		assert.False(t, res[0].RoleIsSystem)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("no roles requested returns an empty non-nil slice", func(t *testing.T) {
		db, mock := newMockGormDBRegex(t)
		mock.ExpectBegin()
		expectTenantSelect(mock, tenantID)
		mock.ExpectCommit()
		m := newRFSvcMocks()
		flow := buildRegistrationFlow()
		m.flowRepo.findByUUIDAndTenantIDFn = func(uuid.UUID, int64, ...string) (*RegistrationFlow, error) { return flow, nil }
		res, err := m.svc(db).AssignRoles(context.Background(), flow.RegistrationFlowUUID, tenantID, testUserUUID, nil)
		require.NoError(t, err)
		require.NotNil(t, res)
		assert.Empty(t, res)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

// ---------------------------------------------------------------------------
// GetRoles
// ---------------------------------------------------------------------------

func TestRegistrationFlowService_GetRoles(t *testing.T) {
	t.Run("flow not found", func(t *testing.T) {
		db, _ := newMockGormDB(t)
		m := newRFSvcMocks()
		m.flowRepo.findByUUIDAndTenantIDFn = func(uuid.UUID, int64, ...string) (*RegistrationFlow, error) { return nil, nil }
		_, err := m.svc(db).GetRoles(context.Background(), uuid.New(), tenantID, 1, 10)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "registration flow not found or access denied")
	})

	t.Run("flow lookup error", func(t *testing.T) {
		db, _ := newMockGormDB(t)
		m := newRFSvcMocks()
		m.flowRepo.findByUUIDAndTenantIDFn = func(uuid.UUID, int64, ...string) (*RegistrationFlow, error) {
			return nil, errors.New("db error")
		}
		_, err := m.svc(db).GetRoles(context.Background(), uuid.New(), tenantID, 1, 10)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "registration flow not found or access denied")
	})

	t.Run("pagination error", func(t *testing.T) {
		db, _ := newMockGormDB(t)
		m := newRFSvcMocks()
		flow := buildRegistrationFlow()
		m.flowRepo.findByUUIDAndTenantIDFn = func(uuid.UUID, int64, ...string) (*RegistrationFlow, error) { return flow, nil }
		m.flowRoleRepo.findByRegistrationFlowIDPaginatedFn = func(int64, int, int) (*PaginationResult[RegistrationFlowRole], error) {
			return nil, errors.New("paginate err")
		}
		_, err := m.svc(db).GetRoles(context.Background(), flow.RegistrationFlowUUID, tenantID, 1, 10)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "paginate err")
	})

	t.Run("rows without a preloaded role are skipped", func(t *testing.T) {
		db, _ := newMockGormDB(t)
		m := newRFSvcMocks()
		flow := buildRegistrationFlow()
		m.flowRepo.findByUUIDAndTenantIDFn = func(uuid.UUID, int64, ...string) (*RegistrationFlow, error) { return flow, nil }
		m.flowRoleRepo.findByRegistrationFlowIDPaginatedFn = func(int64, int, int) (*PaginationResult[RegistrationFlowRole], error) {
			return &PaginationResult[RegistrationFlowRole]{
				Data:  []RegistrationFlowRole{{RegistrationFlowRoleUUID: uuid.New(), Role: nil}},
				Total: 1, Page: 1, Limit: 10, TotalPages: 1,
			}, nil
		}
		res, err := m.svc(db).GetRoles(context.Background(), flow.RegistrationFlowUUID, tenantID, 1, 10)
		require.NoError(t, err)
		assert.Empty(t, res.Data)
		assert.Equal(t, int64(1), res.Total, "the repository total is passed through untouched")
	})

	t.Run("success maps every role field and forwards pagination", func(t *testing.T) {
		db, _ := newMockGormDB(t)
		m := newRFSvcMocks()
		flow := buildRegistrationFlow()
		m.flowRepo.findByUUIDAndTenantIDFn = func(uuid.UUID, int64, ...string) (*RegistrationFlow, error) { return flow, nil }
		roleUUID := uuid.New()
		sfrUUID := uuid.New()
		m.flowRoleRepo.findByRegistrationFlowIDPaginatedFn = func(id int64, page, limit int) (*PaginationResult[RegistrationFlowRole], error) {
			assert.Equal(t, flow.RegistrationFlowID, id)
			assert.Equal(t, 2, page)
			assert.Equal(t, 5, limit)
			return &PaginationResult[RegistrationFlowRole]{
				Data: []RegistrationFlowRole{{
					RegistrationFlowRoleUUID: sfrUUID,
					Role: &Role{
						RoleUUID: roleUUID, Name: "viewer", Description: "read only",
						Status: shared.StatusActive, IsDefault: true, IsSystem: true,
					},
				}},
				Total: 11, Page: 2, Limit: 5, TotalPages: 3,
			}, nil
		}
		res, err := m.svc(db).GetRoles(context.Background(), flow.RegistrationFlowUUID, tenantID, 2, 5)
		require.NoError(t, err)
		require.Len(t, res.Data, 1)
		assert.Equal(t, sfrUUID, res.Data[0].RegistrationFlowRoleUUID)
		assert.Equal(t, flow.RegistrationFlowUUID, res.Data[0].RegistrationFlowUUID)
		assert.Equal(t, roleUUID, res.Data[0].RoleUUID)
		assert.Equal(t, "viewer", res.Data[0].RoleName)
		assert.Equal(t, "read only", res.Data[0].RoleDescription)
		assert.Equal(t, shared.StatusActive, res.Data[0].RoleStatus)
		assert.True(t, res.Data[0].RoleIsDefault)
		assert.True(t, res.Data[0].RoleIsSystem)
		assert.Equal(t, int64(11), res.Total)
		assert.Equal(t, 2, res.Page)
		assert.Equal(t, 5, res.Limit)
		assert.Equal(t, 3, res.TotalPages)
	})

	t.Run("empty page returns an empty non-nil slice", func(t *testing.T) {
		db, _ := newMockGormDB(t)
		m := newRFSvcMocks()
		flow := buildRegistrationFlow()
		m.flowRepo.findByUUIDAndTenantIDFn = func(uuid.UUID, int64, ...string) (*RegistrationFlow, error) { return flow, nil }
		res, err := m.svc(db).GetRoles(context.Background(), flow.RegistrationFlowUUID, tenantID, 1, 10)
		require.NoError(t, err)
		require.NotNil(t, res.Data)
		assert.Empty(t, res.Data)
	})
}

// ---------------------------------------------------------------------------
// RemoveRole
// ---------------------------------------------------------------------------

func TestRegistrationFlowService_RemoveRole(t *testing.T) {
	roleUUID := uuid.New()

	t.Run("flow not found", func(t *testing.T) {
		db, mock := newMockGormDBRegex(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		m := newRFSvcMocks()
		m.flowRepo.findByUUIDAndTenantIDFn = func(uuid.UUID, int64, ...string) (*RegistrationFlow, error) { return nil, nil }
		_, err := m.svc(db).RemoveRole(context.Background(), uuid.New(), tenantID, testUserUUID, roleUUID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "registration flow not found or access denied")
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("actor is validated before the system-flow guard", func(t *testing.T) {
		db, mock := newMockGormDBRegex(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		m := newRFSvcMocks()
		flow := buildRegistrationFlow()
		flow.IsSystem = true
		m.flowRepo.findByUUIDAndTenantIDFn = func(uuid.UUID, int64, ...string) (*RegistrationFlow, error) { return flow, nil }
		m.userRepo.findByUUIDFn = func(_ any, _ ...string) (*User, error) { return nil, nil }
		_, err := m.svc(db).RemoveRole(context.Background(), flow.RegistrationFlowUUID, tenantID, testUserUUID, roleUUID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "actor user not found")
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("system flow roles cannot be removed", func(t *testing.T) {
		db, mock := newMockGormDBRegex(t)
		mock.ExpectBegin()
		expectTenantSelect(mock, tenantID)
		mock.ExpectRollback()
		m := newRFSvcMocks()
		flow := buildRegistrationFlow()
		flow.IsSystem = true
		m.flowRepo.findByUUIDAndTenantIDFn = func(uuid.UUID, int64, ...string) (*RegistrationFlow, error) { return flow, nil }
		removed := false
		m.flowRoleRepo.deleteByRegistrationFlowIDAndRoleIDFn = func(int64, int64) error { removed = true; return nil }
		_, err := m.svc(db).RemoveRole(context.Background(), flow.RegistrationFlowUUID, tenantID, testUserUUID, roleUUID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "system registration flow is not allowed to be modified")
		assert.False(t, removed)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("role not found", func(t *testing.T) {
		db, mock := newMockGormDBRegex(t)
		mock.ExpectBegin()
		expectTenantSelect(mock, tenantID)
		mock.ExpectRollback()
		m := newRFSvcMocks()
		flow := buildRegistrationFlow()
		m.flowRepo.findByUUIDAndTenantIDFn = func(uuid.UUID, int64, ...string) (*RegistrationFlow, error) { return flow, nil }
		m.roleRepo.findByUUIDFn = func(_ any, _ ...string) (*Role, error) { return nil, nil }
		_, err := m.svc(db).RemoveRole(context.Background(), flow.RegistrationFlowUUID, tenantID, testUserUUID, roleUUID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "role not found")
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("role from another tenant is reported as not found", func(t *testing.T) {
		db, mock := newMockGormDBRegex(t)
		mock.ExpectBegin()
		expectTenantSelect(mock, tenantID)
		mock.ExpectRollback()
		m := newRFSvcMocks()
		flow := buildRegistrationFlow()
		m.flowRepo.findByUUIDAndTenantIDFn = func(uuid.UUID, int64, ...string) (*RegistrationFlow, error) { return flow, nil }
		m.roleRepo.findByUUIDFn = func(_ any, _ ...string) (*Role, error) {
			return &Role{RoleID: 10, RoleUUID: roleUUID, TenantID: 999, Status: shared.StatusActive}, nil
		}
		removed := false
		m.flowRoleRepo.deleteByRegistrationFlowIDAndRoleIDFn = func(int64, int64) error { removed = true; return nil }
		_, err := m.svc(db).RemoveRole(context.Background(), flow.RegistrationFlowUUID, tenantID, testUserUUID, roleUUID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "role not found: "+roleUUID.String())
		assert.False(t, removed)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("delete error", func(t *testing.T) {
		db, mock := newMockGormDBRegex(t)
		mock.ExpectBegin()
		expectTenantSelect(mock, tenantID)
		mock.ExpectRollback()
		m := newRFSvcMocks()
		flow := buildRegistrationFlow()
		m.flowRepo.findByUUIDAndTenantIDFn = func(uuid.UUID, int64, ...string) (*RegistrationFlow, error) { return flow, nil }
		m.roleRepo.findByUUIDFn = func(_ any, _ ...string) (*Role, error) {
			return &Role{RoleID: 10, RoleUUID: roleUUID, TenantID: tenantID, Status: shared.StatusActive}, nil
		}
		m.flowRoleRepo.deleteByRegistrationFlowIDAndRoleIDFn = func(int64, int64) error { return errors.New("del err") }
		_, err := m.svc(db).RemoveRole(context.Background(), flow.RegistrationFlowUUID, tenantID, testUserUUID, roleUUID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "del err")
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	// RemoveRole returns the parent flow so the caller can re-render without a
	// follow-up GET.
	t.Run("success removes the membership and returns the parent flow", func(t *testing.T) {
		db, mock := newMockGormDBRegex(t)
		mock.ExpectBegin()
		expectTenantSelect(mock, tenantID)
		mock.ExpectCommit()
		m := newRFSvcMocks()
		flow := buildRegistrationFlow()
		m.flowRepo.findByUUIDAndTenantIDFn = func(uuid.UUID, int64, ...string) (*RegistrationFlow, error) { return flow, nil }
		m.roleRepo.findByUUIDFn = func(_ any, _ ...string) (*Role, error) {
			return &Role{RoleID: 10, RoleUUID: roleUUID, TenantID: tenantID, Status: shared.StatusActive}, nil
		}
		var gotFlowID, gotRoleID int64
		m.flowRoleRepo.deleteByRegistrationFlowIDAndRoleIDFn = func(flowID, roleID int64) error {
			gotFlowID, gotRoleID = flowID, roleID
			return nil
		}
		res, err := m.svc(db).RemoveRole(context.Background(), flow.RegistrationFlowUUID, tenantID, testUserUUID, roleUUID)
		require.NoError(t, err)
		require.NotNil(t, res)
		assert.Equal(t, flow.RegistrationFlowUUID, res.RegistrationFlowUUID)
		assert.Equal(t, flow.Name, res.Name)
		assert.Equal(t, flow.RegistrationFlowID, gotFlowID)
		assert.Equal(t, int64(10), gotRoleID)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

// ---------------------------------------------------------------------------
// constructor
// ---------------------------------------------------------------------------

func TestNewRegistrationFlowService(t *testing.T) {
	db, _ := newMockGormDB(t)
	assert.NotNil(t, newRFSvcMocks().svc(db))
}

// syncRoles writes membership inside the caller's transaction: either write
// failing must abort the whole update, never leave a half-synced set.
func TestRegistrationFlowService_SyncRolesWriteFailures(t *testing.T) {
	keepUUID := uuid.New()

	setup := func(t *testing.T) (*rfSvcMocks, *gorm.DB, sqlmock.Sqlmock, *RegistrationFlow) {
		t.Helper()
		db, mock := newMockGormDBRegex(t)
		mock.ExpectBegin()
		expectTenantSelect(mock, tenantID)
		mock.ExpectRollback()
		m := newRFSvcMocks()
		m.grantsRole()
		flow := buildRegistrationFlow()
		m.flowRepo.findByUUIDAndTenantIDFn = func(uuid.UUID, int64, ...string) (*RegistrationFlow, error) { return flow, nil }
		m.roleRepo.findByUUIDFn = func(_ any, _ ...string) (*Role, error) {
			return &Role{RoleID: 30, RoleUUID: keepUUID, TenantID: tenantID, Status: shared.StatusActive}, nil
		}
		return m, db, mock, flow
	}

	t.Run("removing an extra role fails", func(t *testing.T) {
		m, db, mock, flow := setup(t)
		m.flowRoleRepo.findByRegistrationFlowIDFn = func(int64) ([]RegistrationFlowRole, error) {
			return []RegistrationFlowRole{{RoleID: 20}}, nil // not desired → must be removed
		}
		m.flowRoleRepo.deleteByRegistrationFlowIDAndRoleIDFn = func(int64, int64) error { return errors.New("remove err") }
		created := false
		m.flowRoleRepo.createFn = func(e *RegistrationFlowRole) (*RegistrationFlowRole, error) { created = true; return e, nil }

		in := validUpdateInput(flow.RegistrationFlowUUID)
		in.RoleUUIDs = []uuid.UUID{keepUUID}
		_, err := m.svc(db).Update(context.Background(), in)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "remove err")
		assert.False(t, created, "no additions once a removal has failed")
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("adding a missing role fails", func(t *testing.T) {
		m, db, mock, flow := setup(t)
		m.flowRoleRepo.findByRegistrationFlowIDFn = func(int64) ([]RegistrationFlowRole, error) { return nil, nil }
		m.flowRoleRepo.createFn = func(*RegistrationFlowRole) (*RegistrationFlowRole, error) {
			return nil, errors.New("attach err")
		}
		in := validUpdateInput(flow.RegistrationFlowUUID)
		in.RoleUUIDs = []uuid.UUID{keepUUID}
		_, err := m.svc(db).Update(context.Background(), in)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "attach err")
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("a role already present is neither re-added nor removed", func(t *testing.T) {
		db, mock := newMockGormDBRegex(t)
		mock.ExpectBegin()
		expectTenantSelect(mock, tenantID)
		mock.ExpectCommit()
		m := newRFSvcMocks()
		m.grantsRole()
		flow := buildRegistrationFlow()
		m.flowRepo.findByUUIDAndTenantIDFn = func(uuid.UUID, int64, ...string) (*RegistrationFlow, error) { return flow, nil }
		m.roleRepo.findByUUIDFn = func(_ any, _ ...string) (*Role, error) {
			return &Role{RoleID: 30, RoleUUID: keepUUID, TenantID: tenantID, Status: shared.StatusActive}, nil
		}
		m.flowRoleRepo.findByRegistrationFlowIDFn = func(int64) ([]RegistrationFlowRole, error) {
			return []RegistrationFlowRole{{RoleID: 30}}, nil
		}
		writes := 0
		m.flowRoleRepo.createFn = func(e *RegistrationFlowRole) (*RegistrationFlowRole, error) { writes++; return e, nil }
		m.flowRoleRepo.deleteByRegistrationFlowIDAndRoleIDFn = func(int64, int64) error { writes++; return nil }

		in := validUpdateInput(flow.RegistrationFlowUUID)
		in.RoleUUIDs = []uuid.UUID{keepUUID}
		_, err := m.svc(db).Update(context.Background(), in)
		require.NoError(t, err)
		assert.Zero(t, writes, "an unchanged membership set is a no-op")
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}
