package resthandler

import (
	"encoding/json"
	"net/http"
	"strings"

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

	util.Created(w, response.Tenant, "Tenant created successfully")
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

	util.Created(w, response.User, "Admin user created successfully")
}

// CreateProfile creates the initial profile for the admin user
func (h *SetupHandler) CreateProfile(w http.ResponseWriter, r *http.Request) {
	var req dto.CreateProfileRequestDto

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

	// Create profile
	response, err := h.setupService.CreateProfile(req)
	if err != nil {
		// Check if it's a profile already exists error
		if strings.Contains(err.Error(), "profile already exists") {
			util.Error(w, http.StatusBadRequest, "Profile already exists", "A profile has already been created for the admin user")
			return
		}
		util.Error(w, http.StatusBadRequest, "Failed to create profile", err.Error())
		return
	}

	util.Created(w, response.Profile, "Profile created successfully")
}
