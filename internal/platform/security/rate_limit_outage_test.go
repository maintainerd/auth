package security

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The lockout store going away must not silently switch brute-force protection
// off. "No client wired" and "wired client that will not answer" are treated
// differently on purpose — see limiterOutagePolicy in security.go.
func TestRateLimitFailsClosedOnStoreOutage(t *testing.T) {
	t.Run("configured store that is down fails the attempt closed", func(t *testing.T) {
		mr, cli := newMiniredisClient(t)
		InitRateLimiter(cli)
		t.Cleanup(func() { InitRateLimiter(nil) })

		// Take the store away underneath a configured client.
		mr.Close()

		err := CheckRateLimit("tenant:user@example.com")
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrRateLimiterUnavailable)

		err = CheckRateLimitWithConfig("tenant:user@example.com", &RateLimitConfig{
			Enabled:           true,
			MaxFailedAttempts: 5,
		})
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrRateLimiterUnavailable)
	})

	t.Run("a missing counter key is not an outage", func(t *testing.T) {
		_, cli := newMiniredisClient(t)
		InitRateLimiter(cli)
		t.Cleanup(func() { InitRateLimiter(nil) })

		// redis.Nil is the normal "no failures recorded yet" answer and must
		// still allow the attempt through.
		assert.NoError(t, CheckRateLimit("tenant:never-seen@example.com"))
		assert.NoError(t, CheckRateLimitWithConfig("tenant:never-seen@example.com", &RateLimitConfig{
			Enabled:           true,
			MaxFailedAttempts: 5,
		}))
	})

	t.Run("an unconfigured limiter still allows, so local dev and tests work", func(t *testing.T) {
		InitRateLimiter(nil)
		assert.NoError(t, CheckRateLimit("tenant:user@example.com"))
		assert.NoError(t, CheckRateLimitWithConfig("tenant:user@example.com", nil))
	})

	t.Run("a disabled policy short-circuits before touching the store", func(t *testing.T) {
		mr, cli := newMiniredisClient(t)
		InitRateLimiter(cli)
		t.Cleanup(func() { InitRateLimiter(nil) })
		mr.Close()

		assert.NoError(t, CheckRateLimitWithConfig("tenant:user@example.com", &RateLimitConfig{Enabled: false}))
	})
}
