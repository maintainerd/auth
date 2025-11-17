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
	ServiceRestHandler          *resthandler.ServiceHandler
	APIRestHandler              *resthandler.APIHandler
	PermissionRestHandler       *resthandler.PermissionHandler
	TenantRestHandler           *resthandler.TenantHandler
	IdentityProviderRestHandler *resthandler.IdentityProviderHandler
	AuthClientRestHandler       *resthandler.AuthClientHandler
	RoleRestHandler             *resthandler.RoleHandler
	UserRestHandler             *resthandler.UserHandler
	RegisterRestHandler         *resthandler.RegisterHandler
	LoginRestHandler            *resthandler.LoginHandler
	ProfileRestHandler          *resthandler.ProfileHandler
	UserSettingRestHandler      *resthandler.UserSettingHandler
	InviteRestHandler           *resthandler.InviteHandler
	ForgotPasswordRestHandler   *resthandler.ForgotPasswordHandler
	ResetPasswordRestHandler    *resthandler.ResetPasswordHandler
	SetupRestHandler            *resthandler.SetupHandler
	// Grpc handler
	SeederHandler *grpchandler.SeederHandler
	// Repository
	UserRepository repository.UserRepository
}

func NewApp(db *gorm.DB, redisClient *redis.Client) *App {
	// Repositories
	serviceRepo := repository.NewServiceRepository(db)
	tenantServiceRepo := repository.NewTenantServiceRepository(db)
	apiRepo := repository.NewAPIRepository(db)
	permissionRepo := repository.NewPermissionRepository(db)
	tenantRepo := repository.NewTenantRepository(db)
	idpRepo := repository.NewIdentityProviderRepository(db)
	roleRepo := repository.NewRoleRepository(db)
	rolePermissionRepo := repository.NewRolePermissionRepository(db)
	authClientRepo := repository.NewAuthClientRepository(db)
	authClientPermissionRepo := repository.NewAuthClientPermissionRepository(db)
	authClientRedirectUriRepo := repository.NewAuthClientRedirectURIRepository(db)
	userRepo := repository.NewUserRepository(db)
	userIdentityRepo := repository.NewUserIdentityRepository(db)
	userRoleRepo := repository.NewUserRoleRepository(db)
	userTokenRepo := repository.NewUserTokenRepository(db)
	profileRepo := repository.NewProfileRepository(db)
	userSettingRepo := repository.NewUserSettingRepository(db)
	inviteRepo := repository.NewInviteRepository(db)
	emailTemplateRepo := repository.NewEmailTemplateRepository(db)

	// Services
	serviceService := service.NewServiceService(db, serviceRepo, tenantServiceRepo)
	apiService := service.NewAPIService(db, apiRepo, serviceRepo)
	permissionService := service.NewPermissionService(db, permissionRepo, apiRepo, roleRepo, authClientRepo)
	tenantService := service.NewTenantService(db, tenantRepo)
	idpService := service.NewIdentityProviderService(db, idpRepo, tenantRepo, userRepo)
	authClientService := service.NewAuthClientService(db, authClientRepo, authClientRedirectUriRepo, idpRepo, permissionRepo, authClientPermissionRepo, userRepo, tenantRepo)
	roleService := service.NewRoleService(db, roleRepo, permissionRepo, rolePermissionRepo, userRepo, tenantRepo)
	userService := service.NewUserService(db, userRepo, userIdentityRepo, userRoleRepo, roleRepo, tenantRepo, idpRepo, authClientRepo)
	registerService := service.NewRegistrationService(db, authClientRepo, userRepo, userRoleRepo, userTokenRepo, userIdentityRepo, roleRepo, inviteRepo, idpRepo)
	loginService := service.NewLoginService(db, authClientRepo, userRepo, userTokenRepo, idpRepo)
	profileService := service.NewProfileService(db, profileRepo, userRepo)
	userSettingService := service.NewUserSettingService(db, userSettingRepo, userRepo)
	inviteService := service.NewInviteService(db, inviteRepo, authClientRepo, roleRepo, emailTemplateRepo)
	forgotPasswordService := service.NewForgotPasswordService(db, userRepo, userTokenRepo, authClientRepo, emailTemplateRepo)
	resetPasswordService := service.NewResetPasswordService(db, userRepo, userTokenRepo, authClientRepo)
	setupService := service.NewSetupService(db, userRepo, tenantRepo, authClientRepo, idpRepo, roleRepo, userRoleRepo, userTokenRepo, userIdentityRepo, profileRepo)

	// Rest handlers
	serviceRestHandler := resthandler.NewServiceHandler(serviceService)
	apiRestHandler := resthandler.NewAPIHandler(apiService)
	permissionRestHandler := resthandler.NewPermissionHandler(permissionService)
	tenantRestHandler := resthandler.NewTenantHandler(tenantService)
	idpRestHandler := resthandler.NewIdentityProviderHandler(idpService)
	authClientRestHandler := resthandler.NewAuthClientHandler(authClientService)
	roleRestHandler := resthandler.NewRoleHandler(roleService)
	userRestHandler := resthandler.NewUserHandler(userService)
	registerRestHandler := resthandler.NewRegisterHandler(registerService)
	loginRestHandler := resthandler.NewLoginHandler(loginService)
	profileRestHandler := resthandler.NewProfileHandler(profileService)
	userSettingRestHandler := resthandler.NewUserSettingHandler(userSettingService)
	inviteRestHandler := resthandler.NewInviteHandler(inviteService)
	forgotPasswordRestHandler := resthandler.NewForgotPasswordHandler(forgotPasswordService)
	resetPasswordRestHandler := resthandler.NewResetPasswordHandler(resetPasswordService)
	setupRestHandler := resthandler.NewSetupHandler(setupService)

	// GRPC handlers
	seederGrpcHandler := grpchandler.NewSeederHandler(registerService)

	return &App{
		DB:          db,
		RedisClient: redisClient,
		// Rest handler
		ServiceRestHandler:          serviceRestHandler,
		APIRestHandler:              apiRestHandler,
		PermissionRestHandler:       permissionRestHandler,
		TenantRestHandler:           tenantRestHandler,
		IdentityProviderRestHandler: idpRestHandler,
		AuthClientRestHandler:       authClientRestHandler,
		RoleRestHandler:             roleRestHandler,
		UserRestHandler:             userRestHandler,
		RegisterRestHandler:         registerRestHandler,
		LoginRestHandler:            loginRestHandler,
		ProfileRestHandler:          profileRestHandler,
		UserSettingRestHandler:      userSettingRestHandler,
		InviteRestHandler:           inviteRestHandler,
		ForgotPasswordRestHandler:   forgotPasswordRestHandler,
		ResetPasswordRestHandler:    resetPasswordRestHandler,
		SetupRestHandler:            setupRestHandler,
		// GRPC handler
		SeederHandler: seederGrpcHandler,
		// Repository
		UserRepository: userRepo,
	}
}
