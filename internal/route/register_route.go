package route

import (
	"github.com/go-chi/chi/v5"
	"github.com/maintainerd/auth/internal/handler/resthandler"
)

func RegisterRoute(r chi.Router, registerHandler *resthandler.RegisterHandler) {
	// Universal registration (with auth_client_id and auth_container_id)
	r.Post("/register", registerHandler.Register)

	// Universal registration with invite
	r.Post("/register/invite", registerHandler.RegisterInvite)
}
