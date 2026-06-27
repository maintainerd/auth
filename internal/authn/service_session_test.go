package authn

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/secpolicy"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSessionService_ListSessions(t *testing.T) {
	sessionUUID := uuid.New()
	now := time.Now()

	t.Run("success with sessions", func(t *testing.T) {
		repo := &mockUserTokenRepo{
			findActiveSessionsFn: func(int64) ([]UserToken, error) {
				return []UserToken{
					{UserTokenUUID: sessionUUID, TokenType: "session", CreatedAt: now},
				}, nil
			},
		}
		svc := NewSessionService(repo)
		result, err := svc.ListSessions(context.Background(), 1)
		require.NoError(t, err)
		require.Len(t, result, 1)
		assert.Equal(t, sessionUUID.String(), result[0].SessionID)
	})

	t.Run("empty list", func(t *testing.T) {
		repo := &mockUserTokenRepo{
			findActiveSessionsFn: func(int64) ([]UserToken, error) {
				return []UserToken{}, nil
			},
		}
		svc := NewSessionService(repo)
		result, err := svc.ListSessions(context.Background(), 1)
		require.NoError(t, err)
		assert.Empty(t, result)
	})

	t.Run("repo error", func(t *testing.T) {
		repo := &mockUserTokenRepo{
			findActiveSessionsFn: func(int64) ([]UserToken, error) {
				return nil, errors.New("db error")
			},
		}
		svc := NewSessionService(repo)
		result, err := svc.ListSessions(context.Background(), 1)
		require.Error(t, err)
		assert.Nil(t, result)
	})
}

func TestSessionService_RevokeSession(t *testing.T) {
	sessionUUID := uuid.New()

	t.Run("success", func(t *testing.T) {
		repo := &mockUserTokenRepo{
			findActiveSessionByUUIDFn: func(int64, uuid.UUID) (*UserToken, error) {
				return &UserToken{UserTokenUUID: sessionUUID}, nil
			},
		}
		svc := NewSessionService(repo)
		err := svc.RevokeSession(context.Background(), 1, sessionUUID)
		require.NoError(t, err)
	})

	t.Run("session not found", func(t *testing.T) {
		repo := &mockUserTokenRepo{
			findActiveSessionByUUIDFn: func(int64, uuid.UUID) (*UserToken, error) {
				return nil, nil
			},
		}
		svc := NewSessionService(repo)
		err := svc.RevokeSession(context.Background(), 1, sessionUUID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})

	t.Run("lookup error", func(t *testing.T) {
		repo := &mockUserTokenRepo{
			findActiveSessionByUUIDFn: func(int64, uuid.UUID) (*UserToken, error) {
				return nil, errors.New("db error")
			},
		}
		svc := NewSessionService(repo)
		err := svc.RevokeSession(context.Background(), 1, sessionUUID)
		require.Error(t, err)
	})

	t.Run("revoke error", func(t *testing.T) {
		repo := &mockUserTokenRepo{
			findActiveSessionByUUIDFn: func(int64, uuid.UUID) (*UserToken, error) {
				return &UserToken{}, nil
			},
			revokeSessionByUUIDFn: func(int64, uuid.UUID) error {
				return errors.New("revoke error")
			},
		}
		svc := NewSessionService(repo)
		err := svc.RevokeSession(context.Background(), 1, sessionUUID)
		require.Error(t, err)
	})
}

func TestSessionService_RevokeAllSessions(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		repo := &mockUserTokenRepo{}
		svc := NewSessionService(repo)
		err := svc.RevokeAllSessions(context.Background(), 1)
		require.NoError(t, err)
	})

	t.Run("error", func(t *testing.T) {
		repo := &mockUserTokenRepo{
			revokeAllSessionsByUserIDFn: func(int64) error {
				return errors.New("revoke error")
			},
		}
		svc := NewSessionService(repo)
		err := svc.RevokeAllSessions(context.Background(), 1)
		require.Error(t, err)
	})
}

func TestSessionService_CreateSession(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		repo := &mockUserTokenRepo{
			createFn: func(t *UserToken) (*UserToken, error) {
				t.UserTokenUUID = uuid.New()
				return t, nil
			},
		}
		svc := NewSessionService(repo)
		result, err := svc.CreateSession(context.Background(), 1, "192.168.1.1", "UA/1.0")
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, int64(1), result.UserID)
		assert.Equal(t, "user:session", result.TokenType)
		assert.False(t, result.IsRevoked)
	})

	t.Run("create error", func(t *testing.T) {
		repo := &mockUserTokenRepo{
			createFn: func(*UserToken) (*UserToken, error) {
				return nil, errors.New("db error")
			},
		}
		svc := NewSessionService(repo)
		result, err := svc.CreateSession(context.Background(), 1, "", "")
		require.Error(t, err)
		assert.Nil(t, result)
	})

	t.Run("random token generation error", func(t *testing.T) {
		originalRandRead := randRead
		randRead = func([]byte) (int, error) {
			return 0, errors.New("entropy unavailable")
		}
		t.Cleanup(func() { randRead = originalRandRead })

		repo := &mockUserTokenRepo{}
		svc := NewSessionService(repo)
		result, err := svc.CreateSession(context.Background(), 1, "", "")
		require.Error(t, err)
		assert.Nil(t, result)
	})

	t.Run("uses policy idle and absolute timeouts", func(t *testing.T) {
		repo := &mockUserTokenRepo{
			createFn: func(token *UserToken) (*UserToken, error) {
				token.UserTokenUUID = uuid.New()
				require.NotNil(t, token.IdleTimeoutSeconds)
				assert.Equal(t, 600, *token.IdleTimeoutSeconds)
				require.NotNil(t, token.AbsoluteExpiresAt)
				assert.WithinDuration(t, time.Now().Add(2*time.Hour), *token.AbsoluteExpiresAt, 5*time.Second)
				return token, nil
			},
		}
		svc := NewSessionService(repo).(*sessionService)
		result, err := svc.CreateSessionWithPolicy(context.Background(), 1, "", "", secpolicy.EffectiveSessionPolicy{
			IdleTimeoutSeconds:     600,
			AbsoluteTimeoutSeconds: 7200,
		})
		require.NoError(t, err)
		assert.NotNil(t, result)
	})
}

func TestGenerateRandomToken(t *testing.T) {
	token, err := generateRandomToken(4)
	require.NoError(t, err)
	assert.Len(t, token, 8)
}

func TestGenerateRandomToken_Error(t *testing.T) {
	originalRandRead := randRead
	randRead = func([]byte) (int, error) {
		return 0, errors.New("entropy unavailable")
	}
	t.Cleanup(func() { randRead = originalRandRead })

	token, err := generateRandomToken(4)
	require.Error(t, err)
	assert.Empty(t, token)
}

func TestSessionService_EnforceConcurrentLimit(t *testing.T) {
	userUUID := uuid.New()
	sessionUUID := uuid.New()

	t.Run("under limit", func(t *testing.T) {
		repo := &mockUserTokenRepo{
			countActiveSessionsFn: func(int64) (int64, error) {
				return 2, nil
			},
		}
		svc := NewSessionService(repo)
		err := svc.EnforceConcurrentLimit(context.Background(), userUUID, 1)
		require.NoError(t, err)
	})

	t.Run("policy limit of zero is unlimited", func(t *testing.T) {
		repo := &mockUserTokenRepo{
			countActiveSessionsFn: func(int64) (int64, error) {
				return 99, nil
			},
			findActiveSessionsFn: func(int64) ([]UserToken, error) {
				t.Fatal("must not evict when policy allows unlimited sessions")
				return nil, nil
			},
		}
		svc := NewSessionService(repo).(*sessionService)
		err := svc.EnforceConcurrentLimitWithPolicy(context.Background(), userUUID, 1, secpolicy.EffectiveSessionPolicy{MaxConcurrentSessions: 0})
		require.NoError(t, err)
	})

	t.Run("policy evicts at custom limit", func(t *testing.T) {
		var revoked bool
		repo := &mockUserTokenRepo{
			countActiveSessionsFn: func(int64) (int64, error) {
				return 1, nil
			},
			findActiveSessionsFn: func(int64) ([]UserToken, error) {
				return []UserToken{{UserTokenUUID: sessionUUID}}, nil
			},
			revokeSessionByUUIDFn: func(int64, uuid.UUID) error {
				revoked = true
				return nil
			},
		}
		svc := NewSessionService(repo).(*sessionService)
		err := svc.EnforceConcurrentLimitWithPolicy(context.Background(), userUUID, 1, secpolicy.EffectiveSessionPolicy{MaxConcurrentSessions: 1})
		require.NoError(t, err)
		assert.True(t, revoked)
	})

	t.Run("count error", func(t *testing.T) {
		repo := &mockUserTokenRepo{
			countActiveSessionsFn: func(int64) (int64, error) {
				return 0, errors.New("db error")
			},
		}
		svc := NewSessionService(repo)
		err := svc.EnforceConcurrentLimit(context.Background(), userUUID, 1)
		require.Error(t, err)
	})

	t.Run("evicts oldest when at limit", func(t *testing.T) {
		repo := &mockUserTokenRepo{
			countActiveSessionsFn: func(int64) (int64, error) {
				return 5, nil
			},
			findActiveSessionsFn: func(int64) ([]UserToken, error) {
				return []UserToken{
					{UserTokenUUID: sessionUUID},
				}, nil
			},
		}
		svc := NewSessionService(repo)
		err := svc.EnforceConcurrentLimit(context.Background(), userUUID, 1)
		require.NoError(t, err)
	})

	t.Run("find sessions for eviction error", func(t *testing.T) {
		repo := &mockUserTokenRepo{
			countActiveSessionsFn: func(int64) (int64, error) {
				return 5, nil
			},
			findActiveSessionsFn: func(int64) ([]UserToken, error) {
				return nil, errors.New("db error")
			},
		}
		svc := NewSessionService(repo)
		err := svc.EnforceConcurrentLimit(context.Background(), userUUID, 1)
		require.Error(t, err)
	})

	t.Run("eviction error during revoke", func(t *testing.T) {
		repo := &mockUserTokenRepo{
			countActiveSessionsFn: func(int64) (int64, error) {
				return 5, nil
			},
			findActiveSessionsFn: func(int64) ([]UserToken, error) {
				return []UserToken{
					{UserTokenUUID: sessionUUID},
				}, nil
			},
			revokeSessionByUUIDFn: func(int64, uuid.UUID) error {
				return errors.New("revoke error")
			},
		}
		svc := NewSessionService(repo)
		err := svc.EnforceConcurrentLimit(context.Background(), userUUID, 1)
		require.Error(t, err)
	})

	t.Run("no sessions to evict", func(t *testing.T) {
		repo := &mockUserTokenRepo{
			countActiveSessionsFn: func(int64) (int64, error) {
				return 5, nil
			},
			findActiveSessionsFn: func(int64) ([]UserToken, error) {
				return []UserToken{}, nil
			},
		}
		svc := NewSessionService(repo)
		err := svc.EnforceConcurrentLimit(context.Background(), userUUID, 1)
		require.NoError(t, err)
	})
}

func TestSessionService_ValidateAndTouch(t *testing.T) {
	sessionUUID := uuid.New()

	t.Run("success", func(t *testing.T) {
		repo := &mockUserTokenRepo{
			findActiveSessionByUUIDFn: func(int64, uuid.UUID) (*UserToken, error) {
				return &UserToken{UserTokenUUID: sessionUUID}, nil
			},
		}
		svc := NewSessionService(repo)
		err := svc.ValidateAndTouch(context.Background(), sessionUUID, 1)
		require.NoError(t, err)
	})

	t.Run("session not found", func(t *testing.T) {
		repo := &mockUserTokenRepo{
			findActiveSessionByUUIDFn: func(int64, uuid.UUID) (*UserToken, error) {
				return nil, nil
			},
		}
		svc := NewSessionService(repo)
		err := svc.ValidateAndTouch(context.Background(), sessionUUID, 1)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "revoked")
	})

	t.Run("lookup error", func(t *testing.T) {
		repo := &mockUserTokenRepo{
			findActiveSessionByUUIDFn: func(int64, uuid.UUID) (*UserToken, error) {
				return nil, errors.New("db error")
			},
		}
		svc := NewSessionService(repo)
		err := svc.ValidateAndTouch(context.Background(), sessionUUID, 1)
		require.Error(t, err)
	})

	t.Run("absolute expiry revokes and returns unauthorized", func(t *testing.T) {
		expiredAt := time.Now().Add(-time.Minute)
		revoked := false
		repo := &mockUserTokenRepo{
			findActiveSessionByUUIDFn: func(int64, uuid.UUID) (*UserToken, error) {
				return &UserToken{UserTokenUUID: sessionUUID, AbsoluteExpiresAt: &expiredAt}, nil
			},
			revokeSessionByUUIDFn: func(int64, uuid.UUID) error {
				revoked = true
				return nil
			},
		}
		svc := NewSessionService(repo)
		err := svc.ValidateAndTouch(context.Background(), sessionUUID, 1)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "expired")
		assert.True(t, revoked)
	})

	t.Run("idle expiry revokes and returns unauthorized", func(t *testing.T) {
		lastUsed := time.Now().Add(-time.Hour)
		idleTimeout := 60
		revoked := false
		repo := &mockUserTokenRepo{
			findActiveSessionByUUIDFn: func(int64, uuid.UUID) (*UserToken, error) {
				return &UserToken{
					UserTokenUUID:      sessionUUID,
					LastUsedAt:         &lastUsed,
					IdleTimeoutSeconds: &idleTimeout,
				}, nil
			},
			revokeSessionByUUIDFn: func(int64, uuid.UUID) error {
				revoked = true
				return nil
			},
		}
		svc := NewSessionService(repo)
		err := svc.ValidateAndTouch(context.Background(), sessionUUID, 1)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "inactivity")
		assert.True(t, revoked)
	})

	t.Run("touch error is non fatal", func(t *testing.T) {
		repo := &mockUserTokenRepo{
			findActiveSessionByUUIDFn: func(int64, uuid.UUID) (*UserToken, error) {
				return &UserToken{UserTokenUUID: sessionUUID}, nil
			},
			touchSessionFn: func(int64, uuid.UUID, time.Time) error {
				return errors.New("touch error")
			},
		}
		svc := NewSessionService(repo)
		err := svc.ValidateAndTouch(context.Background(), sessionUUID, 1)
		require.NoError(t, err)
	})
}
