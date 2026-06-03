package tenant

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestParseTenantMemberUUIDFromRoute_Direct(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		r := withChiParam(httptest.NewRequest(http.MethodGet, "/", nil), "tenant_member_uuid", testResourceUUID.String())
		w := httptest.NewRecorder()

		result, ok := parseTenantMemberUUIDFromRoute(w, r)

		assert.True(t, ok)
		assert.Equal(t, testResourceUUID, result)
	})

	t.Run("missing param", func(t *testing.T) {
		w := httptest.NewRecorder()

		result, ok := parseTenantMemberUUIDFromRoute(w, httptest.NewRequest(http.MethodGet, "/", nil))

		assert.False(t, ok)
		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Equal(t, uuid.Nil, result)
	})
}
