package setup

import (
	"encoding/json"
	"net/http"

	resp "github.com/maintainerd/maintainerd-auth/internal/platform/response"
)

type SetupHandler struct {
	setupService SetupService
}

func NewSetupHandler(setupService SetupService) *SetupHandler {
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

func (h *SetupHandler) CompleteSetup(w http.ResponseWriter, r *http.Request) {
	response, err := h.setupService.CompleteSetup(r.Context())
	if err != nil {
		resp.HandleServiceError(w, r, "Failed to complete setup", err)
		return
	}

	resp.Success(w, response, "Setup completed successfully")
}

func (h *SetupHandler) RegisterControlService(w http.ResponseWriter, r *http.Request) {
	var req RegisterControlServiceRequestDTO

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		resp.BadRequestBody(w)
		return
	}

	if err := req.Validate(); err != nil {
		resp.ValidationError(w, err)
		return
	}

	response, err := h.setupService.RegisterControlService(r.Context(), req)
	if err != nil {
		resp.HandleServiceError(w, r, "Failed to register control service", err)
		return
	}

	resp.Created(w, response, "Control service registered successfully")
}

// CreateTenant creates the initial tenant and runs all seeders
func (h *SetupHandler) CreateTenant(w http.ResponseWriter, r *http.Request) {
	var req CreateTenantRequestDTO

	// Validate body payload
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		resp.BadRequestBody(w)
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
	var req CreateAdminRequestDTO

	// Validate body payload
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		resp.BadRequestBody(w)
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
