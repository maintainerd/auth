package route

import (
	"github.com/go-chi/chi/v5"
	"github.com/maintainerd/auth/internal/handler/resthandler"
)

func LoginRoute(r chi.Router, loginHandler *resthandler.LoginHandler) {
	r.Post("/login", loginHandler.LoginPublic)
	r.Post("/internal/login", loginHandler.LoginPrivate)
}
