package client

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
)

func TestAPIKeyRoute_ProtectedRoutesRequireAuth(t *testing.T) {
	router := chi.NewRouter()
	APIKeyRoute(router, NewAPIKeyHandler(&mockAPIKeyService{}), nil, nil)

	tests := []struct {
		method string
		path   string
	}{
		{"GET", "/api_keys/"},
		{"GET", "/api_keys/uuid-1"},
		{"GET", "/api_keys/uuid-1/config"},
		{"POST", "/api_keys/"},
		{"PUT", "/api_keys/uuid-1"},
		{"PUT", "/api_keys/uuid-1/status"},
		{"DELETE", "/api_keys/uuid-1"},
		{"GET", "/api_keys/uuid-1/apis/"},
		{"POST", "/api_keys/uuid-1/apis/"},
		{"DELETE", "/api_keys/uuid-1/apis/uuid-2"},
		{"GET", "/api_keys/uuid-1/apis/uuid-2/permissions/"},
		{"POST", "/api_keys/uuid-1/apis/uuid-2/permissions/"},
		{"DELETE", "/api_keys/uuid-1/apis/uuid-2/permissions/uuid-3"},
	}

	for _, tt := range tests {
		t.Run(tt.method+" "+tt.path, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)
			assert.Equal(t, http.StatusUnauthorized, w.Code)
		})
	}
}

func TestClientRoute_ProtectedRoutesRequireAuth(t *testing.T) {
	router := chi.NewRouter()
	ClientRoute(router, NewClientHandler(&mockClientService{}), nil, nil)

	tests := []struct {
		method string
		path   string
	}{
		{"GET", "/clients/"},
		{"GET", "/clients/uuid-1"},
		{"GET", "/clients/uuid-1/secret"},
		{"POST", "/clients/uuid-1/rotate-secret"},
		{"GET", "/clients/uuid-1/config"},
		{"POST", "/clients/"},
		{"PUT", "/clients/uuid-1"},
		{"PUT", "/clients/uuid-1/status"},
		{"DELETE", "/clients/uuid-1"},
		{"GET", "/clients/uuid-1/uris"},
		{"POST", "/clients/uuid-1/uris"},
		{"PUT", "/clients/uuid-1/uris/uuid-uri"},
		{"DELETE", "/clients/uuid-1/uris/uuid-uri"},
		{"GET", "/clients/uuid-1/apis"},
		{"POST", "/clients/uuid-1/apis"},
		{"DELETE", "/clients/uuid-1/apis/uuid-2"},
		{"GET", "/clients/uuid-1/apis/uuid-2/permissions"},
		{"POST", "/clients/uuid-1/apis/uuid-2/permissions"},
		{"DELETE", "/clients/uuid-1/apis/uuid-2/permissions/uuid-3"},
	}

	for _, tt := range tests {
		t.Run(tt.method+" "+tt.path, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)
			assert.Equal(t, http.StatusUnauthorized, w.Code)
		})
	}
}
