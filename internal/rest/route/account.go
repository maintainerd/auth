package route

import (
	"github.com/go-chi/chi/v5"
	"github.com/maintainerd/auth/internal/cache"
	"github.com/maintainerd/auth/internal/middleware"
	"github.com/maintainerd/auth/internal/rest/handler"
	"github.com/maintainerd/auth/internal/service"
)

// AccountRoute mounts authenticated self-service account management endpoints.
func AccountRoute(
	r chi.Router,
	accountHandler *handler.AccountHandler,
	userService service.UserService,
	appCache *cache.Cache,
) {
	r.Route("/account", func(r chi.Router) {
		r.Use(middleware.JWTAuthMiddleware)
		r.Use(middleware.UserContextMiddleware(userService, appCache))

		// Email change flow
		r.Post("/email/change", accountHandler.InitiateEmailChange)
		r.Post("/email/verify", accountHandler.VerifyEmailChange)

		// Username change
		r.Put("/username", accountHandler.ChangeUsername)

		// Account deletion
		r.Delete("/", accountHandler.DeleteAccount)

		// Account data export (GDPR / data portability)
		r.Get("/export", accountHandler.ExportAccountData)

		// Backup codes — generate and store securely
		r.Post("/backup-codes", accountHandler.GenerateBackupCodes)
	})
}

// RecoveryRoute mounts unauthenticated account recovery endpoints.
func RecoveryRoute(
	r chi.Router,
	accountHandler *handler.AccountHandler,
) {
	r.Route("/recovery", func(r chi.Router) {
		// Backup-code based account recovery (issues tokens)
		r.Post("/backup-code", accountHandler.VerifyBackupCode)
	})
}
