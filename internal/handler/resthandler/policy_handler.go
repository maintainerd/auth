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

type PolicyHandler struct {
	policyService service.PolicyService
}

func NewPolicyHandler(policyService service.PolicyService) *PolicyHandler {
	return &PolicyHandler{
		policyService: policyService,
	}
}

// Get policies with filtering and pagination
func (h *PolicyHandler) Get(w http.ResponseWriter, r *http.Request) {
	// Parse query parameters
	query := r.URL.Query()

	// Parse filters
	var filter dto.PolicyFilterDto
	if name := query.Get("name"); name != "" {
		filter.Name = &name
	}
	if description := query.Get("description"); description != "" {
		filter.Description = &description
	}
	if version := query.Get("version"); version != "" {
		filter.Version = &version
	}
	// Handle status filtering - support both comma-separated and multiple parameters
	var statusValues []string
	if singleStatus := query.Get("status"); singleStatus != "" {
		// Handle comma-separated values like services: ?status=active,inactive
		statusValues = strings.Split(strings.ReplaceAll(singleStatus, " ", ""), ",")
	} else if multipleStatus := query["status"]; len(multipleStatus) > 0 {
		// Handle multiple parameters: ?status=active&status=inactive
		statusValues = multipleStatus
	}

	// Filter out empty status values
	var validStatus []string
	for _, s := range statusValues {
		if s != "" {
			validStatus = append(validStatus, s)
		}
	}
	if len(validStatus) > 0 {
		filter.Status = validStatus
	}
	if isDefault := query.Get("is_default"); isDefault != "" {
		if val, err := strconv.ParseBool(isDefault); err == nil {
			filter.IsDefault = &val
		}
	}
	if isSystem := query.Get("is_system"); isSystem != "" {
		if val, err := strconv.ParseBool(isSystem); err == nil {
			filter.IsSystem = &val
		}
	}
	if serviceID := query.Get("service_id"); serviceID != "" {
		// Only parse if it's a valid UUID format
		if val, err := uuid.Parse(serviceID); err == nil {
			filter.ServiceID = &val
		}
		// If parsing fails, we ignore the service_id filter (don't set filter.ServiceID)
	}

	// Parse pagination
	page, _ := strconv.Atoi(query.Get("page"))
	if page < 1 {
		page = 1
	}
	limit, _ := strconv.Atoi(query.Get("limit"))
	if limit < 1 || limit > 100 {
		limit = 10
	}

	filter.Page = page
	filter.Limit = limit
	filter.SortBy = query.Get("sort_by")
	filter.SortOrder = query.Get("sort_order")

	// Validate filter parameters
	if err := filter.Validate(); err != nil {
		util.ValidationError(w, err)
		return
	}

	// Convert to service filter
	serviceFilter := service.PolicyServiceGetFilter{
		Name:        filter.Name,
		Description: filter.Description,
		Version:     filter.Version,
		Status:      filter.Status,
		IsDefault:   filter.IsDefault,
		IsSystem:    filter.IsSystem,
		ServiceID:   filter.ServiceID,
		Page:        filter.Page,
		Limit:       filter.Limit,
		SortBy:      filter.SortBy,
		SortOrder:   filter.SortOrder,
	}

	result, err := h.policyService.Get(serviceFilter)
	if err != nil {
		util.Error(w, http.StatusInternalServerError, "Failed to get policies", err.Error())
		return
	}

	// Convert to response DTOs
	rows := make([]dto.PolicyResponseDto, len(result.Data))
	for i, policy := range result.Data {
		rows[i] = toPolicyResponseDto(policy)
	}

	response := dto.PaginatedResponseDto[dto.PolicyResponseDto]{
		Rows:       rows,
		Total:      result.Total,
		Page:       result.Page,
		Limit:      result.Limit,
		TotalPages: result.TotalPages,
	}

	util.Success(w, response, "Policies retrieved successfully")
}

// Get policy by UUID
func (h *PolicyHandler) GetByUUID(w http.ResponseWriter, r *http.Request) {
	policyUUIDStr := chi.URLParam(r, "policy_uuid")
	policyUUID, err := uuid.Parse(policyUUIDStr)
	if err != nil {
		util.Error(w, http.StatusBadRequest, "Invalid policy UUID", err.Error())
		return
	}

	policy, err := h.policyService.GetByUUID(policyUUID)
	if err != nil {
		util.Error(w, http.StatusNotFound, "Policy not found", err.Error())
		return
	}

	dtoRes := toPolicyDetailResponseDto(*policy)
	util.Success(w, dtoRes, "Policy retrieved successfully")
}

// Create policy
func (h *PolicyHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req dto.PolicyCreateRequestDto
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		util.Error(w, http.StatusBadRequest, "Invalid request", err.Error())
		return
	}

	if err := req.Validate(); err != nil {
		util.ValidationError(w, err)
		return
	}

	policy, err := h.policyService.Create(
		req.Name,
		req.Description,
		req.Document,
		req.Version,
		req.Status,
		false, // isDefault - only set by seeders
		false, // isSystem - only set by seeders
	)
	if err != nil {
		util.Error(w, http.StatusInternalServerError, "Failed to create policy", err.Error())
		return
	}

	dtoRes := toPolicyDetailResponseDto(*policy)
	util.Created(w, dtoRes, "Policy created successfully")
}

// Update policy
func (h *PolicyHandler) Update(w http.ResponseWriter, r *http.Request) {
	policyUUIDStr := chi.URLParam(r, "policy_uuid")
	policyUUID, err := uuid.Parse(policyUUIDStr)
	if err != nil {
		util.Error(w, http.StatusBadRequest, "Invalid policy UUID", err.Error())
		return
	}

	var req dto.PolicyUpdateRequestDto
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		util.Error(w, http.StatusBadRequest, "Invalid request", err.Error())
		return
	}

	if err := req.Validate(); err != nil {
		util.ValidationError(w, err)
		return
	}

	policy, err := h.policyService.Update(
		policyUUID,
		req.Name,
		req.Description,
		req.Document,
		req.Version,
		req.Status,
	)
	if err != nil {
		util.Error(w, http.StatusInternalServerError, "Failed to update policy", err.Error())
		return
	}

	dtoRes := toPolicyDetailResponseDto(*policy)
	util.Success(w, dtoRes, "Policy updated successfully")
}

// Update policy status
func (h *PolicyHandler) UpdateStatus(w http.ResponseWriter, r *http.Request) {
	policyUUIDStr := chi.URLParam(r, "policy_uuid")
	policyUUID, err := uuid.Parse(policyUUIDStr)
	if err != nil {
		util.Error(w, http.StatusBadRequest, "Invalid policy UUID", err.Error())
		return
	}

	var req dto.PolicyStatusUpdateDto
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		util.Error(w, http.StatusBadRequest, "Invalid request", err.Error())
		return
	}

	if err := req.Validate(); err != nil {
		util.ValidationError(w, err)
		return
	}

	policy, err := h.policyService.SetStatusByUUID(policyUUID, req.Status)
	if err != nil {
		util.Error(w, http.StatusInternalServerError, "Failed to update policy status", err.Error())
		return
	}

	dtoRes := toPolicyDetailResponseDto(*policy)
	util.Success(w, dtoRes, "Policy status updated successfully")
}

// Delete policy
func (h *PolicyHandler) Delete(w http.ResponseWriter, r *http.Request) {
	policyUUIDStr := chi.URLParam(r, "policy_uuid")
	policyUUID, err := uuid.Parse(policyUUIDStr)
	if err != nil {
		util.Error(w, http.StatusBadRequest, "Invalid policy UUID", err.Error())
		return
	}

	policy, err := h.policyService.DeleteByUUID(policyUUID)
	if err != nil {
		util.Error(w, http.StatusInternalServerError, "Failed to delete policy", err.Error())
		return
	}

	dtoRes := toPolicyDetailResponseDto(*policy)
	util.Success(w, dtoRes, "Policy deleted successfully")
}

// Helper function to convert service result to DTO (for listing - without document)
func toPolicyResponseDto(policy service.PolicyServiceDataResult) dto.PolicyResponseDto {
	return dto.PolicyResponseDto{
		PolicyUUID:  policy.PolicyUUID,
		Name:        policy.Name,
		Description: policy.Description,
		Version:     policy.Version,
		Status:      policy.Status,
		IsDefault:   policy.IsDefault,
		IsSystem:    policy.IsSystem,
		CreatedAt:   policy.CreatedAt,
		UpdatedAt:   policy.UpdatedAt,
	}
}

// Helper function to convert service result to detail DTO (for individual retrieval - with document)
func toPolicyDetailResponseDto(policy service.PolicyServiceDataResult) dto.PolicyDetailResponseDto {
	return dto.PolicyDetailResponseDto{
		PolicyUUID:  policy.PolicyUUID,
		Name:        policy.Name,
		Description: policy.Description,
		Document:    policy.Document,
		Version:     policy.Version,
		Status:      policy.Status,
		IsDefault:   policy.IsDefault,
		IsSystem:    policy.IsSystem,
		CreatedAt:   policy.CreatedAt,
		UpdatedAt:   policy.UpdatedAt,
	}
}
