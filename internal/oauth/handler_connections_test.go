package oauth

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/maintainerd/auth/internal/platform/apperror"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOAuthConnectionsHandler_ListConnections(t *testing.T) {
	t.Run("missing client_id returns 400", func(t *testing.T) {
		h := NewOAuthConnectionsHandler(&mockOAuthConnectionsService{})
		r := httptest.NewRequest(http.MethodGet, "/oauth/connections", nil)
		w := httptest.NewRecorder()
		h.ListConnections(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("service not-found returns 404", func(t *testing.T) {
		svc := &mockOAuthConnectionsService{listFn: func(string) (*OAuthConnectionsResult, error) {
			return nil, apperror.NewNotFound("unknown")
		}}
		h := NewOAuthConnectionsHandler(svc)
		r := httptest.NewRequest(http.MethodGet, "/oauth/connections?client_id=x", nil)
		w := httptest.NewRecorder()
		h.ListConnections(w, r)
		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("success returns 200 with connections", func(t *testing.T) {
		svc := &mockOAuthConnectionsService{listFn: func(string) (*OAuthConnectionsResult, error) {
			return &OAuthConnectionsResult{
				PasswordEnabled: true,
				Connections: []OAuthConnectionInfo{
					{Identifier: "google-idp", DisplayName: "Google", Provider: "google", ProviderType: "social", DisplayOrder: 1},
				},
			}, nil
		}}
		h := NewOAuthConnectionsHandler(svc)
		r := httptest.NewRequest(http.MethodGet, "/oauth/connections?client_id=x", nil)
		w := httptest.NewRecorder()
		h.ListConnections(w, r)

		assert.Equal(t, http.StatusOK, w.Code)
		var body map[string]json.RawMessage
		require.NoError(t, json.NewDecoder(w.Body).Decode(&body))
		var data OAuthConnectionsResponseDTO
		require.NoError(t, json.Unmarshal(body["data"], &data))
		assert.True(t, data.PasswordEnabled)
		require.Len(t, data.Connections, 1)
		assert.Equal(t, "google-idp", data.Connections[0].Identifier)
	})
}
