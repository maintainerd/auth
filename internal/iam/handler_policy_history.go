package iam

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/maintainerd/maintainerd-auth/internal/platform/middleware"
	resp "github.com/maintainerd/maintainerd-auth/internal/platform/response"
)

// PolicyHistoryHandler handles policy version history read endpoints on the
// internal port (8080).
type PolicyHistoryHandler struct {
	policyService PolicyService
}

// NewPolicyHistoryHandler creates a new PolicyHistoryHandler.
func NewPolicyHistoryHandler(policyService PolicyService) *PolicyHistoryHandler {
	return &PolicyHistoryHandler{policyService: policyService}
}

// PolicyHistoryResponseDTO is the JSON representation of a policy version
// snapshot entry.
type PolicyHistoryResponseDTO struct {
	UUID          string `json:"uuid"`
	VersionNumber int    `json:"version_number"`
	Name          string `json:"name"`
	Description   string `json:"description,omitempty"`
	PolicyVersion string `json:"policy_version"`
	SnapshotAt    string `json:"snapshot_at"`
}

// PolicyHistoryDetailResponseDTO includes the full snapshotted document for the
// diff/rollback UI.
type PolicyHistoryDetailResponseDTO struct {
	UUID          string          `json:"uuid"`
	VersionNumber int             `json:"version_number"`
	Name          string          `json:"name"`
	Description   string          `json:"description,omitempty"`
	Document      json.RawMessage `json:"document"`
	PolicyVersion string          `json:"policy_version"`
	ChangeReason  string          `json:"change_reason,omitempty"`
	SnapshotAt    string          `json:"snapshot_at"`
}

// ListHistory returns a paginated list of version snapshots for a policy.
//
// GET /policies/{policy_uuid}/history
func (h *PolicyHistoryHandler) ListHistory(w http.ResponseWriter, r *http.Request) {
	tenant := middleware.AuthFromRequest(r).Tenant
	if tenant == nil {
		resp.Error(w, http.StatusUnauthorized, "Tenant not found in context")
		return
	}

	policyUUID, err := uuid.Parse(chi.URLParam(r, "policy_uuid"))
	if err != nil {
		resp.Error(w, http.StatusBadRequest, "Invalid policy UUID")
		return
	}

	page := 1
	limit := 20
	if p := getQueryInt(r, "page"); p > 0 {
		page = p
	}
	if l := getQueryInt(r, "limit"); l > 0 && l <= 100 {
		limit = l
	}

	result, err := h.policyService.GetHistory(r.Context(), policyUUID, tenant.TenantID, page, limit)
	if err != nil {
		resp.HandleServiceError(w, r, "Failed to get policy history", err)
		return
	}

	dtos := make([]PolicyHistoryResponseDTO, len(result.Data))
	for i, e := range result.Data {
		dtos[i] = PolicyHistoryResponseDTO{
			UUID:          e.UUID.String(),
			VersionNumber: e.VersionNumber,
			Name:          e.Name,
			Description:   stringPtrOrEmpty(e.Description),
			PolicyVersion: e.PolicyVersion,
			SnapshotAt:    e.SnapshotAt.Format("2006-01-02T15:04:05Z07:00"),
		}
	}

	resp.Success(w, PaginatedResponseDTO[PolicyHistoryResponseDTO]{
		Rows:       dtos,
		Total:      result.Total,
		Page:       result.Page,
		Limit:      result.Limit,
		TotalPages: result.TotalPages,
	}, "Policy history retrieved successfully")
}

// GetHistoryVersion returns a single version snapshot.
//
// GET /policies/{policy_uuid}/history/{version}
func (h *PolicyHistoryHandler) GetHistoryVersion(w http.ResponseWriter, r *http.Request) {
	tenant := middleware.AuthFromRequest(r).Tenant
	if tenant == nil {
		resp.Error(w, http.StatusUnauthorized, "Tenant not found in context")
		return
	}

	policyUUID, err := uuid.Parse(chi.URLParam(r, "policy_uuid"))
	if err != nil {
		resp.Error(w, http.StatusBadRequest, "Invalid policy UUID")
		return
	}

	versionNumber, err := strconv.Atoi(chi.URLParam(r, "version_number"))
	if err != nil || versionNumber < 1 {
		resp.Error(w, http.StatusBadRequest, "Invalid version number")
		return
	}

	entry, err := h.policyService.GetHistoryVersion(r.Context(), policyUUID, tenant.TenantID, versionNumber)
	if err != nil {
		resp.HandleServiceError(w, r, "Failed to get policy version", err)
		return
	}

	document := json.RawMessage(entry.Document)
	if len(document) == 0 {
		document = json.RawMessage("{}")
	}

	resp.Success(w, PolicyHistoryDetailResponseDTO{
		UUID:          entry.UUID.String(),
		VersionNumber: entry.VersionNumber,
		Name:          entry.Name,
		Description:   stringPtrOrEmpty(entry.Description),
		Document:      document,
		PolicyVersion: entry.PolicyVersion,
		ChangeReason:  stringPtrOrEmpty(entry.ChangeReason),
		SnapshotAt:    entry.SnapshotAt.Format("2006-01-02T15:04:05Z07:00"),
	}, "Policy version retrieved successfully")
}

func stringPtrOrEmpty(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func getQueryInt(r *http.Request, key string) int {
	v := r.URL.Query().Get(key)
	if v == "" {
		return 0
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return 0
	}
	return n
}
