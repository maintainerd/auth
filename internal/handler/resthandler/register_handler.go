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

func (h *RegisterHandler) RegisterPublic(w http.ResponseWriter, r *http.Request) {
	var req dto.AuthRequestDto

	// Validate query parameters
	q := dto.RegisterPublicQueryDto{
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

	// Register
	tokenResponse, err := h.registerService.RegisterPublic(
		req.Username, req.Password, q.ClientID,
	)
	if err != nil {
		util.Error(w, http.StatusInternalServerError, "Registration failed", err.Error())
		return
	}

	// Response
	util.Created(w, tokenResponse, "Registration successful")
}

func (h *RegisterHandler) RegisterPublicInvite(w http.ResponseWriter, r *http.Request) {
	var req dto.AuthRequestDto

	// Validate query parameters
	q := dto.RegisterPublicInviteQueryDto{
		ClientID:    r.URL.Query().Get("client_id"),
		InviteToken: r.URL.Query().Get("invite_token"),
		Expires:     r.URL.Query().Get("expires"),
		Sig:         r.URL.Query().Get("sig"),
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
	tokenResponse, err := h.registerService.RegisterPublicInvite(
		req.Username,
		req.Password,
		q.ClientID,
		q.InviteToken,
	)
	if err != nil {
		util.Error(w, http.StatusInternalServerError, "Registration failed", err.Error())
		return
	}

	// Response
	util.Created(w, tokenResponse, "Registration successful")
}

func (h *RegisterHandler) RegisterPrivate(w http.ResponseWriter, r *http.Request) {
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

	// Register
	tokenResponse, err := h.registerService.RegisterPrivate(
		req.Username, req.Password,
	)
	if err != nil {
		util.Error(w, http.StatusInternalServerError, "Registration failed", err.Error())
		return
	}

	// Response
	util.Created(w, tokenResponse, "Registration successful")
}

func (h *RegisterHandler) RegisterPrivateInvite(w http.ResponseWriter, r *http.Request) {
	var req dto.AuthRequestDto

	// Validate query parameters
	q := dto.RegisterPrivateQueryDto{
		InviteToken: r.URL.Query().Get("invite_token"),
		Expires:     r.URL.Query().Get("expires"),
		Sig:         r.URL.Query().Get("sig"),
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
	tokenResponse, err := h.registerService.RegisterPrivateInvite(
		req.Username, req.Password, q.InviteToken,
	)
	if err != nil {
		util.Error(w, http.StatusInternalServerError, "Registration failed", err.Error())
		return
	}

	// Response
	util.Created(w, tokenResponse, "Registration successful")
}
