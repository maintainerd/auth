package authn

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/maintainerd/maintainerd-auth/internal/secpolicy"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockUserSessionRepo struct {
	findActiveByUserIDFn func(int64) ([]UserSession, error)
	findActiveByUUIDFn   func(int64, uuid.UUID) (*UserSession, error)
	countActiveFn        func(int64) (int64, error)
	createFn             func(*UserSession) error
	touchFn              func(int64, time.Time) error
	revokeByUUIDFn       func(int64, uuid.UUID, string) error
	revokeAllByUserIDFn  func(int64) error
}

func (m *mockUserSessionRepo) FindActiveByUserID(id int64) ([]UserSession, error) {
	if m.findActiveByUserIDFn != nil {
		return m.findActiveByUserIDFn(id)
	}
	return nil, nil
}
func (m *mockUserSessionRepo) FindActiveByUUID(id int64, uid uuid.UUID) (*UserSession, error) {
	if m.findActiveByUUIDFn != nil {
		return m.findActiveByUUIDFn(id, uid)
	}
	return nil, nil
}
func (m *mockUserSessionRepo) CountActive(id int64) (int64, error) {
	if m.countActiveFn != nil {
		return m.countActiveFn(id)
	}
	return 0, nil
}
func (m *mockUserSessionRepo) Create(s *UserSession) error {
	if m.createFn != nil {
		return m.createFn(s)
	}
	return nil
}
func (m *mockUserSessionRepo) Touch(id int64, now time.Time) error {
	if m.touchFn != nil {
		return m.touchFn(id, now)
	}
	return nil
}
func (m *mockUserSessionRepo) RevokeByUUID(id int64, uid uuid.UUID, reason string) error {
	if m.revokeByUUIDFn != nil {
		return m.revokeByUUIDFn(id, uid, reason)
	}
	return nil
}
func (m *mockUserSessionRepo) RevokeAllByUserID(id int64) error {
	if m.revokeAllByUserIDFn != nil {
		return m.revokeAllByUserIDFn(id)
	}
	return nil
}
func (m *mockUserSessionRepo) DeleteExpired() (int64, error) {
	return 0, nil
}

func TestNewSessionService(t *testing.T) {
	svc := NewSessionService(&mockUserSessionRepo{})
	assert.NotNil(t, svc)
}

func TestSessionService_ListSessions(t *testing.T) {
	sessionUUID := uuid.New()
	now := time.Now()

	t.Run("success with sessions", func(t *testing.T) {
		repo := &mockUserSessionRepo{
			findActiveByUserIDFn: func(int64) ([]UserSession, error) {
				return []UserSession{
					{UserSessionUUID: sessionUUID, CreatedAt: now},
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
		repo := &mockUserSessionRepo{
			findActiveByUserIDFn: func(int64) ([]UserSession, error) {
				return []UserSession{}, nil
			},
		}
		svc := NewSessionService(repo)
		result, err := svc.ListSessions(context.Background(), 1)
		require.NoError(t, err)
		assert.Empty(t, result)
	})

	t.Run("repo error", func(t *testing.T) {
		repo := &mockUserSessionRepo{
			findActiveByUserIDFn: func(int64) ([]UserSession, error) {
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
		repo := &mockUserSessionRepo{
			findActiveByUUIDFn: func(int64, uuid.UUID) (*UserSession, error) {
				return &UserSession{UserSessionUUID: sessionUUID}, nil
			},
		}
		svc := NewSessionService(repo)
		err := svc.RevokeSession(context.Background(), 1, sessionUUID)
		require.NoError(t, err)
	})

	t.Run("session not found", func(t *testing.T) {
		repo := &mockUserSessionRepo{
			findActiveByUUIDFn: func(int64, uuid.UUID) (*UserSession, error) {
				return nil, nil
			},
		}
		svc := NewSessionService(repo)
		err := svc.RevokeSession(context.Background(), 1, sessionUUID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})

	t.Run("lookup error", func(t *testing.T) {
		repo := &mockUserSessionRepo{
			findActiveByUUIDFn: func(int64, uuid.UUID) (*UserSession, error) {
				return nil, errors.New("db error")
			},
		}
		svc := NewSessionService(repo)
		err := svc.RevokeSession(context.Background(), 1, sessionUUID)
		require.Error(t, err)
	})

	t.Run("revoke error", func(t *testing.T) {
		repo := &mockUserSessionRepo{
			findActiveByUUIDFn: func(int64, uuid.UUID) (*UserSession, error) {
				return &UserSession{}, nil
			},
			revokeByUUIDFn: func(int64, uuid.UUID, string) error {
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
		repo := &mockUserSessionRepo{}
		svc := NewSessionService(repo)
		err := svc.RevokeAllSessions(context.Background(), 1)
		require.NoError(t, err)
	})

	t.Run("error", func(t *testing.T) {
		repo := &mockUserSessionRepo{
			revokeAllByUserIDFn: func(int64) error {
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
		repo := &mockUserSessionRepo{
			createFn: func(s *UserSession) error {
				s.UserSessionUUID = uuid.New()
				return nil
			},
		}
		svc := NewSessionService(repo)
		result, err := svc.CreateSession(context.Background(), 1, 1, "192.168.1.1", "UA/1.0")
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, int64(1), result.UserID)
	})

	t.Run("create error", func(t *testing.T) {
		repo := &mockUserSessionRepo{
			createFn: func(*UserSession) error {
				return errors.New("db error")
			},
		}
		svc := NewSessionService(repo)
		result, err := svc.CreateSession(context.Background(), 1, 1, "", "")
		require.Error(t, err)
		assert.Nil(t, result)
	})

	t.Run("uses policy idle and absolute timeouts", func(t *testing.T) {
		repo := &mockUserSessionRepo{
			createFn: func(s *UserSession) error {
				s.UserSessionUUID = uuid.New()
				require.Equal(t, 600, s.IdleTimeoutSeconds)
				assert.WithinDuration(t, time.Now().Add(2*time.Hour), s.ExpiresAt, 5*time.Second)
				return nil
			},
		}
		svc := NewSessionService(repo).(*sessionService)
		result, err := svc.CreateSessionWithPolicy(context.Background(), 1, 1, "", "", secpolicy.EffectiveSessionPolicy{
			IdleTimeoutSeconds:     600,
			AbsoluteTimeoutSeconds: 7200,
		})
		require.NoError(t, err)
		assert.NotNil(t, result)
	})
}

func TestSessionService_EnforceConcurrentLimit(t *testing.T) {
	userUUID := uuid.New()
	sessionUUID := uuid.New()

	t.Run("under limit", func(t *testing.T) {
		repo := &mockUserSessionRepo{
			countActiveFn: func(int64) (int64, error) {
				return 2, nil
			},
		}
		svc := NewSessionService(repo)
		err := svc.EnforceConcurrentLimit(context.Background(), userUUID, 1)
		require.NoError(t, err)
	})

	t.Run("policy limit of zero is unlimited", func(t *testing.T) {
		repo := &mockUserSessionRepo{
			countActiveFn: func(int64) (int64, error) {
				return 99, nil
			},
			findActiveByUserIDFn: func(int64) ([]UserSession, error) {
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
		repo := &mockUserSessionRepo{
			countActiveFn: func(int64) (int64, error) {
				return 1, nil
			},
			findActiveByUserIDFn: func(int64) ([]UserSession, error) {
				return []UserSession{{UserSessionUUID: sessionUUID}}, nil
			},
			revokeByUUIDFn: func(int64, uuid.UUID, string) error {
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
		repo := &mockUserSessionRepo{
			countActiveFn: func(int64) (int64, error) {
				return 0, errors.New("db error")
			},
		}
		svc := NewSessionService(repo)
		err := svc.EnforceConcurrentLimit(context.Background(), userUUID, 1)
		require.Error(t, err)
	})

	t.Run("evicts oldest when at limit", func(t *testing.T) {
		repo := &mockUserSessionRepo{
			countActiveFn: func(int64) (int64, error) {
				return 5, nil
			},
			findActiveByUserIDFn: func(int64) ([]UserSession, error) {
				return []UserSession{
					{UserSessionUUID: sessionUUID},
				}, nil
			},
		}
		svc := NewSessionService(repo)
		err := svc.EnforceConcurrentLimit(context.Background(), userUUID, 1)
		require.NoError(t, err)
	})

	t.Run("find sessions for eviction error", func(t *testing.T) {
		repo := &mockUserSessionRepo{
			countActiveFn: func(int64) (int64, error) {
				return 5, nil
			},
			findActiveByUserIDFn: func(int64) ([]UserSession, error) {
				return nil, errors.New("db error")
			},
		}
		svc := NewSessionService(repo)
		err := svc.EnforceConcurrentLimit(context.Background(), userUUID, 1)
		require.Error(t, err)
	})

	t.Run("eviction error during revoke", func(t *testing.T) {
		repo := &mockUserSessionRepo{
			countActiveFn: func(int64) (int64, error) {
				return 5, nil
			},
			findActiveByUserIDFn: func(int64) ([]UserSession, error) {
				return []UserSession{
					{UserSessionUUID: sessionUUID},
				}, nil
			},
			revokeByUUIDFn: func(int64, uuid.UUID, string) error {
				return errors.New("revoke error")
			},
		}
		svc := NewSessionService(repo)
		err := svc.EnforceConcurrentLimit(context.Background(), userUUID, 1)
		require.Error(t, err)
	})

	t.Run("no sessions to evict", func(t *testing.T) {
		repo := &mockUserSessionRepo{
			countActiveFn: func(int64) (int64, error) {
				return 5, nil
			},
			findActiveByUserIDFn: func(int64) ([]UserSession, error) {
				return []UserSession{}, nil
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
		now := time.Now()
		repo := &mockUserSessionRepo{
			findActiveByUUIDFn: func(int64, uuid.UUID) (*UserSession, error) {
				return &UserSession{
					UserSessionUUID:    sessionUUID,
					UserSessionID:      1,
					ExpiresAt:          now.Add(time.Hour),
					LastActiveAt:       now,
					IdleTimeoutSeconds: 3600,
				}, nil
			},
		}
		svc := NewSessionService(repo)
		err := svc.ValidateAndTouch(context.Background(), sessionUUID, 1)
		require.NoError(t, err)
	})

	t.Run("session not found", func(t *testing.T) {
		repo := &mockUserSessionRepo{
			findActiveByUUIDFn: func(int64, uuid.UUID) (*UserSession, error) {
				return nil, nil
			},
		}
		svc := NewSessionService(repo)
		err := svc.ValidateAndTouch(context.Background(), sessionUUID, 1)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "revoked")
	})

	t.Run("lookup error", func(t *testing.T) {
		repo := &mockUserSessionRepo{
			findActiveByUUIDFn: func(int64, uuid.UUID) (*UserSession, error) {
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
		repo := &mockUserSessionRepo{
			findActiveByUUIDFn: func(int64, uuid.UUID) (*UserSession, error) {
				return &UserSession{UserSessionUUID: sessionUUID, ExpiresAt: expiredAt}, nil
			},
			revokeByUUIDFn: func(int64, uuid.UUID, string) error {
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
		lastActive := time.Now().Add(-time.Hour)
		idleTimeout := 60
		revoked := false
		repo := &mockUserSessionRepo{
			findActiveByUUIDFn: func(int64, uuid.UUID) (*UserSession, error) {
				return &UserSession{
					UserSessionUUID:    sessionUUID,
					LastActiveAt:       lastActive,
					IdleTimeoutSeconds: idleTimeout,
				}, nil
			},
			revokeByUUIDFn: func(int64, uuid.UUID, string) error {
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
		var touched int64
		now := time.Now()
		repo := &mockUserSessionRepo{
			findActiveByUUIDFn: func(int64, uuid.UUID) (*UserSession, error) {
				return &UserSession{
					UserSessionUUID:    sessionUUID,
					UserSessionID:      42,
					ExpiresAt:          now.Add(time.Hour),
					LastActiveAt:       now,
					IdleTimeoutSeconds: 3600,
				}, nil
			},
			touchFn: func(id int64, _ time.Time) error {
				touched = id
				return errors.New("touch error")
			},
		}
		svc := NewSessionService(repo)
		err := svc.ValidateAndTouch(context.Background(), sessionUUID, 1)
		require.NoError(t, err)
		assert.Equal(t, int64(42), touched)
	})
}
