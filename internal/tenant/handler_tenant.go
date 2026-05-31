package tenant

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/platform/middleware"
	"github.com/maintainerd/auth/internal/platform/ptr"
	resp "github.com/maintainerd/auth/internal/platform/response"
)

type TenantHandler struct {
	tenantService       TenantService
	tenantMemberService TenantMemberService
}

func NewTenantHandler(tenantService TenantService, tenantMemberService TenantMemberService) *TenantHandler {
	return &TenantHandler{
		tenantService:       tenantService,
		tenantMemberService: tenantMemberService,
	}
}

// Get all tenants with pagination
func (h *TenantHandler) Get(w http.ResponseWriter, r *http.Request) {
	// Parse query parameters
	q := r.URL.Query()

	// Parse pagination

	// Parse bools safely
	var isSystem, isPublic *bool
	if v := q.Get("is_system"); v != "" {
		parsed, err := strconv.ParseBool(v)
		if err == nil {
			isSystem = &parsed
		}
	}
	if v := q.Get("is_public"); v != "" {
		parsed, err := strconv.ParseBool(v)
		if err == nil {
			isPublic = &parsed
		}
	}

	// Parse status array
	var status []string
	if v := q.Get("status"); v != "" {
		status = strings.Split(v, ",")
	}

	// Build request DTO
	reqParams := TenantFilterDTO{
		Name:                 ptr.PtrOrNil(q.Get("name")),
		DisplayName:          ptr.PtrOrNil(q.Get("display_name")),
		Description:          ptr.PtrOrNil(q.Get("description")),
		Identifier:           ptr.PtrOrNil(q.Get("identifier")),
		IsSystem:             isSystem,
		IsPublic:             isPublic,
		Status:               status,
		PaginationRequestDTO: parsePaginationQuery(r),
	}

	if err := reqParams.Validate(); err != nil {
		resp.ValidationError(w, err)
		return
	}

	// Build service filter
	tenantFilter := TenantServiceGetFilter{
		Name:        reqParams.Name,
		DisplayName: reqParams.DisplayName,
		Description: reqParams.Description,
		Identifier:  reqParams.Identifier,
		IsSystem:    reqParams.IsSystem,
		IsPublic:    isPublic,
		Status:      reqParams.Status,
		Page:        reqParams.Page,
		Limit:       reqParams.Limit,
		SortBy:      reqParams.SortBy,
		SortOrder:   reqParams.SortOrder,
	}

	// Fetch Tenants
	result, err := h.tenantService.Get(r.Context(), tenantFilter)
	if err != nil {
		resp.HandleServiceError(w, r, "Failed to fetch tenants", err)
		return
	}

	// Map tenant result to DTO
	rows := make([]TenantResponseDTO, len(result.Data))
	for i, r := range result.Data {
		rows[i] = toTenantResponseDTO(r)
	}

	// Build response data
	response := PaginatedResponseDTO[TenantResponseDTO]{
		Rows:       rows,
		Total:      result.Total,
		Page:       result.Page,
		Limit:      result.Limit,
		TotalPages: result.TotalPages,
	}

	resp.Success(w, response, "Tenants fetched successfully")
}

// Get Tenant by UUID
func (h *TenantHandler) GetByUUID(w http.ResponseWriter, r *http.Request) {
	tenantUUID, err := uuid.Parse(chi.URLParam(r, "tenant_uuid"))
	if err != nil {
		resp.Error(w, http.StatusBadRequest, "Invalid Tenant UUID")
		return
	}

	tenant, err := h.tenantService.GetByUUID(r.Context(), tenantUUID)
	if err != nil {
		resp.HandleServiceError(w, r, "Tenant not found", err)
		return
	}

	dtoRes := toTenantResponseDTO(*tenant)

	resp.Success(w, dtoRes, "Tenant fetched successfully")
}

// GetDefault returns the system tenant, which is the root of the tenant hierarchy.
func (h *TenantHandler) GetDefault(w http.ResponseWriter, r *http.Request) {
	tenant, err := h.tenantService.GetSystem(r.Context())
	if err != nil {
		resp.HandleServiceError(w, r, "System tenant not found", err)
		return
	}

	dtoRes := toTenantResponseDTO(*tenant)

	resp.Success(w, dtoRes, "System tenant fetched successfully")
}

// Get Tenant by Identifier
func (h *TenantHandler) GetByIdentifier(w http.ResponseWriter, r *http.Request) {
	identifier := chi.URLParam(r, "identifier")
	if identifier == "" {
		resp.Error(w, http.StatusBadRequest, "Identifier is required")
		return
	}

	tenant, err := h.tenantService.GetByIdentifier(r.Context(), identifier)
	if err != nil {
		resp.HandleServiceError(w, r, "Tenant not found", err)
		return
	}

	dtoRes := toTenantResponseDTO(*tenant)

	resp.Success(w, dtoRes, "Tenant fetched successfully")
}

// Create Tenant
func (h *TenantHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req TenantCreateRequestDTO
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		resp.Error(w, http.StatusBadRequest, "Invalid request")
		return
	}

	if err := req.Validate(); err != nil {
		resp.ValidationError(w, err)
		return
	}

	tenant, err := h.tenantService.Create(r.Context(), req.Name, req.DisplayName, req.Description, req.Status, req.IsPublic)
	if err != nil {
		resp.HandleServiceError(w, r, "Failed to create tenant", err)
		return
	}

	dtoRes := toTenantResponseDTO(*tenant)

	resp.Created(w, dtoRes, "Tenant created successfully")
}

// Update Tenant
func (h *TenantHandler) Update(w http.ResponseWriter, r *http.Request) {
	user := middleware.AuthFromRequest(r).User
	if user == nil {
		resp.Error(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	tenantUUID, err := uuid.Parse(chi.URLParam(r, "tenant_uuid"))
	if err != nil {
		resp.Error(w, http.StatusBadRequest, "Invalid tenant UUID")
		return
	}

	// Check if user is a member of this tenant
	isMember, err := h.tenantMemberService.IsUserInTenant(r.Context(), user.UserID, tenantUUID)
	if err != nil {
		resp.HandleServiceError(w, r, "Failed to verify tenant membership", err)
		return
	}
	if !isMember {
		resp.Error(w, http.StatusForbidden, "Access denied", "Only tenant members can update this tenant")
		return
	}

	var req TenantUpdateRequestDTO
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		resp.Error(w, http.StatusBadRequest, "Invalid request")
		return
	}

	if err := req.Validate(); err != nil {
		resp.ValidationError(w, err)
		return
	}

	tenant, err := h.tenantService.Update(r.Context(), tenantUUID, req.Name, req.DisplayName, req.Description, req.Status, req.IsPublic)
	if err != nil {
		resp.HandleServiceError(w, r, "Failed to update tenant", err)
		return
	}

	dtoRes := toTenantResponseDTO(*tenant)

	resp.Success(w, dtoRes, "Tenant updated successfully")
}

// Set Tenant status
func (h *TenantHandler) SetStatus(w http.ResponseWriter, r *http.Request) {
	tenantUUID, err := uuid.Parse(chi.URLParam(r, "tenant_uuid"))
	if err != nil {
		resp.Error(w, http.StatusBadRequest, "Invalid tenant UUID")
		return
	}

	var req TenantSetStatusRequestDTO
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		resp.Error(w, http.StatusBadRequest, "Invalid request")
		return
	}
	if err := req.Validate(); err != nil {
		resp.ValidationError(w, err)
		return
	}

	tenant, err := h.tenantService.SetStatusByUUID(r.Context(), tenantUUID, req.Status)
	if err != nil {
		resp.HandleServiceError(w, r, "Failed to update tenant status", err)
		return
	}

	dtoRes := toTenantResponseDTO(*tenant)

	resp.Success(w, dtoRes, "Tenant status updated successfully")
}

// Set Tenant public
func (h *TenantHandler) SetPublic(w http.ResponseWriter, r *http.Request) {
	tenantUUID, err := uuid.Parse(chi.URLParam(r, "tenant_uuid"))
	if err != nil {
		resp.Error(w, http.StatusBadRequest, "Invalid tenant UUID")
		return
	}

	tenant, err := h.tenantService.SetActivePublicByUUID(r.Context(), tenantUUID)
	if err != nil {
		resp.HandleServiceError(w, r, "Failed to update tenant", err)
		return
	}

	dtoRes := toTenantResponseDTO(*tenant)

	resp.Success(w, dtoRes, "Tenant public updated successfully")
}

// Delete Tenant
func (h *TenantHandler) Delete(w http.ResponseWriter, r *http.Request) {
	user := middleware.AuthFromRequest(r).User
	if user == nil {
		resp.Error(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	tenantUUID, err := uuid.Parse(chi.URLParam(r, "tenant_uuid"))
	if err != nil {
		resp.Error(w, http.StatusBadRequest, "Invalid tenant UUID")
		return
	}

	// Check if user is a member of this tenant
	isMember, err := h.tenantMemberService.IsUserInTenant(r.Context(), user.UserID, tenantUUID)
	if err != nil {
		resp.HandleServiceError(w, r, "Failed to verify tenant membership", err)
		return
	}
	if !isMember {
		resp.Error(w, http.StatusForbidden, "Access denied", "Only tenant members can delete this tenant")
		return
	}

	// Get tenant to check if it's a system tenant
	tenant, err := h.tenantService.GetByUUID(r.Context(), tenantUUID)
	if err != nil {
		resp.HandleServiceError(w, r, "Tenant not found", err)
		return
	}

	// Prevent deletion of system tenants
	if tenant.IsSystem {
		resp.Error(w, http.StatusForbidden, "Cannot delete system tenant", "System tenants cannot be deleted")
		return
	}

	deletedTenant, err := h.tenantService.DeleteByUUID(r.Context(), tenantUUID)
	if err != nil {
		resp.HandleServiceError(w, r, "Failed to delete tenant", err)
		return
	}

	dtoRes := toTenantResponseDTO(*deletedTenant)

	resp.Success(w, dtoRes, "Tenant deleted successfully")
}

// Convert service result to DTO
func toTenantResponseDTO(r TenantServiceDataResult) TenantResponseDTO {
	result := TenantResponseDTO{
		TenantUUID:  r.TenantUUID,
		Name:        r.Name,
		DisplayName: r.DisplayName,
		Description: r.Description,
		Identifier:  r.Identifier,
		Status:      r.Status,
		IsPublic:    r.IsPublic,
		IsSystem:    r.IsSystem,
		CreatedAt:   r.CreatedAt,
		UpdatedAt:   r.UpdatedAt,
	}

	return result
}
