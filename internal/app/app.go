package app

import (
	"github.com/maintainerd/auth/internal/handler/grpc"
	"github.com/maintainerd/auth/internal/handler/rest"
	"github.com/maintainerd/auth/internal/repository"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

type App struct {
	DB          *gorm.DB
	RedisClient *redis.Client
	// REST handlers
	ServiceRestHandler           *rest.ServiceHandler
	APIRestHandler               *rest.APIHandler
	PermissionRestHandler        *rest.PermissionHandler
	PolicyRestHandler            *rest.PolicyHandler
	TenantRestHandler            *rest.TenantHandler
	IdentityProviderRestHandler  *rest.IdentityProviderHandler
	ClientRestHandler            *rest.ClientHandler
	RoleRestHandler              *rest.RoleHandler
	UserRestHandler              *rest.UserHandler
	RegisterRestHandler          *rest.RegisterHandler
	LoginRestHandler             *rest.LoginHandler
	ProfileRestHandler           *rest.ProfileHandler
	UserSettingRestHandler       *rest.UserSettingHandler
	InviteRestHandler            *rest.InviteHandler
	ForgotPasswordRestHandler    *rest.ForgotPasswordHandler
	ResetPasswordRestHandler     *rest.ResetPasswordHandler
	SetupRestHandler             *rest.SetupHandler
	APIKeyRestHandler            *rest.APIKeyHandler
	SignupFlowRestHandler        *rest.SignupFlowHandler
	SecuritySettingRestHandler   *rest.SecuritySettingHandler
	IPRestrictionRuleRestHandler *rest.IPRestrictionRuleHandler
	EmailTemplateRestHandler     *rest.EmailTemplateHandler
	SMSTemplateRestHandler       *rest.SMSTemplateHandler
	LoginTemplateRestHandler     *rest.LoginTemplateHandler
	// gRPC handlers
	SeederHandler *grpc.SeederHandler
	// Repositories exposed for middleware
	UserRepository repository.UserRepository
}

// NewApp wires the full dependency graph in three focused steps:
//  1. initRepos  — every repository, bound to db
//  2. initServices — every service, consuming repos
//  3. initHandlers — every handler, consuming services
func NewApp(db *gorm.DB, redisClient *redis.Client) *App {
	r := initRepos(db)
	s := initServices(db, r)
	h := initHandlers(s)

	return &App{
		DB:          db,
		RedisClient: redisClient,
		// REST handlers
		ServiceRestHandler:           h.serviceREST,
		APIRestHandler:               h.apiREST,
		PermissionRestHandler:        h.permissionREST,
		PolicyRestHandler:            h.policyREST,
		TenantRestHandler:            h.tenantREST,
		IdentityProviderRestHandler:  h.idpREST,
		ClientRestHandler:            h.clientREST,
		RoleRestHandler:              h.roleREST,
		UserRestHandler:              h.userREST,
		RegisterRestHandler:          h.registerREST,
		LoginRestHandler:             h.loginREST,
		ProfileRestHandler:           h.profileREST,
		UserSettingRestHandler:       h.userSettingREST,
		InviteRestHandler:            h.inviteREST,
		ForgotPasswordRestHandler:    h.forgotPasswordREST,
		ResetPasswordRestHandler:     h.resetPasswordREST,
		SetupRestHandler:             h.setupREST,
		APIKeyRestHandler:            h.apiKeyREST,
		SignupFlowRestHandler:        h.signupFlowREST,
		SecuritySettingRestHandler:   h.securitySettingREST,
		IPRestrictionRuleRestHandler: h.ipRestrictionRuleREST,
		EmailTemplateRestHandler:     h.emailTemplateREST,
		SMSTemplateRestHandler:       h.smsTemplateREST,
		LoginTemplateRestHandler:     h.loginTemplateREST,
		// gRPC handlers
		SeederHandler: h.seederGRPC,
		// Repositories exposed for middleware
		UserRepository: r.userRepo,
	}
}
