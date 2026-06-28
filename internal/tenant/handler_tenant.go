package tenant

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/branding"
	"github.com/maintainerd/auth/internal/platform/middleware"
	"github.com/maintainerd/auth/internal/platform/pagination"
	"github.com/maintainerd/auth/internal/platform/ptr"
	resp "github.com/maintainerd/auth/internal/platform/response"
	"github.com/maintainerd/auth/internal/secpolicy"
)

type TenantHandler struct {
	tenantService          TenantService
	tenantMemberService    TenantMemberService
	brandingService        branding.BrandingService
	securitySettingService secpolicy.SecuritySettingService
}

func NewTenantHandler(
	tenantService TenantService,
	tenantMemberService TenantMemberService,
	brandingService branding.BrandingService,
	securitySettingService secpolicy.SecuritySettingService,
) *TenantHandler {
	return &TenantHandler{
		tenantService:          tenantService,
		tenantMemberService:    tenantMemberService,
		brandingService:        brandingService,
		securitySettingService: securitySettingService,
	}
}

// Get all tenants with pagination
func (h *TenantHandler) Get(w http.ResponseWriter, r *http.Request) {
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
		Status:               status,
		PaginationRequestDTO: pagination.ParseQuery(r),
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
		Status:      reqParams.Status,
		Page:        reqParams.Page,
		Limit:       reqParams.Limit,
		SortBy:      reqParams.SortBy,
		SortOrder:   reqParams.SortOrder,
	}

	// Scope the listing: members of the system tenant see all tenants; everyone
	// else sees only their own (context) tenant. This keeps tenant records
	// tenant-bound while letting system-tenant admins enumerate every tenant.
	auth := middleware.AuthFromRequest(r)
	if auth == nil || auth.Tenant == nil {
		resp.Error(w, http.StatusUnauthorized, "Unauthorized")
		return
	}
	systemTenant, err := h.tenantService.GetSystem(r.Context())
	if err != nil {
		resp.HandleServiceError(w, r, "Failed to resolve system tenant", err)
		return
	}
	if systemTenant == nil || auth.Tenant.TenantID != systemTenant.TenantID {
		tenantFilter.TenantIDs = []int64{auth.Tenant.TenantID}
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
	auth := middleware.AuthFromRequest(r)
	if auth == nil || auth.Tenant == nil {
		resp.Error(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	tenantUUID, err := uuid.Parse(chi.URLParam(r, "tenant_uuid"))
	if err != nil {
		resp.Error(w, http.StatusBadRequest, "Invalid Tenant UUID")
		return
	}

	if !h.isSystemTenantMember(r) && auth.Tenant.TenantUUID != tenantUUID {
		resp.Error(w, http.StatusForbidden, "Access denied", "You can only view your own tenant")
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

	dtoRes := h.toPublicResponse(r.Context(), *tenant)

	resp.Success(w, dtoRes, "System tenant fetched successfully")
}

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

	dtoRes := h.toPublicResponse(r.Context(), *tenant)

	resp.Success(w, dtoRes, "Tenant fetched successfully")
}

func (h *TenantHandler) toPublicResponse(ctx context.Context, tenant TenantServiceDataResult) TenantPublicResponseDTO {
	res := TenantPublicResponseDTO{
		Identifier:  tenant.Identifier,
		Name:        tenant.Name,
		DisplayName: tenant.DisplayName,
		Description: tenant.Description,
		Status:      tenant.Status,
		IsSystem:    tenant.IsSystem,
	}

	if h.securitySettingService != nil {
		if pwd, err := h.securitySettingService.GetPasswordConfig(ctx, tenant.TenantID); err == nil {
			res.PasswordConfig = &PasswordConfigPublic{
				MinLength:        intFromMap(pwd, "min_length"),
				MaxLength:        intFromMap(pwd, "max_length"),
				RequireUppercase: boolFromMap(pwd, "require_uppercase"),
				RequireLowercase: boolFromMap(pwd, "require_lowercase"),
				RequireNumber:    boolFromMap(pwd, "require_number"),
				RequireSymbol:    boolFromMap(pwd, "require_symbol"),
			}
		}
		if reg, err := h.securitySettingService.GetRegistrationConfig(ctx, tenant.TenantID); err == nil {
			res.RegistrationConfig = &RegistrationConfigPublic{
				SelfRegistrationEnabled:  boolFromMap(reg, "self_registration_enabled"),
				RequireEmailVerification: boolFromMap(reg, "require_email_verification"),
				CaptchaOnSignup:          boolFromMap(reg, "captcha_on_signup"),
			}
		}
	}

	if h.brandingService != nil {
		if b, err := h.brandingService.GetPublic(ctx, tenant.TenantID); err == nil && b != nil {
			res.Branding = &BrandingPublic{
				Layout:            b.Layout,
				CompanyName:       b.CompanyName,
				LogoURL:           b.LogoURL,
				FaviconURL:        b.FaviconURL,
				SupportURL:        b.SupportURL,
				PrivacyPolicyURL:  b.PrivacyPolicyURL,
				TermsOfServiceURL: b.TermsOfServiceURL,
				Metadata:          b.Metadata,
			}
		}
	}

	return res
}

func intFromMap(m map[string]any, key string) int {
	if v, ok := m[key]; ok {
		switch n := v.(type) {
		case float64:
			return int(n)
		case int:
			return n
		}
	}
	return 0
}

func boolFromMap(m map[string]any, key string) bool {
	if v, ok := m[key]; ok {
		if b, ok := v.(bool); ok {
			return b
		}
	}
	return false
}

// Create Tenant
func (h *TenantHandler) Create(w http.ResponseWriter, r *http.Request) {
	// Only members of the system tenant may create tenants. A user's context
	// tenant is resolved from their authenticated identity, so a regular-tenant
	// user is forbidden here even if their own tenant's super-admin role carries
	// the tenant:create permission.
	auth := middleware.AuthFromRequest(r)
	if auth == nil || auth.Tenant == nil {
		resp.Error(w, http.StatusUnauthorized, "Unauthorized")
		return
	}
	systemTenant, err := h.tenantService.GetSystem(r.Context())
	if err != nil {
		resp.HandleServiceError(w, r, "Failed to resolve system tenant", err)
		return
	}
	if systemTenant == nil || auth.Tenant.TenantID != systemTenant.TenantID {
		resp.Error(w, http.StatusForbidden, "Access denied", "Only members of the system tenant can create tenants")
		return
	}

	var req TenantCreateRequestDTO
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		resp.BadRequestBody(w)
		return
	}

	if err := req.Validate(); err != nil {
		resp.ValidationError(w, err)
		return
	}

	tenant, err := h.tenantService.Create(r.Context(), req.Name, req.DisplayName, req.Description, req.Status)
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

	// Tenant-management access: a member of this tenant or of the system tenant.
	canManage, err := h.tenantMemberService.CanManageTenant(r.Context(), user.UserID, tenantUUID)
	if err != nil {
		resp.HandleServiceError(w, r, "Failed to verify tenant access", err)
		return
	}
	if !canManage {
		resp.Error(w, http.StatusForbidden, "Access denied", "You do not have access to update this tenant")
		return
	}

	var req TenantUpdateRequestDTO
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		resp.BadRequestBody(w)
		return
	}

	if err := req.Validate(); err != nil {
		resp.ValidationError(w, err)
		return
	}

	tenant, err := h.tenantService.Update(r.Context(), tenantUUID, req.Name, req.DisplayName, req.Description, req.Status)
	if err != nil {
		resp.HandleServiceError(w, r, "Failed to update tenant", err)
		return
	}

	dtoRes := toTenantResponseDTO(*tenant)

	resp.Success(w, dtoRes, "Tenant updated successfully")
}

// Set Tenant status
func (h *TenantHandler) SetStatus(w http.ResponseWriter, r *http.Request) {
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

	canManage, err := h.tenantMemberService.CanManageTenant(r.Context(), user.UserID, tenantUUID)
	if err != nil {
		resp.HandleServiceError(w, r, "Failed to verify tenant access", err)
		return
	}
	if !canManage {
		resp.Error(w, http.StatusForbidden, "Access denied", "You do not have access to update this tenant")
		return
	}

	var req TenantSetStatusRequestDTO
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		resp.BadRequestBody(w)
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

// Delete Tenant
func (h *TenantHandler) Delete(w http.ResponseWriter, r *http.Request) {
	auth := middleware.AuthFromRequest(r)
	if auth == nil || auth.Tenant == nil || auth.User == nil {
		resp.Error(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	tenantUUID, err := uuid.Parse(chi.URLParam(r, "tenant_uuid"))
	if err != nil {
		resp.Error(w, http.StatusBadRequest, "Invalid tenant UUID")
		return
	}

	systemTenant, err := h.tenantService.GetSystem(r.Context())
	if err != nil {
		resp.HandleServiceError(w, r, "Failed to resolve system tenant", err)
		return
	}
	if systemTenant == nil || auth.Tenant.TenantID != systemTenant.TenantID {
		resp.Error(w, http.StatusForbidden, "Access denied", "Only members of the system tenant can delete tenants")
		return
	}

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

	deletedTenant, err := h.tenantService.DeleteByUUID(r.Context(), tenantUUID, auth.User.UserID)
	if err != nil {
		resp.HandleServiceError(w, r, "Failed to delete tenant", err)
		return
	}

	dtoRes := toTenantResponseDTO(*deletedTenant)

	resp.Success(w, dtoRes, "Tenant deleted successfully")
}

func (h *TenantHandler) isSystemTenantMember(r *http.Request) bool {
	auth := middleware.AuthFromRequest(r)
	if auth == nil || auth.Tenant == nil {
		return false
	}
	systemTenant, err := h.tenantService.GetSystem(r.Context())
	if err != nil || systemTenant == nil {
		return false
	}
	return auth.Tenant.TenantID == systemTenant.TenantID
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
		IsSystem:    r.IsSystem,
		Metadata:    r.Metadata,
		CreatedAt:   r.CreatedAt,
		UpdatedAt:   r.UpdatedAt,
	}

	return result
}
