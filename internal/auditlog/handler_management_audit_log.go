package auditlog

import (
	"encoding/json"
	"net/http"

	"github.com/maintainerd/maintainerd-auth/internal/platform/middleware"
	resp "github.com/maintainerd/maintainerd-auth/internal/platform/response"
)

type ManagementAuditLogHandler struct {
	repo ManagementAuditLogRepository
}

func NewManagementAuditLogHandler(repo ManagementAuditLogRepository) *ManagementAuditLogHandler {
	return &ManagementAuditLogHandler{repo: repo}
}

type ManagementAuditLogFilterDTO struct {
	ResourceType string `json:"resource_type"`
	Action       string `json:"action"`
	ActorUserID  *int64 `json:"actor_user_id"`
	Page         int    `json:"page"`
	Limit        int    `json:"limit"`
}

type ManagementAuditLogResponseDTO struct {
	UUID         string `json:"uuid"`
	Action       string `json:"action"`
	ResourceType string `json:"resource_type"`
	ResourceID   string `json:"resource_id"`
	Outcome      string `json:"outcome"`
	IPAddress    string `json:"ip_address,omitempty"`
	CreatedAt    string `json:"created_at"`
}

// List retrieves management audit log entries with pagination and filtering.
//
// GET /management-audit-log
func (h *ManagementAuditLogHandler) List(w http.ResponseWriter, r *http.Request) {
	auth := middleware.AuthFromRequest(r)
	if auth.Tenant == nil {
		resp.Error(w, http.StatusBadRequest, "Missing tenant context")
		return
	}

	filter := ManagementAuditLogFilter{
		Page:  1,
		Limit: 20,
	}

	if r.URL.Query().Get("resource_type") != "" {
		filter.ResourceType = r.URL.Query().Get("resource_type")
	}
	if r.URL.Query().Get("action") != "" {
		filter.Action = r.URL.Query().Get("action")
	}
	if body := r.Body; body != nil {
		var dto ManagementAuditLogFilterDTO
		if err := json.NewDecoder(body).Decode(&dto); err == nil {
			if dto.ResourceType != "" {
				filter.ResourceType = dto.ResourceType
			}
			if dto.Action != "" {
				filter.Action = dto.Action
			}
			if dto.ActorUserID != nil {
				filter.ActorUserID = dto.ActorUserID
			}
			if dto.Page > 0 {
				filter.Page = dto.Page
			}
			if dto.Limit > 0 {
				filter.Limit = dto.Limit
			}
		}
	}

	logs, total, err := h.repo.FindPaginated(auth.Tenant.TenantID, filter)
	if err != nil {
		resp.Error(w, http.StatusInternalServerError, "Failed to retrieve audit log")
		return
	}

	dtos := make([]ManagementAuditLogResponseDTO, 0, len(logs))
	for _, l := range logs {
		dto := ManagementAuditLogResponseDTO{
			UUID:         l.ManagementAuditLogUUID.String(),
			Action:       l.Action,
			ResourceType: l.ResourceType,
			ResourceID:   l.ResourceID,
			Outcome:      l.Outcome,
			CreatedAt:    l.CreatedAt.Format("2006-01-02T15:04:05Z"),
		}
		if l.IPAddress != nil {
			dto.IPAddress = *l.IPAddress
		}
		dtos = append(dtos, dto)
	}

	resp.Success(w, PaginatedResponseDTO[ManagementAuditLogResponseDTO]{
		Rows:       dtos,
		Total:      total,
		Page:       filter.Page,
		Limit:      filter.Limit,
	}, "Management audit log retrieved successfully")
}
