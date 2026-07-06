package iam

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/maintainerd/maintainerd-auth/internal/auditlog"
	"github.com/maintainerd/maintainerd-auth/internal/platform/middleware"
	"github.com/maintainerd/maintainerd-auth/internal/platform/pagination"
	"github.com/maintainerd/maintainerd-auth/internal/platform/ptr"
	resp "github.com/maintainerd/maintainerd-auth/internal/platform/response"
)

type APIHandler struct {
	apiService  APIService
	auditLogger auditlog.ManagementAuditLogger
}

func NewAPIHandler(apiService APIService) *APIHandler {
	return &APIHandler{apiService: apiService}
}

// SetAuditLogger wires the management audit logger into the handler.
func (h *APIHandler) SetAuditLogger(l auditlog.ManagementAuditLogger) { h.auditLogger = l }

func (h *APIHandler) logAudit(r *http.Request, tenantID int64, actorUserID *int64, action, resourceType, resourceID string, resourceUUID *uuid.UUID, changes, outcome string) {
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

// GetAll APIs with pagination
func (h *APIHandler) Get(w http.ResponseWriter, r *http.Request) {
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
	// Parse status (comma-separated values)
	var status []string
	if statusParam := q.Get("status"); statusParam != "" {
		status = strings.Split(statusParam, ",")
		// Trim whitespace from each status
		for i, s := range status {
			status[i] = strings.TrimSpace(s)
		}
	}
	if v := q.Get("is_system"); v != "" {
		parsed, err := strconv.ParseBool(v)
		if err == nil {
			isSystem = &parsed
		}
	}

	// Build request DTO
	reqParams := APIFilterDTO{
		Name:                 ptr.PtrOrNil(q.Get("name")),
		DisplayName:          ptr.PtrOrNil(q.Get("display_name")),
		Identifier:           ptr.PtrOrNil(q.Get("identifier")),
		ServiceUUID:          ptr.PtrOrNil(q.Get("service_id")),
		Status:               status,
		IsSystem:             isSystem,
		PaginationRequestDTO: pagination.ParseQuery(r),
	}

	if err := reqParams.Validate(); err != nil {
		resp.ValidationError(w, err)
		return
	}

	// Convert service_id (UUID from external API) to internal service_id (int64) for filtering
	var serviceID *int64
	if reqParams.ServiceUUID != nil && *reqParams.ServiceUUID != "" {
		serviceUUID, err := uuid.Parse(*reqParams.ServiceUUID)
		if err != nil {
			resp.Error(w, http.StatusBadRequest, "Invalid service UUID format")
			return
		}

		// Look up service by UUID to get service_id
		serviceIDValue, err := h.apiService.GetServiceIDByUUID(r.Context(), serviceUUID, tenant.TenantID)
		if err != nil {
			resp.HandleServiceError(w, r, "Service not found", err)
			return
		}
		serviceID = &serviceIDValue
	}

	// Build service filter
	apiFilter := APIServiceGetFilter{
		Name:        reqParams.Name,
		DisplayName: reqParams.DisplayName,
		Identifier:  reqParams.Identifier,
		ServiceID:   serviceID,
		Status:      reqParams.Status,
		IsSystem:    reqParams.IsSystem,
		TenantID:    tenant.TenantID,
		Page:        reqParams.Page,
		Limit:       reqParams.Limit,
		SortBy:      reqParams.SortBy,
		SortOrder:   reqParams.SortOrder,
	}

	// Fetch APIs
	result, err := h.apiService.Get(r.Context(), apiFilter)
	if err != nil {
		resp.HandleServiceError(w, r, "Failed to fetch APIs", err)
		return
	}

	// Map service result to DTO
	rows := make([]APIResponseDTO, len(result.Data))
	for i, r := range result.Data {
		rows[i] = toAPIResponseDTO(r)
	}

	// Build response data
	response := PaginatedResponseDTO[APIResponseDTO]{
		Rows:       rows,
		Total:      result.Total,
		Page:       result.Page,
		Limit:      result.Limit,
		TotalPages: result.TotalPages,
	}

	resp.Success(w, response, "APIs fetched successfully")
}

// Get API by UUID
func (h *APIHandler) GetByUUID(w http.ResponseWriter, r *http.Request) {
	// Get tenant from context
	tenant := middleware.AuthFromRequest(r).Tenant
	if tenant == nil {
		resp.Error(w, http.StatusUnauthorized, "Tenant not found in context")
		return
	}

	apiUUID, err := uuid.Parse(chi.URLParam(r, "api_uuid"))
	if err != nil {
		resp.Error(w, http.StatusBadRequest, "Invalid API UUID")
		return
	}

	api, err := h.apiService.GetByUUID(r.Context(), apiUUID, tenant.TenantID)
	if err != nil {
		resp.HandleServiceError(w, r, "API not found", err)
		return
	}

	dtoRes := toAPIResponseDTO(*api)

	resp.Success(w, dtoRes, "API fetched successfully")
}

// Create API
func (h *APIHandler) Create(w http.ResponseWriter, r *http.Request) {
	// Get tenant from context
	tenant := middleware.AuthFromRequest(r).Tenant
	if tenant == nil {
		resp.Error(w, http.StatusUnauthorized, "Tenant not found in context")
		return
	}

	var req APICreateRequestDTO
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		resp.BadRequestBody(w)
		return
	}

	if err := req.Validate(); err != nil {
		resp.ValidationError(w, err)
		return
	}

	api, err := h.apiService.Create(r.Context(), tenant.TenantID, req.Name, req.DisplayName, req.Description, req.Status, false, req.ServiceUUID)
	if err != nil {
		resp.HandleServiceError(w, r, "Failed to create API", err)
		return
	}

	dtoRes := toAPIResponseDTO(*api)
	var actorUserID *int64
	if authCtx := middleware.AuthFromRequest(r); authCtx.User != nil {
		actorUserID = &authCtx.User.UserID
	}
	changesJSON, _ := json.Marshal(map[string]any{"after": dtoRes})
	h.logAudit(r, tenant.TenantID, actorUserID, "api.create", "api", api.APIUUID.String(), &api.APIUUID, string(changesJSON), "success")
	resp.Created(w, dtoRes, "API created successfully")
}

// Update API
func (h *APIHandler) Update(w http.ResponseWriter, r *http.Request) {
	// Get tenant from context
	tenant := middleware.AuthFromRequest(r).Tenant
	if tenant == nil {
		resp.Error(w, http.StatusUnauthorized, "Tenant not found in context")
		return
	}

	apiUUID, err := uuid.Parse(chi.URLParam(r, "api_uuid"))
	if err != nil {
		resp.Error(w, http.StatusBadRequest, "Invalid API UUID")
		return
	}

	var req APIUpdateRequestDTO
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		resp.BadRequestBody(w)
		return
	}

	if err := req.Validate(); err != nil {
		resp.ValidationError(w, err)
		return
	}

	api, err := h.apiService.Update(r.Context(), apiUUID, tenant.TenantID, req.Name, req.DisplayName, req.Description, req.Status, req.ServiceUUID)
	if err != nil {
		resp.HandleServiceError(w, r, "Failed to update API", err)
		return
	}

	dtoRes := toAPIResponseDTO(*api)
	var actorUserID *int64
	if authCtx := middleware.AuthFromRequest(r); authCtx.User != nil {
		actorUserID = &authCtx.User.UserID
	}
	changesJSON, _ := json.Marshal(map[string]any{"update": req, "after": dtoRes})
	h.logAudit(r, tenant.TenantID, actorUserID, "api.update", "api", api.APIUUID.String(), &api.APIUUID, string(changesJSON), "success")
	resp.Success(w, dtoRes, "API updated successfully")
}

// Set API status
func (h *APIHandler) SetStatus(w http.ResponseWriter, r *http.Request) {
	// Get tenant from context
	tenant := middleware.AuthFromRequest(r).Tenant
	if tenant == nil {
		resp.Error(w, http.StatusUnauthorized, "Tenant not found in context")
		return
	}

	apiUUID, err := uuid.Parse(chi.URLParam(r, "api_uuid"))
	if err != nil {
		resp.Error(w, http.StatusBadRequest, "Invalid API UUID")
		return
	}

	// Parse request body
	var req APIStatusUpdateDTO
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		resp.BadRequestBody(w)
		return
	}

	// Validate request
	if err := req.Validate(); err != nil {
		resp.ValidationError(w, err)
		return
	}

	api, err := h.apiService.SetStatusByUUID(r.Context(), apiUUID, tenant.TenantID, req.Status)
	if err != nil {
		resp.HandleServiceError(w, r, "Failed to update API", err)
		return
	}

	dtoRes := toAPIResponseDTO(*api)
	var actorUserID *int64
	if authCtx := middleware.AuthFromRequest(r); authCtx.User != nil {
		actorUserID = &authCtx.User.UserID
	}
	changesJSON, _ := json.Marshal(map[string]any{"update": map[string]any{"status": req.Status}})
	h.logAudit(r, tenant.TenantID, actorUserID, "api.set_status", "api", api.APIUUID.String(), &api.APIUUID, string(changesJSON), "success")
	resp.Success(w, dtoRes, "API status updated successfully")
}

// Delete API
func (h *APIHandler) Delete(w http.ResponseWriter, r *http.Request) {
	// Get tenant from context
	tenant := middleware.AuthFromRequest(r).Tenant
	if tenant == nil {
		resp.Error(w, http.StatusUnauthorized, "Tenant not found in context")
		return
	}

	apiUUID, err := uuid.Parse(chi.URLParam(r, "api_uuid"))
	if err != nil {
		resp.Error(w, http.StatusBadRequest, "Invalid API UUID")
		return
	}

	api, err := h.apiService.DeleteByUUID(r.Context(), apiUUID, tenant.TenantID)
	if err != nil {
		resp.HandleServiceError(w, r, "Failed to delete API", err)
		return
	}

	dtoRes := toAPIResponseDTO(*api)
	var actorUserID *int64
	if authCtx := middleware.AuthFromRequest(r); authCtx.User != nil {
		actorUserID = &authCtx.User.UserID
	}
	changesJSON, _ := json.Marshal(map[string]any{"before": map[string]any{"id": apiUUID.String()}})
	h.logAudit(r, tenant.TenantID, actorUserID, "api.delete", "api", apiUUID.String(), &apiUUID, string(changesJSON), "success")
	resp.Success(w, dtoRes, "API deleted successfully")
}

// Convert service result to DTO
func toAPIResponseDTO(r APIServiceDataResult) APIResponseDTO {
	result := APIResponseDTO{
		APIUUID:     r.APIUUID,
		Name:        r.Name,
		DisplayName: r.DisplayName,
		Description: r.Description,
		Identifier:  r.Identifier,
		Status:      r.Status,
		IsSystem:    r.IsSystem,
		CreatedAt:   r.CreatedAt,
		UpdatedAt:   r.UpdatedAt,
	}

	if r.Service != nil {
		result.Service = &ServiceResponseDTO{
			ServiceUUID: r.Service.ServiceUUID,
			Name:        r.Service.Name,
			DisplayName: r.Service.DisplayName,
			Description: r.Service.Description,
			Version:     r.Service.Version,
			IsSystem:    r.Service.IsSystem,
			Status:      r.Service.Status,
			APICount:    r.Service.APICount,
			PolicyCount: r.Service.PolicyCount,
			CreatedAt:   r.Service.CreatedAt,
			UpdatedAt:   r.Service.UpdatedAt,
		}
	}

	return result
}
