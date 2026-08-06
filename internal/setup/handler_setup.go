package setup

import (
	"encoding/json"
	"net/http"

	"github.com/maintainerd/maintainerd-auth/internal/platform/apperror"
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

// refuseWhenOrchestrated closes the unauthenticated REST wizard on an instance
// an orchestrator provisioned, and reports whether it answered the request.
//
// The single-use credential only protects the gRPC surface. On an instance with
// the control plane on AND a credential issued, these REST endpoints are the same
// "whoever gets here first creates the system tenant and the first admin" race
// the credential exists to close — reachable by anything on the network, with no
// credential to present. Both conditions are required: with the control plane off
// there is no gRPC listener, so closing REST too would leave an instance holding
// a credential with no way to bootstrap at all.
func (h *SetupHandler) refuseWhenOrchestrated(w http.ResponseWriter, r *http.Request) bool {
	if !bootstrapControlPlaneEnabled() || !bootstrapCredentialConfigured() {
		return false
	}
	resp.HandleServiceError(w, r, "Setup is orchestrator-managed", apperror.NewForbidden(
		"this instance is provisioned by an orchestrator: bootstrap it through the gRPC SetupService with its bootstrap credential, not the REST setup wizard"))
	return true
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
	if h.refuseWhenOrchestrated(w, r) {
		return
	}

	response, err := h.setupService.CompleteSetup(r.Context())
	if err != nil {
		resp.HandleServiceError(w, r, "Failed to complete setup", err)
		return
	}

	resp.Success(w, response, "Setup completed successfully")
}

func (h *SetupHandler) RegisterControlService(w http.ResponseWriter, r *http.Request) {
	if h.refuseWhenOrchestrated(w, r) {
		return
	}

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
	if h.refuseWhenOrchestrated(w, r) {
		return
	}

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
	if h.refuseWhenOrchestrated(w, r) {
		return
	}

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

func (h *SetupHandler) CreateProfile(w http.ResponseWriter, r *http.Request) {
	if h.refuseWhenOrchestrated(w, r) {
		return
	}

	var req CreateProfileRequestDTO

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		resp.BadRequestBody(w)
		return
	}

	if err := req.Validate(); err != nil {
		resp.ValidationError(w, err)
		return
	}

	response, err := h.setupService.CreateProfile(r.Context(), req)
	if err != nil {
		resp.HandleServiceError(w, r, "Failed to create profile", err)
		return
	}

	resp.Created(w, response.Profile, "Profile created successfully")
}
