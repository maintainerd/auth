package oauth

import (
	"errors"
	"testing"
	"time"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func brokerSessionRows() *sqlmock.Rows {
	now := time.Now()
	return sqlmock.NewRows([]string{
		"oauth_broker_session_id", "oauth_broker_session_uuid", "tenant_id", "client_id",
		"identity_provider_id", "app_redirect_uri", "app_state", "app_scope", "app_nonce",
		"app_code_challenge", "app_code_challenge_method", "idp_state", "idp_pkce_verifier",
		"idp_nonce", "expires_at", "consumed_at", "created_at",
	}).AddRow(
		int64(1), testResourceUUID.String(), int64(1), int64(10),
		int64(100), "https://app.example.com/cb", "app-state", "openid", "app-nonce",
		"app-challenge", "S256", "idp-state-123", "verifier-xyz",
		"idp-nonce", now.Add(10*time.Minute), nil, now,
	)
}

func TestOAuthBrokerSessionRepository(t *testing.T) {
	db, mock := newOAuthRepoMockDB(t)
	repo := NewOAuthBrokerSessionRepository(db)
	require.NotNil(t, repo)
	assert.NotNil(t, repo.(*oauthBrokerSessionRepository).WithTx(db))

	expectOAuthSelect(mock, "oauth_broker_sessions").WillReturnRows(brokerSessionRows())
	expectOAuthSelect(mock, "oauth_broker_sessions").WillReturnError(gorm.ErrRecordNotFound)
	expectOAuthSelect(mock, "oauth_broker_sessions").WillReturnError(errors.New("db down"))
	mock.ExpectBegin()
	expectOAuthUpdate(mock, "oauth_broker_sessions").WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()
	mock.ExpectBegin()
	expectOAuthDelete(mock, "oauth_broker_sessions").WillReturnResult(sqlmock.NewResult(0, 2))
	mock.ExpectCommit()

	got, err := repo.FindByIdpState("idp-state-123")
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, "verifier-xyz", got.IdpPKCEVerifier)
	assert.False(t, got.IsConsumed())
	assert.False(t, got.IsExpired())

	got, err = repo.FindByIdpState("missing")
	require.NoError(t, err)
	assert.Nil(t, got)

	got, err = repo.FindByIdpState("err")
	require.Error(t, err)
	assert.Nil(t, got)

	require.NoError(t, repo.Consume(1, time.Now()))

	n, err := repo.DeleteExpired(time.Now())
	require.NoError(t, err)
	assert.Equal(t, int64(2), n)

	assertOAuthRepoExpectations(t, mock)
}
