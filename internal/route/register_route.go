package route

import (
	"github.com/go-chi/chi/v5"
	"github.com/maintainerd/auth/internal/handler/resthandler"
)

func RegisterRoute(r chi.Router, registerHandler *resthandler.RegisterHandler) {
	// Public registration routes
	r.Post("/register", registerHandler.RegisterPublic)
	r.Post("/register/invite", registerHandler.RegisterPublic)
	// This route is intended for internal use only, not exposed to public clients
	// TODO: Well move this to a separate internal router with additional security measures (e.g., IP whitelisting, VPN, etc.)
	r.Post("/internal/register", registerHandler.RegisterPrivate)
	r.Post("/internal/register/invite", registerHandler.RegisterPrivateInvite)
}
