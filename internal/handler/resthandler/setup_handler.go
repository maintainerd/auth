package resthandler

import (
	"encoding/json"
	"net/http"

	validation "github.com/go-ozzo/ozzo-validation/v4"

	"github.com/maintainerd/auth/internal/dto"
	"github.com/maintainerd/auth/internal/service"
	"github.com/maintainerd/auth/internal/util"
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
	status, err := h.setupService.GetSetupStatus()
	if err != nil {
		util.Error(w, http.StatusInternalServerError, "Failed to get setup status", err.Error())
		return
	}

	util.Success(w, status, "Setup status retrieved successfully")
}

// CreateTenant creates the initial tenant and runs all seeders
func (h *SetupHandler) CreateTenant(w http.ResponseWriter, r *http.Request) {
	var req dto.CreateTenantRequestDto

	// Validate body payload
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		util.Error(w, http.StatusBadRequest, "Invalid request", err.Error())
		return
	}

	if err := req.Validate(); err != nil {
		if ve, ok := err.(validation.Errors); ok {
			util.Error(w, http.StatusBadRequest, "Validation failed", ve)
			return
		}
		util.Error(w, http.StatusBadRequest, "Validation failed", err.Error())
		return
	}

	// Create tenant
	response, err := h.setupService.CreateTenant(req)
	if err != nil {
		util.Error(w, http.StatusBadRequest, "Failed to create tenant", err.Error())
		return
	}

	util.Created(w, response, "Tenant created successfully")
}

// CreateAdmin creates the initial admin user
func (h *SetupHandler) CreateAdmin(w http.ResponseWriter, r *http.Request) {
	var req dto.CreateAdminRequestDto

	// Validate body payload
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		util.Error(w, http.StatusBadRequest, "Invalid request", err.Error())
		return
	}

	if err := req.Validate(); err != nil {
		if ve, ok := err.(validation.Errors); ok {
			util.Error(w, http.StatusBadRequest, "Validation failed", ve)
			return
		}
		util.Error(w, http.StatusBadRequest, "Validation failed", err.Error())
		return
	}

	// Create admin
	response, err := h.setupService.CreateAdmin(req)
	if err != nil {
		util.Error(w, http.StatusBadRequest, "Failed to create admin", err.Error())
		return
	}

	util.Created(w, response, "Admin user created successfully")
}
