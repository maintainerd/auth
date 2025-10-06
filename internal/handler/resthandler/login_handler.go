package resthandler

import (
	"encoding/json"
	"net/http"

	validation "github.com/go-ozzo/ozzo-validation/v4"

	"github.com/maintainerd/auth/internal/dto"
	"github.com/maintainerd/auth/internal/service"
	"github.com/maintainerd/auth/internal/util"
)

type LoginHandler struct {
	loginService service.LoginService
}

func NewLoginHandler(
	loginService service.LoginService,
) *LoginHandler {
	return &LoginHandler{
		loginService,
	}
}

func (h *LoginHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req dto.AuthRequestDto

	// Validate query parameters
	q := dto.LoginQueryDto{
		AuthClientID:    r.URL.Query().Get("auth_client_id"),
		AuthContainerID: r.URL.Query().Get("auth_container_id"),
	}

	if err := q.Validate(); err != nil {
		util.Error(w, http.StatusBadRequest, "Validation failed", err.Error())
		return
	}

	// Validate body payload
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		util.Error(w, http.StatusBadRequest, "Invalid request", err.Error())
		return
	}

	if err := req.Validate(); err != nil {
		if ve, ok := err.(validation.Errors); ok {
			util.Error(w, http.StatusBadRequest, "Validation failed", ve)
			return
		}
		util.Error(w, http.StatusBadRequest, "Validation failed", err.Error())
		return
	}

	// Login
	tokenResponse, err := h.loginService.Login(
		req.Username, req.Password, q.AuthClientID, q.AuthContainerID,
	)
	if err != nil {
		util.Error(w, http.StatusUnauthorized, "Login failed", err.Error())
		return
	}

	// Convert DTO to map to avoid import cycle
	authResponse := map[string]interface{}{
		"access_token":  tokenResponse.AccessToken,
		"id_token":      tokenResponse.IDToken,
		"refresh_token": tokenResponse.RefreshToken,
		"expires_in":    tokenResponse.ExpiresIn,
		"token_type":    tokenResponse.TokenType,
		"issued_at":     tokenResponse.IssuedAt,
	}

	// Response with optional cookie delivery
	util.AuthSuccess(w, r, authResponse, "Login successful")
}

func (h *LoginHandler) Logout(w http.ResponseWriter, r *http.Request) {
	// Clear authentication cookies if they exist
	util.ClearAuthCookies(w)

	// Return success response
	util.Success(w, nil, "Logout successful")
}
