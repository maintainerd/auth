package client

import (
	"time"

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
	RoleUUID string `json:"role_id"`
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
	ClientRoleUUID string `json:"client_role_id"`
	RoleUUID       string `json:"role_id"`
	// Name and Description identify the role for a human; a UUID alone forces the
	// console into a second lookup per row.
	Name        string `json:"name,omitempty"`
	Description string `json:"description,omitempty"`
	CreatedAt   string `json:"created_at"`
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

	// A role grant decides what the client may do, so it must be attributable: the
	// actor also becomes the created_by stamp, which used to be nil for every grant.
	user := middleware.AuthFromRequest(r).User
	if user == nil {
		resp.Error(w, http.StatusUnauthorized, "User not found in context")
		return
	}

	result, err := h.ClientService.AssignClientRole(r.Context(), clientUUID, roleUUID, tenant.TenantID, user.UserUUID)
	if err != nil {
		resp.HandleServiceError(w, r, "Failed to assign role to client", err)
		return
	}

	authCtxAssignRole := middleware.AuthFromRequest(r)
	var actorUserIDAssignRole *int64
	if authCtxAssignRole.User != nil {
		actorUserIDAssignRole = &authCtxAssignRole.User.UserID
	}
	changesJSONAssignRole, _ := json.Marshal(map[string]any{"update": map[string]any{"role_id": roleUUID.String()}})
	h.logAudit(r, tenant.TenantID, actorUserIDAssignRole, "assign_role", "client", clientUUID.String(), &clientUUID, string(changesJSONAssignRole), "success")

	resp.Created(w, ClientRoleResponseDTO{
		ClientRoleUUID: result.ClientRoleUUID.String(),
		RoleUUID:       roleUUID.String(),
		CreatedAt:      result.CreatedAt.Format(time.RFC3339),
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

	user := middleware.AuthFromRequest(r).User
	if user == nil {
		resp.Error(w, http.StatusUnauthorized, "User not found in context")
		return
	}

	if err := h.ClientService.RemoveClientRole(r.Context(), clientUUID, roleUUID, tenant.TenantID, user.UserUUID); err != nil {
		resp.HandleServiceError(w, r, "Failed to remove role from client", err)
		return
	}

	authCtxRemoveRole := middleware.AuthFromRequest(r)
	var actorUserIDRemoveRole *int64
	if authCtxRemoveRole.User != nil {
		actorUserIDRemoveRole = &authCtxRemoveRole.User.UserID
	}
	changesJSONRemoveRole, _ := json.Marshal(map[string]any{"update": map[string]any{"role_id": roleUUID.String()}})
	h.logAudit(r, tenant.TenantID, actorUserIDRemoveRole, "remove_role", "client", clientUUID.String(), &clientUUID, string(changesJSONRemoveRole), "success")

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
			CreatedAt:      role.CreatedAt.Format(time.RFC3339),
		}
		// role_uuid was declared on the DTO but never populated, so every row came
		// back with an empty id and the endpoint was unusable.
		if role.Role != nil {
			items[i].RoleUUID = role.Role.RoleUUID.String()
			items[i].Name = role.Role.Name
			items[i].Description = role.Role.Description
		}
	}

	resp.Success(w, items, "Client roles retrieved successfully")
}
