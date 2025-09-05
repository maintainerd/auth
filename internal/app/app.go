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
	DB          *gorm.DB
	RedisClient *redis.Client
	// Rest handler
	ServiceRestHandler  *resthandler.ServiceHandler
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
	// Repositories
	authClientRepo := repository.NewAuthClientRepository(db)
	userRepo := repository.NewUserRepository(db)
	userRoleRepo := repository.NewUserRoleRepository(db)
	userTokenRepo := repository.NewUserTokenRepository(db)
	serviceRepo := repository.NewServiceRepository(db)
	roleRepo := repository.NewRoleRepository(db)
	profileRepo := repository.NewProfileRepository(db)
	inviteRepo := repository.NewInviteRepository(db)
	emailTemplateRepo := repository.NewEmailTemplateRepository(db)

	// Services
	serviceService := service.NewServiceService(db, serviceRepo)
	roleService := service.NewRoleService(db, roleRepo)
	registerService := service.NewRegistrationService(db, authClientRepo, userRepo, userRoleRepo, userTokenRepo, roleRepo, inviteRepo)
	loginService := service.NewLoginService(db, authClientRepo, userRepo, userTokenRepo)
	profileService := service.NewProfileService(db, profileRepo)
	inviteService := service.NewInviteService(db, inviteRepo, authClientRepo, roleRepo, emailTemplateRepo)

	// Rest handlers
	serviceRestHandler := resthandler.NewServiceHandler(serviceService)
	roleRestHandler := resthandler.NewRoleHandler(roleService)
	registerRestHandler := resthandler.NewRegisterHandler(registerService)
	loginRestHandler := resthandler.NewLoginHandler(loginService)
	profileRestHandler := resthandler.NewProfileHandler(profileService)
	inviteRestHandler := resthandler.NewInviteHandler(inviteService)

	// GRPC handlers
	seederGrpcHandler := grpchandler.NewSeederHandler(registerService)

	return &App{
		DB:          db,
		RedisClient: redisClient,
		// Rest handler
		ServiceRestHandler:  serviceRestHandler,
		RoleRestHandler:     roleRestHandler,
		RegisterRestHandler: registerRestHandler,
		LoginRestHandler:    loginRestHandler,
		ProfileRestHandler:  profileRestHandler,
		InviteRestHandler:   inviteRestHandler,
		// GRPC handler
		SeederHandler: seederGrpcHandler,
		// Repository
		UserRepository: userRepo,
	}
}
