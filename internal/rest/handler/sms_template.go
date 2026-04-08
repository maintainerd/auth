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

// SMSTemplateHandler handles SMS template management operations.
//
// This handler manages tenant-scoped SMS templates used for sending text messages
// to users (e.g., verification codes, notifications). SMS templates define the
// message content, sender ID, and formatting. All operations are tenant-isolated -
// middleware validates tenant access and stores it in the request context.
type SMSTemplateHandler struct {
	smsTemplateService service.SMSTemplateService
}

// NewSMSTemplateHandler creates a new SMS template handler instance.
func NewSMSTemplateHandler(smsTemplateService service.SMSTemplateService) *SMSTemplateHandler {
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
	tenant, ok := r.Context().Value(middleware.TenantContextKey).(*model.Tenant)
	if !ok || tenant == nil {
		resp.Error(w, http.StatusUnauthorized, "Unauthorized")
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
	filter := dto.SMSTemplateFilterDTO{
		Name:      ptr.PtrOrNil(q.Get("name")),
		Status:    status,
		IsDefault: isDefault,
		IsSystem:  isSystem,
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

	// Fetch SMS templates from service layer
	result, err := h.smsTemplateService.GetAll(tenant.TenantID, filter.Name, filter.Status, filter.IsDefault, filter.IsSystem, filter.Page, filter.Limit, filter.SortBy, filter.SortOrder)
	if err != nil {
		resp.HandleServiceError(w, r, "Failed to get SMS templates", err)
		return
	}

	// Build paginated response
	response := dto.PaginatedResponseDTO[dto.SMSTemplateListResponseDTO]{
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
	tenant, ok := r.Context().Value(middleware.TenantContextKey).(*model.Tenant)
	if !ok || tenant == nil {
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
	template, err := h.smsTemplateService.GetByUUID(smsTemplateUUID, tenant.TenantID)
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
	tenant, ok := r.Context().Value(middleware.TenantContextKey).(*model.Tenant)
	if !ok || tenant == nil {
		resp.Error(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	// Decode and validate request body
	var req dto.SMSTemplateCreateRequestDTO
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		resp.Error(w, http.StatusBadRequest, "Invalid request")
		return
	}

	if err := req.Validate(); err != nil {
		resp.ValidationError(w, err)
		return
	}

	// Set default status if not provided
	status := model.StatusActive
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
	tenant, ok := r.Context().Value(middleware.TenantContextKey).(*model.Tenant)
	if !ok || tenant == nil {
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
	var req dto.SMSTemplateUpdateRequestDTO
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		resp.Error(w, http.StatusBadRequest, "Invalid request")
		return
	}

	if err := req.Validate(); err != nil {
		resp.ValidationError(w, err)
		return
	}

	// Set default status if not provided
	status := model.StatusActive
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
	tenant, ok := r.Context().Value(middleware.TenantContextKey).(*model.Tenant)
	if !ok || tenant == nil {
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
	template, err := h.smsTemplateService.Delete(smsTemplateUUID, tenant.TenantID)
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
	tenant, ok := r.Context().Value(middleware.TenantContextKey).(*model.Tenant)
	if !ok || tenant == nil {
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
	var req dto.SMSTemplateUpdateStatusRequestDTO
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		resp.Error(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if err := req.Validate(); err != nil {
		resp.ValidationError(w, err)
		return
	}

	// Update status (service validates tenant ownership)
	template, err := h.smsTemplateService.UpdateStatus(smsTemplateUUID, tenant.TenantID, req.Status)
	if err != nil {
		resp.HandleServiceError(w, r, "Failed to update SMS template status", err)
		return
	}

	resp.Success(w, toSMSTemplateResponseDTO(*template), "SMS template status updated successfully")
}

// Helper functions for converting service data to response DTOs

// toSMSTemplateListResponseDTO converts a service result to a list response DTO (without full message).
func toSMSTemplateListResponseDTO(template service.SMSTemplateServiceDataResult) dto.SMSTemplateListResponseDTO {
	return dto.SMSTemplateListResponseDTO{
		SMSTemplateID: template.SMSTemplateUUID.String(),
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

// toSMSTemplateListResponseDtoList converts a list of service results to list response DTOs.
func toSMSTemplateListResponseDtoList(templates []service.SMSTemplateServiceDataResult) []dto.SMSTemplateListResponseDTO {
	result := make([]dto.SMSTemplateListResponseDTO, len(templates))
	for i, template := range templates {
		result[i] = toSMSTemplateListResponseDTO(template)
	}
	return result
}

// toSMSTemplateResponseDTO converts a service result to a full response DTO (includes message content).
func toSMSTemplateResponseDTO(template service.SMSTemplateServiceDataResult) dto.SMSTemplateResponseDTO {
	return dto.SMSTemplateResponseDTO{
		SMSTemplateID: template.SMSTemplateUUID.String(),
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
