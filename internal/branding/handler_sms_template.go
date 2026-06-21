package branding

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/platform/middleware"
	"github.com/maintainerd/auth/internal/platform/pagination"
	"github.com/maintainerd/auth/internal/platform/ptr"
	resp "github.com/maintainerd/auth/internal/platform/response"
	"github.com/maintainerd/auth/internal/shared"
)

// SMSTemplateHandler handles SMS template management operations.
//
// This handler manages tenant-scoped SMS templates used for sending text messages
// to users (e.g., verification codes, notifications). SMS templates define the
// message content, sender ID, and formatting. All operations are tenant-isolated -
// middleware validates tenant access and stores it in the request context.
type SMSTemplateHandler struct {
	smsTemplateService SMSTemplateService
}

// NewSMSTemplateHandler creates a new SMS template handler instance.
func NewSMSTemplateHandler(smsTemplateService SMSTemplateService) *SMSTemplateHandler {
	return &SMSTemplateHandler{
		smsTemplateService: smsTemplateService,
	}
}

// GetAll retrieves all SMS templates for the tenant with pagination and filters.
//
// GET /sms-templates
//
// Returns a paginated list of SMS templates belonging to the authenticated tenant.
// Supports filtering by name, status, is_default, and is_system flags.
func (h *SMSTemplateHandler) GetAll(w http.ResponseWriter, r *http.Request) {
	// Get tenant from context (middleware already validated access)
	tenant := middleware.AuthFromRequest(r).Tenant
	if tenant == nil {
		resp.Error(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	// Parse query parameters
	q := r.URL.Query()

	// Parse pagination parameters

	// Parse status filter
	var status []string
	if v := q.Get("status"); v != "" {
		status = append(status, v)
	}

	// Parse boolean filters for default and system templates
	var isDefault *bool
	if v := q.Get("is_default"); v != "" {
		if val, err := strconv.ParseBool(v); err == nil {
			isDefault = &val
		}
	}

	var isSystem *bool
	if v := q.Get("is_system"); v != "" {
		if val, err := strconv.ParseBool(v); err == nil {
			isSystem = &val
		}
	}

	// Build filter DTO for validation
	filter := SMSTemplateFilterDTO{
		Name:                 ptr.PtrOrNil(q.Get("name")),
		Status:               status,
		IsDefault:            isDefault,
		IsSystem:             isSystem,
		PaginationRequestDTO: pagination.ParseQuery(r),
	}

	// Validate filter parameters
	if err := filter.Validate(); err != nil {
		resp.ValidationError(w, err)
		return
	}

	// Fetch SMS templates from service layer
	result, err := h.smsTemplateService.GetAll(r.Context(), tenant.TenantID, filter.Name, filter.Status, filter.IsDefault, filter.IsSystem, filter.Page, filter.Limit, filter.SortBy, filter.SortOrder)
	if err != nil {
		resp.HandleServiceError(w, r, "Failed to get SMS templates", err)
		return
	}

	// Build paginated response
	response := PaginatedResponseDTO[SMSTemplateListResponseDTO]{
		Rows:       toSMSTemplateListResponseDtoList(result.Data),
		Total:      result.Total,
		Page:       result.Page,
		Limit:      result.Limit,
		TotalPages: result.TotalPages,
	}

	resp.Success(w, response, "SMS templates retrieved successfully")
}

// Get retrieves a specific SMS template by UUID.
//
// GET /sms-templates/{sms_template_uuid}
//
// Returns detailed information about a single SMS template including the full message content.
// The service layer validates that the template belongs to the tenant.
func (h *SMSTemplateHandler) Get(w http.ResponseWriter, r *http.Request) {
	// Get tenant from context (middleware already validated access)
	tenant := middleware.AuthFromRequest(r).Tenant
	if tenant == nil {
		resp.Error(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	// Parse and validate SMS template UUID from URL parameter
	smsTemplateUUIDStr := chi.URLParam(r, "sms_template_uuid")
	smsTemplateUUID, err := uuid.Parse(smsTemplateUUIDStr)
	if err != nil {
		resp.Error(w, http.StatusBadRequest, "Invalid SMS template UUID")
		return
	}

	// Fetch SMS template (service validates tenant ownership)
	template, err := h.smsTemplateService.GetByUUID(r.Context(), smsTemplateUUID, tenant.TenantID)
	if err != nil {
		resp.HandleServiceError(w, r, "SMS template not found", err)
		return
	}

	resp.Success(w, toSMSTemplateResponseDTO(*template), "SMS template retrieved successfully")
}

// Create creates a new SMS template for the tenant.
//
// POST /sms-templates
//
// Creates a new SMS template with message content, sender ID, and configuration.
// Templates can be marked as default or system templates with appropriate flags.
func (h *SMSTemplateHandler) Create(w http.ResponseWriter, r *http.Request) {
	// Get tenant from context (middleware already validated access)
	tenant := middleware.AuthFromRequest(r).Tenant
	if tenant == nil {
		resp.Error(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	// Decode and validate request body
	var req SMSTemplateCreateRequestDTO
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		resp.BadRequestBody(w)
		return
	}

	if err := req.Validate(); err != nil {
		resp.ValidationError(w, err)
		return
	}

	// Set default status if not provided
	status := shared.StatusActive
	if req.Status != nil {
		status = *req.Status
	}

	// Create SMS template
	template, err := h.smsTemplateService.Create(
		r.Context(),
		tenant.TenantID,
		req.Name,
		req.Description,
		req.Message,
		status,
	)
	if err != nil {
		resp.HandleServiceError(w, r, "Failed to create SMS template", err)
		return
	}

	resp.Created(w, toSMSTemplateResponseDTO(*template), "SMS template created successfully")
}

// Update updates an existing SMS template.
//
// PUT /sms-templates/{sms_template_uuid}
//
// Updates the content, sender ID, and configuration of an existing SMS template.
// The service layer validates that the template belongs to the tenant.
func (h *SMSTemplateHandler) Update(w http.ResponseWriter, r *http.Request) {
	// Get tenant from context (middleware already validated access)
	tenant := middleware.AuthFromRequest(r).Tenant
	if tenant == nil {
		resp.Error(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	// Parse and validate SMS template UUID from URL parameter
	smsTemplateUUIDStr := chi.URLParam(r, "sms_template_uuid")
	smsTemplateUUID, err := uuid.Parse(smsTemplateUUIDStr)
	if err != nil {
		resp.Error(w, http.StatusBadRequest, "Invalid SMS template UUID")
		return
	}

	// Decode and validate request body
	var req SMSTemplateUpdateRequestDTO
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		resp.BadRequestBody(w)
		return
	}

	if err := req.Validate(); err != nil {
		resp.ValidationError(w, err)
		return
	}

	// Set default status if not provided
	status := shared.StatusActive
	if req.Status != nil {
		status = *req.Status
	}

	// Update SMS template (service validates tenant ownership)
	template, err := h.smsTemplateService.Update(
		r.Context(),
		smsTemplateUUID,
		tenant.TenantID,
		req.Description,
		req.Message,
		status,
	)
	if err != nil {
		resp.HandleServiceError(w, r, "Failed to update SMS template", err)
		return
	}

	resp.Success(w, toSMSTemplateResponseDTO(*template), "SMS template updated successfully")
}

// Delete deletes an SMS template.
//
// DELETE /sms-templates/{sms_template_uuid}
//
// Permanently deletes an SMS template from the tenant. System templates
// may be protected from deletion at the service layer.
func (h *SMSTemplateHandler) Delete(w http.ResponseWriter, r *http.Request) {
	// Get tenant from context (middleware already validated access)
	tenant := middleware.AuthFromRequest(r).Tenant
	if tenant == nil {
		resp.Error(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	// Parse and validate SMS template UUID from URL parameter
	smsTemplateUUIDStr := chi.URLParam(r, "sms_template_uuid")
	smsTemplateUUID, err := uuid.Parse(smsTemplateUUIDStr)
	if err != nil {
		resp.Error(w, http.StatusBadRequest, "Invalid SMS template UUID")
		return
	}

	// Delete SMS template (service validates tenant ownership)
	template, err := h.smsTemplateService.Delete(r.Context(), smsTemplateUUID, tenant.TenantID)
	if err != nil {
		resp.HandleServiceError(w, r, "Failed to delete SMS template", err)
		return
	}

	resp.Success(w, toSMSTemplateResponseDTO(*template), "SMS template deleted successfully")
}

// UpdateStatus updates the status of an SMS template.
//
// PATCH /sms-templates/{sms_template_uuid}/status
//
// Updates only the status field of an SMS template (e.g., active, inactive).
// This is a convenience endpoint for status-only updates.
func (h *SMSTemplateHandler) UpdateStatus(w http.ResponseWriter, r *http.Request) {
	// Get tenant from context (middleware already validated access)
	tenant := middleware.AuthFromRequest(r).Tenant
	if tenant == nil {
		resp.Error(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	// Parse and validate SMS template UUID from URL parameter
	smsTemplateUUIDStr := chi.URLParam(r, "sms_template_uuid")
	smsTemplateUUID, err := uuid.Parse(smsTemplateUUIDStr)
	if err != nil {
		resp.Error(w, http.StatusBadRequest, "Invalid SMS template UUID")
		return
	}

	// Decode and validate request body
	var req SMSTemplateUpdateStatusRequestDTO
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		resp.BadRequestBody(w)
		return
	}

	if err := req.Validate(); err != nil {
		resp.ValidationError(w, err)
		return
	}

	// Update status (service validates tenant ownership)
	template, err := h.smsTemplateService.UpdateStatus(r.Context(), smsTemplateUUID, tenant.TenantID, req.Status)
	if err != nil {
		resp.HandleServiceError(w, r, "Failed to update SMS template status", err)
		return
	}

	resp.Success(w, toSMSTemplateResponseDTO(*template), "SMS template status updated successfully")
}

// Helper functions for converting service data to response DTOs

// toSMSTemplateListResponseDTO converts a service result to a list response DTO (without full message).
func toSMSTemplateListResponseDTO(template SMSTemplateServiceDataResult) SMSTemplateListResponseDTO {
	return SMSTemplateListResponseDTO{
		SMSTemplateID: template.SMSTemplateUUID.String(),
		Name:          template.Name,
		Description:   template.Description,
		Status:        template.Status,
		IsDefault:     template.IsDefault,
		IsSystem:      template.IsSystem,
		CreatedAt:     template.CreatedAt,
		UpdatedAt:     template.UpdatedAt,
	}
}

// toSMSTemplateListResponseDtoList converts a list of service results to list response DTOs.
func toSMSTemplateListResponseDtoList(templates []SMSTemplateServiceDataResult) []SMSTemplateListResponseDTO {
	result := make([]SMSTemplateListResponseDTO, len(templates))
	for i, template := range templates {
		result[i] = toSMSTemplateListResponseDTO(template)
	}
	return result
}

// toSMSTemplateResponseDTO converts a service result to a full response DTO (includes message content).
func toSMSTemplateResponseDTO(template SMSTemplateServiceDataResult) SMSTemplateResponseDTO {
	return SMSTemplateResponseDTO{
		SMSTemplateID: template.SMSTemplateUUID.String(),
		Name:          template.Name,
		Description:   template.Description,
		Message:       template.Message,
		ParametersDoc: template.ParametersDoc,
		Status:        template.Status,
		IsDefault:     template.IsDefault,
		IsSystem:      template.IsSystem,
		CreatedAt:     template.CreatedAt,
		UpdatedAt:     template.UpdatedAt,
	}
}
