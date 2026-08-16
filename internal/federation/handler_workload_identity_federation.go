package federation

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/maintainerd/maintainerd-auth/internal/auditlog"
	"github.com/maintainerd/maintainerd-auth/internal/authctx"
	"github.com/maintainerd/maintainerd-auth/internal/platform/middleware"
	"github.com/maintainerd/maintainerd-auth/internal/platform/pagination"
	resp "github.com/maintainerd/maintainerd-auth/internal/platform/response"
)

// WorkloadIdentityFederationHandler handles HTTP requests for workload identity
// federation management on the internal port (8080).
type WorkloadIdentityFederationHandler struct {
	service     WorkloadIdentityFederationService
	auditLogger auditlog.ManagementAuditLogger
}

// NewWorkloadIdentityFederationHandler creates a new handler.
func NewWorkloadIdentityFederationHandler(service WorkloadIdentityFederationService) *WorkloadIdentityFederationHandler {
	return &WorkloadIdentityFederationHandler{service: service}
}

// SetAuditLogger installs the management audit logger. Call once at startup.
//
// This is the "let an external workload in with no credential" table, so GRANTING a
// federation is at least as audit-worthy as using one. Per-exchange records already
// existed; the CRUD that creates the trust left no trace at all.
func (h *WorkloadIdentityFederationHandler) SetAuditLogger(l auditlog.ManagementAuditLogger) {
	h.auditLogger = l
}

func (h *WorkloadIdentityFederationHandler) logAudit(
	r *http.Request,
	tenantID int64,
	actorUserID *int64,
	action, resourceID string,
	resourceUUID *uuid.UUID,
	changes, outcome string,
) {
	if h.auditLogger == nil {
		return
	}
	_ = h.auditLogger.Log(r.Context(), auditlog.LogEntry{
		TenantID:     tenantID,
		ActorUserID:  actorUserID,
		Action:       action,
		ResourceType: "workload_identity_federation",
		ResourceID:   resourceID,
		ResourceUUID: resourceUUID,
		Changes:      changes,
		Outcome:      outcome,
	})
}

// auditChanges renders the security-relevant fields of a federation for the audit
// trail: the trust boundary (issuer / audience / subject pattern) and what it grants
// (scopes, mapped claims) are exactly what a reviewer needs after the fact.
func auditChanges(d *WorkloadIdentityFederationServiceDataResult) string {
	payload, _ := json.Marshal(map[string]any{
		"name":              d.Name,
		"issuer_url":        d.IssuerURL,
		"audience":          d.Audience,
		"subject_claim":     d.SubjectClaim,
		"subject_pattern":   d.SubjectPattern,
		"allowed_scopes":    d.AllowedScopes,
		"attribute_mapping": d.AttributeMapping,
		"is_active":         d.IsActive,
		"client_id":         d.ClientUUID.String(),
	})
	return string(payload)
}

// GetAll lists workload identity federations for the tenant.
//
// GET /workload-identity-federations
func (h *WorkloadIdentityFederationHandler) GetAll(w http.ResponseWriter, r *http.Request) {
	tenant := middleware.AuthFromRequest(r).Tenant
	if tenant == nil {
		resp.Error(w, http.StatusUnauthorized, "Tenant not found in context")
		return
	}

	q := r.URL.Query()
	filter := WorkloadIdentityFederationFilterDTO{
		PaginationRequestDTO: pagination.ParseQuery(r),
	}
	if name := strings.TrimSpace(q.Get("name")); name != "" {
		filter.Name = &name
	}
	// Accepts the console's human values ("active"/"inactive") as well as booleans,
	// because the listing filter chips use words — matching the is_system
	// ("system"/"regular") convention elsewhere.
	if v := strings.ToLower(strings.TrimSpace(q.Get("is_active"))); v != "" {
		active, inactive := true, false
		switch v {
		case "active":
			filter.IsActive = &active
		case "inactive":
			filter.IsActive = &inactive
		default:
			if parsed, perr := strconv.ParseBool(v); perr == nil {
				filter.IsActive = &parsed
			}
		}
	}
	if err := filter.Validate(); err != nil {
		resp.ValidationError(w, err)
		return
	}

	result, err := h.service.GetAll(r.Context(), tenant.TenantID, WorkloadIdentityFederationListFilter{
		Name:      filter.Name,
		IsActive:  filter.IsActive,
		Page:      filter.Page,
		Limit:     filter.Limit,
		SortBy:    filter.SortBy,
		SortOrder: filter.SortOrder,
	})
	if err != nil {
		resp.HandleServiceError(w, r, "Failed to get workload identity federations", err)
		return
	}

	response := PaginatedResponseDTO[WorkloadIdentityFederationResponseDTO]{
		Rows:       toResponseDTOList(result.Data),
		Total:      result.Total,
		Page:       result.Page,
		Limit:      result.Limit,
		TotalPages: result.TotalPages,
	}
	resp.Success(w, response, "Workload identity federations retrieved successfully")
}

// Get retrieves a single workload identity federation by UUID.
//
// GET /workload-identity-federations/{workload_identity_federation_uuid}
func (h *WorkloadIdentityFederationHandler) Get(w http.ResponseWriter, r *http.Request) {
	tenant := middleware.AuthFromRequest(r).Tenant
	if tenant == nil {
		resp.Error(w, http.StatusUnauthorized, "Tenant not found in context")
		return
	}

	federationUUID, err := uuid.Parse(chi.URLParam(r, "workload_identity_federation_uuid"))
	if err != nil {
		resp.Error(w, http.StatusBadRequest, "Invalid workload identity federation UUID")
		return
	}

	result, err := h.service.GetByUUID(r.Context(), tenant.TenantID, federationUUID)
	if err != nil {
		resp.HandleServiceError(w, r, "Workload identity federation not found", err)
		return
	}

	resp.Success(w, toResponseDTO(*result), "Workload identity federation retrieved successfully")
}

// Create creates a new workload identity federation.
//
// POST /workload-identity-federations
func (h *WorkloadIdentityFederationHandler) Create(w http.ResponseWriter, r *http.Request) {
	auth := middleware.AuthFromRequest(r)
	if auth.Tenant == nil {
		resp.Error(w, http.StatusUnauthorized, "Tenant not found in context")
		return
	}
	// A keyless-auth trust rule must be attributable. Every comparable handler in
	// this codebase requires both tenant and user; this one checked only tenant and
	// then silently recorded created_by = NULL.
	if auth.User == nil {
		resp.Error(w, http.StatusUnauthorized, "User not found in context")
		return
	}

	var req WorkloadIdentityFederationCreateRequestDTO
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		resp.BadRequestBody(w)
		return
	}
	if err := req.Validate(); err != nil {
		resp.ValidationError(w, err)
		return
	}

	clientUUID, err := uuid.Parse(req.ClientUUID)
	if err != nil {
		resp.Error(w, http.StatusBadRequest, "Invalid client UUID")
		return
	}

	result, err := h.service.Create(r.Context(), auth.Tenant.TenantID, WorkloadIdentityFederationCreateInput{
		ClientUUID:       clientUUID,
		Name:             req.Name,
		Description:      req.Description,
		IssuerURL:        req.IssuerURL,
		Audience:         req.Audience,
		SubjectClaim:     req.SubjectClaim,
		SubjectPattern:   req.SubjectPattern,
		AllowedScopes:    req.AllowedScopes,
		AttributeMapping: req.AttributeMapping,
		IsActive:         req.IsActive,
		ActorUserID:      actorUserID(auth),
	})
	if err != nil {
		h.logAudit(r, auth.Tenant.TenantID, actorUserID(auth), "create", req.Name, nil, "", "failure")
		resp.HandleServiceError(w, r, "Failed to create workload identity federation", err)
		return
	}

	h.logAudit(r, auth.Tenant.TenantID, actorUserID(auth), "create",
		result.WorkloadIdentityFederationUUID.String(), &result.WorkloadIdentityFederationUUID,
		auditChanges(result), "success")

	resp.Created(w, toResponseDTO(*result), "Workload identity federation created successfully")
}

// Update updates an existing workload identity federation.
//
// PUT /workload-identity-federations/{workload_identity_federation_uuid}
func (h *WorkloadIdentityFederationHandler) Update(w http.ResponseWriter, r *http.Request) {
	auth := middleware.AuthFromRequest(r)
	if auth.Tenant == nil {
		resp.Error(w, http.StatusUnauthorized, "Tenant not found in context")
		return
	}

	// Changing a keyless-auth trust rule must be attributable.
	if auth.User == nil {
		resp.Error(w, http.StatusUnauthorized, "User not found in context")
		return
	}

	federationUUID, err := uuid.Parse(chi.URLParam(r, "workload_identity_federation_uuid"))
	if err != nil {
		resp.Error(w, http.StatusBadRequest, "Invalid workload identity federation UUID")
		return
	}

	var req WorkloadIdentityFederationUpdateRequestDTO
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		resp.BadRequestBody(w)
		return
	}
	if err := req.Validate(); err != nil {
		resp.ValidationError(w, err)
		return
	}

	result, err := h.service.Update(r.Context(), auth.Tenant.TenantID, federationUUID, WorkloadIdentityFederationUpdateInput{
		Name:             req.Name,
		Description:      req.Description,
		IssuerURL:        req.IssuerURL,
		Audience:         req.Audience,
		SubjectClaim:     req.SubjectClaim,
		SubjectPattern:   req.SubjectPattern,
		AllowedScopes:    req.AllowedScopes,
		AttributeMapping: req.AttributeMapping,
		IsActive:         req.IsActive,
		ActorUserID:      actorUserID(auth),
	})
	if err != nil {
		h.logAudit(r, auth.Tenant.TenantID, actorUserID(auth), "update",
			federationUUID.String(), &federationUUID, "", "failure")
		resp.HandleServiceError(w, r, "Failed to update workload identity federation", err)
		return
	}

	h.logAudit(r, auth.Tenant.TenantID, actorUserID(auth), "update",
		federationUUID.String(), &federationUUID, auditChanges(result), "success")

	resp.Success(w, toResponseDTO(*result), "Workload identity federation updated successfully")
}

// Delete soft-deletes a workload identity federation.
//
// DELETE /workload-identity-federations/{workload_identity_federation_uuid}
func (h *WorkloadIdentityFederationHandler) Delete(w http.ResponseWriter, r *http.Request) {
	auth := middleware.AuthFromRequest(r)
	if auth.Tenant == nil {
		resp.Error(w, http.StatusUnauthorized, "Tenant not found in context")
		return
	}
	// Deleting a federation instantly revokes a live workload's ability to
	// authenticate, so it must be attributable.
	if auth.User == nil {
		resp.Error(w, http.StatusUnauthorized, "User not found in context")
		return
	}
	tenant := auth.Tenant

	federationUUID, err := uuid.Parse(chi.URLParam(r, "workload_identity_federation_uuid"))
	if err != nil {
		resp.Error(w, http.StatusBadRequest, "Invalid workload identity federation UUID")
		return
	}

	result, err := h.service.Delete(r.Context(), tenant.TenantID, federationUUID)
	if err != nil {
		h.logAudit(r, tenant.TenantID, actorUserID(auth), "delete",
			federationUUID.String(), &federationUUID, "", "failure")
		resp.HandleServiceError(w, r, "Failed to delete workload identity federation", err)
		return
	}

	h.logAudit(r, tenant.TenantID, actorUserID(auth), "delete",
		federationUUID.String(), &federationUUID, auditChanges(result), "success")

	resp.Success(w, toResponseDTO(*result), "Workload identity federation deleted successfully")
}

// ---------------------------------------------------------------------------
// Response mapping
// ---------------------------------------------------------------------------

func toResponseDTO(d WorkloadIdentityFederationServiceDataResult) WorkloadIdentityFederationResponseDTO {
	mapping := d.AttributeMapping
	if mapping == nil {
		mapping = map[string]string{}
	}
	scopes := d.AllowedScopes
	if scopes == nil {
		scopes = []string{}
	}
	return WorkloadIdentityFederationResponseDTO{
		WorkloadIdentityFederationUUID: d.WorkloadIdentityFederationUUID,
		ClientUUID:                     d.ClientUUID.String(),
		Name:                           d.Name,
		Description:                    d.Description,
		IssuerURL:                      d.IssuerURL,
		Audience:                       d.Audience,
		SubjectClaim:                   d.SubjectClaim,
		SubjectPattern:                 d.SubjectPattern,
		AllowedScopes:                  scopes,
		AttributeMapping:               mapping,
		IsActive:                       d.IsActive,
		CreatedAt:                      d.CreatedAt,
		UpdatedAt:                      d.UpdatedAt,
	}
}

func toResponseDTOList(items []WorkloadIdentityFederationServiceDataResult) []WorkloadIdentityFederationResponseDTO {
	out := make([]WorkloadIdentityFederationResponseDTO, len(items))
	for i, it := range items {
		out[i] = toResponseDTO(it)
	}
	return out
}

func actorUserID(auth *authctx.AuthContext) *int64 {
	if auth == nil || auth.User == nil {
		return nil
	}
	id := auth.User.UserID
	return &id
}
