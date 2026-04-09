package handler

import (
	"encoding/json"
	"net/http"

	"github.com/maintainerd/auth/internal/dto"
	resp "github.com/maintainerd/auth/internal/rest/response"
	"github.com/maintainerd/auth/internal/service"
)

type SetupHandler struct {
	setupService service.SetupService
}

func NewSetupHandler(setupService service.SetupService) *SetupHandler {
	return &SetupHandler{
		setupService: setupService,
	}
}

// GetSetupStatus checks the current setup status
func (h *SetupHandler) GetSetupStatus(w http.ResponseWriter, r *http.Request) {
	status, err := h.setupService.GetSetupStatus(r.Context())
	if err != nil {
		resp.HandleServiceError(w, r, "Failed to get setup status", err)
		return
	}

	resp.Success(w, status, "Setup status retrieved successfully")
}

// CreateTenant creates the initial tenant and runs all seeders
func (h *SetupHandler) CreateTenant(w http.ResponseWriter, r *http.Request) {
	var req dto.CreateTenantRequestDTO

	// Validate body payload
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		resp.Error(w, http.StatusBadRequest, "Invalid request")
		return
	}

	if err := req.Validate(); err != nil {
		resp.ValidationError(w, err)
		return
	}

	// Create tenant
	response, err := h.setupService.CreateTenant(r.Context(), req)
	if err != nil {
		resp.HandleServiceError(w, r, "Failed to create tenant", err)
		return
	}

	resp.Created(w, response.Tenant, "Tenant created successfully")
}

// CreateAdmin creates the initial admin user
func (h *SetupHandler) CreateAdmin(w http.ResponseWriter, r *http.Request) {
	var req dto.CreateAdminRequestDTO

	// Validate body payload
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		resp.Error(w, http.StatusBadRequest, "Invalid request")
		return
	}

	if err := req.Validate(); err != nil {
		resp.ValidationError(w, err)
		return
	}

	// Create admin
	response, err := h.setupService.CreateAdmin(r.Context(), req)
	if err != nil {
		resp.HandleServiceError(w, r, "Failed to create admin", err)
		return
	}

	resp.Created(w, response.User, "Admin user created successfully")
}

// CreateProfile creates the initial profile for the admin user
func (h *SetupHandler) CreateProfile(w http.ResponseWriter, r *http.Request) {
	var req dto.CreateProfileRequestDTO

	// Validate body payload
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		resp.Error(w, http.StatusBadRequest, "Invalid request")
		return
	}

	if err := req.Validate(); err != nil {
		resp.ValidationError(w, err)
		return
	}

	// Create profile
	response, err := h.setupService.CreateProfile(r.Context(), req)
	if err != nil {
		resp.HandleServiceError(w, r, "Failed to create profile", err)
		return
	}

	resp.Created(w, response.Profile, "Profile created successfully")
}
