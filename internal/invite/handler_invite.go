package invite

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/maintainerd/maintainerd-auth/internal/auditlog"
	"github.com/maintainerd/maintainerd-auth/internal/platform/middleware"
	resp "github.com/maintainerd/maintainerd-auth/internal/platform/response"
	"github.com/maintainerd/maintainerd-auth/internal/shared"
)

// InviteHandler handles HTTP requests for user invitation management.
// All endpoints are tenant-scoped - the middleware validates user access to the tenant
// and sets it in the request context. The service layer ensures invites belong to the tenant.
type InviteHandler struct {
	service     InviteService
	auditLogger auditlog.ManagementAuditLogger
}

// NewInviteHandler creates a new instance of InviteHandler.
func NewInviteHandler(service InviteService) *InviteHandler {
	return &InviteHandler{service: service}
}

// SetAuditLogger wires the management audit logger into the handler.
func (h *InviteHandler) SetAuditLogger(l auditlog.ManagementAuditLogger) {
	h.auditLogger = l
}

func (h *InviteHandler) logAudit(r *http.Request, tenantID int64, actorUserID *int64, action, resourceType, resourceID string, resourceUUID *uuid.UUID, changes, outcome string) {
	if h.auditLogger == nil {
		return
	}
	_ = h.auditLogger.Log(r.Context(), auditlog.LogEntry{
		TenantID:     tenantID,
		ActorUserID:  actorUserID,
		Action:       action,
		ResourceType: resourceType,
		ResourceID:   resourceID,
		ResourceUUID: resourceUUID,
		Changes:      changes,
		Outcome:      outcome,
	})
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
	result, err := h.service.SendInvite(r.Context(), tenant.TenantID, req.Email, user.UserID, req.RegistrationFlowUUID, req.CallbackURL)
	if err != nil {
		resp.HandleServiceError(w, r, "Failed to send invite", err)
		return
	}

	actorUserID := &user.UserID
	changesJSON, _ := json.Marshal(map[string]any{"after": result})
	h.logAudit(r, tenant.TenantID, actorUserID, "invite.send", "invite", result.InviteUUID.String(), &result.InviteUUID, string(changesJSON), "success")

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

	result, err := h.service.ResendInvite(r.Context(), inviteUUID, tenant.TenantID)
	if err != nil {
		resp.HandleServiceError(w, r, "Failed to resend invite", err)
		return
	}

	var actorUserID *int64
	if u := middleware.AuthFromRequest(r).User; u != nil {
		actorUserID = &u.UserID
	}
	changesJSON, _ := json.Marshal(map[string]any{"update": map[string]any{"invite_uuid": inviteUUID.String()}})
	h.logAudit(r, tenant.TenantID, actorUserID, "invite.resend", "invite", result.InviteUUID.String(), &result.InviteUUID, string(changesJSON), "success")

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

	var actorUserID *int64
	if u := middleware.AuthFromRequest(r).User; u != nil {
		actorUserID = &u.UserID
	}
	changesJSON, _ := json.Marshal(map[string]any{"before": map[string]any{"id": inviteUUID.String()}})
	h.logAudit(r, tenant.TenantID, actorUserID, "invite.revoke", "invite", inviteUUID.String(), &inviteUUID, string(changesJSON), "success")

	resp.Success(w, nil, "Invite revoked successfully")
}

// Get returns invite details by UUID, scoped to the authenticated tenant.
//
// GET /invite/{invite_uuid}
func (h *InviteHandler) Get(w http.ResponseWriter, r *http.Request) {
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

	invite, err := h.service.GetByUUID(r.Context(), inviteUUID, tenant.TenantID)
	if err != nil {
		resp.HandleServiceError(w, r, "Failed to get invite", err)
		return
	}

	resp.Success(w, toInviteResponseDTO(*invite), "Invite retrieved successfully")
}

func (h *InviteHandler) GetInviteContext(w http.ResponseWriter, r *http.Request) {
	inviteToken := r.URL.Query().Get("invite_token")
	if inviteToken == "" {
		resp.Error(w, http.StatusBadRequest, "Invite token is required")
		return
	}
	invite, err := h.service.GetByToken(r.Context(), inviteToken)
	if err != nil {
		resp.HandleServiceError(w, r, "Failed to get invite", err)
		return
	}
	if invite.Status != shared.StatusPending {
		resp.Error(w, http.StatusGone, "Invite is no longer valid")
		return
	}
	if invite.ExpiresAt != nil && time.Now().After(*invite.ExpiresAt) {
		resp.Error(w, http.StatusGone, "Invite has expired")
		return
	}
	resp.Success(w, toInviteContextResponseDTO(*invite), "Invite retrieved")
}
