package user

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/maintainerd/maintainerd-auth/internal/authevent"
	"github.com/maintainerd/maintainerd-auth/internal/platform/security"
	"github.com/maintainerd/maintainerd-auth/internal/shared"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
)

// recoveryFixture builds an accountService whose repos all resolve, so the only
// thing under test is the credential check itself.
func recoveryFixture(t *testing.T, storedPassword *string, codeHash string) *accountService {
	t.Helper()
	db, mock := newMockGormDB(t)
	// The verification runs in a transaction that commits on success and rolls
	// back on any credential failure; accept either without pinning the order.
	mock.MatchExpectationsInOrder(false)
	mock.ExpectBegin()
	mock.ExpectCommit()
	mock.ExpectRollback()

	domain := "example.com"
	identifier := "client-id"
	client := &Client{
		ClientID:         1,
		Domain:           &domain,
		Identifier:       &identifier,
		Status:           shared.StatusActive,
		IdentityProvider: &IdentityProvider{Identifier: identifier},
	}
	user := &User{
		UserID:   42,
		UserUUID: uuid.New(),
		Email:    "test@example.com",
		Status:   shared.StatusActive,
		Password: storedPassword,
	}

	return &accountService{
		db:               db,
		sessionCreator:   stubSessionCreator{},
		authEventService: authevent.NoopService(),
		identityProviderRepo: &mockIdentityProviderRepo{
			findByIdentifierFn: func(string) (*IdentityProvider, error) {
				return &IdentityProvider{Identifier: identifier}, nil
			},
		},
		clientRepo: &mockClientRepo{
			findByClientIDAndIdentityProviderFn: func(string, string) (*Client, error) { return client, nil },
		},
		userRepo: &mockUserRepo{
			findByEmailFn: func(string) (*User, error) { return user, nil },
		},
		userIdentityRepo: &mockUserIdentityRepo{
			findByUserIDAndClientIDFn: func(int64, int64) (*UserIdentity, error) {
				return &UserIdentity{UserID: 42, Sub: "test-sub"}, nil
			},
		},
		mfaBackupCodeRepo: &mockUserMFABackupCodeRepo{
			findUnusedByUserIDFn: func(int64) ([]UserMFABackupCode, error) {
				return []UserMFABackupCode{{BackupCodeID: 100, UserID: 42, CodeHash: codeHash}}, nil
			},
		},
	}
}

// A backup code is a recovery SECOND factor. Accepting it alone minted a full
// access + refresh token set from an email address and one short code, which
// bypasses the tenant's enforced-MFA policy outright.
func TestAccountService_VerifyBackupCode_RequiresThePassword(t *testing.T) {
	initJWTKeys(t)

	const password = "correct-horse-battery"
	const code = "12345678"
	pwHash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.MinCost)
	require.NoError(t, err)
	pwStr := string(pwHash)
	codeHash, err := bcrypt.GenerateFromPassword([]byte(code), bcrypt.MinCost)
	require.NoError(t, err)

	req := func(pw, c string) VerifyBackupCodeDTO {
		return VerifyBackupCodeDTO{
			Email: "test@example.com", Password: pw, Code: c,
			ClientID: "test-client", ProviderID: "test-provider",
		}
	}

	t.Run("a valid code with the wrong password is refused", func(t *testing.T) {
		svc := recoveryFixture(t, &pwStr, string(codeHash))

		_, err := svc.VerifyBackupCode(context.Background(), req("not-the-password", code))

		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid email, password, or backup code")
	})

	t.Run("a valid password with the wrong code is refused", func(t *testing.T) {
		svc := recoveryFixture(t, &pwStr, string(codeHash))

		_, err := svc.VerifyBackupCode(context.Background(), req(password, "87654321"))

		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid email, password, or backup code")
	})

	t.Run("a passwordless account cannot recover this way", func(t *testing.T) {
		// A federated account has no password to pair with the code, and the code
		// standing alone is the whole defect.
		svc := recoveryFixture(t, nil, string(codeHash))

		_, err := svc.VerifyBackupCode(context.Background(), req(password, code))

		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid email, password, or backup code")
	})

	t.Run("both factors together succeed", func(t *testing.T) {
		svc := recoveryFixture(t, &pwStr, string(codeHash))

		res, err := svc.VerifyBackupCode(context.Background(), req(password, code))

		require.NoError(t, err)
		require.NotNil(t, res)
		assert.NotEmpty(t, res.AccessToken)
	})
}

// The route is registered outside the strict auth rate-limit group and records
// no login lockout, so without a per-account counter the only ceiling is the
// global 100/min/IP — enough to walk a code space, and enough to brute the
// password with the code held constant.
func TestAccountService_VerifyBackupCode_IsThrottledPerAccount(t *testing.T) {
	initJWTKeys(t)
	withRealRateLimiter(t)

	const password = "correct-horse-battery"
	const code = "12345678"
	pwHash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.MinCost)
	require.NoError(t, err)
	pwStr := string(pwHash)
	codeHash, err := bcrypt.GenerateFromPassword([]byte(code), bcrypt.MinCost)
	require.NoError(t, err)

	bad := VerifyBackupCodeDTO{
		Email: "test@example.com", Password: "wrong", Code: code,
		ClientID: "test-client", ProviderID: "test-provider",
	}

	for i := 0; i < security.MaxLoginAttempts; i++ {
		svc := recoveryFixture(t, &pwStr, string(codeHash))
		_, err := svc.VerifyBackupCode(context.Background(), bad)
		require.Error(t, err)
		assert.NotContains(t, err.Error(), "too many recovery attempts", "attempt %d should still be allowed", i+1)
	}

	svc := recoveryFixture(t, &pwStr, string(codeHash))
	_, err = svc.VerifyBackupCode(context.Background(), bad)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "too many recovery attempts")

	// The lockout must survive the correct credentials being presented — that is
	// what makes it a brute-force ceiling rather than a speed bump.
	good := bad
	good.Password = password
	svc = recoveryFixture(t, &pwStr, string(codeHash))
	_, err = svc.VerifyBackupCode(context.Background(), good)
	require.Error(t, err)
}

// The recovery throttle must not be the login lockout key: an attacker must not
// be able to lock the victim out of normal sign-in by hammering recovery.
func TestRecoveryBackupCodeThrottleKeyIsDistinct(t *testing.T) {
	key := recoveryBackupCodeThrottlePrefix + "app:user@example.com"
	assert.NotEqual(t, security.RateLimitKey("user@example.com", "login"), key)
	assert.NotEqual(t, accountPasswordThrottleKey(testUserUUID), key)
	// Keyed per client + email so a counter exists even when no such account
	// does — an unthrottled "no such user" path is its own enumeration channel.
	assert.NotEqual(t,
		recoveryBackupCodeThrottlePrefix+"app:a@example.com",
		recoveryBackupCodeThrottlePrefix+"app:b@example.com")

}

// stubSessionCreator stands in for authn's session service. Recovery binds its
// token to a real session, so a fixture without one correctly fails closed.
type stubSessionCreator struct{ err error }

func (s stubSessionCreator) CreateSession(context.Context, int64, int64, string, string) (uuid.UUID, error) {
	if s.err != nil {
		return uuid.Nil, s.err
	}
	return uuid.New(), nil
}
