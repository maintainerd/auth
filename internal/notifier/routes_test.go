package notifier

import (
	"net/http"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
)

func TestEmailConfigRouteMountsEndpoints(t *testing.T) {
	router := chi.NewRouter()
	EmailConfigRoute(router, NewEmailConfigHandler(&mockEmailConfigService{}), nil, nil)

	tests := []struct {
		method string
		path   string
	}{
		{http.MethodGet, "/email-config/"},
		{http.MethodPut, "/email-config/"},
	}

	for _, tt := range tests {
		t.Run(tt.method+" "+tt.path, func(t *testing.T) {
			match := chi.NewRouteContext()
			assert.True(t, router.Match(match, tt.method, tt.path))
		})
	}
}

func TestSMSConfigRouteMountsEndpoints(t *testing.T) {
	router := chi.NewRouter()
	SMSConfigRoute(router, NewSMSConfigHandler(&mockSMSConfigService{}), nil, nil)

	tests := []struct {
		method string
		path   string
	}{
		{http.MethodGet, "/sms-config/"},
		{http.MethodPut, "/sms-config/"},
	}

	for _, tt := range tests {
		t.Run(tt.method+" "+tt.path, func(t *testing.T) {
			match := chi.NewRouteContext()
			assert.True(t, router.Match(match, tt.method, tt.path))
		})
	}
}
