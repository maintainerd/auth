package branding

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
)

func TestBrandingRoute_ProtectedRoutesRequireAuth(t *testing.T) {
	router := chi.NewRouter()
	BrandingRoute(router, NewBrandingHandler(&mockBrandingService{}), nil, nil)

	for _, tc := range []struct {
		method string
		path   string
	}{
		{http.MethodGet, "/branding/"},
		{http.MethodPut, "/branding/"},
	} {
		t.Run(tc.method+" "+tc.path, func(t *testing.T) {
			w := httptest.NewRecorder()
			router.ServeHTTP(w, httptest.NewRequest(tc.method, tc.path, nil))
			// JWT middleware returns 401 without a valid token
			assert.Equal(t, http.StatusUnauthorized, w.Code)
		})
	}
}

func TestEmailTemplateRoute_ProtectedRoutesRequireAuth(t *testing.T) {
	router := chi.NewRouter()
	EmailTemplateRoute(router, NewEmailTemplateHandler(&mockEmailTemplateService{}), nil, nil)

	for _, tc := range []struct {
		method string
		path   string
	}{
		{http.MethodGet, "/email_templates/"},
		{http.MethodGet, "/email_templates/00000000-0000-0000-0000-000000000099"},
		{http.MethodPost, "/email_templates/"},
		{http.MethodPut, "/email_templates/00000000-0000-0000-0000-000000000099"},
		{http.MethodDelete, "/email_templates/00000000-0000-0000-0000-000000000099"},
		{http.MethodPatch, "/email_templates/00000000-0000-0000-0000-000000000099/status"},
	} {
		t.Run(tc.method+" "+tc.path, func(t *testing.T) {
			w := httptest.NewRecorder()
			router.ServeHTTP(w, httptest.NewRequest(tc.method, tc.path, nil))
			assert.Equal(t, http.StatusUnauthorized, w.Code)
		})
	}
}


func TestSMSTemplateRoute_ProtectedRoutesRequireAuth(t *testing.T) {
	router := chi.NewRouter()
	SMSTemplateRoute(router, NewSMSTemplateHandler(&mockSMSTemplateService{}), nil, nil)

	for _, tc := range []struct {
		method string
		path   string
	}{
		{http.MethodGet, "/sms_templates/"},
		{http.MethodGet, "/sms_templates/00000000-0000-0000-0000-000000000099"},
		{http.MethodPost, "/sms_templates/"},
		{http.MethodPut, "/sms_templates/00000000-0000-0000-0000-000000000099"},
		{http.MethodDelete, "/sms_templates/00000000-0000-0000-0000-000000000099"},
		{http.MethodPatch, "/sms_templates/00000000-0000-0000-0000-000000000099/status"},
	} {
		t.Run(tc.method+" "+tc.path, func(t *testing.T) {
			w := httptest.NewRecorder()
			router.ServeHTTP(w, httptest.NewRequest(tc.method, tc.path, nil))
			assert.Equal(t, http.StatusUnauthorized, w.Code)
		})
	}
}
