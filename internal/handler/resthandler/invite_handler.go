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

// InviteHandler handles HTTP requests for user invitation management.
// All endpoints are tenant-scoped - the middleware validates user access to the tenant
// and sets it in the request context. The service layer ensures invites belong to the tenant.
type InviteHandler struct {
	service service.InviteService
}

// NewInviteHandler creates a new instance of InviteHandler.
func NewInviteHandler(service service.InviteService) *InviteHandler {
	return &InviteHandler{service}
}

// Send sends an invitation to a user to join the tenant with specified roles.
// Tenant access is validated by middleware.
// The invite is automatically associated with the tenant from context.
func (h *InviteHandler) Send(w http.ResponseWriter, r *http.Request) {
	// Tenant is already validated by middleware - just extract from context
	tenant, ok := r.Context().Value(middleware.TenantContextKey).(*model.Tenant)
	if !ok || tenant == nil {
		util.Error(w, http.StatusUnauthorized, "Tenant not found in context")
		return
	}

	// Extract authenticated user from context (needed for inviter tracking)
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

	// Decode request body
	var req dto.SendInviteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		util.Error(w, http.StatusBadRequest, "Invalid request payload", err.Error())
		return
	}

	// Validate request data
	if err := req.Validate(); err != nil {
		if ve, ok := err.(validation.Errors); ok {
			util.Error(w, http.StatusBadRequest, "Validation failed", ve)
			return
		}
		util.Error(w, http.StatusBadRequest, "Validation failed", err.Error())
		return
	}

	// Convert role UUIDs to strings for service call
	roleUUIDs := make([]string, len(req.Roles))
	for i, roleUUID := range req.Roles {
		roleUUIDs[i] = roleUUID.String()
	}

	// Send invite associated with tenant
	_, err := h.service.SendInvite(tenant.TenantID, req.Email, user.UserID, roleUUIDs)
	if err != nil {
		util.Error(w, http.StatusInternalServerError, "Failed to send invite", err.Error())
		return
	}

	util.Success(w, nil, "Invite sent successfully")
}
