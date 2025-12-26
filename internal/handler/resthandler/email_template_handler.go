package resthandler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/dto"
	"github.com/maintainerd/auth/internal/service"
	"github.com/maintainerd/auth/internal/util"
)

type EmailTemplateHandler struct {
	emailTemplateService service.EmailTemplateService
}

func NewEmailTemplateHandler(emailTemplateService service.EmailTemplateService) *EmailTemplateHandler {
	return &EmailTemplateHandler{
		emailTemplateService: emailTemplateService,
	}
}

func (h *EmailTemplateHandler) GetAll(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()

	// Parse pagination
	page, _ := strconv.Atoi(q.Get("page"))
	limit, _ := strconv.Atoi(q.Get("limit"))

	// Parse status filter (can be multiple)
	var status []string
	if v := q.Get("status"); v != "" {
		status = append(status, v)
	}

	// Parse boolean filters
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

	// Build filter DTO
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

	// Validate filter
	if err := filter.Validate(); err != nil {
		util.ValidationError(w, err)
		return
	}

	result, err := h.emailTemplateService.GetAll(filter.Name, filter.Status, filter.IsDefault, filter.IsSystem, filter.Page, filter.Limit, filter.SortBy, filter.SortOrder)
	if err != nil {
		util.Error(w, http.StatusInternalServerError, "Failed to get email templates", err.Error())
		return
	}

	response := dto.PaginatedResponseDto[dto.EmailTemplateListResponseDto]{
		Rows:       toEmailTemplateListResponseDtoList(result.Data),
		Total:      result.Total,
		Page:       result.Page,
		Limit:      result.Limit,
		TotalPages: result.TotalPages,
	}

	util.Success(w, response, "Email templates retrieved successfully")
}

func (h *EmailTemplateHandler) Get(w http.ResponseWriter, r *http.Request) {
	emailTemplateUUIDStr := chi.URLParam(r, "email_template_uuid")

	emailTemplateUUID, err := uuid.Parse(emailTemplateUUIDStr)
	if err != nil {
		util.Error(w, http.StatusBadRequest, "Invalid email template UUID")
		return
	}

	template, err := h.emailTemplateService.GetByUUID(emailTemplateUUID)
	if err != nil {
		util.Error(w, http.StatusNotFound, "Email template not found")
		return
	}

	util.Success(w, toEmailTemplateResponseDto(*template), "Email template retrieved successfully")
}

func (h *EmailTemplateHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req dto.EmailTemplateCreateRequestDto
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		util.Error(w, http.StatusBadRequest, "Invalid request", err.Error())
		return
	}

	if err := req.Validate(); err != nil {
		util.ValidationError(w, err)
		return
	}

	status := "active"
	if req.Status != nil {
		status = *req.Status
	}

	template, err := h.emailTemplateService.Create(
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

func (h *EmailTemplateHandler) Update(w http.ResponseWriter, r *http.Request) {
	emailTemplateUUIDStr := chi.URLParam(r, "email_template_uuid")

	emailTemplateUUID, err := uuid.Parse(emailTemplateUUIDStr)
	if err != nil {
		util.Error(w, http.StatusBadRequest, "Invalid email template UUID")
		return
	}

	var req dto.EmailTemplateUpdateRequestDto
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		util.Error(w, http.StatusBadRequest, "Invalid request", err.Error())
		return
	}

	if err := req.Validate(); err != nil {
		util.ValidationError(w, err)
		return
	}

	status := "active"
	if req.Status != nil {
		status = *req.Status
	}

	template, err := h.emailTemplateService.Update(
		emailTemplateUUID,
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

func (h *EmailTemplateHandler) Delete(w http.ResponseWriter, r *http.Request) {
	emailTemplateUUIDStr := chi.URLParam(r, "email_template_uuid")

	emailTemplateUUID, err := uuid.Parse(emailTemplateUUIDStr)
	if err != nil {
		util.Error(w, http.StatusBadRequest, "Invalid email template UUID")
		return
	}

	template, err := h.emailTemplateService.Delete(emailTemplateUUID)
	if err != nil {
		util.Error(w, http.StatusBadRequest, "Failed to delete email template", err.Error())
		return
	}

	util.Success(w, toEmailTemplateResponseDto(*template), "Email template deleted successfully")
}

func (h *EmailTemplateHandler) UpdateStatus(w http.ResponseWriter, r *http.Request) {
	emailTemplateUUIDStr := chi.URLParam(r, "email_template_uuid")

	emailTemplateUUID, err := uuid.Parse(emailTemplateUUIDStr)
	if err != nil {
		util.Error(w, http.StatusBadRequest, "Invalid email template UUID")
		return
	}

	var req dto.EmailTemplateUpdateStatusRequestDto
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		util.Error(w, http.StatusBadRequest, "Invalid request body", err.Error())
		return
	}

	if err := req.Validate(); err != nil {
		util.ValidationError(w, err)
		return
	}

	template, err := h.emailTemplateService.UpdateStatus(emailTemplateUUID, req.Status)
	if err != nil {
		util.Error(w, http.StatusBadRequest, "Failed to update email template status", err.Error())
		return
	}

	util.Success(w, toEmailTemplateResponseDto(*template), "Email template status updated successfully")
}

// Helper functions
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

func toEmailTemplateListResponseDtoList(templates []service.EmailTemplateServiceDataResult) []dto.EmailTemplateListResponseDto {
	result := make([]dto.EmailTemplateListResponseDto, len(templates))
	for i, template := range templates {
		result[i] = toEmailTemplateListResponseDto(template)
	}
	return result
}

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

func toEmailTemplateResponseDtoList(templates []service.EmailTemplateServiceDataResult) []dto.EmailTemplateResponseDto {
	result := make([]dto.EmailTemplateResponseDto, len(templates))
	for i, template := range templates {
		result[i] = toEmailTemplateResponseDto(template)
	}
	return result
}
