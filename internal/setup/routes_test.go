package setup

import (
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
}
