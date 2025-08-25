package app

import (
	"github.com/maintainerd/auth/internal/handler/grpchandler"
	"github.com/maintainerd/auth/internal/handler/resthandler"
	"github.com/maintainerd/auth/internal/repository"
	"github.com/maintainerd/auth/internal/service"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

type App struct {
	DB                  *gorm.DB
	RedisClient         *redis.Client
	RoleRestHandler     *resthandler.RoleHandler
	RegisterRestHandler *resthandler.RegisterHandler
	LoginRestHandler    *resthandler.LoginHandler
	ProfileRestHandler  *resthandler.ProfileHandler
	InviteRestHandler   *resthandler.InviteHandler
	// Grpc handler
	SeederHandler *grpchandler.SeederHandler
	// Repository
	UserRepository repository.UserRepository
}

func NewApp(db *gorm.DB, redisClient *redis.Client) *App {
	// repository
	authClientRepo := repository.NewAuthClientRepository(db)
	userRepo := repository.NewUserRepository(db)
	userRoleRepo := repository.NewUserRoleRepository(db)
	userTokenRepo := repository.NewUserTokenRepository(db)
	roleRepo := repository.NewRoleRepository(db)
	profileRepo := repository.NewProfileRepository(db)
	inviteRepo := repository.NewInviteRepository(db)
	emailTemplateRepo := repository.NewEmailTemplateRepository(db)

	// service
	roleService := service.NewRoleService(roleRepo)
	registerService := service.NewRegistrationService(db, authClientRepo, userRepo, userRoleRepo, userTokenRepo, roleRepo, inviteRepo)
	loginService := service.NewLoginService(db, authClientRepo, userRepo, userTokenRepo)
	profileService := service.NewProfileService(db, profileRepo)
	inviteService := service.NewInviteService(db, inviteRepo, authClientRepo, roleRepo, emailTemplateRepo)

	// rest handler
	roleRestHandler := resthandler.NewRoleHandler(roleService)
	registerRestHandler := resthandler.NewRegisterHandler(registerService)
	loginRestHandler := resthandler.NewLoginHandler(loginService)
	profileRestHandler := resthandler.NewProfileHandler(profileService)
	inviteRestHandler := resthandler.NewInviteHandler(inviteService)

	// grpc handler
	seederGrpcHandler := grpchandler.NewSeederHandler(registerService)

	return &App{
		DB:                  db,
		RedisClient:         redisClient,
		RoleRestHandler:     roleRestHandler,
		RegisterRestHandler: registerRestHandler,
		LoginRestHandler:    loginRestHandler,
		ProfileRestHandler:  profileRestHandler,
		InviteRestHandler:   inviteRestHandler,
		// grpc handler
		SeederHandler: seederGrpcHandler,
		// Repository
		UserRepository: userRepo,
	}
}
