package resthandler

import (
	"encoding/json"
	"net/http"

	"github.com/maintainerd/auth/internal/service"
	"github.com/maintainerd/auth/internal/util"
)

type AuthHandler struct {
	authService service.AuthService
}

func NewAuthHandler(authService service.AuthService) *AuthHandler {
	return &AuthHandler{authService}
}

func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Username           string `json:"username"`
		Password           string `json:"password"`
		ClientID           string `json:"client_id"`
		IdentityProviderID string `json:"identity_provider_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		util.Error(w, http.StatusBadRequest, "Invalid request", err.Error())
		return
	}

	if req.Username == "" || req.Password == "" || req.ClientID == "" || req.IdentityProviderID == "" {
		util.Error(w, http.StatusBadRequest, "Missing required fields")
		return
	}

	if len(req.Password) < 6 {
		util.Error(w, http.StatusBadRequest, "Password must be at least 6 characters")
		return
	}

	var email string
	if util.IsValidEmail(req.Username) {
		email = req.Username
	}

	token, err := h.authService.Register(
		req.Username, email, req.Password, req.ClientID, req.IdentityProviderID,
	)
	if err != nil {
		util.Error(w, http.StatusInternalServerError, "Registration failed", err.Error())
		return
	}

	util.Created(w, map[string]string{"token": token}, "Registration successful")
}
