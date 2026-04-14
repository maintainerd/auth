package app

import (
	"github.com/maintainerd/auth/internal/cache"
	"github.com/maintainerd/auth/internal/service"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

type App struct {
	DB          *gorm.DB
	RedisClient *redis.Client
	Cache       *cache.Cache
	// Services
	ServiceService           service.ServiceService
	APIService               service.APIService
	PermissionService        service.PermissionService
	PolicyService            service.PolicyService
	TenantService            service.TenantService
	TenantMemberService      service.TenantMemberService
	IdentityProviderService  service.IdentityProviderService
	ClientService            service.ClientService
	RoleService              service.RoleService
	UserService              service.UserService
	RegisterService          service.RegisterService
	LoginService             service.LoginService
	ProfileService           service.ProfileService
	UserSettingService       service.UserSettingService
	InviteService            service.InviteService
	ForgotPasswordService    service.ForgotPasswordService
	ResetPasswordService     service.ResetPasswordService
	SetupService             service.SetupService
	SignupFlowService        service.SignupFlowService
	APIKeyService            service.APIKeyService
	SecuritySettingService   service.SecuritySettingService
	IPRestrictionRuleService service.IPRestrictionRuleService
	EmailTemplateService     service.EmailTemplateService
	SMSTemplateService       service.SMSTemplateService
	LoginTemplateService     service.LoginTemplateService
	BrandingService          service.BrandingService
	TenantSettingService     service.TenantSettingService
	EmailConfigService       service.EmailConfigService
	SMSConfigService         service.SMSConfigService
	WebhookEndpointService   service.WebhookEndpointService
	AuthEventService         service.AuthEventService
}

// NewApp wires the full dependency graph in two focused steps:
//  1. initRepos    — every repository, bound to db
//  2. initServices — every service, consuming repos
//
// Handler creation is delegated to transport packages (rest, grpcserver).
func NewApp(db *gorm.DB, redisClient *redis.Client) *App {
	r := initRepos(db)
	appCache := cache.New(redisClient)
	s := initServices(db, r, appCache)

	return &App{
		DB:          db,
		RedisClient: redisClient,
		Cache:       appCache,
		// Services
		ServiceService:           s.serviceService,
		APIService:               s.apiService,
		PermissionService:        s.permissionService,
		PolicyService:            s.policyService,
		TenantService:            s.tenantService,
		TenantMemberService:      s.tenantMemberService,
		IdentityProviderService:  s.idpService,
		ClientService:            s.clientService,
		RoleService:              s.roleService,
		UserService:              s.userService,
		RegisterService:          s.registerService,
		LoginService:             s.loginService,
		ProfileService:           s.profileService,
		UserSettingService:       s.userSettingService,
		InviteService:            s.inviteService,
		ForgotPasswordService:    s.forgotPasswordService,
		ResetPasswordService:     s.resetPasswordService,
		SetupService:             s.setupService,
		SignupFlowService:        s.signupFlowService,
		APIKeyService:            s.apiKeyService,
		SecuritySettingService:   s.securitySettingService,
		IPRestrictionRuleService: s.ipRestrictionRuleService,
		EmailTemplateService:     s.emailTemplateService,
		SMSTemplateService:       s.smsTemplateService,
		LoginTemplateService:     s.loginTemplateService,
		BrandingService:          s.brandingService,
		TenantSettingService:     s.tenantSettingService,
		EmailConfigService:       s.emailConfigService,
		SMSConfigService:         s.smsConfigService,
		WebhookEndpointService:   s.webhookEndpointService,
		AuthEventService:         s.authEventService,
	}
}
