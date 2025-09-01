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

func (h *LoginHandler) LoginPublic(w http.ResponseWriter, r *http.Request) {
	var req dto.AuthRequestDto

	// Validate query parameters
	q := dto.LoginPublicQueryDto{
		ClientID: r.URL.Query().Get("client_id"),
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
	tokenResponse, err := h.loginService.LoginPublic(
		req.Username, req.Password, q.ClientID,
	)
	if err != nil {
		util.Error(w, http.StatusUnauthorized, "Login failed", err.Error())
		return
	}

	// Response
	util.Success(w, tokenResponse, "Login successful")
}

func (h *LoginHandler) LoginPrivate(w http.ResponseWriter, r *http.Request) {
	var req dto.AuthRequestDto

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
	tokenResponse, err := h.loginService.LoginPrivate(
		req.Username, req.Password,
	)
	if err != nil {
		util.Error(w, http.StatusUnauthorized, "Login failed", err.Error())
		return
	}

	// Response
	util.Success(w, tokenResponse, "Login successful")
}
