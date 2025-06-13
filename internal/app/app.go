package app

import (
	"github.com/maintainerd/auth/internal/handler/grpchandler"
	"github.com/maintainerd/auth/internal/handler/resthandler"
	"github.com/maintainerd/auth/internal/repository"
	"github.com/maintainerd/auth/internal/service"
	"gorm.io/gorm"
)

type App struct {
	DB            *gorm.DB
	RoleHandler   *resthandler.RoleHandler
	AuthHandler   *resthandler.AuthHandler
	SeederHandler *grpchandler.SeederHandler
}

func NewApp(db *gorm.DB) *App {
	// repository
	roleRepo := repository.NewRoleRepository(db)
	userRepo := repository.NewUserRepository(db)

	// service
	roleService := service.NewRoleService(roleRepo)
	authService := service.NewAuthService(userRepo)

	// rest handler
	roleHandler := resthandler.NewRoleHandler(roleService)
	authHandler := resthandler.NewAuthHandler(authService)

	// grpc handler
	seederHandler := grpchandler.NewSeederHandler(authService)

	return &App{
		DB:          db,
		RoleHandler: roleHandler,
		AuthHandler: authHandler,
		// grpc handler
		SeederHandler: seederHandler,
	}
}
