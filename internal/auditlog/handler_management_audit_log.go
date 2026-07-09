package auditlog

import (
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/maintainerd/maintainerd-auth/internal/platform/middleware"
	resp "github.com/maintainerd/maintainerd-auth/internal/platform/response"
)

// ManagementAuditLogHandler handles HTTP requests for the management audit log.
type ManagementAuditLogHandler struct {
	repo ManagementAuditLogRepository
}

// NewManagementAuditLogHandler creates a ManagementAuditLogHandler backed by repo.
func NewManagementAuditLogHandler(repo ManagementAuditLogRepository) *ManagementAuditLogHandler {
	return &ManagementAuditLogHandler{repo: repo}
}

// List retrieves management audit log entries with pagination and filtering.
//
// GET /management-audit-log
//
// Query params: resource_type, action, actor_user_id, page, limit (max 100).
func (h *ManagementAuditLogHandler) List(w http.ResponseWriter, r *http.Request) {
	auth := middleware.AuthFromRequest(r)
	if auth.Tenant == nil {
		resp.Error(w, http.StatusUnauthorized, "Tenant not found in context")
		return
	}

	const maxLimit = 100
	filter := ManagementAuditLogFilter{
		Page:  1,
		Limit: 20,
	}

	if v := r.URL.Query().Get("resource_type"); v != "" {
		filter.ResourceType = v
	}
	if v := r.URL.Query().Get("action"); v != "" {
		filter.Action = v
	}
	if v := r.URL.Query().Get("actor_user_id"); v != "" {
		if n, err := strconv.ParseInt(v, 10, 64); err == nil {
			filter.ActorUserID = &n
		}
	}
	if v := r.URL.Query().Get("page"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			filter.Page = n
		}
	}
	if v := r.URL.Query().Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			filter.Limit = n
		}
	}
	if filter.Limit <= 0 {
		filter.Limit = 20
	}
	if filter.Limit > maxLimit {
		filter.Limit = maxLimit
	}

	logs, total, err := h.repo.FindPaginated(auth.Tenant.TenantID, filter)
	if err != nil {
		resp.Error(w, http.StatusInternalServerError, "Failed to retrieve audit log")
		return
	}

	dtos := make([]ManagementAuditLogResponseDTO, 0, len(logs))
	for _, l := range logs {
		dtos = append(dtos, toManagementAuditLogResponseDTO(l))
	}

	resp.Success(w, PaginatedResponseDTO[ManagementAuditLogResponseDTO]{
		Rows:       dtos,
		Total:      total,
		Page:       filter.Page,
		Limit:      filter.Limit,
		TotalPages: totalPages(total, filter.Limit),
	}, "Management audit log retrieved successfully")
}

// Get retrieves one management audit log entry by UUID.
//
// GET /management-audit-log/{audit_log_uuid}
func (h *ManagementAuditLogHandler) Get(w http.ResponseWriter, r *http.Request) {
	auth := middleware.AuthFromRequest(r)
	if auth.Tenant == nil {
		resp.Error(w, http.StatusUnauthorized, "Tenant not found in context")
		return
	}

	auditLogUUID, err := uuid.Parse(chi.URLParam(r, "audit_log_uuid"))
	if err != nil {
		resp.Error(w, http.StatusBadRequest, "Invalid audit log UUID")
		return
	}

	log, err := h.repo.FindByUUIDAndTenantID(auditLogUUID, auth.Tenant.TenantID)
	if err != nil {
		resp.Error(w, http.StatusInternalServerError, "Failed to retrieve audit log entry")
		return
	}
	if log == nil {
		resp.Error(w, http.StatusNotFound, "Audit log entry not found")
		return
	}

	resp.Success(w, toManagementAuditLogResponseDTO(*log), "Management audit log entry retrieved successfully")
}

func toManagementAuditLogResponseDTO(l ManagementAuditLog) ManagementAuditLogResponseDTO {
	dto := ManagementAuditLogResponseDTO{
		UUID:            l.ManagementAuditLogUUID.String(),
		Action:          l.Action,
		ResourceType:    l.ResourceType,
		ResourceID:      l.ResourceID,
		Outcome:         l.Outcome,
		CreatedAt:       l.CreatedAt.UTC().Format(time.RFC3339),
		ActorUserID:     l.ActorUserID,
		ActorUserName:   l.ActorUserName,
		ActorClientID:   l.ActorClientID,
		ActorClientName: l.ActorClientName,
		Changes:         string(l.Changes),
		ErrorMessage:    l.ErrorMessage,
		TraceID:         l.TraceID,
	}
	if l.IPAddress != nil {
		dto.IPAddress = *l.IPAddress
	}
	return dto
}

func totalPages(total int64, limit int) int {
	if total <= 0 || limit <= 0 {
		return 0
	}
	return int((total + int64(limit) - 1) / int64(limit))
}
