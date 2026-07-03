package iam

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/maintainerd/maintainerd-auth/internal/platform/middleware"
	"github.com/maintainerd/maintainerd-auth/internal/platform/pagination"
	resp "github.com/maintainerd/maintainerd-auth/internal/platform/response"
)

type PolicyHandler struct {
	policyService PolicyService
}

func NewPolicyHandler(policyService PolicyService) *PolicyHandler {
	return &PolicyHandler{
		policyService: policyService,
	}
}

// Get policies with filtering and pagination
func (h *PolicyHandler) Get(w http.ResponseWriter, r *http.Request) {
	// Get tenant from context
	tenant := middleware.AuthFromRequest(r).Tenant
	if tenant == nil {
		resp.Error(w, http.StatusUnauthorized, "Tenant not found in context")
		return
	}

	// Parse query parameters
	query := r.URL.Query()

	// Parse filters
	var filter PolicyFilterDTO
	if name := query.Get("name"); name != "" {
		filter.Name = &name
	}
	if description := query.Get("description"); description != "" {
		filter.Description = &description
	}
	if version := query.Get("version"); version != "" {
		filter.Version = &version
	}
	// Handle status filtering - support both comma-separated and multiple parameters
	var statusValues []string
	if singleStatus := query.Get("status"); singleStatus != "" {
		// Handle comma-separated values like services: ?status=active,inactive
		statusValues = strings.Split(strings.ReplaceAll(singleStatus, " ", ""), ",")
	} else if multipleStatus := query["status"]; len(multipleStatus) > 0 {
		// Handle multiple parameters: ?status=active&status=inactive
		statusValues = multipleStatus
	}

	// Filter out empty status values
	var validStatus []string
	for _, s := range statusValues {
		if s != "" {
			validStatus = append(validStatus, s)
		}
	}
	if len(validStatus) > 0 {
		filter.Status = validStatus
	}
	if isSystem := query.Get("is_system"); isSystem != "" {
		if val, err := strconv.ParseBool(isSystem); err == nil {
			filter.IsSystem = &val
		}
	}
	if serviceID := query.Get("service_id"); serviceID != "" {
		// Only parse if it's a valid UUID format
		if val, err := uuid.Parse(serviceID); err == nil {
			filter.ServiceID = &val
		}
		// If parsing fails, we ignore the service_id filter (don't set filter.ServiceID)
	}

	// Parse pagination
	pag := pagination.ParseQuery(r)
	filter.Page = pag.Page
	filter.Limit = pag.Limit
	filter.SortBy = pag.SortBy
	filter.SortOrder = pag.SortOrder

	// Validate filter parameters
	if err := filter.Validate(); err != nil {
		resp.ValidationError(w, err)
		return
	}

	// Convert to service filter
	serviceFilter := PolicyServiceGetFilter{
		TenantID:    tenant.TenantID,
		Name:        filter.Name,
		Description: filter.Description,
		Version:     filter.Version,
		Status:      filter.Status,
		IsSystem:    filter.IsSystem,
		ServiceID:   filter.ServiceID,
		Page:        filter.Page,
		Limit:       filter.Limit,
		SortBy:      filter.SortBy,
		SortOrder:   filter.SortOrder,
	}

	result, err := h.policyService.Get(r.Context(), serviceFilter)
	if err != nil {
		resp.HandleServiceError(w, r, "Failed to get policies", err)
		return
	}

	// Convert to response DTOs
	rows := make([]PolicyResponseDTO, len(result.Data))
	for i, policy := range result.Data {
		rows[i] = toPolicyResponseDTO(policy)
	}

	response := PaginatedResponseDTO[PolicyResponseDTO]{
		Rows:       rows,
		Total:      result.Total,
		Page:       result.Page,
		Limit:      result.Limit,
		TotalPages: result.TotalPages,
	}

	resp.Success(w, response, "Policies retrieved successfully")
}

// Get policy by UUID
func (h *PolicyHandler) GetByUUID(w http.ResponseWriter, r *http.Request) {
	// Get tenant from context
	tenant := middleware.AuthFromRequest(r).Tenant
	if tenant == nil {
		resp.Error(w, http.StatusUnauthorized, "Tenant not found in context")
		return
	}

	policyUUIDStr := chi.URLParam(r, "policy_uuid")
	policyUUID, err := uuid.Parse(policyUUIDStr)
	if err != nil {
		resp.Error(w, http.StatusBadRequest, "Invalid policy UUID")
		return
	}

	policy, err := h.policyService.GetByUUID(r.Context(), policyUUID, tenant.TenantID)
	if err != nil {
		resp.HandleServiceError(w, r, "Policy not found", err)
		return
	}

	dtoRes := toPolicyDetailResponseDTO(*policy)
	resp.Success(w, dtoRes, "Policy retrieved successfully")
}

// Create policy
func (h *PolicyHandler) Create(w http.ResponseWriter, r *http.Request) {
	// Get tenant from context
	tenant := middleware.AuthFromRequest(r).Tenant
	if tenant == nil {
		resp.Error(w, http.StatusUnauthorized, "Tenant not found in context")
		return
	}

	var req PolicyCreateRequestDTO
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		resp.BadRequestBody(w)
		return
	}

	if err := req.Validate(); err != nil {
		resp.ValidationError(w, err)
		return
	}

	policy, err := h.policyService.Create(
		r.Context(),
		tenant.TenantID,
		req.Name,
		req.Description,
		req.Document,
		req.Version,
		req.Status,
		false, // isSystem - only set by seeders
	)
	if err != nil {
		resp.HandleServiceError(w, r, "Failed to create policy", err)
		return
	}

	dtoRes := toPolicyDetailResponseDTO(*policy)
	resp.Created(w, dtoRes, "Policy created successfully")
}

// Update policy
func (h *PolicyHandler) Update(w http.ResponseWriter, r *http.Request) {
	// Get tenant from context
	tenant := middleware.AuthFromRequest(r).Tenant
	if tenant == nil {
		resp.Error(w, http.StatusUnauthorized, "Tenant not found in context")
		return
	}

	policyUUIDStr := chi.URLParam(r, "policy_uuid")
	policyUUID, err := uuid.Parse(policyUUIDStr)
	if err != nil {
		resp.Error(w, http.StatusBadRequest, "Invalid policy UUID")
		return
	}

	var req PolicyUpdateRequestDTO
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		resp.BadRequestBody(w)
		return
	}

	if err := req.Validate(); err != nil {
		resp.ValidationError(w, err)
		return
	}

	policy, err := h.policyService.Update(
		r.Context(),
		policyUUID,
		tenant.TenantID,
		req.Name,
		req.Description,
		req.Document,
		req.Version,
		req.Status,
	)
	if err != nil {
		resp.HandleServiceError(w, r, "Failed to update policy", err)
		return
	}

	dtoRes := toPolicyDetailResponseDTO(*policy)
	resp.Success(w, dtoRes, "Policy updated successfully")
}

// Update policy status
func (h *PolicyHandler) UpdateStatus(w http.ResponseWriter, r *http.Request) {
	// Get tenant from context
	tenant := middleware.AuthFromRequest(r).Tenant
	if tenant == nil {
		resp.Error(w, http.StatusUnauthorized, "Tenant not found in context")
		return
	}

	policyUUIDStr := chi.URLParam(r, "policy_uuid")
	policyUUID, err := uuid.Parse(policyUUIDStr)
	if err != nil {
		resp.Error(w, http.StatusBadRequest, "Invalid policy UUID")
		return
	}

	var req PolicyStatusUpdateDTO
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		resp.BadRequestBody(w)
		return
	}

	if err := req.Validate(); err != nil {
		resp.ValidationError(w, err)
		return
	}

	policy, err := h.policyService.SetStatusByUUID(r.Context(), policyUUID, tenant.TenantID, req.Status)
	if err != nil {
		resp.HandleServiceError(w, r, "Failed to update policy status", err)
		return
	}

	dtoRes := toPolicyDetailResponseDTO(*policy)
	resp.Success(w, dtoRes, "Policy status updated successfully")
}

// Delete policy
func (h *PolicyHandler) Delete(w http.ResponseWriter, r *http.Request) {
	// Get tenant from context
	tenant := middleware.AuthFromRequest(r).Tenant
	if tenant == nil {
		resp.Error(w, http.StatusUnauthorized, "Tenant not found in context")
		return
	}

	policyUUIDStr := chi.URLParam(r, "policy_uuid")
	policyUUID, err := uuid.Parse(policyUUIDStr)
	if err != nil {
		resp.Error(w, http.StatusBadRequest, "Invalid policy UUID")
		return
	}

	policy, err := h.policyService.DeleteByUUID(r.Context(), policyUUID, tenant.TenantID)
	if err != nil {
		resp.HandleServiceError(w, r, "Failed to delete policy", err)
		return
	}

	dtoRes := toPolicyDetailResponseDTO(*policy)
	resp.Success(w, dtoRes, "Policy deleted successfully")
}

// Get services that use a specific policy
func (h *PolicyHandler) GetServicesByPolicyUUID(w http.ResponseWriter, r *http.Request) {
	// Get tenant from context
	tenant := middleware.AuthFromRequest(r).Tenant
	if tenant == nil {
		resp.Error(w, http.StatusUnauthorized, "Tenant not found in context")
		return
	}

	policyUUIDStr := chi.URLParam(r, "policy_uuid")
	policyUUID, err := uuid.Parse(policyUUIDStr)
	if err != nil {
		resp.Error(w, http.StatusBadRequest, "Invalid policy UUID")
		return
	}

	q := r.URL.Query()

	// Build filter
	filter := PolicyServicesFilterDTO{
		PaginationRequestDTO: pagination.ParseQuery(r),
	}

	// Parse string filters
	if name := q.Get("name"); name != "" {
		filter.Name = &name
	}
	if displayName := q.Get("display_name"); displayName != "" {
		filter.DisplayName = &displayName
	}
	if description := q.Get("description"); description != "" {
		filter.Description = &description
	}

	// Validate filter parameters
	if err := filter.Validate(); err != nil {
		resp.ValidationError(w, err)
		return
	}

	// Convert to service filter
	serviceFilter := PolicyServiceServicesFilter{
		Name:        filter.Name,
		DisplayName: filter.DisplayName,
		Description: filter.Description,
		Page:        filter.Page,
		Limit:       filter.Limit,
		SortBy:      filter.SortBy,
		SortOrder:   filter.SortOrder,
	}

	// Get services
	result, err := h.policyService.GetServicesByPolicyUUID(r.Context(), policyUUID, tenant.TenantID, serviceFilter)
	if err != nil {
		resp.HandleServiceError(w, r, "Failed to get services", err)
		return
	}

	// Convert to response DTOs. Keep rows as [] instead of null when empty.
	services := make([]ServiceResponseDTO, 0, len(result.Data))
	for _, svc := range result.Data {
		services = append(services, ServiceResponseDTO(svc))
	}

	// Build paginated response
	response := PaginatedResponseDTO[ServiceResponseDTO]{
		Rows:       services,
		Total:      result.Total,
		Page:       result.Page,
		Limit:      result.Limit,
		TotalPages: result.TotalPages,
	}

	resp.Success(w, response, "Services retrieved successfully")
}

// Helper function to convert service result to DTO (for listing - without document)
func toPolicyResponseDTO(policy PolicyServiceDataResult) PolicyResponseDTO {
	return PolicyResponseDTO{
		PolicyUUID:  policy.PolicyUUID,
		Name:        policy.Name,
		Description: policy.Description,
		Version:     policy.Version,
		Status:      policy.Status,
		IsSystem:    policy.IsSystem,
		CreatedAt:   policy.CreatedAt,
		UpdatedAt:   policy.UpdatedAt,
	}
}

// Helper function to convert service result to detail DTO (for individual retrieval - with document)
func toPolicyDetailResponseDTO(policy PolicyServiceDataResult) PolicyDetailResponseDTO {
	return PolicyDetailResponseDTO(policy)
}
