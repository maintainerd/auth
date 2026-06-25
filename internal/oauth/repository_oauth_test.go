package oauth

import (
	"database/sql/driver"
	"errors"
	"testing"
	"time"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func newOAuthRepoMockDB(t *testing.T) (*gorm.DB, sqlmock.Sqlmock) {
	t.Helper()
	sqlDB, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	require.NoError(t, err)
	db, err := gorm.Open(postgres.New(postgres.Config{Conn: sqlDB}), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	require.NoError(t, err)
	return db, mock
}

func expectOAuthSelect(mock sqlmock.Sqlmock, table string) *sqlmock.ExpectedQuery {
	return mock.ExpectQuery(`SELECT .* FROM "` + table + `".*`)
}

func expectOAuthUpdate(mock sqlmock.Sqlmock, table string) *sqlmock.ExpectedExec {
	return mock.ExpectExec(`UPDATE "` + table + `".*`)
}

func expectOAuthDelete(mock sqlmock.Sqlmock, table string) *sqlmock.ExpectedExec {
	return mock.ExpectExec(`DELETE FROM "` + table + `".*`)
}

func expectOAuthCount(mock sqlmock.Sqlmock, table string) *sqlmock.ExpectedQuery {
	return mock.ExpectQuery(`SELECT count\(\*\) FROM "` + table + `".*`)
}

func oauthAuthCodeRows(values ...driver.Value) *sqlmock.Rows {
	if len(values) == 0 {
		values = []driver.Value{int64(1), testResourceUUID.String(), "hash", int64(10), int64(1), int64(1), "https://example.com/cb", "openid", nil, nil, "challenge", "S256", false, nil, time.Now().Add(time.Minute), time.Now()}
	}
	return sqlmock.NewRows([]string{"oauth_authorization_code_id", "oauth_authorization_code_uuid", "code_hash", "client_id", "user_id", "tenant_id", "redirect_uri", "scope", "state", "nonce", "code_challenge", "code_challenge_method", "is_used", "used_at", "expires_at", "created_at"}).AddRow(values...)
}

func oauthCIBARows(values ...driver.Value) *sqlmock.Rows {
	if len(values) == 0 {
		values = []driver.Value{int64(1), testResourceUUID.String(), "hash", int64(10), int64(1), nil, "openid", nil, "", []byte(`[]`), CIBAStatusPending, 5, nil, nil, time.Now().Add(time.Minute), time.Now()}
	}
	return sqlmock.NewRows([]string{"oauth_ciba_request_id", "oauth_ciba_request_uuid", "auth_req_id_hash", "client_id", "tenant_id", "user_id", "scope", "binding_message", "auth_acr", "auth_amr", "status", "interval", "last_poll_at", "notification_sent_at", "expires_at", "created_at"}).AddRow(values...)
}

func oauthConsentChallengeRows(values ...driver.Value) *sqlmock.Rows {
	if len(values) == 0 {
		values = []driver.Value{int64(1), testResourceUUID.String(), int64(10), int64(1), int64(1), "https://example.com/cb", "openid", nil, nil, "challenge", "S256", "code", time.Now().Add(time.Minute), time.Now()}
	}
	return sqlmock.NewRows([]string{"oauth_consent_challenge_id", "oauth_consent_challenge_uuid", "client_id", "user_id", "tenant_id", "redirect_uri", "scope", "state", "nonce", "code_challenge", "code_challenge_method", "response_type", "expires_at", "created_at"}).AddRow(values...)
}

func oauthConsentGrantRows(values ...driver.Value) *sqlmock.Rows {
	if len(values) == 0 {
		values = []driver.Value{int64(1), testResourceUUID.String(), int64(1), int64(10), int64(1), "openid", time.Now(), time.Now()}
	}
	return sqlmock.NewRows([]string{"oauth_consent_grant_id", "oauth_consent_grant_uuid", "user_id", "client_id", "tenant_id", "scopes", "created_at", "updated_at"}).AddRow(values...)
}

func oauthDeviceRows(values ...driver.Value) *sqlmock.Rows {
	if len(values) == 0 {
		values = []driver.Value{int64(1), testResourceUUID.String(), "device-hash", "ABCD-123", int64(10), int64(1), "openid", nil, "", []byte(`[]`), DeviceCodeStatusPending, 5, nil, time.Now().Add(time.Minute), time.Now()}
	}
	return sqlmock.NewRows([]string{"oauth_device_code_id", "oauth_device_code_uuid", "device_code_hash", "user_code", "client_id", "tenant_id", "scope", "user_id", "auth_acr", "auth_amr", "status", "interval", "last_poll_at", "expires_at", "created_at"}).AddRow(values...)
}

func oauthPARRows(values ...driver.Value) *sqlmock.Rows {
	if len(values) == 0 {
		values = []driver.Value{int64(1), testResourceUUID.String(), "request-hash", int64(10), int64(1), "code", "https://example.com/cb", "openid", nil, nil, "challenge", "S256", false, time.Now().Add(time.Minute), time.Now()}
	}
	return sqlmock.NewRows([]string{"oauth_par_request_id", "oauth_par_request_uuid", "request_uri_hash", "client_id", "tenant_id", "response_type", "redirect_uri", "scope", "state", "nonce", "code_challenge", "code_challenge_method", "is_used", "expires_at", "created_at"}).AddRow(values...)
}

func oauthRefreshRows(values ...driver.Value) *sqlmock.Rows {
	if len(values) == 0 {
		values = []driver.Value{int64(1), testResourceUUID.String(), "token-hash", testResourceUUID.String(), int64(10), int64(1), int64(1), "openid", false, nil, time.Now().Add(time.Hour), nil, time.Now()}
	}
	return sqlmock.NewRows([]string{"oauth_refresh_token_id", "oauth_refresh_token_uuid", "token_hash", "family_id", "client_id", "user_id", "tenant_id", "scope", "is_revoked", "revoked_at", "expires_at", "last_used_at", "created_at"}).AddRow(values...)
}

func oauthClientRows() *sqlmock.Rows {
	return sqlmock.NewRows([]string{"client_id", "client_uuid", "tenant_id", "identity_provider_id", "name", "status", "created_at", "updated_at"}).
		AddRow(int64(10), testResourceUUID.String(), int64(1), int64(1), "client", "active", time.Now(), time.Now())
}

func oauthTenantRows() *sqlmock.Rows {
	return sqlmock.NewRows([]string{"tenant_id", "tenant_uuid", "name", "identifier", "status", "created_at", "updated_at"}).
		AddRow(int64(1), testResourceUUID.String(), "Test Tenant", "test-tenant", "active", time.Now(), time.Now())
}

func expectClientPreloads(mock sqlmock.Sqlmock) {
	expectOAuthSelect(mock, "clients").WillReturnRows(oauthClientRows())
	expectOAuthSelect(mock, "tenants").WillReturnRows(oauthTenantRows())
}

func assertOAuthRepoExpectations(t *testing.T, mock sqlmock.Sqlmock) {
	t.Helper()
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestOAuthAuthorizationCodeRepository(t *testing.T) {
	db, mock := newOAuthRepoMockDB(t)
	repo := NewOAuthAuthorizationCodeRepository(db)
	require.NotNil(t, repo)
	assert.NotNil(t, repo.(*oauthAuthorizationCodeRepository).WithTx(db))
	expectOAuthSelect(mock, "oauth_authorization_codes").WillReturnRows(oauthAuthCodeRows())
	expectClientPreloads(mock)
	expectOAuthSelect(mock, "oauth_authorization_codes").WillReturnError(gorm.ErrRecordNotFound)
	expectOAuthSelect(mock, "oauth_authorization_codes").WillReturnError(errors.New("db down"))
	mock.ExpectBegin()
	expectOAuthUpdate(mock, "oauth_authorization_codes").WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()
	mock.ExpectBegin()
	expectOAuthDelete(mock, "oauth_authorization_codes").WillReturnResult(sqlmock.NewResult(0, 2))
	mock.ExpectCommit()

	got, err := repo.FindByCodeHash("hash")
	require.NoError(t, err)
	require.NotNil(t, got)
	got, err = repo.FindByCodeHash("hash")
	require.NoError(t, err)
	assert.Nil(t, got)
	got, err = repo.FindByCodeHash("hash")
	require.Error(t, err)
	assert.Nil(t, got)
	require.NoError(t, repo.MarkUsed(1))
	deleted, err := repo.DeleteExpired(time.Now())
	require.NoError(t, err)
	assert.Equal(t, int64(2), deleted)
	assertOAuthRepoExpectations(t, mock)
}

func TestOAuthCIBARequestRepository(t *testing.T) {
	db, mock := newOAuthRepoMockDB(t)
	repo := NewOAuthCIBARequestRepository(db)
	require.NotNil(t, repo)
	assert.NotNil(t, repo.(*oauthCIBARequestRepository).WithTx(db))

	// ── expectations ──
	expectOAuthSelect(mock, "oauth_ciba_requests").WillReturnRows(oauthCIBARows())
	expectClientPreloads(mock)
	expectOAuthSelect(mock, "oauth_ciba_requests").WillReturnError(gorm.ErrRecordNotFound)
	expectOAuthSelect(mock, "oauth_ciba_requests").WillReturnError(errors.New("db down"))

	// UpdateApprovalContext error path
	mock.ExpectBegin()
	expectOAuthUpdate(mock, "oauth_ciba_requests").WillReturnError(errors.New("db down"))
	mock.ExpectRollback()

	for i := 0; i < 5; i++ {
		mock.ExpectBegin()
		expectOAuthUpdate(mock, "oauth_ciba_requests").WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectCommit()
	}
	mock.ExpectBegin()
	expectOAuthDelete(mock, "oauth_ciba_requests").WillReturnResult(sqlmock.NewResult(0, 3))
	mock.ExpectCommit()

	// ── calls ──
	got, err := repo.FindByAuthReqIDHash("hash")
	require.NoError(t, err)
	require.NotNil(t, got)
	got, err = repo.FindByAuthReqIDHash("hash")
	require.NoError(t, err)
	assert.Nil(t, got)
	got, err = repo.FindByAuthReqIDHash("hash")
	require.Error(t, err)
	assert.Nil(t, got)

	err = repo.UpdateApprovalContext(1, 99, "urn:mace:incommon:iap:silver", []string{"pwd"})
	require.Error(t, err)

	require.NoError(t, repo.UpdateStatus(1, CIBAStatusDenied))
	require.NoError(t, repo.UpdateApproval(1, 99))
	require.NoError(t, repo.UpdateApprovalContext(1, 99, "urn:mace:incommon:iap:silver", []string{"pwd"}))
	require.NoError(t, repo.UpdateLastPollAt(1))
	require.NoError(t, repo.MarkNotificationSent(1))
	deleted, err := repo.DeleteExpired(time.Now())
	require.NoError(t, err)
	assert.Equal(t, int64(3), deleted)
	assertOAuthRepoExpectations(t, mock)
}

func TestOAuthConsentChallengeRepository(t *testing.T) {
	db, mock := newOAuthRepoMockDB(t)
	repo := NewOAuthConsentChallengeRepository(db)
	require.NotNil(t, repo)
	assert.NotNil(t, repo.(*oauthConsentChallengeRepository).WithTx(db))
	expectOAuthSelect(mock, "oauth_consent_challenges").WillReturnRows(oauthConsentChallengeRows())
	expectClientPreloads(mock)
	expectOAuthSelect(mock, "oauth_consent_challenges").WillReturnError(gorm.ErrRecordNotFound)
	expectOAuthSelect(mock, "oauth_consent_challenges").WillReturnError(errors.New("db down"))
	for i := 0; i < 2; i++ {
		mock.ExpectBegin()
		expectOAuthDelete(mock, "oauth_consent_challenges").WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectCommit()
	}

	got, err := repo.FindChallengeByUUID(testResourceUUID)
	require.NoError(t, err)
	require.NotNil(t, got)
	got, err = repo.FindChallengeByUUID(testResourceUUID)
	require.NoError(t, err)
	assert.Nil(t, got)
	got, err = repo.FindChallengeByUUID(testResourceUUID)
	require.Error(t, err)
	assert.Nil(t, got)
	require.NoError(t, repo.DeleteChallengeByUUID(testResourceUUID))
	deleted, err := repo.DeleteExpired(time.Now())
	require.NoError(t, err)
	assert.Equal(t, int64(1), deleted)
	assertOAuthRepoExpectations(t, mock)
}

func TestOAuthConsentGrantRepository(t *testing.T) {
	db, mock := newOAuthRepoMockDB(t)
	repo := NewOAuthConsentGrantRepository(db)
	require.NotNil(t, repo)
	assert.NotNil(t, repo.(*oauthConsentGrantRepository).WithTx(db))
	expectOAuthSelect(mock, "oauth_consent_grants").WillReturnRows(oauthConsentGrantRows())
	expectOAuthSelect(mock, "oauth_consent_grants").WillReturnError(gorm.ErrRecordNotFound)
	expectOAuthSelect(mock, "oauth_consent_grants").WillReturnError(errors.New("db down"))

	// Upsert error path — FindByUserAndClient returns a non-ErrNotFound error.
	expectOAuthSelect(mock, "oauth_consent_grants").WillReturnError(errors.New("db down"))

	// Upsert Save error — FindByUserAndClient returns existing, Save fails.
	expectOAuthSelect(mock, "oauth_consent_grants").WillReturnRows(oauthConsentGrantRows())
	mock.ExpectBegin()
	expectOAuthUpdate(mock, "oauth_consent_grants").WillReturnError(errors.New("db down"))
	mock.ExpectRollback()

	expectOAuthSelect(mock, "oauth_consent_grants").WillReturnRows(oauthConsentGrantRows())
	mock.ExpectBegin()
	mock.ExpectExec(`UPDATE "oauth_consent_grants"|INSERT INTO "oauth_consent_grants"`).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()
	expectOAuthSelect(mock, "oauth_consent_grants").WillReturnError(gorm.ErrRecordNotFound)
	mock.ExpectBegin()
	mock.ExpectQuery(`INSERT INTO "oauth_consent_grants".*RETURNING`).WillReturnRows(
		sqlmock.NewRows([]string{"oauth_consent_grant_id"}).AddRow(int64(2)),
	)
	mock.ExpectCommit()
	mock.ExpectBegin()
	expectOAuthDelete(mock, "oauth_consent_grants").WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()
	expectOAuthSelect(mock, "oauth_consent_grants").WillReturnRows(oauthConsentGrantRows())
	expectOAuthSelect(mock, "clients").WillReturnRows(oauthClientRows())

	got, err := repo.FindByUserAndClient(1, 10)
	require.NoError(t, err)
	require.NotNil(t, got)
	got, err = repo.FindByUserAndClient(1, 10)
	require.NoError(t, err)
	assert.Nil(t, got)
	got, err = repo.FindByUserAndClient(1, 10)
	require.Error(t, err)
	assert.Nil(t, got)

	_, err = repo.Upsert(&OAuthConsentGrant{UserID: 3, ClientID: 10, TenantID: 1, Scopes: "openid"})
	require.Error(t, err)

	_, err = repo.Upsert(&OAuthConsentGrant{UserID: 1, ClientID: 10, TenantID: 1, Scopes: "openid email"})
	require.Error(t, err) // Save error

	updated, err := repo.Upsert(&OAuthConsentGrant{UserID: 1, ClientID: 10, TenantID: 1, Scopes: "openid email"})
	require.NoError(t, err)
	require.NotNil(t, updated)
	created, err := repo.Upsert(&OAuthConsentGrant{UserID: 2, ClientID: 10, TenantID: 1, Scopes: "openid"})
	require.NoError(t, err)
	require.NotNil(t, created)
	require.NoError(t, repo.DeleteByUserAndClient(1, 10))
	grants, err := repo.FindByUserID(1)
	require.NoError(t, err)
	assert.Len(t, grants, 1)
	assertOAuthRepoExpectations(t, mock)
}

func TestOAuthDeviceCodeRepository(t *testing.T) {
	db, mock := newOAuthRepoMockDB(t)
	repo := NewOAuthDeviceCodeRepository(db)
	require.NotNil(t, repo)
	assert.NotNil(t, repo.(*oauthDeviceCodeRepository).WithTx(db))
	expectOAuthSelect(mock, "oauth_device_codes").WillReturnRows(oauthDeviceRows())
	expectClientPreloads(mock)
	expectOAuthSelect(mock, "oauth_device_codes").WillReturnError(gorm.ErrRecordNotFound)
	expectOAuthSelect(mock, "oauth_device_codes").WillReturnError(errors.New("db down"))
	expectOAuthSelect(mock, "oauth_device_codes").WillReturnRows(oauthDeviceRows())
	expectOAuthSelect(mock, "clients").WillReturnRows(oauthClientRows())
	expectOAuthSelect(mock, "oauth_device_codes").WillReturnError(gorm.ErrRecordNotFound)
	expectOAuthSelect(mock, "oauth_device_codes").WillReturnError(errors.New("db down"))

	// UpdateApproval error path — DB returns an error.
	mock.ExpectBegin()
	expectOAuthUpdate(mock, "oauth_device_codes").WillReturnError(errors.New("db down"))
	mock.ExpectRollback()

	for i := 0; i < 3; i++ {
		mock.ExpectBegin()
		expectOAuthUpdate(mock, "oauth_device_codes").WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectCommit()
	}
	mock.ExpectBegin()
	expectOAuthDelete(mock, "oauth_device_codes").WillReturnResult(sqlmock.NewResult(0, 2))
	mock.ExpectCommit()

	got, err := repo.FindByDeviceCodeHash("hash")
	require.NoError(t, err)
	require.NotNil(t, got)
	got, err = repo.FindByDeviceCodeHash("hash")
	require.NoError(t, err)
	assert.Nil(t, got)
	got, err = repo.FindByDeviceCodeHash("hash")
	require.Error(t, err)
	assert.Nil(t, got)
	byUserCode, err := repo.FindByUserCode("ABCD-123")
	require.NoError(t, err)
	require.NotNil(t, byUserCode)
	byUserCode, err = repo.FindByUserCode("ABCD-123")
	require.NoError(t, err)
	assert.Nil(t, byUserCode)
	byUserCode, err = repo.FindByUserCode("ABCD-123")
	require.Error(t, err)
	assert.Nil(t, byUserCode)
	userID := int64(1)
	err = repo.UpdateApproval(1, userID, "2", []string{"pwd"})
	require.Error(t, err)
	require.NoError(t, repo.UpdateStatus(1, DeviceCodeStatusApproved, &userID))
	require.NoError(t, repo.UpdateApproval(1, userID, "2", []string{"pwd"}))
	require.NoError(t, repo.UpdateLastPollAt(1))
	deleted, err := repo.DeleteExpired(time.Now())
	require.NoError(t, err)
	assert.Equal(t, int64(2), deleted)
	assertOAuthRepoExpectations(t, mock)
}

func TestOAuthPARRequestRepository(t *testing.T) {
	db, mock := newOAuthRepoMockDB(t)
	repo := NewOAuthPARRequestRepository(db)
	require.NotNil(t, repo)
	assert.NotNil(t, repo.(*oauthPARRequestRepository).WithTx(db))
	expectOAuthSelect(mock, "oauth_par_requests").WillReturnRows(oauthPARRows())
	expectOAuthSelect(mock, "clients").WillReturnRows(oauthClientRows())
	expectOAuthSelect(mock, "client_uris").WillReturnRows(sqlmock.NewRows([]string{"client_uri_id", "client_id", "uri", "type"}))
	expectOAuthSelect(mock, "oauth_par_requests").WillReturnError(gorm.ErrRecordNotFound)
	expectOAuthSelect(mock, "oauth_par_requests").WillReturnError(errors.New("db down"))
	mock.ExpectBegin()
	expectOAuthUpdate(mock, "oauth_par_requests").WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()
	mock.ExpectBegin()
	expectOAuthDelete(mock, "oauth_par_requests").WillReturnResult(sqlmock.NewResult(0, 2))
	mock.ExpectCommit()

	got, err := repo.FindByRequestURIHash("hash")
	require.NoError(t, err)
	require.NotNil(t, got)
	got, err = repo.FindByRequestURIHash("hash")
	require.NoError(t, err)
	assert.Nil(t, got)
	got, err = repo.FindByRequestURIHash("hash")
	require.Error(t, err)
	assert.Nil(t, got)
	require.NoError(t, repo.MarkUsed(1))
	deleted, err := repo.DeleteExpired(time.Now())
	require.NoError(t, err)
	assert.Equal(t, int64(2), deleted)
	assertOAuthRepoExpectations(t, mock)
}

func TestOAuthRefreshTokenRepository(t *testing.T) {
	db, mock := newOAuthRepoMockDB(t)
	repo := NewOAuthRefreshTokenRepository(db)
	require.NotNil(t, repo)
	assert.NotNil(t, repo.(*oauthRefreshTokenRepository).WithTx(db))
	expectOAuthSelect(mock, "oauth_refresh_tokens").WillReturnRows(oauthRefreshRows())
	expectClientPreloads(mock)
	expectOAuthSelect(mock, "oauth_refresh_tokens").WillReturnError(gorm.ErrRecordNotFound)
	expectOAuthSelect(mock, "oauth_refresh_tokens").WillReturnError(errors.New("db down"))
	expectOAuthSelect(mock, "oauth_refresh_tokens").WillReturnRows(oauthRefreshRows())
	for i := 0; i < 5; i++ {
		mock.ExpectBegin()
		expectOAuthUpdate(mock, "oauth_refresh_tokens").WillReturnResult(sqlmock.NewResult(0, 2))
		mock.ExpectCommit()
	}
	mock.ExpectBegin()
	expectOAuthDelete(mock, "oauth_refresh_tokens").WillReturnResult(sqlmock.NewResult(0, 3))
	mock.ExpectCommit()
	expectOAuthCount(mock, "oauth_refresh_tokens").WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(4))

	got, err := repo.FindByTokenHash("hash")
	require.NoError(t, err)
	require.NotNil(t, got)
	got, err = repo.FindByTokenHash("hash")
	require.NoError(t, err)
	assert.Nil(t, got)
	got, err = repo.FindByTokenHash("hash")
	require.Error(t, err)
	assert.Nil(t, got)
	active, err := repo.FindActiveByUserAndClient(1, 10)
	require.NoError(t, err)
	assert.Len(t, active, 1)
	require.NoError(t, repo.RevokeByID(1))
	n, err := repo.RevokeByFamily(testResourceUUID)
	require.NoError(t, err)
	assert.Equal(t, int64(2), n)
	n, err = repo.RevokeByUserAndClient(1, 10)
	require.NoError(t, err)
	assert.Equal(t, int64(2), n)
	n, err = repo.RevokeByUserID(1)
	require.NoError(t, err)
	assert.Equal(t, int64(2), n)
	require.NoError(t, repo.UpdateLastUsed(1))
	deleted, err := repo.DeleteExpired(time.Now())
	require.NoError(t, err)
	assert.Equal(t, int64(3), deleted)
	count, err := repo.CountByUserAndClient(1, 10)
	require.NoError(t, err)
	assert.Equal(t, int64(4), count)
	assertOAuthRepoExpectations(t, mock)
}
