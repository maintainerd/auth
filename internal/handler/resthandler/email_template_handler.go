package resthandler

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
	"github.com/maintainerd/auth/internal/util"
)

// EmailTemplateHandler handles HTTP requests for email template management.
// All endpoints are tenant-scoped - the middleware validates user access to the tenant
// and sets it in the request context. The service layer ensures templates belong to the tenant.
type EmailTemplateHandler struct {
	emailTemplateService service.EmailTemplateService
}

// NewEmailTemplateHandler creates a new instance of EmailTemplateHandler.
func NewEmailTemplateHandler(emailTemplateService service.EmailTemplateService) *EmailTemplateHandler {
	return &EmailTemplateHandler{
		emailTemplateService: emailTemplateService,
	}
}

// GetAll retrieves all email templates for the tenant with optional filtering and pagination.
// Tenant access is validated by middleware; this handler only needs to extract tenant from context.
// The service layer filters templates by tenant_id to ensure data isolation.
func (h *EmailTemplateHandler) GetAll(w http.ResponseWriter, r *http.Request) {
	// Tenant is already validated by middleware - just extract from context
	tenant, ok := r.Context().Value(middleware.TenantContextKey).(*model.Tenant)
	if !ok || tenant == nil {
		util.Error(w, http.StatusUnauthorized, "Tenant not found in context")
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

	// Parse boolean filters for default and system templates
	var isDefault *bool
	if v := q.Get("is_default"); v != "" {
		val := v == "true"
		isDefault = &val
	}

	var isSystem *bool
	if v := q.Get("is_system"); v != "" {
		val := v == "true"
		isSystem = &val
	}

	// Build filter DTO with all query parameters
	filter := dto.EmailTemplateFilterDto{
		Name:      util.PtrOrNil(q.Get("name")),
		Status:    status,
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
	result, err := h.emailTemplateService.GetAll(tenant.TenantID, filter.Name, filter.Status, filter.IsDefault, filter.IsSystem, filter.Page, filter.Limit, filter.SortBy, filter.SortOrder)
	if err != nil {
		util.Error(w, http.StatusInternalServerError, "Failed to get email templates", err.Error())
		return
	}

	// Build paginated response
	response := dto.PaginatedResponseDto[dto.EmailTemplateListResponseDto]{
		Rows:       toEmailTemplateListResponseDtoList(result.Data),
		Total:      result.Total,
		Page:       result.Page,
		Limit:      result.Limit,
		TotalPages: result.TotalPages,
	}

	util.Success(w, response, "Email templates retrieved successfully")
}

// Get retrieves a specific email template by UUID.
// Tenant access is validated by middleware.
// The service layer verifies the template belongs to the tenant.
func (h *EmailTemplateHandler) Get(w http.ResponseWriter, r *http.Request) {
	// Tenant is already validated by middleware - just extract from context
	tenant, ok := r.Context().Value(middleware.TenantContextKey).(*model.Tenant)
	if !ok || tenant == nil {
		util.Error(w, http.StatusUnauthorized, "Tenant not found in context")
		return
	}

	// Extract template UUID from URL parameter
	emailTemplateUUIDStr := chi.URLParam(r, "email_template_uuid")

	// Parse and validate UUID format
	emailTemplateUUID, err := uuid.Parse(emailTemplateUUIDStr)
	if err != nil {
		util.Error(w, http.StatusBadRequest, "Invalid email template UUID")
		return
	}

	// Fetch template - service validates it belongs to tenant
	template, err := h.emailTemplateService.GetByUUID(emailTemplateUUID, tenant.TenantID)
	if err != nil {
		util.Error(w, http.StatusNotFound, "Email template not found")
		return
	}

	util.Success(w, toEmailTemplateResponseDto(*template), "Email template retrieved successfully")
}

// Create creates a new email template for the tenant.
// Tenant access is validated by middleware.
// The template is automatically associated with the tenant from context.
func (h *EmailTemplateHandler) Create(w http.ResponseWriter, r *http.Request) {
	// Tenant is already validated by middleware - just extract from context
	tenant, ok := r.Context().Value(middleware.TenantContextKey).(*model.Tenant)
	if !ok || tenant == nil {
		util.Error(w, http.StatusUnauthorized, "Tenant not found in context")
		return
	}

	// Decode request body
	var req dto.EmailTemplateCreateRequestDto
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

	// Create template associated with tenant
	template, err := h.emailTemplateService.Create(
		tenant.TenantID,
		req.Name,
		req.Subject,
		req.BodyHtml,
		req.BodyPlain,
		status,
		false, // is_default always false on create
	)
	if err != nil {
		util.Error(w, http.StatusBadRequest, "Failed to create email template", err.Error())
		return
	}

	util.Created(w, toEmailTemplateResponseDto(*template), "Email template created successfully")
}

// Update updates an existing email template.
// Tenant access is validated by middleware.
// The service layer verifies the template belongs to the tenant before updating.
func (h *EmailTemplateHandler) Update(w http.ResponseWriter, r *http.Request) {
	// Tenant is already validated by middleware - just extract from context
	tenant, ok := r.Context().Value(middleware.TenantContextKey).(*model.Tenant)
	if !ok || tenant == nil {
		util.Error(w, http.StatusUnauthorized, "Tenant not found in context")
		return
	}

	// Extract template UUID from URL parameter
	emailTemplateUUIDStr := chi.URLParam(r, "email_template_uuid")

	// Parse and validate UUID format
	emailTemplateUUID, err := uuid.Parse(emailTemplateUUIDStr)
	if err != nil {
		util.Error(w, http.StatusBadRequest, "Invalid email template UUID")
		return
	}

	// Decode request body
	var req dto.EmailTemplateUpdateRequestDto
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

	// Update template - service validates it belongs to tenant
	template, err := h.emailTemplateService.Update(
		emailTemplateUUID,
		tenant.TenantID,
		req.Name,
		req.Subject,
		req.BodyHtml,
		req.BodyPlain,
		status,
	)
	if err != nil {
		util.Error(w, http.StatusBadRequest, "Failed to update email template", err.Error())
		return
	}

	util.Success(w, toEmailTemplateResponseDto(*template), "Email template updated successfully")
}

// Delete soft-deletes an email template.
// Tenant access is validated by middleware.
// The service layer verifies the template belongs to the tenant before deletion.
func (h *EmailTemplateHandler) Delete(w http.ResponseWriter, r *http.Request) {
	// Tenant is already validated by middleware - just extract from context
	tenant, ok := r.Context().Value(middleware.TenantContextKey).(*model.Tenant)
	if !ok || tenant == nil {
		util.Error(w, http.StatusUnauthorized, "Tenant not found in context")
		return
	}

	// Extract template UUID from URL parameter
	emailTemplateUUIDStr := chi.URLParam(r, "email_template_uuid")

	// Parse and validate UUID format
	emailTemplateUUID, err := uuid.Parse(emailTemplateUUIDStr)
	if err != nil {
		util.Error(w, http.StatusBadRequest, "Invalid email template UUID")
		return
	}

	// Delete template - service validates it belongs to tenant
	template, err := h.emailTemplateService.Delete(emailTemplateUUID, tenant.TenantID)
	if err != nil {
		util.Error(w, http.StatusBadRequest, "Failed to delete email template", err.Error())
		return
	}

	util.Success(w, toEmailTemplateResponseDto(*template), "Email template deleted successfully")
}

// UpdateStatus updates the status of an email template (active/inactive).
// Tenant access is validated by middleware.
// The service layer verifies the template belongs to the tenant before updating status.
func (h *EmailTemplateHandler) UpdateStatus(w http.ResponseWriter, r *http.Request) {
	// Tenant is already validated by middleware - just extract from context
	tenant, ok := r.Context().Value(middleware.TenantContextKey).(*model.Tenant)
	if !ok || tenant == nil {
		util.Error(w, http.StatusUnauthorized, "Tenant not found in context")
		return
	}

	// Extract template UUID from URL parameter
	emailTemplateUUIDStr := chi.URLParam(r, "email_template_uuid")

	// Parse and validate UUID format
	emailTemplateUUID, err := uuid.Parse(emailTemplateUUIDStr)
	if err != nil {
		util.Error(w, http.StatusBadRequest, "Invalid email template UUID")
		return
	}

	// Decode request body
	var req dto.EmailTemplateUpdateStatusRequestDto
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		util.Error(w, http.StatusBadRequest, "Invalid request body", err.Error())
		return
	}

	// Validate request data
	if err := req.Validate(); err != nil {
		util.ValidationError(w, err)
		return
	}

	// Update status - service validates template belongs to tenant
	template, err := h.emailTemplateService.UpdateStatus(emailTemplateUUID, tenant.TenantID, req.Status)
	if err != nil {
		util.Error(w, http.StatusBadRequest, "Failed to update email template status", err.Error())
		return
	}

	util.Success(w, toEmailTemplateResponseDto(*template), "Email template status updated successfully")
}

// Helper functions for converting service data to response DTOs

// toEmailTemplateListResponseDto converts a service result to a list response DTO.
func toEmailTemplateListResponseDto(template service.EmailTemplateServiceDataResult) dto.EmailTemplateListResponseDto {
	return dto.EmailTemplateListResponseDto{
		EmailTemplateID: template.EmailTemplateUUID.String(),
		Name:            template.Name,
		Subject:         template.Subject,
		Status:          template.Status,
		IsDefault:       template.IsDefault,
		IsSystem:        template.IsSystem,
		CreatedAt:       template.CreatedAt,
		UpdatedAt:       template.UpdatedAt,
	}
}

// toEmailTemplateListResponseDtoList converts a slice of service results to list response DTOs.
func toEmailTemplateListResponseDtoList(templates []service.EmailTemplateServiceDataResult) []dto.EmailTemplateListResponseDto {
	result := make([]dto.EmailTemplateListResponseDto, len(templates))
	for i, template := range templates {
		result[i] = toEmailTemplateListResponseDto(template)
	}
	return result
}

// toEmailTemplateResponseDto converts a service result to a detailed response DTO.
func toEmailTemplateResponseDto(template service.EmailTemplateServiceDataResult) dto.EmailTemplateResponseDto {
	return dto.EmailTemplateResponseDto{
		EmailTemplateID: template.EmailTemplateUUID.String(),
		Name:            template.Name,
		Subject:         template.Subject,
		BodyHtml:        template.BodyHTML,
		BodyPlain:       template.BodyPlain,
		Status:          template.Status,
		IsDefault:       template.IsDefault,
		IsSystem:        template.IsSystem,
		CreatedAt:       template.CreatedAt,
		UpdatedAt:       template.UpdatedAt,
	}
}

// toEmailTemplateResponseDtoList converts a slice of service results to detailed response DTOs.
func toEmailTemplateResponseDtoList(templates []service.EmailTemplateServiceDataResult) []dto.EmailTemplateResponseDto {
	result := make([]dto.EmailTemplateResponseDto, len(templates))
	for i, template := range templates {
		result[i] = toEmailTemplateResponseDto(template)
	}
	return result
}
