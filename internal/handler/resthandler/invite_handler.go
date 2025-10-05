package resthandler

import (
	"encoding/json"
	"net/http"

	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/maintainerd/auth/internal/dto"
	"github.com/maintainerd/auth/internal/middleware"
	"github.com/maintainerd/auth/internal/model"
	"github.com/maintainerd/auth/internal/service"
	"github.com/maintainerd/auth/internal/util"
)

type InviteHandler struct {
	service service.InviteService
}

func NewInviteHandler(service service.InviteService) *InviteHandler {
	return &InviteHandler{service}
}

func (h *InviteHandler) Send(w http.ResponseWriter, r *http.Request) {
	// validate request body
	var req dto.SendInviteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		util.Error(w, http.StatusBadRequest, "Invalid request payload", err.Error())
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

	// Extract authenticated user from context
	userVal := r.Context().Value(middleware.UserContextKey)
	if userVal == nil {
		util.Error(w, http.StatusUnauthorized, "User not found in context")
		return
	}

	user, ok := userVal.(*model.User)
	if !ok {
		util.Error(w, http.StatusInternalServerError, "Invalid user in context")
		return
	}

	// Convert UUIDs to strings for service call
	roleUUIDs := make([]string, len(req.Roles))
	for i, roleUUID := range req.Roles {
		roleUUIDs[i] = roleUUID.String()
	}

	// Send invite
	_, err := h.service.SendInvite(req.Email, user.UserID, roleUUIDs)
	if err != nil {
		util.Error(w, http.StatusInternalServerError, "Failed to send invite", err.Error())
		return
	}

	// Respond
	util.Success(w, nil, "Invite sent successfully")
}
