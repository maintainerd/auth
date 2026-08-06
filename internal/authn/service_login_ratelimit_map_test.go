package authn

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/maintainerd/maintainerd-auth/internal/platform/apperror"
	resp "github.com/maintainerd/maintainerd-auth/internal/platform/response"
	"github.com/maintainerd/maintainerd-auth/internal/platform/security"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// A locked-out login used to return a bare error, so it rendered as 500 with no
// Retry-After: the client could not tell "back off" from "server broke", and
// every routine lockout looked like a server fault in monitoring.
func TestLoginRateLimitErrorMapping(t *testing.T) {
	t.Run("a lockout is 429 and carries Retry-After", func(t *testing.T) {
		err := loginRateLimitError(errors.New("account is locked for 15m0s"),
			&security.RateLimitConfig{Enabled: true, LockoutDuration: 15 * time.Minute})

		var throttled *apperror.TooManyRequestsError
		require.True(t, errors.As(err, &throttled), "must be typed as a throttle")
		assert.Equal(t, 15*time.Minute, throttled.RetryAfter)

		rec := httptest.NewRecorder()
		resp.HandleServiceError(rec, httptest.NewRequest(http.MethodPost, "/login", nil), "login failed", err)
		assert.Equal(t, http.StatusTooManyRequests, rec.Code)
		assert.Equal(t, "900", rec.Header().Get("Retry-After"))
	})

	t.Run("a limiter outage is 503, not 429", func(t *testing.T) {
		err := loginRateLimitError(security.ErrRateLimiterUnavailable, nil)

		var unavailable *apperror.ServiceUnavailableError
		require.True(t, errors.As(err, &unavailable), "an outage is not the caller's fault")

		rec := httptest.NewRecorder()
		resp.HandleServiceError(rec, httptest.NewRequest(http.MethodPost, "/login", nil), "login failed", err)
		assert.Equal(t, http.StatusServiceUnavailable, rec.Code)
		assert.Empty(t, rec.Header().Get("Retry-After"), "no rate to back off from")
	})

	t.Run("a lockout with no tenant policy falls back to the default duration", func(t *testing.T) {
		err := loginRateLimitError(errors.New("account is locked"), nil)
		var throttled *apperror.TooManyRequestsError
		require.True(t, errors.As(err, &throttled))
		assert.Equal(t, security.AccountLockoutTime, throttled.RetryAfter)
	})
}
