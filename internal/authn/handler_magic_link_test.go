package authn

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/maintainerd/auth/internal/platform/apperror"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockMagicLinkService struct {
	sendMagicLinkFn      func(email string, clientID, providerID *string, isInternal bool) (*SendMagicLinkResponseDTO, error)
	loginWithMagicLinkFn func(token, clientID, providerID string) (*LoginResponseDTO, error)
}

func (m *mockMagicLinkService) SendMagicLink(_ context.Context, email string, clientID, providerID *string, isInternal bool) (*SendMagicLinkResponseDTO, error) {
	if m.sendMagicLinkFn != nil {
		return m.sendMagicLinkFn(email, clientID, providerID, isInternal)
	}
	return nil, nil
}

func (m *mockMagicLinkService) LoginWithMagicLink(_ context.Context, token, clientID, providerID string) (*LoginResponseDTO, error) {
	if m.loginWithMagicLinkFn != nil {
		return m.loginWithMagicLinkFn(token, clientID, providerID)
	}
	return nil, nil
}

func magicLinkJSONReq(t *testing.T, method, url string, body any) *http.Request {
	t.Helper()
	var buf bytes.Buffer
	if body != nil {
		require.NoError(t, json.NewEncoder(&buf).Encode(body))
	}
	r := httptest.NewRequest(method, url, &buf)
	r.Header.Set("Content-Type", "application/json")
	return r
}

func TestMagicLinkHandler_SendMagicLinkPublic(t *testing.T) {
	t.Run("missing client_id returns 400", func(t *testing.T) {
		r := withSecurityCtx(httptest.NewRequest(http.MethodPost, "/magic-link/send?provider_id=idp", nil))
		w := httptest.NewRecorder()
		NewMagicLinkHandler(&mockMagicLinkService{}).SendMagicLinkPublic(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("missing provider_id returns 400", func(t *testing.T) {
		r := withSecurityCtx(httptest.NewRequest(http.MethodPost, "/magic-link/send?client_id=app", nil))
		w := httptest.NewRecorder()
		NewMagicLinkHandler(&mockMagicLinkService{}).SendMagicLinkPublic(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("invalid JSON returns 400", func(t *testing.T) {
		r := withSecurityCtx(httptest.NewRequest(http.MethodPost, "/magic-link/send?client_id=app&provider_id=idp",
			bytes.NewBufferString(`{bad json}`)))
		r.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		NewMagicLinkHandler(&mockMagicLinkService{}).SendMagicLinkPublic(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("validation error returns 400", func(t *testing.T) {
		r := withSecurityCtx(magicLinkJSONReq(t, http.MethodPost, "/magic-link/send?client_id=app&provider_id=idp",
			map[string]string{"email": ""}))
		w := httptest.NewRecorder()
		NewMagicLinkHandler(&mockMagicLinkService{}).SendMagicLinkPublic(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("service error returns 500", func(t *testing.T) {
		svc := &mockMagicLinkService{
			sendMagicLinkFn: func(email string, clientID, providerID *string, isInternal bool) (*SendMagicLinkResponseDTO, error) {
				return nil, errors.New("db error")
			},
		}
		r := withSecurityCtx(magicLinkJSONReq(t, http.MethodPost, "/magic-link/send?client_id=app&provider_id=idp",
			map[string]string{"email": "user@example.com"}))
		w := httptest.NewRecorder()
		NewMagicLinkHandler(svc).SendMagicLinkPublic(w, r)
		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})

	t.Run("success returns 200", func(t *testing.T) {
		svc := &mockMagicLinkService{
			sendMagicLinkFn: func(email string, clientID, providerID *string, isInternal bool) (*SendMagicLinkResponseDTO, error) {
				return &SendMagicLinkResponseDTO{Message: "sent", Success: true}, nil
			},
		}
		r := withSecurityCtx(magicLinkJSONReq(t, http.MethodPost, "/magic-link/send?client_id=app&provider_id=idp",
			map[string]string{"email": "user@example.com"}))
		w := httptest.NewRecorder()
		NewMagicLinkHandler(svc).SendMagicLinkPublic(w, r)
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("service returns unauthorized error", func(t *testing.T) {
		svc := &mockMagicLinkService{
			sendMagicLinkFn: func(email string, clientID, providerID *string, isInternal bool) (*SendMagicLinkResponseDTO, error) {
				return nil, apperror.NewUnauthorized("invalid")
			},
		}
		r := withSecurityCtx(magicLinkJSONReq(t, http.MethodPost, "/magic-link/send?client_id=app&provider_id=idp",
			map[string]string{"email": "user@example.com"}))
		w := httptest.NewRecorder()
		NewMagicLinkHandler(svc).SendMagicLinkPublic(w, r)
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})
}

func TestMagicLinkHandler_SendMagicLink(t *testing.T) {
	t.Run("invalid JSON returns 400", func(t *testing.T) {
		r := withSecurityCtx(httptest.NewRequest(http.MethodPost, "/magic-link/send",
			bytes.NewBufferString(`{bad json}`)))
		r.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		NewMagicLinkHandler(&mockMagicLinkService{}).SendMagicLink(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("success without params", func(t *testing.T) {
		svc := &mockMagicLinkService{
			sendMagicLinkFn: func(email string, clientID, providerID *string, isInternal bool) (*SendMagicLinkResponseDTO, error) {
				assert.True(t, isInternal)
				return &SendMagicLinkResponseDTO{Success: true}, nil
			},
		}
		r := withSecurityCtx(magicLinkJSONReq(t, http.MethodPost, "/magic-link/send",
			map[string]string{"email": "user@example.com"}))
		w := httptest.NewRecorder()
		NewMagicLinkHandler(svc).SendMagicLink(w, r)
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("success with params", func(t *testing.T) {
		svc := &mockMagicLinkService{
			sendMagicLinkFn: func(email string, clientID, providerID *string, isInternal bool) (*SendMagicLinkResponseDTO, error) {
				assert.NotNil(t, clientID)
				assert.NotNil(t, providerID)
				return &SendMagicLinkResponseDTO{Success: true}, nil
			},
		}
		r := withSecurityCtx(magicLinkJSONReq(t, http.MethodPost, "/magic-link/send?client_id=app&provider_id=idp",
			map[string]string{"email": "user@example.com"}))
		w := httptest.NewRecorder()
		NewMagicLinkHandler(svc).SendMagicLink(w, r)
		assert.Equal(t, http.StatusOK, w.Code)
	})
}

func TestMagicLinkHandler_VerifyMagicLink(t *testing.T) {
	t.Run("missing client_id returns 400", func(t *testing.T) {
		r := withSecurityCtx(httptest.NewRequest(http.MethodPost, "/magic-link/verify?provider_id=idp", nil))
		w := httptest.NewRecorder()
		NewMagicLinkHandler(&mockMagicLinkService{}).VerifyMagicLink(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("invalid JSON returns 400", func(t *testing.T) {
		r := withSecurityCtx(httptest.NewRequest(http.MethodPost, "/magic-link/verify?client_id=app&provider_id=idp",
			bytes.NewBufferString(`{bad json}`)))
		r.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		NewMagicLinkHandler(&mockMagicLinkService{}).VerifyMagicLink(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("validation error returns 400", func(t *testing.T) {
		r := withSecurityCtx(magicLinkJSONReq(t, http.MethodPost, "/magic-link/verify?client_id=app&provider_id=idp",
			map[string]string{"token": ""}))
		w := httptest.NewRecorder()
		NewMagicLinkHandler(&mockMagicLinkService{}).VerifyMagicLink(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("service error returns 500", func(t *testing.T) {
		svc := &mockMagicLinkService{
			loginWithMagicLinkFn: func(token, clientID, providerID string) (*LoginResponseDTO, error) {
				return nil, errors.New("expired")
			},
		}
		r := withSecurityCtx(magicLinkJSONReq(t, http.MethodPost, "/magic-link/verify?client_id=app&provider_id=idp",
			map[string]string{"token": "abcdef1234567890"}))
		w := httptest.NewRecorder()
		NewMagicLinkHandler(svc).VerifyMagicLink(w, r)
		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})

	t.Run("success returns 200", func(t *testing.T) {
		svc := &mockMagicLinkService{
			loginWithMagicLinkFn: func(token, clientID, providerID string) (*LoginResponseDTO, error) {
				return &LoginResponseDTO{AccessToken: "at"}, nil
			},
		}
		r := withSecurityCtx(magicLinkJSONReq(t, http.MethodPost, "/magic-link/verify?client_id=app&provider_id=idp",
			map[string]string{"token": "abcdef1234567890"}))
		w := httptest.NewRecorder()
		NewMagicLinkHandler(svc).VerifyMagicLink(w, r)
		assert.Equal(t, http.StatusOK, w.Code)
	})
}
