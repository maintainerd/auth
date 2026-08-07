package tenant

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/maintainerd/maintainerd-auth/internal/auditlog"
	"github.com/maintainerd/maintainerd-auth/internal/branding"
	"github.com/maintainerd/maintainerd-auth/internal/platform/middleware"
	"github.com/maintainerd/maintainerd-auth/internal/platform/pagination"
	"github.com/maintainerd/maintainerd-auth/internal/platform/ptr"
	resp "github.com/maintainerd/maintainerd-auth/internal/platform/response"
	"github.com/maintainerd/maintainerd-auth/internal/secpolicy"
	"github.com/maintainerd/maintainerd-auth/internal/shared"
)

type TenantHandler struct {
	tenantService          TenantService
	tenantMemberService    TenantMemberService
	brandingService        branding.BrandingService
	securitySettingService secpolicy.SecuritySettingService
	clientReader           SurfaceClientReader
	clientBrandingReader   PublicClientBrandingReader
	connectionsReader      SurfaceConnectionsReader
	auditLogger            auditlog.ManagementAuditLogger
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

func (h *TenantHandler) SetAuditLogger(l auditlog.ManagementAuditLogger) { h.auditLogger = l }

// SetSurfaceClientReader injects the resolver used by the domain-bootstrap
// endpoint to advertise a tenant's seeded system client per surface. It is
// optional: when unset, the bootstrap response simply omits the client field.
func (h *TenantHandler) SetSurfaceClientReader(r SurfaceClientReader) { h.clientReader = r }

// SetClientBrandingReader injects the optional resolver used by the domain
// bootstrap endpoint when an OAuth client_id should select a client-attached
// theme before falling back to the tenant's active theme.
func (h *TenantHandler) SetClientBrandingReader(r PublicClientBrandingReader) {
	h.clientBrandingReader = r
}

// SetSurfaceConnectionsReader injects the resolver that lists the federated
// login options enabled on the resolved surface client. It is optional: when
// unset, the bootstrap response carries an empty connections array rather than
// failing, so a misconfigured wiring degrades to password-only login.
func (h *TenantHandler) SetSurfaceConnectionsReader(r SurfaceConnectionsReader) {
	h.connectionsReader = r
}

func (h *TenantHandler) logAudit(r *http.Request, tenantID int64, actorUserID *int64, action, resourceType, resourceID string, resourceUUID *uuid.UUID, changes, outcome string) {
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

// GetDefault serves GET /tenant. With ?domain=<host> it becomes the
// domain-based bootstrap endpoint (resolve host → tenant + surface + client).
// Without ?domain it keeps the original behaviour: return the system tenant.
func (h *TenantHandler) GetDefault(w http.ResponseWriter, r *http.Request) {
	if domain := strings.TrimSpace(r.URL.Query().Get("domain")); domain != "" {
		h.getBootstrap(w, r, domain)
		return
	}

	tenant, err := h.tenantService.GetSystem(r.Context())
	if err != nil {
		resp.HandleServiceError(w, r, "System tenant not found", err)
		return
	}

	dtoRes := h.toPublicResponse(r.Context(), *tenant)

	resp.Success(w, dtoRes, "System tenant fetched successfully")
}

// getBootstrap resolves an incoming host to its tenant + surface and returns the
// public bootstrap payload the frontend needs before login: tenant identity,
// canonical per-surface URLs, public branding, and the seeded system client for
// the resolved surface. It is a public (unauthenticated) endpoint and returns
// only public fields.
func (h *TenantHandler) getBootstrap(w http.ResponseWriter, r *http.Request, domain string) {
	surface, slug, isSystem, ok := shared.ResolveTenantHost(domain)
	if !ok {
		resp.Error(w, http.StatusNotFound, "Unknown domain")
		return
	}

	var (
		tenant *TenantServiceDataResult
		err    error
	)
	if isSystem {
		tenant, err = h.tenantService.GetSystem(r.Context())
	} else {
		tenant, err = h.tenantService.GetByName(r.Context(), slug)
	}
	if err != nil {
		resp.HandleServiceError(w, r, "Tenant not found", err)
		return
	}

	res := TenantBootstrapResponseDTO{
		Tenant: TenantBootstrapTenantDTO{
			TenantUUID:  tenant.TenantUUID,
			Name:        tenant.Name,
			DisplayName: tenant.DisplayName,
			Description: tenant.Description,
			Status:      tenant.Status,
			IsSystem:    tenant.IsSystem,
		},
		Surface:     surface,
		IdentityURL: shared.FrontendURL(shared.FrontendSurfaceIdentity, tenant.Name, tenant.IsSystem, ""),
		ConsoleURL:  shared.FrontendURL(shared.FrontendSurfaceConsole, tenant.Name, tenant.IsSystem, ""),
		Branding:    h.publicBranding(r.Context(), tenant.TenantID, r.URL.Query().Get("client_id")),
		Connections: []TenantBootstrapConnectionDTO{},
	}
	res.PasswordConfig, res.RegistrationConfig = h.publicSecurityConfigs(r.Context(), tenant.TenantID)

	// Advertise the tenant's seeded system client for the resolved surface.
	// Best-effort: a missing client (e.g. not yet seeded) leaves the field unset
	// rather than failing the whole bootstrap.
	if h.clientReader != nil {
		if c, cerr := h.clientReader.GetSurfaceClient(r.Context(), tenant.Name, surface); cerr == nil && c != nil {
			res.Client = &TenantBootstrapClientDTO{
				ClientID:    c.ClientID,
				Name:        c.Name,
				DisplayName: c.DisplayName,
				ClientType:  c.ClientType,
			}
			methods := h.surfaceLoginMethods(r.Context(), c.ClientID)
			res.Connections = methods.Connections
			res.MagicLinkEnabled = methods.MagicLinkEnabled
		}
	}

	resp.Success(w, res, "Tenant bootstrap fetched successfully")
}

// surfaceLoginMethods loads the login options enabled on the resolved surface
// client. Best-effort by design: an unwired reader or a lookup error yields an
// empty (never nil) connection slice and magic link off, so the login page
// degrades to password-only instead of the whole bootstrap failing.
func (h *TenantHandler) surfaceLoginMethods(ctx context.Context, clientIdentifier string) bootstrapLoginMethods {
	out := bootstrapLoginMethods{Connections: []TenantBootstrapConnectionDTO{}}
	if h.connectionsReader == nil || strings.TrimSpace(clientIdentifier) == "" {
		return out
	}
	methods, err := h.connectionsReader.ListSurfaceConnections(ctx, clientIdentifier)
	if err != nil {
		return out
	}
	out.MagicLinkEnabled = methods.MagicLinkEnabled
	for _, c := range methods.Connections {
		out.Connections = append(out.Connections, TenantBootstrapConnectionDTO(c))
	}
	return out
}

// bootstrapLoginMethods is the handler-local projection assembled for the
// bootstrap response.
type bootstrapLoginMethods struct {
	Connections      []TenantBootstrapConnectionDTO
	MagicLinkEnabled bool
}

// publicBranding loads the public branding for bootstrap/public tenant
// responses. A non-empty clientID may select a client-attached theme; any miss
// or resolver error falls back to the tenant's active branding.
func (h *TenantHandler) publicBranding(ctx context.Context, tenantID int64, clientID string) *BrandingPublic {
	clientID = strings.TrimSpace(clientID)
	if clientID != "" && h.clientBrandingReader != nil {
		if b, err := h.clientBrandingReader.GetPublicClientBranding(ctx, tenantID, clientID); err == nil && b != nil {
			return toBrandingPublic(b)
		}
	}

	if h.brandingService == nil {
		return nil
	}
	b, err := h.brandingService.GetPublic(ctx, tenantID)
	if err != nil || b == nil {
		return nil
	}
	return toBrandingPublic(b)
}

func toBrandingPublic(b *branding.BrandingServiceDataResult) *BrandingPublic {
	if b == nil {
		return nil
	}
	return &BrandingPublic{
		Layout:                b.Layout,
		CompanyName:           b.CompanyName,
		LogoLabel:             b.LogoLabel,
		LogoDetail:            b.LogoDetail,
		ShowLogoLabel:         b.ShowLogoLabel,
		IdentityLogoLabel:     b.IdentityLogoLabel,
		IdentityShowLogoLabel: b.IdentityShowLogoLabel,
		LogoURL:               b.LogoURL,
		FaviconURL:            b.FaviconURL,
		SupportURL:            b.SupportURL,
		PrivacyPolicyURL:      b.PrivacyPolicyURL,
		TermsOfServiceURL:     b.TermsOfServiceURL,
		Metadata:              b.Metadata,
	}
}

func (h *TenantHandler) GetByName(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	if name == "" {
		resp.Error(w, http.StatusBadRequest, "Name is required")
		return
	}

	tenant, err := h.tenantService.GetByName(r.Context(), name)
	if err != nil {
		resp.HandleServiceError(w, r, "Tenant not found", err)
		return
	}

	dtoRes := h.toPublicResponse(r.Context(), *tenant)

	resp.Success(w, dtoRes, "Tenant fetched successfully")
}

func (h *TenantHandler) toPublicResponse(ctx context.Context, tenant TenantServiceDataResult) TenantPublicResponseDTO {
	res := TenantPublicResponseDTO{
		Name:        tenant.Name,
		DisplayName: tenant.DisplayName,
		Description: tenant.Description,
		Status:      tenant.Status,
		IsSystem:    tenant.IsSystem,
	}

	res.PasswordConfig, res.RegistrationConfig = h.publicSecurityConfigs(ctx, tenant.TenantID)
	res.Branding = h.publicBranding(ctx, tenant.TenantID, "")

	return res
}

// publicSecurityConfigs loads a tenant's public password and registration
// policy, returning nil for either when unavailable. Shared by the bootstrap
// and public tenant responses so both surfaces enforce the same tenant-managed
// config (password policy, self-registration gating).
func (h *TenantHandler) publicSecurityConfigs(ctx context.Context, tenantID int64) (*PasswordConfigPublic, *RegistrationConfigPublic) {
	if h.securitySettingService == nil {
		return nil, nil
	}
	var pwdCfg *PasswordConfigPublic
	var regCfg *RegistrationConfigPublic
	if pwd, err := h.securitySettingService.GetPasswordConfig(ctx, tenantID); err == nil {
		pwdCfg = &PasswordConfigPublic{
			MinLength:             intFromMap(pwd, "min_length"),
			MaxLength:             intFromMap(pwd, "max_length"),
			RequireUppercase:      boolFromMap(pwd, "require_uppercase"),
			RequireLowercase:      boolFromMap(pwd, "require_lowercase"),
			RequireNumber:         boolFromMap(pwd, "require_number"),
			RequireSymbol:         boolFromMap(pwd, "require_symbol"),
			MinStrengthScore:      intFromMap(pwd, "min_strength_score"),
			RejectCommonPasswords: boolFromMap(pwd, "reject_common_passwords"),
			CheckHibp:             boolFromMap(pwd, "check_hibp"),
		}
	}
	if reg, err := h.securitySettingService.GetRegistrationConfig(ctx, tenantID); err == nil {
		regCfg = &RegistrationConfigPublic{
			SelfRegistrationEnabled:  boolFromMap(reg, "self_registration_enabled"),
			RequireEmailVerification: boolFromMap(reg, "require_email_verification"),
			CaptchaOnSignup:          boolFromMap(reg, "captcha_on_signup"),
		}
	}
	return pwdCfg, regCfg
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

	tenantID := int64(0)
	if auth.Tenant != nil {
		tenantID = auth.Tenant.TenantID
	}
	var actorUserID *int64
	if auth.User != nil {
		actorUserID = &auth.User.UserID
	}
	changesJSON, _ := json.Marshal(map[string]any{"after": dtoRes})
	h.logAudit(r, tenantID, actorUserID, "tenant.create", "tenant", tenant.TenantUUID.String(), &tenant.TenantUUID, string(changesJSON), "success")

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

	var actorUserID *int64
	if user != nil {
		actorUserID = &user.UserID
	}
	changesJSON, _ := json.Marshal(map[string]any{"update": req, "after": dtoRes})
	h.logAudit(r, tenant.TenantID, actorUserID, "tenant.update", "tenant", tenant.TenantUUID.String(), &tenant.TenantUUID, string(changesJSON), "success")

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

	var actorUserID *int64
	if user != nil {
		actorUserID = &user.UserID
	}
	changesJSON, _ := json.Marshal(map[string]any{"update": req, "after": dtoRes})
	h.logAudit(r, tenant.TenantID, actorUserID, "tenant.set_status", "tenant", tenant.TenantUUID.String(), &tenant.TenantUUID, string(changesJSON), "success")

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

	actorTenantID := int64(0)
	if auth.Tenant != nil {
		actorTenantID = auth.Tenant.TenantID
	}
	var actorUserID *int64
	if auth.User != nil {
		actorUserID = &auth.User.UserID
	}
	changesJSON, _ := json.Marshal(map[string]any{"before": map[string]any{"id": deletedTenant.TenantUUID.String()}})
	h.logAudit(r, actorTenantID, actorUserID, "tenant.delete", "tenant", deletedTenant.TenantUUID.String(), &deletedTenant.TenantUUID, string(changesJSON), "success")

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
		Status:      r.Status,
		IsSystem:    r.IsSystem,
		Metadata:    r.Metadata,
		CreatedAt:   r.CreatedAt,
		UpdatedAt:   r.UpdatedAt,
	}

	return result
}
