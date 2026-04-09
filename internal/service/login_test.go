package service

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"context"
	"errors"
	"testing"
	"time"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/config"
	"github.com/maintainerd/auth/internal/jwt"
	"github.com/maintainerd/auth/internal/model"
	"github.com/maintainerd/auth/internal/repository"
	"github.com/maintainerd/auth/internal/security"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// ---------------------------------------------------------------------------
// Mock: ClientRepository
// ---------------------------------------------------------------------------

type mockClientRepo struct {
	findByClientIDAndIdentityProviderFn func(clientID, providerID string) (*model.Client, error)
	findDefaultFn                       func() (*model.Client, error)
	findByUUIDFn                        func(any, ...string) (*model.Client, error)
	findByUUIDAndTenantIDFn             func(uuid.UUID, int64) (*model.Client, error)
	findPaginatedFn                     func(repository.ClientRepositoryGetFilter) (*repository.PaginationResult[model.Client], error)
	findByNameAndIdentityProviderFn     func(string, int64, int64) (*model.Client, error)
	findDefaultByTenantIDFn             func(tID int64) (*model.Client, error)
	createOrUpdateFn                    func(*model.Client) (*model.Client, error)
	deleteByUUIDFn                      func(any) error
	findByIDFn                          func(any, ...string) (*model.Client, error)
}

func (m *mockClientRepo) WithTx(_ *gorm.DB) repository.ClientRepository { return m }
func (m *mockClientRepo) FindByClientIDAndIdentityProvider(a, b string) (*model.Client, error) {
	if m.findByClientIDAndIdentityProviderFn != nil {
		return m.findByClientIDAndIdentityProviderFn(a, b)
	}
	return nil, nil
}
func (m *mockClientRepo) FindDefault() (*model.Client, error) {
	if m.findDefaultFn != nil {
		return m.findDefaultFn()
	}
	return nil, nil
}
func (m *mockClientRepo) Create(e *model.Client) (*model.Client, error) { return e, nil }
func (m *mockClientRepo) CreateOrUpdate(e *model.Client) (*model.Client, error) {
	if m.createOrUpdateFn != nil {
		return m.createOrUpdateFn(e)
	}
	return e, nil
}
func (m *mockClientRepo) FindAll(p ...string) ([]model.Client, error) { return nil, nil }
func (m *mockClientRepo) FindByUUID(id any, p ...string) (*model.Client, error) {
	if m.findByUUIDFn != nil {
		return m.findByUUIDFn(id, p...)
	}
	return nil, nil
}
func (m *mockClientRepo) FindByUUIDs(ids []string, p ...string) ([]model.Client, error) {
	return nil, nil
}
func (m *mockClientRepo) FindByID(id any, p ...string) (*model.Client, error) {
	if m.findByIDFn != nil {
		return m.findByIDFn(id, p...)
	}
	return nil, nil
}
func (m *mockClientRepo) UpdateByUUID(id, data any) (*model.Client, error) { return nil, nil }
func (m *mockClientRepo) UpdateByID(id, data any) (*model.Client, error)   { return nil, nil }
func (m *mockClientRepo) DeleteByUUID(id any) error {
	if m.deleteByUUIDFn != nil {
		return m.deleteByUUIDFn(id)
	}
	return nil
}
func (m *mockClientRepo) DeleteByID(id any) error { return nil }
func (m *mockClientRepo) Paginate(c map[string]any, pg, lim int, p ...string) (*repository.PaginationResult[model.Client], error) {
	return nil, nil
}
func (m *mockClientRepo) FindByUUIDAndTenantID(id uuid.UUID, tID int64) (*model.Client, error) {
	if m.findByUUIDAndTenantIDFn != nil {
		return m.findByUUIDAndTenantIDFn(id, tID)
	}
	return nil, nil
}
func (m *mockClientRepo) FindByNameAndIdentityProvider(n string, ipID, tID int64) (*model.Client, error) {
	if m.findByNameAndIdentityProviderFn != nil {
		return m.findByNameAndIdentityProviderFn(n, ipID, tID)
	}
	return nil, nil
}
func (m *mockClientRepo) FindByClientID(cID string, tID int64) (*model.Client, error) {
	return nil, nil
}
func (m *mockClientRepo) FindAllByTenantID(tID int64) ([]model.Client, error) { return nil, nil }
func (m *mockClientRepo) FindDefaultByTenantID(tID int64) (*model.Client, error) {
	if m.findDefaultByTenantIDFn != nil {
		return m.findDefaultByTenantIDFn(tID)
	}
	return nil, nil
}
func (m *mockClientRepo) FindPaginated(f repository.ClientRepositoryGetFilter) (*repository.PaginationResult[model.Client], error) {
	if m.findPaginatedFn != nil {
		return m.findPaginatedFn(f)
	}
	return &repository.PaginationResult[model.Client]{}, nil
}
func (m *mockClientRepo) SetStatusByUUID(id uuid.UUID, tID int64, s string) error { return nil }
func (m *mockClientRepo) DeleteByUUIDAndTenantID(id uuid.UUID, tID int64) error   { return nil }

// ---------------------------------------------------------------------------
// Mock: UserRepository
// ---------------------------------------------------------------------------

type mockUserRepo struct {
	findByUsernameFn         func(username string) (*model.User, error)
	findByEmailFn            func(email string) (*model.User, error)
	findByEmailAndTenantIDFn func(email string, tenantID int64) (*model.User, error)
	findByUUIDFn             func(id any, preloads ...string) (*model.User, error)
	findByIDFn               func(id any, preloads ...string) (*model.User, error)
	findSuperAdminFn         func() (*model.User, error)
	findPaginatedFn          func(repository.UserRepositoryGetFilter) (*repository.PaginationResult[model.User], error)
	createFn                 func(*model.User) (*model.User, error)
	updateByUUIDFn           func(id, data any) (*model.User, error)
	updateByIDFn             func(id, data any) (*model.User, error)
	findRolesFn              func(userID int64) ([]model.Role, error)
	findByPhoneFn            func(phone string) (*model.User, error)
	setStatusFn              func(id uuid.UUID, s string) error
	deleteByUUIDFn           func(id any) error
}

func (m *mockUserRepo) WithTx(_ *gorm.DB) repository.UserRepository { return m }
func (m *mockUserRepo) FindByUsername(u string) (*model.User, error) {
	if m.findByUsernameFn != nil {
		return m.findByUsernameFn(u)
	}
	return nil, nil
}
func (m *mockUserRepo) FindByEmail(e string) (*model.User, error) {
	if m.findByEmailFn != nil {
		return m.findByEmailFn(e)
	}
	return nil, nil
}
func (m *mockUserRepo) FindByEmailAndTenantID(e string, tID int64) (*model.User, error) {
	if m.findByEmailAndTenantIDFn != nil {
		return m.findByEmailAndTenantIDFn(e, tID)
	}
	return nil, nil
}
func (m *mockUserRepo) Create(e *model.User) (*model.User, error) {
	if m.createFn != nil {
		return m.createFn(e)
	}
	return e, nil
}
func (m *mockUserRepo) CreateOrUpdate(e *model.User) (*model.User, error) { return nil, nil }
func (m *mockUserRepo) FindAll(p ...string) ([]model.User, error)         { return nil, nil }
func (m *mockUserRepo) FindByUUID(id any, p ...string) (*model.User, error) {
	if m.findByUUIDFn != nil {
		return m.findByUUIDFn(id, p...)
	}
	return nil, nil
}
func (m *mockUserRepo) FindByUUIDs(ids []string, p ...string) ([]model.User, error) { return nil, nil }
func (m *mockUserRepo) FindByID(id any, p ...string) (*model.User, error) {
	if m.findByIDFn != nil {
		return m.findByIDFn(id, p...)
	}
	return nil, nil
}
func (m *mockUserRepo) UpdateByUUID(id, data any) (*model.User, error) {
	if m.updateByUUIDFn != nil {
		return m.updateByUUIDFn(id, data)
	}
	return nil, nil
}
func (m *mockUserRepo) UpdateByID(id, data any) (*model.User, error) {
	if m.updateByIDFn != nil {
		return m.updateByIDFn(id, data)
	}
	return nil, nil
}
func (m *mockUserRepo) DeleteByUUID(id any) error {
	if m.deleteByUUIDFn != nil {
		return m.deleteByUUIDFn(id)
	}
	return nil
}
func (m *mockUserRepo) DeleteByID(id any) error { return nil }
func (m *mockUserRepo) Paginate(c map[string]any, pg, lim int, p ...string) (*repository.PaginationResult[model.User], error) {
	return nil, nil
}
func (m *mockUserRepo) FindByPhone(phone string) (*model.User, error) {
	if m.findByPhoneFn != nil {
		return m.findByPhoneFn(phone)
	}
	return nil, nil
}
func (m *mockUserRepo) FindSuperAdmin() (*model.User, error) {
	if m.findSuperAdminFn != nil {
		return m.findSuperAdminFn()
	}
	return nil, nil
}
func (m *mockUserRepo) FindRoles(userID int64) ([]model.Role, error) {
	if m.findRolesFn != nil {
		return m.findRolesFn(userID)
	}
	return nil, nil
}
func (m *mockUserRepo) FindBySubAndClientID(sub, cID string) (*model.User, error) { return nil, nil }
func (m *mockUserRepo) FindPaginated(f repository.UserRepositoryGetFilter) (*repository.PaginationResult[model.User], error) {
	if m.findPaginatedFn != nil {
		return m.findPaginatedFn(f)
	}
	return &repository.PaginationResult[model.User]{}, nil
}
func (m *mockUserRepo) SetEmailVerified(id uuid.UUID, v bool) error { return nil }
func (m *mockUserRepo) SetStatus(id uuid.UUID, s string) error {
	if m.setStatusFn != nil {
		return m.setStatusFn(id, s)
	}
	return nil
}

// ---------------------------------------------------------------------------
// Mock: UserIdentityRepository
// ---------------------------------------------------------------------------

type mockUserIdentityRepo struct {
	findByUserIDAndClientIDFn func(userID, clientID int64) (*model.UserIdentity, error)
	createFn                  func(*model.UserIdentity) (*model.UserIdentity, error)
	findByUserIDFn            func(int64) ([]model.UserIdentity, error)
}

func (m *mockUserIdentityRepo) WithTx(_ *gorm.DB) repository.UserIdentityRepository { return m }
func (m *mockUserIdentityRepo) FindByUserIDAndClientID(uID, cID int64) (*model.UserIdentity, error) {
	return m.findByUserIDAndClientIDFn(uID, cID)
}
func (m *mockUserIdentityRepo) Create(e *model.UserIdentity) (*model.UserIdentity, error) {
	if m.createFn != nil {
		return m.createFn(e)
	}
	return e, nil
}
func (m *mockUserIdentityRepo) CreateOrUpdate(e *model.UserIdentity) (*model.UserIdentity, error) {
	return nil, nil
}
func (m *mockUserIdentityRepo) FindAll(p ...string) ([]model.UserIdentity, error) { return nil, nil }
func (m *mockUserIdentityRepo) FindByUUID(id any, p ...string) (*model.UserIdentity, error) {
	return nil, nil
}
func (m *mockUserIdentityRepo) FindByUUIDs(ids []string, p ...string) ([]model.UserIdentity, error) {
	return nil, nil
}
func (m *mockUserIdentityRepo) FindByID(id any, p ...string) (*model.UserIdentity, error) {
	return nil, nil
}
func (m *mockUserIdentityRepo) UpdateByUUID(id, data any) (*model.UserIdentity, error) {
	return nil, nil
}
func (m *mockUserIdentityRepo) UpdateByID(id, data any) (*model.UserIdentity, error) { return nil, nil }
func (m *mockUserIdentityRepo) DeleteByUUID(id any) error                            { return nil }
func (m *mockUserIdentityRepo) DeleteByID(id any) error                              { return nil }
func (m *mockUserIdentityRepo) Paginate(c map[string]any, pg, lim int, p ...string) (*repository.PaginationResult[model.UserIdentity], error) {
	return nil, nil
}
func (m *mockUserIdentityRepo) FindByUserID(uID int64) ([]model.UserIdentity, error) {
	if m.findByUserIDFn != nil {
		return m.findByUserIDFn(uID)
	}
	return nil, nil
}
func (m *mockUserIdentityRepo) FindByProviderAndUserID(prov, pUID string) (*model.UserIdentity, error) {
	return nil, nil
}
func (m *mockUserIdentityRepo) FindByEmail(e string) ([]model.UserIdentity, error) { return nil, nil }
func (m *mockUserIdentityRepo) DeleteByUserID(uID int64) error                     { return nil }

// ---------------------------------------------------------------------------
// Mock: IdentityProviderRepository
// ---------------------------------------------------------------------------

type mockIdentityProviderRepo struct {
	findByIdentifierFn func(identifier string) (*model.IdentityProvider, error)
	findByUUIDFn       func(id any, preloads ...string) (*model.IdentityProvider, error)
	findByNameFn       func(name string, tenantID int64) (*model.IdentityProvider, error)
	findPaginatedFn    func(repository.IdentityProviderRepositoryGetFilter) (*repository.PaginationResult[model.IdentityProvider], error)
	createOrUpdateFn   func(*model.IdentityProvider) (*model.IdentityProvider, error)
	deleteByUUIDFn     func(id any) error
}

func (m *mockIdentityProviderRepo) WithTx(_ *gorm.DB) repository.IdentityProviderRepository { return m }
func (m *mockIdentityProviderRepo) FindByIdentifier(id string) (*model.IdentityProvider, error) {
	if m.findByIdentifierFn != nil {
		return m.findByIdentifierFn(id)
	}
	return nil, nil
}
func (m *mockIdentityProviderRepo) Create(e *model.IdentityProvider) (*model.IdentityProvider, error) {
	return nil, nil
}
func (m *mockIdentityProviderRepo) CreateOrUpdate(e *model.IdentityProvider) (*model.IdentityProvider, error) {
	if m.createOrUpdateFn != nil {
		return m.createOrUpdateFn(e)
	}
	return e, nil
}
func (m *mockIdentityProviderRepo) FindAll(p ...string) ([]model.IdentityProvider, error) {
	return nil, nil
}
func (m *mockIdentityProviderRepo) FindByUUID(id any, p ...string) (*model.IdentityProvider, error) {
	if m.findByUUIDFn != nil {
		return m.findByUUIDFn(id, p...)
	}
	return nil, nil
}
func (m *mockIdentityProviderRepo) FindByUUIDs(ids []string, p ...string) ([]model.IdentityProvider, error) {
	return nil, nil
}
func (m *mockIdentityProviderRepo) FindByID(id any, p ...string) (*model.IdentityProvider, error) {
	return nil, nil
}
func (m *mockIdentityProviderRepo) UpdateByUUID(id, data any) (*model.IdentityProvider, error) {
	return nil, nil
}
func (m *mockIdentityProviderRepo) UpdateByID(id, data any) (*model.IdentityProvider, error) {
	return nil, nil
}
func (m *mockIdentityProviderRepo) DeleteByUUID(id any) error {
	if m.deleteByUUIDFn != nil {
		return m.deleteByUUIDFn(id)
	}
	return nil
}
func (m *mockIdentityProviderRepo) DeleteByID(id any) error { return nil }
func (m *mockIdentityProviderRepo) Paginate(c map[string]any, pg, lim int, p ...string) (*repository.PaginationResult[model.IdentityProvider], error) {
	return nil, nil
}
func (m *mockIdentityProviderRepo) FindByName(n string, tID int64) (*model.IdentityProvider, error) {
	if m.findByNameFn != nil {
		return m.findByNameFn(n, tID)
	}
	return nil, nil
}
func (m *mockIdentityProviderRepo) FindDefaultByTenantID(tID int64) (*model.IdentityProvider, error) {
	return nil, nil
}
func (m *mockIdentityProviderRepo) FindPaginated(f repository.IdentityProviderRepositoryGetFilter) (*repository.PaginationResult[model.IdentityProvider], error) {
	if m.findPaginatedFn != nil {
		return m.findPaginatedFn(f)
	}
	return &repository.PaginationResult[model.IdentityProvider]{}, nil
}

// ---------------------------------------------------------------------------
// Mock: UserTokenRepository
// ---------------------------------------------------------------------------

type mockUserTokenRepo struct {
	createFn                   func(*model.UserToken) (*model.UserToken, error)
	findByUserIDAndTokenTypeFn func(userID int64, tokenType string) ([]model.UserToken, error)
	revokeByUUIDFn             func(id uuid.UUID) error
}

func (m *mockUserTokenRepo) WithTx(_ *gorm.DB) repository.UserTokenRepository { return m }
func (m *mockUserTokenRepo) Create(e *model.UserToken) (*model.UserToken, error) {
	if m.createFn != nil {
		return m.createFn(e)
	}
	return e, nil
}
func (m *mockUserTokenRepo) CreateOrUpdate(e *model.UserToken) (*model.UserToken, error) {
	return nil, nil
}
func (m *mockUserTokenRepo) FindAll(p ...string) ([]model.UserToken, error) { return nil, nil }
func (m *mockUserTokenRepo) FindByUUID(id any, p ...string) (*model.UserToken, error) {
	return nil, nil
}
func (m *mockUserTokenRepo) FindByUUIDs(ids []string, p ...string) ([]model.UserToken, error) {
	return nil, nil
}
func (m *mockUserTokenRepo) FindByID(id any, p ...string) (*model.UserToken, error) { return nil, nil }
func (m *mockUserTokenRepo) UpdateByUUID(id, data any) (*model.UserToken, error)    { return nil, nil }
func (m *mockUserTokenRepo) UpdateByID(id, data any) (*model.UserToken, error)      { return nil, nil }
func (m *mockUserTokenRepo) DeleteByUUID(id any) error                              { return nil }
func (m *mockUserTokenRepo) DeleteByID(id any) error                                { return nil }
func (m *mockUserTokenRepo) Paginate(c map[string]any, pg, lim int, p ...string) (*repository.PaginationResult[model.UserToken], error) {
	return nil, nil
}
func (m *mockUserTokenRepo) FindByUserID(uID int64) ([]model.UserToken, error) { return nil, nil }
func (m *mockUserTokenRepo) FindActiveTokensByUserID(uID int64) ([]model.UserToken, error) {
	return nil, nil
}
func (m *mockUserTokenRepo) FindByUserIDAndTokenType(uID int64, tt string) ([]model.UserToken, error) {
	if m.findByUserIDAndTokenTypeFn != nil {
		return m.findByUserIDAndTokenTypeFn(uID, tt)
	}
	return nil, nil
}
func (m *mockUserTokenRepo) RevokeByUUID(id uuid.UUID) error {
	if m.revokeByUUIDFn != nil {
		return m.revokeByUUIDFn(id)
	}
	return nil
}
func (m *mockUserTokenRepo) RevokeAllByUserID(uID int64) error          { return nil }
func (m *mockUserTokenRepo) DeleteByUserID(uID int64) error             { return nil }
func (m *mockUserTokenRepo) DeleteExpiredTokens(before time.Time) error { return nil }

// ---------------------------------------------------------------------------
// Test helpers
// ---------------------------------------------------------------------------

// initTestJWTKeysService generates a fresh RSA-2048 key pair and wires it into
// the package-level config variables that GenerateAccessToken / GenerateIDToken
// / GenerateRefreshToken read from.
func initTestJWTKeysService(t *testing.T) {
	t.Helper()
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	privPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(priv)})
	pubPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PUBLIC KEY", Bytes: x509.MarshalPKCS1PublicKey(&priv.PublicKey)})
	config.JWTPrivateKey = privPEM
	config.JWTPublicKey = pubPEM
	require.NoError(t, jwt.InitJWTKeys())
}

// strPtr returns a pointer to the given string literal — handy for Client fields.
func strPtr(s string) *string { return &s }

// newMockGormDB creates a *gorm.DB backed by sqlmock so service tests can
// verify BEGIN / COMMIT / ROLLBACK without a real database.
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

// buildActiveIdentityProvider returns a minimal active identity provider for tests.
func buildActiveIdentityProvider() *model.IdentityProvider {
	return &model.IdentityProvider{
		IdentityProviderID: 1,
		Name:               "default",
		Provider:           model.IDPProviderInternal,
		ProviderType:       model.IDPTypeIdentity,
		Identifier:         "test-provider",
		Status:             model.StatusActive,
	}
}

// buildActiveClient returns a minimal active client whose Domain and Identifier
// are both populated (required by generateTokenResponse).
func buildActiveClient() *model.Client {
	idp := buildActiveIdentityProvider()
	return &model.Client{
		ClientID:         1,
		Name:             "test-client",
		Domain:           strPtr("https://auth.example.com"),
		Identifier:       strPtr("test-client"),
		Status:           model.StatusActive,
		IdentityProvider: idp,
	}
}

// buildActiveUser bcrypt-hashes the given plaintext password and returns an
// active user that the service can authenticate successfully.
func buildActiveUser(t *testing.T, password string) *model.User {
	t.Helper()
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.MinCost)
	require.NoError(t, err)
	hashStr := string(hash)
	return &model.User{
		UserID:   1,
		UserUUID: uuid.New(),
		Username: "testuser",
		Email:    "testuser@example.com",
		Password: &hashStr,
		Status:   model.StatusActive,
	}
}

// ---------------------------------------------------------------------------
// TestGetUserByEmail
// ---------------------------------------------------------------------------

func TestGetUserByEmail(t *testing.T) {
	cases := []struct {
		name      string
		email     string
		tenantID  int64
		setupRepo func(m *mockUserRepo)
		wantErr   bool
	}{
		{
			name:     "global lookup (tenantID=0) returns user",
			email:    "a@b.com",
			tenantID: 0,
			setupRepo: func(m *mockUserRepo) {
				m.findByEmailFn = func(e string) (*model.User, error) { return &model.User{Email: e}, nil }
			},
		},
		{
			name:     "tenant-scoped lookup returns user",
			email:    "a@b.com",
			tenantID: 42,
			setupRepo: func(m *mockUserRepo) {
				m.findByEmailAndTenantIDFn = func(e string, _ int64) (*model.User, error) {
					return &model.User{Email: e}, nil
				}
			},
		},
		{
			name:     "global lookup - user not found",
			email:    "nope@x.com",
			tenantID: 0,
			setupRepo: func(m *mockUserRepo) {
				m.findByEmailFn = func(_ string) (*model.User, error) { return nil, errors.New("not found") }
			},
			wantErr: true,
		},
		{
			name:     "tenant-scoped lookup - user not found",
			email:    "nope@x.com",
			tenantID: 5,
			setupRepo: func(m *mockUserRepo) {
				m.findByEmailAndTenantIDFn = func(_ string, _ int64) (*model.User, error) {
					return nil, errors.New("not found")
				}
			},
			wantErr: true,
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			userRepo := &mockUserRepo{}
			tc.setupRepo(userRepo)
			svc := &loginService{userRepo: userRepo}
			got, err := svc.GetUserByEmail(context.Background(), tc.email, tc.tenantID)
			if tc.wantErr {
				assert.Error(t, err)
				assert.Nil(t, got)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, got)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// TestLoginPublic
// ---------------------------------------------------------------------------

func TestLoginPublic(t *testing.T) {
	const correctPassword = "S3cur3P@ss!"

	type repoSetup struct {
		clientRepo   *mockClientRepo
		userRepo     *mockUserRepo
		userIdentity *mockUserIdentityRepo
		idpRepo      *mockIdentityProviderRepo
	}

	cases := []struct {
		name           string
		username       string // unique per case to avoid rate-limiter cross-talk
		password       string
		clientID       string
		providerID     string
		setup          func(t *testing.T, r repoSetup)
		expectCommit   bool // false → expect rollback (callback returned error)
		wantErr        bool
		wantErrContain string
	}{
		{
			name:         "success",
			username:     "pub-success",
			password:     correctPassword,
			clientID:     "client-1",
			providerID:   "provider-1",
			expectCommit: true,
			setup: func(t *testing.T, r repoSetup) {
				r.idpRepo.findByIdentifierFn = func(_ string) (*model.IdentityProvider, error) {
					return buildActiveIdentityProvider(), nil
				}
				r.clientRepo.findByClientIDAndIdentityProviderFn = func(_, _ string) (*model.Client, error) {
					return buildActiveClient(), nil
				}
				r.userRepo.findByUsernameFn = func(_ string) (*model.User, error) {
					return buildActiveUser(t, correctPassword), nil
				}
				r.userIdentity.findByUserIDAndClientIDFn = func(_, _ int64) (*model.UserIdentity, error) {
					return &model.UserIdentity{Sub: "sub-123"}, nil
				}
			},
		},
		{
			name:           "identity provider not found",
			username:       "pub-no-idp",
			password:       correctPassword,
			clientID:       "client-1",
			providerID:     "provider-1",
			expectCommit:   false,
			wantErr:        true,
			wantErrContain: "authentication failed",
			setup: func(t *testing.T, r repoSetup) {
				r.idpRepo.findByIdentifierFn = func(_ string) (*model.IdentityProvider, error) {
					return nil, errors.New("db error")
				}
			},
		},
		{
			name:           "identity provider returns nil",
			username:       "pub-nil-idp",
			password:       correctPassword,
			clientID:       "client-1",
			providerID:     "provider-1",
			expectCommit:   false,
			wantErr:        true,
			wantErrContain: "authentication failed",
			setup: func(t *testing.T, r repoSetup) {
				r.idpRepo.findByIdentifierFn = func(_ string) (*model.IdentityProvider, error) {
					return nil, nil
				}
			},
		},
		{
			name:           "client inactive",
			username:       "pub-inactive-client",
			password:       correctPassword,
			clientID:       "client-1",
			providerID:     "provider-1",
			expectCommit:   false,
			wantErr:        true,
			wantErrContain: "authentication failed",
			setup: func(t *testing.T, r repoSetup) {
				r.idpRepo.findByIdentifierFn = func(_ string) (*model.IdentityProvider, error) {
					return buildActiveIdentityProvider(), nil
				}
				r.clientRepo.findByClientIDAndIdentityProviderFn = func(_, _ string) (*model.Client, error) {
					c := buildActiveClient()
					c.Status = model.StatusInactive
					return c, nil
				}
			},
		},
		{
			name:           "wrong password",
			username:       "pub-wrong-pass",
			password:       "W0ngP@ss!",
			clientID:       "client-1",
			providerID:     "provider-1",
			expectCommit:   true,
			wantErr:        true,
			wantErrContain: "invalid credentials",
			setup: func(t *testing.T, r repoSetup) {
				r.idpRepo.findByIdentifierFn = func(_ string) (*model.IdentityProvider, error) {
					return buildActiveIdentityProvider(), nil
				}
				r.clientRepo.findByClientIDAndIdentityProviderFn = func(_, _ string) (*model.Client, error) {
					return buildActiveClient(), nil
				}
				r.userRepo.findByUsernameFn = func(_ string) (*model.User, error) {
					return buildActiveUser(t, correctPassword), nil
				}
				r.userIdentity.findByUserIDAndClientIDFn = func(_, _ int64) (*model.UserIdentity, error) {
					return nil, errors.New("not found")
				}
			},
		},
		{
			name:           "user account inactive",
			username:       "pub-inactive-user",
			password:       correctPassword,
			clientID:       "client-1",
			providerID:     "provider-1",
			expectCommit:   true,
			wantErr:        true,
			wantErrContain: "account is not active",
			setup: func(t *testing.T, r repoSetup) {
				r.idpRepo.findByIdentifierFn = func(_ string) (*model.IdentityProvider, error) {
					return buildActiveIdentityProvider(), nil
				}
				r.clientRepo.findByClientIDAndIdentityProviderFn = func(_, _ string) (*model.Client, error) {
					return buildActiveClient(), nil
				}
				r.userRepo.findByUsernameFn = func(_ string) (*model.User, error) {
					u := buildActiveUser(t, correctPassword)
					u.Status = model.StatusInactive
					return u, nil
				}
				r.userIdentity.findByUserIDAndClientIDFn = func(_, _ int64) (*model.UserIdentity, error) {
					return &model.UserIdentity{Sub: "sub-123"}, nil
				}
			},
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			initTestJWTKeysService(t)
			gormDB, mock := newMockGormDB(t)
			mock.ExpectBegin()
			if tc.expectCommit {
				mock.ExpectCommit()
			} else {
				mock.ExpectRollback()
			}

			repos := repoSetup{
				clientRepo:   &mockClientRepo{},
				userRepo:     &mockUserRepo{},
				userIdentity: &mockUserIdentityRepo{},
				idpRepo:      &mockIdentityProviderRepo{},
			}
			tc.setup(t, repos)

			svc := NewLoginService(gormDB, repos.clientRepo, repos.userRepo, &mockUserTokenRepo{}, repos.userIdentity, repos.idpRepo)
			resp, err := svc.LoginPublic(context.Background(), tc.username, tc.password, tc.clientID, tc.providerID)

			if tc.wantErr {
				require.Error(t, err)
				assert.Nil(t, resp)
				if tc.wantErrContain != "" {
					assert.Contains(t, err.Error(), tc.wantErrContain)
				}
			} else {
				require.NoError(t, err)
				assert.NotNil(t, resp)
				assert.NotEmpty(t, resp.AccessToken)
				assert.NotEmpty(t, resp.IDToken)
				assert.NotEmpty(t, resp.RefreshToken)
				assert.Equal(t, "Bearer", resp.TokenType)
			}
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

// ---------------------------------------------------------------------------
// TestLogin
// ---------------------------------------------------------------------------

func TestLogin(t *testing.T) {
	const correctPassword = "S3cur3P@ss!"

	type repoSetup struct {
		clientRepo   *mockClientRepo
		userRepo     *mockUserRepo
		userIdentity *mockUserIdentityRepo
	}

	cases := []struct {
		name           string
		username       string
		password       string
		clientID       *string
		providerID     *string
		setup          func(t *testing.T, r repoSetup)
		expectCommit   bool
		wantErr        bool
		wantErrContain string
	}{
		{
			name:         "success with default client",
			username:     "int-success-default",
			password:     correctPassword,
			clientID:     nil,
			providerID:   nil,
			expectCommit: true,
			setup: func(t *testing.T, r repoSetup) {
				r.clientRepo.findDefaultFn = func() (*model.Client, error) { return buildActiveClient(), nil }
				r.userRepo.findByUsernameFn = func(_ string) (*model.User, error) {
					return buildActiveUser(t, correctPassword), nil
				}
				r.userIdentity.findByUserIDAndClientIDFn = func(_, _ int64) (*model.UserIdentity, error) {
					return &model.UserIdentity{Sub: "sub-456"}, nil
				}
			},
		},
		{
			name:         "success with explicit client",
			username:     "int-success-explicit",
			password:     correctPassword,
			clientID:     strPtr("client-2"),
			providerID:   strPtr("provider-2"),
			expectCommit: true,
			setup: func(t *testing.T, r repoSetup) {
				r.clientRepo.findByClientIDAndIdentityProviderFn = func(_, _ string) (*model.Client, error) {
					return buildActiveClient(), nil
				}
				r.userRepo.findByUsernameFn = func(_ string) (*model.User, error) {
					return buildActiveUser(t, correctPassword), nil
				}
				r.userIdentity.findByUserIDAndClientIDFn = func(_, _ int64) (*model.UserIdentity, error) {
					return &model.UserIdentity{Sub: "sub-789"}, nil
				}
			},
		},
		{
			name:           "default client lookup fails",
			username:       "int-no-client",
			password:       correctPassword,
			clientID:       nil,
			providerID:     nil,
			expectCommit:   false,
			wantErr:        true,
			wantErrContain: "authentication failed",
			setup: func(t *testing.T, r repoSetup) {
				r.clientRepo.findDefaultFn = func() (*model.Client, error) { return nil, errors.New("db error") }
			},
		},
		{
			name:           "default client inactive",
			username:       "int-inactive-client",
			password:       correctPassword,
			clientID:       nil,
			providerID:     nil,
			expectCommit:   false,
			wantErr:        true,
			wantErrContain: "authentication failed",
			setup: func(t *testing.T, r repoSetup) {
				r.clientRepo.findDefaultFn = func() (*model.Client, error) {
					c := buildActiveClient()
					c.Status = model.StatusInactive
					return c, nil
				}
			},
		},
		{
			name:           "wrong password",
			username:       "int-wrong-pass",
			password:       "W0ngP@ss!",
			clientID:       nil,
			providerID:     nil,
			expectCommit:   true,
			wantErr:        true,
			wantErrContain: "invalid credentials",
			setup: func(t *testing.T, r repoSetup) {
				r.clientRepo.findDefaultFn = func() (*model.Client, error) { return buildActiveClient(), nil }
				r.userRepo.findByUsernameFn = func(_ string) (*model.User, error) {
					return buildActiveUser(t, correctPassword), nil
				}
				r.userIdentity.findByUserIDAndClientIDFn = func(_, _ int64) (*model.UserIdentity, error) {
					return nil, errors.New("not found")
				}
			},
		},
		{
			name:           "user account inactive",
			username:       "int-inactive-user",
			password:       correctPassword,
			clientID:       nil,
			providerID:     nil,
			expectCommit:   true,
			wantErr:        true,
			wantErrContain: "account is not active",
			setup: func(t *testing.T, r repoSetup) {
				r.clientRepo.findDefaultFn = func() (*model.Client, error) { return buildActiveClient(), nil }
				r.userRepo.findByUsernameFn = func(_ string) (*model.User, error) {
					u := buildActiveUser(t, correctPassword)
					u.Status = model.StatusInactive
					return u, nil
				}
				r.userIdentity.findByUserIDAndClientIDFn = func(_, _ int64) (*model.UserIdentity, error) {
					return &model.UserIdentity{Sub: "sub-456"}, nil
				}
			},
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			initTestJWTKeysService(t)
			gormDB, mock := newMockGormDB(t)
			mock.ExpectBegin()
			if tc.expectCommit {
				mock.ExpectCommit()
			} else {
				mock.ExpectRollback()
			}

			repos := repoSetup{
				clientRepo:   &mockClientRepo{},
				userRepo:     &mockUserRepo{},
				userIdentity: &mockUserIdentityRepo{},
			}
			tc.setup(t, repos)

			svc := NewLoginService(gormDB, repos.clientRepo, repos.userRepo, &mockUserTokenRepo{}, repos.userIdentity, &mockIdentityProviderRepo{})
			resp, err := svc.Login(context.Background(), tc.username, tc.password, tc.clientID, tc.providerID)

			if tc.wantErr {
				require.Error(t, err)
				assert.Nil(t, resp)
				if tc.wantErrContain != "" {
					assert.Contains(t, err.Error(), tc.wantErrContain)
				}
			} else {
				require.NoError(t, err)
				assert.NotNil(t, resp)
				assert.NotEmpty(t, resp.AccessToken)
				assert.NotEmpty(t, resp.IDToken)
				assert.NotEmpty(t, resp.RefreshToken)
				assert.Equal(t, "Bearer", resp.TokenType)
			}
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

// ---------------------------------------------------------------------------
// lockedRateLimiterLogin starts a miniredis instance, pre-sets the lock key
// for the given identifier, wires it into util.CheckRateLimit, and returns a
// cleanup function that resets the rate limiter to nil after the test.
// ---------------------------------------------------------------------------
func lockedRateLimiterLogin(t *testing.T, identifier string) func() {
	t.Helper()
	mr, err := miniredis.Run()
	require.NoError(t, err)

	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	security.InitRateLimiter(rdb)

	// Pre-set the lock key so CheckRateLimit returns an error immediately.
	require.NoError(t, mr.Set("rl:lock:"+identifier, "1"))

	return func() {
		security.InitRateLimiter(nil)
		rdb.Close()
		mr.Close()
	}
}

// ---------------------------------------------------------------------------
// TestLoginPublic – additional cases
// ---------------------------------------------------------------------------

func TestLoginPublic_RateLimited(t *testing.T) {
	username := "pub-rate-limited"
	cleanup := lockedRateLimiterLogin(t, username)
	defer cleanup()

	gormDB, mock := newMockGormDB(t)
	// No DB operations expected — rate limit fires before transaction
	_ = mock

	svc := NewLoginService(gormDB, &mockClientRepo{}, &mockUserRepo{}, &mockUserTokenRepo{},
		&mockUserIdentityRepo{findByUserIDAndClientIDFn: func(_, _ int64) (*model.UserIdentity, error) { return nil, nil }},
		&mockIdentityProviderRepo{})
	_, err := svc.LoginPublic(context.Background(), username, "pass", "c1", "p1")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "locked")
}

func TestLoginPublic_ClientLookupError(t *testing.T) {
	initTestJWTKeysService(t)
	gormDB, mock := newMockGormDB(t)
	mock.ExpectBegin()
	mock.ExpectRollback()

	idpRepo := &mockIdentityProviderRepo{
		findByIdentifierFn: func(_ string) (*model.IdentityProvider, error) {
			return buildActiveIdentityProvider(), nil
		},
	}
	clientRepo := &mockClientRepo{
		findByClientIDAndIdentityProviderFn: func(_, _ string) (*model.Client, error) {
			return nil, errors.New("client db err")
		},
	}

	svc := NewLoginService(gormDB, clientRepo, &mockUserRepo{}, &mockUserTokenRepo{},
		&mockUserIdentityRepo{findByUserIDAndClientIDFn: func(_, _ int64) (*model.UserIdentity, error) { return nil, nil }},
		idpRepo)
	_, err := svc.LoginPublic(context.Background(), "pub-client-err", "pass", "c1", "p1")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "authentication failed")
}

func TestLoginPublic_UserNotFound(t *testing.T) {
	initTestJWTKeysService(t)
	gormDB, mock := newMockGormDB(t)
	mock.ExpectBegin()
	mock.ExpectCommit()

	idpRepo := &mockIdentityProviderRepo{
		findByIdentifierFn: func(_ string) (*model.IdentityProvider, error) {
			return buildActiveIdentityProvider(), nil
		},
	}
	clientRepo := &mockClientRepo{
		findByClientIDAndIdentityProviderFn: func(_, _ string) (*model.Client, error) {
			return buildActiveClient(), nil
		},
	}
	userRepo := &mockUserRepo{
		findByUsernameFn: func(_ string) (*model.User, error) {
			return nil, errors.New("not found")
		},
	}

	svc := NewLoginService(gormDB, clientRepo, userRepo, &mockUserTokenRepo{},
		&mockUserIdentityRepo{findByUserIDAndClientIDFn: func(_, _ int64) (*model.UserIdentity, error) { return nil, nil }},
		idpRepo)
	_, err := svc.LoginPublic(context.Background(), "pub-user-missing", "pass", "c1", "p1")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid credentials")
}

// ---------------------------------------------------------------------------
// TestLogin – additional cases
// ---------------------------------------------------------------------------

func TestLogin_RateLimited(t *testing.T) {
	username := "int-rate-limited"
	cleanup := lockedRateLimiterLogin(t, username)
	defer cleanup()

	gormDB, mock := newMockGormDB(t)
	_ = mock

	svc := NewLoginService(gormDB, &mockClientRepo{}, &mockUserRepo{}, &mockUserTokenRepo{},
		&mockUserIdentityRepo{findByUserIDAndClientIDFn: func(_, _ int64) (*model.UserIdentity, error) { return nil, nil }},
		&mockIdentityProviderRepo{})
	_, err := svc.Login(context.Background(), username, "pass", nil, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "locked")
}

func TestLogin_ExplicitClientLookupError(t *testing.T) {
	initTestJWTKeysService(t)
	gormDB, mock := newMockGormDB(t)
	mock.ExpectBegin()
	mock.ExpectRollback()

	clientRepo := &mockClientRepo{
		findByClientIDAndIdentityProviderFn: func(_, _ string) (*model.Client, error) {
			return nil, errors.New("client db err")
		},
	}

	cID := "client-x"
	pID := "provider-x"
	svc := NewLoginService(gormDB, clientRepo, &mockUserRepo{}, &mockUserTokenRepo{},
		&mockUserIdentityRepo{findByUserIDAndClientIDFn: func(_, _ int64) (*model.UserIdentity, error) { return nil, nil }},
		&mockIdentityProviderRepo{})
	_, err := svc.Login(context.Background(), "int-explicit-err", "pass", &cID, &pID)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "authentication failed")
}

func TestLogin_UserNotFound(t *testing.T) {
	initTestJWTKeysService(t)
	gormDB, mock := newMockGormDB(t)
	mock.ExpectBegin()
	mock.ExpectCommit()

	clientRepo := &mockClientRepo{
		findDefaultFn: func() (*model.Client, error) { return buildActiveClient(), nil },
	}
	userRepo := &mockUserRepo{
		findByUsernameFn: func(_ string) (*model.User, error) { return nil, nil },
	}

	svc := NewLoginService(gormDB, clientRepo, userRepo, &mockUserTokenRepo{},
		&mockUserIdentityRepo{findByUserIDAndClientIDFn: func(_, _ int64) (*model.UserIdentity, error) { return nil, nil }},
		&mockIdentityProviderRepo{})
	_, err := svc.Login(context.Background(), "int-user-missing", "pass", nil, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid credentials")
}

// ---------------------------------------------------------------------------
// TestGenerateTokenResponse – error paths
// ---------------------------------------------------------------------------

func TestLoginPublic_GenerateAccessTokenError(t *testing.T) {
	// Reset JWT keys so privateKey is nil → GenerateAccessToken fails
	jwt.ResetJWTKeys()
	defer initTestJWTKeysService(t) // restore for subsequent tests

	gormDB, mock := newMockGormDB(t)
	mock.ExpectBegin()
	mock.ExpectCommit()

	const correctPassword = "S3cur3P@ss!"

	idpRepo := &mockIdentityProviderRepo{
		findByIdentifierFn: func(_ string) (*model.IdentityProvider, error) {
			return buildActiveIdentityProvider(), nil
		},
	}
	clientRepo := &mockClientRepo{
		findByClientIDAndIdentityProviderFn: func(_, _ string) (*model.Client, error) {
			return buildActiveClient(), nil
		},
	}
	userRepo := &mockUserRepo{
		findByUsernameFn: func(_ string) (*model.User, error) {
			return buildActiveUser(t, correctPassword), nil
		},
	}
	userIdentityRepo := &mockUserIdentityRepo{
		findByUserIDAndClientIDFn: func(_, _ int64) (*model.UserIdentity, error) {
			return &model.UserIdentity{Sub: "sub-token-err"}, nil
		},
	}

	svc := NewLoginService(gormDB, clientRepo, userRepo, &mockUserTokenRepo{}, userIdentityRepo, idpRepo)
	_, err := svc.LoginPublic(context.Background(), "pub-token-err", correctPassword, "c1", "p1")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "private key not initialized")
}

func TestLogin_GenerateAccessTokenError(t *testing.T) {
	// Reset JWT keys so privateKey is nil
	jwt.ResetJWTKeys()
	defer initTestJWTKeysService(t) // restore for subsequent tests

	gormDB, mock := newMockGormDB(t)
	mock.ExpectBegin()
	mock.ExpectCommit()

	const correctPassword = "S3cur3P@ss!"

	clientRepo := &mockClientRepo{
		findDefaultFn: func() (*model.Client, error) { return buildActiveClient(), nil },
	}
	userRepo := &mockUserRepo{
		findByUsernameFn: func(_ string) (*model.User, error) {
			return buildActiveUser(t, correctPassword), nil
		},
	}
	userIdentityRepo := &mockUserIdentityRepo{
		findByUserIDAndClientIDFn: func(_, _ int64) (*model.UserIdentity, error) {
			return &model.UserIdentity{Sub: "sub-token-err"}, nil
		},
	}

	svc := NewLoginService(gormDB, clientRepo, userRepo, &mockUserTokenRepo{}, userIdentityRepo, &mockIdentityProviderRepo{})
	_, err := svc.Login(context.Background(), "int-token-err", correctPassword, nil, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "private key not initialized")
}

func TestLoginPublic_GenerateIDTokenError(t *testing.T) {
	initTestJWTKeysService(t)
	gormDB, mock := newMockGormDB(t)
	mock.ExpectBegin()
	mock.ExpectCommit()

	const correctPassword = "S3cur3P@ss!"

	// Stub generateIDTokenFn to return an error
	orig := generateIDTokenFn
	generateIDTokenFn = func(string, string, string, string, *jwt.UserProfile, string) (string, error) {
		return "", errors.New("id token error")
	}
	defer func() { generateIDTokenFn = orig }()

	idpRepo := &mockIdentityProviderRepo{
		findByIdentifierFn: func(_ string) (*model.IdentityProvider, error) {
			return buildActiveIdentityProvider(), nil
		},
	}
	clientRepo := &mockClientRepo{
		findByClientIDAndIdentityProviderFn: func(_, _ string) (*model.Client, error) {
			return buildActiveClient(), nil
		},
	}
	userRepo := &mockUserRepo{
		findByUsernameFn: func(_ string) (*model.User, error) {
			return buildActiveUser(t, correctPassword), nil
		},
	}
	userIdentityRepo := &mockUserIdentityRepo{
		findByUserIDAndClientIDFn: func(_, _ int64) (*model.UserIdentity, error) {
			return &model.UserIdentity{Sub: "sub-idtoken-err"}, nil
		},
	}

	svc := NewLoginService(gormDB, clientRepo, userRepo, &mockUserTokenRepo{}, userIdentityRepo, idpRepo)
	_, err := svc.LoginPublic(context.Background(), "pub-idtoken-err", correctPassword, "c1", "p1")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "id token error")
}

func TestLoginPublic_GenerateRefreshTokenError(t *testing.T) {
	initTestJWTKeysService(t)
	gormDB, mock := newMockGormDB(t)
	mock.ExpectBegin()
	mock.ExpectCommit()

	const correctPassword = "S3cur3P@ss!"

	// Stub generateRefreshTokenFn to return an error
	orig := generateRefreshTokenFn
	generateRefreshTokenFn = func(string, string, string, string) (string, error) {
		return "", errors.New("refresh token error")
	}
	defer func() { generateRefreshTokenFn = orig }()

	idpRepo := &mockIdentityProviderRepo{
		findByIdentifierFn: func(_ string) (*model.IdentityProvider, error) {
			return buildActiveIdentityProvider(), nil
		},
	}
	clientRepo := &mockClientRepo{
		findByClientIDAndIdentityProviderFn: func(_, _ string) (*model.Client, error) {
			return buildActiveClient(), nil
		},
	}
	userRepo := &mockUserRepo{
		findByUsernameFn: func(_ string) (*model.User, error) {
			return buildActiveUser(t, correctPassword), nil
		},
	}
	userIdentityRepo := &mockUserIdentityRepo{
		findByUserIDAndClientIDFn: func(_, _ int64) (*model.UserIdentity, error) {
			return &model.UserIdentity{Sub: "sub-refresh-err"}, nil
		},
	}

	svc := NewLoginService(gormDB, clientRepo, userRepo, &mockUserTokenRepo{}, userIdentityRepo, idpRepo)
	_, err := svc.LoginPublic(context.Background(), "pub-refresh-err", correctPassword, "c1", "p1")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "refresh token error")
}
