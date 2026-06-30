package oauth

import (
	"net/http"

	resp "github.com/maintainerd/auth/internal/platform/response"
)

// OAuthConnectionsHandler serves the public login-page connections endpoint.
type OAuthConnectionsHandler struct {
	connectionsService OAuthConnectionsService
}

func NewOAuthConnectionsHandler(connectionsService OAuthConnectionsService) *OAuthConnectionsHandler {
	return &OAuthConnectionsHandler{connectionsService: connectionsService}
}

// ListConnections handles GET /oauth/connections?client_id=…&registration_flow=…
// (public,
// unauthenticated). It returns the login options for a client — whether
// username/password is available and the connected OAuth2 providers — so the
// hosted identity app can render its login page. Provider secrets/config are
// never returned.
func (h *OAuthConnectionsHandler) ListConnections(w http.ResponseWriter, r *http.Request) {
	clientID := r.URL.Query().Get("client_id")
	if clientID == "" {
		resp.Error(w, http.StatusBadRequest, "client_id is required")
		return
	}

	result, err := h.connectionsService.ListConnections(r.Context(), clientID)
	if err != nil {
		resp.HandleServiceError(w, r, "Failed to load connections", err)
		return
	}

	dto := OAuthConnectionsResponseDTO{
		PasswordEnabled:     result.PasswordEnabled,
		RegistrationEnabled: result.RegistrationEnabled,
		Branding:            result.Branding,
		Connections:         make([]OAuthConnectionDTO, 0, len(result.Connections)),
	}
	for _, c := range result.Connections {
		dto.Connections = append(dto.Connections, OAuthConnectionDTO(c))
	}

	resp.Success(w, dto, "Connections retrieved")
}
