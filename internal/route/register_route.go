package route

import (
	"github.com/go-chi/chi/v5"
	"github.com/maintainerd/auth/internal/handler/resthandler"
)

func RegisterRoute(r chi.Router, registerHandler *resthandler.RegisterHandler) {
	r.Post("/register", registerHandler.RegisterPublic)
	r.Post("/register/invite", registerHandler.RegisterPublic)
}

func RegisterInternalRoute(r chi.Router, registerHandler *resthandler.RegisterHandler) {
	r.Post("/register", registerHandler.RegisterPrivate)
	r.Post("/register/invite", registerHandler.RegisterPrivateInvite)
}
