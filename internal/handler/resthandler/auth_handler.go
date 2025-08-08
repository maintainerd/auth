package resthandler

import (
	"encoding/json"
	"net/http"

	validation "github.com/go-ozzo/ozzo-validation/v4"

	"github.com/maintainerd/auth/internal/dto"
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
	var req dto.AuthRequest

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

	var email string
	if util.IsValidEmail(req.Username) {
		email = req.Username
	}

	tokenResponse, err := h.authService.Register(
		req.Username, email, req.Password, req.ClientID, req.IdentityProviderID,
	)
	if err != nil {
		util.Error(w, http.StatusInternalServerError, "Registration failed", err.Error())
		return
	}

	util.Created(w, tokenResponse, "Registration successful")
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req dto.AuthRequest

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

	tokenResponse, err := h.authService.Login(
		req.Username, req.Password, req.ClientID, req.IdentityProviderID,
	)
	if err != nil {
		util.Error(w, http.StatusUnauthorized, "Login failed", err.Error())
		return
	}

	util.Success(w, tokenResponse, "Login successful")
}
