package user

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/maintainerd/maintainerd-auth/internal/auditlog"
	"github.com/maintainerd/maintainerd-auth/internal/platform/middleware"
	resp "github.com/maintainerd/maintainerd-auth/internal/platform/response"
)

// ──────────────────────────────────────────────────────────────────────────────
// DTO + validation
// ──────────────────────────────────────────────────────────────────────────────

// ErasureRequestDTO is the POST body for /users/{uuid}/erasure-requests and
// /me/erasure-request. Every field is optional — when the body is empty a
// default request is still created.
type ErasureRequestDTO struct {
	Reason string `json:"reason"`
}

func (r *ErasureRequestDTO) Validate() error { return nil }

// ErasureResponseDTO is the JSON representation of a data_erasure_requests row.
type ErasureResponseDTO struct {
	UUID        string `json:"uuid"`
	Status      string `json:"status"`
	Reason      string `json:"reason,omitempty"`
	ScheduledAt string `json:"scheduled_at"`
	CreatedAt   string `json:"created_at"`
}

// ──────────────────────────────────────────────────────────────────────────────
// Handler
// ──────────────────────────────────────────────────────────────────────────────

// DataErasureHandler handles GDPR Article 17 erasure-request endpoints on the
// internal port (admin) and public port (self-service).
type DataErasureHandler struct {
	service     DataErasureService
	userRepo    UserRepository
	auditLogger auditlog.ManagementAuditLogger
}

// NewDataErasureHandler creates a new DataErasureHandler.
func NewDataErasureHandler(service DataErasureService, userRepo UserRepository) *DataErasureHandler {
	return &DataErasureHandler{service: service, userRepo: userRepo}
}

// SetAuditLogger injects the audit logger (called by the wiring layer).
func (h *DataErasureHandler) SetAuditLogger(l auditlog.ManagementAuditLogger) { h.auditLogger = l }

func (h *DataErasureHandler) logErasureAudit(r *http.Request, tenantID int64, actorUserID *int64, action string, changes map[string]any, outcome string) {
	if h.auditLogger == nil {
		return
	}
	diff, _ := json.Marshal(changes)
	_ = h.auditLogger.Log(r.Context(), auditlog.LogEntry{
		TenantID:     tenantID,
		ActorUserID:  actorUserID,
		Action:       action,
		ResourceType: "data_erasure_request",
		Changes:      string(diff),
		Outcome:      outcome,
	})
}

// ──────────────────────────────────────────────────────────────────────────────
// Admin endpoint
// ──────────────────────────────────────────────────────────────────────────────

// RequestAdmin creates an erasure request for a target user on behalf of an admin.
//
// POST /users/{user_uuid}/erasure-requests
func (h *DataErasureHandler) RequestAdmin(w http.ResponseWriter, r *http.Request) {
	auth := middleware.AuthFromRequest(r)
	if auth.Tenant == nil {
		resp.Error(w, http.StatusUnauthorized, "Tenant not found in context")
		return
	}
	tenantID := auth.Tenant.TenantID

	userUUID, err := uuid.Parse(chi.URLParam(r, "user_uuid"))
	if err != nil {
		resp.Error(w, http.StatusBadRequest, "Invalid user UUID")
		return
	}

	target, err := h.userRepo.FindByUUID(userUUID)
	if err != nil || target == nil {
		resp.Error(w, http.StatusNotFound, "User not found")
		return
	}
	if target.TenantID != tenantID {
		resp.Error(w, http.StatusNotFound, "User not found")
		return
	}

	var body ErasureRequestDTO
	// Empty body is fine — a default request is created.
	if r.Body != nil && r.ContentLength > 0 {
		if derr := json.NewDecoder(r.Body).Decode(&body); derr != nil {
			resp.BadRequestBody(w)
			return
		}
	}

	var actorUserID *int64
	if auth.User != nil {
		actorUserID = &auth.User.UserID
	}

	result, err := h.service.RequestErasure(r.Context(), RequestErasureInput{
		TenantID:           tenantID,
		UserID:             target.UserID,
		RequestedByAdminID: actorUserID,
		Reason:             body.Reason,
	})
	if err != nil {
		resp.HandleServiceError(w, r, "Failed to create erasure request", err)
		return
	}

	changes := map[string]any{"action": "erasure_request_created", "target_user_id": userUUID.String(), "reason": body.Reason}
	h.logErasureAudit(r, tenantID, actorUserID, "erasure.request_create", changes, "success")

	resp.Success(w, toErasureDTO(result), "Data erasure request created successfully")
}

// ──────────────────────────────────────────────────────────────────────────────
// Self-service endpoint
// ──────────────────────────────────────────────────────────────────────────────

// RequestSelf creates an erasure request for the calling user.
//
// POST /me/erasure-request
func (h *DataErasureHandler) RequestSelf(w http.ResponseWriter, r *http.Request) {
	auth := middleware.AuthFromRequest(r)
	if auth.User == nil || auth.Tenant == nil {
		resp.Error(w, http.StatusUnauthorized, "Unauthorized")
		return
	}
	tenantID := auth.Tenant.TenantID
	userID := auth.User.UserID

	var body ErasureRequestDTO
	if r.Body != nil && r.ContentLength > 0 {
		if derr := json.NewDecoder(r.Body).Decode(&body); derr != nil {
			resp.BadRequestBody(w)
			return
		}
	}

	result, err := h.service.RequestErasure(r.Context(), RequestErasureInput{
		TenantID:          tenantID,
		UserID:            userID,
		RequestedByUserID: &userID,
		Reason:            body.Reason,
	})
	if err != nil {
		resp.HandleServiceError(w, r, "Failed to create erasure request", err)
		return
	}

	actorUserID := &userID
	changes := map[string]any{"action": "erasure_request_created", "reason": body.Reason}
	h.logErasureAudit(r, tenantID, actorUserID, "erasure.request_create", changes, "success")

	resp.Success(w, toErasureDTO(result), "Data erasure request created successfully")
}

// ──────────────────────────────────────────────────────────────────────────────
// Response mapping
// ──────────────────────────────────────────────────────────────────────────────

func toErasureDTO(r *DataErasureRequestResult) ErasureResponseDTO {
	return ErasureResponseDTO{
		UUID:        r.UUID.String(),
		Status:      r.Status,
		Reason:      r.Reason,
		ScheduledAt: r.ScheduledAt.Format("2006-01-02T15:04:05Z07:00"),
		CreatedAt:   r.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}
