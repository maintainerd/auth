package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEnforceJSONContentType_GetSkipsCheck(t *testing.T) {
	handler := EnforceJSONContentType(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	r := httptest.NewRequest(http.MethodGet, "/tenants", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestEnforceJSONContentType_HeadDeleteOptionsSkipsCheck(t *testing.T) {
	methods := []string{http.MethodHead, http.MethodDelete, http.MethodOptions}
	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			handler := EnforceJSONContentType(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			}))
			r := httptest.NewRequest(method, "/tenants", nil)
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, r)
			assert.Equal(t, http.StatusOK, w.Code)
		})
	}
}

func TestEnforceJSONContentType_PostJSONPasses(t *testing.T) {
	handler := EnforceJSONContentType(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	r := httptest.NewRequest(http.MethodPost, "/tenants", nil)
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestEnforceJSONContentType_PostJSONCharsetPasses(t *testing.T) {
	handler := EnforceJSONContentType(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	r := httptest.NewRequest(http.MethodPost, "/tenants", nil)
	r.Header.Set("Content-Type", "application/json; charset=utf-8")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestEnforceJSONContentType_PostNoContentType(t *testing.T) {
	handler := EnforceJSONContentType(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler must not be called")
	}))

	r := httptest.NewRequest(http.MethodPost, "/tenants", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)

	assert.Equal(t, http.StatusUnsupportedMediaType, w.Code)
}

func TestEnforceJSONContentType_PostTextPlain(t *testing.T) {
	handler := EnforceJSONContentType(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler must not be called")
	}))

	r := httptest.NewRequest(http.MethodPost, "/tenants", nil)
	r.Header.Set("Content-Type", "text/plain")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)

	assert.Equal(t, http.StatusUnsupportedMediaType, w.Code)
}

func TestEnforceJSONContentType_OAuthFormEncodedPathsExempt(t *testing.T) {
	oauthPaths := []string{
		"/oauth/token",
		"/oauth/revoke",
		"/oauth/introspect",
		"/oauth/par",
		"/oauth/end_session",
	}
	for _, path := range oauthPaths {
		t.Run(path, func(t *testing.T) {
			handler := EnforceJSONContentType(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			}))
			r := httptest.NewRequest(http.MethodPost, path, nil)
			// No Content-Type set — should still pass.
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, r)
			assert.Equal(t, http.StatusOK, w.Code)
		})
	}
}

func TestEnforceJSONContentType_PutJSONPasses(t *testing.T) {
	handler := EnforceJSONContentType(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	r := httptest.NewRequest(http.MethodPut, "/tenants", nil)
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestEnforceJSONContentType_PatchJSONPasses(t *testing.T) {
	handler := EnforceJSONContentType(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	r := httptest.NewRequest(http.MethodPatch, "/tenants", nil)
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestIsFormEncodedPath_NonOAuthPath(t *testing.T) {
	assert.False(t, isFormEncodedPath("/tenants"))
	assert.False(t, isFormEncodedPath("/login"))
}
