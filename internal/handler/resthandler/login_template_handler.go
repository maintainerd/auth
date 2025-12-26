package resthandler

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/dto"
	"github.com/maintainerd/auth/internal/service"
	"github.com/maintainerd/auth/internal/util"
)

type LoginTemplateHandler struct {
	loginTemplateService service.LoginTemplateService
}

func NewLoginTemplateHandler(loginTemplateService service.LoginTemplateService) *LoginTemplateHandler {
	return &LoginTemplateHandler{
		loginTemplateService: loginTemplateService,
	}
}

func (h *LoginTemplateHandler) GetAll(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()

	// Parse pagination
	page, _ := strconv.Atoi(q.Get("page"))
	if page < 1 {
		page = 1
	}

	limit, _ := strconv.Atoi(q.Get("limit"))
	if limit < 1 {
		limit = 10
	}

	// Parse filters
	var status []string
	if q.Get("status") != "" {
		status = strings.Split(q.Get("status"), ",")
	}

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

	// Validate filter
	if err := filter.Validate(); err != nil {
		util.ValidationError(w, err)
		return
	}

	result, err := h.loginTemplateService.GetAll(filter.Name, filter.Status, filter.Template, filter.IsDefault, filter.IsSystem, filter.Page, filter.Limit, filter.SortBy, filter.SortOrder)
	if err != nil {
		util.Error(w, http.StatusInternalServerError, "Failed to retrieve login templates", err.Error())
		return
	}

	response := dto.PaginatedResponseDto[dto.LoginTemplateListResponseDto]{
		Rows:       toLoginTemplateListResponseDtoList(result.Data),
		Total:      result.Total,
		Page:       result.Page,
		Limit:      result.Limit,
		TotalPages: result.TotalPages,
	}

	util.Success(w, response, "Login templates retrieved successfully")
}

func (h *LoginTemplateHandler) Get(w http.ResponseWriter, r *http.Request) {
	loginTemplateUUIDStr := chi.URLParam(r, "login_template_uuid")

	loginTemplateUUID, err := uuid.Parse(loginTemplateUUIDStr)
	if err != nil {
		util.Error(w, http.StatusBadRequest, "Invalid login template UUID")
		return
	}

	template, err := h.loginTemplateService.GetByUUID(loginTemplateUUID)
	if err != nil {
		util.Error(w, http.StatusNotFound, "Login template not found")
		return
	}

	util.Success(w, toLoginTemplateResponseDto(*template), "Login template retrieved successfully")
}

func (h *LoginTemplateHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req dto.LoginTemplateCreateRequestDto
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

	metadata := req.Metadata
	if metadata == nil {
		metadata = make(map[string]any)
	}

	template, err := h.loginTemplateService.Create(
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

func (h *LoginTemplateHandler) Update(w http.ResponseWriter, r *http.Request) {
	loginTemplateUUIDStr := chi.URLParam(r, "login_template_uuid")

	loginTemplateUUID, err := uuid.Parse(loginTemplateUUIDStr)
	if err != nil {
		util.Error(w, http.StatusBadRequest, "Invalid login template UUID")
		return
	}

	var req dto.LoginTemplateUpdateRequestDto
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

	metadata := req.Metadata
	if metadata == nil {
		metadata = make(map[string]any)
	}

	template, err := h.loginTemplateService.Update(
		loginTemplateUUID,
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

func (h *LoginTemplateHandler) Delete(w http.ResponseWriter, r *http.Request) {
	loginTemplateUUIDStr := chi.URLParam(r, "login_template_uuid")

	loginTemplateUUID, err := uuid.Parse(loginTemplateUUIDStr)
	if err != nil {
		util.Error(w, http.StatusBadRequest, "Invalid login template UUID")
		return
	}

	template, err := h.loginTemplateService.Delete(loginTemplateUUID)
	if err != nil {
		util.Error(w, http.StatusBadRequest, "Failed to delete login template", err.Error())
		return
	}

	util.Success(w, toLoginTemplateResponseDto(*template), "Login template deleted successfully")
}

func (h *LoginTemplateHandler) UpdateStatus(w http.ResponseWriter, r *http.Request) {
	loginTemplateUUIDStr := chi.URLParam(r, "login_template_uuid")

	loginTemplateUUID, err := uuid.Parse(loginTemplateUUIDStr)
	if err != nil {
		util.Error(w, http.StatusBadRequest, "Invalid login template UUID")
		return
	}

	var req dto.LoginTemplateUpdateStatusRequestDto
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		util.Error(w, http.StatusBadRequest, "Invalid request", err.Error())
		return
	}

	if err := req.Validate(); err != nil {
		util.ValidationError(w, err)
		return
	}

	template, err := h.loginTemplateService.UpdateStatus(loginTemplateUUID, req.Status)
	if err != nil {
		util.Error(w, http.StatusBadRequest, "Failed to update login template status", err.Error())
		return
	}

	util.Success(w, toLoginTemplateResponseDto(*template), "Login template status updated successfully")
}

// Helper functions
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

func toLoginTemplateListResponseDtoList(templates []service.LoginTemplateServiceDataResult) []dto.LoginTemplateListResponseDto {
	result := make([]dto.LoginTemplateListResponseDto, len(templates))
	for i, template := range templates {
		result[i] = toLoginTemplateListResponseDto(template)
	}
	return result
}

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
