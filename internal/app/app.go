package app

import (
	"github.com/maintainerd/auth/internal/handler/grpchandler"
	"github.com/maintainerd/auth/internal/handler/resthandler"
	"github.com/maintainerd/auth/internal/repository"
	"github.com/maintainerd/auth/internal/service"
	"gorm.io/gorm"
)

type App struct {
	DB             *gorm.DB
	RoleHandler    *resthandler.RoleHandler
	AuthHandler    *resthandler.AuthHandler
	ProfileHandler *resthandler.ProfileHandler
	// Grpc handler
	SeederHandler *grpchandler.SeederHandler
	// Repository
	UserRepository repository.UserRepository
}

func NewApp(db *gorm.DB) *App {
	// repository
	roleRepo := repository.NewRoleRepository(db)
	userRepo := repository.NewUserRepository(db)
	authClientRepo := repository.NewAuthClientRepository(db)
	profileRepo := repository.NewProfileRepository(db)

	// service
	roleService := service.NewRoleService(roleRepo)
	authService := service.NewAuthService(userRepo, authClientRepo)
	profileService := service.NewProfileService(profileRepo)

	// rest handler
	roleHandler := resthandler.NewRoleHandler(roleService)
	authHandler := resthandler.NewAuthHandler(authService)
	profileHandler := resthandler.NewProfileHandler(profileService)

	// grpc handler
	seederHandler := grpchandler.NewSeederHandler(authService)

	return &App{
		DB:             db,
		RoleHandler:    roleHandler,
		AuthHandler:    authHandler,
		ProfileHandler: profileHandler,
		// grpc handler
		SeederHandler: seederHandler,
		// Repository
		UserRepository: userRepo,
	}
}
