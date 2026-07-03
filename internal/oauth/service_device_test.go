package oauth

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"
	"time"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/maintainerd/maintainerd-auth/internal/platform/config"
	"github.com/maintainerd/maintainerd-auth/internal/platform/crypto"
	"github.com/maintainerd/maintainerd-auth/internal/platform/jwt"
	"github.com/maintainerd/maintainerd-auth/internal/platform/middleware"
	"github.com/maintainerd/maintainerd-auth/internal/platform/ptr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func newOAuthDeviceSvc(
	db *gorm.DB,
	deviceCodeRepo *mockOAuthDeviceCodeRepo,
	userRepo *mockUserRepo,
	authEventSvc *mockAuthEventService,
) OAuthDeviceService {
	return &oauthDeviceService{
		db:               db,
		clientRepo:       &mockClientRepo{},
		deviceCodeRepo:   deviceCodeRepo,
		userRepo:         userRepo,
		userIdentityRepo: &mockUserIdentityRepo{},
		authEventService: authEventSvc,
	}
}

func TestOAuthDeviceAuthContextHelpers(t *testing.T) {
	t.Run("auth context defaults without claims", func(t *testing.T) {
		acr, amr := authContextFromContext(context.Background())

		assert.Equal(t, jwt.ACRLevel1, acr)
		assert.Equal(t, []string{jwt.AMRPassword}, amr)
	})

	t.Run("auth context uses claims", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req = middleware.WithJWTClaims(req, &middleware.JWTClaims{
			ACR: jwt.ACRLevel2,
			AMR: []string{jwt.AMRPassword, jwt.AMRMFA},
		})

		acr, amr := authContextFromContext(req.Context())

		assert.Equal(t, jwt.ACRLevel2, acr)
		assert.Equal(t, []string{jwt.AMRPassword, jwt.AMRMFA}, amr)
	})

	t.Run("auth context fills missing claim values", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req = middleware.WithJWTClaims(req, &middleware.JWTClaims{})

		acr, amr := authContextFromContext(req.Context())

		assert.Equal(t, jwt.ACRLevel1, acr)
		assert.Equal(t, []string{jwt.AMRPassword}, amr)
	})

	t.Run("persisted context decodes values", func(t *testing.T) {
		acr, amr := persistedAuthContext(jwt.ACRLevel2, []byte(`["pwd","mfa"]`))

		assert.Equal(t, jwt.ACRLevel2, acr)
		assert.Equal(t, []string{jwt.AMRPassword, jwt.AMRMFA}, amr)
	})

	t.Run("persisted context defaults invalid values", func(t *testing.T) {
		acr, amr := persistedAuthContext("", []byte(`{`))

		assert.Equal(t, jwt.ACRLevel1, acr)
		assert.Equal(t, []string{jwt.AMRPassword}, amr)
	})
}

type mockOAuthDeviceCodeRepo struct {
	findByDeviceCodeHashFn func(string) (*OAuthDeviceCode, error)
	findByUserCodeFn       func(string) (*OAuthDeviceCode, error)
	createFn               func(*OAuthDeviceCode) (*OAuthDeviceCode, error)
	updateStatusFn         func(int64, string, *int64) error
	updateApprovalFn       func(int64, int64, string, []string) error
	updateLastPollAtFn     func(int64) error
}

func (m *mockOAuthDeviceCodeRepo) WithTx(_ *gorm.DB) OAuthDeviceCodeRepository { return m }
func (m *mockOAuthDeviceCodeRepo) FindByDeviceCodeHash(hash string) (*OAuthDeviceCode, error) {
	if m.findByDeviceCodeHashFn != nil {
		return m.findByDeviceCodeHashFn(hash)
	}
	return nil, nil
}
func (m *mockOAuthDeviceCodeRepo) FindByUserCode(userCode string) (*OAuthDeviceCode, error) {
	if m.findByUserCodeFn != nil {
		return m.findByUserCodeFn(userCode)
	}
	return nil, nil
}
func (m *mockOAuthDeviceCodeRepo) UpdateStatus(id int64, status string, userID *int64) error {
	if m.updateStatusFn != nil {
		return m.updateStatusFn(id, status, userID)
	}
	return nil
}
func (m *mockOAuthDeviceCodeRepo) UpdateApproval(id int64, userID int64, acr string, amr []string) error {
	if m.updateApprovalFn != nil {
		return m.updateApprovalFn(id, userID, acr, amr)
	}
	return m.UpdateStatus(id, DeviceCodeStatusApproved, &userID)
}
func (m *mockOAuthDeviceCodeRepo) UpdateLastPollAt(id int64) error {
	if m.updateLastPollAtFn != nil {
		return m.updateLastPollAtFn(id)
	}
	return nil
}
func (m *mockOAuthDeviceCodeRepo) DeleteExpired(_ time.Time) (int64, error) { return 0, nil }
func (m *mockOAuthDeviceCodeRepo) Create(e *OAuthDeviceCode) (*OAuthDeviceCode, error) {
	if m.createFn != nil {
		return m.createFn(e)
	}
	return e, nil
}
func (m *mockOAuthDeviceCodeRepo) CreateOrUpdate(e *OAuthDeviceCode) (*OAuthDeviceCode, error) {
	return e, nil
}
func (m *mockOAuthDeviceCodeRepo) FindAll(_ ...string) ([]OAuthDeviceCode, error) { return nil, nil }
func (m *mockOAuthDeviceCodeRepo) FindByUUID(_ any, _ ...string) (*OAuthDeviceCode, error) {
	return nil, nil
}
func (m *mockOAuthDeviceCodeRepo) FindByUUIDs(_ []string, _ ...string) ([]OAuthDeviceCode, error) {
	return nil, nil
}
func (m *mockOAuthDeviceCodeRepo) FindByID(_ any, _ ...string) (*OAuthDeviceCode, error) {
	return nil, nil
}
func (m *mockOAuthDeviceCodeRepo) UpdateByUUID(_, _ any) (*OAuthDeviceCode, error) { return nil, nil }
func (m *mockOAuthDeviceCodeRepo) UpdateByID(_, _ any) (*OAuthDeviceCode, error)   { return nil, nil }
func (m *mockOAuthDeviceCodeRepo) DeleteByUUID(_ any) error                        { return nil }
func (m *mockOAuthDeviceCodeRepo) DeleteByID(_ any) error                          { return nil }
func (m *mockOAuthDeviceCodeRepo) Paginate(_ map[string]any, _, _ int, _ ...string) (*PaginationResult[OAuthDeviceCode], error) {
	return nil, nil
}

func expectDeviceClientLookup(mock sqlmock.Sqlmock) {
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
}

// ── TestOAuthDeviceService_Authorize ────────────────────────────────────────

func TestOAuthDeviceService_Authorize(t *testing.T) {
	ctx := context.Background()

	t.Run("client not found", func(t *testing.T) {
		db, mock := newMockDB(t)
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT`)).WillReturnError(gorm.ErrRecordNotFound)

		svc := newOAuthDeviceSvc(db, &mockOAuthDeviceCodeRepo{}, &mockUserRepo{}, &mockAuthEventService{})

		_, oerr := svc.Authorize(ctx, OAuthDeviceAuthorizationRequestDTO{}, OAuthClientCredentials{ClientID: "unknown"})
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

		svc := newOAuthDeviceSvc(db, &mockOAuthDeviceCodeRepo{}, &mockUserRepo{}, &mockAuthEventService{})

		_, oerr := svc.Authorize(ctx, OAuthDeviceAuthorizationRequestDTO{}, OAuthClientCredentials{ClientID: "my-client"})
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
			pq.StringArray{GrantTypeDeviceCode}, pq.StringArray{ResponseTypeCode}, nil, nil,
			false, pq.StringArray{"openid"}, time.Now(), time.Now(),
		)
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT`)).WillReturnRows(rows)
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT`)).WillReturnRows(sqlmock.NewRows(nil))
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT`)).WillReturnRows(sqlmock.NewRows(nil))

		svc := newOAuthDeviceSvc(db, &mockOAuthDeviceCodeRepo{}, &mockUserRepo{}, &mockAuthEventService{})

		_, oerr := svc.Authorize(ctx, OAuthDeviceAuthorizationRequestDTO{Scope: "profile admin"}, OAuthClientCredentials{ClientID: "my-client"})
		require.NotNil(t, oerr)
		assert.Equal(t, "invalid_scope", oerr.Code)
	})

	t.Run("client with restricted allowed_scopes", func(t *testing.T) {
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
			pq.StringArray{GrantTypeDeviceCode}, pq.StringArray{ResponseTypeCode}, nil, nil,
			false, pq.StringArray{"openid"}, time.Now(), time.Now(),
		)
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT`)).WillReturnRows(rows)
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT`)).WillReturnRows(sqlmock.NewRows(nil))
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT`)).WillReturnRows(sqlmock.NewRows(nil))

		svc := newOAuthDeviceSvc(db, &mockOAuthDeviceCodeRepo{}, &mockUserRepo{}, &mockAuthEventService{})

		_, oerr := svc.Authorize(ctx, OAuthDeviceAuthorizationRequestDTO{Scope: "email"}, OAuthClientCredentials{ClientID: "my-client"})
		require.NotNil(t, oerr)
		assert.Equal(t, "invalid_scope", oerr.Code)
	})

	t.Run("create error", func(t *testing.T) {
		db, mock := newMockDB(t)
		expectDeviceClientLookup(mock)

		svc := newOAuthDeviceSvc(db,
			&mockOAuthDeviceCodeRepo{
				createFn: func(_ *OAuthDeviceCode) (*OAuthDeviceCode, error) {
					return nil, errors.New("db error")
				},
			},
			&mockUserRepo{}, &mockAuthEventService{})

		_, oerr := svc.Authorize(ctx, OAuthDeviceAuthorizationRequestDTO{Scope: "openid"}, OAuthClientCredentials{ClientID: "my-client"})
		require.NotNil(t, oerr)
		assert.Equal(t, "server_error", oerr.Code)
	})

	t.Run("success", func(t *testing.T) {
		db, mock := newMockDB(t)
		expectDeviceClientLookup(mock)

		svc := newOAuthDeviceSvc(db, &mockOAuthDeviceCodeRepo{}, &mockUserRepo{}, &mockAuthEventService{})

		result, oerr := svc.Authorize(ctx, OAuthDeviceAuthorizationRequestDTO{Scope: "openid"}, OAuthClientCredentials{ClientID: "my-client"})
		require.Nil(t, oerr)
		require.NotNil(t, result)
		assert.NotEmpty(t, result.DeviceCode)
		assert.NotEmpty(t, result.UserCode)
		assert.Contains(t, result.VerificationURI, "/device")
		assert.Equal(t, 900, result.ExpiresIn)
		assert.Equal(t, 5, result.Interval)
	})

	t.Run("GenerateRandomString error", func(t *testing.T) {
		db, mock := newMockDB(t)
		expectDeviceClientLookup(mock)

		orig := crypto.GenerateRandomString
		defer func() { crypto.GenerateRandomString = orig }()
		crypto.GenerateRandomString = func(int) (string, error) { return "", errors.New("rand failure") }

		svc := newOAuthDeviceSvc(db, &mockOAuthDeviceCodeRepo{}, &mockUserRepo{}, &mockAuthEventService{})

		_, oerr := svc.Authorize(ctx, OAuthDeviceAuthorizationRequestDTO{Scope: "openid"}, OAuthClientCredentials{ClientID: "my-client"})
		require.NotNil(t, oerr)
		assert.Equal(t, "server_error", oerr.Code)
	})

	t.Run("generateUserCode error", func(t *testing.T) {
		db, mock := newMockDB(t)
		expectDeviceClientLookup(mock)

		orig := crypto.GenerateRandomString
		defer func() { crypto.GenerateRandomString = orig }()
		callCount := 0
		crypto.GenerateRandomString = func(int) (string, error) {
			callCount++
			if callCount == 1 {
				return "valid-device-code-with-thirty-two-bytes!", nil
			}
			return "", errors.New("rand failure")
		}

		svc := newOAuthDeviceSvc(db, &mockOAuthDeviceCodeRepo{}, &mockUserRepo{}, &mockAuthEventService{})

		_, oerr := svc.Authorize(ctx, OAuthDeviceAuthorizationRequestDTO{Scope: "openid"}, OAuthClientCredentials{ClientID: "my-client"})
		require.NotNil(t, oerr)
		assert.Equal(t, "server_error", oerr.Code)
	})
}

// ── TestOAuthDeviceService_VerifyUserCode ────────────────────────────────────

func TestOAuthDeviceService_VerifyUserCode(t *testing.T) {
	ctx := context.Background()

	t.Run("not found", func(t *testing.T) {
		db, _ := newMockDB(t)
		svc := &oauthDeviceService{
			db: db,
			deviceCodeRepo: &mockOAuthDeviceCodeRepo{
				findByUserCodeFn: func(_ string) (*OAuthDeviceCode, error) {
					return nil, nil
				},
			},
			authEventService: &mockAuthEventService{},
		}

		oerr := svc.VerifyUserCode(ctx, OAuthDeviceVerifyRequestDTO{UserCode: "ABCD-EFGH"}, 1)
		require.NotNil(t, oerr)
		assert.Equal(t, "invalid_grant", oerr.Code)
	})

	t.Run("repo error", func(t *testing.T) {
		db, _ := newMockDB(t)
		svc := &oauthDeviceService{
			db: db,
			deviceCodeRepo: &mockOAuthDeviceCodeRepo{
				findByUserCodeFn: func(_ string) (*OAuthDeviceCode, error) {
					return nil, errors.New("db error")
				},
			},
			authEventService: &mockAuthEventService{},
		}

		oerr := svc.VerifyUserCode(ctx, OAuthDeviceVerifyRequestDTO{UserCode: "ABCD-EFGH"}, 1)
		require.NotNil(t, oerr)
		assert.Equal(t, "server_error", oerr.Code)
	})

	t.Run("expired", func(t *testing.T) {
		db, _ := newMockDB(t)
		svc := &oauthDeviceService{
			db: db,
			deviceCodeRepo: &mockOAuthDeviceCodeRepo{
				findByUserCodeFn: func(_ string) (*OAuthDeviceCode, error) {
					return &OAuthDeviceCode{
						OAuthDeviceCodeID: 1,
						Status:            DeviceCodeStatusPending,
						ExpiresAt:         time.Now().Add(-1 * time.Minute),
					}, nil
				},
			},
			authEventService: &mockAuthEventService{},
		}

		oerr := svc.VerifyUserCode(ctx, OAuthDeviceVerifyRequestDTO{UserCode: "ABCD-EFGH"}, 1)
		require.NotNil(t, oerr)
		assert.Equal(t, "invalid_grant", oerr.Code)
		assert.Contains(t, oerr.Description, "expired")
	})

	t.Run("update approval error", func(t *testing.T) {
		db, _ := newMockDB(t)
		svc := &oauthDeviceService{
			db: db,
			deviceCodeRepo: &mockOAuthDeviceCodeRepo{
				findByUserCodeFn: func(_ string) (*OAuthDeviceCode, error) {
					return &OAuthDeviceCode{
						OAuthDeviceCodeID: 1,
						Status:            DeviceCodeStatusPending,
						ExpiresAt:         time.Now().Add(5 * time.Minute),
					}, nil
				},
				updateApprovalFn: func(_ int64, _ int64, _ string, _ []string) error {
					return errors.New("db error")
				},
			},
			authEventService: &mockAuthEventService{},
		}

		oerr := svc.VerifyUserCode(ctx, OAuthDeviceVerifyRequestDTO{UserCode: "ABCD-EFGH"}, 42)
		require.NotNil(t, oerr)
		assert.Equal(t, "server_error", oerr.Code)
	})

	t.Run("success", func(t *testing.T) {
		db, _ := newMockDB(t)
		var updatedID int64
		var updatedStatus string
		var updatedUserID *int64

		svc := &oauthDeviceService{
			db: db,
			deviceCodeRepo: &mockOAuthDeviceCodeRepo{
				findByUserCodeFn: func(_ string) (*OAuthDeviceCode, error) {
					return &OAuthDeviceCode{
						OAuthDeviceCodeID: 1,
						TenantID:          1,
						Status:            DeviceCodeStatusPending,
						ExpiresAt:         time.Now().Add(5 * time.Minute),
					}, nil
				},
				updateStatusFn: func(id int64, status string, userID *int64) error {
					updatedID = id
					updatedStatus = status
					updatedUserID = userID
					return nil
				},
			},
			authEventService: &mockAuthEventService{},
		}

		oerr := svc.VerifyUserCode(ctx, OAuthDeviceVerifyRequestDTO{UserCode: "ABCD-EFGH"}, 42)
		require.Nil(t, oerr)
		assert.Equal(t, int64(1), updatedID)
		assert.Equal(t, DeviceCodeStatusApproved, updatedStatus)
		require.NotNil(t, updatedUserID)
		assert.Equal(t, int64(42), *updatedUserID)
	})
}

// ── TestOAuthDeviceService_ExchangeToken ─────────────────────────────────────

func TestOAuthDeviceService_ExchangeToken(t *testing.T) {
	ctx := context.Background()

	t.Run("client auth error", func(t *testing.T) {
		db, mock := newMockDB(t)
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT`)).WillReturnError(gorm.ErrRecordNotFound)

		svc := newOAuthDeviceSvc(db, &mockOAuthDeviceCodeRepo{}, &mockUserRepo{}, &mockAuthEventService{})

		_, oerr := svc.ExchangeToken(ctx, OAuthDeviceTokenRequestDTO{DeviceCode: "xxxx"}, OAuthClientCredentials{ClientID: "unknown"})
		require.NotNil(t, oerr)
		assert.Equal(t, "invalid_client", oerr.Code)
	})

	t.Run("device_code not found", func(t *testing.T) {
		db, mock := newMockDB(t)
		expectDeviceClientLookup(mock)

		svc := newOAuthDeviceSvc(db,
			&mockOAuthDeviceCodeRepo{
				findByDeviceCodeHashFn: func(_ string) (*OAuthDeviceCode, error) {
					return nil, nil
				},
			},
			&mockUserRepo{}, &mockAuthEventService{})

		_, oerr := svc.ExchangeToken(ctx, OAuthDeviceTokenRequestDTO{DeviceCode: "xxxx"}, OAuthClientCredentials{ClientID: "my-client"})
		require.NotNil(t, oerr)
		assert.Equal(t, "invalid_grant", oerr.Code)
	})

	t.Run("client mismatch", func(t *testing.T) {
		db, mock := newMockDB(t)
		expectDeviceClientLookup(mock)

		svc := newOAuthDeviceSvc(db,
			&mockOAuthDeviceCodeRepo{
				findByDeviceCodeHashFn: func(_ string) (*OAuthDeviceCode, error) {
					return &OAuthDeviceCode{
						OAuthDeviceCodeID: 1,
						ClientID:          999,
						Status:            DeviceCodeStatusPending,
						Interval:          5,
						ExpiresAt:         time.Now().Add(5 * time.Minute),
					}, nil
				},
			},
			&mockUserRepo{}, &mockAuthEventService{})

		_, oerr := svc.ExchangeToken(ctx, OAuthDeviceTokenRequestDTO{DeviceCode: "xxxx"}, OAuthClientCredentials{ClientID: "my-client"})
		require.NotNil(t, oerr)
		assert.Equal(t, "invalid_grant", oerr.Code)
		assert.Contains(t, oerr.Description, "does not belong")
	})

	t.Run("slow_down", func(t *testing.T) {
		db, mock := newMockDB(t)
		expectDeviceClientLookup(mock)
		lastPoll := time.Now().Add(-1 * time.Second)

		svc := newOAuthDeviceSvc(db,
			&mockOAuthDeviceCodeRepo{
				findByDeviceCodeHashFn: func(_ string) (*OAuthDeviceCode, error) {
					return &OAuthDeviceCode{
						OAuthDeviceCodeID: 1,
						ClientID:          10,
						Status:            DeviceCodeStatusPending,
						Interval:          5,
						LastPollAt:        &lastPoll,
						ExpiresAt:         time.Now().Add(5 * time.Minute),
					}, nil
				},
			},
			&mockUserRepo{}, &mockAuthEventService{})

		_, oerr := svc.ExchangeToken(ctx, OAuthDeviceTokenRequestDTO{DeviceCode: "xxxx"}, OAuthClientCredentials{ClientID: "my-client"})
		require.NotNil(t, oerr)
		assert.Equal(t, "slow_down", oerr.Code)
	})

	t.Run("denied", func(t *testing.T) {
		db, mock := newMockDB(t)
		expectDeviceClientLookup(mock)

		svc := newOAuthDeviceSvc(db,
			&mockOAuthDeviceCodeRepo{
				findByDeviceCodeHashFn: func(_ string) (*OAuthDeviceCode, error) {
					return &OAuthDeviceCode{
						OAuthDeviceCodeID: 1,
						ClientID:          10,
						Status:            DeviceCodeStatusDenied,
						Interval:          5,
						ExpiresAt:         time.Now().Add(5 * time.Minute),
					}, nil
				},
			},
			&mockUserRepo{}, &mockAuthEventService{})

		_, oerr := svc.ExchangeToken(ctx, OAuthDeviceTokenRequestDTO{DeviceCode: "xxxx"}, OAuthClientCredentials{ClientID: "my-client"})
		require.NotNil(t, oerr)
		assert.Equal(t, "access_denied", oerr.Code)
	})

	t.Run("expired status", func(t *testing.T) {
		db, mock := newMockDB(t)
		expectDeviceClientLookup(mock)

		svc := newOAuthDeviceSvc(db,
			&mockOAuthDeviceCodeRepo{
				findByDeviceCodeHashFn: func(_ string) (*OAuthDeviceCode, error) {
					return &OAuthDeviceCode{
						ClientID:  10,
						Status:    DeviceCodeStatusExpired,
						ExpiresAt: time.Now().Add(-1 * time.Minute),
					}, nil
				},
			},
			&mockUserRepo{}, &mockAuthEventService{})

		_, oerr := svc.ExchangeToken(ctx, OAuthDeviceTokenRequestDTO{DeviceCode: "xxxx"}, OAuthClientCredentials{ClientID: "my-client"})
		require.NotNil(t, oerr)
		assert.Equal(t, "expired_token", oerr.Code)
	})

	t.Run("approved with nil userID", func(t *testing.T) {
		db, mock := newMockDB(t)
		expectDeviceClientLookup(mock)

		svc := newOAuthDeviceSvc(db,
			&mockOAuthDeviceCodeRepo{
				findByDeviceCodeHashFn: func(_ string) (*OAuthDeviceCode, error) {
					return &OAuthDeviceCode{
						OAuthDeviceCodeID: 1,
						ClientID:          10,
						UserID:            nil,
						Status:            DeviceCodeStatusApproved,
						Interval:          5,
						ExpiresAt:         time.Now().Add(5 * time.Minute),
					}, nil
				},
			},
			&mockUserRepo{}, &mockAuthEventService{})

		_, oerr := svc.ExchangeToken(ctx, OAuthDeviceTokenRequestDTO{DeviceCode: "xxxx"}, OAuthClientCredentials{ClientID: "my-client"})
		require.NotNil(t, oerr)
		assert.Equal(t, "server_error", oerr.Code)
	})

	t.Run("user not found after approval", func(t *testing.T) {
		db, mock := newMockDB(t)
		expectDeviceClientLookup(mock)
		userID := int64(1)

		svc := newOAuthDeviceSvc(db,
			&mockOAuthDeviceCodeRepo{
				findByDeviceCodeHashFn: func(_ string) (*OAuthDeviceCode, error) {
					return &OAuthDeviceCode{
						OAuthDeviceCodeID: 1,
						ClientID:          10,
						TenantID:          1,
						UserID:            &userID,
						Scope:             "openid",
						Status:            DeviceCodeStatusApproved,
						Interval:          5,
						ExpiresAt:         time.Now().Add(5 * time.Minute),
					}, nil
				},
			},
			&mockUserRepo{
				findByIDFn: func(_ any, _ ...string) (*User, error) {
					return nil, nil
				},
			},
			&mockAuthEventService{})

		_, oerr := svc.ExchangeToken(ctx, OAuthDeviceTokenRequestDTO{DeviceCode: "xxxx"}, OAuthClientCredentials{ClientID: "my-client"})
		require.NotNil(t, oerr)
		assert.Equal(t, "server_error", oerr.Code)
	})

	t.Run("repo lookup error", func(t *testing.T) {
		db, mock := newMockDB(t)
		expectDeviceClientLookup(mock)

		svc := newOAuthDeviceSvc(db,
			&mockOAuthDeviceCodeRepo{
				findByDeviceCodeHashFn: func(_ string) (*OAuthDeviceCode, error) {
					return nil, errors.New("db error")
				},
			},
			&mockUserRepo{}, &mockAuthEventService{})

		_, oerr := svc.ExchangeToken(ctx, OAuthDeviceTokenRequestDTO{DeviceCode: "xxxx"}, OAuthClientCredentials{ClientID: "my-client"})
		require.NotNil(t, oerr)
		assert.Equal(t, "server_error", oerr.Code)
	})

	t.Run("expired", func(t *testing.T) {
		db, mock := newMockDB(t)
		expectDeviceClientLookup(mock)

		svc := newOAuthDeviceSvc(db,
			&mockOAuthDeviceCodeRepo{
				findByDeviceCodeHashFn: func(_ string) (*OAuthDeviceCode, error) {
					return &OAuthDeviceCode{
						ClientID:  10,
						Status:    DeviceCodeStatusPending,
						ExpiresAt: time.Now().Add(-1 * time.Minute),
					}, nil
				},
			},
			&mockUserRepo{}, &mockAuthEventService{})

		_, oerr := svc.ExchangeToken(ctx, OAuthDeviceTokenRequestDTO{DeviceCode: "xxxx"}, OAuthClientCredentials{ClientID: "my-client"})
		require.NotNil(t, oerr)
		assert.Equal(t, "expired_token", oerr.Code)
	})

	t.Run("success", func(t *testing.T) {
		initTestJWTKeysService(t)
		origHost := config.AppPublicHostname
		config.AppPublicHostname = "https://auth.example.com"
		defer func() { config.AppPublicHostname = origHost }()

		db, mock := newMockDB(t)
		expectDeviceClientLookup(mock)
		userID := int64(1)

		svc := newOAuthDeviceSvc(db,
			&mockOAuthDeviceCodeRepo{
				findByDeviceCodeHashFn: func(_ string) (*OAuthDeviceCode, error) {
					return &OAuthDeviceCode{
						OAuthDeviceCodeID: 1,
						ClientID:          10,
						TenantID:          1,
						UserID:            &userID,
						Scope:             "openid",
						Status:            DeviceCodeStatusApproved,
						Interval:          5,
						ExpiresAt:         time.Now().Add(5 * time.Minute),
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

		result, oerr := svc.ExchangeToken(ctx, OAuthDeviceTokenRequestDTO{DeviceCode: "xxxx"}, OAuthClientCredentials{ClientID: "my-client"})
		require.Nil(t, oerr)
		require.NotNil(t, result)
		assert.NotEmpty(t, result.AccessToken)
		assert.Equal(t, "Bearer", result.TokenType)
		assert.Equal(t, "openid", result.Scope)
	})

	t.Run("authorization_pending", func(t *testing.T) {
		db, mock := newMockDB(t)
		expectDeviceClientLookup(mock)

		svc := newOAuthDeviceSvc(db,
			&mockOAuthDeviceCodeRepo{
				findByDeviceCodeHashFn: func(_ string) (*OAuthDeviceCode, error) {
					return &OAuthDeviceCode{
						OAuthDeviceCodeID: 1,
						ClientID:          10,
						Status:            DeviceCodeStatusPending,
						Interval:          5,
						ExpiresAt:         time.Now().Add(5 * time.Minute),
					}, nil
				},
			},
			&mockUserRepo{}, &mockAuthEventService{})

		_, oerr := svc.ExchangeToken(ctx, OAuthDeviceTokenRequestDTO{DeviceCode: "xxxx"}, OAuthClientCredentials{ClientID: "my-client"})
		require.NotNil(t, oerr)
		assert.Equal(t, "authorization_pending", oerr.Code)
	})

	t.Run("unexpected status", func(t *testing.T) {
		db, mock := newMockDB(t)
		expectDeviceClientLookup(mock)

		svc := newOAuthDeviceSvc(db,
			&mockOAuthDeviceCodeRepo{
				findByDeviceCodeHashFn: func(_ string) (*OAuthDeviceCode, error) {
					return &OAuthDeviceCode{
						OAuthDeviceCodeID: 1,
						ClientID:          10,
						Status:            "unknown_status",
						Interval:          5,
						ExpiresAt:         time.Now().Add(5 * time.Minute),
					}, nil
				},
			},
			&mockUserRepo{}, &mockAuthEventService{})

		_, oerr := svc.ExchangeToken(ctx, OAuthDeviceTokenRequestDTO{DeviceCode: "xxxx"}, OAuthClientCredentials{ClientID: "my-client"})
		require.NotNil(t, oerr)
		assert.Equal(t, "invalid_grant", oerr.Code)
		assert.Contains(t, oerr.Description, "unexpected")
	})

	t.Run("empty user email", func(t *testing.T) {
		initTestJWTKeysService(t)
		origHost := config.AppPublicHostname
		config.AppPublicHostname = "https://auth.example.com"
		defer func() { config.AppPublicHostname = origHost }()

		db, mock := newMockDB(t)
		expectDeviceClientLookup(mock)
		userID := int64(1)

		svc := newOAuthDeviceSvc(db,
			&mockOAuthDeviceCodeRepo{
				findByDeviceCodeHashFn: func(_ string) (*OAuthDeviceCode, error) {
					return &OAuthDeviceCode{
						OAuthDeviceCodeID: 1,
						ClientID:          10,
						TenantID:          1,
						UserID:            &userID,
						Scope:             "openid",
						Status:            DeviceCodeStatusApproved,
						Interval:          5,
						ExpiresAt:         time.Now().Add(5 * time.Minute),
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
					return &User{UserID: 1, UserUUID: uuid.New(), Email: ""}, nil
				},
			},
			&mockAuthEventService{})

		result, oerr := svc.ExchangeToken(ctx, OAuthDeviceTokenRequestDTO{DeviceCode: "xxxx"}, OAuthClientCredentials{ClientID: "my-client"})
		require.Nil(t, oerr)
		require.NotNil(t, result)
		assert.NotEmpty(t, result.AccessToken)
		assert.Equal(t, "Bearer", result.TokenType)
		assert.Equal(t, "openid", result.Scope)
	})

	// NOTE: The nil-client path in sendDeviceApprovalEmail is currently
	// unreachable in a success flow because jwt.GenerateAccessTokenWithOptionsContext
	// requires a non-empty providerID (derived from record.Client.IdentityProvider).
	// When record.Client is nil, token generation fails with a server_error.
	// A nil-client sendDeviceApprovalEmail test can be added after that dependency
	// is relaxed.

	t.Run("JWT generation error", func(t *testing.T) {
		jwt.ResetJWTKeys()

		db, mock := newMockDB(t)
		expectDeviceClientLookup(mock)
		userID := int64(1)

		svc := newOAuthDeviceSvc(db,
			&mockOAuthDeviceCodeRepo{
				findByDeviceCodeHashFn: func(_ string) (*OAuthDeviceCode, error) {
					return &OAuthDeviceCode{
						OAuthDeviceCodeID: 1,
						ClientID:          10,
						TenantID:          1,
						UserID:            &userID,
						Scope:             "openid",
						Status:            DeviceCodeStatusApproved,
						Interval:          5,
						ExpiresAt:         time.Now().Add(5 * time.Minute),
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

		_, oerr := svc.ExchangeToken(ctx, OAuthDeviceTokenRequestDTO{DeviceCode: "xxxx"}, OAuthClientCredentials{ClientID: "my-client"})
		require.NotNil(t, oerr)
		assert.Equal(t, "server_error", oerr.Code)
	})
}

// ── TestOAuthDeviceService_DenyUserCode ──────────────────────────────────────

func TestOAuthDeviceService_DenyUserCode(t *testing.T) {
	ctx := context.Background()

	t.Run("not found", func(t *testing.T) {
		db, _ := newMockDB(t)
		svc := &oauthDeviceService{
			db: db,
			deviceCodeRepo: &mockOAuthDeviceCodeRepo{
				findByUserCodeFn: func(_ string) (*OAuthDeviceCode, error) {
					return nil, nil
				},
			},
			authEventService: &mockAuthEventService{},
		}

		oerr := svc.DenyUserCode(ctx, OAuthDeviceVerifyRequestDTO{UserCode: "ABCD-EFGH"}, 1)
		require.NotNil(t, oerr)
		assert.Equal(t, "invalid_grant", oerr.Code)
	})

	t.Run("repo error", func(t *testing.T) {
		db, _ := newMockDB(t)
		svc := &oauthDeviceService{
			db: db,
			deviceCodeRepo: &mockOAuthDeviceCodeRepo{
				findByUserCodeFn: func(_ string) (*OAuthDeviceCode, error) {
					return nil, errors.New("db error")
				},
			},
			authEventService: &mockAuthEventService{},
		}

		oerr := svc.DenyUserCode(ctx, OAuthDeviceVerifyRequestDTO{UserCode: "ABCD-EFGH"}, 1)
		require.NotNil(t, oerr)
		assert.Equal(t, "server_error", oerr.Code)
	})

	t.Run("update status error", func(t *testing.T) {
		db, _ := newMockDB(t)
		svc := &oauthDeviceService{
			db: db,
			deviceCodeRepo: &mockOAuthDeviceCodeRepo{
				findByUserCodeFn: func(_ string) (*OAuthDeviceCode, error) {
					return &OAuthDeviceCode{
						OAuthDeviceCodeID: 1,
						Status:            DeviceCodeStatusPending,
						ExpiresAt:         time.Now().Add(5 * time.Minute),
					}, nil
				},
				updateStatusFn: func(_ int64, _ string, _ *int64) error {
					return errors.New("db error")
				},
			},
			authEventService: &mockAuthEventService{},
		}

		oerr := svc.DenyUserCode(ctx, OAuthDeviceVerifyRequestDTO{UserCode: "ABCD-EFGH"}, 42)
		require.NotNil(t, oerr)
		assert.Equal(t, "server_error", oerr.Code)
	})

	t.Run("success", func(t *testing.T) {
		db, _ := newMockDB(t)
		var updatedID int64
		var updatedStatus string

		svc := &oauthDeviceService{
			db: db,
			deviceCodeRepo: &mockOAuthDeviceCodeRepo{
				findByUserCodeFn: func(_ string) (*OAuthDeviceCode, error) {
					return &OAuthDeviceCode{
						OAuthDeviceCodeID: 1,
						TenantID:          1,
						Status:            DeviceCodeStatusPending,
						ExpiresAt:         time.Now().Add(5 * time.Minute),
					}, nil
				},
				updateStatusFn: func(id int64, status string, _ *int64) error {
					updatedID = id
					updatedStatus = status
					return nil
				},
			},
			authEventService: &mockAuthEventService{},
		}

		oerr := svc.DenyUserCode(ctx, OAuthDeviceVerifyRequestDTO{UserCode: "ABCD-EFGH"}, 42)
		require.Nil(t, oerr)
		assert.Equal(t, int64(1), updatedID)
		assert.Equal(t, DeviceCodeStatusDenied, updatedStatus)
	})
}
