package route

import (
	"github.com/go-chi/chi/v5"
	"github.com/maintainerd/auth/internal/handler/resthandler"
)

func RegisterAuthRoute(r chi.Router, authHandler *resthandler.AuthHandler) {
	r.Post("/register", authHandler.Register)
	r.Post("/login", authHandler.Login)
}
