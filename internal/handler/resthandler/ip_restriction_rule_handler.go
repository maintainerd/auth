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

type IpRestrictionRuleHandler struct {
	ipRestrictionRuleService service.IpRestrictionRuleService
}

func NewIpRestrictionRuleHandler(ipRestrictionRuleService service.IpRestrictionRuleService) *IpRestrictionRuleHandler {
	return &IpRestrictionRuleHandler{
		ipRestrictionRuleService: ipRestrictionRuleService,
	}
}

func (h *IpRestrictionRuleHandler) GetAll(w http.ResponseWriter, r *http.Request) {
	user := r.Context().Value(middleware.UserContextKey).(*model.User)
	q := r.URL.Query()

	// Parse pagination
	page, _ := strconv.Atoi(q.Get("page"))
	limit, _ := strconv.Atoi(q.Get("limit"))

	// Parse status filter (can be multiple)
	var status []string
	if v := q.Get("status"); v != "" {
		status = append(status, v)
	}

	// Build filter DTO
	filter := dto.IpRestrictionRuleFilterDto{
		Type:        util.PtrOrNil(q.Get("type")),
		Status:      status,
		IpAddress:   util.PtrOrNil(q.Get("ip_address")),
		Description: util.PtrOrNil(q.Get("description")),
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

	result, err := h.ipRestrictionRuleService.GetAll(user.TenantID, filter.Type, filter.Status, filter.IpAddress, filter.Description, filter.Page, filter.Limit, filter.SortBy, filter.SortOrder)
	if err != nil {
		util.Error(w, http.StatusInternalServerError, "Failed to get IP restriction rules", err.Error())
		return
	}

	response := dto.PaginatedResponseDto[dto.IpRestrictionRuleResponseDto]{
		Rows:       toIpRestrictionRuleResponseDtoList(result.Data),
		Total:      result.Total,
		Page:       result.Page,
		Limit:      result.Limit,
		TotalPages: result.TotalPages,
	}

	util.Success(w, response, "IP restriction rules retrieved successfully")
}

func (h *IpRestrictionRuleHandler) Get(w http.ResponseWriter, r *http.Request) {
	user := r.Context().Value(middleware.UserContextKey).(*model.User)
	ipRestrictionRuleUUIDStr := chi.URLParam(r, "ip_restriction_rule_uuid")

	ipRestrictionRuleUUID, err := uuid.Parse(ipRestrictionRuleUUIDStr)
	if err != nil {
		util.Error(w, http.StatusBadRequest, "Invalid IP restriction rule UUID")
		return
	}

	rule, err := h.ipRestrictionRuleService.GetByUUID(user.TenantID, ipRestrictionRuleUUID)
	if err != nil {
		util.Error(w, http.StatusNotFound, "IP restriction rule not found")
		return
	}

	util.Success(w, toIpRestrictionRuleResponseDto(*rule), "IP restriction rule retrieved successfully")
}

func (h *IpRestrictionRuleHandler) Create(w http.ResponseWriter, r *http.Request) {
	user := r.Context().Value(middleware.UserContextKey).(*model.User)

	var req dto.IpRestrictionRuleCreateRequestDto
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

	rule, err := h.ipRestrictionRuleService.Create(
		user.TenantID,
		req.Description,
		req.Type,
		req.IpAddress,
		status,
		user.UserID,
	)
	if err != nil {
		util.Error(w, http.StatusBadRequest, "Failed to create IP restriction rule", err.Error())
		return
	}

	util.Created(w, toIpRestrictionRuleResponseDto(*rule), "IP restriction rule created successfully")
}

func (h *IpRestrictionRuleHandler) Update(w http.ResponseWriter, r *http.Request) {
	user := r.Context().Value(middleware.UserContextKey).(*model.User)
	ipRestrictionRuleUUIDStr := chi.URLParam(r, "ip_restriction_rule_uuid")

	ipRestrictionRuleUUID, err := uuid.Parse(ipRestrictionRuleUUIDStr)
	if err != nil {
		util.Error(w, http.StatusBadRequest, "Invalid IP restriction rule UUID")
		return
	}

	var req dto.IpRestrictionRuleUpdateRequestDto
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

	rule, err := h.ipRestrictionRuleService.Update(
		user.TenantID,
		ipRestrictionRuleUUID,
		req.Description,
		req.Type,
		req.IpAddress,
		status,
		user.UserID,
	)
	if err != nil {
		util.Error(w, http.StatusBadRequest, "Failed to update IP restriction rule", err.Error())
		return
	}

	util.Success(w, toIpRestrictionRuleResponseDto(*rule), "IP restriction rule updated successfully")
}

func (h *IpRestrictionRuleHandler) Delete(w http.ResponseWriter, r *http.Request) {
	user := r.Context().Value(middleware.UserContextKey).(*model.User)
	ipRestrictionRuleUUIDStr := chi.URLParam(r, "ip_restriction_rule_uuid")

	ipRestrictionRuleUUID, err := uuid.Parse(ipRestrictionRuleUUIDStr)
	if err != nil {
		util.Error(w, http.StatusBadRequest, "Invalid IP restriction rule UUID")
		return
	}

	rule, err := h.ipRestrictionRuleService.Delete(user.TenantID, ipRestrictionRuleUUID)
	if err != nil {
		util.Error(w, http.StatusBadRequest, "Failed to delete IP restriction rule", err.Error())
		return
	}

	util.Success(w, toIpRestrictionRuleResponseDto(*rule), "IP restriction rule deleted successfully")
}

func (h *IpRestrictionRuleHandler) UpdateStatus(w http.ResponseWriter, r *http.Request) {
	user := r.Context().Value(middleware.UserContextKey).(*model.User)
	ipRestrictionRuleUUIDStr := chi.URLParam(r, "ip_restriction_rule_uuid")

	ipRestrictionRuleUUID, err := uuid.Parse(ipRestrictionRuleUUIDStr)
	if err != nil {
		util.Error(w, http.StatusBadRequest, "Invalid IP restriction rule UUID")
		return
	}

	var req dto.IpRestrictionRuleUpdateStatusRequestDto
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		util.Error(w, http.StatusBadRequest, "Invalid request body", err.Error())
		return
	}

	if err := req.Validate(); err != nil {
		util.ValidationError(w, err)
		return
	}

	rule, err := h.ipRestrictionRuleService.UpdateStatus(user.TenantID, ipRestrictionRuleUUID, req.Status, user.UserID)
	if err != nil {
		util.Error(w, http.StatusBadRequest, "Failed to update IP restriction rule status", err.Error())
		return
	}

	util.Success(w, toIpRestrictionRuleResponseDto(*rule), "IP restriction rule status updated successfully")
}

// Helper functions
func toIpRestrictionRuleResponseDto(rule service.IpRestrictionRuleServiceDataResult) dto.IpRestrictionRuleResponseDto {
	return dto.IpRestrictionRuleResponseDto{
		IpRestrictionRuleID: rule.IpRestrictionRuleUUID.String(),
		Description:         rule.Description,
		Type:                rule.Type,
		IpAddress:           rule.IpAddress,
		Status:              rule.Status,
		CreatedAt:           rule.CreatedAt,
		UpdatedAt:           rule.UpdatedAt,
	}
}

func toIpRestrictionRuleResponseDtoList(rules []service.IpRestrictionRuleServiceDataResult) []dto.IpRestrictionRuleResponseDto {
	result := make([]dto.IpRestrictionRuleResponseDto, len(rules))
	for i, rule := range rules {
		result[i] = toIpRestrictionRuleResponseDto(rule)
	}
	return result
}
