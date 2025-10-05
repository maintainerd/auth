package route

import (
	"github.com/go-chi/chi/v5"
	"github.com/maintainerd/auth/internal/handler/resthandler"
)

func LoginRoute(r chi.Router, loginHandler *resthandler.LoginHandler) {
	// Universal login (with auth_client_id and auth_container_id)
	r.Post("/login", loginHandler.Login)
}
