package tenant

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/maintainerd/maintainerd-auth/internal/platform/middleware"
	"github.com/maintainerd/maintainerd-auth/internal/platform/pagination"
	"github.com/maintainerd/maintainerd-auth/internal/platform/ptr"
	resp "github.com/maintainerd/maintainerd-auth/internal/platform/response"
)

// GetMembers retrieves all members in a tenant.
func (h *TenantHandler) GetMembers(w http.ResponseWriter, r *http.Request) {
	tenant, ok := h.resolveTenantFromRoute(w, r)
	if !ok {
		return
	}

	q := r.URL.Query()
	reqParams := TenantMemberFilterDTO{
		Role:                 ptr.PtrOrNil(q.Get("role")),
		PaginationRequestDTO: pagination.ParseQuery(r),
	}
	if err := reqParams.Validate(); err != nil {
		resp.ValidationError(w, err)
		return
	}

	members, err := h.tenantMemberService.ListByTenant(r.Context(), TenantMemberServiceListFilter{
		TenantID:  tenant.TenantID,
		Role:      reqParams.Role,
		Page:      reqParams.Page,
		Limit:     reqParams.Limit,
		SortBy:    reqParams.SortBy,
		SortOrder: reqParams.SortOrder,
	})
	if err != nil {
		resp.HandleServiceError(w, r, "Failed to fetch members", err)
		return
	}

	rows := make([]TenantMemberResponseDTO, len(members.Data))
	for i, member := range members.Data {
		rows[i] = toTenantMemberResponseDTO(member)
	}

	response := PaginatedResponseDTO[TenantMemberResponseDTO]{
		Rows:       rows,
		Total:      members.Total,
		Page:       members.Page,
		Limit:      members.Limit,
		TotalPages: members.TotalPages,
	}
	resp.Success(w, response, "Members retrieved successfully")
}

// AddMember adds a member to a tenant.
func (h *TenantHandler) AddMember(w http.ResponseWriter, r *http.Request) {
	tenant, ok := h.resolveTenantFromRoute(w, r)
	if !ok {
		return
	}

	var req TenantMemberAddMemberRequestDTO
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		resp.BadRequestBody(w)
		return
	}
	if err := req.Validate(); err != nil {
		resp.ValidationError(w, err)
		return
	}

	auth := middleware.AuthFromRequest(r)
	if auth == nil || auth.User == nil {
		resp.Error(w, http.StatusUnauthorized, "Unauthorized")
		return
	}
	member, err := h.tenantMemberService.CreateByUserUUID(r.Context(), tenant.TenantID, req.UserUUID, req.Role, auth.User.UserID)
	if err != nil {
		resp.HandleServiceError(w, r, "Failed to add member", err)
		return
	}

	response := toTenantMemberResponseDTO(*member)

	var actorUserID *int64
	if auth.User != nil {
		actorUserID = &auth.User.UserID
	}
	changesJSON, _ := json.Marshal(map[string]any{"after": response})
	h.logAudit(r, tenant.TenantID, actorUserID, "tenant_member.create", "tenant_member", member.TenantMemberUUID.String(), &member.TenantMemberUUID, string(changesJSON), "success")

	resp.Created(w, response, "Member added successfully")
}

// UpdateMemberRole updates a member's role in a tenant.
func (h *TenantHandler) UpdateMemberRole(w http.ResponseWriter, r *http.Request) {
	tenant, ok := h.resolveTenantFromRoute(w, r)
	if !ok {
		return
	}

	tenantMemberUUID, ok := parseTenantMemberUUIDFromRoute(w, r)
	if !ok {
		return
	}

	var req TenantMemberUpdateRoleRequestDTO
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		resp.BadRequestBody(w)
		return
	}
	if err := req.Validate(); err != nil {
		resp.ValidationError(w, err)
		return
	}

	auth := middleware.AuthFromRequest(r)
	if auth == nil || auth.User == nil {
		resp.Error(w, http.StatusUnauthorized, "Unauthorized")
		return
	}
	member, err := h.tenantMemberService.UpdateRole(r.Context(), tenant.TenantID, tenantMemberUUID, req.Role, auth.User.UserID)
	if err != nil {
		resp.HandleServiceError(w, r, "Failed to update member role", err)
		return
	}

	response := toTenantMemberResponseDTO(*member)

	var actorUserID *int64
	if auth.User != nil {
		actorUserID = &auth.User.UserID
	}
	changesJSON, _ := json.Marshal(map[string]any{"update": req, "after": response})
	h.logAudit(r, tenant.TenantID, actorUserID, "tenant_member.update_role", "tenant_member", member.TenantMemberUUID.String(), &member.TenantMemberUUID, string(changesJSON), "success")

	resp.Success(w, response, "Member role updated successfully")
}

// RemoveMember removes a member from a tenant.
func (h *TenantHandler) RemoveMember(w http.ResponseWriter, r *http.Request) {
	tenant, ok := h.resolveTenantFromRoute(w, r)
	if !ok {
		return
	}

	tenantMemberUUID, ok := parseTenantMemberUUIDFromRoute(w, r)
	if !ok {
		return
	}

	auth := middleware.AuthFromRequest(r)
	if auth == nil || auth.User == nil {
		resp.Error(w, http.StatusUnauthorized, "Unauthorized")
		return
	}
	if err := h.tenantMemberService.DeleteByUUID(r.Context(), tenant.TenantID, tenantMemberUUID, auth.User.UserID); err != nil {
		resp.HandleServiceError(w, r, "Failed to remove member", err)
		return
	}

	var actorUserID *int64
	if auth.User != nil {
		actorUserID = &auth.User.UserID
	}
	changesJSON, _ := json.Marshal(map[string]any{"before": map[string]any{"id": tenantMemberUUID.String()}})
	h.logAudit(r, tenant.TenantID, actorUserID, "tenant_member.delete", "tenant_member", tenantMemberUUID.String(), &tenantMemberUUID, string(changesJSON), "success")

	resp.Success(w, nil, "Member removed successfully")
}

func (h *TenantHandler) resolveTenantFromRoute(w http.ResponseWriter, r *http.Request) (*TenantServiceDataResult, bool) {
	tenantUUIDStr := chi.URLParam(r, "tenant_uuid")
	if tenantUUIDStr == "" {
		resp.Error(w, http.StatusBadRequest, "Invalid tenant UUID", "UUID parameter is required")
		return nil, false
	}

	tenantUUID, err := uuid.Parse(tenantUUIDStr)
	if err != nil {
		resp.Error(w, http.StatusBadRequest, "Invalid UUID format")
		return nil, false
	}

	tenant, err := h.tenantService.GetByUUID(r.Context(), tenantUUID)
	if err != nil {
		resp.HandleServiceError(w, r, "Tenant not found", err)
		return nil, false
	}

	// Tenant-management access gate: the actor must be a member of this tenant
	// or of the system tenant. This keeps members scoped per-tenant (a regular
	// tenant's user cannot read or modify another tenant's members) while still
	// letting system-tenant admins manage any tenant.
	auth := middleware.AuthFromRequest(r)
	if auth == nil || auth.User == nil {
		resp.Error(w, http.StatusUnauthorized, "Unauthorized")
		return nil, false
	}
	allowed, err := h.tenantMemberService.CanManageTenant(r.Context(), auth.User.UserID, tenantUUID)
	if err != nil {
		resp.HandleServiceError(w, r, "Failed to verify tenant access", err)
		return nil, false
	}
	if !allowed {
		resp.Error(w, http.StatusForbidden, "Access denied", "You do not have access to manage this tenant")
		return nil, false
	}

	return tenant, true
}

func parseTenantMemberUUIDFromRoute(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	tenantMemberUUIDStr := chi.URLParam(r, "tenant_member_uuid")
	if tenantMemberUUIDStr == "" {
		resp.Error(w, http.StatusBadRequest, "Invalid tenant member UUID", "UUID parameter is required")
		return uuid.Nil, false
	}

	tenantMemberUUID, err := uuid.Parse(tenantMemberUUIDStr)
	if err != nil {
		resp.Error(w, http.StatusBadRequest, "Invalid UUID format")
		return uuid.Nil, false
	}
	return tenantMemberUUID, true
}

func toTenantMemberResponseDTO(r TenantMemberServiceDataResult) TenantMemberResponseDTO {
	resp := TenantMemberResponseDTO{
		TenantMemberUUID: r.TenantMemberUUID,
		Role:             r.Role,
		CreatedAt:        r.CreatedAt,
		UpdatedAt:        r.UpdatedAt,
	}

	if r.User != nil {
		userDTO := toMemberUserResponseDTO(*r.User)
		resp.User = &userDTO
	}

	return resp
}

// toMemberUserResponseDTO maps tenant's MemberUser projection to its response DTO.
func toMemberUserResponseDTO(u MemberUser) MemberUserResponseDTO {
	return MemberUserResponseDTO{
		UserUUID:           u.UserUUID,
		Username:           u.Username,
		Fullname:           u.Fullname,
		Email:              u.Email,
		Phone:              u.Phone,
		IsEmailVerified:    u.IsEmailVerified,
		IsPhoneVerified:    u.IsPhoneVerified,
		Status:             u.Status,
		Metadata:           u.Metadata,
		CreatedAt:          u.CreatedAt,
		UpdatedAt:          u.UpdatedAt,
	}
}
