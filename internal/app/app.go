package app

import (
	"github.com/maintainerd/auth/internal/handlers"
	"github.com/maintainerd/auth/internal/repositories"
	"github.com/maintainerd/auth/internal/services"
	"gorm.io/gorm"
)

type App struct {
	DB          *gorm.DB
	RoleHandler *handlers.RoleHandler
	AuthHandler *handlers.AuthHandler
}

func NewApp(db *gorm.DB) *App {
	// Repositories
	roleRepo := repositories.NewRoleRepository(db)
	userRepo := repositories.NewUserRepository(db)

	// Services
	roleService := services.NewRoleService(roleRepo)
	authService := services.NewAuthService(userRepo)

	// Handlers
	roleHandler := handlers.NewRoleHandler(roleService)
	authHandler := handlers.NewAuthHandler(authService)

	return &App{
		DB:          db,
		RoleHandler: roleHandler,
		AuthHandler: authHandler,
	}
}
