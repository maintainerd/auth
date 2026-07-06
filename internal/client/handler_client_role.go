package client

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/maintainerd/maintainerd-auth/internal/platform/middleware"
	resp "github.com/maintainerd/maintainerd-auth/internal/platform/response"
)

// ClientRoleAssignRequestDTO submits a role assignment to a client.
type ClientRoleAssignRequestDTO struct {
	RoleUUID string `json:"role_uuid"`
}

func (r ClientRoleAssignRequestDTO) Validate() error {
	if r.RoleUUID == "" {
		return fmt.Errorf("role_uuid is required")
	}
	if _, err := uuid.Parse(r.RoleUUID); err != nil {
		return fmt.Errorf("role_uuid must be a valid UUID")
	}
	return nil
}

// ClientRoleResponseDTO is a single client_role entry.
type ClientRoleResponseDTO struct {
	ClientRoleUUID string `json:"client_role_uuid"`
	RoleUUID       string `json:"role_uuid"`
	CreatedAt      string `json:"created_at"`
}

// AssignRole adds a role to a client.
//
// POST /clients/{client_uuid}/roles
func (h *ClientHandler) AssignRole(w http.ResponseWriter, r *http.Request) {
	tenant := middleware.AuthFromRequest(r).Tenant
	if tenant == nil {
		resp.Error(w, http.StatusUnauthorized, "Tenant not found in context")
		return
	}

	clientUUID, err := uuid.Parse(chi.URLParam(r, "client_uuid"))
	if err != nil {
		resp.Error(w, http.StatusBadRequest, "Invalid client UUID")
		return
	}

	var req ClientRoleAssignRequestDTO
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		resp.BadRequestBody(w)
		return
	}
	if err := req.Validate(); err != nil {
		resp.ValidationError(w, err)
		return
	}

	roleUUID, _ := uuid.Parse(req.RoleUUID)

	result, err := h.ClientService.AssignClientRole(r.Context(), clientUUID, roleUUID, tenant.TenantID, nil)
	if err != nil {
		resp.HandleServiceError(w, r, "Failed to assign role to client", err)
		return
	}

	resp.Created(w, ClientRoleResponseDTO{
		ClientRoleUUID: result.ClientRoleUUID.String(),
		RoleUUID:       roleUUID.String(),
		CreatedAt:      result.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}, "Role assigned to client successfully")
}

// RemoveRole removes a role from a client.
//
// DELETE /clients/{client_uuid}/roles/{role_uuid}
func (h *ClientHandler) RemoveRole(w http.ResponseWriter, r *http.Request) {
	tenant := middleware.AuthFromRequest(r).Tenant
	if tenant == nil {
		resp.Error(w, http.StatusUnauthorized, "Tenant not found in context")
		return
	}

	clientUUID, err := uuid.Parse(chi.URLParam(r, "client_uuid"))
	if err != nil {
		resp.Error(w, http.StatusBadRequest, "Invalid client UUID")
		return
	}

	roleUUID, err := uuid.Parse(chi.URLParam(r, "role_uuid"))
	if err != nil {
		resp.Error(w, http.StatusBadRequest, "Invalid role UUID")
		return
	}

	if err := h.ClientService.RemoveClientRole(r.Context(), clientUUID, roleUUID, tenant.TenantID); err != nil {
		resp.HandleServiceError(w, r, "Failed to remove role from client", err)
		return
	}

	resp.Success(w, nil, "Role removed from client successfully")
}

// ListRoles returns all roles assigned to a client.
//
// GET /clients/{client_uuid}/roles
func (h *ClientHandler) ListRoles(w http.ResponseWriter, r *http.Request) {
	tenant := middleware.AuthFromRequest(r).Tenant
	if tenant == nil {
		resp.Error(w, http.StatusUnauthorized, "Tenant not found in context")
		return
	}

	clientUUID, err := uuid.Parse(chi.URLParam(r, "client_uuid"))
	if err != nil {
		resp.Error(w, http.StatusBadRequest, "Invalid client UUID")
		return
	}

	roles, err := h.ClientService.ListClientRoles(r.Context(), clientUUID, tenant.TenantID)
	if err != nil {
		resp.HandleServiceError(w, r, "Failed to list client roles", err)
		return
	}

	items := make([]ClientRoleResponseDTO, len(roles))
	for i, role := range roles {
		items[i] = ClientRoleResponseDTO{
			ClientRoleUUID: role.ClientRoleUUID.String(),
			CreatedAt:      role.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		}
	}

	resp.Success(w, items, "Client roles retrieved successfully")
}
