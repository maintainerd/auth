package authn

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/maintainerd/auth/internal/platform/security"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func evJSONReq(t *testing.T, method, url string, body any) *http.Request {
	t.Helper()
	var buf bytes.Buffer
	if body != nil {
		require.NoError(t, json.NewEncoder(&buf).Encode(body))
	}
	r := httptest.NewRequest(method, url, &buf)
	r.Header.Set("Content-Type", "application/json")
	return r
}

func TestEmailVerificationHandler_SendVerificationEmailPublic(t *testing.T) {
	t.Run("missing client_id returns 400", func(t *testing.T) {
		r := withSecurityCtx(httptest.NewRequest(http.MethodPost, "/email-verification/send?provider_id=idp", nil))
		w := httptest.NewRecorder()
		NewEmailVerificationHandler(&mockEmailVerificationService{}).SendVerificationEmailPublic(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("invalid JSON returns 400", func(t *testing.T) {
		r := withSecurityCtx(httptest.NewRequest(http.MethodPost, "/email-verification/send?client_id=app&provider_id=idp",
			bytes.NewBufferString(`{bad`)))
		r.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		NewEmailVerificationHandler(&mockEmailVerificationService{}).SendVerificationEmailPublic(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("validation error returns 400", func(t *testing.T) {
		r := withSecurityCtx(evJSONReq(t, http.MethodPost, "/email-verification/send?client_id=app&provider_id=idp",
			map[string]string{"email": ""}))
		w := httptest.NewRecorder()
		NewEmailVerificationHandler(&mockEmailVerificationService{}).SendVerificationEmailPublic(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("service error returns 500", func(t *testing.T) {
		svc := &mockEmailVerificationService{
			sendVerificationEmailFn: func(ctx context.Context, email string, clientID, providerID *string) (*SendEmailVerificationResponseDTO, error) {
				return nil, errors.New("db error")
			},
		}
		r := withSecurityCtx(evJSONReq(t, http.MethodPost, "/email-verification/send?client_id=app&provider_id=idp",
			map[string]string{"email": "user@example.com"}))
		w := httptest.NewRecorder()
		NewEmailVerificationHandler(svc).SendVerificationEmailPublic(w, r)
		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})

	t.Run("success returns 200", func(t *testing.T) {
		svc := &mockEmailVerificationService{
			sendVerificationEmailFn: func(ctx context.Context, email string, clientID, providerID *string) (*SendEmailVerificationResponseDTO, error) {
				return &SendEmailVerificationResponseDTO{Success: true}, nil
			},
		}
		r := withSecurityCtx(evJSONReq(t, http.MethodPost, "/email-verification/send?client_id=app&provider_id=idp",
			map[string]string{"email": "user@example.com"}))
		w := httptest.NewRecorder()
		NewEmailVerificationHandler(svc).SendVerificationEmailPublic(w, r)
		assert.Equal(t, http.StatusOK, w.Code)
	})
}

func TestEmailVerificationHandler_SendVerificationEmail(t *testing.T) {
	t.Run("invalid JSON returns 400", func(t *testing.T) {
		r := withSecurityCtx(httptest.NewRequest(http.MethodPost, "/email-verification/send",
			bytes.NewBufferString(`{bad`)))
		r.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		NewEmailVerificationHandler(&mockEmailVerificationService{}).SendVerificationEmail(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("success without params", func(t *testing.T) {
		svc := &mockEmailVerificationService{
			sendVerificationEmailFn: func(ctx context.Context, email string, clientID, providerID *string) (*SendEmailVerificationResponseDTO, error) {
				return &SendEmailVerificationResponseDTO{Success: true}, nil
			},
		}
		r := withSecurityCtx(evJSONReq(t, http.MethodPost, "/email-verification/send",
			map[string]string{"email": "user@example.com"}))
		w := httptest.NewRecorder()
		NewEmailVerificationHandler(svc).SendVerificationEmail(w, r)
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("success with optional params", func(t *testing.T) {
		svc := &mockEmailVerificationService{
			sendVerificationEmailFn: func(ctx context.Context, email string, clientID, providerID *string) (*SendEmailVerificationResponseDTO, error) {
				require.NotNil(t, clientID)
				require.NotNil(t, providerID)
				assert.Equal(t, "app", *clientID)
				assert.Equal(t, "idp", *providerID)
				return &SendEmailVerificationResponseDTO{Success: true}, nil
			},
		}
		r := withSecurityCtx(evJSONReq(t, http.MethodPost, "/email-verification/send?client_id=app&provider_id=idp",
			map[string]string{"email": "user@example.com"}))
		w := httptest.NewRecorder()
		NewEmailVerificationHandler(svc).SendVerificationEmail(w, r)
		assert.Equal(t, http.StatusOK, w.Code)
	})
}

func TestEmailVerificationHandler_VerifyEmail(t *testing.T) {
	t.Run("invalid JSON returns 400", func(t *testing.T) {
		r := withSecurityCtx(httptest.NewRequest(http.MethodPost, "/email-verification/verify",
			bytes.NewBufferString(`{bad`)))
		r.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		NewEmailVerificationHandler(&mockEmailVerificationService{}).VerifyEmail(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("validation error returns 400", func(t *testing.T) {
		r := withSecurityCtx(evJSONReq(t, http.MethodPost, "/email-verification/verify",
			map[string]string{"email": "", "otp": ""}))
		w := httptest.NewRecorder()
		NewEmailVerificationHandler(&mockEmailVerificationService{}).VerifyEmail(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("service error returns 500", func(t *testing.T) {
		svc := &mockEmailVerificationService{
			verifyEmailFn: func(ctx context.Context, email, otp string) (*VerifyEmailResponseDTO, error) {
				return nil, errors.New("invalid code")
			},
		}
		r := withSecurityCtx(evJSONReq(t, http.MethodPost, "/email-verification/verify",
			map[string]string{"email": "user@example.com", "otp": "123456"}))
		w := httptest.NewRecorder()
		NewEmailVerificationHandler(svc).VerifyEmail(w, r)
		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})

	t.Run("success returns 200", func(t *testing.T) {
		svc := &mockEmailVerificationService{
			verifyEmailFn: func(ctx context.Context, email, otp string) (*VerifyEmailResponseDTO, error) {
				return &VerifyEmailResponseDTO{Message: "verified", Success: true}, nil
			},
		}
		r := withSecurityCtx(evJSONReq(t, http.MethodPost, "/email-verification/verify",
			map[string]string{"email": "user@example.com", "otp": "123456"}))
		w := httptest.NewRecorder()
		NewEmailVerificationHandler(svc).VerifyEmail(w, r)
		assert.Equal(t, http.StatusOK, w.Code)
	})
}

func lockedRateLimiterEV(t *testing.T, identifier string) func() {
	t.Helper()
	mr, err := miniredis.Run()
	require.NoError(t, err)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	security.InitRateLimiter(rdb)
	require.NoError(t, mr.Set("rl:lock:"+identifier, "1"))
	return func() {
		security.InitRateLimiter(nil)
		_ = rdb.Close()
		mr.Close()
	}
}

func TestEmailVerificationHandler_HandleSendVerification_RateLimited(t *testing.T) {
	email := "ratelimited-ev@example.com"
	cleanup := lockedRateLimiterEV(t, email)
	defer cleanup()

	r := withSecurityCtx(evJSONReq(t, http.MethodPost, "/email-verification/send?client_id=app&provider_id=idp",
		map[string]string{"email": email}))
	w := httptest.NewRecorder()
	NewEmailVerificationHandler(&mockEmailVerificationService{}).SendVerificationEmailPublic(w, r)
	assert.Equal(t, http.StatusTooManyRequests, w.Code)
}

func TestEmailVerificationHandler_VerifyEmail_RateLimited(t *testing.T) {
	email := "ratelimited-verify@example.com"
	cleanup := lockedRateLimiterEV(t, email)
	defer cleanup()

	r := withSecurityCtx(evJSONReq(t, http.MethodPost, "/email-verification/verify",
		map[string]string{"email": email, "otp": "123456"}))
	w := httptest.NewRecorder()
	NewEmailVerificationHandler(&mockEmailVerificationService{}).VerifyEmail(w, r)
	assert.Equal(t, http.StatusTooManyRequests, w.Code)
}
