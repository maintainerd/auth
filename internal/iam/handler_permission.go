package iam

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/maintainerd/maintainerd-auth/internal/auditlog"
	"github.com/maintainerd/maintainerd-auth/internal/platform/middleware"
	"github.com/maintainerd/maintainerd-auth/internal/platform/pagination"
	"github.com/maintainerd/maintainerd-auth/internal/platform/ptr"
	resp "github.com/maintainerd/maintainerd-auth/internal/platform/response"
)

type PermissionHandler struct {
	permissionService PermissionService
	auditLogger       auditlog.ManagementAuditLogger
}

func NewPermissionHandler(permissionService PermissionService) *PermissionHandler {
	return &PermissionHandler{permissionService: permissionService}
}

// SetAuditLogger wires the management audit logger into the handler.
func (h *PermissionHandler) SetAuditLogger(l auditlog.ManagementAuditLogger) { h.auditLogger = l }

func (h *PermissionHandler) logAudit(r *http.Request, tenantID int64, actorUserID *int64, action, resourceType, resourceID string, resourceUUID *uuid.UUID, changes, outcome string) {
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

// Get permissions with pagination
func (h *PermissionHandler) Get(w http.ResponseWriter, r *http.Request) {
	// Get tenant from context
	tenant := middleware.AuthFromRequest(r).Tenant
	if tenant == nil {
		resp.Error(w, http.StatusUnauthorized, "Tenant not found in context")
		return
	}

	// Parse query parameters
	q := r.URL.Query()

	// Parse pagination

	// Parse bools safely
	var isSystem *bool

	if v := q.Get("is_system"); v != "" {
		parsed, err := strconv.ParseBool(v)
		if err == nil {
			isSystem = &parsed
		}
	}

	// Build request DTO
	reqParams := PermissionFilterDTO{
		Name:                 ptr.PtrOrNil(q.Get("name")),
		Description:          ptr.PtrOrNil(q.Get("description")),
		APIUUID:              ptr.PtrOrNil(q.Get("api_id")),
		RoleUUID:             ptr.PtrOrNil(q.Get("role_id")),
		ClientUUID:           ptr.PtrOrNil(q.Get("client_id")),
		Status:               ptr.PtrOrNil(q.Get("status")),
		IsSystem:             isSystem,
		PaginationRequestDTO: pagination.ParseQuery(r),
	}

	if err := reqParams.Validate(); err != nil {
		resp.ValidationError(w, err)
		return
	}

	// Build permission filter
	permissionFilter := PermissionServiceGetFilter{
		TenantID:    tenant.TenantID,
		Name:        reqParams.Name,
		Description: reqParams.Description,
		APIUUID:     reqParams.APIUUID,
		RoleUUID:    reqParams.RoleUUID,
		ClientUUID:  reqParams.ClientUUID,
		Status:      reqParams.Status,
		IsSystem:    reqParams.IsSystem,
		Page:        reqParams.Page,
		Limit:       reqParams.Limit,
		SortBy:      reqParams.SortBy,
		SortOrder:   reqParams.SortOrder,
	}

	// Fetch permissions
	result, err := h.permissionService.Get(r.Context(), permissionFilter)
	if err != nil {
		resp.HandleServiceError(w, r, "Failed to fetch permissions", err)
		return
	}

	// Map permissions result to DTO
	rows := make([]PermissionResponseDTO, len(result.Data))
	for i, r := range result.Data {
		rows[i] = toPermissionResponseDTO(r)
	}

	// Build response data
	response := PaginatedResponseDTO[PermissionResponseDTO]{
		Rows:       rows,
		Total:      result.Total,
		Page:       result.Page,
		Limit:      result.Limit,
		TotalPages: result.TotalPages,
	}

	resp.Success(w, response, "Permissions fetched successfully")
}

// Get permission by UUID
func (h *PermissionHandler) GetByUUID(w http.ResponseWriter, r *http.Request) {
	// Get tenant from context
	tenant := middleware.AuthFromRequest(r).Tenant
	if tenant == nil {
		resp.Error(w, http.StatusUnauthorized, "Tenant not found in context")
		return
	}

	permissonUUID, err := uuid.Parse(chi.URLParam(r, "permission_uuid"))
	if err != nil {
		resp.Error(w, http.StatusBadRequest, "Invalid permission UUID")
		return
	}

	permission, err := h.permissionService.GetByUUID(r.Context(), permissonUUID, tenant.TenantID)
	if err != nil {
		resp.HandleServiceError(w, r, "Permission not found", err)
		return
	}

	dtoRes := toPermissionResponseDTO(*permission)

	resp.Success(w, dtoRes, "Permission fetched successfully")
}

// Create permission
func (h *PermissionHandler) Create(w http.ResponseWriter, r *http.Request) {
	// Get tenant from context
	tenant := middleware.AuthFromRequest(r).Tenant
	if tenant == nil {
		resp.Error(w, http.StatusUnauthorized, "Tenant not found in context")
		return
	}

	var req PermissionCreateRequestDTO
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		resp.BadRequestBody(w)
		return
	}

	if err := req.Validate(); err != nil {
		resp.ValidationError(w, err)
		return
	}

	permission, err := h.permissionService.Create(r.Context(), tenant.TenantID, req.Name, req.Description, req.Status, false, req.APIUUID)
	if err != nil {
		resp.HandleServiceError(w, r, "Failed to create permission", err)
		return
	}

	dtoRes := toPermissionResponseDTO(*permission)
	var actorUserID *int64
	if authCtx := middleware.AuthFromRequest(r); authCtx.User != nil {
		actorUserID = &authCtx.User.UserID
	}
	changesJSON, _ := json.Marshal(map[string]any{"after": dtoRes})
	h.logAudit(r, tenant.TenantID, actorUserID, "permission.create", "permission", permission.PermissionUUID.String(), &permission.PermissionUUID, string(changesJSON), "success")
	resp.Created(w, dtoRes, "Permission created successfully")
}

// Update permission
func (h *PermissionHandler) Update(w http.ResponseWriter, r *http.Request) {
	// Get tenant from context
	tenant := middleware.AuthFromRequest(r).Tenant
	if tenant == nil {
		resp.Error(w, http.StatusUnauthorized, "Tenant not found in context")
		return
	}

	permissionUUID, err := uuid.Parse(chi.URLParam(r, "permission_uuid"))
	if err != nil {
		resp.Error(w, http.StatusBadRequest, "Invalid permission UUID")
		return
	}

	var req PermissionUpdateRequestDTO
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		resp.BadRequestBody(w)
		return
	}

	if err := req.Validate(); err != nil {
		resp.ValidationError(w, err)
		return
	}

	permission, err := h.permissionService.Update(r.Context(), permissionUUID, tenant.TenantID, req.Name, req.Description, req.Status)
	if err != nil {
		resp.HandleServiceError(w, r, "Failed to update permission", err)
		return
	}

	dtoRes := toPermissionResponseDTO(*permission)
	var actorUserID *int64
	if authCtx := middleware.AuthFromRequest(r); authCtx.User != nil {
		actorUserID = &authCtx.User.UserID
	}
	changesJSON, _ := json.Marshal(map[string]any{"update": req, "after": dtoRes})
	h.logAudit(r, tenant.TenantID, actorUserID, "permission.update", "permission", permission.PermissionUUID.String(), &permission.PermissionUUID, string(changesJSON), "success")
	resp.Success(w, dtoRes, "Permission updated successfully")
}

// Set permission status
func (h *PermissionHandler) SetStatus(w http.ResponseWriter, r *http.Request) {
	// Get tenant from context
	tenant := middleware.AuthFromRequest(r).Tenant
	if tenant == nil {
		resp.Error(w, http.StatusUnauthorized, "Tenant not found in context")
		return
	}

	permissionUUID, err := uuid.Parse(chi.URLParam(r, "permission_uuid"))
	if err != nil {
		resp.Error(w, http.StatusBadRequest, "Invalid permission UUID")
		return
	}

	var req PermissionStatusUpdateDTO
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		resp.BadRequestBody(w)
		return
	}

	if err := req.Validate(); err != nil {
		resp.ValidationError(w, err)
		return
	}

	permission, err := h.permissionService.SetStatus(r.Context(), permissionUUID, tenant.TenantID, req.Status)
	if err != nil {
		resp.HandleServiceError(w, r, "Failed to update permission status", err)
		return
	}

	dtoRes := toPermissionResponseDTO(*permission)
	var actorUserID *int64
	if authCtx := middleware.AuthFromRequest(r); authCtx.User != nil {
		actorUserID = &authCtx.User.UserID
	}
	changesJSON, _ := json.Marshal(map[string]any{"update": map[string]any{"status": req.Status}})
	h.logAudit(r, tenant.TenantID, actorUserID, "permission.set_status", "permission", permission.PermissionUUID.String(), &permission.PermissionUUID, string(changesJSON), "success")
	resp.Success(w, dtoRes, "Permission status updated successfully")
}

// Delete permission
func (h *PermissionHandler) Delete(w http.ResponseWriter, r *http.Request) {
	// Get tenant from context
	tenant := middleware.AuthFromRequest(r).Tenant
	if tenant == nil {
		resp.Error(w, http.StatusUnauthorized, "Tenant not found in context")
		return
	}

	permissionUUID, err := uuid.Parse(chi.URLParam(r, "permission_uuid"))
	if err != nil {
		resp.Error(w, http.StatusBadRequest, "Invalid permission UUID")
		return
	}

	permission, err := h.permissionService.DeleteByUUID(r.Context(), permissionUUID, tenant.TenantID)
	if err != nil {
		resp.HandleServiceError(w, r, "Failed to delete permission", err)
		return
	}

	dtoRes := toPermissionResponseDTO(*permission)
	var actorUserID *int64
	if authCtx := middleware.AuthFromRequest(r); authCtx.User != nil {
		actorUserID = &authCtx.User.UserID
	}
	changesJSON, _ := json.Marshal(map[string]any{"before": map[string]any{"id": permissionUUID.String()}})
	h.logAudit(r, tenant.TenantID, actorUserID, "permission.delete", "permission", permissionUUID.String(), &permissionUUID, string(changesJSON), "success")
	resp.Success(w, dtoRes, "Permission deleted successfully")
}

// Convert permission result to DTO
func toPermissionResponseDTO(r PermissionServiceDataResult) PermissionResponseDTO {
	result := PermissionResponseDTO{
		PermissionUUID: r.PermissionUUID,
		Name:           r.Name,
		Description:    r.Description,
		Status:         r.Status,
		IsSystem:       r.IsSystem,
		CreatedAt:      r.CreatedAt,
		UpdatedAt:      r.UpdatedAt,
	}

	if r.API != nil {
		result.API = &APIResponseDTO{
			APIUUID:     r.API.APIUUID,
			Name:        r.API.Name,
			DisplayName: r.API.DisplayName,
			Description: r.API.Description,
			Identifier:  r.API.Identifier,
			Status:      r.API.Status,
			CreatedAt:   r.API.CreatedAt,
			UpdatedAt:   r.API.UpdatedAt,
		}
	}

	return result
}
