package dashboard

import (
	"net/http"

	"github.com/maintainerd/auth/internal/platform/middleware"
	resp "github.com/maintainerd/auth/internal/platform/response"
)

type Handler struct {
	service Service
}

func NewHandler(service Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) GetSummary(w http.ResponseWriter, r *http.Request) {
	auth := middleware.AuthFromRequest(r)
	if auth == nil || auth.Tenant == nil {
		resp.Error(w, http.StatusUnauthorized, "Authentication required")
		return
	}

	summary, err := h.service.GetSummary(r.Context(), auth.Tenant.TenantID)
	if err != nil {
		resp.HandleServiceError(w, r, "Failed to fetch dashboard summary", err)
		return
	}

	resp.Success(w, summary, "Dashboard summary fetched successfully")
}
