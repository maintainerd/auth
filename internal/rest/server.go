package restserver

import (
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/maintainerd/auth/internal/app"
	"github.com/maintainerd/auth/internal/route"
)

func StartRESTServer(application *app.App) {
	r := chi.NewRouter()

	// Built-in Chi middlewares
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	r.Route("/api/v1", func(api chi.Router) {
		// Universal Authentication Routes (no separation needed)
		route.RegisterRoute(api, application.RegisterRestHandler)
		route.LoginRoute(api, application.LoginRestHandler)
		route.ProfileRoute(api, application.ProfileRestHandler, application.UserRepository, application.RedisClient)

		// Management Routes (all available on single server)
		route.OrganizationRoute(api, application.OrganizationRestHandler, application.UserRepository, application.RedisClient)
		route.ServiceRoute(api, application.ServiceRestHandler, application.UserRepository, application.RedisClient)
		route.APIRoute(api, application.APIRestHandler, application.UserRepository, application.RedisClient)
		route.PermissionRoute(api, application.PermissionRestHandler, application.UserRepository, application.RedisClient)
		route.AuthContainerRoute(api, application.AuthContainerRestHandler, application.UserRepository, application.RedisClient)
		route.IdentityProviderRoute(api, application.IdentityProviderRestHandler, application.UserRepository, application.RedisClient)
		route.AuthClientRoute(api, application.AuthClientRestHandler, application.UserRepository, application.RedisClient)
		route.RoleRoute(api, application.RoleRestHandler, application.UserRepository, application.RedisClient)
		route.UserRoute(api, application.UserRestHandler, application.UserRepository, application.RedisClient)
		route.InviteRoute(api, application.InviteRestHandler, application.UserRepository, application.RedisClient)
	})

	log.Println("Universal REST server running on port 8080")
	if err := http.ListenAndServe(":8080", r); err != nil {
		log.Fatal("REST server failed:", err)
	}
}
