package oauth

import (
	"context"
	"regexp"
	"testing"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	jwtlib "github.com/golang-jwt/jwt/v5"
	"github.com/maintainerd/maintainerd-auth/internal/platform/jwt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// exchangeSubjectToken mints an access token shaped like one this server issued
// for the client expectTokenExchangeClientLookup mocks.
func exchangeSubjectToken(t *testing.T, opts *jwt.AccessTokenOptions) string {
	t.Helper()
	token, err := jwt.GenerateAccessTokenWithOptionsContext(
		context.Background(), "user-sub", "openid",
		"https://auth.example.com", "my-client", "my-client", testExchangeRealm, opts)
	require.NoError(t, err)
	return token
}

func TestOAuthTokenExchangeService_SubjectTokenBinding(t *testing.T) {
	ctx := context.Background()

	// An ID token is handed to every relying party by design. Exchanging one used
	// to mint a first-class access token for its subject, because the generic
	// validator never looked at token_type.
	t.Run("an ID token is not exchangeable", func(t *testing.T) {
		initTestJWTKeysService(t)
		db, mock := newMockDB(t)
		expectTokenExchangeClientLookup(mock)

		idToken, err := jwt.GenerateIDTokenWithContext(ctx, "user-sub",
			"https://auth.example.com", "my-client", testExchangeRealm,
			&jwt.UserProfile{}, "", nil)
		require.NoError(t, err)

		svc := newOAuthTokenExchangeSvc(db, &mockUserRepo{}, &mockAuthEventService{})
		_, oerr := svc.Exchange(ctx, OAuthTokenExchangeRequestDTO{
			SubjectToken:     idToken,
			SubjectTokenType: tokenTypeAccessToken,
		}, OAuthClientCredentials{ClientID: "my-client"})
		require.NotNil(t, oerr)
		assert.Equal(t, "invalid_grant", oerr.Code)
	})

	t.Run("subject_token_type=id_token is refused outright", func(t *testing.T) {
		initTestJWTKeysService(t)
		db, mock := newMockDB(t)
		expectTokenExchangeClientLookup(mock)

		svc := newOAuthTokenExchangeSvc(db, &mockUserRepo{}, &mockAuthEventService{})
		_, oerr := svc.Exchange(ctx, OAuthTokenExchangeRequestDTO{
			SubjectToken:     exchangeSubjectToken(t, nil),
			SubjectTokenType: tokenTypeIDTokenURI,
		}, OAuthClientCredentials{ClientID: "my-client"})
		require.NotNil(t, oerr)
		assert.Equal(t, "invalid_request", oerr.Code)
	})

	// One process-wide signing key means the signature proves nothing about which
	// tenant issued the token, so another tenant's access token used to be
	// exchangeable for a first-class token for that subject.
	t.Run("another tenant's token is not exchangeable", func(t *testing.T) {
		initTestJWTKeysService(t)
		db, mock := newMockDB(t)
		expectTokenExchangeClientLookup(mock)

		foreign, err := jwt.GenerateAccessToken("user-sub", "openid",
			"https://auth.example.com", "my-client", "my-client", "some-other-tenant")
		require.NoError(t, err)

		svc := newOAuthTokenExchangeSvc(db, &mockUserRepo{}, &mockAuthEventService{})
		_, oerr := svc.Exchange(ctx, OAuthTokenExchangeRequestDTO{
			SubjectToken:     foreign,
			SubjectTokenType: tokenTypeAccessToken,
		}, OAuthClientCredentials{ClientID: "my-client"})
		require.NotNil(t, oerr)
		assert.Equal(t, "invalid_grant", oerr.Code)
		assert.Contains(t, oerr.Description, "tenant")
	})

	// Fails CLOSED: a token with no provider_id is one this check cannot reason
	// about.
	t.Run("a subject token with no provider_id is refused", func(t *testing.T) {
		orig := oauthTokenExchangeValidateTokenWithContext
		defer func() { oauthTokenExchangeValidateTokenWithContext = orig }()
		oauthTokenExchangeValidateTokenWithContext = func(context.Context, string) (jwtlib.MapClaims, error) {
			return jwtlib.MapClaims{"sub": "user-sub", "token_type": jwt.TokenTypeAccess}, nil
		}

		db, mock := newMockDB(t)
		expectTokenExchangeClientLookup(mock)

		svc := newOAuthTokenExchangeSvc(db, &mockUserRepo{}, &mockAuthEventService{})
		_, oerr := svc.Exchange(ctx, OAuthTokenExchangeRequestDTO{
			SubjectToken:     "x",
			SubjectTokenType: tokenTypeAccessToken,
		}, OAuthClientCredentials{ClientID: "my-client"})
		require.NotNil(t, oerr)
		assert.Equal(t, "invalid_grant", oerr.Code)
	})
}

func TestOAuthTokenExchangeService_SessionBinding(t *testing.T) {
	ctx := context.Background()

	// sub_type=exchange is treated as SESSIONLESS by the session middleware, so
	// stamping it unconditionally let a client launder a session-bound user token
	// into one that survives logout and "sign out everywhere" for its full TTL.
	t.Run("the subject token's session rides through the exchange", func(t *testing.T) {
		initTestJWTKeysService(t)
		db, mock := newMockDB(t)
		expectTokenExchangeClientLookup(mock)

		sid := "6f1b6f56-6a0e-4e2a-9a5f-1f0f4c7a1d10"
		subject := exchangeSubjectToken(t, &jwt.AccessTokenOptions{SessionID: sid})

		svc := newOAuthTokenExchangeSvc(db, &mockUserRepo{}, &mockAuthEventService{})
		result, oerr := svc.Exchange(ctx, OAuthTokenExchangeRequestDTO{
			SubjectToken:     subject,
			SubjectTokenType: tokenTypeAccessToken,
		}, OAuthClientCredentials{ClientID: "my-client"})
		require.Nil(t, oerr)

		claims, err := jwt.ValidateToken(result.AccessToken)
		require.NoError(t, err)
		assert.Equal(t, sid, claims["sid"])
		assert.NotEqual(t, subjectTypeExchange, claims["sub_type"],
			"a session-bound exchange must stay revocable, so it must not be labelled sessionless")
	})

	t.Run("a subject token with no session keeps the sessionless label", func(t *testing.T) {
		initTestJWTKeysService(t)
		db, mock := newMockDB(t)
		expectTokenExchangeClientLookup(mock)

		svc := newOAuthTokenExchangeSvc(db, &mockUserRepo{}, &mockAuthEventService{})
		result, oerr := svc.Exchange(ctx, OAuthTokenExchangeRequestDTO{
			SubjectToken:     exchangeSubjectToken(t, nil),
			SubjectTokenType: tokenTypeAccessToken,
		}, OAuthClientCredentials{ClientID: "my-client"})
		require.Nil(t, oerr)

		claims, err := jwt.ValidateToken(result.AccessToken)
		require.NoError(t, err)
		assert.Equal(t, subjectTypeExchange, claims["sub_type"])
	})
}

func TestOAuthTokenExchangeService_ActorToken(t *testing.T) {
	ctx := context.Background()

	// actor_token was read only to label the audit row "delegation": never
	// signature-verified, never bound to the authenticated client, and never
	// reflected in an `act` claim, so RFC 8693 §4.1's delegation chain did not
	// exist.
	t.Run("a garbage actor_token is refused", func(t *testing.T) {
		initTestJWTKeysService(t)
		db, mock := newMockDB(t)
		expectTokenExchangeClientLookup(mock)

		svc := newOAuthTokenExchangeSvc(db, &mockUserRepo{}, &mockAuthEventService{})
		_, oerr := svc.Exchange(ctx, OAuthTokenExchangeRequestDTO{
			SubjectToken:     exchangeSubjectToken(t, nil),
			SubjectTokenType: tokenTypeAccessToken,
			ActorToken:       "garbage",
			ActorTokenType:   tokenTypeAccessToken,
		}, OAuthClientCredentials{ClientID: "my-client"})
		require.NotNil(t, oerr)
		assert.Equal(t, "invalid_grant", oerr.Code)
	})

	t.Run("an actor_token issued to another client is refused", func(t *testing.T) {
		initTestJWTKeysService(t)
		db, mock := newMockDB(t)
		expectTokenExchangeClientLookup(mock)

		other, err := jwt.GenerateAccessToken("actor-sub", "openid",
			"https://auth.example.com", "other-client", "other-client", testExchangeRealm)
		require.NoError(t, err)

		svc := newOAuthTokenExchangeSvc(db, &mockUserRepo{}, &mockAuthEventService{})
		_, oerr := svc.Exchange(ctx, OAuthTokenExchangeRequestDTO{
			SubjectToken:     exchangeSubjectToken(t, nil),
			SubjectTokenType: tokenTypeAccessToken,
			ActorToken:       other,
			ActorTokenType:   tokenTypeAccessToken,
		}, OAuthClientCredentials{ClientID: "my-client"})
		require.NotNil(t, oerr)
		assert.Equal(t, "invalid_grant", oerr.Code)
	})

	t.Run("a verified actor is recorded in the act claim", func(t *testing.T) {
		initTestJWTKeysService(t)
		db, mock := newMockDB(t)
		expectTokenExchangeClientLookup(mock)

		actor, err := jwt.GenerateAccessToken("actor-sub", "openid",
			"https://auth.example.com", "my-client", "my-client", testExchangeRealm)
		require.NoError(t, err)

		svc := newOAuthTokenExchangeSvc(db, &mockUserRepo{}, &mockAuthEventService{})
		result, oerr := svc.Exchange(ctx, OAuthTokenExchangeRequestDTO{
			SubjectToken:     exchangeSubjectToken(t, nil),
			SubjectTokenType: tokenTypeAccessToken,
			ActorToken:       actor,
			ActorTokenType:   tokenTypeAccessToken,
		}, OAuthClientCredentials{ClientID: "my-client"})
		require.Nil(t, oerr)

		claims, err := jwt.ValidateToken(result.AccessToken)
		require.NoError(t, err)
		act, ok := claims["act"].(map[string]any)
		require.True(t, ok, "the issued token must carry an act claim")
		assert.Equal(t, "actor-sub", act["sub"])
	})
}

func TestOAuthTokenExchangeService_Audience(t *testing.T) {
	ctx := context.Background()

	// The caller's `audience` used to be copied straight onto the token, so a
	// client could mint a token addressed to any resource server it named.
	t.Run("an unregistered audience is invalid_target", func(t *testing.T) {
		initTestJWTKeysService(t)
		db, mock := newMockDB(t)
		expectTokenExchangeClientLookup(mock)
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT count(*)`)).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))

		svc := newOAuthTokenExchangeSvc(db, &mockUserRepo{}, &mockAuthEventService{})
		_, oerr := svc.Exchange(ctx, OAuthTokenExchangeRequestDTO{
			SubjectToken:     exchangeSubjectToken(t, nil),
			SubjectTokenType: tokenTypeAccessToken,
			Audience:         "https://not-my-api.example.com",
		}, OAuthClientCredentials{ClientID: "my-client"})
		require.NotNil(t, oerr)
		assert.Equal(t, "invalid_target", oerr.Code)
	})

	t.Run("a granted API becomes the token's aud", func(t *testing.T) {
		initTestJWTKeysService(t)
		db, mock := newMockDB(t)
		expectTokenExchangeClientLookup(mock)
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT count(*)`)).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

		svc := newOAuthTokenExchangeSvc(db, &mockUserRepo{}, &mockAuthEventService{})
		result, oerr := svc.Exchange(ctx, OAuthTokenExchangeRequestDTO{
			SubjectToken:     exchangeSubjectToken(t, nil),
			SubjectTokenType: tokenTypeAccessToken,
			Audience:         "https://orders.example.com",
		}, OAuthClientCredentials{ClientID: "my-client"})
		require.Nil(t, oerr)

		claims, err := jwt.ValidateToken(result.AccessToken)
		require.NoError(t, err)
		assert.Equal(t, "https://orders.example.com", claims["aud"])
	})
}
