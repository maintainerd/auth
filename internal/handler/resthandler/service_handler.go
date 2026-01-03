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

// ServiceHandler handles service management operations.
//
// This handler manages services (external applications/systems) that integrate with
// the authentication system. Services can be public (available to all tenants) or
// tenant-specific. The handler supports CRUD operations, status management, and
// policy assignment for access control.
type ServiceHandler struct {
	service service.ServiceService
}

// NewServiceHandler creates a new service handler instance.
func NewServiceHandler(service service.ServiceService) *ServiceHandler {
	return &ServiceHandler{service}
}

// Get retrieves all services with pagination and filters.
//
// GET /services
//
// Returns a paginated list of services. Supports filtering by name, display name,
// description, version, status, and flags (is_public, is_default, is_system).
// Public services are available to all tenants, while non-public services are tenant-specific.
func (h *ServiceHandler) Get(w http.ResponseWriter, r *http.Request) {
	// Parse query parameters
	q := r.URL.Query()

	// Parse pagination parameters
	page, _ := strconv.Atoi(q.Get("page"))
	limit, _ := strconv.Atoi(q.Get("limit"))

	// Parse filter flags
	var isDefault, isSystem, isPublic *bool
	var status []string
	if v := q.Get("is_default"); v != "" {
		if parsed, err := strconv.ParseBool(v); err == nil {
			isDefault = &parsed
		}
	}
	if v := q.Get("is_system"); v != "" {
		if parsed, err := strconv.ParseBool(v); err == nil {
			isSystem = &parsed
		}
	}
	if v := q.Get("status"); v != "" {
		// Parse comma-separated status values
		status = strings.Split(strings.ReplaceAll(v, " ", ""), ",")
	}
	if v := q.Get("is_public"); v != "" {
		if parsed, err := strconv.ParseBool(v); err == nil {
			isPublic = &parsed
		}
	}

	// Build request DTO for validation
	reqParams := dto.ServiceFilterDto{
		Name:        util.PtrOrNil(q.Get("name")),
		DisplayName: util.PtrOrNil(q.Get("display_name")),
		Description: util.PtrOrNil(q.Get("description")),
		Version:     util.PtrOrNil(q.Get("version")),
		Status:      status,
		IsPublic:    isPublic,
		IsDefault:   isDefault,
		IsSystem:    isSystem,
		PaginationRequestDto: dto.PaginationRequestDto{
			Page:      page,
			Limit:     limit,
			SortBy:    q.Get("sort_by"),
			SortOrder: q.Get("sort_order"),
		},
	}

	// Validate request parameters
	if err := reqParams.Validate(); err != nil {
		util.ValidationError(w, err)
		return
	}

	// Build service filter for query
	filter := service.ServiceServiceGetFilter{
		Name:        reqParams.Name,
		DisplayName: reqParams.DisplayName,
		Description: reqParams.Description,
		Version:     reqParams.Version,
		Status:      reqParams.Status,
		IsPublic:    reqParams.IsPublic,
		IsDefault:   reqParams.IsDefault,
		IsSystem:    reqParams.IsSystem,
		Page:        reqParams.Page,
		Limit:       reqParams.Limit,
		SortBy:      reqParams.SortBy,
		SortOrder:   reqParams.SortOrder,
	}

	// Fetch services from service layer
	result, err := h.service.Get(filter)
	if err != nil {
		util.Error(w, http.StatusInternalServerError, "Failed to fetch services", err.Error())
		return
	}

	// Map service results to DTOs
	rows := make([]dto.ServiceResponseDto, len(result.Data))
	for i, s := range result.Data {
		rows[i] = toServiceResponseDto(s)
	}

	// Build paginated response
	response := dto.PaginatedResponseDto[dto.ServiceResponseDto]{
		Rows:       rows,
		Total:      result.Total,
		Page:       result.Page,
		Limit:      result.Limit,
		TotalPages: result.TotalPages,
	}

	util.Success(w, response, "Services fetched successfully")
}

// GetByUUID retrieves a specific service by its UUID.
//
// GET /services/{service_uuid}
//
// Returns detailed information about a single service. Services can be public
// (available to all tenants) or tenant-specific.
func (h *ServiceHandler) GetByUUID(w http.ResponseWriter, r *http.Request) {
	// Parse and validate service UUID from URL parameter
	serviceUUID, err := uuid.Parse(chi.URLParam(r, "service_uuid"))
	if err != nil {
		util.Error(w, http.StatusBadRequest, "Invalid service UUID")
		return
	}

	// Fetch service by UUID
	svc, err := h.service.GetByUUID(serviceUUID)
	if err != nil {
		util.Error(w, http.StatusNotFound, "Service not found")
		return
	}

	// Build response
	dtoRes := toServiceResponseDto(*svc)

	util.Success(w, dtoRes, "Service fetched successfully")
}

// Create creates a new service.
//
// POST /services
//
// Creates a new service (external application/system). Services can be marked as public
// to make them available to all tenants, or kept tenant-specific. System and default
// flags are reserved for seeded services only.
func (h *ServiceHandler) Create(w http.ResponseWriter, r *http.Request) {
	// Get tenant from context (middleware already validated access)
	tenant, ok := r.Context().Value(middleware.TenantContextKey).(*model.Tenant)
	if !ok || tenant == nil {
		util.Error(w, http.StatusUnauthorized, "Tenant not found in context")
		return
	}

	// Decode and validate request body
	var req dto.ServiceCreateOrUpdateRequestDto
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		util.Error(w, http.StatusBadRequest, "Invalid request", err.Error())
		return
	}

	if err := req.Validate(); err != nil {
		util.ValidationError(w, err)
		return
	}

	// Create service
	svc, err := h.service.Create(
		req.Name,
		req.DisplayName,
		req.Description,
		req.Version,
		false, // isDefault - only set by seeders
		false, // isSystem - only set by seeders
		req.Status,
		req.IsPublic,
		tenant.TenantID,
	)
	if err != nil {
		util.Error(w, http.StatusInternalServerError, "Failed to create service", err.Error())
		return
	}

	// Build response
	dtoRes := toServiceResponseDto(*svc)
	util.Created(w, dtoRes, "Service created successfully")
}

// Update updates an existing service.
//
// PUT /services/{service_uuid}
//
// Updates service information including name, display name, description, version,
// status, and public visibility. System and default flags cannot be modified
// (reserved for seeded services).
func (h *ServiceHandler) Update(w http.ResponseWriter, r *http.Request) {
	// Parse and validate service UUID from URL parameter
	serviceUUID, err := uuid.Parse(chi.URLParam(r, "service_uuid"))
	if err != nil {
		util.Error(w, http.StatusBadRequest, "Invalid service UUID")
		return
	}

	// Decode and validate request body
	var req dto.ServiceCreateOrUpdateRequestDto
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		util.Error(w, http.StatusBadRequest, "Invalid request", err.Error())
		return
	}

	if err := req.Validate(); err != nil {
		util.ValidationError(w, err)
		return
	}

	// Update service
	svc, err := h.service.Update(
		serviceUUID,
		req.Name,
		req.DisplayName,
		req.Description,
		req.Version,
		false, // isDefault - only set by seeders
		false, // isSystem - only set by seeders
		req.Status,
		req.IsPublic,
	)
	if err != nil {
		util.Error(w, http.StatusInternalServerError, "Failed to update service", err.Error())
		return
	}

	// Build response
	dtoRes := toServiceResponseDto(*svc)
	util.Success(w, dtoRes, "Service updated successfully")
}

// SetStatus updates the status of a service.
//
// PATCH /services/{service_uuid}/status
//
// Updates only the status field of a service (e.g., active, inactive).
// This is a convenience endpoint for status-only updates.
func (h *ServiceHandler) SetStatus(w http.ResponseWriter, r *http.Request) {
	// Parse and validate service UUID from URL parameter
	serviceUUID, err := uuid.Parse(chi.URLParam(r, "service_uuid"))
	if err != nil {
		util.Error(w, http.StatusBadRequest, "Invalid service UUID")
		return
	}

	// Decode and validate request body
	var req dto.ServiceStatusUpdateRequestDto
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		util.Error(w, http.StatusBadRequest, "Invalid request", err.Error())
		return
	}

	if err := req.Validate(); err != nil {
		util.ValidationError(w, err)
		return
	}

	// Update service status
	service, err := h.service.SetStatusByUUID(serviceUUID, req.Status)
	if err != nil {
		util.Error(w, http.StatusInternalServerError, "Failed to update service", err.Error())
		return
	}

	// Build response
	dtoRes := toServiceResponseDto(*service)

	util.Success(w, dtoRes, "Service updated successfully")
}

// SetPublic marks a service as public.
//
// PATCH /services/{service_uuid}/public
//
// Makes a service public (available to all tenants). This is useful for shared
// services that should be accessible across multiple tenants. Once public, the
// service can be used by any tenant.
func (h *ServiceHandler) SetPublic(w http.ResponseWriter, r *http.Request) {
	// Parse and validate service UUID from URL parameter
	serviceUUID, err := uuid.Parse(chi.URLParam(r, "service_uuid"))
	if err != nil {
		util.Error(w, http.StatusBadRequest, "Invalid service UUID")
		return
	}

	// Update service to public
	service, err := h.service.SetPublicStatusByUUID(serviceUUID)
	if err != nil {
		util.Error(w, http.StatusInternalServerError, "Failed to update service", err.Error())
		return
	}

	// Build response
	dtoRes := toServiceResponseDto(*service)

	util.Success(w, dtoRes, "Service updated successfully")
}

// Delete deletes a service.
//
// DELETE /services/{service_uuid}
//
// Permanently deletes a service from the system. This will also remove any
// associated policies and API relationships. System services cannot be deleted.
func (h *ServiceHandler) Delete(w http.ResponseWriter, r *http.Request) {
	// Parse and validate service UUID from URL parameter
	serviceUUID, err := uuid.Parse(chi.URLParam(r, "service_uuid"))
	if err != nil {
		util.Error(w, http.StatusBadRequest, "Invalid service UUID")
		return
	}

	// Delete service
	svc, err := h.service.DeleteByUUID(serviceUUID)
	if err != nil {
		util.Error(w, http.StatusInternalServerError, "Failed to delete service", err.Error())
		return
	}

	// Build response
	dtoRes := toServiceResponseDto(*svc)
	util.Success(w, dtoRes, "Service deleted successfully")
}

// Helper function for converting service data to response DTO

// toServiceResponseDto converts a service result to a service response DTO.
func toServiceResponseDto(s service.ServiceServiceDataResult) dto.ServiceResponseDto {
	return dto.ServiceResponseDto{
		ServiceUUID: s.ServiceUUID,
		Name:        s.Name,
		DisplayName: s.DisplayName,
		Description: s.Description,
		Version:     s.Version,
		Status:      s.Status,
		IsPublic:    s.IsPublic,
		IsDefault:   s.IsDefault,
		IsSystem:    s.IsSystem,
		APICount:    s.APICount,
		PolicyCount: s.PolicyCount,
		CreatedAt:   s.CreatedAt,
		UpdatedAt:   s.UpdatedAt,
	}
}

// AssignPolicy assigns a policy to a service for the tenant.
//
// POST /services/{service_uuid}/policies/{policy_uuid}
//
// Associates a policy with a service for access control. The policy and service
// must both belong to the same tenant. This enables fine-grained permission
// management for service access.
func (h *ServiceHandler) AssignPolicy(w http.ResponseWriter, r *http.Request) {
	// Get tenant from context (middleware already validated access)
	tenant, ok := r.Context().Value(middleware.TenantContextKey).(*model.Tenant)
	if !ok || tenant == nil {
		util.Error(w, http.StatusUnauthorized, "Tenant not found in context")
		return
	}

	// Parse and validate service UUID from URL parameter
	serviceUUID, err := uuid.Parse(chi.URLParam(r, "service_uuid"))
	if err != nil {
		util.Error(w, http.StatusBadRequest, "Invalid service UUID")
		return
	}

	// Parse and validate policy UUID from URL parameter
	policyUUID, err := uuid.Parse(chi.URLParam(r, "policy_uuid"))
	if err != nil {
		util.Error(w, http.StatusBadRequest, "Invalid policy UUID")
		return
	}

	// Assign policy to service (service validates tenant ownership)
	err = h.service.AssignPolicy(serviceUUID, policyUUID, tenant.TenantID)
	if err != nil {
		util.Error(w, http.StatusInternalServerError, "Failed to assign policy to service", err.Error())
		return
	}

	util.Success(w, nil, "Policy assigned to service successfully")
}

// RemovePolicy removes a policy from a service for the tenant.
//
// DELETE /services/{service_uuid}/policies/{policy_uuid}
//
// Removes the association between a policy and a service. This revokes the
// access control rules defined by that policy for the service. The policy
// and service must belong to the tenant.
func (h *ServiceHandler) RemovePolicy(w http.ResponseWriter, r *http.Request) {
	// Get tenant from context (middleware already validated access)
	tenant, ok := r.Context().Value(middleware.TenantContextKey).(*model.Tenant)
	if !ok || tenant == nil {
		util.Error(w, http.StatusUnauthorized, "Tenant not found in context")
		return
	}

	// Parse and validate service UUID from URL parameter
	serviceUUID, err := uuid.Parse(chi.URLParam(r, "service_uuid"))
	if err != nil {
		util.Error(w, http.StatusBadRequest, "Invalid service UUID")
		return
	}

	// Parse and validate policy UUID from URL parameter
	policyUUID, err := uuid.Parse(chi.URLParam(r, "policy_uuid"))
	if err != nil {
		util.Error(w, http.StatusBadRequest, "Invalid policy UUID")
		return
	}

	// Remove policy from service (service validates tenant ownership)
	err = h.service.RemovePolicy(serviceUUID, policyUUID, tenant.TenantID)
	if err != nil {
		util.Error(w, http.StatusInternalServerError, "Failed to remove policy from service", err.Error())
		return
	}

	util.Success(w, nil, "Policy removed from service successfully")
}
