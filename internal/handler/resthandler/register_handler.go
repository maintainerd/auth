package resthandler

import (
	"encoding/json"
	"net/http"

	validation "github.com/go-ozzo/ozzo-validation/v4"

	"github.com/maintainerd/auth/internal/dto"
	"github.com/maintainerd/auth/internal/service"
	"github.com/maintainerd/auth/internal/util"
)

type RegisterHandler struct {
	registerService service.RegisterService
}

func NewRegisterHandler(
	registerService service.RegisterService,
) *RegisterHandler {
	return &RegisterHandler{
		registerService,
	}
}

func (h *RegisterHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req dto.AuthRequestDto

	// Validate query parameters
	q := dto.RegisterQueryDto{
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

	// Register
	tokenResponse, err := h.registerService.Register(
		req.Username, req.Password, q.AuthClientID, q.AuthContainerID,
	)
	if err != nil {
		util.Error(w, http.StatusInternalServerError, "Registration failed", err.Error())
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
	util.AuthCreated(w, r, authResponse, "Registration successful")
}

func (h *RegisterHandler) RegisterInvite(w http.ResponseWriter, r *http.Request) {
	var req dto.AuthRequestDto

	// Validate query parameters
	q := dto.RegisterInviteQueryDto{
		AuthClientID:    r.URL.Query().Get("auth_client_id"),
		AuthContainerID: r.URL.Query().Get("auth_container_id"),
		InviteToken:     r.URL.Query().Get("invite_token"),
		Expires:         r.URL.Query().Get("expires"),
		Sig:             r.URL.Query().Get("sig"),
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

	// Register
	tokenResponse, err := h.registerService.RegisterInvite(
		req.Username,
		req.Password,
		q.AuthClientID,
		q.AuthContainerID,
		q.InviteToken,
	)
	if err != nil {
		util.Error(w, http.StatusInternalServerError, "Registration failed", err.Error())
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
	util.AuthCreated(w, r, authResponse, "Registration successful")
}
