package invite

import (
	"os"
	"testing"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/branding"
	"github.com/maintainerd/auth/internal/platform/signedurl"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/stretchr/testify/require"
)

func TestMain(m *testing.M) {
	_ = signedurl.Configure([]byte("test-secret-key-for-hmac"))
	os.Exit(m.Run())
}

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

// ---------------------------------------------------------------------------
// Mock: ClientRepository
// ---------------------------------------------------------------------------

type mockClientRepo struct {
	findSystemFn func() (*Client, error)
}

func (m *mockClientRepo) WithTx(_ *gorm.DB) ClientRepository { return m }
func (m *mockClientRepo) FindSystem() (*Client, error) {
	if m.findSystemFn != nil {
		return m.findSystemFn()
	}
	return nil, nil
}
func (m *mockClientRepo) Create(e *Client) (*Client, error)               { return e, nil }
func (m *mockClientRepo) CreateOrUpdate(e *Client) (*Client, error)       { return e, nil }
func (m *mockClientRepo) FindAll(p ...string) ([]Client, error)           { return nil, nil }
func (m *mockClientRepo) FindByUUID(id any, p ...string) (*Client, error) { return nil, nil }
func (m *mockClientRepo) FindByUUIDs(ids []string, p ...string) ([]Client, error) {
	return nil, nil
}
func (m *mockClientRepo) FindByID(id any, p ...string) (*Client, error) { return nil, nil }
func (m *mockClientRepo) UpdateByUUID(id, data any) (*Client, error)    { return nil, nil }
func (m *mockClientRepo) UpdateByID(id, data any) (*Client, error)      { return nil, nil }
func (m *mockClientRepo) DeleteByUUID(id any) error                     { return nil }
func (m *mockClientRepo) DeleteByID(id any) error                       { return nil }
func (m *mockClientRepo) Paginate(c map[string]any, pg, lim int, p ...string) (*PaginationResult[Client], error) {
	return nil, nil
}

// ---------------------------------------------------------------------------
// Mock: RoleRepository
// ---------------------------------------------------------------------------

type mockRoleRepo struct {
	findByUUIDsFn func([]string, ...string) ([]Role, error)
}

func (m *mockRoleRepo) WithTx(_ *gorm.DB) RoleRepository { return m }
func (m *mockRoleRepo) FindByUUIDs(ids []string, p ...string) ([]Role, error) {
	if m.findByUUIDsFn != nil {
		return m.findByUUIDsFn(ids, p...)
	}
	return nil, nil
}
func (m *mockRoleRepo) Create(e *Role) (*Role, error)                 { return e, nil }
func (m *mockRoleRepo) CreateOrUpdate(e *Role) (*Role, error)         { return e, nil }
func (m *mockRoleRepo) FindAll(p ...string) ([]Role, error)           { return nil, nil }
func (m *mockRoleRepo) FindByUUID(id any, p ...string) (*Role, error) { return nil, nil }
func (m *mockRoleRepo) FindByID(id any, p ...string) (*Role, error)   { return nil, nil }
func (m *mockRoleRepo) UpdateByUUID(id, data any) (*Role, error)      { return nil, nil }
func (m *mockRoleRepo) UpdateByID(id, data any) (*Role, error)        { return nil, nil }
func (m *mockRoleRepo) DeleteByUUID(id any) error                     { return nil }
func (m *mockRoleRepo) DeleteByID(id any) error                       { return nil }
func (m *mockRoleRepo) Paginate(c map[string]any, pg, lim int, p ...string) (*PaginationResult[Role], error) {
	return nil, nil
}

// ---------------------------------------------------------------------------
// Mock: InviteRepository
// ---------------------------------------------------------------------------

type mockInviteRepo struct {
	createFn func(*Invite) (*Invite, error)
}

func (m *mockInviteRepo) WithTx(_ *gorm.DB) InviteRepository { return m }
func (m *mockInviteRepo) Create(e *Invite) (*Invite, error) {
	if m.createFn != nil {
		return m.createFn(e)
	}
	return e, nil
}
func (m *mockInviteRepo) CreateOrUpdate(e *Invite) (*Invite, error)       { return e, nil }
func (m *mockInviteRepo) FindAll(p ...string) ([]Invite, error)           { return nil, nil }
func (m *mockInviteRepo) FindByUUID(id any, p ...string) (*Invite, error) { return nil, nil }
func (m *mockInviteRepo) FindByUUIDs(ids []string, p ...string) ([]Invite, error) {
	return nil, nil
}
func (m *mockInviteRepo) FindByID(id any, p ...string) (*Invite, error) { return nil, nil }
func (m *mockInviteRepo) UpdateByUUID(id, data any) (*Invite, error)    { return nil, nil }
func (m *mockInviteRepo) UpdateByID(id, data any) (*Invite, error)      { return nil, nil }
func (m *mockInviteRepo) DeleteByUUID(id any) error                     { return nil }
func (m *mockInviteRepo) DeleteByID(id any) error                       { return nil }
func (m *mockInviteRepo) Paginate(c map[string]any, pg, lim int, p ...string) (*PaginationResult[Invite], error) {
	return nil, nil
}
func (m *mockInviteRepo) FindByUUIDAndTenantID(id uuid.UUID, tenantID int64, p ...string) (*Invite, error) {
	return nil, nil
}
func (m *mockInviteRepo) FindByToken(token string) (*Invite, error)          { return nil, nil }
func (m *mockInviteRepo) FindAllByClientID(clientID int64) ([]Invite, error) { return nil, nil }
func (m *mockInviteRepo) FindAllByTenantID(tenantID int64) ([]Invite, error) { return nil, nil }
func (m *mockInviteRepo) MarkAsUsed(id uuid.UUID) error                      { return nil }
func (m *mockInviteRepo) RevokeByUUID(id uuid.UUID) error                    { return nil }

// ---------------------------------------------------------------------------
// Mock: branding.EmailTemplateRepository
// ---------------------------------------------------------------------------

type mockEmailTemplateRepo struct {
	findByNameFn func(string) (*branding.EmailTemplate, error)
}

func (m *mockEmailTemplateRepo) Create(e *branding.EmailTemplate) (*branding.EmailTemplate, error) {
	return e, nil
}
func (m *mockEmailTemplateRepo) CreateOrUpdate(e *branding.EmailTemplate) (*branding.EmailTemplate, error) {
	return e, nil
}
func (m *mockEmailTemplateRepo) FindAll(p ...string) ([]branding.EmailTemplate, error) {
	return nil, nil
}
func (m *mockEmailTemplateRepo) FindByUUID(id any, p ...string) (*branding.EmailTemplate, error) {
	return nil, nil
}
func (m *mockEmailTemplateRepo) FindByUUIDs(ids []string, p ...string) ([]branding.EmailTemplate, error) {
	return nil, nil
}
func (m *mockEmailTemplateRepo) FindByID(id any, p ...string) (*branding.EmailTemplate, error) {
	return nil, nil
}
func (m *mockEmailTemplateRepo) UpdateByUUID(id, data any) (*branding.EmailTemplate, error) {
	return nil, nil
}
func (m *mockEmailTemplateRepo) UpdateByID(id, data any) (*branding.EmailTemplate, error) {
	return nil, nil
}
func (m *mockEmailTemplateRepo) DeleteByUUID(id any) error { return nil }
func (m *mockEmailTemplateRepo) DeleteByID(id any) error   { return nil }
func (m *mockEmailTemplateRepo) Paginate(c map[string]any, pg, lim int, p ...string) (*branding.PaginationResult[branding.EmailTemplate], error) {
	return nil, nil
}
func (m *mockEmailTemplateRepo) FindByUUIDAndTenantID(id uuid.UUID, tenantID int64, p ...string) (*branding.EmailTemplate, error) {
	return nil, nil
}
func (m *mockEmailTemplateRepo) FindByName(name string) (*branding.EmailTemplate, error) {
	if m.findByNameFn != nil {
		return m.findByNameFn(name)
	}
	return nil, nil
}
func (m *mockEmailTemplateRepo) FindPaginated(f branding.EmailTemplateRepositoryGetFilter) (*branding.PaginationResult[branding.EmailTemplate], error) {
	return nil, nil
}
