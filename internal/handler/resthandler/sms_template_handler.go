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

type SmsTemplateHandler struct {
	smsTemplateService service.SmsTemplateService
}

func NewSmsTemplateHandler(smsTemplateService service.SmsTemplateService) *SmsTemplateHandler {
	return &SmsTemplateHandler{
		smsTemplateService: smsTemplateService,
	}
}

func (h *SmsTemplateHandler) GetAll(w http.ResponseWriter, r *http.Request) {
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

	// Validate filter
	if err := filter.Validate(); err != nil {
		util.ValidationError(w, err)
		return
	}

	result, err := h.smsTemplateService.GetAll(filter.Name, filter.Status, filter.IsDefault, filter.IsSystem, filter.Page, filter.Limit, filter.SortBy, filter.SortOrder)
	if err != nil {
		util.Error(w, http.StatusInternalServerError, "Failed to get SMS templates", err.Error())
		return
	}

	response := dto.PaginatedResponseDto[dto.SmsTemplateListResponseDto]{
		Rows:       toSmsTemplateListResponseDtoList(result.Data),
		Total:      result.Total,
		Page:       result.Page,
		Limit:      result.Limit,
		TotalPages: result.TotalPages,
	}

	util.Success(w, response, "SMS templates retrieved successfully")
}

func (h *SmsTemplateHandler) Get(w http.ResponseWriter, r *http.Request) {
	smsTemplateUUIDStr := chi.URLParam(r, "sms_template_uuid")

	smsTemplateUUID, err := uuid.Parse(smsTemplateUUIDStr)
	if err != nil {
		util.Error(w, http.StatusBadRequest, "Invalid SMS template UUID")
		return
	}

	template, err := h.smsTemplateService.GetByUUID(smsTemplateUUID)
	if err != nil {
		util.Error(w, http.StatusNotFound, "SMS template not found")
		return
	}

	util.Success(w, toSmsTemplateResponseDto(*template), "SMS template retrieved successfully")
}

func (h *SmsTemplateHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req dto.SmsTemplateCreateRequestDto
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

	template, err := h.smsTemplateService.Create(
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

func (h *SmsTemplateHandler) Update(w http.ResponseWriter, r *http.Request) {
	smsTemplateUUIDStr := chi.URLParam(r, "sms_template_uuid")

	smsTemplateUUID, err := uuid.Parse(smsTemplateUUIDStr)
	if err != nil {
		util.Error(w, http.StatusBadRequest, "Invalid SMS template UUID")
		return
	}

	var req dto.SmsTemplateUpdateRequestDto
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

	template, err := h.smsTemplateService.Update(
		smsTemplateUUID,
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

func (h *SmsTemplateHandler) Delete(w http.ResponseWriter, r *http.Request) {
	smsTemplateUUIDStr := chi.URLParam(r, "sms_template_uuid")

	smsTemplateUUID, err := uuid.Parse(smsTemplateUUIDStr)
	if err != nil {
		util.Error(w, http.StatusBadRequest, "Invalid SMS template UUID")
		return
	}

	template, err := h.smsTemplateService.Delete(smsTemplateUUID)
	if err != nil {
		util.Error(w, http.StatusBadRequest, "Failed to delete SMS template", err.Error())
		return
	}

	util.Success(w, toSmsTemplateResponseDto(*template), "SMS template deleted successfully")
}

func (h *SmsTemplateHandler) UpdateStatus(w http.ResponseWriter, r *http.Request) {
	smsTemplateUUIDStr := chi.URLParam(r, "sms_template_uuid")

	smsTemplateUUID, err := uuid.Parse(smsTemplateUUIDStr)
	if err != nil {
		util.Error(w, http.StatusBadRequest, "Invalid SMS template UUID")
		return
	}

	var req dto.SmsTemplateUpdateStatusRequestDto
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		util.Error(w, http.StatusBadRequest, "Invalid request body", err.Error())
		return
	}

	if err := req.Validate(); err != nil {
		util.ValidationError(w, err)
		return
	}

	template, err := h.smsTemplateService.UpdateStatus(smsTemplateUUID, req.Status)
	if err != nil {
		util.Error(w, http.StatusBadRequest, "Failed to update SMS template status", err.Error())
		return
	}

	util.Success(w, toSmsTemplateResponseDto(*template), "SMS template status updated successfully")
}

// Helper functions
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

func toSmsTemplateListResponseDtoList(templates []service.SmsTemplateServiceDataResult) []dto.SmsTemplateListResponseDto {
	result := make([]dto.SmsTemplateListResponseDto, len(templates))
	for i, template := range templates {
		result[i] = toSmsTemplateListResponseDto(template)
	}
	return result
}

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
