package oauth

import (
	"context"
	"errors"
	"regexp"
	"strings"
	"testing"
	"time"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/maintainerd/auth/internal/platform/crypto"
	"github.com/maintainerd/auth/internal/platform/ptr"
	"github.com/maintainerd/auth/internal/shared"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func newOAuthPARSvc(
	db *gorm.DB,
	clientURIRepo *mockClientURIRepo,
	parRepo *mockOAuthPARRepo,
	authEventSvc *mockAuthEventService,
) OAuthPARService {
	return &oauthPARService{
		db:               db,
		clientRepo:       &mockClientRepo{},
		clientURIRepo:    clientURIRepo,
		parRepo:          parRepo,
		authEventService: authEventSvc,
	}
}

type mockOAuthPARRepo struct {
	findByRequestURIHashFn func(string) (*OAuthPARRequest, error)
	markUsedFn             func(int64) error
	createFn               func(*OAuthPARRequest) (*OAuthPARRequest, error)
}

func (m *mockOAuthPARRepo) WithTx(_ *gorm.DB) OAuthPARRequestRepository { return m }
func (m *mockOAuthPARRepo) FindByRequestURIHash(hash string) (*OAuthPARRequest, error) {
	if m.findByRequestURIHashFn != nil {
		return m.findByRequestURIHashFn(hash)
	}
	return nil, nil
}
func (m *mockOAuthPARRepo) MarkUsed(id int64) error {
	if m.markUsedFn != nil {
		return m.markUsedFn(id)
	}
	return nil
}
func (m *mockOAuthPARRepo) Create(e *OAuthPARRequest) (*OAuthPARRequest, error) {
	if m.createFn != nil {
		return m.createFn(e)
	}
	return e, nil
}
func (m *mockOAuthPARRepo) DeleteExpired(_ time.Time) (int64, error) { return 0, nil }
func (m *mockOAuthPARRepo) CreateOrUpdate(e *OAuthPARRequest) (*OAuthPARRequest, error) {
	return e, nil
}
func (m *mockOAuthPARRepo) FindAll(_ ...string) ([]OAuthPARRequest, error)          { return nil, nil }
func (m *mockOAuthPARRepo) FindByUUID(_ any, _ ...string) (*OAuthPARRequest, error) { return nil, nil }
func (m *mockOAuthPARRepo) FindByUUIDs(_ []string, _ ...string) ([]OAuthPARRequest, error) {
	return nil, nil
}
func (m *mockOAuthPARRepo) FindByID(_ any, _ ...string) (*OAuthPARRequest, error) { return nil, nil }
func (m *mockOAuthPARRepo) UpdateByUUID(_, _ any) (*OAuthPARRequest, error)       { return nil, nil }
func (m *mockOAuthPARRepo) UpdateByID(_, _ any) (*OAuthPARRequest, error)         { return nil, nil }
func (m *mockOAuthPARRepo) DeleteByUUID(_ any) error                              { return nil }
func (m *mockOAuthPARRepo) DeleteByID(_ any) error                                { return nil }
func (m *mockOAuthPARRepo) Paginate(_ map[string]any, _, _ int, _ ...string) (*PaginationResult[OAuthPARRequest], error) {
	return nil, nil
}

// ── TestOAuthPARService_Push ────────────────────────────────────────────────

func TestOAuthPARService_Push(t *testing.T) {
	ctx := context.Background()

	t.Run("client not found", func(t *testing.T) {
		db, mock := newMockDB(t)
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT`)).WillReturnError(gorm.ErrRecordNotFound)

		svc := newOAuthPARSvc(db, &mockClientURIRepo{}, &mockOAuthPARRepo{}, &mockAuthEventService{})

		_, oerr := svc.Push(ctx, OAuthPARRequestDTO{}, OAuthClientCredentials{ClientID: "unknown"})
		require.NotNil(t, oerr)
		assert.Equal(t, "invalid_client", oerr.Code)
	})

	t.Run("db error resolving client", func(t *testing.T) {
		db, mock := newMockDB(t)
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT`)).WillReturnError(errors.New("db error"))

		svc := newOAuthPARSvc(db, &mockClientURIRepo{}, &mockOAuthPARRepo{}, &mockAuthEventService{})

		_, oerr := svc.Push(ctx, OAuthPARRequestDTO{}, OAuthClientCredentials{ClientID: "unknown"})
		require.NotNil(t, oerr)
		assert.Equal(t, "server_error", oerr.Code)
	})

	t.Run("secret mismatch", func(t *testing.T) {
		db, mock := newMockDB(t)
		mock.MatchExpectationsInOrder(false)
		secretHash := "hashed-secret"
		rows := sqlmock.NewRows([]string{
			"client_id", "client_uuid", "tenant_id", "identity_provider_id", "name", "display_name",
			"client_type", "domain", "identifier", "secret_hash", "status",
			"is_default", "is_system", "token_endpoint_auth_method",
			"grant_types", "response_types", "access_token_ttl", "refresh_token_ttl",
			"require_consent", "created_at", "updated_at",
		}).AddRow(
			10, uuid.New(), 1, int64(100), "test-client", "Test Client",
			"spa", nil, "my-client", secretHash, "active",
			false, false, TokenAuthMethodSecretBasic,
			pq.StringArray{GrantTypeAuthorizationCode}, pq.StringArray{ResponseTypeCode}, nil, nil,
			false, time.Now(), time.Now(),
		)
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT`)).WillReturnRows(rows)
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT`)).WillReturnRows(sqlmock.NewRows(nil))
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT`)).WillReturnRows(sqlmock.NewRows(nil))

		svc := newOAuthPARSvc(db, &mockClientURIRepo{}, &mockOAuthPARRepo{}, &mockAuthEventService{})

		_, oerr := svc.Push(ctx, OAuthPARRequestDTO{
			RedirectURI:  "https://example.com/callback",
			ResponseType: ResponseTypeCode,
		}, OAuthClientCredentials{ClientID: "my-client", ClientSecret: "wrong-secret"})
		require.NotNil(t, oerr)
		assert.Equal(t, "invalid_client", oerr.Code)
	})

	t.Run("redirect URI mismatch", func(t *testing.T) {
		db, mock := newMockDB(t)
		mock.MatchExpectationsInOrder(false)
		rows := sqlmock.NewRows([]string{
			"client_id", "client_uuid", "tenant_id", "identity_provider_id", "name", "display_name",
			"client_type", "domain", "identifier", "secret", "secret_hash", "status",
			"is_default", "is_system", "token_endpoint_auth_method",
			"grant_types", "response_types", "access_token_ttl", "refresh_token_ttl",
			"require_consent", "created_at", "updated_at",
		}).AddRow(
			10, uuid.New(), 1, int64(100), "test-client", "Test Client",
			"spa", nil, "my-client", nil, nil, "active",
			false, false, "none",
			pq.StringArray{GrantTypeAuthorizationCode}, pq.StringArray{ResponseTypeCode}, nil, nil,
			false, time.Now(), time.Now(),
		)
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT`)).WillReturnRows(rows)
		mock.ExpectQuery(`FROM "identity_providers"`).WillReturnRows(sqlmock.NewRows(nil))
		mock.ExpectQuery(`FROM "client_uris"`).WillReturnRows(
			sqlmock.NewRows([]string{"client_uri_id", "client_uri_uuid", "tenant_id", "client_id", "uri", "type", "created_at", "updated_at"}).
				AddRow(1, uuid.New(), 1, 10, "https://other.example.com/cb", shared.ClientURITypeRedirect, time.Now(), time.Now()),
		)

		svc := newOAuthPARSvc(db, &mockClientURIRepo{}, &mockOAuthPARRepo{}, &mockAuthEventService{})

		_, oerr := svc.Push(ctx, OAuthPARRequestDTO{
			RedirectURI:         "https://example.com/callback",
			ResponseType:        ResponseTypeCode,
			CodeChallenge:       strings.Repeat("A", 43),
			CodeChallengeMethod: "S256",
		}, OAuthClientCredentials{ClientID: "my-client"})
		require.NotNil(t, oerr)
		assert.Equal(t, "invalid_request", oerr.Code)
		assert.Contains(t, oerr.Description, "registered redirect")
	})

	t.Run("create error", func(t *testing.T) {
		db, mock := newMockDB(t)
		mock.MatchExpectationsInOrder(false)
		rows := sqlmock.NewRows([]string{
			"client_id", "client_uuid", "tenant_id", "identity_provider_id", "name", "display_name",
			"client_type", "domain", "identifier", "secret", "secret_hash", "status",
			"is_default", "is_system", "token_endpoint_auth_method",
			"grant_types", "response_types", "access_token_ttl", "refresh_token_ttl",
			"require_consent", "created_at", "updated_at",
		}).AddRow(
			10, uuid.New(), 1, int64(100), "test-client", "Test Client",
			"spa", nil, "my-client", nil, nil, "active",
			false, false, "none",
			pq.StringArray{GrantTypeAuthorizationCode}, pq.StringArray{ResponseTypeCode}, nil, nil,
			false, time.Now(), time.Now(),
		)
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT`)).WillReturnRows(rows)
		mock.ExpectQuery(`FROM "identity_providers"`).WillReturnRows(sqlmock.NewRows(nil))
		mock.ExpectQuery(`FROM "client_uris"`).WillReturnRows(
			sqlmock.NewRows([]string{"client_uri_id", "client_uri_uuid", "tenant_id", "client_id", "uri", "type", "created_at", "updated_at"}).
				AddRow(1, uuid.New(), 1, 10, "https://example.com/callback", shared.ClientURITypeRedirect, time.Now(), time.Now()),
		)

		svc := newOAuthPARSvc(db, &mockClientURIRepo{},
			&mockOAuthPARRepo{
				createFn: func(_ *OAuthPARRequest) (*OAuthPARRequest, error) {
					return nil, errors.New("db error")
				},
			},
			&mockAuthEventService{})

		_, oerr := svc.Push(ctx, OAuthPARRequestDTO{
			RedirectURI:         "https://example.com/callback",
			ResponseType:        ResponseTypeCode,
			CodeChallenge:       strings.Repeat("A", 43),
			CodeChallengeMethod: "S256",
		}, OAuthClientCredentials{ClientID: "my-client"})
		require.NotNil(t, oerr)
		assert.Equal(t, "server_error", oerr.Code)
	})

	t.Run("no redirect URIs registered", func(t *testing.T) {
		db, mock := newMockDB(t)
		mock.MatchExpectationsInOrder(false)
		rows := sqlmock.NewRows([]string{
			"client_id", "client_uuid", "tenant_id", "identity_provider_id", "name", "display_name",
			"client_type", "domain", "identifier", "secret", "status",
			"is_default", "is_system", "token_endpoint_auth_method",
			"grant_types", "response_types", "access_token_ttl", "refresh_token_ttl",
			"require_consent", "created_at", "updated_at",
		}).AddRow(
			10, uuid.New(), 1, int64(100), "test-client", "Test Client",
			"spa", nil, "my-client", nil, "active",
			false, false, "none",
			pq.StringArray{GrantTypeAuthorizationCode}, pq.StringArray{ResponseTypeCode}, nil, nil,
			false, time.Now(), time.Now(),
		)
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT`)).WillReturnRows(rows)
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT`)).WillReturnRows(sqlmock.NewRows(nil))
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT`)).WillReturnRows(sqlmock.NewRows(nil))

		svc := newOAuthPARSvc(db, &mockClientURIRepo{}, &mockOAuthPARRepo{}, &mockAuthEventService{})

		_, oerr := svc.Push(ctx, OAuthPARRequestDTO{
			RedirectURI:         "https://example.com/callback",
			ResponseType:        ResponseTypeCode,
			CodeChallenge:       strings.Repeat("A", 43),
			CodeChallengeMethod: "S256",
		}, OAuthClientCredentials{ClientID: "my-client"})
		require.NotNil(t, oerr)
		assert.Equal(t, "invalid_request", oerr.Code)
	})

	t.Run("success", func(t *testing.T) {
		db, mock := newMockDB(t)
		mock.MatchExpectationsInOrder(false)
		rows := sqlmock.NewRows([]string{
			"client_id", "client_uuid", "tenant_id", "identity_provider_id", "name", "display_name",
			"client_type", "domain", "identifier", "secret", "status",
			"is_default", "is_system", "token_endpoint_auth_method",
			"grant_types", "response_types", "access_token_ttl", "refresh_token_ttl",
			"require_consent", "created_at", "updated_at",
		}).AddRow(
			10, uuid.New(), 1, int64(100), "test-client", "Test Client",
			"spa", nil, "my-client", nil, "active",
			false, false, "none",
			pq.StringArray{GrantTypeAuthorizationCode}, pq.StringArray{ResponseTypeCode}, nil, nil,
			false, time.Now(), time.Now(),
		)
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT`)).WillReturnRows(rows)
		mock.ExpectQuery(`FROM "identity_providers"`).WillReturnRows(sqlmock.NewRows(nil))
		mock.ExpectQuery(`FROM "client_uris"`).WillReturnRows(
			sqlmock.NewRows([]string{"client_uri_id", "client_uri_uuid", "tenant_id", "client_id", "uri", "type", "created_at", "updated_at"}).
				AddRow(1, uuid.New(), 1, 10, "https://example.com/callback", shared.ClientURITypeRedirect, time.Now(), time.Now()),
		)

		svc := newOAuthPARSvc(db, &mockClientURIRepo{}, &mockOAuthPARRepo{}, &mockAuthEventService{})

		result, oerr := svc.Push(ctx, OAuthPARRequestDTO{
			RedirectURI:         "https://example.com/callback",
			ResponseType:        ResponseTypeCode,
			CodeChallenge:       strings.Repeat("A", 43),
			CodeChallengeMethod: "S256",
		}, OAuthClientCredentials{ClientID: "my-client"})
		require.Nil(t, oerr)
		require.NotNil(t, result)
		assert.NotEmpty(t, result.RequestURI)
		assert.Contains(t, result.RequestURI, "urn:ietf:params:oauth:request-uri:")
		assert.Equal(t, 90, result.ExpiresIn)
	})

	t.Run("grant not allowed", func(t *testing.T) {
		db, mock := newMockDB(t)
		rows := sqlmock.NewRows([]string{
			"client_id", "client_uuid", "tenant_id", "identity_provider_id", "name", "display_name",
			"client_type", "domain", "identifier", "secret", "status",
			"is_default", "is_system", "token_endpoint_auth_method",
			"grant_types", "response_types", "access_token_ttl", "refresh_token_ttl",
			"require_consent", "created_at", "updated_at",
		}).AddRow(
			10, uuid.New(), 1, int64(100), "test-client", "Test Client",
			"spa", nil, "my-client", nil, "active",
			false, false, "none",
			pq.StringArray{GrantTypeDeviceCode}, pq.StringArray{ResponseTypeCode}, nil, nil,
			false, time.Now(), time.Now(),
		)
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT`)).WillReturnRows(rows)
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT`)).WillReturnRows(sqlmock.NewRows(nil))
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT`)).WillReturnRows(sqlmock.NewRows(nil))

		svc := newOAuthPARSvc(db, &mockClientURIRepo{}, &mockOAuthPARRepo{}, &mockAuthEventService{})

		_, oerr := svc.Push(ctx, OAuthPARRequestDTO{
			RedirectURI:  "https://example.com/callback",
			ResponseType: ResponseTypeCode,
		}, OAuthClientCredentials{ClientID: "my-client"})
		require.NotNil(t, oerr)
		assert.Equal(t, "unauthorized_client", oerr.Code)
	})

	t.Run("scope not allowed", func(t *testing.T) {
		db, mock := newMockDB(t)
		rows := sqlmock.NewRows([]string{
			"client_id", "client_uuid", "tenant_id", "identity_provider_id", "name", "display_name",
			"client_type", "domain", "identifier", "secret", "status",
			"is_default", "is_system", "token_endpoint_auth_method",
			"grant_types", "response_types", "access_token_ttl", "refresh_token_ttl",
			"require_consent", "allowed_scopes", "created_at", "updated_at",
		}).AddRow(
			10, uuid.New(), 1, int64(100), "test-client", "Test Client",
			"spa", nil, "my-client", nil, "active",
			false, false, "none",
			pq.StringArray{GrantTypeAuthorizationCode}, pq.StringArray{ResponseTypeCode}, nil, nil,
			false, pq.StringArray{"openid"}, time.Now(), time.Now(),
		)
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT`)).WillReturnRows(rows)
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT`)).WillReturnRows(sqlmock.NewRows(nil))
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT`)).WillReturnRows(sqlmock.NewRows(nil))

		svc := newOAuthPARSvc(db, &mockClientURIRepo{}, &mockOAuthPARRepo{}, &mockAuthEventService{})

		_, oerr := svc.Push(ctx, OAuthPARRequestDTO{
			RedirectURI:  "https://example.com/callback",
			ResponseType: ResponseTypeCode,
			Scope:        "admin",
		}, OAuthClientCredentials{ClientID: "my-client"})
		require.NotNil(t, oerr)
		assert.Equal(t, "invalid_scope", oerr.Code)
	})

	t.Run("with state and nonce", func(t *testing.T) {
		db, mock := newMockDB(t)
		mock.MatchExpectationsInOrder(false)
		rows := sqlmock.NewRows([]string{
			"client_id", "client_uuid", "tenant_id", "identity_provider_id", "name", "display_name",
			"client_type", "domain", "identifier", "secret", "status",
			"is_default", "is_system", "token_endpoint_auth_method",
			"grant_types", "response_types", "access_token_ttl", "refresh_token_ttl",
			"require_consent", "created_at", "updated_at",
		}).AddRow(
			10, uuid.New(), 1, int64(100), "test-client", "Test Client",
			"spa", nil, "my-client", nil, "active",
			false, false, "none",
			pq.StringArray{GrantTypeAuthorizationCode}, pq.StringArray{ResponseTypeCode}, nil, nil,
			false, time.Now(), time.Now(),
		)
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT`)).WillReturnRows(rows)
		mock.ExpectQuery(`FROM "identity_providers"`).WillReturnRows(sqlmock.NewRows(nil))
		mock.ExpectQuery(`FROM "client_uris"`).WillReturnRows(
			sqlmock.NewRows([]string{"client_uri_id", "client_uri_uuid", "tenant_id", "client_id", "uri", "type", "created_at", "updated_at"}).
				AddRow(1, uuid.New(), 1, 10, "https://example.com/callback", shared.ClientURITypeRedirect, time.Now(), time.Now()),
		)

		var capturedPAR *OAuthPARRequest
		svc := newOAuthPARSvc(db, &mockClientURIRepo{},
			&mockOAuthPARRepo{
				createFn: func(p *OAuthPARRequest) (*OAuthPARRequest, error) {
					capturedPAR = p
					return p, nil
				},
			},
			&mockAuthEventService{})

		result, oerr := svc.Push(ctx, OAuthPARRequestDTO{
			RedirectURI:         "https://example.com/callback",
			ResponseType:        ResponseTypeCode,
			CodeChallenge:       strings.Repeat("A", 43),
			CodeChallengeMethod: "S256",
			State:               "mystate",
			Nonce:               "mynonce",
		}, OAuthClientCredentials{ClientID: "my-client"})
		require.Nil(t, oerr)
		require.NotNil(t, result)
		assert.NotEmpty(t, result.RequestURI)
		require.NotNil(t, capturedPAR)
		require.NotNil(t, capturedPAR.State)
		assert.Equal(t, "mystate", *capturedPAR.State)
		require.NotNil(t, capturedPAR.Nonce)
		assert.Equal(t, "mynonce", *capturedPAR.Nonce)
	})

	t.Run("dangerous redirect URI", func(t *testing.T) {
		db, mock := newMockDB(t)
		mock.MatchExpectationsInOrder(false)
		rows := sqlmock.NewRows([]string{
			"client_id", "client_uuid", "tenant_id", "identity_provider_id", "name", "display_name",
			"client_type", "domain", "identifier", "secret", "status",
			"is_default", "is_system", "token_endpoint_auth_method",
			"grant_types", "response_types", "access_token_ttl", "refresh_token_ttl",
			"require_consent", "created_at", "updated_at",
		}).AddRow(
			10, uuid.New(), 1, int64(100), "test-client", "Test Client",
			"spa", nil, "my-client", nil, "active",
			false, false, "none",
			pq.StringArray{GrantTypeAuthorizationCode}, pq.StringArray{ResponseTypeCode}, nil, nil,
			false, time.Now(), time.Now(),
		)
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT`)).WillReturnRows(rows)
		mock.ExpectQuery(`FROM "identity_providers"`).WillReturnRows(sqlmock.NewRows(nil))
		mock.ExpectQuery(`FROM "client_uris"`).WillReturnRows(
			sqlmock.NewRows([]string{"client_uri_id", "client_uri_uuid", "tenant_id", "client_id", "uri", "type", "created_at", "updated_at"}).
				AddRow(1, uuid.New(), 1, 10, "javascript:alert(1)", shared.ClientURITypeRedirect, time.Now(), time.Now()),
		)

		svc := newOAuthPARSvc(db, &mockClientURIRepo{}, &mockOAuthPARRepo{}, &mockAuthEventService{})

		_, oerr := svc.Push(ctx, OAuthPARRequestDTO{
			RedirectURI:         "javascript:alert(1)",
			ResponseType:        ResponseTypeCode,
			CodeChallenge:       strings.Repeat("A", 43),
			CodeChallengeMethod: "S256",
		}, OAuthClientCredentials{ClientID: "my-client"})
		require.NotNil(t, oerr)
		assert.Equal(t, "invalid_request", oerr.Code)
		assert.Contains(t, oerr.Description, "forbidden scheme")
	})

	t.Run("GenerateRandomString error", func(t *testing.T) {
		db, mock := newMockDB(t)
		mock.MatchExpectationsInOrder(false)
		rows := sqlmock.NewRows([]string{
			"client_id", "client_uuid", "tenant_id", "identity_provider_id", "name", "display_name",
			"client_type", "domain", "identifier", "secret", "status",
			"is_default", "is_system", "token_endpoint_auth_method",
			"grant_types", "response_types", "access_token_ttl", "refresh_token_ttl",
			"require_consent", "created_at", "updated_at",
		}).AddRow(
			10, uuid.New(), 1, int64(100), "test-client", "Test Client",
			"spa", nil, "my-client", nil, "active",
			false, false, "none",
			pq.StringArray{GrantTypeAuthorizationCode}, pq.StringArray{ResponseTypeCode}, nil, nil,
			false, time.Now(), time.Now(),
		)
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT`)).WillReturnRows(rows)
		mock.ExpectQuery(`FROM "identity_providers"`).WillReturnRows(sqlmock.NewRows(nil))
		mock.ExpectQuery(`FROM "client_uris"`).WillReturnRows(
			sqlmock.NewRows([]string{"client_uri_id", "client_uri_uuid", "tenant_id", "client_id", "uri", "type", "created_at", "updated_at"}).
				AddRow(1, uuid.New(), 1, 10, "https://example.com/callback", shared.ClientURITypeRedirect, time.Now(), time.Now()),
		)

		orig := crypto.GenerateRandomString
		defer func() { crypto.GenerateRandomString = orig }()
		crypto.GenerateRandomString = func(int) (string, error) { return "", errors.New("rand failure") }

		svc := newOAuthPARSvc(db, &mockClientURIRepo{}, &mockOAuthPARRepo{}, &mockAuthEventService{})

		_, oerr := svc.Push(ctx, OAuthPARRequestDTO{
			RedirectURI:         "https://example.com/callback",
			ResponseType:        ResponseTypeCode,
			CodeChallenge:       strings.Repeat("A", 43),
			CodeChallengeMethod: "S256",
		}, OAuthClientCredentials{ClientID: "my-client"})
		require.NotNil(t, oerr)
		assert.Equal(t, "server_error", oerr.Code)
	})
}

// ── TestOAuthPARService_ConsumeRequestURI ────────────────────────────────────

func TestOAuthPARService_ConsumeRequestURI(t *testing.T) {
	ctx := context.Background()

	t.Run("not found", func(t *testing.T) {
		db, _ := newMockDB(t)
		svc := &oauthPARService{
			db: db,
			parRepo: &mockOAuthPARRepo{
				findByRequestURIHashFn: func(_ string) (*OAuthPARRequest, error) {
					return nil, nil
				},
			},
			authEventService: &mockAuthEventService{},
		}

		_, oerr := svc.ConsumeRequestURI(ctx, "urn:ietf:params:oauth:request-uri:abc")
		require.NotNil(t, oerr)
		assert.Equal(t, "invalid_request", oerr.Code)
	})

	t.Run("expired", func(t *testing.T) {
		db, _ := newMockDB(t)
		svc := &oauthPARService{
			db: db,
			parRepo: &mockOAuthPARRepo{
				findByRequestURIHashFn: func(_ string) (*OAuthPARRequest, error) {
					return &OAuthPARRequest{
						OAuthPARRequestID: 1,
						ExpiresAt:         time.Now().Add(-1 * time.Minute),
					}, nil
				},
			},
			authEventService: &mockAuthEventService{},
		}

		_, oerr := svc.ConsumeRequestURI(ctx, "urn:ietf:params:oauth:request-uri:abc")
		require.NotNil(t, oerr)
		assert.Equal(t, "invalid_request", oerr.Code)
		assert.Contains(t, oerr.Description, "expired")
	})

	t.Run("success", func(t *testing.T) {
		db, _ := newMockDB(t)
		stateVal := "mystate"
		nonceVal := "mynonce"

		svc := &oauthPARService{
			db: db,
			parRepo: &mockOAuthPARRepo{
				findByRequestURIHashFn: func(_ string) (*OAuthPARRequest, error) {
					return &OAuthPARRequest{
						OAuthPARRequestID:   1,
						ResponseType:        ResponseTypeCode,
						RedirectURI:         "https://example.com/callback",
						Scope:               "openid",
						State:               &stateVal,
						Nonce:               &nonceVal,
						CodeChallenge:       "challenge",
						CodeChallengeMethod: "S256",
						ExpiresAt:           time.Now().Add(30 * time.Second),
						Client: &Client{
							Identifier: ptr.Ptr("my-client"),
						},
					}, nil
				},
				markUsedFn: func(_ int64) error { return nil },
			},
			authEventService: &mockAuthEventService{},
		}

		result, oerr := svc.ConsumeRequestURI(ctx, "urn:ietf:params:oauth:request-uri:abc")
		require.Nil(t, oerr)
		require.NotNil(t, result)
		assert.Equal(t, ResponseTypeCode, result.ResponseType)
		assert.Equal(t, "my-client", result.ClientID)
		assert.Equal(t, "https://example.com/callback", result.RedirectURI)
		assert.Equal(t, "openid", result.Scope)
		assert.Equal(t, "mystate", result.State)
		assert.Equal(t, "mynonce", result.Nonce)
	})
}

// ── TestOAuthPARService_ConsumeRequestURI_ParseError ─────────────────────────

func TestOAuthPARService_ConsumeRequestURI_ParseError(t *testing.T) {
	ctx := context.Background()
	db, _ := newMockDB(t)
	svc := &oauthPARService{
		db:               db,
		parRepo:          &mockOAuthPARRepo{},
		authEventService: &mockAuthEventService{},
	}

	t.Run("no prefix", func(t *testing.T) {
		_, oerr := svc.ConsumeRequestURI(ctx, "not-a-valid-uri")
		require.NotNil(t, oerr)
		assert.Equal(t, "invalid_request", oerr.Code)
		assert.Contains(t, oerr.Description, "not a valid PAR URI")
	})
}

func TestOAuthPARService_ConsumeRequestURI_RepoError(t *testing.T) {
	ctx := context.Background()
	db, _ := newMockDB(t)
	svc := &oauthPARService{
		db: db,
		parRepo: &mockOAuthPARRepo{
			findByRequestURIHashFn: func(_ string) (*OAuthPARRequest, error) {
				return nil, errors.New("db error")
			},
		},
		authEventService: &mockAuthEventService{},
	}

	t.Run("lookup error", func(t *testing.T) {
		_, oerr := svc.ConsumeRequestURI(ctx, "urn:ietf:params:oauth:request-uri:abc")
		require.NotNil(t, oerr)
		assert.Equal(t, "server_error", oerr.Code)
	})
}

func TestOAuthPARService_ConsumeRequestURI_MarkUsedError(t *testing.T) {
	ctx := context.Background()
	db, _ := newMockDB(t)
	svc := &oauthPARService{
		db: db,
		parRepo: &mockOAuthPARRepo{
			findByRequestURIHashFn: func(_ string) (*OAuthPARRequest, error) {
				return &OAuthPARRequest{
					OAuthPARRequestID: 1,
					ExpiresAt:         time.Now().Add(30 * time.Second),
				}, nil
			},
			markUsedFn: func(_ int64) error {
				return errors.New("db error")
			},
		},
		authEventService: &mockAuthEventService{},
	}

	t.Run("markUsed error", func(t *testing.T) {
		_, oerr := svc.ConsumeRequestURI(ctx, "urn:ietf:params:oauth:request-uri:abc")
		require.NotNil(t, oerr)
		assert.Equal(t, "server_error", oerr.Code)
	})
}

func TestResolveClientIdentifier(t *testing.T) {
	assert.Equal(t, "", resolveClientIdentifier(nil))

	identifier := "test-client"
	assert.Equal(t, "test-client", resolveClientIdentifier(&Client{Identifier: &identifier}))
	assert.Equal(t, "", resolveClientIdentifier(&Client{}))
}

func TestValidateClientRedirectURI(t *testing.T) {
	redirectURI := "https://example.com/callback"

	t.Run("nil ClientURIs", func(t *testing.T) {
		oerr := validateClientRedirectURI(&Client{}, redirectURI)
		require.NotNil(t, oerr)
		assert.Contains(t, oerr.Description, "no redirect URIs")
	})

	t.Run("matches registered URI", func(t *testing.T) {
		client := &Client{
			ClientURIs: &[]ClientURI{
				{URI: redirectURI, Type: shared.ClientURITypeRedirect},
			},
		}
		assert.Nil(t, validateClientRedirectURI(client, redirectURI))
	})

	t.Run("no match", func(t *testing.T) {
		client := &Client{
			ClientURIs: &[]ClientURI{
				{URI: "https://other.com/cb", Type: shared.ClientURITypeRedirect},
			},
		}
		oerr := validateClientRedirectURI(client, redirectURI)
		require.NotNil(t, oerr)
		assert.Contains(t, oerr.Description, "does not match")
	})
}
