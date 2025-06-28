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
}

func NewApp(db *gorm.DB) *App {
	// Repositories
	roleRepo := repositories.NewRoleRepository(db)

	// Services
	roleService := services.NewRoleService(roleRepo)

	// Handlers
	roleHandler := handlers.NewRoleHandler(roleService)

	return &App{
		DB:          db,
		RoleHandler: roleHandler,
	}
}
