package authn

import (
	"testing"
	"time"

	"github.com/maintainerd/maintainerd-auth/internal/platform/security"
	"github.com/stretchr/testify/assert"
)

func timePtr(t time.Time) *time.Time { return &t }

// The observation window is the fix for the worst lockout bug: without it the
// failure count only ever grew, so a once-locked account re-locked on every
// later failure forever. A failure outside the window must start a fresh count.
func TestNextFailedCount_ObservationWindow(t *testing.T) {
	now := time.Now()

	t.Run("first failure starts at 1", func(t *testing.T) {
		assert.Equal(t, 1, nextFailedCount(0, nil, now, 15*time.Minute))
	})

	t.Run("failure inside the window increments", func(t *testing.T) {
		last := now.Add(-5 * time.Minute)
		assert.Equal(t, 4, nextFailedCount(3, timePtr(last), now, 15*time.Minute))
	})

	t.Run("failure outside the window resets to 1", func(t *testing.T) {
		last := now.Add(-20 * time.Minute)
		assert.Equal(t, 1, nextFailedCount(3, timePtr(last), now, 15*time.Minute))
	})

	t.Run("window of 0 never resets (accumulates forever)", func(t *testing.T) {
		last := now.Add(-100 * time.Hour)
		assert.Equal(t, 4, nextFailedCount(3, timePtr(last), now, 0))
	})
}

func TestUserLockout_IsCurrentlyLocked(t *testing.T) {
	now := time.Now()

	t.Run("no lock", func(t *testing.T) {
		l := &UserLockout{}
		assert.False(t, l.isCurrentlyLocked(now))
	})

	t.Run("timed lock in the future", func(t *testing.T) {
		l := &UserLockout{LockedUntil: timePtr(now.Add(10 * time.Minute))}
		assert.True(t, l.isCurrentlyLocked(now))
	})

	t.Run("timed lock in the past is not locked", func(t *testing.T) {
		l := &UserLockout{LockedUntil: timePtr(now.Add(-1 * time.Minute))}
		assert.False(t, l.isCurrentlyLocked(now))
	})

	t.Run("permanent lock has no expiry but is locked", func(t *testing.T) {
		l := &UserLockout{Locked: true}
		assert.True(t, l.isCurrentlyLocked(now))
	})
}

func TestApplyLockout_FlatDuration(t *testing.T) {
	now := time.Now()
	l := &UserLockout{}
	applyLockout(l, now, LockoutRules{BaseDuration: 30 * time.Minute, AutoUnlock: true})

	assert.False(t, l.Locked, "auto_unlock=true is a timed lock, not permanent")
	assert.NotNil(t, l.LockedUntil)
	assert.WithinDuration(t, now.Add(30*time.Minute), *l.LockedUntil, time.Second)
	assert.Equal(t, 1, l.LockoutLevel)
}

// auto_unlock=false must produce a PERMANENT lock (no expiry) that only an admin
// clears — the state the old flat-duration path could not represent at all.
func TestApplyLockout_PermanentWhenAutoUnlockFalse(t *testing.T) {
	now := time.Now()
	l := &UserLockout{}
	applyLockout(l, now, LockoutRules{BaseDuration: 30 * time.Minute, AutoUnlock: false})

	assert.True(t, l.Locked, "auto_unlock=false must be a permanent lock")
	assert.Nil(t, l.LockedUntil, "a permanent lock has no expiry")
}

// Progressive lockout escalates the duration each time; the level resets once
// progression_reset has elapsed since the previous lock.
func TestApplyLockout_ProgressiveEscalation(t *testing.T) {
	now := time.Now()
	rules := LockoutRules{
		BaseDuration:     10 * time.Minute,
		Progressive:      true,
		MaxDuration:      60 * time.Minute,
		ProgressionReset: 24 * time.Hour,
		AutoUnlock:       true,
	}

	t.Run("first lock is level 1 = base", func(t *testing.T) {
		l := &UserLockout{}
		applyLockout(l, now, rules)
		assert.Equal(t, 1, l.LockoutLevel)
		assert.WithinDuration(t, now.Add(10*time.Minute), *l.LockedUntil, time.Second)
	})

	t.Run("second lock within the reset window escalates to level 2", func(t *testing.T) {
		l := &UserLockout{LockoutLevel: 1, LastLockedAt: timePtr(now.Add(-1 * time.Hour))}
		applyLockout(l, now, rules)
		assert.Equal(t, 2, l.LockoutLevel)
		assert.WithinDuration(t, now.Add(20*time.Minute), *l.LockedUntil, time.Second)
	})

	t.Run("escalation is capped at max_lockout_duration", func(t *testing.T) {
		l := &UserLockout{LockoutLevel: 8, LastLockedAt: timePtr(now.Add(-1 * time.Hour))}
		applyLockout(l, now, rules)
		assert.Equal(t, 9, l.LockoutLevel)
		// 9 * 10min = 90min, capped to 60min.
		assert.WithinDuration(t, now.Add(60*time.Minute), *l.LockedUntil, time.Second)
	})

	t.Run("level resets to 1 once progression_reset has elapsed", func(t *testing.T) {
		l := &UserLockout{LockoutLevel: 5, LastLockedAt: timePtr(now.Add(-48 * time.Hour))}
		applyLockout(l, now, rules)
		assert.Equal(t, 1, l.LockoutLevel, "an old last-lock must reset the escalation")
		assert.WithinDuration(t, now.Add(10*time.Minute), *l.LockedUntil, time.Second)
	})
}

// Non-progressive policy stays flat regardless of prior level.
func TestApplyLockout_NonProgressiveStaysFlat(t *testing.T) {
	now := time.Now()
	l := &UserLockout{LockoutLevel: 5, LastLockedAt: timePtr(now.Add(-1 * time.Minute))}
	applyLockout(l, now, LockoutRules{BaseDuration: 30 * time.Minute, Progressive: false, AutoUnlock: true})
	assert.WithinDuration(t, now.Add(30*time.Minute), *l.LockedUntil, time.Second)
}

func TestLockoutRulesFromPolicy_MapsEveryField(t *testing.T) {
	policy := &security.RateLimitConfig{
		MaxFailedAttempts:  5,
		LockoutDuration:    30 * time.Minute,
		ObservationWindow:  15 * time.Minute,
		ProgressiveLockout: true,
		MaxLockoutDuration: 60 * time.Minute,
		ProgressionReset:   24 * time.Hour,
		AutoUnlock:         false,
	}
	rules := lockoutRulesFromPolicy(policy)

	assert.Equal(t, 5, rules.MaxAttempts)
	assert.Equal(t, 30*time.Minute, rules.BaseDuration)
	assert.Equal(t, 15*time.Minute, rules.ObservationWindow)
	assert.True(t, rules.Progressive)
	assert.Equal(t, 60*time.Minute, rules.MaxDuration)
	assert.Equal(t, 24*time.Hour, rules.ProgressionReset)
	assert.False(t, rules.AutoUnlock)
}
