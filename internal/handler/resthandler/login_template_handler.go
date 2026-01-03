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

// LoginTemplateHandler handles HTTP requests for login template management.
// All endpoints are tenant-scoped - the middleware validates user access to the tenant
// and sets it in the request context. The service layer ensures templates belong to the tenant.
type LoginTemplateHandler struct {
	loginTemplateService service.LoginTemplateService
}

// NewLoginTemplateHandler creates a new instance of LoginTemplateHandler.
func NewLoginTemplateHandler(loginTemplateService service.LoginTemplateService) *LoginTemplateHandler {
	return &LoginTemplateHandler{
		loginTemplateService: loginTemplateService,
	}
}

// GetAll retrieves all login templates for the tenant with optional filtering and pagination.
// Tenant access is validated by middleware; this handler only needs to extract tenant from context.
// The service layer filters templates by tenant_id to ensure data isolation.
func (h *LoginTemplateHandler) GetAll(w http.ResponseWriter, r *http.Request) {
	// Tenant is already validated by middleware - just extract from context
	tenant, ok := r.Context().Value(middleware.TenantContextKey).(*model.Tenant)
	if !ok || tenant == nil {
		util.Error(w, http.StatusUnauthorized, "Tenant not found in context")
		return
	}

	// Extract query parameters
	q := r.URL.Query()

	// Parse pagination parameters with defaults
	page, _ := strconv.Atoi(q.Get("page"))
	if page < 1 {
		page = 1
	}

	limit, _ := strconv.Atoi(q.Get("limit"))
	if limit < 1 {
		limit = 10
	}

	// Parse status filter (can be comma-separated)
	var status []string
	if q.Get("status") != "" {
		status = strings.Split(q.Get("status"), ",")
	}

	// Parse boolean filters for default and system templates
	var isDefault *bool
	if q.Get("is_default") != "" {
		val := q.Get("is_default") == "true"
		isDefault = &val
	}

	var isSystem *bool
	if q.Get("is_system") != "" {
		val := q.Get("is_system") == "true"
		isSystem = &val
	}

	// Build filter DTO with all query parameters
	filter := dto.LoginTemplateFilterDto{
		Name:      util.PtrOrNil(q.Get("name")),
		Status:    status,
		Template:  util.PtrOrNil(q.Get("template")),
		IsDefault: isDefault,
		IsSystem:  isSystem,
		PaginationRequestDto: dto.PaginationRequestDto{
			Page:      page,
			Limit:     limit,
			SortBy:    q.Get("sort_by"),
			SortOrder: q.Get("sort_order"),
		},
	}

	// Validate filter parameters
	if err := filter.Validate(); err != nil {
		util.ValidationError(w, err)
		return
	}

	// Fetch templates from service - service filters by tenant_id
	result, err := h.loginTemplateService.GetAll(tenant.TenantID, filter.Name, filter.Status, filter.Template, filter.IsDefault, filter.IsSystem, filter.Page, filter.Limit, filter.SortBy, filter.SortOrder)
	if err != nil {
		util.Error(w, http.StatusInternalServerError, "Failed to retrieve login templates", err.Error())
		return
	}

	// Build paginated response
	response := dto.PaginatedResponseDto[dto.LoginTemplateListResponseDto]{
		Rows:       toLoginTemplateListResponseDtoList(result.Data),
		Total:      result.Total,
		Page:       result.Page,
		Limit:      result.Limit,
		TotalPages: result.TotalPages,
	}

	util.Success(w, response, "Login templates retrieved successfully")
}

// Get retrieves a specific login template by UUID.
// Tenant access is validated by middleware.
// The service layer verifies the template belongs to the tenant.
func (h *LoginTemplateHandler) Get(w http.ResponseWriter, r *http.Request) {
	// Tenant is already validated by middleware - just extract from context
	tenant, ok := r.Context().Value(middleware.TenantContextKey).(*model.Tenant)
	if !ok || tenant == nil {
		util.Error(w, http.StatusUnauthorized, "Tenant not found in context")
		return
	}

	// Extract template UUID from URL parameter
	loginTemplateUUIDStr := chi.URLParam(r, "login_template_uuid")

	// Parse and validate UUID format
	loginTemplateUUID, err := uuid.Parse(loginTemplateUUIDStr)
	if err != nil {
		util.Error(w, http.StatusBadRequest, "Invalid login template UUID")
		return
	}

	// Fetch template - service validates it belongs to tenant
	template, err := h.loginTemplateService.GetByUUID(loginTemplateUUID, tenant.TenantID)
	if err != nil {
		util.Error(w, http.StatusNotFound, "Login template not found")
		return
	}

	util.Success(w, toLoginTemplateResponseDto(*template), "Login template retrieved successfully")
}

// Create creates a new login template for the tenant.
// Tenant access is validated by middleware.
// The template is automatically associated with the tenant from context.
func (h *LoginTemplateHandler) Create(w http.ResponseWriter, r *http.Request) {
	// Tenant is already validated by middleware - just extract from context
	tenant, ok := r.Context().Value(middleware.TenantContextKey).(*model.Tenant)
	if !ok || tenant == nil {
		util.Error(w, http.StatusUnauthorized, "Tenant not found in context")
		return
	}

	// Decode request body
	var req dto.LoginTemplateCreateRequestDto
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		util.Error(w, http.StatusBadRequest, "Invalid request", err.Error())
		return
	}

	// Validate request data
	if err := req.Validate(); err != nil {
		util.ValidationError(w, err)
		return
	}

	// Set default status if not provided
	status := "active"
	if req.Status != nil {
		status = *req.Status
	}

	// Initialize metadata if not provided
	metadata := req.Metadata
	if metadata == nil {
		metadata = make(map[string]any)
	}

	// Create template associated with tenant
	template, err := h.loginTemplateService.Create(
		tenant.TenantID,
		req.Name,
		req.Description,
		req.Template,
		metadata,
		status,
	)
	if err != nil {
		util.Error(w, http.StatusBadRequest, "Failed to create login template", err.Error())
		return
	}

	util.Created(w, toLoginTemplateResponseDto(*template), "Login template created successfully")
}

// Update updates an existing login template.
// Tenant access is validated by middleware.
// The service layer verifies the template belongs to the tenant before updating.
func (h *LoginTemplateHandler) Update(w http.ResponseWriter, r *http.Request) {
	// Tenant is already validated by middleware - just extract from context
	tenant, ok := r.Context().Value(middleware.TenantContextKey).(*model.Tenant)
	if !ok || tenant == nil {
		util.Error(w, http.StatusUnauthorized, "Tenant not found in context")
		return
	}

	// Extract template UUID from URL parameter
	loginTemplateUUIDStr := chi.URLParam(r, "login_template_uuid")

	// Parse and validate UUID format
	loginTemplateUUID, err := uuid.Parse(loginTemplateUUIDStr)
	if err != nil {
		util.Error(w, http.StatusBadRequest, "Invalid login template UUID")
		return
	}

	// Decode request body
	var req dto.LoginTemplateUpdateRequestDto
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		util.Error(w, http.StatusBadRequest, "Invalid request", err.Error())
		return
	}

	// Validate request data
	if err := req.Validate(); err != nil {
		util.ValidationError(w, err)
		return
	}

	// Set default status if not provided
	status := "active"
	if req.Status != nil {
		status = *req.Status
	}

	// Initialize metadata if not provided
	metadata := req.Metadata
	if metadata == nil {
		metadata = make(map[string]any)
	}

	// Update template - service validates it belongs to tenant
	template, err := h.loginTemplateService.Update(
		loginTemplateUUID,
		tenant.TenantID,
		req.Name,
		req.Description,
		req.Template,
		metadata,
		status,
	)
	if err != nil {
		util.Error(w, http.StatusBadRequest, "Failed to update login template", err.Error())
		return
	}

	util.Success(w, toLoginTemplateResponseDto(*template), "Login template updated successfully")
}

// Delete soft-deletes a login template.
// Tenant access is validated by middleware.
// The service layer verifies the template belongs to the tenant before deletion.
func (h *LoginTemplateHandler) Delete(w http.ResponseWriter, r *http.Request) {
	// Tenant is already validated by middleware - just extract from context
	tenant, ok := r.Context().Value(middleware.TenantContextKey).(*model.Tenant)
	if !ok || tenant == nil {
		util.Error(w, http.StatusUnauthorized, "Tenant not found in context")
		return
	}

	// Extract template UUID from URL parameter
	loginTemplateUUIDStr := chi.URLParam(r, "login_template_uuid")

	// Parse and validate UUID format
	loginTemplateUUID, err := uuid.Parse(loginTemplateUUIDStr)
	if err != nil {
		util.Error(w, http.StatusBadRequest, "Invalid login template UUID")
		return
	}

	// Delete template - service validates it belongs to tenant
	template, err := h.loginTemplateService.Delete(loginTemplateUUID, tenant.TenantID)
	if err != nil {
		util.Error(w, http.StatusBadRequest, "Failed to delete login template", err.Error())
		return
	}

	util.Success(w, toLoginTemplateResponseDto(*template), "Login template deleted successfully")
}

// UpdateStatus updates the status of a login template (active/inactive).
// Tenant access is validated by middleware.
// The service layer verifies the template belongs to the tenant before updating status.
func (h *LoginTemplateHandler) UpdateStatus(w http.ResponseWriter, r *http.Request) {
	// Tenant is already validated by middleware - just extract from context
	tenant, ok := r.Context().Value(middleware.TenantContextKey).(*model.Tenant)
	if !ok || tenant == nil {
		util.Error(w, http.StatusUnauthorized, "Tenant not found in context")
		return
	}

	// Extract template UUID from URL parameter
	loginTemplateUUIDStr := chi.URLParam(r, "login_template_uuid")

	// Parse and validate UUID format
	loginTemplateUUID, err := uuid.Parse(loginTemplateUUIDStr)
	if err != nil {
		util.Error(w, http.StatusBadRequest, "Invalid login template UUID")
		return
	}

	// Decode request body
	var req dto.LoginTemplateUpdateStatusRequestDto
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		util.Error(w, http.StatusBadRequest, "Invalid request", err.Error())
		return
	}

	// Validate request data
	if err := req.Validate(); err != nil {
		util.ValidationError(w, err)
		return
	}

	// Update status - service validates template belongs to tenant
	template, err := h.loginTemplateService.UpdateStatus(loginTemplateUUID, tenant.TenantID, req.Status)
	if err != nil {
		util.Error(w, http.StatusBadRequest, "Failed to update login template status", err.Error())
		return
	}

	util.Success(w, toLoginTemplateResponseDto(*template), "Login template status updated successfully")
}

// Helper functions for converting service data to response DTOs

// toLoginTemplateListResponseDto converts a service result to a list response DTO.
func toLoginTemplateListResponseDto(template service.LoginTemplateServiceDataResult) dto.LoginTemplateListResponseDto {
	return dto.LoginTemplateListResponseDto{
		LoginTemplateID: template.LoginTemplateUUID.String(),
		Name:            template.Name,
		Description:     template.Description,
		Template:        template.Template,
		Status:          template.Status,
		IsDefault:       template.IsDefault,
		IsSystem:        template.IsSystem,
		CreatedAt:       template.CreatedAt,
		UpdatedAt:       template.UpdatedAt,
	}
}

// toLoginTemplateListResponseDtoList converts a slice of service results to list response DTOs.
func toLoginTemplateListResponseDtoList(templates []service.LoginTemplateServiceDataResult) []dto.LoginTemplateListResponseDto {
	result := make([]dto.LoginTemplateListResponseDto, len(templates))
	for i, template := range templates {
		result[i] = toLoginTemplateListResponseDto(template)
	}
	return result
}

// toLoginTemplateResponseDto converts a service result to a detailed response DTO.
func toLoginTemplateResponseDto(template service.LoginTemplateServiceDataResult) dto.LoginTemplateResponseDto {
	return dto.LoginTemplateResponseDto{
		LoginTemplateID: template.LoginTemplateUUID.String(),
		Name:            template.Name,
		Description:     template.Description,
		Template:        template.Template,
		Status:          template.Status,
		Metadata:        template.Metadata,
		IsDefault:       template.IsDefault,
		IsSystem:        template.IsSystem,
		CreatedAt:       template.CreatedAt,
		UpdatedAt:       template.UpdatedAt,
	}
}
