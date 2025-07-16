package resthandler

import (
	"encoding/json"
	"net/http"

	"github.com/maintainerd/auth/internal/service"
	"github.com/maintainerd/auth/internal/util"
	"github.com/maintainerd/auth/internal/validator"
)

type AuthHandler struct {
	authService service.AuthService
}

func NewAuthHandler(authService service.AuthService) *AuthHandler {
	return &AuthHandler{authService}
}

type RegisterRequest struct {
	Username           string `json:"username"`
	Password           string `json:"password"`
	ClientID           string `json:"client_id"`
	IdentityProviderID string `json:"identity_provider_id"`
}

func (r RegisterRequest) Validate() error {
	return validator.ValidateStruct(
		validator.Field(r.Username, validator.Required(), validator.MinLength(3), validator.MaxLength(50)),
		validator.Field(r.Password, validator.Required(), validator.MinLength(6), validator.MaxLength(100)),
		validator.Field(r.ClientID, validator.Required()),
		validator.Field(r.IdentityProviderID, validator.Required()),
	)
}

func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req RegisterRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		util.Error(w, http.StatusBadRequest, "Invalid request", err.Error())
		return
	}

	if err := req.Validate(); err != nil {
		util.Error(w, http.StatusBadRequest, "Validation failed", err.Error())
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
