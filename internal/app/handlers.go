package app

import (
	"github.com/maintainerd/auth/internal/handler/grpc"
	"github.com/maintainerd/auth/internal/handler/rest"
)

// hdlrs holds every handler instance. Private to the app package.
type hdlrs struct {
	// REST
	serviceREST           *rest.ServiceHandler
	apiREST               *rest.APIHandler
	permissionREST        *rest.PermissionHandler
	policyREST            *rest.PolicyHandler
	tenantREST            *rest.TenantHandler
	idpREST               *rest.IdentityProviderHandler
	clientREST            *rest.ClientHandler
	roleREST              *rest.RoleHandler
	userREST              *rest.UserHandler
	registerREST          *rest.RegisterHandler
	loginREST             *rest.LoginHandler
	profileREST           *rest.ProfileHandler
	userSettingREST       *rest.UserSettingHandler
	inviteREST            *rest.InviteHandler
	forgotPasswordREST    *rest.ForgotPasswordHandler
	resetPasswordREST     *rest.ResetPasswordHandler
	setupREST             *rest.SetupHandler
	apiKeyREST            *rest.APIKeyHandler
	signupFlowREST        *rest.SignupFlowHandler
	securitySettingREST   *rest.SecuritySettingHandler
	ipRestrictionRuleREST *rest.IPRestrictionRuleHandler
	emailTemplateREST     *rest.EmailTemplateHandler
	smsTemplateREST       *rest.SMSTemplateHandler
	loginTemplateREST     *rest.LoginTemplateHandler
	// gRPC
	seederGRPC *grpc.SeederHandler
}

func initHandlers(s *svcs) *hdlrs {
	return &hdlrs{
		serviceREST:           rest.NewServiceHandler(s.serviceService),
		apiREST:               rest.NewAPIHandler(s.apiService),
		permissionREST:        rest.NewPermissionHandler(s.permissionService),
		policyREST:            rest.NewPolicyHandler(s.policyService),
		tenantREST:            rest.NewTenantHandler(s.tenantService, s.tenantMemberService),
		idpREST:               rest.NewIdentityProviderHandler(s.idpService),
		clientREST:            rest.NewClientHandler(s.clientService),
		roleREST:              rest.NewRoleHandler(s.roleService),
		userREST:              rest.NewUserHandler(s.userService),
		registerREST:          rest.NewRegisterHandler(s.registerService),
		loginREST:             rest.NewLoginHandler(s.loginService),
		profileREST:           rest.NewProfileHandler(s.profileService),
		userSettingREST:       rest.NewUserSettingHandler(s.userSettingService),
		inviteREST:            rest.NewInviteHandler(s.inviteService),
		forgotPasswordREST:    rest.NewForgotPasswordHandler(s.forgotPasswordService),
		resetPasswordREST:     rest.NewResetPasswordHandler(s.resetPasswordService),
		setupREST:             rest.NewSetupHandler(s.setupService),
		apiKeyREST:            rest.NewAPIKeyHandler(s.apiKeyService),
		signupFlowREST:        rest.NewSignupFlowHandler(s.signupFlowService),
		securitySettingREST:   rest.NewSecuritySettingHandler(s.securitySettingService),
		ipRestrictionRuleREST: rest.NewIPRestrictionRuleHandler(s.ipRestrictionRuleService),
		emailTemplateREST:     rest.NewEmailTemplateHandler(s.emailTemplateService),
		smsTemplateREST:       rest.NewSMSTemplateHandler(s.smsTemplateService),
		loginTemplateREST:     rest.NewLoginTemplateHandler(s.loginTemplateService),
		seederGRPC:            grpc.NewSeederHandler(s.registerService),
	}
}
