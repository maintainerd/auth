package user

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	"github.com/maintainerd/maintainerd-auth/internal/platform/security"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// withRealRateLimiter points security's rate limiter at an in-process Redis, so
// CheckRateLimit / RecordFailedAttempt actually count. Without this the limiter
// is nil and every check returns nil.
func withRealRateLimiter(t *testing.T) {
	t.Helper()
	mr := miniredis.RunT(t)
	security.InitRateLimiter(redis.NewClient(&redis.Options{Addr: mr.Addr()}))
	t.Cleanup(func() { security.InitRateLimiter(nil) })
}

// /account is mounted under the global per-IP limiter only, not the strict auth
// rate-limit group. ChangePassword had a throttle and documented exactly why;
// InitiateEmailChange, ChangeUsername and DeleteAccount called
// security.ComparePassword with nothing counting the misses, so an attacker
// holding a stolen access token just guessed against one of those instead — the
// single throttle bought nothing.
func TestAccountHandler_PasswordVerifyingEndpointsAreThrottled(t *testing.T) {
	cases := []struct {
		name   string
		invoke func(*AccountHandler, http.ResponseWriter, *http.Request)
		newReq func(*testing.T) *http.Request
	}{
		{
			name:   "InitiateEmailChange",
			invoke: (*AccountHandler).InitiateEmailChange,
			newReq: func(t *testing.T) *http.Request {
				return withAuthUser(jsonReq(t, http.MethodPost, "/account/email/change",
					map[string]string{"new_email": "new@example.com", "current_password": "wrong"}))
			},
		},
		{
			name:   "ChangeUsername",
			invoke: (*AccountHandler).ChangeUsername,
			newReq: func(t *testing.T) *http.Request {
				return withAuthUser(jsonReq(t, http.MethodPut, "/account/username",
					map[string]string{"new_username": "newname", "current_password": "wrong"}))
			},
		},
		{
			name:   "DeleteAccount",
			invoke: (*AccountHandler).DeleteAccount,
			newReq: func(t *testing.T) *http.Request {
				return withAuthUser(jsonReq(t, http.MethodDelete, "/account",
					map[string]string{"current_password": "wrong"}))
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			withRealRateLimiter(t)

			svc := &mockAccountService{
				initiateEmailChangeFn: func(int64, string, string) error { return errors.New("invalid current password") },
				changeUsernameFn:      func(int64, string, string) error { return errors.New("invalid current password") },
				deleteAccountFn:       func(int64, string) error { return errors.New("invalid current password") },
			}
			h := NewAccountHandler(svc, &mockSessionService{}, nil)

			// security.MaxLoginAttempts wrong guesses are tolerated; the next one
			// must be refused without ever reaching the password comparison.
			for i := 0; i < security.MaxLoginAttempts; i++ {
				w := httptest.NewRecorder()
				tc.invoke(h, w, tc.newReq(t))
				require.NotEqual(t, http.StatusTooManyRequests, w.Code, "attempt %d should still be allowed", i+1)
			}

			w := httptest.NewRecorder()
			tc.invoke(h, w, tc.newReq(t))
			assert.Equal(t, http.StatusTooManyRequests, w.Code)
		})
	}
}

// One key for all four endpoints, so the budget cannot be multiplied by
// rotating between them.
func TestAccountHandler_PasswordThrottleBudgetIsShared(t *testing.T) {
	withRealRateLimiter(t)

	svc := &mockAccountService{
		initiateEmailChangeFn: func(int64, string, string) error { return errors.New("invalid current password") },
		changeUsernameFn:      func(int64, string, string) error { return errors.New("invalid current password") },
		deleteAccountFn:       func(int64, string) error { return errors.New("invalid current password") },
		changePasswordFn: func(int64, string, string, *uuid.UUID) (*ChangePasswordResponseDTO, error) {
			return nil, errors.New("invalid current password")
		},
	}
	h := NewAccountHandler(svc, &mockSessionService{}, nil)

	// Spread the allowance across every endpoint that verifies a password.
	burn := []func(){
		func() {
			h.InitiateEmailChange(httptest.NewRecorder(), withAuthUser(jsonReq(t, http.MethodPost, "/account/email/change",
				map[string]string{"new_email": "new@example.com", "current_password": "wrong"})))
		},
		func() {
			h.ChangeUsername(httptest.NewRecorder(), withAuthUser(jsonReq(t, http.MethodPut, "/account/username",
				map[string]string{"new_username": "newname", "current_password": "wrong"})))
		},
		func() {
			h.DeleteAccount(httptest.NewRecorder(), withAuthUser(jsonReq(t, http.MethodDelete, "/account",
				map[string]string{"current_password": "wrong"})))
		},
		func() {
			h.ChangePassword(httptest.NewRecorder(), withAuthUser(jsonReq(t, http.MethodPut, "/account/password",
				map[string]string{"current_password": "wrong", "new_password": "Whatever1!"})))
		},
	}
	for i := 0; i < security.MaxLoginAttempts; i++ {
		burn[i%len(burn)]()
	}

	// A different endpoint from the last one used must still be locked out.
	w := httptest.NewRecorder()
	h.ChangeUsername(w, withAuthUser(jsonReq(t, http.MethodPut, "/account/username",
		map[string]string{"new_username": "newname", "current_password": "wrong"})))
	assert.Equal(t, http.StatusTooManyRequests, w.Code)
}

// Feeding these misses into the LOGIN lockout key would let a stolen access
// token lock the victim out of signing in at all — a confidentiality problem
// turned into a denial of service.
func TestAccountPasswordThrottleKeyIsNotTheLoginLockoutKey(t *testing.T) {
	key := accountPasswordThrottleKey(testUserUUID)
	assert.NotEqual(t, security.RateLimitKey(testUserUUID.String(), "login"), key)
	assert.Contains(t, key, testUserUUID.String())
}
