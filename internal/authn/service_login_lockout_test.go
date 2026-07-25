package authn

import (
	"context"
	"testing"

	"github.com/maintainerd/maintainerd-auth/internal/platform/security"
	"github.com/stretchr/testify/assert"
)

type mockLockoutRepo struct {
	recordResult     LockoutResult
	recordCalled     bool
	lastRules        LockoutRules
	clearCalled      bool
	isLockedResponse bool
}

func (m *mockLockoutRepo) RecordFailure(_ context.Context, _ int64, _, _ string, rules LockoutRules) (LockoutResult, error) {
	m.recordCalled = true
	m.lastRules = rules
	return m.recordResult, nil
}
func (m *mockLockoutRepo) IsLocked(_ context.Context, _ int64, _ string) (bool, error) {
	return m.isLockedResponse, nil
}
func (m *mockLockoutRepo) ClearLockout(_ context.Context, _ int64, _ string) error {
	m.clearCalled = true
	return nil
}
func (m *mockLockoutRepo) ResetExpiredLockouts() (int64, error) { return 0, nil }

// notify_user_on_lockout must fire the OnAccountLockout hook, but only on the
// transition into a lock (JustLocked) and only when the tenant enabled it. It
// used to fire only from the dead Redis path, so a real DB lockout never
// notified anyone.
func TestRecordLockoutFailure_Notify(t *testing.T) {
	restore := security.OnAccountLockout
	t.Cleanup(func() { security.OnAccountLockout = restore })

	t.Run("fires once when a fresh lock is applied and notify is on", func(t *testing.T) {
		fired := 0
		security.OnAccountLockout = func(context.Context, string) { fired++ }
		repo := &mockLockoutRepo{recordResult: LockoutResult{Locked: true, JustLocked: true}}
		svc := &loginService{lockoutRepo: repo}

		svc.recordLockoutFailure(context.Background(), 1, "user@x.test",
			&security.RateLimitConfig{Enabled: true, NotifyUserOnLockout: true})

		assert.True(t, repo.recordCalled)
		assert.Equal(t, 1, fired)
	})

	t.Run("does not fire when the account was already locked", func(t *testing.T) {
		fired := 0
		security.OnAccountLockout = func(context.Context, string) { fired++ }
		repo := &mockLockoutRepo{recordResult: LockoutResult{Locked: true, JustLocked: false}}
		svc := &loginService{lockoutRepo: repo}

		svc.recordLockoutFailure(context.Background(), 1, "user@x.test",
			&security.RateLimitConfig{Enabled: true, NotifyUserOnLockout: true})

		assert.Equal(t, 0, fired, "no notification without a fresh lock transition")
	})

	t.Run("does not fire when notify is off", func(t *testing.T) {
		fired := 0
		security.OnAccountLockout = func(context.Context, string) { fired++ }
		repo := &mockLockoutRepo{recordResult: LockoutResult{Locked: true, JustLocked: true}}
		svc := &loginService{lockoutRepo: repo}

		svc.recordLockoutFailure(context.Background(), 1, "user@x.test",
			&security.RateLimitConfig{Enabled: true, NotifyUserOnLockout: false})

		assert.Equal(t, 0, fired)
	})

	t.Run("does nothing when lockout is disabled", func(t *testing.T) {
		repo := &mockLockoutRepo{}
		svc := &loginService{lockoutRepo: repo}

		svc.recordLockoutFailure(context.Background(), 1, "user@x.test",
			&security.RateLimitConfig{Enabled: false})

		assert.False(t, repo.recordCalled)
	})
}

// reset_count_on_success=false must leave the accumulated failure count in place
// across a successful login; the default (true) clears it.
func TestClearLockout_RespectsResetCountOnSuccess(t *testing.T) {
	t.Run("clears when reset_count_on_success is true", func(t *testing.T) {
		repo := &mockLockoutRepo{}
		svc := &loginService{lockoutRepo: repo}
		svc.clearLockout(context.Background(), 1, "user@x.test",
			&security.RateLimitConfig{Enabled: true, ResetCountOnSuccess: true})
		assert.True(t, repo.clearCalled)
	})

	t.Run("does not clear when reset_count_on_success is false", func(t *testing.T) {
		repo := &mockLockoutRepo{}
		svc := &loginService{lockoutRepo: repo}
		svc.clearLockout(context.Background(), 1, "user@x.test",
			&security.RateLimitConfig{Enabled: true, ResetCountOnSuccess: false})
		assert.False(t, repo.clearCalled)
	})
}
