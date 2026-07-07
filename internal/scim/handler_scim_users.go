package scim

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	resp "github.com/maintainerd/maintainerd-auth/internal/platform/response"
)

type SCIMUserHandler struct {
	svc SCIMUserService
}

func NewSCIMUserHandler(svc SCIMUserService) *SCIMUserHandler {
	return &SCIMUserHandler{svc: svc}
}

func (h *SCIMUserHandler) List(w http.ResponseWriter, r *http.Request) {
	tenantID := tenantFromRequest(r)
	if tenantID == 0 {
		writeSCIMError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	startIndex := 1
	if si := r.URL.Query().Get("startIndex"); si != "" {
		if v, err := strconv.Atoi(si); err == nil && v > 0 {
			startIndex = v
		}
	}
	count := 20
	if c := r.URL.Query().Get("count"); c != "" {
		if v, err := strconv.Atoi(c); err == nil && v > 0 {
			count = v
		}
	}
	filter := r.URL.Query().Get("filter")

	result, err := h.svc.ListUsers(r.Context(), tenantID, startIndex, count, filter)
	if err != nil {
		resp.HandleServiceError(w, r, "Failed to list SCIM users", err)
		return
	}

	w.Header().Set("Content-Type", "application/scim+json")
	_ = json.NewEncoder(w).Encode(result)
}

func (h *SCIMUserHandler) Get(w http.ResponseWriter, r *http.Request) {
	tenantID := tenantFromRequest(r)
	if tenantID == 0 {
		writeSCIMError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	userID := chi.URLParam(r, "userID")
	result, err := h.svc.GetUser(r.Context(), userID, tenantID)
	if err != nil {
		resp.HandleServiceError(w, r, "Failed to get SCIM user", err)
		return
	}

	w.Header().Set("Content-Type", "application/scim+json")
	_ = json.NewEncoder(w).Encode(result)
}

func (h *SCIMUserHandler) Create(w http.ResponseWriter, r *http.Request) {
	tenantID := tenantFromRequest(r)
	if tenantID == 0 {
		writeSCIMError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	var req SCIMUserCreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeSCIMError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	result, err := h.svc.CreateUser(r.Context(), tenantID, &req)
	if err != nil {
		resp.HandleServiceError(w, r, "Failed to create SCIM user", err)
		return
	}

	w.Header().Set("Content-Type", "application/scim+json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(result)
}

func (h *SCIMUserHandler) Update(w http.ResponseWriter, r *http.Request) {
	tenantID := tenantFromRequest(r)
	if tenantID == 0 {
		writeSCIMError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	userID := chi.URLParam(r, "userID")
	var req SCIMUserUpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeSCIMError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	result, err := h.svc.UpdateUser(r.Context(), userID, tenantID, &req)
	if err != nil {
		resp.HandleServiceError(w, r, "Failed to update SCIM user", err)
		return
	}

	w.Header().Set("Content-Type", "application/scim+json")
	_ = json.NewEncoder(w).Encode(result)
}

func (h *SCIMUserHandler) Patch(w http.ResponseWriter, r *http.Request) {
	tenantID := tenantFromRequest(r)
	if tenantID == 0 {
		writeSCIMError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	userID := chi.URLParam(r, "userID")
	var req SCIMPatchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeSCIMError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	result, err := h.svc.PatchUser(r.Context(), userID, tenantID, &req)
	if err != nil {
		resp.HandleServiceError(w, r, "Failed to patch SCIM user", err)
		return
	}

	w.Header().Set("Content-Type", "application/scim+json")
	_ = json.NewEncoder(w).Encode(result)
}

func (h *SCIMUserHandler) Delete(w http.ResponseWriter, r *http.Request) {
	tenantID := tenantFromRequest(r)
	if tenantID == 0 {
		writeSCIMError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	userID := chi.URLParam(r, "userID")
	if err := h.svc.DeleteUser(r.Context(), userID, tenantID); err != nil {
		resp.HandleServiceError(w, r, "Failed to delete SCIM user", err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func tenantFromRequest(r *http.Request) int64 {
	tenantID, ok := r.Context().Value(scimTenantIDKey{}).(int64)
	if !ok {
		return 0
	}
	return tenantID
}
