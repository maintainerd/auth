package federation

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/maintainerd/maintainerd-auth/internal/authctx"
	"github.com/maintainerd/maintainerd-auth/internal/platform/middleware"
	"github.com/maintainerd/maintainerd-auth/internal/platform/pagination"
	resp "github.com/maintainerd/maintainerd-auth/internal/platform/response"
)

// WorkloadIdentityFederationHandler handles HTTP requests for workload identity
// federation management on the internal port (8080).
type WorkloadIdentityFederationHandler struct {
	service WorkloadIdentityFederationService
}

// NewWorkloadIdentityFederationHandler creates a new handler.
func NewWorkloadIdentityFederationHandler(service WorkloadIdentityFederationService) *WorkloadIdentityFederationHandler {
	return &WorkloadIdentityFederationHandler{service: service}
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

	filter := WorkloadIdentityFederationFilterDTO{
		PaginationRequestDTO: pagination.ParseQuery(r),
	}
	if err := filter.Validate(); err != nil {
		resp.ValidationError(w, err)
		return
	}

	result, err := h.service.GetAll(r.Context(), tenant.TenantID, filter.Page, filter.Limit, filter.SortBy, filter.SortOrder)
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
		IsActive:         boolOrDefault(req.IsActive, true),
		ActorUserID:      actorUserID(auth),
	})
	if err != nil {
		resp.HandleServiceError(w, r, "Failed to create workload identity federation", err)
		return
	}

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
		IsActive:         boolOrDefault(req.IsActive, true),
		ActorUserID:      actorUserID(auth),
	})
	if err != nil {
		resp.HandleServiceError(w, r, "Failed to update workload identity federation", err)
		return
	}

	resp.Success(w, toResponseDTO(*result), "Workload identity federation updated successfully")
}

// Delete soft-deletes a workload identity federation.
//
// DELETE /workload-identity-federations/{workload_identity_federation_uuid}
func (h *WorkloadIdentityFederationHandler) Delete(w http.ResponseWriter, r *http.Request) {
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

	result, err := h.service.Delete(r.Context(), tenant.TenantID, federationUUID)
	if err != nil {
		resp.HandleServiceError(w, r, "Failed to delete workload identity federation", err)
		return
	}

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

func boolOrDefault(v *bool, def bool) bool {
	if v == nil {
		return def
	}
	return *v
}

func actorUserID(auth *authctx.AuthContext) *int64 {
	if auth == nil || auth.User == nil {
		return nil
	}
	id := auth.User.UserID
	return &id
}
