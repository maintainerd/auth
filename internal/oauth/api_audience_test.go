package oauth

import (
	"regexp"
	"testing"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolveRequestedAudience(t *testing.T) {
	client := &Client{ClientID: 10, TenantID: 1}

	t.Run("no audience and no resource means the caller named nothing", func(t *testing.T) {
		db, _ := newMockDB(t)
		aud, oerr := resolveRequestedAudience(db, client, "", "")
		require.Nil(t, oerr)
		assert.Empty(t, aud)
	})

	t.Run("an API the client is granted resolves", func(t *testing.T) {
		db, mock := newMockDB(t)
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT count(*)`)).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

		aud, oerr := resolveRequestedAudience(db, client, "https://api.example.com", "")
		require.Nil(t, oerr)
		assert.Equal(t, "https://api.example.com", aud)
	})

	t.Run("an API the client is NOT granted is invalid_target", func(t *testing.T) {
		db, mock := newMockDB(t)
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT count(*)`)).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))

		_, oerr := resolveRequestedAudience(db, client, "https://api.example.com", "")
		require.NotNil(t, oerr)
		assert.Equal(t, "invalid_target", oerr.Code)
	})

	// Fails CLOSED: an audience that cannot be verified must never be minted onto
	// a token, because `aud` is what a resource server trusts.
	t.Run("a query error is invalid_target, not a pass", func(t *testing.T) {
		db, mock := newMockDB(t)
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT count(*)`)).WillReturnError(assert.AnError)

		_, oerr := resolveRequestedAudience(db, client, "https://api.example.com", "")
		require.NotNil(t, oerr)
		assert.Equal(t, "invalid_target", oerr.Code)
	})

	t.Run("a nil db is invalid_target", func(t *testing.T) {
		_, oerr := resolveRequestedAudience(nil, client, "https://api.example.com", "")
		require.NotNil(t, oerr)
		assert.Equal(t, "invalid_target", oerr.Code)
	})

	t.Run("resource must be an absolute URI without a fragment", func(t *testing.T) {
		db, _ := newMockDB(t)
		for _, bad := range []string{"not-a-uri", "https://api.example.com/#frag"} {
			_, oerr := resolveRequestedAudience(db, client, "", bad)
			require.NotNil(t, oerr, bad)
			assert.Equal(t, "invalid_target", oerr.Code)
		}
	})

	t.Run("audience and resource may not name different targets", func(t *testing.T) {
		db, _ := newMockDB(t)
		_, oerr := resolveRequestedAudience(db, client, "https://a.example.com", "https://b.example.com")
		require.NotNil(t, oerr)
		assert.Equal(t, "invalid_target", oerr.Code)
	})
}

// normalizeRequestedTarget is the single parser both token-exchange legs run on
// the caller's `audience`/`resource`. The keyless workload-identity leg reaches
// it in OAuthTokenExchangeHandler.Exchange before the federation exchanger sees a
// target, so the federation domain never parses a raw caller parameter itself.
func TestNormalizeRequestedTarget(t *testing.T) {
	t.Run("naming nothing is not an error", func(t *testing.T) {
		target, oerr := normalizeRequestedTarget("", "")
		require.Nil(t, oerr)
		assert.Empty(t, target)
	})

	t.Run("resource alone becomes the target", func(t *testing.T) {
		target, oerr := normalizeRequestedTarget("", " https://api.example.com ")
		require.Nil(t, oerr)
		assert.Equal(t, "https://api.example.com", target)
	})

	t.Run("audience and a matching resource collapse to one target", func(t *testing.T) {
		target, oerr := normalizeRequestedTarget("https://api.example.com", "https://api.example.com")
		require.Nil(t, oerr)
		assert.Equal(t, "https://api.example.com", target)
	})

	t.Run("conflicting targets are invalid_target", func(t *testing.T) {
		_, oerr := normalizeRequestedTarget("https://a.example.com", "https://b.example.com")
		require.NotNil(t, oerr)
		assert.Equal(t, "invalid_target", oerr.Code)
	})

	t.Run("a resource that is not an absolute fragment-free URI is invalid_target", func(t *testing.T) {
		for _, bad := range []string{"not-a-uri", "https://api.example.com/#frag"} {
			_, oerr := normalizeRequestedTarget("", bad)
			require.NotNil(t, oerr, bad)
			assert.Equal(t, "invalid_target", oerr.Code)
		}
	})
}

func TestVerifyClientAudience(t *testing.T) {
	t.Run("an API the client is granted verifies", func(t *testing.T) {
		db, mock := newMockDB(t)
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT count(*)`)).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

		assert.Nil(t, verifyClientAudience(db, 1, 10, "https://api.example.com"))
	})

	t.Run("an API the client is not granted is refused", func(t *testing.T) {
		db, mock := newMockDB(t)
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT count(*)`)).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))

		oerr := verifyClientAudience(db, 1, 10, "https://evil.example.com")
		require.NotNil(t, oerr)
		assert.Equal(t, "invalid_target", oerr.Code)
	})

	// Fails CLOSED on every "cannot tell": a target nobody checked must not end up
	// in `aud`, because that claim is exactly what a resource server trusts.
	t.Run("a query error is refused, not allowed", func(t *testing.T) {
		db, mock := newMockDB(t)
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT count(*)`)).WillReturnError(assert.AnError)

		oerr := verifyClientAudience(db, 1, 10, "https://api.example.com")
		require.NotNil(t, oerr)
		assert.Equal(t, "invalid_target", oerr.Code)
	})

	t.Run("a nil db is refused, not allowed", func(t *testing.T) {
		oerr := verifyClientAudience(nil, 1, 10, "https://api.example.com")
		require.NotNil(t, oerr)
		assert.Equal(t, "invalid_target", oerr.Code)
	})

	t.Run("a blank target is refused rather than read as no restriction", func(t *testing.T) {
		db, _ := newMockDB(t)
		oerr := verifyClientAudience(db, 1, 10, "   ")
		require.NotNil(t, oerr)
		assert.Equal(t, "invalid_target", oerr.Code)
	})
}

// The keyless workload-identity audience rule is NOT tested here. It used to be
// (TestResolveWorkloadAudience, against oauth.ResolveWorkloadAudience), but that
// resolver had no caller: the keyless path is decided by
// federation.resolveWorkloadAudience, which admits only the federation's
// registered audience and is covered by internal/federation/workload_audience_test.go.
// The two rules had already drifted — the dead one also admitted any
// client_apis-linked target — so this file was asserting a looser rule than the
// server enforces, which is worse than no test at all. What this package still
// owns for that path is the parameter parsing, covered by
// TestNormalizeRequestedTarget above and by
// TestOAuthTokenExchangeHandler_WorkloadIdentityTargetIsValidated.
