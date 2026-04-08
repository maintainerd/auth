package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/dto"
	"github.com/maintainerd/auth/internal/middleware"
	"github.com/maintainerd/auth/internal/model"
	"github.com/maintainerd/auth/internal/service"
	"github.com/maintainerd/auth/internal/ptr"
	resp "github.com/maintainerd/auth/internal/rest/response"
)

// IPRestrictionRuleHandler handles HTTP requests for IP restriction rule management.
// All endpoints are tenant-scoped - the middleware validates user access to the tenant
// and sets it in the request context. The service layer ensures rules belong to the tenant.
type IPRestrictionRuleHandler struct {
	ipRestrictionRuleService service.IPRestrictionRuleService
}

// NewIPRestrictionRuleHandler creates a new instance of IPRestrictionRuleHandler.
func NewIPRestrictionRuleHandler(ipRestrictionRuleService service.IPRestrictionRuleService) *IPRestrictionRuleHandler {
	return &IPRestrictionRuleHandler{
		ipRestrictionRuleService: ipRestrictionRuleService,
	}
}

// GetAll retrieves all IP restriction rules for the tenant with optional filtering and pagination.
// Tenant access is validated by middleware; this handler only needs to extract tenant from context.
// The service layer filters rules by tenant_id to ensure data isolation.
func (h *IPRestrictionRuleHandler) GetAll(w http.ResponseWriter, r *http.Request) {
	// Tenant is already validated by middleware - just extract from context
	tenant, ok := r.Context().Value(middleware.TenantContextKey).(*model.Tenant)
	if !ok || tenant == nil {
		resp.Error(w, http.StatusUnauthorized, "Tenant not found in context")
		return
	}

	// Extract query parameters
	q := r.URL.Query()

	// Parse pagination parameters
	page, _ := strconv.Atoi(q.Get("page"))
	limit, _ := strconv.Atoi(q.Get("limit"))

	// Parse status filter (can be multiple)
	var status []string
	if v := q.Get("status"); v != "" {
		status = append(status, v)
	}

	// Build filter DTO with all query parameters
	filter := dto.IPRestrictionRuleFilterDTO{
		Type:        ptr.PtrOrNil(q.Get("type")),
		Status:      status,
		IPAddress:   ptr.PtrOrNil(q.Get("ip_address")),
		Description: ptr.PtrOrNil(q.Get("description")),
		PaginationRequestDTO: dto.PaginationRequestDTO{
			Page:      page,
			Limit:     limit,
			SortBy:    q.Get("sort_by"),
			SortOrder: q.Get("sort_order"),
		},
	}

	// Validate filter parameters
	if err := filter.Validate(); err != nil {
		resp.ValidationError(w, err)
		return
	}

	// Fetch rules from service - service filters by tenant_id
	result, err := h.ipRestrictionRuleService.GetAll(tenant.TenantID, filter.Type, filter.Status, filter.IPAddress, filter.Description, filter.Page, filter.Limit, filter.SortBy, filter.SortOrder)
	if err != nil {
		resp.HandleServiceError(w, r, "Failed to get IP restriction rules", err)
		return
	}

	// Build paginated response
	response := dto.PaginatedResponseDTO[dto.IPRestrictionRuleResponseDTO]{
		Rows:       toIPRestrictionRuleResponseDtoList(result.Data),
		Total:      result.Total,
		Page:       result.Page,
		Limit:      result.Limit,
		TotalPages: result.TotalPages,
	}

	resp.Success(w, response, "IP restriction rules retrieved successfully")
}

// Get retrieves a specific IP restriction rule by UUID.
// Tenant access is validated by middleware.
// The service layer verifies the rule belongs to the tenant.
func (h *IPRestrictionRuleHandler) Get(w http.ResponseWriter, r *http.Request) {
	// Tenant is already validated by middleware - just extract from context
	tenant, ok := r.Context().Value(middleware.TenantContextKey).(*model.Tenant)
	if !ok || tenant == nil {
		resp.Error(w, http.StatusUnauthorized, "Tenant not found in context")
		return
	}

	// Extract rule UUID from URL parameter
	ipRestrictionRuleUUIDStr := chi.URLParam(r, "ip_restriction_rule_uuid")

	// Parse and validate UUID format
	ipRestrictionRuleUUID, err := uuid.Parse(ipRestrictionRuleUUIDStr)
	if err != nil {
		resp.Error(w, http.StatusBadRequest, "Invalid IP restriction rule UUID")
		return
	}

	// Fetch rule - service validates it belongs to tenant
	rule, err := h.ipRestrictionRuleService.GetByUUID(tenant.TenantID, ipRestrictionRuleUUID)
	if err != nil {
		resp.HandleServiceError(w, r, "IP restriction rule not found", err)
		return
	}

	resp.Success(w, toIPRestrictionRuleResponseDTO(*rule), "IP restriction rule retrieved successfully")
}

// Create creates a new IP restriction rule for the tenant.
// Tenant access is validated by middleware.
// The rule is automatically associated with the tenant from context.
func (h *IPRestrictionRuleHandler) Create(w http.ResponseWriter, r *http.Request) {
	// Tenant is already validated by middleware - just extract from context
	tenant, ok := r.Context().Value(middleware.TenantContextKey).(*model.Tenant)
	if !ok || tenant == nil {
		resp.Error(w, http.StatusUnauthorized, "Tenant not found in context")
		return
	}

	// Extract authenticated user from context (needed for audit tracking)
	user, ok := r.Context().Value(middleware.UserContextKey).(*model.User)
	if !ok || user == nil {
		resp.Error(w, http.StatusUnauthorized, "User not found in context")
		return
	}

	// Decode request body
	var req dto.IPRestrictionRuleCreateRequestDTO
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		resp.Error(w, http.StatusBadRequest, "Invalid request")
		return
	}

	// Validate request data
	if err := req.Validate(); err != nil {
		resp.ValidationError(w, err)
		return
	}

	// Set default status if not provided
	status := model.StatusActive
	if req.Status != nil {
		status = *req.Status
	}

	// Create rule associated with tenant
	rule, err := h.ipRestrictionRuleService.Create(
		tenant.TenantID,
		req.Description,
		req.Type,
		req.IPAddress,
		status,
		user.UserID,
	)
	if err != nil {
		resp.HandleServiceError(w, r, "Failed to create IP restriction rule", err)
		return
	}

	resp.Created(w, toIPRestrictionRuleResponseDTO(*rule), "IP restriction rule created successfully")
}

// Update updates an existing IP restriction rule.
// Tenant access is validated by middleware.
// The service layer verifies the rule belongs to the tenant before updating.
func (h *IPRestrictionRuleHandler) Update(w http.ResponseWriter, r *http.Request) {
	// Tenant is already validated by middleware - just extract from context
	tenant, ok := r.Context().Value(middleware.TenantContextKey).(*model.Tenant)
	if !ok || tenant == nil {
		resp.Error(w, http.StatusUnauthorized, "Tenant not found in context")
		return
	}

	// Extract authenticated user from context (needed for audit tracking)
	user, ok := r.Context().Value(middleware.UserContextKey).(*model.User)
	if !ok || user == nil {
		resp.Error(w, http.StatusUnauthorized, "User not found in context")
		return
	}

	// Extract rule UUID from URL parameter
	ipRestrictionRuleUUIDStr := chi.URLParam(r, "ip_restriction_rule_uuid")

	// Parse and validate UUID format
	ipRestrictionRuleUUID, err := uuid.Parse(ipRestrictionRuleUUIDStr)
	if err != nil {
		resp.Error(w, http.StatusBadRequest, "Invalid IP restriction rule UUID")
		return
	}

	// Decode request body
	var req dto.IPRestrictionRuleUpdateRequestDTO
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		resp.Error(w, http.StatusBadRequest, "Invalid request")
		return
	}

	// Validate request data
	if err := req.Validate(); err != nil {
		resp.ValidationError(w, err)
		return
	}

	// Set default status if not provided
	status := model.StatusActive
	if req.Status != nil {
		status = *req.Status
	}

	// Update rule - service validates it belongs to tenant
	rule, err := h.ipRestrictionRuleService.Update(
		tenant.TenantID,
		ipRestrictionRuleUUID,
		req.Description,
		req.Type,
		req.IPAddress,
		status,
		user.UserID,
	)
	if err != nil {
		resp.HandleServiceError(w, r, "Failed to update IP restriction rule", err)
		return
	}

	resp.Success(w, toIPRestrictionRuleResponseDTO(*rule), "IP restriction rule updated successfully")
}

// Delete soft-deletes an IP restriction rule.
// Tenant access is validated by middleware.
// The service layer verifies the rule belongs to the tenant before deletion.
func (h *IPRestrictionRuleHandler) Delete(w http.ResponseWriter, r *http.Request) {
	// Tenant is already validated by middleware - just extract from context
	tenant, ok := r.Context().Value(middleware.TenantContextKey).(*model.Tenant)
	if !ok || tenant == nil {
		resp.Error(w, http.StatusUnauthorized, "Tenant not found in context")
		return
	}

	// Extract rule UUID from URL parameter
	ipRestrictionRuleUUIDStr := chi.URLParam(r, "ip_restriction_rule_uuid")

	// Parse and validate UUID format
	ipRestrictionRuleUUID, err := uuid.Parse(ipRestrictionRuleUUIDStr)
	if err != nil {
		resp.Error(w, http.StatusBadRequest, "Invalid IP restriction rule UUID")
		return
	}

	// Delete rule - service validates it belongs to tenant
	rule, err := h.ipRestrictionRuleService.Delete(tenant.TenantID, ipRestrictionRuleUUID)
	if err != nil {
		resp.HandleServiceError(w, r, "Failed to delete IP restriction rule", err)
		return
	}

	resp.Success(w, toIPRestrictionRuleResponseDTO(*rule), "IP restriction rule deleted successfully")
}

// UpdateStatus updates the status of an IP restriction rule (active/inactive).
// Tenant access is validated by middleware.
// The service layer verifies the rule belongs to the tenant before updating status.
func (h *IPRestrictionRuleHandler) UpdateStatus(w http.ResponseWriter, r *http.Request) {
	// Tenant is already validated by middleware - just extract from context
	tenant, ok := r.Context().Value(middleware.TenantContextKey).(*model.Tenant)
	if !ok || tenant == nil {
		resp.Error(w, http.StatusUnauthorized, "Tenant not found in context")
		return
	}

	// Extract authenticated user from context (needed for audit tracking)
	user, ok := r.Context().Value(middleware.UserContextKey).(*model.User)
	if !ok || user == nil {
		resp.Error(w, http.StatusUnauthorized, "User not found in context")
		return
	}

	// Extract rule UUID from URL parameter
	ipRestrictionRuleUUIDStr := chi.URLParam(r, "ip_restriction_rule_uuid")

	// Parse and validate UUID format
	ipRestrictionRuleUUID, err := uuid.Parse(ipRestrictionRuleUUIDStr)
	if err != nil {
		resp.Error(w, http.StatusBadRequest, "Invalid IP restriction rule UUID")
		return
	}

	// Decode request body
	var req dto.IPRestrictionRuleUpdateStatusRequestDTO
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		resp.Error(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Validate request data
	if err := req.Validate(); err != nil {
		resp.ValidationError(w, err)
		return
	}

	// Update status - service validates rule belongs to tenant
	rule, err := h.ipRestrictionRuleService.UpdateStatus(tenant.TenantID, ipRestrictionRuleUUID, req.Status, user.UserID)
	if err != nil {
		resp.HandleServiceError(w, r, "Failed to update IP restriction rule status", err)
		return
	}

	resp.Success(w, toIPRestrictionRuleResponseDTO(*rule), "IP restriction rule status updated successfully")
}

// Helper functions for converting service data to response DTOs

// toIPRestrictionRuleResponseDTO converts a service result to a response DTO.
func toIPRestrictionRuleResponseDTO(rule service.IPRestrictionRuleServiceDataResult) dto.IPRestrictionRuleResponseDTO {
	return dto.IPRestrictionRuleResponseDTO{
		IPRestrictionRuleID: rule.IPRestrictionRuleUUID.String(),
		Description:         rule.Description,
		Type:                rule.Type,
		IPAddress:           rule.IPAddress,
		Status:              rule.Status,
		CreatedAt:           rule.CreatedAt,
		UpdatedAt:           rule.UpdatedAt,
	}
}

// toIPRestrictionRuleResponseDtoList converts a slice of service results to response DTOs.
func toIPRestrictionRuleResponseDtoList(rules []service.IPRestrictionRuleServiceDataResult) []dto.IPRestrictionRuleResponseDTO {
	result := make([]dto.IPRestrictionRuleResponseDTO, len(rules))
	for i, rule := range rules {
		result[i] = toIPRestrictionRuleResponseDTO(rule)
	}
	return result
}
