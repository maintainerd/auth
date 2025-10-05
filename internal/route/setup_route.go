package route

import (
	"github.com/go-chi/chi/v5"
	"github.com/maintainerd/auth/internal/handler/resthandler"
)

func SetupRoute(r chi.Router, setupHandler *resthandler.SetupHandler) {
	// Setup status endpoint (always available)
	r.Get("/setup/status", setupHandler.GetSetupStatus)

	// Organization setup (one-time only)
	r.Post("/setup/create_organization", setupHandler.CreateOrganization)

	// Admin setup (one-time only, requires organization to exist)
	r.Post("/setup/create_admin", setupHandler.CreateAdmin)
}
