package authn

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"errors"
	"testing"
	"time"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/platform/config"
	"github.com/maintainerd/auth/internal/platform/jwt"
	"github.com/maintainerd/auth/internal/platform/security"
	"github.com/maintainerd/auth/internal/secpolicy"
	"github.com/maintainerd/auth/internal/shared"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/datatypes"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// ---------------------------------------------------------------------------
// Mock: ClientRepository
// ---------------------------------------------------------------------------

type mockClientRepo struct {
	findByClientIDAndIdentityProviderFn func(clientID, providerID string) (*Client, error)
	findByIdentifierFn                  func(string) (*Client, error)
	findSystemByTenantIdentifierFn      func(string) (*Client, error)
	findSystemFn                        func() (*Client, error)
	findByUUIDFn                        func(any, ...string) (*Client, error)
	findByUUIDAndTenantIDFn             func(uuid.UUID, int64) (*Client, error)
	findPaginatedFn                     func(ClientRepositoryGetFilter) (*PaginationResult[Client], error)
	findByNameAndIdentityProviderFn     func(string, int64, int64) (*Client, error)
	findByNameAndTenantIDFn             func(string, int64) (*Client, error)
	findDefaultByTenantIDFn             func(tID int64) (*Client, error)
	createOrUpdateFn                    func(*Client) (*Client, error)
	deleteByUUIDFn                      func(any) error
	findByIDFn                          func(any, ...string) (*Client, error)
}

func (m *mockClientRepo) WithTx(_ *gorm.DB) ClientRepository { return m }
func (m *mockClientRepo) FindByClientIDAndIdentityProvider(a, b string) (*Client, error) {
	if m.findByClientIDAndIdentityProviderFn != nil {
		return m.findByClientIDAndIdentityProviderFn(a, b)
	}
	return nil, nil
}
func (m *mockClientRepo) FindByIdentifier(identifier string) (*Client, error) {
	if m.findByIdentifierFn != nil {
		return m.findByIdentifierFn(identifier)
	}
	if m.findByClientIDAndIdentityProviderFn != nil {
		return m.findByClientIDAndIdentityProviderFn(identifier, "")
	}
	return nil, nil
}
func (m *mockClientRepo) FindSystemByTenantIdentifier(tenantIdentifier string) (*Client, error) {
	if m.findSystemByTenantIdentifierFn != nil {
		return m.findSystemByTenantIdentifierFn(tenantIdentifier)
	}
	return nil, nil
}
func (m *mockClientRepo) FindSystem() (*Client, error) {
	if m.findSystemFn != nil {
		return m.findSystemFn()
	}
	return nil, nil
}
func (m *mockClientRepo) Create(e *Client) (*Client, error) { return e, nil }
func (m *mockClientRepo) CreateOrUpdate(e *Client) (*Client, error) {
	if m.createOrUpdateFn != nil {
		return m.createOrUpdateFn(e)
	}
	return e, nil
}
func (m *mockClientRepo) FindAll(p ...string) ([]Client, error) { return nil, nil }
func (m *mockClientRepo) FindByUUID(id any, p ...string) (*Client, error) {
	if m.findByUUIDFn != nil {
		return m.findByUUIDFn(id, p...)
	}
	return nil, nil
}
func (m *mockClientRepo) FindByUUIDs(ids []string, p ...string) ([]Client, error) {
	return nil, nil
}
func (m *mockClientRepo) FindByID(id any, p ...string) (*Client, error) {
	if m.findByIDFn != nil {
		return m.findByIDFn(id, p...)
	}
	return nil, nil
}
func (m *mockClientRepo) UpdateByUUID(id, data any) (*Client, error) { return nil, nil }
func (m *mockClientRepo) UpdateByID(id, data any) (*Client, error)   { return nil, nil }
func (m *mockClientRepo) DeleteByUUID(id any) error {
	if m.deleteByUUIDFn != nil {
		return m.deleteByUUIDFn(id)
	}
	return nil
}
func (m *mockClientRepo) DeleteByID(id any) error { return nil }
func (m *mockClientRepo) Paginate(c map[string]any, pg, lim int, p ...string) (*PaginationResult[Client], error) {
	return nil, nil
}
func (m *mockClientRepo) FindByUUIDAndTenantID(id uuid.UUID, tID int64) (*Client, error) {
	if m.findByUUIDAndTenantIDFn != nil {
		return m.findByUUIDAndTenantIDFn(id, tID)
	}
	return nil, nil
}
func (m *mockClientRepo) FindByNameAndIdentityProvider(n string, ipID, tID int64) (*Client, error) {
	if m.findByNameAndIdentityProviderFn != nil {
		return m.findByNameAndIdentityProviderFn(n, ipID, tID)
	}
	return nil, nil
}
func (m *mockClientRepo) FindByNameAndTenantID(n string, tID int64) (*Client, error) {
	if m.findByNameAndTenantIDFn != nil {
		return m.findByNameAndTenantIDFn(n, tID)
	}
	return nil, nil
}
func (m *mockClientRepo) FindByClientID(cID string, tID int64) (*Client, error) {
	return nil, nil
}
func (m *mockClientRepo) FindAllByTenantID(tID int64) ([]Client, error) { return nil, nil }
func (m *mockClientRepo) FindDefaultByTenantID(tID int64) (*Client, error) {
	if m.findDefaultByTenantIDFn != nil {
		return m.findDefaultByTenantIDFn(tID)
	}
	return nil, nil
}
func (m *mockClientRepo) FindPaginated(f ClientRepositoryGetFilter) (*PaginationResult[Client], error) {
	if m.findPaginatedFn != nil {
		return m.findPaginatedFn(f)
	}
	return &PaginationResult[Client]{}, nil
}
func (m *mockClientRepo) SetStatusByUUID(id uuid.UUID, tID int64, s string) error { return nil }
func (m *mockClientRepo) DeleteByUUIDAndTenantID(id uuid.UUID, tID int64) error   { return nil }

// ---------------------------------------------------------------------------
// Mock: UserRepository
// ---------------------------------------------------------------------------

type mockUserRepo struct {
	findByUsernameFn         func(username string) (*User, error)
	findByEmailFn            func(email string) (*User, error)
	findByEmailAndTenantIDFn func(email string, tenantID int64) (*User, error)
	findByUUIDFn             func(id any, preloads ...string) (*User, error)
	findByIDFn               func(id any, preloads ...string) (*User, error)
	findSuperAdminFn         func() (*User, error)
	findPaginatedFn          func(UserRepositoryGetFilter) (*PaginationResult[User], error)
	createFn                 func(*User) (*User, error)
	updateByUUIDFn           func(id, data any) (*User, error)
	updateByIDFn             func(id, data any) (*User, error)
	findRolesFn              func(userID int64) ([]Role, error)
	findByPhoneFn            func(phone string) (*User, error)
	setStatusFn              func(id uuid.UUID, s string) error
	deleteByUUIDFn           func(id any) error
	findBySubAndClientIDFn   func(sub, clientID string) (*User, error)
}

func (m *mockUserRepo) WithTx(_ *gorm.DB) UserRepository { return m }
func (m *mockUserRepo) FindByUsername(u string) (*User, error) {
	if m.findByUsernameFn != nil {
		return m.findByUsernameFn(u)
	}
	return nil, nil
}
func (m *mockUserRepo) FindByEmail(e string) (*User, error) {
	if m.findByEmailFn != nil {
		return m.findByEmailFn(e)
	}
	return nil, nil
}
func (m *mockUserRepo) FindByEmailAndTenantID(e string, tID int64) (*User, error) {
	if m.findByEmailAndTenantIDFn != nil {
		return m.findByEmailAndTenantIDFn(e, tID)
	}
	if m.findByEmailFn != nil {
		return m.findByEmailFn(e)
	}
	return nil, nil
}
func (m *mockUserRepo) FindByUsernameAndTenantID(u string, tID int64) (*User, error) {
	if m.findByUsernameFn != nil {
		return m.findByUsernameFn(u)
	}
	return nil, nil
}
func (m *mockUserRepo) FindByPhoneAndTenantID(phone string, tID int64) (*User, error) {
	if m.findByPhoneFn != nil {
		return m.findByPhoneFn(phone)
	}
	return nil, nil
}
func (m *mockUserRepo) FindByPendingEmailAndTenantID(_ string, _ int64) (*User, error) {
	return nil, nil
}
func (m *mockUserRepo) Create(e *User) (*User, error) {
	if m.createFn != nil {
		return m.createFn(e)
	}
	return e, nil
}
func (m *mockUserRepo) CreateOrUpdate(e *User) (*User, error) { return nil, nil }
func (m *mockUserRepo) FindAll(p ...string) ([]User, error)   { return nil, nil }
func (m *mockUserRepo) FindByUUID(id any, p ...string) (*User, error) {
	if m.findByUUIDFn != nil {
		return m.findByUUIDFn(id, p...)
	}
	return nil, nil
}
func (m *mockUserRepo) FindByUUIDs(ids []string, p ...string) ([]User, error) { return nil, nil }
func (m *mockUserRepo) FindByID(id any, p ...string) (*User, error) {
	if m.findByIDFn != nil {
		return m.findByIDFn(id, p...)
	}
	return nil, nil
}
func (m *mockUserRepo) UpdateByUUID(id, data any) (*User, error) {
	if m.updateByUUIDFn != nil {
		return m.updateByUUIDFn(id, data)
	}
	return nil, nil
}
func (m *mockUserRepo) UpdateByID(id, data any) (*User, error) {
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
func (m *mockUserRepo) Paginate(c map[string]any, pg, lim int, p ...string) (*PaginationResult[User], error) {
	return nil, nil
}
func (m *mockUserRepo) FindByPhone(phone string) (*User, error) {
	if m.findByPhoneFn != nil {
		return m.findByPhoneFn(phone)
	}
	return nil, nil
}
func (m *mockUserRepo) FindSuperAdmin() (*User, error) {
	if m.findSuperAdminFn != nil {
		return m.findSuperAdminFn()
	}
	return nil, nil
}
func (m *mockUserRepo) FindRoles(userID int64) ([]Role, error) {
	if m.findRolesFn != nil {
		return m.findRolesFn(userID)
	}
	return nil, nil
}
func (m *mockUserRepo) FindBySubAndClientID(sub, cID string) (*User, error) {
	if m.findBySubAndClientIDFn != nil {
		return m.findBySubAndClientIDFn(sub, cID)
	}
	return nil, nil
}
func (m *mockUserRepo) FindPaginated(f UserRepositoryGetFilter) (*PaginationResult[User], error) {
	if m.findPaginatedFn != nil {
		return m.findPaginatedFn(f)
	}
	return &PaginationResult[User]{}, nil
}
func (m *mockUserRepo) SetEmailVerified(id uuid.UUID, v bool) error { return nil }
func (m *mockUserRepo) SetStatus(id uuid.UUID, s string) error {
	if m.setStatusFn != nil {
		return m.setStatusFn(id, s)
	}
	return nil
}
func (m *mockUserRepo) SetForcePasswordChange(_ uuid.UUID, _ bool) error            { return nil }
func (m *mockUserRepo) SetPendingEmail(_ uuid.UUID, _, _ string, _ time.Time) error { return nil }
func (m *mockUserRepo) ClearEmailChange(_ uuid.UUID) error                          { return nil }
func (m *mockUserRepo) UpdateEmail(_ uuid.UUID, _ string) error                     { return nil }
func (m *mockUserRepo) UpdateUsername(_ uuid.UUID, _ string) error                  { return nil }
func (m *mockUserRepo) FindByPendingEmail(_ string) (*User, error)                  { return nil, nil }

// ---------------------------------------------------------------------------
// Mock: UserIdentityRepository
// ---------------------------------------------------------------------------

type mockUserIdentityRepo struct {
	findByUserIDAndClientIDFn func(userID, clientID int64) (*UserIdentity, error)
	createFn                  func(*UserIdentity) (*UserIdentity, error)
	findByUserIDFn            func(int64) ([]UserIdentity, error)
}

func (m *mockUserIdentityRepo) WithTx(_ *gorm.DB) UserIdentityRepository { return m }
func (m *mockUserIdentityRepo) FindByUserIDAndClientID(uID, cID int64) (*UserIdentity, error) {
	return m.findByUserIDAndClientIDFn(uID, cID)
}
func (m *mockUserIdentityRepo) Create(e *UserIdentity) (*UserIdentity, error) {
	if m.createFn != nil {
		return m.createFn(e)
	}
	return e, nil
}
func (m *mockUserIdentityRepo) CreateOrUpdate(e *UserIdentity) (*UserIdentity, error) {
	return nil, nil
}
func (m *mockUserIdentityRepo) FindAll(p ...string) ([]UserIdentity, error) { return nil, nil }
func (m *mockUserIdentityRepo) FindByUUID(id any, p ...string) (*UserIdentity, error) {
	return nil, nil
}
func (m *mockUserIdentityRepo) FindByUUIDs(ids []string, p ...string) ([]UserIdentity, error) {
	return nil, nil
}
func (m *mockUserIdentityRepo) FindByID(id any, p ...string) (*UserIdentity, error) {
	return nil, nil
}
func (m *mockUserIdentityRepo) UpdateByUUID(id, data any) (*UserIdentity, error) {
	return nil, nil
}
func (m *mockUserIdentityRepo) UpdateByID(id, data any) (*UserIdentity, error) { return nil, nil }
func (m *mockUserIdentityRepo) DeleteByUUID(id any) error                      { return nil }
func (m *mockUserIdentityRepo) DeleteByID(id any) error                        { return nil }
func (m *mockUserIdentityRepo) Paginate(c map[string]any, pg, lim int, p ...string) (*PaginationResult[UserIdentity], error) {
	return nil, nil
}
func (m *mockUserIdentityRepo) FindByUserID(uID int64) ([]UserIdentity, error) {
	if m.findByUserIDFn != nil {
		return m.findByUserIDFn(uID)
	}
	return nil, nil
}
func (m *mockUserIdentityRepo) FindByProviderAndSub(_, _ string) (*UserIdentity, error) {
	return nil, nil
}
func (m *mockUserIdentityRepo) FindByUserIDAndProvider(_ int64, _ string) (*UserIdentity, error) {
	return nil, nil
}
func (m *mockUserIdentityRepo) FindByIdentityProviderID(_ int64) ([]UserIdentity, error) {
	return nil, nil
}
func (m *mockUserIdentityRepo) DeleteByUserID(uID int64) error { return nil }

// ---------------------------------------------------------------------------
// Mock: IdentityProviderRepository
// ---------------------------------------------------------------------------

type mockIdentityProviderRepo struct {
	findByIdentifierFn func(identifier string) (*IdentityProvider, error)
	findByUUIDFn       func(id any, preloads ...string) (*IdentityProvider, error)
	findByNameFn       func(name string, tenantID int64) (*IdentityProvider, error)
	findPaginatedFn    func(IdentityProviderRepositoryGetFilter) (*PaginationResult[IdentityProvider], error)
	createOrUpdateFn   func(*IdentityProvider) (*IdentityProvider, error)
	deleteByUUIDFn     func(id any) error
}

func (m *mockIdentityProviderRepo) WithTx(_ *gorm.DB) IdentityProviderRepository { return m }
func (m *mockIdentityProviderRepo) FindByIdentifier(id string) (*IdentityProvider, error) {
	if m.findByIdentifierFn != nil {
		return m.findByIdentifierFn(id)
	}
	return nil, nil
}
func (m *mockIdentityProviderRepo) Create(e *IdentityProvider) (*IdentityProvider, error) {
	return nil, nil
}
func (m *mockIdentityProviderRepo) CreateOrUpdate(e *IdentityProvider) (*IdentityProvider, error) {
	if m.createOrUpdateFn != nil {
		return m.createOrUpdateFn(e)
	}
	return e, nil
}
func (m *mockIdentityProviderRepo) FindAll(p ...string) ([]IdentityProvider, error) {
	return nil, nil
}
func (m *mockIdentityProviderRepo) FindByUUID(id any, p ...string) (*IdentityProvider, error) {
	if m.findByUUIDFn != nil {
		return m.findByUUIDFn(id, p...)
	}
	return nil, nil
}
func (m *mockIdentityProviderRepo) FindByUUIDs(ids []string, p ...string) ([]IdentityProvider, error) {
	return nil, nil
}
func (m *mockIdentityProviderRepo) FindByID(id any, p ...string) (*IdentityProvider, error) {
	return nil, nil
}
func (m *mockIdentityProviderRepo) UpdateByUUID(id, data any) (*IdentityProvider, error) {
	return nil, nil
}
func (m *mockIdentityProviderRepo) UpdateByID(id, data any) (*IdentityProvider, error) {
	return nil, nil
}
func (m *mockIdentityProviderRepo) DeleteByUUID(id any) error {
	if m.deleteByUUIDFn != nil {
		return m.deleteByUUIDFn(id)
	}
	return nil
}
func (m *mockIdentityProviderRepo) DeleteByID(id any) error { return nil }
func (m *mockIdentityProviderRepo) Paginate(c map[string]any, pg, lim int, p ...string) (*PaginationResult[IdentityProvider], error) {
	return nil, nil
}
func (m *mockIdentityProviderRepo) FindByName(n string, tID int64) (*IdentityProvider, error) {
	if m.findByNameFn != nil {
		return m.findByNameFn(n, tID)
	}
	return nil, nil
}
func (m *mockIdentityProviderRepo) FindDefaultByTenantID(tID int64) (*IdentityProvider, error) {
	return nil, nil
}
func (m *mockIdentityProviderRepo) FindPaginated(f IdentityProviderRepositoryGetFilter) (*PaginationResult[IdentityProvider], error) {
	if m.findPaginatedFn != nil {
		return m.findPaginatedFn(f)
	}
	return &PaginationResult[IdentityProvider]{}, nil
}
func (m *mockIdentityProviderRepo) FindAllByTenantID(_ int64) ([]IdentityProvider, error) {
	return nil, nil
}
func (m *mockIdentityProviderRepo) FindByTenantAndProvider(_ int64, _ string) (*IdentityProvider, error) {
	return nil, nil
}

// ---------------------------------------------------------------------------
// Mock: UserTokenRepository
// ---------------------------------------------------------------------------

type mockUserTokenRepo struct {
	createFn                    func(*UserToken) (*UserToken, error)
	findByUserIDAndTokenTypeFn  func(userID int64, tokenType string) ([]UserToken, error)
	revokeByUUIDFn              func(id uuid.UUID) error
	findActiveSessionsFn        func(int64) ([]UserToken, error)
	findActiveSessionByUUIDFn   func(int64, uuid.UUID) (*UserToken, error)
	countActiveSessionsFn       func(int64) (int64, error)
	revokeSessionByUUIDFn       func(int64, uuid.UUID) error
	revokeAllSessionsByUserIDFn func(int64) error
	touchSessionFn              func(uuid.UUID, time.Time) error
}

func (m *mockUserTokenRepo) WithTx(_ *gorm.DB) UserTokenRepository { return m }
func (m *mockUserTokenRepo) Create(e *UserToken) (*UserToken, error) {
	if m.createFn != nil {
		return m.createFn(e)
	}
	return e, nil
}
func (m *mockUserTokenRepo) CreateOrUpdate(e *UserToken) (*UserToken, error) {
	return nil, nil
}
func (m *mockUserTokenRepo) FindAll(p ...string) ([]UserToken, error) { return nil, nil }
func (m *mockUserTokenRepo) FindByUUID(id any, p ...string) (*UserToken, error) {
	return nil, nil
}
func (m *mockUserTokenRepo) FindByUUIDs(ids []string, p ...string) ([]UserToken, error) {
	return nil, nil
}
func (m *mockUserTokenRepo) FindByID(id any, p ...string) (*UserToken, error) { return nil, nil }
func (m *mockUserTokenRepo) UpdateByUUID(id, data any) (*UserToken, error)    { return nil, nil }
func (m *mockUserTokenRepo) UpdateByID(id, data any) (*UserToken, error)      { return nil, nil }
func (m *mockUserTokenRepo) DeleteByUUID(id any) error                        { return nil }
func (m *mockUserTokenRepo) DeleteByID(id any) error                          { return nil }
func (m *mockUserTokenRepo) Paginate(c map[string]any, pg, lim int, p ...string) (*PaginationResult[UserToken], error) {
	return nil, nil
}
func (m *mockUserTokenRepo) FindByUserID(uID int64) ([]UserToken, error) { return nil, nil }
func (m *mockUserTokenRepo) FindActiveTokensByUserID(uID int64) ([]UserToken, error) {
	return nil, nil
}
func (m *mockUserTokenRepo) FindByUserIDAndTokenType(uID int64, tt string) ([]UserToken, error) {
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
func (m *mockUserTokenRepo) FindActiveSessions(userID int64) ([]UserToken, error) {
	if m.findActiveSessionsFn != nil {
		return m.findActiveSessionsFn(userID)
	}
	return nil, nil
}
func (m *mockUserTokenRepo) FindActiveSessionByUUID(userID int64, sessionUUID uuid.UUID) (*UserToken, error) {
	if m.findActiveSessionByUUIDFn != nil {
		return m.findActiveSessionByUUIDFn(userID, sessionUUID)
	}
	return nil, nil
}
func (m *mockUserTokenRepo) CountActiveSessions(userID int64) (int64, error) {
	if m.countActiveSessionsFn != nil {
		return m.countActiveSessionsFn(userID)
	}
	return 0, nil
}
func (m *mockUserTokenRepo) TouchSession(sessionUUID uuid.UUID, now time.Time) error {
	if m.touchSessionFn != nil {
		return m.touchSessionFn(sessionUUID, now)
	}
	return nil
}
func (m *mockUserTokenRepo) RevokeSessionByUUID(userID int64, sessionUUID uuid.UUID) error {
	if m.revokeSessionByUUIDFn != nil {
		return m.revokeSessionByUUIDFn(userID, sessionUUID)
	}
	return nil
}
func (m *mockUserTokenRepo) RevokeAllSessionsByUserID(userID int64) error {
	if m.revokeAllSessionsByUserIDFn != nil {
		return m.revokeAllSessionsByUserIDFn(userID)
	}
	return nil
}

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

func TestNewLoginService_WithJTIDenylist(t *testing.T) {
	denylist := &recordingLogoutJTIDenylister{}
	svc := NewLoginService(nil, nil, nil, nil, nil, nil, nil, nil, nil, denylist)

	typed, ok := svc.(*loginService)
	require.True(t, ok)
	assert.Same(t, denylist, typed.jtiDenylist)
}

func TestFindLoginUser(t *testing.T) {
	t.Run("username lookup succeeds", func(t *testing.T) {
		expected := &User{UserID: 1, Username: "user1"}
		repo := &mockUserRepo{
			findByUsernameFn: func(username string) (*User, error) {
				assert.Equal(t, "user1", username)
				return expected, nil
			},
		}

		user, err := findLoginUser(repo, "user1", 1)

		require.NoError(t, err)
		assert.Same(t, expected, user)
	})

	t.Run("email fallback succeeds after username miss", func(t *testing.T) {
		expected := &User{UserID: 2, Email: "user@example.com"}
		repo := &mockUserRepo{
			findByUsernameFn: func(username string) (*User, error) {
				return nil, gorm.ErrRecordNotFound
			},
			findByEmailAndTenantIDFn: func(email string, tenantID int64) (*User, error) {
				assert.Equal(t, "user@example.com", email)
				assert.Equal(t, int64(9), tenantID)
				return expected, nil
			},
		}

		user, err := findLoginUser(repo, "user@example.com", 9)

		require.NoError(t, err)
		assert.Same(t, expected, user)
	})

	t.Run("email fallback returns email lookup error when username had no error", func(t *testing.T) {
		repo := &mockUserRepo{
			findByUsernameFn: func(username string) (*User, error) {
				return nil, nil
			},
			findByEmailAndTenantIDFn: func(email string, tenantID int64) (*User, error) {
				return nil, assert.AnError
			},
		}

		user, err := findLoginUser(repo, "user@example.com", 9)

		require.ErrorIs(t, err, assert.AnError)
		assert.Nil(t, user)
	})
}

func TestJWTClaimTTL(t *testing.T) {
	future := time.Now().Add(time.Hour).Unix()

	tests := []struct {
		name  string
		claim any
		want  bool
	}{
		{name: "float64", claim: float64(future), want: true},
		{name: "int64", claim: int64(future), want: true},
		{name: "int", claim: int(future), want: true},
		{name: "json number", claim: json.Number("4102444800"), want: true},
		{name: "invalid json number", claim: json.Number("not-number"), want: false},
		{name: "unsupported type", claim: "4102444800", want: false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ttl := jwtClaimTTL(tc.claim)
			if tc.want {
				assert.Positive(t, ttl)
			} else {
				assert.Zero(t, ttl)
			}
		})
	}
}

func TestLoginService_DenylistLogoutJTI(t *testing.T) {
	t.Run("blank jti skips denylist", func(t *testing.T) {
		denylist := &recordingLogoutJTIDenylister{}
		svc := &loginService{jtiDenylist: denylist}

		err := svc.denylistLogoutJTI(context.Background(), map[string]any{"jti": "  ", "exp": time.Now().Add(time.Hour).Unix()})

		require.NoError(t, err)
		assert.Empty(t, denylist.jti)
	})

	t.Run("expired token skips denylist", func(t *testing.T) {
		denylist := &recordingLogoutJTIDenylister{}
		svc := &loginService{jtiDenylist: denylist}

		err := svc.denylistLogoutJTI(context.Background(), map[string]any{"jti": "jti-1", "exp": time.Now().Add(-time.Hour).Unix()})

		require.NoError(t, err)
		assert.Empty(t, denylist.jti)
	})

	t.Run("denylist error is returned", func(t *testing.T) {
		denylist := &recordingLogoutJTIDenylister{err: assert.AnError}
		svc := &loginService{jtiDenylist: denylist}

		err := svc.denylistLogoutJTI(context.Background(), map[string]any{"jti": "jti-1", "exp": time.Now().Add(time.Hour).Unix()})

		require.ErrorIs(t, err, assert.AnError)
		assert.Equal(t, "jti-1", denylist.jti)
	})
}

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
func buildActiveIdentityProvider() *IdentityProvider {
	return &IdentityProvider{
		IdentityProviderID: 1,
		Name:               "default",
		Provider:           shared.IDPProviderMaintainerd,
		ProviderType:       shared.IDPTypeIdentity,
		Identifier:         "test-provider",
		Status:             shared.StatusActive,
		Tenant:             &Tenant{Identifier: "system"},
	}
}

// buildActiveClient returns a minimal active client whose Domain and Identifier
// are both populated (required by generateTokenResponse).
func buildActiveClient() *Client {
	idp := buildActiveIdentityProvider()
	return &Client{
		ClientID:         1,
		Name:             "test-client",
		Domain:           strPtr("https://auth.example.com"),
		Identifier:       strPtr("test-client"),
		Status:           shared.StatusActive,
		IdentityProvider: idp,
	}
}

// buildActiveUser bcrypt-hashes the given plaintext password and returns an
// active user that the service can authenticate successfully.
func buildActiveUser(t *testing.T, password string) *User {
	t.Helper()
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.MinCost)
	require.NoError(t, err)
	hashStr := string(hash)
	return &User{
		UserID:          1,
		UserUUID:        uuid.New(),
		Username:        "testuser",
		Email:           "testuser@example.com",
		Password:        &hashStr,
		Status:          shared.StatusActive,
		IsEmailVerified: true,
	}
}

func registrationPolicyRepo(config string) *mockSecuritySettingRepo {
	return &mockSecuritySettingRepo{
		findDefaultByTenantIDFn: func(int64) (*secpolicy.SecuritySetting, error) {
			return &secpolicy.SecuritySetting{RegistrationConfig: datatypes.JSON([]byte(config))}, nil
		},
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
				m.findByEmailFn = func(e string) (*User, error) { return &User{Email: e}, nil }
			},
		},
		{
			name:     "tenant-scoped lookup returns user",
			email:    "a@b.com",
			tenantID: 42,
			setupRepo: func(m *mockUserRepo) {
				m.findByEmailAndTenantIDFn = func(e string, _ int64) (*User, error) {
					return &User{Email: e}, nil
				}
			},
		},
		{
			name:     "global lookup - user not found",
			email:    "nope@x.com",
			tenantID: 0,
			setupRepo: func(m *mockUserRepo) {
				m.findByEmailFn = func(_ string) (*User, error) { return nil, errors.New("not found") }
			},
			wantErr: true,
		},
		{
			name:     "tenant-scoped lookup - user not found",
			email:    "nope@x.com",
			tenantID: 5,
			setupRepo: func(m *mockUserRepo) {
				m.findByEmailAndTenantIDFn = func(_ string, _ int64) (*User, error) {
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
		name               string
		username           string // unique per case to avoid rate-limiter cross-talk
		password           string
		clientID           string
		providerID         string
		setup              func(t *testing.T, r repoSetup)
		expectCommit       bool // false → expect rollback (callback returned error)
		wantErr            bool
		wantErrContain     string
		wantPasswordChange bool
		securitySettings   secpolicy.SecuritySettingRepository
	}{
		{
			name:         "success",
			username:     "pub-success",
			password:     correctPassword,
			clientID:     "client-1",
			providerID:   "provider-1",
			expectCommit: true,
			setup: func(t *testing.T, r repoSetup) {
				r.idpRepo.findByIdentifierFn = func(_ string) (*IdentityProvider, error) {
					return buildActiveIdentityProvider(), nil
				}
				r.clientRepo.findByClientIDAndIdentityProviderFn = func(_, _ string) (*Client, error) {
					return buildActiveClient(), nil
				}
				r.userRepo.findByUsernameFn = func(_ string) (*User, error) {
					return buildActiveUser(t, correctPassword), nil
				}
				r.userIdentity.findByUserIDAndClientIDFn = func(_, _ int64) (*UserIdentity, error) {
					return &UserIdentity{Sub: "sub-123"}, nil
				}
			},
		},
		{
			name:         "success with tenant-scoped email",
			username:     "pub-success@example.com",
			password:     correctPassword,
			clientID:     "client-1",
			providerID:   "provider-1",
			expectCommit: true,
			setup: func(t *testing.T, r repoSetup) {
				idp := buildActiveIdentityProvider()
				idp.TenantID = 42
				client := buildActiveClient()
				client.IdentityProvider = idp
				r.idpRepo.findByIdentifierFn = func(_ string) (*IdentityProvider, error) {
					return idp, nil
				}
				r.clientRepo.findByClientIDAndIdentityProviderFn = func(_, _ string) (*Client, error) {
					return client, nil
				}
				r.userRepo.findByUsernameFn = func(_ string) (*User, error) {
					return nil, errors.New("not found")
				}
				r.userRepo.findByEmailAndTenantIDFn = func(email string, tenantID int64) (*User, error) {
					assert.Equal(t, "pub-success@example.com", email)
					assert.Equal(t, int64(42), tenantID)
					return buildActiveUser(t, correctPassword), nil
				}
				r.userIdentity.findByUserIDAndClientIDFn = func(_, _ int64) (*UserIdentity, error) {
					return &UserIdentity{Sub: "sub-123"}, nil
				}
			},
		},
		{
			name:               "force password change blocks token issuance",
			username:           "pub-force-change",
			password:           correctPassword,
			clientID:           "client-1",
			providerID:         "provider-1",
			expectCommit:       true,
			wantPasswordChange: true,
			setup: func(t *testing.T, r repoSetup) {
				r.idpRepo.findByIdentifierFn = func(_ string) (*IdentityProvider, error) {
					return buildActiveIdentityProvider(), nil
				}
				r.clientRepo.findByClientIDAndIdentityProviderFn = func(_, _ string) (*Client, error) {
					return buildActiveClient(), nil
				}
				r.userRepo.findByUsernameFn = func(_ string) (*User, error) {
					u := buildActiveUser(t, correctPassword)
					u.ForcePasswordChange = true
					return u, nil
				}
				r.userIdentity.findByUserIDAndClientIDFn = func(_, _ int64) (*UserIdentity, error) {
					return &UserIdentity{Sub: "sub-123"}, nil
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
				r.idpRepo.findByIdentifierFn = func(_ string) (*IdentityProvider, error) {
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
				r.idpRepo.findByIdentifierFn = func(_ string) (*IdentityProvider, error) {
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
				r.idpRepo.findByIdentifierFn = func(_ string) (*IdentityProvider, error) {
					return buildActiveIdentityProvider(), nil
				}
				r.clientRepo.findByClientIDAndIdentityProviderFn = func(_, _ string) (*Client, error) {
					c := buildActiveClient()
					c.Status = shared.StatusInactive
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
				r.idpRepo.findByIdentifierFn = func(_ string) (*IdentityProvider, error) {
					return buildActiveIdentityProvider(), nil
				}
				r.clientRepo.findByClientIDAndIdentityProviderFn = func(_, _ string) (*Client, error) {
					return buildActiveClient(), nil
				}
				r.userRepo.findByUsernameFn = func(_ string) (*User, error) {
					return buildActiveUser(t, correctPassword), nil
				}
				r.userIdentity.findByUserIDAndClientIDFn = func(_, _ int64) (*UserIdentity, error) {
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
				r.idpRepo.findByIdentifierFn = func(_ string) (*IdentityProvider, error) {
					return buildActiveIdentityProvider(), nil
				}
				r.clientRepo.findByClientIDAndIdentityProviderFn = func(_, _ string) (*Client, error) {
					return buildActiveClient(), nil
				}
				r.userRepo.findByUsernameFn = func(_ string) (*User, error) {
					u := buildActiveUser(t, correctPassword)
					u.Status = shared.StatusInactive
					return u, nil
				}
				r.userIdentity.findByUserIDAndClientIDFn = func(_, _ int64) (*UserIdentity, error) {
					return &UserIdentity{Sub: "sub-123"}, nil
				}
			},
		},
		{
			name:             "requires verified email from registration config",
			username:         "pub-unverified-email",
			password:         correctPassword,
			clientID:         "client-1",
			providerID:       "provider-1",
			expectCommit:     true,
			wantErr:          true,
			wantErrContain:   "email is not verified",
			securitySettings: registrationPolicyRepo(`{"require_email_verification":true}`),
			setup: func(t *testing.T, r repoSetup) {
				r.idpRepo.findByIdentifierFn = func(_ string) (*IdentityProvider, error) {
					return buildActiveIdentityProvider(), nil
				}
				r.clientRepo.findByClientIDAndIdentityProviderFn = func(_, _ string) (*Client, error) {
					return buildActiveClient(), nil
				}
				r.userRepo.findByUsernameFn = func(_ string) (*User, error) {
					u := buildActiveUser(t, correctPassword)
					u.IsEmailVerified = false
					return u, nil
				}
				r.userIdentity.findByUserIDAndClientIDFn = func(_, _ int64) (*UserIdentity, error) {
					return &UserIdentity{Sub: "sub-123"}, nil
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

			svc := NewLoginService(gormDB, repos.clientRepo, repos.userRepo, &mockUserTokenRepo{}, repos.userIdentity, repos.idpRepo, &mockAuthEventService{}, nil, tc.securitySettings)
			resp, err := svc.LoginPublic(context.Background(), tc.username, tc.password, strPtr(tc.clientID), strPtr(tc.providerID))

			if tc.wantErr {
				require.Error(t, err)
				assert.Nil(t, resp)
				if tc.wantErrContain != "" {
					assert.Contains(t, err.Error(), tc.wantErrContain)
				}
			} else {
				require.NoError(t, err)
				assert.NotNil(t, resp)
				if tc.wantPasswordChange {
					assert.True(t, resp.RequirePasswordChange)
					assert.Empty(t, resp.AccessToken)
					assert.Empty(t, resp.IDToken)
					assert.Empty(t, resp.RefreshToken)
					assert.Empty(t, resp.TokenType)
				} else {
					assert.NotEmpty(t, resp.AccessToken)
					assert.NotEmpty(t, resp.IDToken)
					assert.NotEmpty(t, resp.RefreshToken)
					assert.Equal(t, "Bearer", resp.TokenType)
				}
			}
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestLoginPublic_TenantMFAPolicyRequiresChallengeBeforeTokens(t *testing.T) {
	const correctPassword = "S3cur3P@ss!"

	initTestJWTKeysService(t)
	gormDB, mock := newMockGormDB(t)
	mock.ExpectBegin()
	mock.ExpectCommit()

	idp := buildActiveIdentityProvider()
	idp.TenantID = 42
	client := buildActiveClient()
	client.IdentityProvider = idp
	user := buildActiveUser(t, correctPassword)
	user.IsTOTPEnabled = true
	now := time.Now()
	user.MFAEnabledAt = &now

	userRepo := &mockUserRepo{
		findByUsernameFn: func(_ string) (*User, error) {
			return user, nil
		},
	}
	clientRepo := &mockClientRepo{
		findByClientIDAndIdentityProviderFn: func(_, _ string) (*Client, error) {
			return client, nil
		},
	}
	idpRepo := &mockIdentityProviderRepo{
		findByIdentifierFn: func(_ string) (*IdentityProvider, error) {
			return idp, nil
		},
	}
	userIdentityRepo := &mockUserIdentityRepo{
		findByUserIDAndClientIDFn: func(_, _ int64) (*UserIdentity, error) {
			return &UserIdentity{Sub: "sub-123"}, nil
		},
	}
	securitySettingRepo := &mockSecuritySettingRepo{
		findDefaultByTenantIDFn: func(tenantID int64) (*secpolicy.SecuritySetting, error) {
			require.Equal(t, int64(42), tenantID)
			return &secpolicy.SecuritySetting{
				MFAConfig: datatypes.JSON([]byte(`{"required":true,"allowed_methods":["totp","backup_code","webauthn"]}`)),
			}, nil
		},
	}

	svc := NewLoginService(
		gormDB,
		clientRepo,
		userRepo,
		&mockUserTokenRepo{},
		userIdentityRepo,
		idpRepo,
		&mockAuthEventService{},
		nil,
		securitySettingRepo,
	)

	svc.SetMFAFactorAuthenticator(&mockMFAAuthenticator{
		enrolledFn: func(int64) ([]string, error) { return []string{"totp", "backup_code"}, nil },
	})

	resp, err := svc.LoginPublic(context.Background(), "pub-mfa-required", correctPassword, strPtr("client-1"), strPtr("provider-1"))
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.True(t, resp.MFARequired)
	assert.Empty(t, resp.AccessToken)
	assert.Empty(t, resp.IDToken)
	assert.Empty(t, resp.RefreshToken)
	require.NotNil(t, resp.MFAChallengeToken)
	assert.ElementsMatch(t, []string{"totp", "backup_code"}, resp.MFAAllowedMethods)

	claims, err := jwt.ValidateStepUpChallengeToken(*resp.MFAChallengeToken)
	require.NoError(t, err)
	assert.Equal(t, user.UserUUID.String(), claims["sub"])
	assert.ElementsMatch(t, []any{"totp", "backup_code"}, claims["allowed_methods"])
	assert.NoError(t, mock.ExpectationsWereMet())
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
		name             string
		username         string
		password         string
		clientID         *string
		providerID       *string
		setup            func(t *testing.T, r repoSetup)
		expectCommit     bool
		wantErr          bool
		wantErrContain   string
		securitySettings secpolicy.SecuritySettingRepository
	}{
		{
			name:         "success with default client",
			username:     "int-success-default",
			password:     correctPassword,
			clientID:     nil,
			providerID:   nil,
			expectCommit: true,
			setup: func(t *testing.T, r repoSetup) {
				r.clientRepo.findSystemFn = func() (*Client, error) { return buildActiveClient(), nil }
				r.userRepo.findByUsernameFn = func(_ string) (*User, error) {
					return buildActiveUser(t, correctPassword), nil
				}
				r.userIdentity.findByUserIDAndClientIDFn = func(_, _ int64) (*UserIdentity, error) {
					return &UserIdentity{Sub: "sub-456"}, nil
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
				r.clientRepo.findByClientIDAndIdentityProviderFn = func(_, _ string) (*Client, error) {
					return buildActiveClient(), nil
				}
				r.userRepo.findByUsernameFn = func(_ string) (*User, error) {
					return buildActiveUser(t, correctPassword), nil
				}
				r.userIdentity.findByUserIDAndClientIDFn = func(_, _ int64) (*UserIdentity, error) {
					return &UserIdentity{Sub: "sub-789"}, nil
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
				r.clientRepo.findSystemFn = func() (*Client, error) { return nil, errors.New("db error") }
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
				r.clientRepo.findSystemFn = func() (*Client, error) {
					c := buildActiveClient()
					c.Status = shared.StatusInactive
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
				r.clientRepo.findSystemFn = func() (*Client, error) { return buildActiveClient(), nil }
				r.userRepo.findByUsernameFn = func(_ string) (*User, error) {
					return buildActiveUser(t, correctPassword), nil
				}
				r.userIdentity.findByUserIDAndClientIDFn = func(_, _ int64) (*UserIdentity, error) {
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
				r.clientRepo.findSystemFn = func() (*Client, error) { return buildActiveClient(), nil }
				r.userRepo.findByUsernameFn = func(_ string) (*User, error) {
					u := buildActiveUser(t, correctPassword)
					u.Status = shared.StatusInactive
					return u, nil
				}
				r.userIdentity.findByUserIDAndClientIDFn = func(_, _ int64) (*UserIdentity, error) {
					return &UserIdentity{Sub: "sub-456"}, nil
				}
			},
		},
		{
			name:             "requires verified email from registration config",
			username:         "int-unverified-email",
			password:         correctPassword,
			clientID:         nil,
			providerID:       nil,
			expectCommit:     true,
			wantErr:          true,
			wantErrContain:   "email is not verified",
			securitySettings: registrationPolicyRepo(`{"require_email_verification":true}`),
			setup: func(t *testing.T, r repoSetup) {
				r.clientRepo.findSystemFn = func() (*Client, error) { return buildActiveClient(), nil }
				r.userRepo.findByUsernameFn = func(_ string) (*User, error) {
					u := buildActiveUser(t, correctPassword)
					u.IsEmailVerified = false
					return u, nil
				}
				r.userIdentity.findByUserIDAndClientIDFn = func(_, _ int64) (*UserIdentity, error) {
					return &UserIdentity{Sub: "sub-456"}, nil
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

			svc := NewLoginService(gormDB, repos.clientRepo, repos.userRepo, &mockUserTokenRepo{}, repos.userIdentity, &mockIdentityProviderRepo{}, &mockAuthEventService{}, nil, tc.securitySettings)
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
	require.NoError(t, mr.Set("rl:lock:0:"+identifier, "1"))

	return func() {
		security.InitRateLimiter(nil)
		_ = rdb.Close()
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
		&mockUserIdentityRepo{findByUserIDAndClientIDFn: func(_, _ int64) (*UserIdentity, error) { return nil, nil }},
		&mockIdentityProviderRepo{}, &mockAuthEventService{}, nil, nil)
	_, err := svc.LoginPublic(context.Background(), username, "pass", strPtr("c1"), strPtr("p1"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "locked")
}

func TestLoginPublic_ClientLookupError(t *testing.T) {
	initTestJWTKeysService(t)
	gormDB, mock := newMockGormDB(t)
	mock.ExpectBegin()
	mock.ExpectRollback()

	idpRepo := &mockIdentityProviderRepo{
		findByIdentifierFn: func(_ string) (*IdentityProvider, error) {
			return buildActiveIdentityProvider(), nil
		},
	}
	clientRepo := &mockClientRepo{
		findByClientIDAndIdentityProviderFn: func(_, _ string) (*Client, error) {
			return nil, errors.New("client db err")
		},
	}

	svc := NewLoginService(gormDB, clientRepo, &mockUserRepo{}, &mockUserTokenRepo{},
		&mockUserIdentityRepo{findByUserIDAndClientIDFn: func(_, _ int64) (*UserIdentity, error) { return nil, nil }},
		idpRepo, &mockAuthEventService{}, nil, nil)
	_, err := svc.LoginPublic(context.Background(), "pub-client-err", "pass", strPtr("c1"), strPtr("p1"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "authentication failed")
}

func TestLoginPublic_UserNotFound(t *testing.T) {
	initTestJWTKeysService(t)
	gormDB, mock := newMockGormDB(t)
	mock.ExpectBegin()
	mock.ExpectCommit()

	idpRepo := &mockIdentityProviderRepo{
		findByIdentifierFn: func(_ string) (*IdentityProvider, error) {
			return buildActiveIdentityProvider(), nil
		},
	}
	clientRepo := &mockClientRepo{
		findByClientIDAndIdentityProviderFn: func(_, _ string) (*Client, error) {
			return buildActiveClient(), nil
		},
	}
	userRepo := &mockUserRepo{
		findByUsernameFn: func(_ string) (*User, error) {
			return nil, errors.New("not found")
		},
	}

	svc := NewLoginService(gormDB, clientRepo, userRepo, &mockUserTokenRepo{},
		&mockUserIdentityRepo{findByUserIDAndClientIDFn: func(_, _ int64) (*UserIdentity, error) { return nil, nil }},
		idpRepo, &mockAuthEventService{}, nil, nil)
	_, err := svc.LoginPublic(context.Background(), "pub-user-missing", "pass", strPtr("c1"), strPtr("p1"))
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
		&mockUserIdentityRepo{findByUserIDAndClientIDFn: func(_, _ int64) (*UserIdentity, error) { return nil, nil }},
		&mockIdentityProviderRepo{}, &mockAuthEventService{}, nil, nil)
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
		findByClientIDAndIdentityProviderFn: func(_, _ string) (*Client, error) {
			return nil, errors.New("client db err")
		},
	}

	cID := "client-x"
	pID := "provider-x"
	svc := NewLoginService(gormDB, clientRepo, &mockUserRepo{}, &mockUserTokenRepo{},
		&mockUserIdentityRepo{findByUserIDAndClientIDFn: func(_, _ int64) (*UserIdentity, error) { return nil, nil }},
		&mockIdentityProviderRepo{}, &mockAuthEventService{}, nil, nil)
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
		findSystemFn: func() (*Client, error) { return buildActiveClient(), nil },
	}
	userRepo := &mockUserRepo{
		findByUsernameFn: func(_ string) (*User, error) { return nil, nil },
	}

	svc := NewLoginService(gormDB, clientRepo, userRepo, &mockUserTokenRepo{},
		&mockUserIdentityRepo{findByUserIDAndClientIDFn: func(_, _ int64) (*UserIdentity, error) { return nil, nil }},
		&mockIdentityProviderRepo{}, &mockAuthEventService{}, nil, nil)
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
		findByIdentifierFn: func(_ string) (*IdentityProvider, error) {
			return buildActiveIdentityProvider(), nil
		},
	}
	clientRepo := &mockClientRepo{
		findByClientIDAndIdentityProviderFn: func(_, _ string) (*Client, error) {
			return buildActiveClient(), nil
		},
	}
	userRepo := &mockUserRepo{
		findByUsernameFn: func(_ string) (*User, error) {
			return buildActiveUser(t, correctPassword), nil
		},
	}
	userIdentityRepo := &mockUserIdentityRepo{
		findByUserIDAndClientIDFn: func(_, _ int64) (*UserIdentity, error) {
			return &UserIdentity{Sub: "sub-token-err"}, nil
		},
	}

	svc := NewLoginService(gormDB, clientRepo, userRepo, &mockUserTokenRepo{}, userIdentityRepo, idpRepo, &mockAuthEventService{}, nil, nil)
	_, err := svc.LoginPublic(context.Background(), "pub-token-err", correctPassword, strPtr("c1"), strPtr("p1"))
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
		findSystemFn: func() (*Client, error) { return buildActiveClient(), nil },
	}
	userRepo := &mockUserRepo{
		findByUsernameFn: func(_ string) (*User, error) {
			return buildActiveUser(t, correctPassword), nil
		},
	}
	userIdentityRepo := &mockUserIdentityRepo{
		findByUserIDAndClientIDFn: func(_, _ int64) (*UserIdentity, error) {
			return &UserIdentity{Sub: "sub-token-err"}, nil
		},
	}

	svc := NewLoginService(gormDB, clientRepo, userRepo, &mockUserTokenRepo{}, userIdentityRepo, &mockIdentityProviderRepo{}, &mockAuthEventService{}, nil, nil)
	_, err := svc.Login(context.Background(), "int-token-err", correctPassword, nil, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "private key not initialized")
}

// ---------------------------------------------------------------------------
// mockSessionService
// ---------------------------------------------------------------------------

type mockSessionService struct {
	enforceConcurrentLimitFn func(ctx context.Context, userUUID uuid.UUID, userID int64) error
	createSessionFn          func(ctx context.Context, userID int64, ipAddress, userAgent string) (*UserToken, error)
	validateAndTouchFn       func(ctx context.Context, sessionUUID uuid.UUID, userID int64) error
}

func (m *mockSessionService) ListSessions(ctx context.Context, userID int64) ([]*SessionDataResult, error) {
	return nil, nil
}
func (m *mockSessionService) RevokeSession(ctx context.Context, userID int64, sessionUUID uuid.UUID) error {
	return nil
}
func (m *mockSessionService) RevokeAllSessions(ctx context.Context, userID int64) error {
	return nil
}
func (m *mockSessionService) CreateSession(ctx context.Context, userID int64, ipAddress, userAgent string) (*UserToken, error) {
	if m.createSessionFn != nil {
		return m.createSessionFn(ctx, userID, ipAddress, userAgent)
	}
	return &UserToken{UserTokenUUID: uuid.New(), TokenType: "session"}, nil
}
func (m *mockSessionService) EnforceConcurrentLimit(ctx context.Context, userUUID uuid.UUID, userID int64) error {
	if m.enforceConcurrentLimitFn != nil {
		return m.enforceConcurrentLimitFn(ctx, userUUID, userID)
	}
	return nil
}
func (m *mockSessionService) ValidateAndTouch(ctx context.Context, sessionUUID uuid.UUID, userID int64) error {
	if m.validateAndTouchFn != nil {
		return m.validateAndTouchFn(ctx, sessionUUID, userID)
	}
	return nil
}

// ---------------------------------------------------------------------------
// TestLoginPublic_WithSession – verify generateTokenResponse session paths
// ---------------------------------------------------------------------------

func TestLoginPublic_WithSession(t *testing.T) {
	const correctPassword = "S3cur3P@ss!"

	t.Run("enforce concurrent limit error", func(t *testing.T) {
		initTestJWTKeysService(t)
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectCommit()

		idpRepo := &mockIdentityProviderRepo{
			findByIdentifierFn: func(_ string) (*IdentityProvider, error) {
				return buildActiveIdentityProvider(), nil
			},
		}
		clientRepo := &mockClientRepo{
			findByClientIDAndIdentityProviderFn: func(_, _ string) (*Client, error) {
				return buildActiveClient(), nil
			},
		}
		userRepo := &mockUserRepo{
			findByUsernameFn: func(_ string) (*User, error) {
				return buildActiveUser(t, correctPassword), nil
			},
		}
		userIdentityRepo := &mockUserIdentityRepo{
			findByUserIDAndClientIDFn: func(_, _ int64) (*UserIdentity, error) {
				return &UserIdentity{Sub: "sub-session-limit"}, nil
			},
		}
		sessionSvc := &mockSessionService{
			enforceConcurrentLimitFn: func(_ context.Context, _ uuid.UUID, _ int64) error {
				return errors.New("too many sessions")
			},
		}

		svc := NewLoginService(gormDB, clientRepo, userRepo, &mockUserTokenRepo{},
			userIdentityRepo, idpRepo, &mockAuthEventService{}, sessionSvc, nil)
		_, err := svc.LoginPublic(context.Background(), "pub-session-limit", correctPassword, strPtr("c1"), strPtr("p1"))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "too many sessions")
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("create session error", func(t *testing.T) {
		initTestJWTKeysService(t)
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectCommit()

		idpRepo := &mockIdentityProviderRepo{
			findByIdentifierFn: func(_ string) (*IdentityProvider, error) {
				return buildActiveIdentityProvider(), nil
			},
		}
		clientRepo := &mockClientRepo{
			findByClientIDAndIdentityProviderFn: func(_, _ string) (*Client, error) {
				return buildActiveClient(), nil
			},
		}
		userRepo := &mockUserRepo{
			findByUsernameFn: func(_ string) (*User, error) {
				return buildActiveUser(t, correctPassword), nil
			},
		}
		userIdentityRepo := &mockUserIdentityRepo{
			findByUserIDAndClientIDFn: func(_, _ int64) (*UserIdentity, error) {
				return &UserIdentity{Sub: "sub-session-create"}, nil
			},
		}
		sessionSvc := &mockSessionService{
			createSessionFn: func(_ context.Context, _ int64, _, _ string) (*UserToken, error) {
				return nil, errors.New("create session failed")
			},
		}

		svc := NewLoginService(gormDB, clientRepo, userRepo, &mockUserTokenRepo{},
			userIdentityRepo, idpRepo, &mockAuthEventService{}, sessionSvc, nil)
		_, err := svc.LoginPublic(context.Background(), "pub-session-create", correctPassword, strPtr("c1"), strPtr("p1"))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "create session failed")
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("success with session id in response", func(t *testing.T) {
		initTestJWTKeysService(t)
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectCommit()

		idpRepo := &mockIdentityProviderRepo{
			findByIdentifierFn: func(_ string) (*IdentityProvider, error) {
				return buildActiveIdentityProvider(), nil
			},
		}
		clientRepo := &mockClientRepo{
			findByClientIDAndIdentityProviderFn: func(_, _ string) (*Client, error) {
				return buildActiveClient(), nil
			},
		}
		userRepo := &mockUserRepo{
			findByUsernameFn: func(_ string) (*User, error) {
				return buildActiveUser(t, correctPassword), nil
			},
		}
		userIdentityRepo := &mockUserIdentityRepo{
			findByUserIDAndClientIDFn: func(_, _ int64) (*UserIdentity, error) {
				return &UserIdentity{Sub: "sub-session-ok"}, nil
			},
		}

		svc := NewLoginService(gormDB, clientRepo, userRepo, &mockUserTokenRepo{},
			userIdentityRepo, idpRepo, &mockAuthEventService{}, &mockSessionService{}, nil)
		resp, err := svc.LoginPublic(context.Background(), "pub-session-ok", correctPassword, strPtr("c1"), strPtr("p1"))
		require.NoError(t, err)
		require.NotNil(t, resp)
		require.NotNil(t, resp.SessionID)
		assert.NotEmpty(t, *resp.SessionID)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

// ---------------------------------------------------------------------------
// TestLoginMFAChallengeResponse
// ---------------------------------------------------------------------------

func TestLoginMFAChallengeResponse(t *testing.T) {
	t.Run("FindDefaultByTenantID error returns nil nil", func(t *testing.T) {
		settingRepo := &mockSecuritySettingRepo{
			findDefaultByTenantIDFn: func(int64) (*secpolicy.SecuritySetting, error) {
				return nil, errors.New("db error")
			},
		}
		svc := &loginService{securitySettingRepo: settingRepo}
		resp, err := svc.loginMFAChallengeResponse(context.Background(), &User{UserID: 1}, 1, false)
		assert.NoError(t, err)
		assert.Nil(t, resp)
	})

	t.Run("setting nil returns nil nil", func(t *testing.T) {
		settingRepo := &mockSecuritySettingRepo{
			findDefaultByTenantIDFn: func(int64) (*secpolicy.SecuritySetting, error) {
				return nil, nil
			},
		}
		svc := &loginService{securitySettingRepo: settingRepo}
		resp, err := svc.loginMFAChallengeResponse(context.Background(), &User{UserID: 1}, 1, false)
		assert.NoError(t, err)
		assert.Nil(t, resp)
	})

	t.Run("MFAConfig empty returns nil nil", func(t *testing.T) {
		settingRepo := &mockSecuritySettingRepo{
			findDefaultByTenantIDFn: func(int64) (*secpolicy.SecuritySetting, error) {
				return &secpolicy.SecuritySetting{}, nil
			},
		}
		svc := &loginService{securitySettingRepo: settingRepo}
		resp, err := svc.loginMFAChallengeResponse(context.Background(), &User{UserID: 1}, 1, false)
		assert.NoError(t, err)
		assert.Nil(t, resp)
	})

	t.Run("invalid JSON unmarshal error returns nil nil", func(t *testing.T) {
		settingRepo := &mockSecuritySettingRepo{
			findDefaultByTenantIDFn: func(int64) (*secpolicy.SecuritySetting, error) {
				return &secpolicy.SecuritySetting{
					MFAConfig: datatypes.JSON([]byte("not-json")),
				}, nil
			},
		}
		svc := &loginService{securitySettingRepo: settingRepo}
		resp, err := svc.loginMFAChallengeResponse(context.Background(), &User{UserID: 1}, 1, false)
		assert.NoError(t, err)
		assert.Nil(t, resp)
	})

	t.Run("not required and not enforce MFA returns nil nil", func(t *testing.T) {
		settingRepo := &mockSecuritySettingRepo{
			findDefaultByTenantIDFn: func(int64) (*secpolicy.SecuritySetting, error) {
				return &secpolicy.SecuritySetting{
					MFAConfig: datatypes.JSON([]byte(`{"required":false,"enforce_mfa":false}`)),
				}, nil
			},
		}
		svc := &loginService{securitySettingRepo: settingRepo}
		resp, err := svc.loginMFAChallengeResponse(context.Background(), &User{UserID: 1}, 1, false)
		assert.NoError(t, err)
		assert.Nil(t, resp)
	})

	t.Run("disabled mode skips challenge even when user has enrolled MFA", func(t *testing.T) {
		settingRepo := &mockSecuritySettingRepo{
			findDefaultByTenantIDFn: func(int64) (*secpolicy.SecuritySetting, error) {
				return &secpolicy.SecuritySetting{
					MFAConfig: datatypes.JSON([]byte(`{"mode":"disabled","allowed_methods":["totp"]}`)),
				}, nil
			},
		}
		svc := &loginService{
			securitySettingRepo: settingRepo,
			mfaAuthenticator:    &mockMFAAuthenticator{enrolledFn: func(int64) ([]string, error) { return []string{"totp"}, nil }},
		}

		resp, err := svc.loginMFAChallengeResponse(context.Background(), &User{UserID: 1, IsTOTPEnabled: true}, 1, false)

		require.NoError(t, err)
		assert.Nil(t, resp)
	})

	t.Run("preferred method is offered first", func(t *testing.T) {
		initTestJWTKeysService(t)
		settingRepo := &mockSecuritySettingRepo{
			findDefaultByTenantIDFn: func(int64) (*secpolicy.SecuritySetting, error) {
				return &secpolicy.SecuritySetting{
					MFAConfig: datatypes.JSON([]byte(`{"mode":"enforced","allowed_methods":["totp","webauthn"],"preferred_method":"webauthn"}`)),
				}, nil
			},
		}
		svc := &loginService{
			securitySettingRepo: settingRepo,
			mfaAuthenticator:    &mockMFAAuthenticator{enrolledFn: func(int64) ([]string, error) { return []string{"totp", "webauthn"}, nil }},
		}

		resp, err := svc.loginMFAChallengeResponse(context.Background(), &User{UserID: 1, UserUUID: uuid.New(), IsTOTPEnabled: true, IsWebAuthnEnabled: true}, 1, false)

		require.NoError(t, err)
		require.NotNil(t, resp)
		assert.Equal(t, []string{"webauthn", "totp"}, resp.MFAAllowedMethods)
	})

	t.Run("admin grace period applies only to tenant super admin", func(t *testing.T) {
		settingRepo := &mockSecuritySettingRepo{
			findDefaultByTenantIDFn: func(int64) (*secpolicy.SecuritySetting, error) {
				return &secpolicy.SecuritySetting{
					MFAConfig: datatypes.JSON([]byte(`{"mode":"enforced","allowed_methods":["totp"],"grace_period_days":0,"admin_grace_period_days":30}`)),
				}, nil
			},
		}
		user := &User{UserID: 1, CreatedAt: time.Now()}
		svc := &loginService{
			userRepo: &mockUserRepo{findRolesFn: func(int64) ([]Role, error) {
				return []Role{{TenantID: 1, Name: shared.RoleSuperAdmin}}, nil
			}},
			securitySettingRepo: settingRepo,
			mfaAuthenticator:    &mockMFAAuthenticator{enrolledFn: func(int64) ([]string, error) { return nil, nil }},
		}

		resp, err := svc.loginMFAChallengeResponse(context.Background(), user, 1, false)

		require.NoError(t, err)
		assert.Nil(t, resp)

		svc.userRepo = &mockUserRepo{findRolesFn: func(int64) ([]Role, error) {
			return []Role{{TenantID: 2, Name: shared.RoleSuperAdmin}}, nil
		}}
		resp, err = svc.loginMFAChallengeResponse(context.Background(), user, 1, false)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "MFA is required but no supported factors are enrolled")
		assert.Nil(t, resp)
	})

	t.Run("no allowed methods returns error", func(t *testing.T) {
		settingRepo := &mockSecuritySettingRepo{
			findDefaultByTenantIDFn: func(int64) (*secpolicy.SecuritySetting, error) {
				return &secpolicy.SecuritySetting{
					MFAConfig: datatypes.JSON([]byte(`{"required":true,"allowed_methods":["totp"]}`)),
				}, nil
			},
		}
		// Tenant requires totp, but the user has only backup codes enrolled.
		svc := &loginService{
			securitySettingRepo: settingRepo,
			mfaAuthenticator:    &mockMFAAuthenticator{enrolledFn: func(int64) ([]string, error) { return []string{"backup_code"}, nil }},
		}
		user := &User{UserID: 1, IsTOTPEnabled: false, IsWebAuthnEnabled: false}
		resp, err := svc.loginMFAChallengeResponse(context.Background(), user, 1, false)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "MFA is required but no supported factors are enrolled")
		assert.Nil(t, resp)
	})

	t.Run("GenerateStepUpChallengeToken error returns error", func(t *testing.T) {
		jwt.ResetJWTKeys()
		defer initTestJWTKeysService(t)

		now := time.Now()
		settingRepo := &mockSecuritySettingRepo{
			findDefaultByTenantIDFn: func(int64) (*secpolicy.SecuritySetting, error) {
				return &secpolicy.SecuritySetting{
					MFAConfig: datatypes.JSON([]byte(`{"required":true,"allowed_methods":["totp","backup_code"]}`)),
				}, nil
			},
		}
		svc := &loginService{
			securitySettingRepo: settingRepo,
			mfaAuthenticator:    &mockMFAAuthenticator{enrolledFn: func(int64) ([]string, error) { return []string{"totp", "backup_code"}, nil }},
		}
		user := &User{UserID: 1, IsTOTPEnabled: true, MFAEnabledAt: &now}
		resp, err := svc.loginMFAChallengeResponse(context.Background(), user, 1, false)
		require.Error(t, err)
		assert.Nil(t, resp)
	})
}

func TestMagicLinkMFAChallenge(t *testing.T) {
	settingRepo := func(config string) *mockSecuritySettingRepo {
		return &mockSecuritySettingRepo{
			findDefaultByTenantIDFn: func(int64) (*secpolicy.SecuritySetting, error) {
				return &secpolicy.SecuritySetting{MFAConfig: datatypes.JSON([]byte(config))}, nil
			},
		}
	}
	user := &User{UserID: 1, UserUUID: uuid.New(), CreatedAt: time.Now()}

	t.Run("disabled skips MFA even when a factor is enrolled", func(t *testing.T) {
		svc := &loginService{
			securitySettingRepo: settingRepo(`{"mode":"disabled","allowed_methods":["totp"]}`),
			mfaAuthenticator:    &mockMFAAuthenticator{enrolledFn: func(int64) ([]string, error) { return []string{"totp"}, nil }},
		}

		resp, err := svc.MagicLinkMFAChallenge(context.Background(), user, 1)

		require.NoError(t, err)
		assert.Nil(t, resp)
	})

	t.Run("optional skips MFA when the user has no enrolled factor", func(t *testing.T) {
		svc := &loginService{
			securitySettingRepo: settingRepo(`{"mode":"optional","allowed_methods":["totp"]}`),
			mfaAuthenticator:    &mockMFAAuthenticator{enrolledFn: func(int64) ([]string, error) { return nil, nil }},
		}

		resp, err := svc.MagicLinkMFAChallenge(context.Background(), user, 1)

		require.NoError(t, err)
		assert.Nil(t, resp)
	})

	t.Run("optional challenges when a non-email factor is enrolled", func(t *testing.T) {
		initTestJWTKeysService(t)
		svc := &loginService{
			securitySettingRepo: settingRepo(`{"mode":"optional","allowed_methods":["email_otp","totp"],"allow_email_otp":true}`),
			mfaAuthenticator:    &mockMFAAuthenticator{enrolledFn: func(int64) ([]string, error) { return []string{"email_otp", "totp"}, nil }},
		}

		resp, err := svc.MagicLinkMFAChallenge(context.Background(), user, 1)

		require.NoError(t, err)
		require.NotNil(t, resp)
		assert.True(t, resp.MFARequired)
		assert.Equal(t, []string{"totp"}, resp.MFAAllowedMethods)
		claims, err := jwt.ValidateStepUpChallengeToken(*resp.MFAChallengeToken)
		require.NoError(t, err)
		assert.Equal(t, jwt.AMRMagicLink, claims["primary_amr"])
	})

	t.Run("email OTP alone cannot satisfy magic-link MFA", func(t *testing.T) {
		svc := &loginService{
			securitySettingRepo: settingRepo(`{"mode":"optional","allowed_methods":["email_otp"],"allow_email_otp":true}`),
			mfaAuthenticator:    &mockMFAAuthenticator{enrolledFn: func(int64) ([]string, error) { return []string{"email_otp"}, nil }},
		}

		resp, err := svc.MagicLinkMFAChallenge(context.Background(), user, 1)

		require.Error(t, err)
		assert.Nil(t, resp)
		assert.Contains(t, err.Error(), "no supported non-email factors")
	})

	t.Run("enforced blocks when no factor is enrolled", func(t *testing.T) {
		svc := &loginService{
			securitySettingRepo: settingRepo(`{"mode":"enforced","allowed_methods":["totp"],"grace_period_days":30}`),
			mfaAuthenticator:    &mockMFAAuthenticator{enrolledFn: func(int64) ([]string, error) { return nil, nil }},
		}

		resp, err := svc.MagicLinkMFAChallenge(context.Background(), user, 1)

		require.Error(t, err)
		assert.Nil(t, resp)
		assert.Contains(t, err.Error(), "no supported non-email factors")
	})
}

func TestReplacePrimaryAMR(t *testing.T) {
	assert.Equal(t, []string{"magic_link", "otp"}, replacePrimaryAMR([]string{"pwd", "otp"}, "magic_link"))
	assert.Equal(t, []string{"magic_link", "user", "hwk"}, replacePrimaryAMR([]string{"pwd", "user", "hwk"}, "magic_link"))
	assert.Equal(t, []string{"magic_link", "otp"}, replacePrimaryAMR([]string{"otp"}, "magic_link"))
}

func TestIssueMagicLinkSessionUsesPasswordlessAMR(t *testing.T) {
	initTestJWTKeysService(t)
	svc := &loginService{}
	user := &User{UserID: 1, UserUUID: uuid.New(), Username: "magic-user", Status: shared.StatusActive}

	resp, err := svc.IssueMagicLinkSession(context.Background(), "magic-sub", user, buildActiveClient())

	require.NoError(t, err)
	claims, err := jwt.ValidateToken(resp.AccessToken)
	require.NoError(t, err)
	assert.Equal(t, []any{jwt.AMRMagicLink}, claims["amr"])
	assert.Equal(t, jwt.ACRLevel1, claims["acr"])
}

// ---------------------------------------------------------------------------
// TestLoginMFAAllowedMethods
// ---------------------------------------------------------------------------

func TestFilterMFAMethodsByPolicy(t *testing.T) {
	t.Run("empty policy offers all enrolled methods", func(t *testing.T) {
		enrolled := []string{"totp", "webauthn", "sms", "backup_code"}
		assert.Equal(t, enrolled, filterMFAMethodsByPolicy(enrolled, nil))
	})

	t.Run("restricts to policy-allowed methods, preserving enrolled order", func(t *testing.T) {
		enrolled := []string{"totp", "webauthn", "sms", "backup_code"}
		got := filterMFAMethodsByPolicy(enrolled, []string{"sms", "totp"})
		assert.Equal(t, []string{"totp", "sms"}, got)
	})

	t.Run("returns empty when no enrolled method is allowed", func(t *testing.T) {
		got := filterMFAMethodsByPolicy([]string{"backup_code"}, []string{"totp"})
		assert.Empty(t, got)
	})
}

func TestHasPrimaryMFAFactor(t *testing.T) {
	assert.True(t, hasPrimaryMFAFactor([]string{"totp"}))
	assert.True(t, hasPrimaryMFAFactor([]string{"backup_code", "webauthn"}))
	assert.True(t, hasPrimaryMFAFactor([]string{"sms"}))
	assert.False(t, hasPrimaryMFAFactor([]string{"backup_code"}))
	assert.False(t, hasPrimaryMFAFactor(nil))
}

// ---------------------------------------------------------------------------
// TestLoginCheckPasswordExpiry
// ---------------------------------------------------------------------------

func TestLoginCheckPasswordExpiry(t *testing.T) {
	t.Run("password expired sets ForcePasswordChange true", func(t *testing.T) {
		userRepo := &mockUserRepo{}
		pastTime := time.Now().AddDate(0, 0, -10)
		user := &User{UserID: 42, PasswordChangedAt: &pastTime}
		settingRepo := &mockSecuritySettingRepo{
			findDefaultByTenantIDFn: func(int64) (*secpolicy.SecuritySetting, error) {
				return &secpolicy.SecuritySetting{
					PasswordConfig: datatypes.JSON([]byte(`{"expiry_days":1}`)),
				}, nil
			},
		}
		svc := &loginService{
			userRepo:            userRepo,
			securitySettingRepo: settingRepo,
		}
		svc.checkPasswordExpiry(context.Background(), user, 1)
		assert.True(t, user.ForcePasswordChange)
	})

	t.Run("password expired UpdateByID error does not panic", func(t *testing.T) {
		userRepo := &mockUserRepo{
			updateByIDFn: func(id any, data any) (*User, error) {
				return nil, errors.New("db error")
			},
		}
		pastTime := time.Now().AddDate(0, 0, -10)
		user := &User{UserID: 42, PasswordChangedAt: &pastTime}
		settingRepo := &mockSecuritySettingRepo{
			findDefaultByTenantIDFn: func(int64) (*secpolicy.SecuritySetting, error) {
				return &secpolicy.SecuritySetting{
					PasswordConfig: datatypes.JSON([]byte(`{"expiry_days":1}`)),
				}, nil
			},
		}
		svc := &loginService{
			userRepo:            userRepo,
			securitySettingRepo: settingRepo,
		}
		svc.checkPasswordExpiry(context.Background(), user, 1)
		assert.True(t, user.ForcePasswordChange)
	})
}

// ---------------------------------------------------------------------------
// TestLogout
// ---------------------------------------------------------------------------

func TestLogout(t *testing.T) {
	t.Run("invalid token format returns nil", func(t *testing.T) {
		svc := &loginService{}
		err := svc.Logout(context.Background(), "not.a.valid.token")
		assert.NoError(t, err)
	})
}

// ---------------------------------------------------------------------------
// TestLogin_ForcePasswordChange
// ---------------------------------------------------------------------------

// ---------------------------------------------------------------------------
// TestLogin_MFAChallenge — covers the MFA path in the internal Login function
// (service_login.go:461-462)
// ---------------------------------------------------------------------------

func TestLogin_MFAChallenge(t *testing.T) {
	const correctPassword = "S3cur3P@ss!"

	initTestJWTKeysService(t)
	gormDB, mock := newMockGormDB(t)
	mock.ExpectBegin()
	mock.ExpectCommit()

	idp := buildActiveIdentityProvider()
	idp.TenantID = 42
	client := buildActiveClient()
	client.IdentityProvider = idp
	user := buildActiveUser(t, correctPassword)
	user.IsTOTPEnabled = true
	now := time.Now()
	user.MFAEnabledAt = &now

	userRepo := &mockUserRepo{
		findByUsernameFn: func(_ string) (*User, error) {
			return user, nil
		},
	}
	clientRepo := &mockClientRepo{
		findSystemFn: func() (*Client, error) {
			return client, nil
		},
	}
	userIdentityRepo := &mockUserIdentityRepo{
		findByUserIDAndClientIDFn: func(_, _ int64) (*UserIdentity, error) {
			return &UserIdentity{Sub: "sub-mfa"}, nil
		},
	}
	securitySettingRepo := &mockSecuritySettingRepo{
		findDefaultByTenantIDFn: func(tenantID int64) (*secpolicy.SecuritySetting, error) {
			require.Equal(t, int64(42), tenantID)
			return &secpolicy.SecuritySetting{
				MFAConfig: datatypes.JSON([]byte(`{"required":true,"allowed_methods":["totp","backup_code","webauthn"]}`)),
			}, nil
		},
	}

	svc := NewLoginService(
		gormDB,
		clientRepo,
		userRepo,
		&mockUserTokenRepo{},
		userIdentityRepo,
		&mockIdentityProviderRepo{},
		&mockAuthEventService{},
		nil,
		securitySettingRepo,
	)

	svc.SetMFAFactorAuthenticator(&mockMFAAuthenticator{
		enrolledFn: func(int64) ([]string, error) { return []string{"totp", "backup_code"}, nil },
	})

	resp, err := svc.Login(context.Background(), "int-mfa-required", correctPassword, nil, nil)
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.True(t, resp.MFARequired)
	assert.Empty(t, resp.AccessToken)
	assert.Empty(t, resp.IDToken)
	assert.Empty(t, resp.RefreshToken)
	require.NotNil(t, resp.MFAChallengeToken)
	assert.ElementsMatch(t, []string{"totp", "backup_code"}, resp.MFAAllowedMethods)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// ---------------------------------------------------------------------------
// TestLogout_InvalidClaims — covers the !ok path in Logout
// (service_login.go:504-507) where ParseUnverified returns claims that
// are not MapClaims.
// ---------------------------------------------------------------------------

func TestLogout_InvalidClaims(t *testing.T) {
	svc := &loginService{sessionService: &mockLogoutSessionService{}, userRepo: &mockLogoutUserRepo{}}
	err := svc.Logout(context.Background(), "eyJhbGciOiJub25lIn0.eyJzdWIiOiIxMjM0NTY3ODkwIn0.")
	require.NoError(t, err)
}

// ---------------------------------------------------------------------------
// TestLogin_ForcePasswordChange
// ---------------------------------------------------------------------------

func TestLogin_ForcePasswordChange(t *testing.T) {
	const correctPassword = "S3cur3P@ss!"

	initTestJWTKeysService(t)
	gormDB, mock := newMockGormDB(t)
	mock.ExpectBegin()
	mock.ExpectCommit()

	clientRepo := &mockClientRepo{
		findSystemFn: func() (*Client, error) { return buildActiveClient(), nil },
	}
	userRepo := &mockUserRepo{
		findByUsernameFn: func(_ string) (*User, error) {
			u := buildActiveUser(t, correctPassword)
			u.ForcePasswordChange = true
			return u, nil
		},
	}
	userIdentityRepo := &mockUserIdentityRepo{
		findByUserIDAndClientIDFn: func(_, _ int64) (*UserIdentity, error) {
			return &UserIdentity{Sub: "sub-fpc"}, nil
		},
	}

	svc := NewLoginService(gormDB, clientRepo, userRepo, &mockUserTokenRepo{},
		userIdentityRepo, &mockIdentityProviderRepo{}, &mockAuthEventService{}, nil, nil)
	resp, err := svc.Login(context.Background(), "int-force-change", correctPassword, nil, nil)
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.True(t, resp.RequirePasswordChange)
	assert.Empty(t, resp.AccessToken)
	assert.Empty(t, resp.IDToken)
	assert.Empty(t, resp.RefreshToken)
	assert.NoError(t, mock.ExpectationsWereMet())
}
