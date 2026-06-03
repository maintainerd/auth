package oauth

import (
	"context"
	"regexp"
	"testing"
	"time"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/maintainerd/auth/internal/platform/config"
	"github.com/maintainerd/auth/internal/platform/jwt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func newOAuthTokenExchangeSvc(
	db *gorm.DB,
	userRepo *mockUserRepo,
	authEventSvc *mockAuthEventService,
) OAuthTokenExchangeService {
	return &oauthTokenExchangeService{
		db:               db,
		clientRepo:       &mockClientRepo{},
		userRepo:         userRepo,
		authEventService: authEventSvc,
	}
}

func expectTokenExchangeClientLookup(mock sqlmock.Sqlmock) {
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
		pq.StringArray{GrantTypeTokenExchange}, pq.StringArray{}, nil, nil,
		false, time.Now(), time.Now(),
	)
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT`)).WillReturnRows(rows)
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT`)).WillReturnRows(sqlmock.NewRows([]string{
		"identity_provider_id", "identity_provider_uuid", "tenant_id",
		"name", "display_name", "provider", "provider_type",
		"identifier", "config", "status", "is_default", "is_system",
		"created_at", "updated_at",
	}).AddRow(
		100, uuid.New(), 1,
		"default", "Default Provider", "local", "local",
		"default-provider", `{}`, "active", true, false,
		time.Now(), time.Now(),
	))
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT`)).WillReturnRows(sqlmock.NewRows(nil))
}

// ── TestOAuthTokenExchangeService_Exchange ──────────────────────────────────

func TestOAuthTokenExchangeService_Exchange(t *testing.T) {
	ctx := context.Background()

	t.Run("client not found", func(t *testing.T) {
		db, mock := newMockDB(t)
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT`)).WillReturnError(gorm.ErrRecordNotFound)

		svc := newOAuthTokenExchangeSvc(db, &mockUserRepo{}, &mockAuthEventService{})

		_, oerr := svc.Exchange(ctx, OAuthTokenExchangeRequestDTO{
			SubjectToken: "t",
		}, OAuthClientCredentials{ClientID: "unknown"})
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
			pq.StringArray{GrantTypeAuthorizationCode}, pq.StringArray{}, nil, nil,
			false, time.Now(), time.Now(),
		)
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT`)).WillReturnRows(rows)
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT`)).WillReturnRows(sqlmock.NewRows(nil))
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT`)).WillReturnRows(sqlmock.NewRows(nil))

		svc := newOAuthTokenExchangeSvc(db, &mockUserRepo{}, &mockAuthEventService{})

		_, oerr := svc.Exchange(ctx, OAuthTokenExchangeRequestDTO{
			SubjectToken: "t",
		}, OAuthClientCredentials{ClientID: "my-client"})
		require.NotNil(t, oerr)
		assert.Equal(t, "unauthorized_client", oerr.Code)
	})

	t.Run("subject_token not found (invalid)", func(t *testing.T) {
		db, mock := newMockDB(t)
		expectTokenExchangeClientLookup(mock)

		svc := newOAuthTokenExchangeSvc(db, &mockUserRepo{}, &mockAuthEventService{})

		_, oerr := svc.Exchange(ctx, OAuthTokenExchangeRequestDTO{
			SubjectToken: "garbage-token",
		}, OAuthClientCredentials{ClientID: "my-client"})
		require.NotNil(t, oerr)
		assert.Equal(t, "invalid_grant", oerr.Code)
		assert.Contains(t, oerr.Description, "invalid or expired")
	})

	t.Run("invalid requested_token_type", func(t *testing.T) {
		initTestJWTKeysService(t)
		db, mock := newMockDB(t)
		expectTokenExchangeClientLookup(mock)

		token, err := jwt.GenerateAccessToken("user-sub", "openid", "https://auth.example.com", "my-client", "my-client", "default-provider")
		require.NoError(t, err)

		svc := newOAuthTokenExchangeSvc(db, &mockUserRepo{}, &mockAuthEventService{})

		_, oerr := svc.Exchange(ctx, OAuthTokenExchangeRequestDTO{
			SubjectToken:     token,
			SubjectTokenType: "urn:ietf:params:oauth:token-type:access_token",
			Scope:            "profile",
		}, OAuthClientCredentials{ClientID: "my-client"})
		require.NotNil(t, oerr)
		assert.Equal(t, "invalid_scope", oerr.Code)
	})

	t.Run("scope from claims fallback", func(t *testing.T) {
		initTestJWTKeysService(t)
		origHost := config.AppPublicHostname
		config.AppPublicHostname = "https://auth.example.com"
		defer func() { config.AppPublicHostname = origHost }()

		db, mock := newMockDB(t)
		expectTokenExchangeClientLookup(mock)

		token, err := jwt.GenerateAccessToken("user-sub", "openid profile", "https://auth.example.com", "my-client", "my-client", "default-provider")
		require.NoError(t, err)

		svc := newOAuthTokenExchangeSvc(db, &mockUserRepo{}, &mockAuthEventService{})

		result, oerr := svc.Exchange(ctx, OAuthTokenExchangeRequestDTO{
			SubjectToken:     token,
			SubjectTokenType: "urn:ietf:params:oauth:token-type:access_token",
		}, OAuthClientCredentials{ClientID: "my-client"})
		require.Nil(t, oerr)
		require.NotNil(t, result)
		assert.Equal(t, "openid profile", result.Scope)
	})

	t.Run("success", func(t *testing.T) {
		initTestJWTKeysService(t)
		origHost := config.AppPublicHostname
		config.AppPublicHostname = "https://auth.example.com"
		defer func() { config.AppPublicHostname = origHost }()

		db, mock := newMockDB(t)
		expectTokenExchangeClientLookup(mock)

		token, err := jwt.GenerateAccessToken("user-sub", "openid profile", "https://auth.example.com", "my-client", "my-client", "default-provider")
		require.NoError(t, err)

		svc := newOAuthTokenExchangeSvc(db, &mockUserRepo{}, &mockAuthEventService{})

		result, oerr := svc.Exchange(ctx, OAuthTokenExchangeRequestDTO{
			SubjectToken:     token,
			SubjectTokenType: "urn:ietf:params:oauth:token-type:access_token",
		}, OAuthClientCredentials{ClientID: "my-client"})
		require.Nil(t, oerr)
		require.NotNil(t, result)
		assert.NotEmpty(t, result.AccessToken)
		assert.Equal(t, "Bearer", result.TokenType)
		assert.Equal(t, "urn:ietf:params:oauth:token-type:access_token", result.IssuedTokenType)
		assert.Equal(t, "openid profile", result.Scope)
	})

	t.Run("nil Domain, nil Identifier, nil IdentityProvider", func(t *testing.T) {
		initTestJWTKeysService(t)
		origHost := config.AppPublicHostname
		config.AppPublicHostname = "https://auth.example.com"
		defer func() { config.AppPublicHostname = origHost }()

		db, mock := newMockDB(t)
		rows := sqlmock.NewRows([]string{
			"client_id", "client_uuid", "tenant_id", "identity_provider_id", "name", "display_name",
			"client_type", "domain", "identifier", "secret", "status",
			"is_default", "is_system", "token_endpoint_auth_method",
			"grant_types", "response_types", "access_token_ttl", "refresh_token_ttl",
			"require_consent", "created_at", "updated_at",
		}).AddRow(
			10, uuid.New(), 1, int64(0), "test-client", "Test Client",
			"spa", nil, nil, nil, "active",
			false, false, "none",
			pq.StringArray{GrantTypeTokenExchange}, pq.StringArray{}, nil, nil,
			false, time.Now(), time.Now(),
		)
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT`)).WillReturnRows(rows)
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT`)).WillReturnRows(sqlmock.NewRows(nil))

		token, err := jwt.GenerateAccessToken("user-sub", "openid", "https://auth.example.com", "my-client", "my-client", "default-provider")
		require.NoError(t, err)

		svc := newOAuthTokenExchangeSvc(db, &mockUserRepo{}, &mockAuthEventService{})

		_, oerr := svc.Exchange(ctx, OAuthTokenExchangeRequestDTO{
			SubjectToken:     token,
			SubjectTokenType: "urn:ietf:params:oauth:token-type:access_token",
		}, OAuthClientCredentials{ClientID: "my-client"})
		require.NotNil(t, oerr)
		assert.Equal(t, "server_error", oerr.Code)
	})

	t.Run("non-nil IdentityProvider", func(t *testing.T) {
		initTestJWTKeysService(t)
		origHost := config.AppPublicHostname
		config.AppPublicHostname = "https://auth.example.com"
		defer func() { config.AppPublicHostname = origHost }()

		db, mock := newMockDB(t)
		expectTokenExchangeClientLookup(mock)

		token, err := jwt.GenerateAccessToken("user-sub", "openid", "https://auth.example.com", "my-client", "my-client", "default-provider")
		require.NoError(t, err)

		svc := newOAuthTokenExchangeSvc(db, &mockUserRepo{}, &mockAuthEventService{})

		result, oerr := svc.Exchange(ctx, OAuthTokenExchangeRequestDTO{
			SubjectToken:     token,
			SubjectTokenType: "urn:ietf:params:oauth:token-type:access_token",
		}, OAuthClientCredentials{ClientID: "my-client"})
		require.Nil(t, oerr)
		require.NotNil(t, result)
		assert.NotEmpty(t, result.AccessToken)
	})
}

// ── TestAmrClaimValues ──────────────────────────────────────────────────────

func TestAmrClaimValues(t *testing.T) {
	t.Run("non-array input returns nil", func(t *testing.T) {
		assert.Nil(t, amrClaimValues("not-an-array"))
	})

	t.Run("nil input returns nil", func(t *testing.T) {
		assert.Nil(t, amrClaimValues(nil))
	})

	t.Run("empty array returns empty slice", func(t *testing.T) {
		assert.Equal(t, []string{}, amrClaimValues([]any{}))
	})

	t.Run("array with string values", func(t *testing.T) {
		result := amrClaimValues([]any{"pwd", "mfa", "otp"})
		assert.Equal(t, []string{"pwd", "mfa", "otp"}, result)
	})

	t.Run("array with mixed types filters non-strings", func(t *testing.T) {
		result := amrClaimValues([]any{"pwd", 42, "mfa", true, "otp"})
		assert.Equal(t, []string{"pwd", "mfa", "otp"}, result)
	})

	t.Run("array with empty strings filtered out", func(t *testing.T) {
		result := amrClaimValues([]any{"pwd", "", "mfa"})
		assert.Equal(t, []string{"pwd", "mfa"}, result)
	})

	t.Run("map input returns nil", func(t *testing.T) {
		assert.Nil(t, amrClaimValues(map[string]any{}))
	})
}

// ── TestOAuthTokenExchangeService_Exchange_ScopeValidation ──────────────────

func TestOAuthTokenExchangeService_Exchange_ScopeValidation(t *testing.T) {
	ctx := context.Background()

	t.Run("scope not allowed by client AllowedScopes", func(t *testing.T) {
		initTestJWTKeysService(t)
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
			pq.StringArray{GrantTypeTokenExchange}, pq.StringArray{}, nil, nil,
			false, pq.StringArray{"openid"}, time.Now(), time.Now(),
		)
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT`)).WillReturnRows(rows)
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT`)).WillReturnRows(sqlmock.NewRows([]string{
			"identity_provider_id", "identity_provider_uuid", "tenant_id",
			"name", "display_name", "provider", "provider_type",
			"identifier", "config", "status", "is_default", "is_system",
			"created_at", "updated_at",
		}).AddRow(
			100, uuid.New(), 1,
			"default", "Default Provider", "local", "local",
			"default-provider", `{}`, "active", true, false,
			time.Now(), time.Now(),
		))
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT`)).WillReturnRows(sqlmock.NewRows(nil))

		token, err := jwt.GenerateAccessToken("user-sub", "openid admin", "https://auth.example.com", "my-client", "my-client", "default-provider")
		require.NoError(t, err)

		svc := newOAuthTokenExchangeSvc(db, &mockUserRepo{}, &mockAuthEventService{})

		_, oerr := svc.Exchange(ctx, OAuthTokenExchangeRequestDTO{
			SubjectToken:     token,
			SubjectTokenType: "urn:ietf:params:oauth:token-type:access_token",
			Scope:            "admin",
		}, OAuthClientCredentials{ClientID: "my-client"})
		require.NotNil(t, oerr)
		assert.Equal(t, "invalid_scope", oerr.Code)
		assert.Contains(t, oerr.Description, "not allowed")
	})

	t.Run("scope from claims is validated against AllowedScopes", func(t *testing.T) {
		initTestJWTKeysService(t)
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
			pq.StringArray{GrantTypeTokenExchange}, pq.StringArray{}, nil, nil,
			false, pq.StringArray{"openid"}, time.Now(), time.Now(),
		)
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT`)).WillReturnRows(rows)
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT`)).WillReturnRows(sqlmock.NewRows([]string{
			"identity_provider_id", "identity_provider_uuid", "tenant_id",
			"name", "display_name", "provider", "provider_type",
			"identifier", "config", "status", "is_default", "is_system",
			"created_at", "updated_at",
		}).AddRow(
			100, uuid.New(), 1,
			"default", "Default Provider", "local", "local",
			"default-provider", `{}`, "active", true, false,
			time.Now(), time.Now(),
		))
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT`)).WillReturnRows(sqlmock.NewRows(nil))

		token, err := jwt.GenerateAccessToken("user-sub", "openid profile", "https://auth.example.com", "my-client", "my-client", "default-provider")
		require.NoError(t, err)

		svc := newOAuthTokenExchangeSvc(db, &mockUserRepo{}, &mockAuthEventService{})

		_, oerr := svc.Exchange(ctx, OAuthTokenExchangeRequestDTO{
			SubjectToken:     token,
			SubjectTokenType: "urn:ietf:params:oauth:token-type:access_token",
			Scope:            "profile",
		}, OAuthClientCredentials{ClientID: "my-client"})
		require.NotNil(t, oerr)
		assert.Equal(t, "invalid_scope", oerr.Code)
	})
}

// ── TestOAuthTokenExchangeService_Exchange_ACR ──────────────────────────────

func TestOAuthTokenExchangeService_Exchange_ACR(t *testing.T) {
	initTestJWTKeysService(t)
	origHost := config.AppPublicHostname
	config.AppPublicHostname = "https://auth.example.com"
	defer func() { config.AppPublicHostname = origHost }()

	ctx := context.Background()
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
		pq.StringArray{GrantTypeTokenExchange}, pq.StringArray{}, nil, nil,
		false, pq.StringArray{"openid", "profile"}, time.Now(), time.Now(),
	)
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT`)).WillReturnRows(rows)
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT`)).WillReturnRows(sqlmock.NewRows([]string{
		"identity_provider_id", "identity_provider_uuid", "tenant_id",
		"name", "display_name", "provider", "provider_type",
		"identifier", "config", "status", "is_default", "is_system",
		"created_at", "updated_at",
	}).AddRow(
		100, uuid.New(), 1,
		"default", "Default Provider", "local", "local",
		"default-provider", `{}`, "active", true, false,
		time.Now(), time.Now(),
	))
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT`)).WillReturnRows(sqlmock.NewRows(nil))

	opts := &jwt.AccessTokenOptions{ACR: jwt.ACRLevel1}
	token, err := jwt.GenerateAccessTokenWithOptionsContext(ctx, "user-sub", "openid profile", "https://auth.example.com", "my-client", "my-client", "default-provider", opts)
	require.NoError(t, err)

	svc := newOAuthTokenExchangeSvc(db, &mockUserRepo{}, &mockAuthEventService{})

	result, oerr := svc.Exchange(ctx, OAuthTokenExchangeRequestDTO{
		SubjectToken:     token,
		SubjectTokenType: "urn:ietf:params:oauth:token-type:access_token",
		Scope:            "openid",
	}, OAuthClientCredentials{ClientID: "my-client"})
	require.Nil(t, oerr)
	require.NotNil(t, result)
	assert.NotEmpty(t, result.AccessToken)
}

// ── TestOAuthTokenExchangeService_Exchange_UnsupportedTokenType ──────────────

func TestOAuthTokenExchangeService_Exchange_UnsupportedTokenType(t *testing.T) {
	initTestJWTKeysService(t)
	ctx := context.Background()
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
		pq.StringArray{GrantTypeTokenExchange}, pq.StringArray{}, nil, nil,
		false, pq.StringArray{"openid", "profile"}, time.Now(), time.Now(),
	)
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT`)).WillReturnRows(rows)
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT`)).WillReturnRows(sqlmock.NewRows([]string{
		"identity_provider_id", "identity_provider_uuid", "tenant_id",
		"name", "display_name", "provider", "provider_type",
		"identifier", "config", "status", "is_default", "is_system",
		"created_at", "updated_at",
	}).AddRow(
		100, uuid.New(), 1,
		"default", "Default Provider", "local", "local",
		"default-provider", `{}`, "active", true, false,
		time.Now(), time.Now(),
	))
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT`)).WillReturnRows(sqlmock.NewRows(nil))

	token, err := jwt.GenerateAccessToken("user-sub", "openid", "https://auth.example.com", "my-client", "my-client", "default-provider")
	require.NoError(t, err)

	svc := newOAuthTokenExchangeSvc(db, &mockUserRepo{}, &mockAuthEventService{})

	_, oerr := svc.Exchange(ctx, OAuthTokenExchangeRequestDTO{
		SubjectToken:       token,
		SubjectTokenType:   "urn:ietf:params:oauth:token-type:access_token",
		RequestedTokenType: "urn:ietf:params:oauth:token-type:refresh_token",
	}, OAuthClientCredentials{ClientID: "my-client"})
	require.NotNil(t, oerr)
	assert.Equal(t, "invalid_request", oerr.Code)
	assert.Contains(t, oerr.Description, "only access_token")
}
