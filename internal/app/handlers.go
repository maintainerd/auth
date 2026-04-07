package app

import (
	"github.com/maintainerd/auth/internal/handler/grpchandler"
	"github.com/maintainerd/auth/internal/handler/resthandler"
)

// hdlrs holds every handler instance. Private to the app package.
type hdlrs struct {
	// REST
	serviceREST           *resthandler.ServiceHandler
	apiREST               *resthandler.APIHandler
	permissionREST        *resthandler.PermissionHandler
	policyREST            *resthandler.PolicyHandler
	tenantREST            *resthandler.TenantHandler
	idpREST               *resthandler.IdentityProviderHandler
	clientREST            *resthandler.ClientHandler
	roleREST              *resthandler.RoleHandler
	userREST              *resthandler.UserHandler
	registerREST          *resthandler.RegisterHandler
	loginREST             *resthandler.LoginHandler
	profileREST           *resthandler.ProfileHandler
	userSettingREST       *resthandler.UserSettingHandler
	inviteREST            *resthandler.InviteHandler
	forgotPasswordREST    *resthandler.ForgotPasswordHandler
	resetPasswordREST     *resthandler.ResetPasswordHandler
	setupREST             *resthandler.SetupHandler
	apiKeyREST            *resthandler.APIKeyHandler
	signupFlowREST        *resthandler.SignupFlowHandler
	securitySettingREST   *resthandler.SecuritySettingHandler
	ipRestrictionRuleREST *resthandler.IPRestrictionRuleHandler
	emailTemplateREST     *resthandler.EmailTemplateHandler
	smsTemplateREST       *resthandler.SMSTemplateHandler
	loginTemplateREST     *resthandler.LoginTemplateHandler
	// gRPC
	seederGRPC *grpchandler.SeederHandler
}

func initHandlers(s *svcs) *hdlrs {
	return &hdlrs{
		serviceREST:           resthandler.NewServiceHandler(s.serviceService),
		apiREST:               resthandler.NewAPIHandler(s.apiService),
		permissionREST:        resthandler.NewPermissionHandler(s.permissionService),
		policyREST:            resthandler.NewPolicyHandler(s.policyService),
		tenantREST:            resthandler.NewTenantHandler(s.tenantService, s.tenantMemberService),
		idpREST:               resthandler.NewIdentityProviderHandler(s.idpService),
		clientREST:            resthandler.NewClientHandler(s.clientService),
		roleREST:              resthandler.NewRoleHandler(s.roleService),
		userREST:              resthandler.NewUserHandler(s.userService),
		registerREST:          resthandler.NewRegisterHandler(s.registerService),
		loginREST:             resthandler.NewLoginHandler(s.loginService),
		profileREST:           resthandler.NewProfileHandler(s.profileService),
		userSettingREST:       resthandler.NewUserSettingHandler(s.userSettingService),
		inviteREST:            resthandler.NewInviteHandler(s.inviteService),
		forgotPasswordREST:    resthandler.NewForgotPasswordHandler(s.forgotPasswordService),
		resetPasswordREST:     resthandler.NewResetPasswordHandler(s.resetPasswordService),
		setupREST:             resthandler.NewSetupHandler(s.setupService),
		apiKeyREST:            resthandler.NewAPIKeyHandler(s.apiKeyService),
		signupFlowREST:        resthandler.NewSignupFlowHandler(s.signupFlowService),
		securitySettingREST:   resthandler.NewSecuritySettingHandler(s.securitySettingService),
		ipRestrictionRuleREST: resthandler.NewIPRestrictionRuleHandler(s.ipRestrictionRuleService),
		emailTemplateREST:     resthandler.NewEmailTemplateHandler(s.emailTemplateService),
		smsTemplateREST:       resthandler.NewSMSTemplateHandler(s.smsTemplateService),
		loginTemplateREST:     resthandler.NewLoginTemplateHandler(s.loginTemplateService),
		seederGRPC:            grpchandler.NewSeederHandler(s.registerService),
	}
}
