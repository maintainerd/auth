package app

import (
	"github.com/maintainerd/auth/internal/cache"
	"github.com/maintainerd/auth/internal/service"
	"gorm.io/gorm"
)

// svcs holds every service instance. Private to the app package.
type svcs struct {
	serviceService           service.ServiceService
	apiService               service.APIService
	permissionService        service.PermissionService
	tenantService            service.TenantService
	tenantMemberService      service.TenantMemberService
	idpService               service.IdentityProviderService
	clientService            service.ClientService
	roleService              service.RoleService
	userService              service.UserService
	registerService          service.RegisterService
	loginService             service.LoginService
	profileService           service.ProfileService
	userSettingService       service.UserSettingService
	inviteService            service.InviteService
	forgotPasswordService    service.ForgotPasswordService
	resetPasswordService     service.ResetPasswordService
	emailVerificationService service.EmailVerificationService
	magicLinkService         service.MagicLinkService
	setupService             service.SetupService
	signupFlowService        service.SignupFlowService
	policyService            service.PolicyService
	apiKeyService            service.APIKeyService
	securitySettingService   service.SecuritySettingService
	ipRestrictionRuleService service.IPRestrictionRuleService
	emailTemplateService     service.EmailTemplateService
	smsTemplateService       service.SMSTemplateService
	loginTemplateService     service.LoginTemplateService
	brandingService          service.BrandingService
	tenantSettingService     service.TenantSettingService
	emailConfigService       service.EmailConfigService
	smsConfigService         service.SMSConfigService
	webhookEndpointService   service.WebhookEndpointService
	authEventService         service.AuthEventService
	oauthAuthorizeService     service.OAuthAuthorizeService
	oauthTokenService         service.OAuthTokenService
	oauthConsentService       service.OAuthConsentService
	oauthPARService           service.OAuthPARService
	oauthDeviceService        service.OAuthDeviceService
	oauthTokenExchangeService service.OAuthTokenExchangeService
	oauthSessionService       service.OAuthSessionService
	oauthCIBAService          service.OAuthCIBAService
	oauthRegisterService      service.OAuthRegisterService
	accountService            service.AccountService
	sessionService            service.SessionService
	smsLoginService           service.SMSLoginService
	mfaService                service.MFAService
	webAuthnService           service.WebAuthnService
	federationService         service.FederationService
}

func initServices(db *gorm.DB, r *repos, appCache *cache.Cache) (*svcs, error) {
	// Create authEventService first — it is injected into other services that
	// need structured audit logging.
	authEventSvc := service.NewAuthEventService(r.authEventRepo)

	sessionSvc := service.NewSessionService(r.userTokenRepo)

	s := &svcs{
		serviceService:           service.NewServiceService(db, r.serviceRepo, r.tenantServiceRepo, r.apiRepo, r.servicePolicyRepo, r.policyRepo),
		apiService:               service.NewAPIService(db, r.apiRepo, r.serviceRepo, r.tenantServiceRepo),
		permissionService:        service.NewPermissionService(db, r.permissionRepo, r.apiRepo, r.roleRepo, r.clientRepo, appCache),
		tenantService:            service.NewTenantService(db, r.tenantRepo),
		tenantMemberService:      service.NewTenantMemberService(db, r.tenantMemberRepo, r.userRepo, r.tenantRepo),
		idpService:               service.NewIdentityProviderService(db, r.idpRepo, r.tenantRepo, r.userRepo),
		clientService:            service.NewClientService(db, r.clientRepo, r.clientURIRepo, r.idpRepo, r.permissionRepo, r.clientPermissionRepo, r.clientAPIRepo, r.apiRepo, r.userRepo, r.tenantRepo),
		roleService:              service.NewRoleService(db, r.roleRepo, r.permissionRepo, r.rolePermissionRepo, r.userRepo, r.tenantRepo, appCache),
		userService:              service.NewUserService(db, r.userRepo, r.userIdentityRepo, r.userRoleRepo, r.roleRepo, r.tenantRepo, r.idpRepo, r.clientRepo, r.userPoolRepo, appCache, r.userTokenRepo, r.securitySettingRepo, r.userPasswordHistoryRepo),
		registerService:          service.NewRegistrationService(db, r.clientRepo, r.userRepo, r.userRoleRepo, r.userTokenRepo, r.userIdentityRepo, r.roleRepo, r.inviteRepo, r.idpRepo, r.securitySettingRepo, r.userPasswordHistoryRepo),
		loginService:             service.NewLoginService(db, r.clientRepo, r.userRepo, r.userTokenRepo, r.userIdentityRepo, r.idpRepo, authEventSvc, sessionSvc, r.securitySettingRepo),
		sessionService:           sessionSvc,
		profileService:           service.NewProfileService(db, r.profileRepo, r.userRepo),
		userSettingService:       service.NewUserSettingService(db, r.userSettingRepo, r.userRepo),
		inviteService:            service.NewInviteService(db, r.inviteRepo, r.clientRepo, r.roleRepo, r.emailTemplateRepo),
		forgotPasswordService:    service.NewForgotPasswordService(db, r.userRepo, r.userTokenRepo, r.clientRepo, r.emailTemplateRepo),
		resetPasswordService:     service.NewResetPasswordService(db, r.userRepo, r.userTokenRepo, r.clientRepo, r.securitySettingRepo, r.userPasswordHistoryRepo),
		emailVerificationService: service.NewEmailVerificationService(db, r.userRepo, r.userTokenRepo, r.clientRepo, r.emailTemplateRepo),
		magicLinkService:         service.NewMagicLinkService(db, r.userRepo, r.userTokenRepo, r.clientRepo, r.userIdentityRepo, r.idpRepo, r.emailTemplateRepo),
		setupService:             service.NewSetupService(db, r.userRepo, r.tenantRepo, r.tenantMemberRepo, r.clientRepo, r.idpRepo, r.roleRepo, r.userRoleRepo, r.userTokenRepo, r.userIdentityRepo, r.profileRepo),
		signupFlowService:        service.NewSignupFlowService(db, r.signupFlowRepo, r.signupFlowRoleRepo, r.roleRepo, r.clientRepo),
		policyService:            service.NewPolicyService(db, r.policyRepo, r.serviceRepo, r.apiRepo),
		apiKeyService:            service.NewAPIKeyService(db, r.apiKeyRepo, r.apiKeyAPIRepo, r.apiKeyPermissionRepo, r.apiRepo, r.userRepo, r.permissionRepo),
		securitySettingService:   service.NewSecuritySettingService(db, r.securitySettingRepo, r.securitySettingsAuditRepo),
		ipRestrictionRuleService: service.NewIPRestrictionRuleService(db, r.ipRestrictionRuleRepo),
		emailTemplateService:     service.NewEmailTemplateService(db, r.emailTemplateRepo),
		smsTemplateService:       service.NewSMSTemplateService(db, r.smsTemplateRepo),
		loginTemplateService:     service.NewLoginTemplateService(r.loginTemplateRepo),
		brandingService:          service.NewBrandingService(r.brandingRepo),
		tenantSettingService:     service.NewTenantSettingService(r.tenantSettingRepo),
		emailConfigService:       service.NewEmailConfigService(r.emailConfigRepo),
		smsConfigService:         service.NewSMSConfigService(r.smsConfigRepo),
		webhookEndpointService:   service.NewWebhookEndpointService(r.webhookEndpointRepo),
		authEventService:         authEventSvc,
		oauthAuthorizeService:     service.NewOAuthAuthorizeService(db, r.clientRepo, r.clientURIRepo, r.oauthAuthCodeRepo, r.oauthConsentGrantRepo, r.oauthConsentChallengeRepo, authEventSvc),
		oauthTokenService:         service.NewOAuthTokenService(db, r.clientRepo, r.oauthAuthCodeRepo, r.oauthRefreshTokenRepo, r.userRepo, r.userIdentityRepo, authEventSvc),
		oauthConsentService:       service.NewOAuthConsentService(r.oauthConsentGrantRepo),
		oauthPARService:           service.NewOAuthPARService(db, r.clientRepo, r.clientURIRepo, r.oauthPARRequestRepo, authEventSvc),
		oauthDeviceService:        service.NewOAuthDeviceService(db, r.clientRepo, r.oauthDeviceCodeRepo, r.userRepo, r.userIdentityRepo, authEventSvc),
		oauthTokenExchangeService: service.NewOAuthTokenExchangeService(db, r.clientRepo, r.userRepo, authEventSvc),
		oauthSessionService:       service.NewOAuthSessionService(r.clientRepo, r.userRepo, r.oauthRefreshTokenRepo, authEventSvc),
		oauthCIBAService:          service.NewOAuthCIBAService(db, r.clientRepo, r.oauthCIBARequestRepo, r.userRepo, authEventSvc),
		oauthRegisterService:      service.NewOAuthRegisterService(db, r.clientRepo, r.clientURIRepo, r.tenantRepo, authEventSvc),
		accountService:            service.NewAccountService(db, r.userRepo, r.userTokenRepo, r.profileRepo, r.userSettingRepo, r.roleRepo, r.clientRepo, r.userBackupCodeRepo, r.userIdentityRepo, r.idpRepo, authEventSvc),
		smsLoginService:           service.NewSMSLoginService(db, r.userRepo, r.smsOtpRepo, r.clientRepo, r.userIdentityRepo, r.idpRepo, authEventSvc),
		mfaService:        service.NewMFAService(db, r.userRepo, r.totpSecretRepo, r.webAuthnCredRepo, r.userBackupCodeRepo, r.securitySettingRepo, authEventSvc),
		federationService: service.NewFederationService(db, r.userRepo, r.userIdentityRepo, r.idpRepo, r.clientRepo, r.userRoleRepo, r.roleRepo, authEventSvc),
	}

	waSvc, err := service.NewWebAuthnService(db, r.userRepo, r.webAuthnCredRepo, appCache, authEventSvc)
	if err != nil {
		return nil, err
	}
	s.webAuthnService = waSvc
	return s, nil
}
