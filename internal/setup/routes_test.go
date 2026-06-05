package setup

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
)

func TestSetupRoute(t *testing.T) {
	t.Run("registers setup status route", func(t *testing.T) {
		router := chi.NewRouter()
		SetupRoute(router, NewSetupHandler(&mockSetupService{
			getSetupStatusFn: func() (*SetupStatusResponseDTO, error) {
				return &SetupStatusResponseDTO{}, nil
			},
		}))

		w := httptest.NewRecorder()
		router.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/setup/status", nil))

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("registers control service route", func(t *testing.T) {
		router := chi.NewRouter()
		SetupRoute(router, NewSetupHandler(&mockSetupService{
			registerControlServiceFn: func(RegisterControlServiceRequestDTO) (*RegisterControlServiceResponseDTO, error) {
				return &RegisterControlServiceResponseDTO{Name: "core"}, nil
			},
		}))

		w := httptest.NewRecorder()
		body := bytes.NewBufferString(`{"name":"core","display_name":"Core"}`)
		router.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/setup/register-control-service", body))

		assert.Equal(t, http.StatusCreated, w.Code)
	})
}
