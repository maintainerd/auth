package authn

import (
	"net/http"

	resp "github.com/maintainerd/maintainerd-auth/internal/platform/response"
)

// RegistrationContextHandler serves the public signup-requirements read.
type RegistrationContextHandler struct {
	service RegistrationContextService
}

func NewRegistrationContextHandler(service RegistrationContextService) *RegistrationContextHandler {
	return &RegistrationContextHandler{service: service}
}

// GetPublic returns what the hosted signup form must collect.
//
// GET /registration_context?client_id=<client identifier>[&registration_flow=<flow name>]
//
// Public and unauthenticated. It exposes only the effective required-field set
// and the effective email-verification requirement — never the flow's roles,
// description, status, is_system, ids or timestamps. Roles in particular are
// withheld deliberately: the flow name is guessable by design, so publishing
// which roles a flow grants would hand an attacker a ranked target list.
//
// Both values are already derivable by POSTing to /register and reading the 400,
// so returning them discloses nothing new — it just removes a guess-and-fail
// round trip that the UI cannot otherwise avoid.
func (h *RegistrationContextHandler) GetPublic(w http.ResponseWriter, r *http.Request) {
	q := RegistrationContextQueryDTO{
		ClientID:         r.URL.Query().Get("client_id"),
		TenantID:         r.URL.Query().Get("tenant_id"),
		RegistrationFlow: r.URL.Query().Get("registration_flow"),
	}

	// Same surface contract as every other public auth route: client_id is
	// required and tenant_id is rejected.
	clientIDPtr, tenantIDPtr, ok := authenticationContextQuery(r)
	if !ok {
		resp.Error(w, http.StatusBadRequest, "Public registration context requires client_id and does not accept tenant_id")
		return
	}
	if err := q.Validate(); err != nil {
		resp.ValidationError(w, err)
		return
	}

	result, err := h.service.Get(r.Context(), clientIDPtr, tenantIDPtr, q.RegistrationFlow)
	if err != nil {
		resp.HandleServiceError(w, r, "Failed to load registration context", err)
		return
	}

	// Never cache: a flow's status is the operator's kill switch for a published
	// link, so a cached copy is exactly the window in which a revoked link keeps
	// looking valid.
	w.Header().Set("Cache-Control", "no-store")

	resp.Success(w, toRegistrationContextResponseDTO(*result), "Registration context retrieved")
}

func toRegistrationContextResponseDTO(r RegistrationContextResult) RegistrationContextResponseDTO {
	fields := r.RequiredFields
	if fields == nil {
		fields = []string{}
	}
	return RegistrationContextResponseDTO{
		RegistrationFlow:     r.RegistrationFlow,
		RequiredFields:       fields,
		VerificationRequired: r.VerificationRequired,
	}
}
