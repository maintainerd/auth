package app

import (
	"github.com/maintainerd/auth/internal/service"
	"gorm.io/gorm"
)

// svcs holds every service instance. Private to the app package.
type svcs struct {
	serviceService          service.ServiceService
	apiService              service.APIService
	permissionService       service.PermissionService
	tenantService           service.TenantService
	tenantMemberService     service.TenantMemberService
	idpService              service.IdentityProviderService
	clientService           service.ClientService
	roleService             service.RoleService
	userService             service.UserService
	registerService         service.RegisterService
	loginService            service.LoginService
	profileService          service.ProfileService
	userSettingService      service.UserSettingService
	inviteService           service.InviteService
	forgotPasswordService   service.ForgotPasswordService
	resetPasswordService    service.ResetPasswordService
	setupService            service.SetupService
	signupFlowService       service.SignupFlowService
	policyService           service.PolicyService
	apiKeyService           service.APIKeyService
	securitySettingService  service.SecuritySettingService
	ipRestrictionRuleService service.IPRestrictionRuleService
	emailTemplateService    service.EmailTemplateService
	smsTemplateService      service.SMSTemplateService
	loginTemplateService    service.LoginTemplateService
}

func initServices(db *gorm.DB, r *repos) *svcs {
	return &svcs{
		serviceService:          service.NewServiceService(db, r.serviceRepo, r.tenantServiceRepo, r.apiRepo, r.servicePolicyRepo, r.policyRepo),
		apiService:              service.NewAPIService(db, r.apiRepo, r.serviceRepo, r.tenantServiceRepo),
		permissionService:       service.NewPermissionService(db, r.permissionRepo, r.apiRepo, r.roleRepo, r.clientRepo),
		tenantService:           service.NewTenantService(db, r.tenantRepo),
		tenantMemberService:     service.NewTenantMemberService(db, r.tenantMemberRepo, r.userRepo, r.tenantRepo),
		idpService:              service.NewIdentityProviderService(db, r.idpRepo, r.tenantRepo, r.userRepo),
		clientService:           service.NewClientService(db, r.clientRepo, r.clientURIRepo, r.idpRepo, r.permissionRepo, r.clientPermissionRepo, r.clientAPIRepo, r.apiRepo, r.userRepo, r.tenantRepo),
		roleService:             service.NewRoleService(db, r.roleRepo, r.permissionRepo, r.rolePermissionRepo, r.userRepo, r.tenantRepo),
		userService:             service.NewUserService(db, r.userRepo, r.userIdentityRepo, r.userRoleRepo, r.roleRepo, r.tenantRepo, r.idpRepo, r.clientRepo, r.tenantUserRepo),
		registerService:         service.NewRegistrationService(db, r.clientRepo, r.userRepo, r.userRoleRepo, r.userTokenRepo, r.userIdentityRepo, r.roleRepo, r.inviteRepo, r.idpRepo, r.tenantUserRepo),
		loginService:            service.NewLoginService(db, r.clientRepo, r.userRepo, r.userTokenRepo, r.userIdentityRepo, r.idpRepo),
		profileService:          service.NewProfileService(db, r.profileRepo, r.userRepo),
		userSettingService:      service.NewUserSettingService(db, r.userSettingRepo, r.userRepo),
		inviteService:           service.NewInviteService(db, r.inviteRepo, r.clientRepo, r.roleRepo, r.emailTemplateRepo),
		forgotPasswordService:   service.NewForgotPasswordService(db, r.userRepo, r.userTokenRepo, r.clientRepo, r.emailTemplateRepo),
		resetPasswordService:    service.NewResetPasswordService(db, r.userRepo, r.userTokenRepo, r.clientRepo),
		setupService:            service.NewSetupService(db, r.userRepo, r.tenantRepo, r.tenantMemberRepo, r.tenantUserRepo, r.clientRepo, r.idpRepo, r.roleRepo, r.userRoleRepo, r.userTokenRepo, r.userIdentityRepo, r.profileRepo),
		signupFlowService:       service.NewSignupFlowService(db, r.signupFlowRepo, r.signupFlowRoleRepo, r.roleRepo, r.clientRepo),
		policyService:           service.NewPolicyService(db, r.policyRepo, r.serviceRepo, r.apiRepo),
		apiKeyService:           service.NewAPIKeyService(db, r.apiKeyRepo, r.apiKeyAPIRepo, r.apiKeyPermissionRepo, r.apiRepo, r.userRepo, r.permissionRepo),
		securitySettingService:  service.NewSecuritySettingService(db, r.securitySettingRepo, r.securitySettingsAuditRepo),
		ipRestrictionRuleService: service.NewIPRestrictionRuleService(db, r.ipRestrictionRuleRepo),
		emailTemplateService:    service.NewEmailTemplateService(db, r.emailTemplateRepo),
		smsTemplateService:      service.NewSMSTemplateService(db, r.smsTemplateRepo),
		loginTemplateService:    service.NewLoginTemplateService(r.loginTemplateRepo),
	}
}
