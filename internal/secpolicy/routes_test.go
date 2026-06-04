package secpolicy

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
)

func TestIPRestrictionRuleRoute_ProtectedRoutesRequireAuth(t *testing.T) {
	router := chi.NewRouter()
	IPRestrictionRuleRoute(router, NewIPRestrictionRuleHandler(&mockIPRestrictionRuleService{}), nil, nil)

	tests := []struct {
		method string
		path   string
	}{
		{"GET", "/ip-restriction-rules/"},
		{"GET", "/ip-restriction-rules/uuid-1"},
		{"POST", "/ip-restriction-rules/"},
		{"PUT", "/ip-restriction-rules/uuid-1"},
		{"DELETE", "/ip-restriction-rules/uuid-1"},
		{"PATCH", "/ip-restriction-rules/uuid-1/status"},
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

func TestSecuritySettingRoute_ProtectedRoutesRequireAuth(t *testing.T) {
	router := chi.NewRouter()
	SecuritySettingRoute(router, NewSecuritySettingHandler(&mockSecuritySettingService{}), nil, nil)

	tests := []struct {
		method string
		path   string
	}{
		{"GET", "/security-settings/mfa"},
		{"PUT", "/security-settings/mfa"},
		{"GET", "/security-settings/password"},
		{"PUT", "/security-settings/password"},
		{"GET", "/security-settings/session"},
		{"PUT", "/security-settings/session"},
		{"GET", "/security-settings/threat"},
		{"PUT", "/security-settings/threat"},
		{"GET", "/security-settings/lockout"},
		{"PUT", "/security-settings/lockout"},
		{"GET", "/security-settings/registration"},
		{"PUT", "/security-settings/registration"},
		{"GET", "/security-settings/token"},
		{"PUT", "/security-settings/token"},
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
