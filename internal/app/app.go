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
	ServiceRestHandler           *resthandler.ServiceHandler
	APIRestHandler               *resthandler.APIHandler
	PermissionRestHandler        *resthandler.PermissionHandler
	PolicyRestHandler            *resthandler.PolicyHandler
	TenantRestHandler            *resthandler.TenantHandler
	IdentityProviderRestHandler  *resthandler.IdentityProviderHandler
	ClientRestHandler        *resthandler.ClientHandler
	RoleRestHandler              *resthandler.RoleHandler
	UserRestHandler              *resthandler.UserHandler
	RegisterRestHandler          *resthandler.RegisterHandler
	LoginRestHandler             *resthandler.LoginHandler
	ProfileRestHandler           *resthandler.ProfileHandler
	UserSettingRestHandler       *resthandler.UserSettingHandler
	InviteRestHandler            *resthandler.InviteHandler
	ForgotPasswordRestHandler    *resthandler.ForgotPasswordHandler
	ResetPasswordRestHandler     *resthandler.ResetPasswordHandler
	SetupRestHandler             *resthandler.SetupHandler
	APIKeyRestHandler            *resthandler.APIKeyHandler
	SignupFlowRestHandler        *resthandler.SignupFlowHandler
	SecuritySettingRestHandler   *resthandler.SecuritySettingHandler
	IpRestrictionRuleRestHandler *resthandler.IpRestrictionRuleHandler
	EmailTemplateRestHandler     *resthandler.EmailTemplateHandler
	SmsTemplateRestHandler       *resthandler.SmsTemplateHandler
	LoginTemplateRestHandler     *resthandler.LoginTemplateHandler
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
	tenantMemberRepo := repository.NewTenantMemberRepository(db)
	tenantUserRepo := repository.NewTenantUserRepository(db)
	idpRepo := repository.NewIdentityProviderRepository(db)
	roleRepo := repository.NewRoleRepository(db)
	rolePermissionRepo := repository.NewRolePermissionRepository(db)
	ClientRepo := repository.NewClientRepository(db)
	ClientPermissionRepo := repository.NewClientPermissionRepository(db)
	ClientApiRepo := repository.NewClientApiRepository(db)
	ClientUriRepo := repository.NewClientURIRepository(db)
	userRepo := repository.NewUserRepository(db)
	userIdentityRepo := repository.NewUserIdentityRepository(db)
	userRoleRepo := repository.NewUserRoleRepository(db)
	userTokenRepo := repository.NewUserTokenRepository(db)
	profileRepo := repository.NewProfileRepository(db)
	userSettingRepo := repository.NewUserSettingRepository(db)
	inviteRepo := repository.NewInviteRepository(db)
	emailTemplateRepo := repository.NewEmailTemplateRepository(db)
	smsTemplateRepo := repository.NewSmsTemplateRepository(db)
	loginTemplateRepo := repository.NewLoginTemplateRepository(db)
	policyRepo := repository.NewPolicyRepository(db)
	servicePolicyRepo := repository.NewServicePolicyRepository(db)
	apiKeyRepo := repository.NewAPIKeyRepository(db)
	apiKeyApiRepo := repository.NewAPIKeyApiRepository(db)
	apiKeyPermissionRepo := repository.NewAPIKeyPermissionRepository(db)
	signupFlowRepo := repository.NewSignupFlowRepository(db)
	signupFlowRoleRepo := repository.NewSignupFlowRoleRepository(db)
	securitySettingRepo := repository.NewSecuritySettingRepository(db)
	securitySettingsAuditRepo := repository.NewSecuritySettingsAuditRepository(db)
	ipRestrictionRuleRepo := repository.NewIpRestrictionRuleRepository(db)

	// Services
	serviceService := service.NewServiceService(db, serviceRepo, tenantServiceRepo, apiRepo, servicePolicyRepo, policyRepo)
	apiService := service.NewAPIService(db, apiRepo, serviceRepo, tenantServiceRepo)
	permissionService := service.NewPermissionService(db, permissionRepo, apiRepo, roleRepo, ClientRepo)
	tenantService := service.NewTenantService(db, tenantRepo)
	tenantMemberService := service.NewTenantMemberService(db, tenantMemberRepo, userRepo, tenantRepo)
	idpService := service.NewIdentityProviderService(db, idpRepo, tenantRepo, userRepo)
	ClientService := service.NewClientService(db, ClientRepo, ClientUriRepo, idpRepo, permissionRepo, ClientPermissionRepo, ClientApiRepo, apiRepo, userRepo, tenantRepo)
	roleService := service.NewRoleService(db, roleRepo, permissionRepo, rolePermissionRepo, userRepo, tenantRepo)
	userService := service.NewUserService(db, userRepo, userIdentityRepo, userRoleRepo, roleRepo, tenantRepo, idpRepo, ClientRepo, tenantUserRepo)
	registerService := service.NewRegistrationService(db, ClientRepo, userRepo, userRoleRepo, userTokenRepo, userIdentityRepo, roleRepo, inviteRepo, idpRepo, tenantUserRepo)
	loginService := service.NewLoginService(db, ClientRepo, userRepo, userTokenRepo, userIdentityRepo, idpRepo)
	profileService := service.NewProfileService(db, profileRepo, userRepo)
	userSettingService := service.NewUserSettingService(db, userSettingRepo, userRepo)
	inviteService := service.NewInviteService(db, inviteRepo, ClientRepo, roleRepo, emailTemplateRepo)
	forgotPasswordService := service.NewForgotPasswordService(db, userRepo, userTokenRepo, ClientRepo, emailTemplateRepo)
	resetPasswordService := service.NewResetPasswordService(db, userRepo, userTokenRepo, ClientRepo)
	setupService := service.NewSetupService(db, userRepo, tenantRepo, tenantMemberRepo, tenantUserRepo, ClientRepo, idpRepo, roleRepo, userRoleRepo, userTokenRepo, userIdentityRepo, profileRepo)
	signupFlowService := service.NewSignupFlowService(db, signupFlowRepo, signupFlowRoleRepo, roleRepo, ClientRepo)
	policyService := service.NewPolicyService(db, policyRepo, serviceRepo, apiRepo)
	apiKeyService := service.NewAPIKeyService(db, apiKeyRepo, apiKeyApiRepo, apiKeyPermissionRepo, apiRepo, userRepo, permissionRepo)
	securitySettingService := service.NewSecuritySettingService(db, securitySettingRepo, securitySettingsAuditRepo)
	ipRestrictionRuleService := service.NewIpRestrictionRuleService(db, ipRestrictionRuleRepo)
	emailTemplateService := service.NewEmailTemplateService(db, emailTemplateRepo)
	smsTemplateService := service.NewSmsTemplateService(db, smsTemplateRepo)
	loginTemplateService := service.NewLoginTemplateService(loginTemplateRepo)

	// Rest handlers
	serviceRestHandler := resthandler.NewServiceHandler(serviceService)
	apiRestHandler := resthandler.NewAPIHandler(apiService)
	permissionRestHandler := resthandler.NewPermissionHandler(permissionService)
	tenantRestHandler := resthandler.NewTenantHandler(tenantService, tenantMemberService)
	idpRestHandler := resthandler.NewIdentityProviderHandler(idpService)
	ClientRestHandler := resthandler.NewClientHandler(ClientService)
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
	policyRestHandler := resthandler.NewPolicyHandler(policyService)
	signupFlowRestHandler := resthandler.NewSignupFlowHandler(signupFlowService)
	apiKeyRestHandler := resthandler.NewAPIKeyHandler(apiKeyService)
	securitySettingRestHandler := resthandler.NewSecuritySettingHandler(securitySettingService)
	ipRestrictionRuleRestHandler := resthandler.NewIpRestrictionRuleHandler(ipRestrictionRuleService)
	emailTemplateRestHandler := resthandler.NewEmailTemplateHandler(emailTemplateService)
	smsTemplateRestHandler := resthandler.NewSmsTemplateHandler(smsTemplateService)
	loginTemplateRestHandler := resthandler.NewLoginTemplateHandler(loginTemplateService)

	// GRPC handlers
	seederGrpcHandler := grpchandler.NewSeederHandler(registerService)

	return &App{
		DB:          db,
		RedisClient: redisClient,
		// Rest handler
		ServiceRestHandler:           serviceRestHandler,
		APIRestHandler:               apiRestHandler,
		PermissionRestHandler:        permissionRestHandler,
		PolicyRestHandler:            policyRestHandler,
		TenantRestHandler:            tenantRestHandler,
		IdentityProviderRestHandler:  idpRestHandler,
		ClientRestHandler:        ClientRestHandler,
		RoleRestHandler:              roleRestHandler,
		UserRestHandler:              userRestHandler,
		RegisterRestHandler:          registerRestHandler,
		LoginRestHandler:             loginRestHandler,
		ProfileRestHandler:           profileRestHandler,
		UserSettingRestHandler:       userSettingRestHandler,
		InviteRestHandler:            inviteRestHandler,
		ForgotPasswordRestHandler:    forgotPasswordRestHandler,
		ResetPasswordRestHandler:     resetPasswordRestHandler,
		SignupFlowRestHandler:        signupFlowRestHandler,
		SetupRestHandler:             setupRestHandler,
		APIKeyRestHandler:            apiKeyRestHandler,
		SecuritySettingRestHandler:   securitySettingRestHandler,
		IpRestrictionRuleRestHandler: ipRestrictionRuleRestHandler,
		EmailTemplateRestHandler:     emailTemplateRestHandler,
		SmsTemplateRestHandler:       smsTemplateRestHandler,
		LoginTemplateRestHandler:     loginTemplateRestHandler,
		// GRPC handler
		SeederHandler: seederGrpcHandler,
		// Repository
		UserRepository: userRepo,
	}
}
