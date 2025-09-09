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
	OrganizationRestHandler  *resthandler.OrganizationHandler
	ServiceRestHandler       *resthandler.ServiceHandler
	APIRestHandler           *resthandler.APIHandler
	AuthContainerRestHandler *resthandler.AuthContainerHandler
	RoleRestHandler          *resthandler.RoleHandler
	RegisterRestHandler      *resthandler.RegisterHandler
	LoginRestHandler         *resthandler.LoginHandler
	ProfileRestHandler       *resthandler.ProfileHandler
	InviteRestHandler        *resthandler.InviteHandler
	// Grpc handler
	SeederHandler *grpchandler.SeederHandler
	// Repository
	UserRepository repository.UserRepository
}

func NewApp(db *gorm.DB, redisClient *redis.Client) *App {
	// Repositories
	organizationRepo := repository.NewOrganizationRepository(db)
	organizationServiceRepo := repository.NewOrganizationServiceRepository(db)
	serviceRepo := repository.NewServiceRepository(db)
	apiRepo := repository.NewAPIRepository(db)
	authContainerRepo := repository.NewAuthContainerRepository(db)
	roleRepo := repository.NewRoleRepository(db)
	authClientRepo := repository.NewAuthClientRepository(db)
	userRepo := repository.NewUserRepository(db)
	userRoleRepo := repository.NewUserRoleRepository(db)
	userTokenRepo := repository.NewUserTokenRepository(db)
	profileRepo := repository.NewProfileRepository(db)
	inviteRepo := repository.NewInviteRepository(db)
	emailTemplateRepo := repository.NewEmailTemplateRepository(db)

	// Services
	organizationService := service.NewOrganizationService(db, organizationRepo)
	serviceService := service.NewServiceService(db, serviceRepo, organizationServiceRepo)
	apiService := service.NewAPIService(db, apiRepo, serviceRepo)
	authContainerService := service.NewAuthContainerService(db, authContainerRepo, organizationRepo)
	roleService := service.NewRoleService(db, roleRepo)
	registerService := service.NewRegistrationService(db, authClientRepo, userRepo, userRoleRepo, userTokenRepo, roleRepo, inviteRepo)
	loginService := service.NewLoginService(db, authClientRepo, userRepo, userTokenRepo)
	profileService := service.NewProfileService(db, profileRepo)
	inviteService := service.NewInviteService(db, inviteRepo, authClientRepo, roleRepo, emailTemplateRepo)

	// Rest handlers
	organizationHandler := resthandler.NewOrganizationHandler(organizationService)
	serviceRestHandler := resthandler.NewServiceHandler(serviceService)
	apiRestHandler := resthandler.NewAPIHandler(apiService)
	authContainerRestHandler := resthandler.NewAuthContainerHandler(authContainerService)
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
		OrganizationRestHandler:  organizationHandler,
		ServiceRestHandler:       serviceRestHandler,
		APIRestHandler:           apiRestHandler,
		AuthContainerRestHandler: authContainerRestHandler,
		RoleRestHandler:          roleRestHandler,
		RegisterRestHandler:      registerRestHandler,
		LoginRestHandler:         loginRestHandler,
		ProfileRestHandler:       profileRestHandler,
		InviteRestHandler:        inviteRestHandler,
		// GRPC handler
		SeederHandler: seederGrpcHandler,
		// Repository
		UserRepository: userRepo,
	}
}
