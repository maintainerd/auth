package app

import (
	"github.com/maintainerd/auth/internal/handler"
	"github.com/maintainerd/auth/internal/repository"
	"github.com/maintainerd/auth/internal/service"
	"gorm.io/gorm"
)

type App struct {
	DB          *gorm.DB
	RoleHandler *handler.RoleHandler
	AuthHandler *handler.AuthHandler
}

func NewApp(db *gorm.DB) *App {
	// repository
	roleRepo := repository.NewRoleRepository(db)
	userRepo := repository.NewUserRepository(db)

	// service
	roleService := service.NewRoleService(roleRepo)
	authService := service.NewAuthService(userRepo)

	// handler
	roleHandler := handler.NewRoleHandler(roleService)
	authHandler := handler.NewAuthHandler(authService)

	return &App{
		DB:          db,
		RoleHandler: roleHandler,
		AuthHandler: authHandler,
	}
}
