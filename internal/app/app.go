package app

import (
	"github.com/maintainerd/auth/internal/handler/grpchandler"
	"github.com/maintainerd/auth/internal/handler/resthandler"
	"github.com/maintainerd/auth/internal/repository"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

type App struct {
	DB          *gorm.DB
	RedisClient *redis.Client
	// REST handlers
	ServiceRestHandler           *resthandler.ServiceHandler
	APIRestHandler               *resthandler.APIHandler
	PermissionRestHandler        *resthandler.PermissionHandler
	PolicyRestHandler            *resthandler.PolicyHandler
	TenantRestHandler            *resthandler.TenantHandler
	IdentityProviderRestHandler  *resthandler.IdentityProviderHandler
	ClientRestHandler            *resthandler.ClientHandler
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
	IPRestrictionRuleRestHandler *resthandler.IPRestrictionRuleHandler
	EmailTemplateRestHandler     *resthandler.EmailTemplateHandler
	SMSTemplateRestHandler       *resthandler.SMSTemplateHandler
	LoginTemplateRestHandler     *resthandler.LoginTemplateHandler
	// gRPC handlers
	SeederHandler *grpchandler.SeederHandler
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
