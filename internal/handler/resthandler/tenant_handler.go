package resthandler

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/dto"
	"github.com/maintainerd/auth/internal/middleware"
	"github.com/maintainerd/auth/internal/model"
	"github.com/maintainerd/auth/internal/service"
	"github.com/maintainerd/auth/internal/util"
)

type TenantHandler struct {
	tenantService       service.TenantService
	tenantMemberService service.TenantMemberService
}

func NewTenantHandler(tenantService service.TenantService, tenantMemberService service.TenantMemberService) *TenantHandler {
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
	page, _ := strconv.Atoi(q.Get("page"))
	limit, _ := strconv.Atoi(q.Get("limit"))

	// Parse bools safely
	var isDefault, isSystem, isPublic *bool
	if v := q.Get("is_default"); v != "" {
		parsed, err := strconv.ParseBool(v)
		if err == nil {
			isDefault = &parsed
		}
	}
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
	reqParams := dto.TenantFilterDto{
		Name:        util.PtrOrNil(q.Get("name")),
		DisplayName: util.PtrOrNil(q.Get("display_name")),
		Description: util.PtrOrNil(q.Get("description")),
		Identifier:  util.PtrOrNil(q.Get("identifier")),
		IsDefault:   isDefault,
		IsSystem:    isSystem,
		IsPublic:    isPublic,
		Status:      status,
		PaginationRequestDto: dto.PaginationRequestDto{
			Page:      page,
			Limit:     limit,
			SortBy:    q.Get("sort_by"),
			SortOrder: q.Get("sort_order"),
		},
	}

	if err := reqParams.Validate(); err != nil {
		util.ValidationError(w, err)
		return
	}

	// Build service filter
	tenantFilter := service.TenantServiceGetFilter{
		Name:        reqParams.Name,
		DisplayName: reqParams.DisplayName,
		Description: reqParams.Description,
		Identifier:  reqParams.Identifier,
		IsDefault:   reqParams.IsDefault,
		IsSystem:    reqParams.IsSystem,
		IsPublic:    isPublic,
		Status:      reqParams.Status,
		Page:        reqParams.Page,
		Limit:       reqParams.Limit,
		SortBy:      reqParams.SortBy,
		SortOrder:   reqParams.SortOrder,
	}

	// Fetch Tenants
	result, err := h.tenantService.Get(tenantFilter)
	if err != nil {
		util.Error(w, http.StatusInternalServerError, "Failed to fetch tenants", err.Error())
		return
	}

	// Map tenant result to DTO
	rows := make([]dto.TenantResponseDto, len(result.Data))
	for i, r := range result.Data {
		rows[i] = toTenantResponseDto(r)
	}

	// Build response data
	response := dto.PaginatedResponseDto[dto.TenantResponseDto]{
		Rows:       rows,
		Total:      result.Total,
		Page:       result.Page,
		Limit:      result.Limit,
		TotalPages: result.TotalPages,
	}

	util.Success(w, response, "Tenants fetched successfully")
}

// Get Tenant by UUID
func (h *TenantHandler) GetByUUID(w http.ResponseWriter, r *http.Request) {
	tenantUUID, err := uuid.Parse(chi.URLParam(r, "tenant_uuid"))
	if err != nil {
		util.Error(w, http.StatusBadRequest, "Invalid Tenant UUID")
		return
	}

	tenant, err := h.tenantService.GetByUUID(tenantUUID)
	if err != nil {
		util.Error(w, http.StatusNotFound, "Tenant not found")
		return
	}

	dtoRes := toTenantResponseDto(*tenant)

	util.Success(w, dtoRes, "Tenant fetched successfully")
}

// Get Default Tenant
func (h *TenantHandler) GetDefault(w http.ResponseWriter, r *http.Request) {
	tenant, err := h.tenantService.GetDefault()
	if err != nil {
		util.Error(w, http.StatusNotFound, "Default tenant not found")
		return
	}

	dtoRes := toTenantResponseDto(*tenant)

	util.Success(w, dtoRes, "Default tenant fetched successfully")
}

// Get Tenant by Identifier
func (h *TenantHandler) GetByIdentifier(w http.ResponseWriter, r *http.Request) {
	identifier := chi.URLParam(r, "identifier")
	if identifier == "" {
		util.Error(w, http.StatusBadRequest, "Identifier is required")
		return
	}

	tenant, err := h.tenantService.GetByIdentifier(identifier)
	if err != nil {
		util.Error(w, http.StatusNotFound, "Tenant not found")
		return
	}

	dtoRes := toTenantResponseDto(*tenant)

	util.Success(w, dtoRes, "Tenant fetched successfully")
}

// Create Tenant
func (h *TenantHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req dto.TenantCreateRequestDto
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		util.Error(w, http.StatusBadRequest, "Invalid request", err.Error())
		return
	}

	if err := req.Validate(); err != nil {
		util.ValidationError(w, err)
		return
	}

	tenant, err := h.tenantService.Create(req.Name, req.DisplayName, req.Description, req.Status, req.IsPublic, false)
	if err != nil {
		util.Error(w, http.StatusInternalServerError, "Failed to create tenant", err.Error())
		return
	}

	dtoRes := toTenantResponseDto(*tenant)

	util.Created(w, dtoRes, "Tenant created successfully")
}

// Update Tenant
func (h *TenantHandler) Update(w http.ResponseWriter, r *http.Request) {
	user, ok := r.Context().Value(middleware.UserContextKey).(*model.User)
	if !ok || user == nil {
		util.Error(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	tenantUUID, err := uuid.Parse(chi.URLParam(r, "tenant_uuid"))
	if err != nil {
		util.Error(w, http.StatusBadRequest, "Invalid tenant UUID")
		return
	}

	// Check if user is a member of this tenant
	isMember, err := h.tenantMemberService.IsUserInTenant(user.UserID, tenantUUID)
	if err != nil {
		util.Error(w, http.StatusInternalServerError, "Failed to verify tenant membership", err.Error())
		return
	}
	if !isMember {
		util.Error(w, http.StatusForbidden, "Access denied", "Only tenant members can update this tenant")
		return
	}

	var req dto.TenantUpdateRequestDto
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		util.Error(w, http.StatusBadRequest, "Invalid request", err.Error())
		return
	}

	if err := req.Validate(); err != nil {
		util.ValidationError(w, err)
		return
	}

	tenant, err := h.tenantService.Update(tenantUUID, req.Name, req.DisplayName, req.Description, req.Status, req.IsPublic)
	if err != nil {
		util.Error(w, http.StatusInternalServerError, "Failed to update tenant", err.Error())
		return
	}

	dtoRes := toTenantResponseDto(*tenant)

	util.Success(w, dtoRes, "Tenant updated successfully")
}

// Set Tenant status
func (h *TenantHandler) SetStatus(w http.ResponseWriter, r *http.Request) {
	tenantUUID, err := uuid.Parse(chi.URLParam(r, "tenant_uuid"))
	if err != nil {
		util.Error(w, http.StatusBadRequest, "Invalid tenant UUID")
		return
	}

	var req struct {
		Status string `json:"status"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		util.Error(w, http.StatusBadRequest, "Invalid request", err.Error())
		return
	}

	tenant, err := h.tenantService.SetStatusByUUID(tenantUUID, req.Status)
	if err != nil {
		util.Error(w, http.StatusInternalServerError, "Failed to update API", err.Error())
		return
	}

	dtoRes := toTenantResponseDto(*tenant)

	util.Success(w, dtoRes, "Tenant status updated successfully")
}

// Set Tenant public
func (h *TenantHandler) SetPublic(w http.ResponseWriter, r *http.Request) {
	tenantUUID, err := uuid.Parse(chi.URLParam(r, "tenant_uuid"))
	if err != nil {
		util.Error(w, http.StatusBadRequest, "Invalid tenant UUID")
		return
	}

	tenant, err := h.tenantService.SetActivePublicByUUID(tenantUUID)
	if err != nil {
		util.Error(w, http.StatusInternalServerError, "Failed to update API", err.Error())
		return
	}

	dtoRes := toTenantResponseDto(*tenant)

	util.Success(w, dtoRes, "Tenant public updated successfully")
}

// Set Tenant default
func (h *TenantHandler) SetDefault(w http.ResponseWriter, r *http.Request) {
	tenantUUID, err := uuid.Parse(chi.URLParam(r, "tenant_uuid"))
	if err != nil {
		util.Error(w, http.StatusBadRequest, "Invalid tenant UUID")
		return
	}

	tenant, err := h.tenantService.SetDefaultStatusByUUID(tenantUUID)
	if err != nil {
		util.Error(w, http.StatusInternalServerError, "Failed to update tenant", err.Error())
		return
	}

	dtoRes := toTenantResponseDto(*tenant)

	util.Success(w, dtoRes, "Tenant default updated successfully")
}

// Delete Tenant
func (h *TenantHandler) Delete(w http.ResponseWriter, r *http.Request) {
	user, ok := r.Context().Value(middleware.UserContextKey).(*model.User)
	if !ok || user == nil {
		util.Error(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	tenantUUID, err := uuid.Parse(chi.URLParam(r, "tenant_uuid"))
	if err != nil {
		util.Error(w, http.StatusBadRequest, "Invalid tenant UUID")
		return
	}

	// Check if user is a member of this tenant
	isMember, err := h.tenantMemberService.IsUserInTenant(user.UserID, tenantUUID)
	if err != nil {
		util.Error(w, http.StatusInternalServerError, "Failed to verify tenant membership", err.Error())
		return
	}
	if !isMember {
		util.Error(w, http.StatusForbidden, "Access denied", "Only tenant members can delete this tenant")
		return
	}

	// Get tenant to check if it's a system tenant
	tenant, err := h.tenantService.GetByUUID(tenantUUID)
	if err != nil {
		util.Error(w, http.StatusNotFound, "Tenant not found", err.Error())
		return
	}

	// Prevent deletion of system tenants
	if tenant.IsSystem {
		util.Error(w, http.StatusForbidden, "Cannot delete system tenant", "System tenants cannot be deleted")
		return
	}

	deletedTenant, err := h.tenantService.DeleteByUUID(tenantUUID)
	if err != nil {
		util.Error(w, http.StatusInternalServerError, "Failed to delete tenant", err.Error())
		return
	}

	dtoRes := toTenantResponseDto(*deletedTenant)

	util.Success(w, dtoRes, "Tenant deleted successfully")
}

// GetMembers retrieves all members in a tenant
func (h *TenantHandler) GetMembers(w http.ResponseWriter, r *http.Request) {
	tenantUUIDStr := chi.URLParam(r, "tenant_uuid")
	if tenantUUIDStr == "" {
		util.Error(w, http.StatusBadRequest, "Invalid tenant UUID", "UUID parameter is required")
		return
	}

	tenantUUID, err := uuid.Parse(tenantUUIDStr)
	if err != nil {
		util.Error(w, http.StatusBadRequest, "Invalid UUID format", err.Error())
		return
	}

	// Parse query parameters
	q := r.URL.Query()

	// Parse pagination
	page, _ := strconv.Atoi(q.Get("page"))
	limit, _ := strconv.Atoi(q.Get("limit"))

	// Build request DTO
	reqParams := dto.TenantMemberFilterDto{
		Role: util.PtrOrNil(q.Get("role")),
		PaginationRequestDto: dto.PaginationRequestDto{
			Page:      page,
			Limit:     limit,
			SortBy:    q.Get("sort_by"),
			SortOrder: q.Get("sort_order"),
		},
	}

	if err := reqParams.Validate(); err != nil {
		util.ValidationError(w, err)
		return
	}

	// Get tenant to retrieve tenant_id
	tenant, err := h.tenantService.GetByUUID(tenantUUID)
	if err != nil {
		util.Error(w, http.StatusNotFound, "Tenant not found", err.Error())
		return
	}

	members, err := h.tenantMemberService.ListByTenant(tenant.TenantID)
	if err != nil {
		util.Error(w, http.StatusInternalServerError, "Failed to fetch members", err.Error())
		return
	}

	response := make([]dto.TenantMemberResponseDto, len(members))
	for i, member := range members {
		response[i] = toTenantMemberResponseDto(member)
	}

	util.Success(w, response, "Members retrieved successfully")
}

// AddMember adds a member to a tenant
func (h *TenantHandler) AddMember(w http.ResponseWriter, r *http.Request) {
	tenantUUIDStr := chi.URLParam(r, "tenant_uuid")
	if tenantUUIDStr == "" {
		util.Error(w, http.StatusBadRequest, "Invalid tenant UUID", "UUID parameter is required")
		return
	}

	tenantUUID, err := uuid.Parse(tenantUUIDStr)
	if err != nil {
		util.Error(w, http.StatusBadRequest, "Invalid UUID format", err.Error())
		return
	}

	var req dto.TenantMemberAddMemberRequestDto
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		util.Error(w, http.StatusBadRequest, "Invalid request body", err.Error())
		return
	}

	if err := req.Validate(); err != nil {
		util.ValidationError(w, err)
		return
	}

	// Get tenant to retrieve tenant_id
	tenant, err := h.tenantService.GetByUUID(tenantUUID)
	if err != nil {
		util.Error(w, http.StatusNotFound, "Tenant not found", err.Error())
		return
	}

	member, err := h.tenantMemberService.CreateByUserUUID(tenant.TenantID, req.UserUUID, req.Role)
	if err != nil {
		util.Error(w, http.StatusBadRequest, "Failed to add member", err.Error())
		return
	}

	response := toTenantMemberResponseDto(*member)
	util.Created(w, response, "Member added successfully")
}

// UpdateMemberRole updates a member's role in a tenant
func (h *TenantHandler) UpdateMemberRole(w http.ResponseWriter, r *http.Request) {
	tenantMemberUUIDStr := chi.URLParam(r, "tenant_member_uuid")
	if tenantMemberUUIDStr == "" {
		util.Error(w, http.StatusBadRequest, "Invalid tenant member UUID", "UUID parameter is required")
		return
	}

	tenantMemberUUID, err := uuid.Parse(tenantMemberUUIDStr)
	if err != nil {
		util.Error(w, http.StatusBadRequest, "Invalid UUID format", err.Error())
		return
	}

	var req dto.TenantMemberUpdateRoleRequestDto
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		util.Error(w, http.StatusBadRequest, "Invalid request body", err.Error())
		return
	}

	if err := req.Validate(); err != nil {
		util.ValidationError(w, err)
		return
	}

	member, err := h.tenantMemberService.UpdateRole(tenantMemberUUID, req.Role)
	if err != nil {
		util.Error(w, http.StatusBadRequest, "Failed to update member role", err.Error())
		return
	}

	response := toTenantMemberResponseDto(*member)
	util.Success(w, response, "Member role updated successfully")
}

// RemoveMember removes a member from a tenant
func (h *TenantHandler) RemoveMember(w http.ResponseWriter, r *http.Request) {
	tenantMemberUUIDStr := chi.URLParam(r, "tenant_member_uuid")
	if tenantMemberUUIDStr == "" {
		util.Error(w, http.StatusBadRequest, "Invalid tenant member UUID", "UUID parameter is required")
		return
	}

	tenantMemberUUID, err := uuid.Parse(tenantMemberUUIDStr)
	if err != nil {
		util.Error(w, http.StatusBadRequest, "Invalid UUID format", err.Error())
		return
	}

	if err := h.tenantMemberService.DeleteByUUID(tenantMemberUUID); err != nil {
		util.Error(w, http.StatusBadRequest, "Failed to remove member", err.Error())
		return
	}

	util.Success(w, nil, "Member removed successfully")
}

// Convert service result to DTO
func toTenantResponseDto(r service.TenantServiceDataResult) dto.TenantResponseDto {
	result := dto.TenantResponseDto{
		TenantUUID:  r.TenantUUID,
		Name:        r.Name,
		DisplayName: r.DisplayName,
		Description: r.Description,
		Identifier:  r.Identifier,
		Status:      r.Status,
		IsPublic:    r.IsPublic,
		IsDefault:   r.IsDefault,
		IsSystem:    r.IsSystem,
		CreatedAt:   r.CreatedAt,
		UpdatedAt:   r.UpdatedAt,
	}

	return result
}

func toTenantMemberResponseDto(r service.TenantMemberServiceDataResult) dto.TenantMemberResponseDto {
	resp := dto.TenantMemberResponseDto{
		TenantMemberUUID: r.TenantMemberUUID,
		Role:             r.Role,
		CreatedAt:        r.CreatedAt,
		UpdatedAt:        r.UpdatedAt,
	}

	if r.User != nil {
		userDto := toUserResponseDto(*r.User)
		resp.User = &userDto
	}

	return resp
}
