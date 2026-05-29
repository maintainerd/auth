package oauth

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/authevent"
	"github.com/maintainerd/auth/internal/platform/apperror"
	"gorm.io/gorm"
)

// ---------------------------------------------------------------------------
// Mock: ClientRepository
// ---------------------------------------------------------------------------

type mockClientRepo struct {
	findByClientIDAndIdentityProviderFn func(clientID, providerID string) (*Client, error)
	findSystemFn                        func() (*Client, error)
	createFn                            func(*Client) (*Client, error)
}

func (m *mockClientRepo) WithTx(_ *gorm.DB) ClientRepository { return m }
func (m *mockClientRepo) FindSystem() (*Client, error) {
	if m.findSystemFn != nil {
		return m.findSystemFn()
	}
	return nil, nil
}
func (m *mockClientRepo) FindByClientIDAndIdentityProvider(clientID, providerID string) (*Client, error) {
	if m.findByClientIDAndIdentityProviderFn != nil {
		return m.findByClientIDAndIdentityProviderFn(clientID, providerID)
	}
	return nil, nil
}
func (m *mockClientRepo) Create(e *Client) (*Client, error) {
	if m.createFn != nil {
		return m.createFn(e)
	}
	return e, nil
}
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
// Mock: ClientURIRepository
// ---------------------------------------------------------------------------

type mockClientURIRepo struct {
	createFn func(*ClientURI) (*ClientURI, error)
}

func (m *mockClientURIRepo) WithTx(_ *gorm.DB) ClientURIRepository { return m }
func (m *mockClientURIRepo) Create(e *ClientURI) (*ClientURI, error) {
	if m.createFn != nil {
		return m.createFn(e)
	}
	return e, nil
}
func (m *mockClientURIRepo) CreateOrUpdate(e *ClientURI) (*ClientURI, error)    { return e, nil }
func (m *mockClientURIRepo) FindAll(p ...string) ([]ClientURI, error)           { return nil, nil }
func (m *mockClientURIRepo) FindByUUID(id any, p ...string) (*ClientURI, error) { return nil, nil }
func (m *mockClientURIRepo) FindByUUIDs(ids []string, p ...string) ([]ClientURI, error) {
	return nil, nil
}
func (m *mockClientURIRepo) FindByID(id any, p ...string) (*ClientURI, error) { return nil, nil }
func (m *mockClientURIRepo) UpdateByUUID(id, data any) (*ClientURI, error)    { return nil, nil }
func (m *mockClientURIRepo) UpdateByID(id, data any) (*ClientURI, error)      { return nil, nil }
func (m *mockClientURIRepo) DeleteByUUID(id any) error                        { return nil }
func (m *mockClientURIRepo) DeleteByID(id any) error                          { return nil }
func (m *mockClientURIRepo) Paginate(c map[string]any, pg, lim int, p ...string) (*PaginationResult[ClientURI], error) {
	return nil, nil
}

// ---------------------------------------------------------------------------
// Mock: OAuthAuthorizationCodeRepository
// ---------------------------------------------------------------------------

type mockOAuthAuthCodeRepo struct {
	createFn         func(*OAuthAuthorizationCode) (*OAuthAuthorizationCode, error)
	findByCodeHashFn func(string) (*OAuthAuthorizationCode, error)
	markUsedFn       func(int64) error
}

func (m *mockOAuthAuthCodeRepo) WithTx(_ *gorm.DB) OAuthAuthorizationCodeRepository { return m }
func (m *mockOAuthAuthCodeRepo) Create(e *OAuthAuthorizationCode) (*OAuthAuthorizationCode, error) {
	if m.createFn != nil {
		return m.createFn(e)
	}
	return e, nil
}
func (m *mockOAuthAuthCodeRepo) FindByCodeHash(hash string) (*OAuthAuthorizationCode, error) {
	if m.findByCodeHashFn != nil {
		return m.findByCodeHashFn(hash)
	}
	return nil, nil
}
func (m *mockOAuthAuthCodeRepo) MarkUsed(id int64) error {
	if m.markUsedFn != nil {
		return m.markUsedFn(id)
	}
	return nil
}
func (m *mockOAuthAuthCodeRepo) DeleteExpired(before time.Time) (int64, error) { return 0, nil }
func (m *mockOAuthAuthCodeRepo) CreateOrUpdate(e *OAuthAuthorizationCode) (*OAuthAuthorizationCode, error) {
	return e, nil
}
func (m *mockOAuthAuthCodeRepo) FindAll(p ...string) ([]OAuthAuthorizationCode, error) {
	return nil, nil
}
func (m *mockOAuthAuthCodeRepo) FindByUUID(id any, p ...string) (*OAuthAuthorizationCode, error) {
	return nil, nil
}
func (m *mockOAuthAuthCodeRepo) FindByUUIDs(ids []string, p ...string) ([]OAuthAuthorizationCode, error) {
	return nil, nil
}
func (m *mockOAuthAuthCodeRepo) FindByID(id any, p ...string) (*OAuthAuthorizationCode, error) {
	return nil, nil
}
func (m *mockOAuthAuthCodeRepo) UpdateByUUID(id, data any) (*OAuthAuthorizationCode, error) {
	return nil, nil
}
func (m *mockOAuthAuthCodeRepo) UpdateByID(id, data any) (*OAuthAuthorizationCode, error) {
	return nil, nil
}
func (m *mockOAuthAuthCodeRepo) DeleteByUUID(id any) error { return nil }
func (m *mockOAuthAuthCodeRepo) DeleteByID(id any) error   { return nil }
func (m *mockOAuthAuthCodeRepo) Paginate(c map[string]any, pg, lim int, p ...string) (*PaginationResult[OAuthAuthorizationCode], error) {
	return nil, nil
}

// ---------------------------------------------------------------------------
// Mock: OAuthConsentChallengeRepository
// ---------------------------------------------------------------------------

type mockOAuthConsentChallRepo struct {
	createFn                func(*OAuthConsentChallenge) (*OAuthConsentChallenge, error)
	findChallengeByUUIDFn   func(uuid.UUID) (*OAuthConsentChallenge, error)
	deleteChallengeByUUIDFn func(uuid.UUID) error
}

func (m *mockOAuthConsentChallRepo) WithTx(_ *gorm.DB) OAuthConsentChallengeRepository { return m }
func (m *mockOAuthConsentChallRepo) Create(e *OAuthConsentChallenge) (*OAuthConsentChallenge, error) {
	if m.createFn != nil {
		return m.createFn(e)
	}
	return e, nil
}
func (m *mockOAuthConsentChallRepo) FindChallengeByUUID(id uuid.UUID) (*OAuthConsentChallenge, error) {
	if m.findChallengeByUUIDFn != nil {
		return m.findChallengeByUUIDFn(id)
	}
	return nil, nil
}
func (m *mockOAuthConsentChallRepo) DeleteChallengeByUUID(id uuid.UUID) error {
	if m.deleteChallengeByUUIDFn != nil {
		return m.deleteChallengeByUUIDFn(id)
	}
	return nil
}
func (m *mockOAuthConsentChallRepo) DeleteExpired(before time.Time) (int64, error) { return 0, nil }
func (m *mockOAuthConsentChallRepo) CreateOrUpdate(e *OAuthConsentChallenge) (*OAuthConsentChallenge, error) {
	return e, nil
}
func (m *mockOAuthConsentChallRepo) FindAll(p ...string) ([]OAuthConsentChallenge, error) {
	return nil, nil
}
func (m *mockOAuthConsentChallRepo) FindByUUID(id any, p ...string) (*OAuthConsentChallenge, error) {
	return nil, nil
}
func (m *mockOAuthConsentChallRepo) FindByUUIDs(ids []string, p ...string) ([]OAuthConsentChallenge, error) {
	return nil, nil
}
func (m *mockOAuthConsentChallRepo) FindByID(id any, p ...string) (*OAuthConsentChallenge, error) {
	return nil, nil
}
func (m *mockOAuthConsentChallRepo) UpdateByUUID(id, data any) (*OAuthConsentChallenge, error) {
	return nil, nil
}
func (m *mockOAuthConsentChallRepo) UpdateByID(id, data any) (*OAuthConsentChallenge, error) {
	return nil, nil
}
func (m *mockOAuthConsentChallRepo) DeleteByUUID(id any) error { return nil }
func (m *mockOAuthConsentChallRepo) DeleteByID(id any) error   { return nil }
func (m *mockOAuthConsentChallRepo) Paginate(c map[string]any, pg, lim int, p ...string) (*PaginationResult[OAuthConsentChallenge], error) {
	return nil, nil
}

// ---------------------------------------------------------------------------
// Mock: OAuthConsentGrantRepository
// ---------------------------------------------------------------------------

type mockOAuthConsentGrantRepo struct {
	findByUserAndClientFn   func(int64, int64) (*OAuthConsentGrant, error)
	upsertFn                func(*OAuthConsentGrant) (*OAuthConsentGrant, error)
	deleteByUserAndClientFn func(int64, int64) error
	findByUserIDFn          func(int64) ([]OAuthConsentGrant, error)
}

func (m *mockOAuthConsentGrantRepo) WithTx(_ *gorm.DB) OAuthConsentGrantRepository { return m }
func (m *mockOAuthConsentGrantRepo) FindByUserAndClient(userID, clientID int64) (*OAuthConsentGrant, error) {
	if m.findByUserAndClientFn != nil {
		return m.findByUserAndClientFn(userID, clientID)
	}
	return nil, nil
}
func (m *mockOAuthConsentGrantRepo) Upsert(g *OAuthConsentGrant) (*OAuthConsentGrant, error) {
	if m.upsertFn != nil {
		return m.upsertFn(g)
	}
	return g, nil
}
func (m *mockOAuthConsentGrantRepo) DeleteByUserAndClient(userID, clientID int64) error {
	if m.deleteByUserAndClientFn != nil {
		return m.deleteByUserAndClientFn(userID, clientID)
	}
	return nil
}
func (m *mockOAuthConsentGrantRepo) FindByUserID(userID int64) ([]OAuthConsentGrant, error) {
	if m.findByUserIDFn != nil {
		return m.findByUserIDFn(userID)
	}
	return nil, nil
}
func (m *mockOAuthConsentGrantRepo) Create(e *OAuthConsentGrant) (*OAuthConsentGrant, error) {
	return e, nil
}
func (m *mockOAuthConsentGrantRepo) CreateOrUpdate(e *OAuthConsentGrant) (*OAuthConsentGrant, error) {
	return e, nil
}
func (m *mockOAuthConsentGrantRepo) FindAll(p ...string) ([]OAuthConsentGrant, error) {
	return nil, nil
}
func (m *mockOAuthConsentGrantRepo) FindByUUID(id any, p ...string) (*OAuthConsentGrant, error) {
	return nil, nil
}
func (m *mockOAuthConsentGrantRepo) FindByUUIDs(ids []string, p ...string) ([]OAuthConsentGrant, error) {
	return nil, nil
}
func (m *mockOAuthConsentGrantRepo) FindByID(id any, p ...string) (*OAuthConsentGrant, error) {
	return nil, nil
}
func (m *mockOAuthConsentGrantRepo) UpdateByUUID(id, data any) (*OAuthConsentGrant, error) {
	return nil, nil
}
func (m *mockOAuthConsentGrantRepo) UpdateByID(id, data any) (*OAuthConsentGrant, error) {
	return nil, nil
}
func (m *mockOAuthConsentGrantRepo) DeleteByUUID(id any) error { return nil }
func (m *mockOAuthConsentGrantRepo) DeleteByID(id any) error   { return nil }
func (m *mockOAuthConsentGrantRepo) Paginate(c map[string]any, pg, lim int, p ...string) (*PaginationResult[OAuthConsentGrant], error) {
	return nil, nil
}

// ---------------------------------------------------------------------------
// Mock: OAuthRefreshTokenRepository
// ---------------------------------------------------------------------------

type mockOAuthRefreshTokenRepo struct {
	findByTokenHashFn func(string) (*OAuthRefreshToken, error)
	revokeByIDFn      func(int64) error
}

func (m *mockOAuthRefreshTokenRepo) WithTx(_ *gorm.DB) OAuthRefreshTokenRepository { return m }
func (m *mockOAuthRefreshTokenRepo) FindByTokenHash(hash string) (*OAuthRefreshToken, error) {
	if m.findByTokenHashFn != nil {
		return m.findByTokenHashFn(hash)
	}
	return nil, nil
}
func (m *mockOAuthRefreshTokenRepo) RevokeByID(id int64) error {
	if m.revokeByIDFn != nil {
		return m.revokeByIDFn(id)
	}
	return nil
}
func (m *mockOAuthRefreshTokenRepo) FindActiveByUserAndClient(userID, clientID int64) ([]OAuthRefreshToken, error) {
	return nil, nil
}
func (m *mockOAuthRefreshTokenRepo) RevokeByFamily(familyID uuid.UUID) (int64, error) {
	return 0, nil
}
func (m *mockOAuthRefreshTokenRepo) RevokeByUserAndClient(userID, clientID int64) (int64, error) {
	return 0, nil
}
func (m *mockOAuthRefreshTokenRepo) RevokeByUserID(userID int64) (int64, error) { return 0, nil }
func (m *mockOAuthRefreshTokenRepo) UpdateLastUsed(tokenID int64) error         { return nil }
func (m *mockOAuthRefreshTokenRepo) DeleteExpired(before time.Time) (int64, error) {
	return 0, nil
}
func (m *mockOAuthRefreshTokenRepo) CountByUserAndClient(userID, clientID int64) (int64, error) {
	return 0, nil
}
func (m *mockOAuthRefreshTokenRepo) Create(e *OAuthRefreshToken) (*OAuthRefreshToken, error) {
	return e, nil
}
func (m *mockOAuthRefreshTokenRepo) CreateOrUpdate(e *OAuthRefreshToken) (*OAuthRefreshToken, error) {
	return e, nil
}
func (m *mockOAuthRefreshTokenRepo) FindAll(p ...string) ([]OAuthRefreshToken, error) {
	return nil, nil
}
func (m *mockOAuthRefreshTokenRepo) FindByUUID(id any, p ...string) (*OAuthRefreshToken, error) {
	return nil, nil
}
func (m *mockOAuthRefreshTokenRepo) FindByUUIDs(ids []string, p ...string) ([]OAuthRefreshToken, error) {
	return nil, nil
}
func (m *mockOAuthRefreshTokenRepo) FindByID(id any, p ...string) (*OAuthRefreshToken, error) {
	return nil, nil
}
func (m *mockOAuthRefreshTokenRepo) UpdateByUUID(id, data any) (*OAuthRefreshToken, error) {
	return nil, nil
}
func (m *mockOAuthRefreshTokenRepo) UpdateByID(id, data any) (*OAuthRefreshToken, error) {
	return nil, nil
}
func (m *mockOAuthRefreshTokenRepo) DeleteByUUID(id any) error { return nil }
func (m *mockOAuthRefreshTokenRepo) DeleteByID(id any) error   { return nil }
func (m *mockOAuthRefreshTokenRepo) Paginate(c map[string]any, pg, lim int, p ...string) (*PaginationResult[OAuthRefreshToken], error) {
	return nil, nil
}

// ---------------------------------------------------------------------------
// Mock: UserRepository
// ---------------------------------------------------------------------------

type mockUserRepo struct {
	findByIDFn             func(any, ...string) (*User, error)
	findByEmailFn          func(string) (*User, error)
	findBySubAndClientIDFn func(string, string) (*User, error)
}

func (m *mockUserRepo) WithTx(_ *gorm.DB) UserRepository { return m }
func (m *mockUserRepo) FindByEmail(email string) (*User, error) {
	if m.findByEmailFn != nil {
		return m.findByEmailFn(email)
	}
	return nil, nil
}
func (m *mockUserRepo) FindBySubAndClientID(sub, clientID string) (*User, error) {
	if m.findBySubAndClientIDFn != nil {
		return m.findBySubAndClientIDFn(sub, clientID)
	}
	return nil, nil
}
func (m *mockUserRepo) FindByID(id any, p ...string) (*User, error) {
	if m.findByIDFn != nil {
		return m.findByIDFn(id, p...)
	}
	return nil, nil
}
func (m *mockUserRepo) Create(e *User) (*User, error)                 { return e, nil }
func (m *mockUserRepo) CreateOrUpdate(e *User) (*User, error)         { return e, nil }
func (m *mockUserRepo) FindAll(p ...string) ([]User, error)           { return nil, nil }
func (m *mockUserRepo) FindByUUID(id any, p ...string) (*User, error) { return nil, nil }
func (m *mockUserRepo) FindByUUIDs(ids []string, p ...string) ([]User, error) {
	return nil, nil
}
func (m *mockUserRepo) UpdateByUUID(id, data any) (*User, error) { return nil, nil }
func (m *mockUserRepo) UpdateByID(id, data any) (*User, error)   { return nil, nil }
func (m *mockUserRepo) DeleteByUUID(id any) error                { return nil }
func (m *mockUserRepo) DeleteByID(id any) error                  { return nil }
func (m *mockUserRepo) Paginate(c map[string]any, pg, lim int, p ...string) (*PaginationResult[User], error) {
	return nil, nil
}

// ---------------------------------------------------------------------------
// Mock: UserIdentityRepository
// ---------------------------------------------------------------------------

type mockUserIdentityRepo struct {
	findByUserIDAndClientIDFn func(int64, int64) (*UserIdentity, error)
}

func (m *mockUserIdentityRepo) WithTx(_ *gorm.DB) UserIdentityRepository { return m }
func (m *mockUserIdentityRepo) FindByUserIDAndClientID(userID, clientID int64) (*UserIdentity, error) {
	if m.findByUserIDAndClientIDFn != nil {
		return m.findByUserIDAndClientIDFn(userID, clientID)
	}
	return nil, nil
}
func (m *mockUserIdentityRepo) Create(e *UserIdentity) (*UserIdentity, error)         { return e, nil }
func (m *mockUserIdentityRepo) CreateOrUpdate(e *UserIdentity) (*UserIdentity, error) { return e, nil }
func (m *mockUserIdentityRepo) FindAll(p ...string) ([]UserIdentity, error)           { return nil, nil }
func (m *mockUserIdentityRepo) FindByUUID(id any, p ...string) (*UserIdentity, error) {
	return nil, nil
}
func (m *mockUserIdentityRepo) FindByUUIDs(ids []string, p ...string) ([]UserIdentity, error) {
	return nil, nil
}
func (m *mockUserIdentityRepo) FindByID(id any, p ...string) (*UserIdentity, error) {
	return nil, nil
}
func (m *mockUserIdentityRepo) UpdateByUUID(id, data any) (*UserIdentity, error) { return nil, nil }
func (m *mockUserIdentityRepo) UpdateByID(id, data any) (*UserIdentity, error)   { return nil, nil }
func (m *mockUserIdentityRepo) DeleteByUUID(id any) error                        { return nil }
func (m *mockUserIdentityRepo) DeleteByID(id any) error                          { return nil }
func (m *mockUserIdentityRepo) Paginate(c map[string]any, pg, lim int, p ...string) (*PaginationResult[UserIdentity], error) {
	return nil, nil
}

// ---------------------------------------------------------------------------
// Mock: authevent.AuthEventService
// ---------------------------------------------------------------------------

type mockAuthEventService struct{}

func (m *mockAuthEventService) Log(_ context.Context, _ authevent.AuthEventInput) {}
func (m *mockAuthEventService) FindPaginated(_ context.Context, _ authevent.AuthEventRepositoryGetFilter) (*authevent.PaginationResult[authevent.AuthEventServiceDataResult], error) {
	return &authevent.PaginationResult[authevent.AuthEventServiceDataResult]{}, nil
}
func (m *mockAuthEventService) FindByUUID(_ context.Context, _ int64, _ uuid.UUID) (*authevent.AuthEventServiceDataResult, error) {
	return nil, nil
}
func (m *mockAuthEventService) CountByEventType(_ context.Context, _ string, _ int64) (int64, error) {
	return 0, nil
}
func (m *mockAuthEventService) DeleteOlderThan(_ context.Context, _ time.Time) (int64, error) {
	return 0, nil
}

// ---------------------------------------------------------------------------
// Mock: OAuthTokenService (handler tests)
// ---------------------------------------------------------------------------

type mockOAuthTokenService struct {
	exchangeFn   func(context.Context, OAuthTokenRequestDTO, OAuthClientCredentials) (*OAuthTokenResult, *apperror.OAuthError)
	revokeFn     func(context.Context, OAuthRevokeRequestDTO, OAuthClientCredentials) *apperror.OAuthError
	introspectFn func(context.Context, OAuthIntrospectRequestDTO) (*OAuthIntrospectResponseDTO, *apperror.OAuthError)
}

func (m *mockOAuthTokenService) Exchange(ctx context.Context, req OAuthTokenRequestDTO, creds OAuthClientCredentials) (*OAuthTokenResult, *apperror.OAuthError) {
	if m.exchangeFn != nil {
		return m.exchangeFn(ctx, req, creds)
	}
	return nil, nil
}
func (m *mockOAuthTokenService) Revoke(ctx context.Context, req OAuthRevokeRequestDTO, creds OAuthClientCredentials) *apperror.OAuthError {
	if m.revokeFn != nil {
		return m.revokeFn(ctx, req, creds)
	}
	return nil
}
func (m *mockOAuthTokenService) Introspect(ctx context.Context, req OAuthIntrospectRequestDTO) (*OAuthIntrospectResponseDTO, *apperror.OAuthError) {
	if m.introspectFn != nil {
		return m.introspectFn(ctx, req)
	}
	return nil, nil
}

// ---------------------------------------------------------------------------
// Mock: OAuthConsentService (handler tests)
// ---------------------------------------------------------------------------

type mockOAuthConsentService struct {
	listGrantsFn  func(context.Context, int64) ([]OAuthConsentGrantResponseDTO, error)
	revokeGrantFn func(context.Context, uuid.UUID, int64) error
}

func (m *mockOAuthConsentService) ListGrants(ctx context.Context, userID int64) ([]OAuthConsentGrantResponseDTO, error) {
	if m.listGrantsFn != nil {
		return m.listGrantsFn(ctx, userID)
	}
	return nil, nil
}
func (m *mockOAuthConsentService) RevokeGrant(ctx context.Context, grantUUID uuid.UUID, userID int64) error {
	if m.revokeGrantFn != nil {
		return m.revokeGrantFn(ctx, grantUUID, userID)
	}
	return nil
}

// ---------------------------------------------------------------------------
// Mock: OAuthAuthorizeService (handler tests)
// ---------------------------------------------------------------------------

type mockOAuthAuthorizeService struct {
	authorizeFn           func(context.Context, OAuthAuthorizeRequestDTO, int64) (*OAuthAuthorizeResult, *apperror.OAuthError)
	getConsentChallengeFn func(context.Context, uuid.UUID, int64) (*OAuthConsentChallengeResponseDTO, error)
	handleConsentFn       func(context.Context, OAuthConsentDecisionDTO, int64) (*OAuthConsentDecisionResult, *apperror.OAuthError)
}

func (m *mockOAuthAuthorizeService) Authorize(ctx context.Context, req OAuthAuthorizeRequestDTO, userID int64) (*OAuthAuthorizeResult, *apperror.OAuthError) {
	if m.authorizeFn != nil {
		return m.authorizeFn(ctx, req, userID)
	}
	return nil, nil
}
func (m *mockOAuthAuthorizeService) GetConsentChallenge(ctx context.Context, challengeUUID uuid.UUID, userID int64) (*OAuthConsentChallengeResponseDTO, error) {
	if m.getConsentChallengeFn != nil {
		return m.getConsentChallengeFn(ctx, challengeUUID, userID)
	}
	return nil, nil
}
func (m *mockOAuthAuthorizeService) HandleConsent(ctx context.Context, decision OAuthConsentDecisionDTO, userID int64) (*OAuthConsentDecisionResult, *apperror.OAuthError) {
	if m.handleConsentFn != nil {
		return m.handleConsentFn(ctx, decision, userID)
	}
	return nil, nil
}
