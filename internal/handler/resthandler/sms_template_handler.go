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

// SmsTemplateHandler handles SMS template management operations.
//
// This handler manages tenant-scoped SMS templates used for sending text messages
// to users (e.g., verification codes, notifications). SMS templates define the
// message content, sender ID, and formatting. All operations are tenant-isolated -
// middleware validates tenant access and stores it in the request context.
type SmsTemplateHandler struct {
	smsTemplateService service.SmsTemplateService
}

// NewSmsTemplateHandler creates a new SMS template handler instance.
func NewSmsTemplateHandler(smsTemplateService service.SmsTemplateService) *SmsTemplateHandler {
	return &SmsTemplateHandler{
		smsTemplateService: smsTemplateService,
	}
}

// GetAll retrieves all SMS templates for the tenant with pagination and filters.
//
// GET /sms-templates
//
// Returns a paginated list of SMS templates belonging to the authenticated tenant.
// Supports filtering by name, status, is_default, and is_system flags.
func (h *SmsTemplateHandler) GetAll(w http.ResponseWriter, r *http.Request) {
	// Get tenant from context (middleware already validated access)
	tenant, ok := r.Context().Value(middleware.TenantContextKey).(*model.Tenant)
	if !ok || tenant == nil {
		util.Error(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	// Parse query parameters
	q := r.URL.Query()

	// Parse pagination parameters
	page, _ := strconv.Atoi(q.Get("page"))
	limit, _ := strconv.Atoi(q.Get("limit"))

	// Parse status filter
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

	// Build filter DTO for validation
	filter := dto.SmsTemplateFilterDto{
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

	// Fetch SMS templates from service layer
	result, err := h.smsTemplateService.GetAll(tenant.TenantID, filter.Name, filter.Status, filter.IsDefault, filter.IsSystem, filter.Page, filter.Limit, filter.SortBy, filter.SortOrder)
	if err != nil {
		util.Error(w, http.StatusInternalServerError, "Failed to get SMS templates", err.Error())
		return
	}

	// Build paginated response
	response := dto.PaginatedResponseDto[dto.SmsTemplateListResponseDto]{
		Rows:       toSmsTemplateListResponseDtoList(result.Data),
		Total:      result.Total,
		Page:       result.Page,
		Limit:      result.Limit,
		TotalPages: result.TotalPages,
	}

	util.Success(w, response, "SMS templates retrieved successfully")
}

// Get retrieves a specific SMS template by UUID.
//
// GET /sms-templates/{sms_template_uuid}
//
// Returns detailed information about a single SMS template including the full message content.
// The service layer validates that the template belongs to the tenant.
func (h *SmsTemplateHandler) Get(w http.ResponseWriter, r *http.Request) {
	// Get tenant from context (middleware already validated access)
	tenant, ok := r.Context().Value(middleware.TenantContextKey).(*model.Tenant)
	if !ok || tenant == nil {
		util.Error(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	// Parse and validate SMS template UUID from URL parameter
	smsTemplateUUIDStr := chi.URLParam(r, "sms_template_uuid")
	smsTemplateUUID, err := uuid.Parse(smsTemplateUUIDStr)
	if err != nil {
		util.Error(w, http.StatusBadRequest, "Invalid SMS template UUID")
		return
	}

	// Fetch SMS template (service validates tenant ownership)
	template, err := h.smsTemplateService.GetByUUID(smsTemplateUUID, tenant.TenantID)
	if err != nil {
		util.Error(w, http.StatusNotFound, "SMS template not found")
		return
	}

	util.Success(w, toSmsTemplateResponseDto(*template), "SMS template retrieved successfully")
}

// Create creates a new SMS template for the tenant.
//
// POST /sms-templates
//
// Creates a new SMS template with message content, sender ID, and configuration.
// Templates can be marked as default or system templates with appropriate flags.
func (h *SmsTemplateHandler) Create(w http.ResponseWriter, r *http.Request) {
	// Get tenant from context (middleware already validated access)
	tenant, ok := r.Context().Value(middleware.TenantContextKey).(*model.Tenant)
	if !ok || tenant == nil {
		util.Error(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	// Decode and validate request body
	var req dto.SmsTemplateCreateRequestDto
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		util.Error(w, http.StatusBadRequest, "Invalid request", err.Error())
		return
	}

	if err := req.Validate(); err != nil {
		util.ValidationError(w, err)
		return
	}

	// Set default status if not provided
	status := "active"
	if req.Status != nil {
		status = *req.Status
	}

	// Create SMS template
	template, err := h.smsTemplateService.Create(
		tenant.TenantID,
		req.Name,
		req.Description,
		req.Message,
		req.SenderID,
		status,
	)
	if err != nil {
		util.Error(w, http.StatusBadRequest, "Failed to create SMS template", err.Error())
		return
	}

	util.Created(w, toSmsTemplateResponseDto(*template), "SMS template created successfully")
}

// Update updates an existing SMS template.
//
// PUT /sms-templates/{sms_template_uuid}
//
// Updates the content, sender ID, and configuration of an existing SMS template.
// The service layer validates that the template belongs to the tenant.
func (h *SmsTemplateHandler) Update(w http.ResponseWriter, r *http.Request) {
	// Get tenant from context (middleware already validated access)
	tenant, ok := r.Context().Value(middleware.TenantContextKey).(*model.Tenant)
	if !ok || tenant == nil {
		util.Error(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	// Parse and validate SMS template UUID from URL parameter
	smsTemplateUUIDStr := chi.URLParam(r, "sms_template_uuid")
	smsTemplateUUID, err := uuid.Parse(smsTemplateUUIDStr)
	if err != nil {
		util.Error(w, http.StatusBadRequest, "Invalid SMS template UUID")
		return
	}

	// Decode and validate request body
	var req dto.SmsTemplateUpdateRequestDto
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		util.Error(w, http.StatusBadRequest, "Invalid request", err.Error())
		return
	}

	if err := req.Validate(); err != nil {
		util.ValidationError(w, err)
		return
	}

	// Set default status if not provided
	status := "active"
	if req.Status != nil {
		status = *req.Status
	}

	// Update SMS template (service validates tenant ownership)
	template, err := h.smsTemplateService.Update(
		smsTemplateUUID,
		tenant.TenantID,
		req.Name,
		req.Description,
		req.Message,
		req.SenderID,
		status,
	)
	if err != nil {
		util.Error(w, http.StatusBadRequest, "Failed to update SMS template", err.Error())
		return
	}

	util.Success(w, toSmsTemplateResponseDto(*template), "SMS template updated successfully")
}

// Delete deletes an SMS template.
//
// DELETE /sms-templates/{sms_template_uuid}
//
// Permanently deletes an SMS template from the tenant. System templates
// may be protected from deletion at the service layer.
func (h *SmsTemplateHandler) Delete(w http.ResponseWriter, r *http.Request) {
	// Get tenant from context (middleware already validated access)
	tenant, ok := r.Context().Value(middleware.TenantContextKey).(*model.Tenant)
	if !ok || tenant == nil {
		util.Error(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	// Parse and validate SMS template UUID from URL parameter
	smsTemplateUUIDStr := chi.URLParam(r, "sms_template_uuid")
	smsTemplateUUID, err := uuid.Parse(smsTemplateUUIDStr)
	if err != nil {
		util.Error(w, http.StatusBadRequest, "Invalid SMS template UUID")
		return
	}

	// Delete SMS template (service validates tenant ownership)
	template, err := h.smsTemplateService.Delete(smsTemplateUUID, tenant.TenantID)
	if err != nil {
		util.Error(w, http.StatusBadRequest, "Failed to delete SMS template", err.Error())
		return
	}

	util.Success(w, toSmsTemplateResponseDto(*template), "SMS template deleted successfully")
}

// UpdateStatus updates the status of an SMS template.
//
// PATCH /sms-templates/{sms_template_uuid}/status
//
// Updates only the status field of an SMS template (e.g., active, inactive).
// This is a convenience endpoint for status-only updates.
func (h *SmsTemplateHandler) UpdateStatus(w http.ResponseWriter, r *http.Request) {
	// Get tenant from context (middleware already validated access)
	tenant, ok := r.Context().Value(middleware.TenantContextKey).(*model.Tenant)
	if !ok || tenant == nil {
		util.Error(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	// Parse and validate SMS template UUID from URL parameter
	smsTemplateUUIDStr := chi.URLParam(r, "sms_template_uuid")
	smsTemplateUUID, err := uuid.Parse(smsTemplateUUIDStr)
	if err != nil {
		util.Error(w, http.StatusBadRequest, "Invalid SMS template UUID")
		return
	}

	// Decode and validate request body
	var req dto.SmsTemplateUpdateStatusRequestDto
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		util.Error(w, http.StatusBadRequest, "Invalid request body", err.Error())
		return
	}

	if err := req.Validate(); err != nil {
		util.ValidationError(w, err)
		return
	}

	// Update status (service validates tenant ownership)
	template, err := h.smsTemplateService.UpdateStatus(smsTemplateUUID, tenant.TenantID, req.Status)
	if err != nil {
		util.Error(w, http.StatusBadRequest, "Failed to update SMS template status", err.Error())
		return
	}

	util.Success(w, toSmsTemplateResponseDto(*template), "SMS template status updated successfully")
}

// Helper functions for converting service data to response DTOs

// toSmsTemplateListResponseDto converts a service result to a list response DTO (without full message).
func toSmsTemplateListResponseDto(template service.SmsTemplateServiceDataResult) dto.SmsTemplateListResponseDto {
	return dto.SmsTemplateListResponseDto{
		SmsTemplateID: template.SmsTemplateUUID.String(),
		Name:          template.Name,
		Description:   template.Description,
		SenderID:      template.SenderID,
		Status:        template.Status,
		IsDefault:     template.IsDefault,
		IsSystem:      template.IsSystem,
		CreatedAt:     template.CreatedAt,
		UpdatedAt:     template.UpdatedAt,
	}
}

// toSmsTemplateListResponseDtoList converts a list of service results to list response DTOs.
func toSmsTemplateListResponseDtoList(templates []service.SmsTemplateServiceDataResult) []dto.SmsTemplateListResponseDto {
	result := make([]dto.SmsTemplateListResponseDto, len(templates))
	for i, template := range templates {
		result[i] = toSmsTemplateListResponseDto(template)
	}
	return result
}

// toSmsTemplateResponseDto converts a service result to a full response DTO (includes message content).
func toSmsTemplateResponseDto(template service.SmsTemplateServiceDataResult) dto.SmsTemplateResponseDto {
	return dto.SmsTemplateResponseDto{
		SmsTemplateID: template.SmsTemplateUUID.String(),
		Name:          template.Name,
		Description:   template.Description,
		Message:       template.Message,
		SenderID:      template.SenderID,
		Status:        template.Status,
		IsDefault:     template.IsDefault,
		IsSystem:      template.IsSystem,
		CreatedAt:     template.CreatedAt,
		UpdatedAt:     template.UpdatedAt,
	}
}
