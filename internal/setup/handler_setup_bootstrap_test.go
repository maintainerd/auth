package setup

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// restSetupMutations is every REST endpoint that can take ownership of a fresh
// instance. GetSetupStatus is deliberately absent: it reads booleans and creates
// nothing.
func restSetupMutations(h *SetupHandler) map[string]func(http.ResponseWriter, *http.Request) {
	return map[string]func(http.ResponseWriter, *http.Request){
		"create_tenant":            h.CreateTenant,
		"create_admin":             h.CreateAdmin,
		"create_profile":           h.CreateProfile,
		"register-control-service": h.RegisterControlService,
		"complete":                 h.CompleteSetup,
	}
}

// The credential closes the gRPC door; this closes the unauthenticated REST one
// on the same instance. Without it, core's single-use credential is beside the
// point — anyone who can reach port 8081 first still creates the system tenant
// and the first admin.
func TestSetupHandler_RefusesTheRESTWizardOnAnOrchestratedInstance(t *testing.T) {
	withControlPlaneEnabled(t, true)
	withConfiguredCredential(t, "issued-by-core")

	served := false
	h := NewSetupHandler(&mockSetupService{
		createTenantFn: func(CreateTenantRequestDTO) (*CreateTenantResponseDTO, error) {
			served = true
			return &CreateTenantResponseDTO{}, nil
		},
		completeSetupFn: func() (*CompleteSetupResponseDTO, error) {
			served = true
			return &CompleteSetupResponseDTO{}, nil
		},
	})

	for name, handle := range restSetupMutations(h) {
		t.Run(name, func(t *testing.T) {
			w := httptest.NewRecorder()
			handle(w, setupRequest(t, map[string]any{"name": "core", "display_name": "Core"}))

			assert.Equal(t, http.StatusForbidden, w.Code)
			assert.Contains(t, w.Body.String(), "orchestrator")
		})
	}
	assert.False(t, served, "no orchestrated REST setup call may reach the service")
}

// Standalone is the default deployment and must be untouched: no credential
// means the wizard is the only way in.
func TestSetupHandler_KeepsTheRESTWizardOpenForStandalone(t *testing.T) {
	withControlPlaneEnabled(t, false)
	withConfiguredCredential(t, "")

	h := NewSetupHandler(&mockSetupService{})
	w := httptest.NewRecorder()
	h.CreateTenant(w, setupRequest(t, map[string]any{"name": "acme", "display_name": "Acme"}))

	require.NotEqual(t, http.StatusForbidden, w.Code, "a standalone deployment must still be able to bootstrap")
}

// A credential with no control plane means there is no gRPC listener at all.
// Closing REST as well would leave the instance with no bootstrap path — a
// bricked deployment, not a safer one.
func TestSetupHandler_KeepsTheRESTWizardOpenWhenThereIsNoControlPlane(t *testing.T) {
	withControlPlaneEnabled(t, false)
	withConfiguredCredential(t, "issued-by-core")

	h := NewSetupHandler(&mockSetupService{})
	w := httptest.NewRecorder()
	h.CreateTenant(w, setupRequest(t, map[string]any{"name": "acme", "display_name": "Acme"}))

	require.NotEqual(t, http.StatusForbidden, w.Code)
}

// The control plane can be on without a credential (gRPC setup disabled); the
// wizard remains the bootstrap path.
func TestSetupHandler_KeepsTheRESTWizardOpenWithoutACredential(t *testing.T) {
	withControlPlaneEnabled(t, true)
	withConfiguredCredential(t, "")

	h := NewSetupHandler(&mockSetupService{})
	w := httptest.NewRecorder()
	h.CreateTenant(w, setupRequest(t, map[string]any{"name": "acme", "display_name": "Acme"}))

	require.NotEqual(t, http.StatusForbidden, w.Code)
}

func TestSetupHandler_StatusStaysReadableOnAnOrchestratedInstance(t *testing.T) {
	withControlPlaneEnabled(t, true)
	withConfiguredCredential(t, "issued-by-core")

	h := NewSetupHandler(&mockSetupService{})
	w := httptest.NewRecorder()
	h.GetSetupStatus(w, httptest.NewRequest(http.MethodGet, "/setup/status", nil))

	assert.Equal(t, http.StatusOK, w.Code, "core polls status before and after provisioning")
}
