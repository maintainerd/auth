package oauth

import (
	"context"
	"errors"
	"regexp"
	"testing"
	"time"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/maintainerd/auth/internal/platform/config"
	"github.com/maintainerd/auth/internal/platform/crypto"
	"github.com/maintainerd/auth/internal/platform/jwt"
	"github.com/maintainerd/auth/internal/platform/ptr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func newOAuthCIBASvc(
	db *gorm.DB,
	cibaRepo *mockOAuthCIBARepo,
	userRepo *mockUserRepo,
	authEventSvc *mockAuthEventService,
) OAuthCIBAService {
	return &oauthCIBAService{
		db:               db,
		clientRepo:       &mockClientRepo{},
		cibaRepo:         cibaRepo,
		userRepo:         userRepo,
		authEventService: authEventSvc,
	}
}

type mockOAuthCIBARepo struct {
	findCIBAReqByHashFn func(string) (*OAuthCIBARequest, error)
	createFn            func(*OAuthCIBARequest) (*OAuthCIBARequest, error)
	updateStatusFn      func(int64, string) error
	updateApprovalFn    func(int64, int64) error
	updateApprovalCtxFn func(int64, int64, string, []string) error
	updateLastPollAtFn  func(int64) error
}

func (m *mockOAuthCIBARepo) WithTx(_ *gorm.DB) OAuthCIBARequestRepository { return m }
func (m *mockOAuthCIBARepo) FindByAuthReqIDHash(hash string) (*OAuthCIBARequest, error) {
	if m.findCIBAReqByHashFn != nil {
		return m.findCIBAReqByHashFn(hash)
	}
	return nil, nil
}
func (m *mockOAuthCIBARepo) UpdateStatus(id int64, status string) error {
	if m.updateStatusFn != nil {
		return m.updateStatusFn(id, status)
	}
	return nil
}
func (m *mockOAuthCIBARepo) UpdateApproval(id int64, userID int64) error {
	if m.updateApprovalFn != nil {
		return m.updateApprovalFn(id, userID)
	}
	return nil
}
func (m *mockOAuthCIBARepo) UpdateApprovalContext(id int64, userID int64, acr string, amr []string) error {
	if m.updateApprovalCtxFn != nil {
		return m.updateApprovalCtxFn(id, userID, acr, amr)
	}
	return m.UpdateApproval(id, userID)
}
func (m *mockOAuthCIBARepo) UpdateLastPollAt(id int64) error {
	if m.updateLastPollAtFn != nil {
		return m.updateLastPollAtFn(id)
	}
	return nil
}
func (m *mockOAuthCIBARepo) MarkNotificationSent(_ int64) error       { return nil }
func (m *mockOAuthCIBARepo) DeleteExpired(_ time.Time) (int64, error) { return 0, nil }
func (m *mockOAuthCIBARepo) Create(e *OAuthCIBARequest) (*OAuthCIBARequest, error) {
	if m.createFn != nil {
		return m.createFn(e)
	}
	return e, nil
}
func (m *mockOAuthCIBARepo) CreateOrUpdate(e *OAuthCIBARequest) (*OAuthCIBARequest, error) {
	return e, nil
}
func (m *mockOAuthCIBARepo) FindAll(_ ...string) ([]OAuthCIBARequest, error) { return nil, nil }
func (m *mockOAuthCIBARepo) FindByUUID(_ any, _ ...string) (*OAuthCIBARequest, error) {
	return nil, nil
}
func (m *mockOAuthCIBARepo) FindByUUIDs(_ []string, _ ...string) ([]OAuthCIBARequest, error) {
	return nil, nil
}
func (m *mockOAuthCIBARepo) FindByID(_ any, _ ...string) (*OAuthCIBARequest, error) { return nil, nil }
func (m *mockOAuthCIBARepo) UpdateByUUID(_, _ any) (*OAuthCIBARequest, error)       { return nil, nil }
func (m *mockOAuthCIBARepo) UpdateByID(_, _ any) (*OAuthCIBARequest, error)         { return nil, nil }
func (m *mockOAuthCIBARepo) DeleteByUUID(_ any) error                               { return nil }
func (m *mockOAuthCIBARepo) DeleteByID(_ any) error                                 { return nil }
func (m *mockOAuthCIBARepo) Paginate(_ map[string]any, _, _ int, _ ...string) (*PaginationResult[OAuthCIBARequest], error) {
	return nil, nil
}

func expectCIBAClientLookup(mock sqlmock.Sqlmock) {
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
		pq.StringArray{GrantTypeCIBA}, pq.StringArray{ResponseTypeCode}, nil, nil,
		false, time.Now(), time.Now(),
	)
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT`)).WillReturnRows(rows)
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT`)).WillReturnRows(sqlmock.NewRows(nil))
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT`)).WillReturnRows(sqlmock.NewRows(nil))
}

func expectCIBAClientNotFound(mock sqlmock.Sqlmock) {
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT`)).WillReturnError(gorm.ErrRecordNotFound)
}

// ── TestOAuthCIBAService_Initiate ───────────────────────────────────────────

func TestOAuthCIBAService_Initiate(t *testing.T) {
	ctx := context.Background()

	t.Run("client not found", func(t *testing.T) {
		db, mock := newMockDB(t)
		expectCIBAClientNotFound(mock)

		svc := newOAuthCIBASvc(db, &mockOAuthCIBARepo{}, &mockUserRepo{}, &mockAuthEventService{})

		_, oerr := svc.Initiate(ctx, OAuthCIBARequestDTO{LoginHint: "a@b.com"}, OAuthClientCredentials{ClientID: "unknown"})
		require.NotNil(t, oerr)
		assert.Equal(t, "invalid_client", oerr.Code)
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
			pq.StringArray{GrantTypeAuthorizationCode}, pq.StringArray{ResponseTypeCode}, nil, nil,
			false, time.Now(), time.Now(),
		)
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT`)).WillReturnRows(rows)
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT`)).WillReturnRows(sqlmock.NewRows(nil))
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT`)).WillReturnRows(sqlmock.NewRows(nil))

		svc := newOAuthCIBASvc(db, &mockOAuthCIBARepo{}, &mockUserRepo{}, &mockAuthEventService{})

		_, oerr := svc.Initiate(ctx, OAuthCIBARequestDTO{LoginHint: "a@b.com"}, OAuthClientCredentials{ClientID: "my-client"})
		require.NotNil(t, oerr)
		assert.Equal(t, "unauthorized_client", oerr.Code)
	})

	t.Run("empty login_hint", func(t *testing.T) {
		db, mock := newMockDB(t)
		expectCIBAClientLookup(mock)

		svc := newOAuthCIBASvc(db, &mockOAuthCIBARepo{}, &mockUserRepo{}, &mockAuthEventService{})

		_, oerr := svc.Initiate(ctx, OAuthCIBARequestDTO{}, OAuthClientCredentials{ClientID: "my-client"})
		require.NotNil(t, oerr)
		assert.Equal(t, "invalid_request", oerr.Code)
		assert.Contains(t, oerr.Description, "login_hint")
	})

	t.Run("user repo error on login_hint lookup", func(t *testing.T) {
		db, mock := newMockDB(t)
		expectCIBAClientLookup(mock)

		svc := newOAuthCIBASvc(db, &mockOAuthCIBARepo{},
			&mockUserRepo{
				findByEmailFn: func(_ string) (*User, error) {
					return nil, errors.New("db error")
				},
			},
			&mockAuthEventService{})

		_, oerr := svc.Initiate(ctx, OAuthCIBARequestDTO{LoginHint: "a@b.com"}, OAuthClientCredentials{ClientID: "my-client"})
		require.NotNil(t, oerr)
		assert.Equal(t, "server_error", oerr.Code)
	})

	t.Run("user not found by login_hint", func(t *testing.T) {
		db, mock := newMockDB(t)
		expectCIBAClientLookup(mock)

		svc := newOAuthCIBASvc(db, &mockOAuthCIBARepo{},
			&mockUserRepo{
				findByEmailFn: func(_ string) (*User, error) {
					return nil, nil
				},
			},
			&mockAuthEventService{})

		_, oerr := svc.Initiate(ctx, OAuthCIBARequestDTO{LoginHint: "a@b.com"}, OAuthClientCredentials{ClientID: "my-client"})
		require.NotNil(t, oerr)
		assert.Equal(t, "invalid_request", oerr.Code)
		assert.Contains(t, oerr.Description, "no user found")
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
			pq.StringArray{GrantTypeCIBA}, pq.StringArray{ResponseTypeCode}, nil, nil,
			false, pq.StringArray{"openid"}, time.Now(), time.Now(),
		)
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT`)).WillReturnRows(rows)
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT`)).WillReturnRows(sqlmock.NewRows(nil))
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT`)).WillReturnRows(sqlmock.NewRows(nil))

		svc := newOAuthCIBASvc(db, &mockOAuthCIBARepo{},
			&mockUserRepo{
				findByEmailFn: func(_ string) (*User, error) {
					return &User{UserID: 1, UserUUID: uuid.New(), Email: "a@b.com"}, nil
				},
			},
			&mockAuthEventService{})

		_, oerr := svc.Initiate(ctx, OAuthCIBARequestDTO{LoginHint: "a@b.com", Scope: "profile admin"}, OAuthClientCredentials{ClientID: "my-client"})
		require.NotNil(t, oerr)
		assert.Equal(t, "invalid_scope", oerr.Code)
	})

	t.Run("repo create error", func(t *testing.T) {
		db, mock := newMockDB(t)
		expectCIBAClientLookup(mock)

		svc := newOAuthCIBASvc(db,
			&mockOAuthCIBARepo{
				createFn: func(_ *OAuthCIBARequest) (*OAuthCIBARequest, error) {
					return nil, errors.New("db error")
				},
			},
			&mockUserRepo{
				findByEmailFn: func(_ string) (*User, error) {
					return &User{UserID: 1, UserUUID: uuid.New(), Email: "a@b.com"}, nil
				},
			},
			&mockAuthEventService{})

		_, oerr := svc.Initiate(ctx, OAuthCIBARequestDTO{LoginHint: "a@b.com"}, OAuthClientCredentials{ClientID: "my-client"})
		require.NotNil(t, oerr)
		assert.Equal(t, "server_error", oerr.Code)
	})

	t.Run("success", func(t *testing.T) {
		db, mock := newMockDB(t)
		expectCIBAClientLookup(mock)

		svc := newOAuthCIBASvc(db, &mockOAuthCIBARepo{},
			&mockUserRepo{
				findByEmailFn: func(_ string) (*User, error) {
					return &User{UserID: 1, UserUUID: uuid.New(), Email: "a@b.com"}, nil
				},
			},
			&mockAuthEventService{})

		result, oerr := svc.Initiate(ctx, OAuthCIBARequestDTO{LoginHint: "a@b.com", Scope: "openid"}, OAuthClientCredentials{ClientID: "my-client"})
		require.Nil(t, oerr)
		require.NotNil(t, result)
		assert.NotEmpty(t, result.AuthReqID)
		assert.Equal(t, 300, result.ExpiresIn)
		assert.Equal(t, 5, result.Interval)
	})

	t.Run("with binding_message", func(t *testing.T) {
		db, mock := newMockDB(t)
		expectCIBAClientLookup(mock)

		var capturedBinding *string
		svc := newOAuthCIBASvc(db,
			&mockOAuthCIBARepo{
				createFn: func(r *OAuthCIBARequest) (*OAuthCIBARequest, error) {
					capturedBinding = r.BindingMessage
					return r, nil
				},
			},
			&mockUserRepo{
				findByEmailFn: func(_ string) (*User, error) {
					return &User{UserID: 1, UserUUID: uuid.New(), Email: "a@b.com"}, nil
				},
			},
			&mockAuthEventService{})

		result, oerr := svc.Initiate(ctx, OAuthCIBARequestDTO{
			LoginHint:      "a@b.com",
			Scope:          "openid",
			BindingMessage: "confirm transaction 12345",
		}, OAuthClientCredentials{ClientID: "my-client"})
		require.Nil(t, oerr)
		require.NotNil(t, result)
		assert.NotEmpty(t, result.AuthReqID)
		require.NotNil(t, capturedBinding)
		assert.Equal(t, "confirm transaction 12345", *capturedBinding)
	})

	t.Run("empty email skips notification", func(t *testing.T) {
		db, mock := newMockDB(t)
		expectCIBAClientLookup(mock)

		svc := newOAuthCIBASvc(db, &mockOAuthCIBARepo{},
			&mockUserRepo{
				findByEmailFn: func(_ string) (*User, error) {
					return &User{UserID: 1, UserUUID: uuid.New(), Email: ""}, nil
				},
			},
			&mockAuthEventService{})

		result, oerr := svc.Initiate(ctx, OAuthCIBARequestDTO{LoginHint: "user@noemail.com", Scope: "openid"}, OAuthClientCredentials{ClientID: "my-client"})
		require.Nil(t, oerr)
		require.NotNil(t, result)
		assert.NotEmpty(t, result.AuthReqID)
	})

	t.Run("GenerateRandomString error", func(t *testing.T) {
		db, mock := newMockDB(t)
		expectCIBAClientLookup(mock)

		orig := crypto.GenerateRandomString
		defer func() { crypto.GenerateRandomString = orig }()
		crypto.GenerateRandomString = func(int) (string, error) { return "", errors.New("rand failure") }

		svc := newOAuthCIBASvc(db, &mockOAuthCIBARepo{},
			&mockUserRepo{
				findByEmailFn: func(_ string) (*User, error) {
					return &User{UserID: 1, UserUUID: uuid.New(), Email: "a@b.com"}, nil
				},
			},
			&mockAuthEventService{})

		_, oerr := svc.Initiate(ctx, OAuthCIBARequestDTO{LoginHint: "a@b.com", Scope: "openid"}, OAuthClientCredentials{ClientID: "my-client"})
		require.NotNil(t, oerr)
		assert.Equal(t, "server_error", oerr.Code)
	})
}

// ── TestOAuthCIBAService_ExchangeToken ──────────────────────────────────────

func TestOAuthCIBAService_ExchangeToken(t *testing.T) {
	ctx := context.Background()

	t.Run("client auth error", func(t *testing.T) {
		db, mock := newMockDB(t)
		expectCIBAClientNotFound(mock)

		svc := newOAuthCIBASvc(db, &mockOAuthCIBARepo{}, &mockUserRepo{}, &mockAuthEventService{})

		_, oerr := svc.ExchangeToken(ctx, OAuthCIBATokenRequestDTO{AuthReqID: "xxxx"}, OAuthClientCredentials{ClientID: "unknown"})
		require.NotNil(t, oerr)
		assert.Equal(t, "invalid_client", oerr.Code)
	})

	t.Run("auth_req_id not found", func(t *testing.T) {
		db, mock := newMockDB(t)
		expectCIBAClientLookup(mock)

		svc := newOAuthCIBASvc(db,
			&mockOAuthCIBARepo{
				findCIBAReqByHashFn: func(_ string) (*OAuthCIBARequest, error) {
					return nil, nil
				},
			},
			&mockUserRepo{}, &mockAuthEventService{})

		_, oerr := svc.ExchangeToken(ctx, OAuthCIBATokenRequestDTO{AuthReqID: "xxxx"}, OAuthClientCredentials{ClientID: "my-client"})
		require.NotNil(t, oerr)
		assert.Equal(t, "invalid_grant", oerr.Code)
	})

	t.Run("repo error", func(t *testing.T) {
		db, mock := newMockDB(t)
		expectCIBAClientLookup(mock)

		svc := newOAuthCIBASvc(db,
			&mockOAuthCIBARepo{
				findCIBAReqByHashFn: func(_ string) (*OAuthCIBARequest, error) {
					return nil, errors.New("db error")
				},
			},
			&mockUserRepo{}, &mockAuthEventService{})

		_, oerr := svc.ExchangeToken(ctx, OAuthCIBATokenRequestDTO{AuthReqID: "xxxx"}, OAuthClientCredentials{ClientID: "my-client"})
		require.NotNil(t, oerr)
		assert.Equal(t, "server_error", oerr.Code)
	})

	t.Run("client mismatch", func(t *testing.T) {
		db, mock := newMockDB(t)
		expectCIBAClientLookup(mock)

		svc := newOAuthCIBASvc(db,
			&mockOAuthCIBARepo{
				findCIBAReqByHashFn: func(_ string) (*OAuthCIBARequest, error) {
					return &OAuthCIBARequest{
						OAuthCIBARRequestID: 1,
						ClientID:            999,
						Status:              CIBAStatusPending,
						Interval:            5,
						ExpiresAt:           time.Now().Add(5 * time.Minute),
					}, nil
				},
			},
			&mockUserRepo{}, &mockAuthEventService{})

		_, oerr := svc.ExchangeToken(ctx, OAuthCIBATokenRequestDTO{AuthReqID: "xxxx"}, OAuthClientCredentials{ClientID: "my-client"})
		require.NotNil(t, oerr)
		assert.Equal(t, "invalid_grant", oerr.Code)
		assert.Contains(t, oerr.Description, "does not belong")
	})

	t.Run("slow_down", func(t *testing.T) {
		db, mock := newMockDB(t)
		expectCIBAClientLookup(mock)
		lastPoll := time.Now().Add(-1 * time.Second)

		svc := newOAuthCIBASvc(db,
			&mockOAuthCIBARepo{
				findCIBAReqByHashFn: func(_ string) (*OAuthCIBARequest, error) {
					return &OAuthCIBARequest{
						OAuthCIBARRequestID: 1,
						ClientID:            10,
						Status:              CIBAStatusPending,
						Interval:            5,
						LastPollAt:          &lastPoll,
						ExpiresAt:           time.Now().Add(5 * time.Minute),
					}, nil
				},
			},
			&mockUserRepo{}, &mockAuthEventService{})

		_, oerr := svc.ExchangeToken(ctx, OAuthCIBATokenRequestDTO{AuthReqID: "xxxx"}, OAuthClientCredentials{ClientID: "my-client"})
		require.NotNil(t, oerr)
		assert.Equal(t, "slow_down", oerr.Code)
	})

	t.Run("denied", func(t *testing.T) {
		db, mock := newMockDB(t)
		expectCIBAClientLookup(mock)

		svc := newOAuthCIBASvc(db,
			&mockOAuthCIBARepo{
				findCIBAReqByHashFn: func(_ string) (*OAuthCIBARequest, error) {
					return &OAuthCIBARequest{
						OAuthCIBARRequestID: 1,
						ClientID:            10,
						Status:              CIBAStatusDenied,
						Interval:            5,
						ExpiresAt:           time.Now().Add(5 * time.Minute),
					}, nil
				},
			},
			&mockUserRepo{}, &mockAuthEventService{})

		_, oerr := svc.ExchangeToken(ctx, OAuthCIBATokenRequestDTO{AuthReqID: "xxxx"}, OAuthClientCredentials{ClientID: "my-client"})
		require.NotNil(t, oerr)
		assert.Equal(t, "access_denied", oerr.Code)
	})

	t.Run("expired status", func(t *testing.T) {
		db, mock := newMockDB(t)
		expectCIBAClientLookup(mock)

		svc := newOAuthCIBASvc(db,
			&mockOAuthCIBARepo{
				findCIBAReqByHashFn: func(_ string) (*OAuthCIBARequest, error) {
					return &OAuthCIBARequest{
						ClientID:  10,
						Status:    CIBAStatusExpired,
						ExpiresAt: time.Now().Add(-1 * time.Minute),
					}, nil
				},
			},
			&mockUserRepo{}, &mockAuthEventService{})

		_, oerr := svc.ExchangeToken(ctx, OAuthCIBATokenRequestDTO{AuthReqID: "xxxx"}, OAuthClientCredentials{ClientID: "my-client"})
		require.NotNil(t, oerr)
		assert.Equal(t, "expired_token", oerr.Code)
	})

	t.Run("approved with nil userID", func(t *testing.T) {
		db, mock := newMockDB(t)
		expectCIBAClientLookup(mock)

		svc := newOAuthCIBASvc(db,
			&mockOAuthCIBARepo{
				findCIBAReqByHashFn: func(_ string) (*OAuthCIBARequest, error) {
					return &OAuthCIBARequest{
						OAuthCIBARRequestID: 1,
						ClientID:            10,
						UserID:              nil,
						Status:              CIBAStatusApproved,
						Interval:            5,
						ExpiresAt:           time.Now().Add(5 * time.Minute),
					}, nil
				},
			},
			&mockUserRepo{}, &mockAuthEventService{})

		_, oerr := svc.ExchangeToken(ctx, OAuthCIBATokenRequestDTO{AuthReqID: "xxxx"}, OAuthClientCredentials{ClientID: "my-client"})
		require.NotNil(t, oerr)
		assert.Equal(t, "server_error", oerr.Code)
	})

	t.Run("user not found after approval", func(t *testing.T) {
		db, mock := newMockDB(t)
		expectCIBAClientLookup(mock)
		userID := int64(1)

		svc := newOAuthCIBASvc(db,
			&mockOAuthCIBARepo{
				findCIBAReqByHashFn: func(_ string) (*OAuthCIBARequest, error) {
					return &OAuthCIBARequest{
						OAuthCIBARRequestID: 1,
						ClientID:            10,
						TenantID:            1,
						UserID:              &userID,
						Scope:               "openid",
						Status:              CIBAStatusApproved,
						Interval:            5,
						ExpiresAt:           time.Now().Add(5 * time.Minute),
					}, nil
				},
			},
			&mockUserRepo{
				findByIDFn: func(_ any, _ ...string) (*User, error) {
					return nil, nil
				},
			},
			&mockAuthEventService{})

		_, oerr := svc.ExchangeToken(ctx, OAuthCIBATokenRequestDTO{AuthReqID: "xxxx"}, OAuthClientCredentials{ClientID: "my-client"})
		require.NotNil(t, oerr)
		assert.Equal(t, "server_error", oerr.Code)
	})

	t.Run("expired", func(t *testing.T) {
		db, mock := newMockDB(t)
		expectCIBAClientLookup(mock)

		svc := newOAuthCIBASvc(db,
			&mockOAuthCIBARepo{
				findCIBAReqByHashFn: func(_ string) (*OAuthCIBARequest, error) {
					return &OAuthCIBARequest{
						ClientID:  10,
						Status:    CIBAStatusPending,
						ExpiresAt: time.Now().Add(-1 * time.Minute),
					}, nil
				},
			},
			&mockUserRepo{}, &mockAuthEventService{})

		_, oerr := svc.ExchangeToken(ctx, OAuthCIBATokenRequestDTO{AuthReqID: "xxxx"}, OAuthClientCredentials{ClientID: "my-client"})
		require.NotNil(t, oerr)
		assert.Equal(t, "expired_token", oerr.Code)
	})

	t.Run("authorization_pending", func(t *testing.T) {
		db, mock := newMockDB(t)
		expectCIBAClientLookup(mock)

		svc := newOAuthCIBASvc(db,
			&mockOAuthCIBARepo{
				findCIBAReqByHashFn: func(_ string) (*OAuthCIBARequest, error) {
					return &OAuthCIBARequest{
						OAuthCIBARRequestID: 1,
						ClientID:            10,
						Status:              CIBAStatusPending,
						Interval:            5,
						ExpiresAt:           time.Now().Add(5 * time.Minute),
					}, nil
				},
			},
			&mockUserRepo{}, &mockAuthEventService{})

		_, oerr := svc.ExchangeToken(ctx, OAuthCIBATokenRequestDTO{AuthReqID: "xxxx"}, OAuthClientCredentials{ClientID: "my-client"})
		require.NotNil(t, oerr)
		assert.Equal(t, "authorization_pending", oerr.Code)
	})

	t.Run("denied", func(t *testing.T) {
		db, mock := newMockDB(t)
		expectCIBAClientLookup(mock)

		svc := newOAuthCIBASvc(db,
			&mockOAuthCIBARepo{
				findCIBAReqByHashFn: func(_ string) (*OAuthCIBARequest, error) {
					return &OAuthCIBARequest{
						OAuthCIBARRequestID: 1,
						ClientID:            10,
						Status:              CIBAStatusDenied,
						Interval:            5,
						ExpiresAt:           time.Now().Add(5 * time.Minute),
					}, nil
				},
			},
			&mockUserRepo{}, &mockAuthEventService{})

		_, oerr := svc.ExchangeToken(ctx, OAuthCIBATokenRequestDTO{AuthReqID: "xxxx"}, OAuthClientCredentials{ClientID: "my-client"})
		require.NotNil(t, oerr)
		assert.Equal(t, "access_denied", oerr.Code)
	})

	t.Run("unexpected status", func(t *testing.T) {
		db, mock := newMockDB(t)
		expectCIBAClientLookup(mock)

		svc := newOAuthCIBASvc(db,
			&mockOAuthCIBARepo{
				findCIBAReqByHashFn: func(_ string) (*OAuthCIBARequest, error) {
					return &OAuthCIBARequest{
						OAuthCIBARRequestID: 1,
						ClientID:            10,
						Status:              "unknown_status",
						Interval:            5,
						ExpiresAt:           time.Now().Add(5 * time.Minute),
					}, nil
				},
			},
			&mockUserRepo{}, &mockAuthEventService{})

		_, oerr := svc.ExchangeToken(ctx, OAuthCIBATokenRequestDTO{AuthReqID: "xxxx"}, OAuthClientCredentials{ClientID: "my-client"})
		require.NotNil(t, oerr)
		assert.Equal(t, "invalid_grant", oerr.Code)
		assert.Contains(t, oerr.Description, "unexpected CIBA")
	})

	t.Run("success", func(t *testing.T) {
		initTestJWTKeysService(t)
		origHost := config.AppPublicHostname
		config.AppPublicHostname = "https://auth.example.com"
		defer func() { config.AppPublicHostname = origHost }()

		db, mock := newMockDB(t)
		expectCIBAClientLookup(mock)
		userID := int64(1)

		svc := newOAuthCIBASvc(db,
			&mockOAuthCIBARepo{
				findCIBAReqByHashFn: func(_ string) (*OAuthCIBARequest, error) {
					return &OAuthCIBARequest{
						OAuthCIBARRequestID: 1,
						ClientID:            10,
						TenantID:            1,
						UserID:              &userID,
						Scope:               "openid",
						Status:              CIBAStatusApproved,
						Interval:            5,
						ExpiresAt:           time.Now().Add(5 * time.Minute),
						Client: &Client{
							ClientUUID: uuid.New(),
							Identifier: ptr.Ptr("my-client"),
							IdentityProvider: &IdentityProvider{
								IdentityProviderUUID: uuid.New(),
							},
						},
					}, nil
				},
			},
			&mockUserRepo{
				findByIDFn: func(_ any, _ ...string) (*User, error) {
					return &User{UserID: 1, UserUUID: uuid.New(), Email: "a@b.com"}, nil
				},
			},
			&mockAuthEventService{})

		result, oerr := svc.ExchangeToken(ctx, OAuthCIBATokenRequestDTO{AuthReqID: "xxxx"}, OAuthClientCredentials{ClientID: "my-client"})
		require.Nil(t, oerr)
		require.NotNil(t, result)
		assert.NotEmpty(t, result.AccessToken)
		assert.Equal(t, "Bearer", result.TokenType)
		assert.Equal(t, "openid", result.Scope)
	})

	t.Run("JWT generation error", func(t *testing.T) {
		jwt.ResetJWTKeys()

		db, mock := newMockDB(t)
		expectCIBAClientLookup(mock)
		userID := int64(1)

		svc := newOAuthCIBASvc(db,
			&mockOAuthCIBARepo{
				findCIBAReqByHashFn: func(_ string) (*OAuthCIBARequest, error) {
					return &OAuthCIBARequest{
						OAuthCIBARRequestID: 1,
						ClientID:            10,
						TenantID:            1,
						UserID:              &userID,
						Scope:               "openid",
						Status:              CIBAStatusApproved,
						Interval:            5,
						ExpiresAt:           time.Now().Add(5 * time.Minute),
						Client: &Client{
							ClientUUID: uuid.New(),
							Identifier: ptr.Ptr("my-client"),
							IdentityProvider: &IdentityProvider{
								IdentityProviderUUID: uuid.New(),
							},
						},
					}, nil
				},
			},
			&mockUserRepo{
				findByIDFn: func(_ any, _ ...string) (*User, error) {
					return &User{UserID: 1, UserUUID: uuid.New(), Email: "a@b.com"}, nil
				},
			},
			&mockAuthEventService{})

		_, oerr := svc.ExchangeToken(ctx, OAuthCIBATokenRequestDTO{AuthReqID: "xxxx"}, OAuthClientCredentials{ClientID: "my-client"})
		require.NotNil(t, oerr)
		assert.Equal(t, "server_error", oerr.Code)
	})
}

// ── TestOAuthCIBAService_ApproveRequest ─────────────────────────────────────

func TestOAuthCIBAService_ApproveRequest(t *testing.T) {
	ctx := context.Background()

	t.Run("not found", func(t *testing.T) {
		db, _ := newMockDB(t)
		svc := &oauthCIBAService{
			db:               db,
			cibaRepo:         &mockOAuthCIBARepo{},
			authEventService: &mockAuthEventService{},
		}

		oerr := svc.ApproveRequest(ctx, "xxxx", 1)
		require.NotNil(t, oerr)
		assert.Equal(t, "invalid_grant", oerr.Code)
		assert.Contains(t, oerr.Description, "not found")
	})

	t.Run("repo error", func(t *testing.T) {
		db, _ := newMockDB(t)
		svc := &oauthCIBAService{
			db: db,
			cibaRepo: &mockOAuthCIBARepo{
				findCIBAReqByHashFn: func(_ string) (*OAuthCIBARequest, error) {
					return nil, errors.New("db error")
				},
			},
			authEventService: &mockAuthEventService{},
		}

		oerr := svc.ApproveRequest(ctx, "xxxx", 1)
		require.NotNil(t, oerr)
		assert.Equal(t, "server_error", oerr.Code)
	})

	t.Run("already processed (expired)", func(t *testing.T) {
		db, _ := newMockDB(t)
		svc := &oauthCIBAService{
			db: db,
			cibaRepo: &mockOAuthCIBARepo{
				findCIBAReqByHashFn: func(_ string) (*OAuthCIBARequest, error) {
					return &OAuthCIBARequest{
						OAuthCIBARRequestID: 1,
						Status:              CIBAStatusPending,
						ExpiresAt:           time.Now().Add(-1 * time.Minute),
					}, nil
				},
			},
			authEventService: &mockAuthEventService{},
		}

		oerr := svc.ApproveRequest(ctx, "xxxx", 1)
		require.NotNil(t, oerr)
		assert.Equal(t, "invalid_grant", oerr.Code)
		assert.Contains(t, oerr.Description, "expired")
	})

	t.Run("success", func(t *testing.T) {
		db, _ := newMockDB(t)
		var approvedID int64
		var approvedUser int64

		svc := &oauthCIBAService{
			db: db,
			cibaRepo: &mockOAuthCIBARepo{
				findCIBAReqByHashFn: func(_ string) (*OAuthCIBARequest, error) {
					return &OAuthCIBARequest{
						OAuthCIBARRequestID: 1,
						TenantID:            1,
						Status:              CIBAStatusPending,
						ExpiresAt:           time.Now().Add(5 * time.Minute),
					}, nil
				},
				updateApprovalFn: func(id int64, userID int64) error {
					approvedID = id
					approvedUser = userID
					return nil
				},
			},
			authEventService: &mockAuthEventService{},
		}

		oerr := svc.ApproveRequest(ctx, "xxxx", 42)
		require.Nil(t, oerr)
		assert.Equal(t, int64(1), approvedID)
		assert.Equal(t, int64(42), approvedUser)
	})

	t.Run("updateApprovalContext error", func(t *testing.T) {
		db, _ := newMockDB(t)
		svc := &oauthCIBAService{
			db: db,
			cibaRepo: &mockOAuthCIBARepo{
				findCIBAReqByHashFn: func(_ string) (*OAuthCIBARequest, error) {
					return &OAuthCIBARequest{
						OAuthCIBARRequestID: 1,
						TenantID:            1,
						Status:              CIBAStatusPending,
						ExpiresAt:           time.Now().Add(5 * time.Minute),
					}, nil
				},
				updateApprovalCtxFn: func(_ int64, _ int64, _ string, _ []string) error {
					return errors.New("db error")
				},
			},
			authEventService: &mockAuthEventService{},
		}

		oerr := svc.ApproveRequest(ctx, "xxxx", 42)
		require.NotNil(t, oerr)
		assert.Equal(t, "server_error", oerr.Code)
	})
}

// ── TestOAuthCIBAService_DenyRequest ────────────────────────────────────────

func TestOAuthCIBAService_DenyRequest(t *testing.T) {
	ctx := context.Background()

	t.Run("not found", func(t *testing.T) {
		db, _ := newMockDB(t)
		svc := &oauthCIBAService{
			db:               db,
			cibaRepo:         &mockOAuthCIBARepo{},
			authEventService: &mockAuthEventService{},
		}

		oerr := svc.DenyRequest(ctx, "xxxx", 1)
		require.NotNil(t, oerr)
		assert.Equal(t, "invalid_grant", oerr.Code)
		assert.Contains(t, oerr.Description, "not found")
	})

	t.Run("repo error", func(t *testing.T) {
		db, _ := newMockDB(t)
		svc := &oauthCIBAService{
			db: db,
			cibaRepo: &mockOAuthCIBARepo{
				findCIBAReqByHashFn: func(_ string) (*OAuthCIBARequest, error) {
					return nil, errors.New("db error")
				},
			},
			authEventService: &mockAuthEventService{},
		}

		oerr := svc.DenyRequest(ctx, "xxxx", 1)
		require.NotNil(t, oerr)
		assert.Equal(t, "server_error", oerr.Code)
	})

	t.Run("update status error", func(t *testing.T) {
		db, _ := newMockDB(t)
		svc := &oauthCIBAService{
			db: db,
			cibaRepo: &mockOAuthCIBARepo{
				findCIBAReqByHashFn: func(_ string) (*OAuthCIBARequest, error) {
					return &OAuthCIBARequest{
						OAuthCIBARRequestID: 1,
						Status:              CIBAStatusPending,
						ExpiresAt:           time.Now().Add(5 * time.Minute),
					}, nil
				},
				updateStatusFn: func(_ int64, _ string) error {
					return errors.New("db error")
				},
			},
			authEventService: &mockAuthEventService{},
		}

		oerr := svc.DenyRequest(ctx, "xxxx", 1)
		require.NotNil(t, oerr)
		assert.Equal(t, "server_error", oerr.Code)
	})

	t.Run("success", func(t *testing.T) {
		db, _ := newMockDB(t)
		var deniedID int64

		svc := &oauthCIBAService{
			db: db,
			cibaRepo: &mockOAuthCIBARepo{
				findCIBAReqByHashFn: func(_ string) (*OAuthCIBARequest, error) {
					return &OAuthCIBARequest{
						OAuthCIBARRequestID: 1,
						TenantID:            1,
						Status:              CIBAStatusPending,
						ExpiresAt:           time.Now().Add(5 * time.Minute),
					}, nil
				},
				updateStatusFn: func(id int64, status string) error {
					deniedID = id
					return nil
				},
			},
			authEventService: &mockAuthEventService{},
		}

		oerr := svc.DenyRequest(ctx, "xxxx", 42)
		require.Nil(t, oerr)
		assert.Equal(t, int64(1), deniedID)
	})
}
