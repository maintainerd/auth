package invite

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/platform/middleware"
	resp "github.com/maintainerd/auth/internal/platform/response"
)

// InviteHandler handles HTTP requests for user invitation management.
// All endpoints are tenant-scoped - the middleware validates user access to the tenant
// and sets it in the request context. The service layer ensures invites belong to the tenant.
type InviteHandler struct {
	service InviteService
}

// NewInviteHandler creates a new instance of InviteHandler.
func NewInviteHandler(service InviteService) *InviteHandler {
	return &InviteHandler{service}
}

// Send sends an invitation to a user to join the tenant with specified roles.
// Tenant access is validated by middleware.
// The invite is automatically associated with the tenant from context.
func (h *InviteHandler) Send(w http.ResponseWriter, r *http.Request) {
	// Tenant is already validated by middleware - just extract from context
	tenant := middleware.AuthFromRequest(r).Tenant
	if tenant == nil {
		resp.Error(w, http.StatusUnauthorized, "Tenant not found in context")
		return
	}

	// Extract authenticated user from context (needed for inviter tracking)
	user := middleware.AuthFromRequest(r).User
	if user == nil {
		resp.Error(w, http.StatusUnauthorized, "User not found in context")
		return
	}

	// Decode request body
	var req SendInviteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		resp.BadRequestBody(w)
		return
	}

	// Validate request data
	if err := req.Validate(); err != nil {
		resp.ValidationError(w, err)
		return
	}

	// Send invite associated with tenant
	_, err := h.service.SendInvite(r.Context(), tenant.TenantID, req.Email, user.UserID, req.AuthFlowUUID)
	if err != nil {
		resp.HandleServiceError(w, r, "Failed to send invite", err)
		return
	}

	resp.Success(w, nil, "Invite sent successfully")
}

// Resend regenerates the invite token and re-sends the invitation email.
//
// POST /invite/{invite_uuid}/resend
func (h *InviteHandler) Resend(w http.ResponseWriter, r *http.Request) {
	tenant := middleware.AuthFromRequest(r).Tenant
	if tenant == nil {
		resp.Error(w, http.StatusUnauthorized, "Tenant not found in context")
		return
	}

	inviteUUIDStr := chi.URLParam(r, "invite_uuid")
	inviteUUID, err := uuid.Parse(inviteUUIDStr)
	if err != nil {
		resp.Error(w, http.StatusBadRequest, "Invalid invite UUID")
		return
	}

	_, err = h.service.ResendInvite(r.Context(), inviteUUID, tenant.TenantID)
	if err != nil {
		resp.HandleServiceError(w, r, "Failed to resend invite", err)
		return
	}

	resp.Success(w, nil, "Invite resent successfully")
}

// List returns all invitations for the authenticated tenant.
//
// GET /invite
func (h *InviteHandler) List(w http.ResponseWriter, r *http.Request) {
	tenant := middleware.AuthFromRequest(r).Tenant
	if tenant == nil {
		resp.Error(w, http.StatusUnauthorized, "Tenant not found in context")
		return
	}

	invites, err := h.service.ListInvites(r.Context(), tenant.TenantID)
	if err != nil {
		resp.HandleServiceError(w, r, "Failed to list invites", err)
		return
	}

	rows := make([]InviteResponseDTO, 0, len(invites))
	for _, inv := range invites {
		rows = append(rows, toInviteResponseDTO(inv))
	}
	resp.Success(w, rows, "Invites retrieved successfully")
}

// Revoke marks a pending invitation as revoked.
//
// DELETE /invite/{invite_uuid}
func (h *InviteHandler) Revoke(w http.ResponseWriter, r *http.Request) {
	tenant := middleware.AuthFromRequest(r).Tenant
	if tenant == nil {
		resp.Error(w, http.StatusUnauthorized, "Tenant not found in context")
		return
	}

	inviteUUID, err := uuid.Parse(chi.URLParam(r, "invite_uuid"))
	if err != nil {
		resp.Error(w, http.StatusBadRequest, "Invalid invite UUID")
		return
	}

	if err := h.service.RevokeInvite(r.Context(), inviteUUID, tenant.TenantID); err != nil {
		resp.HandleServiceError(w, r, "Failed to revoke invite", err)
		return
	}

	resp.Success(w, nil, "Invite revoked successfully")
}
