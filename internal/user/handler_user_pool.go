package user

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/platform/middleware"
	resp "github.com/maintainerd/auth/internal/platform/response"
)

// UserPoolHandler handles tenant-scoped user pool management operations.
//
// A user pool is the isolation boundary for users, roles, and settings within a
// single tenant deployment. All operations are tenant-isolated: the tenant is
// resolved from the request context (populated by middleware) and the service
// layer enforces ownership.
type UserPoolHandler struct {
	userPoolService UserPoolService
}

// NewUserPoolHandler creates a new user pool handler instance.
func NewUserPoolHandler(userPoolService UserPoolService) *UserPoolHandler {
	return &UserPoolHandler{userPoolService: userPoolService}
}

// GetUserPools lists all user pools for the authenticated tenant.
//
// GET /user-pools
func (h *UserPoolHandler) GetUserPools(w http.ResponseWriter, r *http.Request) {
	tenant := middleware.AuthFromRequest(r).Tenant
	if tenant == nil {
		resp.Error(w, http.StatusUnauthorized, "Tenant not found in context")
		return
	}

	pools, err := h.userPoolService.List(r.Context(), tenant.TenantID)
	if err != nil {
		resp.HandleServiceError(w, r, "Failed to fetch user pools", err)
		return
	}

	rows := make([]UserPoolResponseDTO, len(pools))
	for i, p := range pools {
		rows[i] = toUserPoolResponseDTO(p)
	}

	resp.Success(w, rows, "User pools fetched successfully")
}

// GetUserPool retrieves a single user pool by UUID.
//
// GET /user-pools/{user_pool_uuid}
func (h *UserPoolHandler) GetUserPool(w http.ResponseWriter, r *http.Request) {
	tenant := middleware.AuthFromRequest(r).Tenant
	if tenant == nil {
		resp.Error(w, http.StatusUnauthorized, "Tenant not found in context")
		return
	}

	userPoolUUID, err := uuid.Parse(chi.URLParam(r, "user_pool_uuid"))
	if err != nil {
		resp.Error(w, http.StatusBadRequest, "Invalid user pool UUID")
		return
	}

	pool, err := h.userPoolService.GetByUUID(r.Context(), userPoolUUID, tenant.TenantID)
	if err != nil {
		resp.HandleServiceError(w, r, "User pool not found", err)
		return
	}

	resp.Success(w, toUserPoolResponseDTO(pool), "User pool fetched successfully")
}

// CreateUserPool creates a new user pool for the authenticated tenant.
//
// POST /user-pools
func (h *UserPoolHandler) CreateUserPool(w http.ResponseWriter, r *http.Request) {
	tenant := middleware.AuthFromRequest(r).Tenant
	if tenant == nil {
		resp.Error(w, http.StatusUnauthorized, "Tenant not found in context")
		return
	}

	var req UserPoolCreateRequestDTO
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		resp.BadRequestBody(w)
		return
	}

	if err := req.Validate(); err != nil {
		resp.ValidationError(w, err)
		return
	}

	pool, err := h.userPoolService.Create(r.Context(), tenant.TenantID, req.Name, req.DisplayName, req.Status, req.Metadata, creatorUserID(r))
	if err != nil {
		resp.HandleServiceError(w, r, "Failed to create user pool", err)
		return
	}

	resp.Created(w, toUserPoolResponseDTO(pool), "User pool created successfully")
}

// UpdateUserPool updates an existing user pool.
//
// PUT /user-pools/{user_pool_uuid}
func (h *UserPoolHandler) UpdateUserPool(w http.ResponseWriter, r *http.Request) {
	tenant := middleware.AuthFromRequest(r).Tenant
	if tenant == nil {
		resp.Error(w, http.StatusUnauthorized, "Tenant not found in context")
		return
	}

	userPoolUUID, err := uuid.Parse(chi.URLParam(r, "user_pool_uuid"))
	if err != nil {
		resp.Error(w, http.StatusBadRequest, "Invalid user pool UUID")
		return
	}

	var req UserPoolUpdateRequestDTO
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		resp.BadRequestBody(w)
		return
	}

	if err := req.Validate(); err != nil {
		resp.ValidationError(w, err)
		return
	}

	pool, err := h.userPoolService.Update(r.Context(), userPoolUUID, tenant.TenantID, req.Name, req.DisplayName, req.Status, req.Metadata, creatorUserID(r))
	if err != nil {
		resp.HandleServiceError(w, r, "Failed to update user pool", err)
		return
	}

	resp.Success(w, toUserPoolResponseDTO(pool), "User pool updated successfully")
}

// SetStatus updates only the status of a user pool.
//
// PATCH /user-pools/{user_pool_uuid}/status
func (h *UserPoolHandler) SetStatus(w http.ResponseWriter, r *http.Request) {
	tenant := middleware.AuthFromRequest(r).Tenant
	if tenant == nil {
		resp.Error(w, http.StatusUnauthorized, "Tenant not found in context")
		return
	}

	userPoolUUID, err := uuid.Parse(chi.URLParam(r, "user_pool_uuid"))
	if err != nil {
		resp.Error(w, http.StatusBadRequest, "Invalid user pool UUID")
		return
	}

	var req UserPoolSetStatusRequestDTO
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		resp.BadRequestBody(w)
		return
	}

	if err := req.Validate(); err != nil {
		resp.ValidationError(w, err)
		return
	}

	pool, err := h.userPoolService.SetStatus(r.Context(), userPoolUUID, tenant.TenantID, req.Status, creatorUserID(r))
	if err != nil {
		resp.HandleServiceError(w, r, "Failed to update user pool status", err)
		return
	}

	resp.Success(w, toUserPoolResponseDTO(pool), "User pool status updated successfully")
}

// DeleteUserPool deletes a user pool.
//
// DELETE /user-pools/{user_pool_uuid}
func (h *UserPoolHandler) DeleteUserPool(w http.ResponseWriter, r *http.Request) {
	tenant := middleware.AuthFromRequest(r).Tenant
	if tenant == nil {
		resp.Error(w, http.StatusUnauthorized, "Tenant not found in context")
		return
	}

	userPoolUUID, err := uuid.Parse(chi.URLParam(r, "user_pool_uuid"))
	if err != nil {
		resp.Error(w, http.StatusBadRequest, "Invalid user pool UUID")
		return
	}

	pool, err := h.userPoolService.Delete(r.Context(), userPoolUUID, tenant.TenantID)
	if err != nil {
		resp.HandleServiceError(w, r, "Failed to delete user pool", err)
		return
	}

	resp.Success(w, toUserPoolResponseDTO(pool), "User pool deleted successfully")
}

// creatorUserID returns the acting user's ID from the request context for audit
// tracking, or nil when no authenticated user is present.
func creatorUserID(r *http.Request) *int64 {
	user := middleware.AuthFromRequest(r).User
	if user == nil {
		return nil
	}
	id := user.UserID
	return &id
}

func toUserPoolResponseDTO(p *UserPoolServiceDataResult) UserPoolResponseDTO {
	return UserPoolResponseDTO{
		UserPoolUUID: p.UserPoolUUID,
		Name:         p.Name,
		DisplayName:  p.DisplayName,
		Identifier:   p.Identifier,
		IsSystem:     p.IsSystem,
		Status:       p.Status,
		Metadata:     p.Metadata,
		CreatedAt:    p.CreatedAt,
		UpdatedAt:    p.UpdatedAt,
	}
}
