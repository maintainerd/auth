package app

import (
	"github.com/maintainerd/auth/internal/authevent"
	"github.com/maintainerd/auth/internal/authn"
	"github.com/maintainerd/auth/internal/branding"
	"github.com/maintainerd/auth/internal/client"
	"github.com/maintainerd/auth/internal/iam"
	"github.com/maintainerd/auth/internal/idp"
	"github.com/maintainerd/auth/internal/invite"
	"github.com/maintainerd/auth/internal/mfa"
	"github.com/maintainerd/auth/internal/notifier"
	"github.com/maintainerd/auth/internal/oauth"
	"github.com/maintainerd/auth/internal/platform/cache"
	"github.com/maintainerd/auth/internal/secpolicy"
	"github.com/maintainerd/auth/internal/setup"
	"github.com/maintainerd/auth/internal/tenant"
	"github.com/maintainerd/auth/internal/user"
	"github.com/maintainerd/auth/internal/webhook"
	"gorm.io/gorm"
)

// svcs holds every service instance. Private to the app package.
type svcs struct {
	serviceService            iam.ServiceService
	apiService                iam.APIService
	permissionService         iam.PermissionService
	tenantService             tenant.TenantService
	tenantMemberService       tenant.TenantMemberService
	idpService                idp.IdentityProviderService
	clientService             client.ClientService
	roleService               iam.RoleService
	userService               user.UserService
	registerService           authn.RegisterService
	loginService              authn.LoginService
	profileService            user.ProfileService
	userSettingService        user.UserSettingService
	inviteService             invite.InviteService
	forgotPasswordService     authn.ForgotPasswordService
	resetPasswordService      authn.ResetPasswordService
	emailVerificationService  authn.EmailVerificationService
	magicLinkService          authn.MagicLinkService
	setupService              setup.SetupService
	signupFlowService         idp.SignupFlowService
	policyService             iam.PolicyService
	apiKeyService             client.APIKeyService
	securitySettingService    secpolicy.SecuritySettingService
	ipRestrictionRuleService  secpolicy.IPRestrictionRuleService
	emailTemplateService      branding.EmailTemplateService
	smsTemplateService        branding.SMSTemplateService
	loginTemplateService      branding.LoginTemplateService
	brandingService           branding.BrandingService
	tenantSettingService      tenant.TenantSettingService
	emailConfigService        notifier.EmailConfigService
	smsConfigService          notifier.SMSConfigService
	webhookEndpointService    webhook.WebhookEndpointService
	authEventService          authevent.AuthEventService
	oauthAuthorizeService     oauth.OAuthAuthorizeService
	oauthTokenService         oauth.OAuthTokenService
	oauthConsentService       oauth.OAuthConsentService
	oauthPARService           oauth.OAuthPARService
	oauthDeviceService        oauth.OAuthDeviceService
	oauthTokenExchangeService oauth.OAuthTokenExchangeService
	oauthSessionService       oauth.OAuthSessionService
	oauthCIBAService          oauth.OAuthCIBAService
	oauthRegisterService      oauth.OAuthRegisterService
	accountService            user.AccountService
	sessionService            authn.SessionService
	smsLoginService           authn.SMSLoginService
	mfaService                mfa.MFAService
	webAuthnService           mfa.WebAuthnService
	federationService         idp.FederationService
}

func initServices(db *gorm.DB, r *repos, appCache *cache.Cache) (*svcs, error) {
	// Create authEventService first — it is injected into other services that
	// need structured audit logging.
	authEventSvc := authevent.NewAuthEventService(r.authEventRepo, webhook.NewDispatcher(r.webhookEndpointRepo))

	sessionSvc := authn.NewSessionService(r.userTokenRepo)

	s := &svcs{
		serviceService:            iam.NewServiceService(db, r.serviceRepo, r.tenantServiceRepo, r.apiRepo, r.servicePolicyRepo, r.policyRepo),
		apiService:                iam.NewAPIService(db, r.apiRepo, r.serviceRepo, r.tenantServiceRepo),
		permissionService:         iam.NewPermissionService(db, r.permissionRepo, r.apiRepo, r.roleRepo, r.clientRepo, appCache),
		tenantService:             tenant.NewTenantService(db, r.tenantRepo),
		tenantMemberService:       tenant.NewTenantMemberService(db, r.tenantMemberRepo, r.userRepo, r.tenantRepo),
		idpService:                idp.NewIdentityProviderService(db, r.idpRepo, r.tenantRepo, r.userRepo),
		clientService:             client.NewClientService(db, r.clientRepo, r.clientURIRepo, r.idpRepo, r.permissionRepo, r.clientPermissionRepo, r.clientAPIRepo, r.apiRepo, r.userRepo, r.tenantRepo, authEventSvc),
		roleService:               iam.NewRoleService(db, r.roleRepo, r.permissionRepo, r.rolePermissionRepo, r.userRepo, r.tenantRepo, appCache, authEventSvc),
		userService:               user.NewUserService(db, r.userRepo, r.userIdentityRepo, r.userRoleRepo, r.roleRepo, r.tenantRepo, r.idpRepo, r.clientRepo, r.userPoolRepo, appCache, r.userTokenRepo, r.securitySettingRepo, r.userPasswordHistoryRepo, authEventSvc),
		registerService:           authn.NewRegistrationService(db, r.clientRepo, r.userRepo, r.userRoleRepo, r.userTokenRepo, r.userIdentityRepo, r.roleRepo, r.inviteRepo, r.idpRepo, r.securitySettingRepo, r.userPasswordHistoryRepo),
		loginService:              authn.NewLoginService(db, r.clientRepo, r.userRepo, r.userTokenRepo, r.userIdentityRepo, r.idpRepo, authEventSvc, sessionSvc, r.securitySettingRepo),
		sessionService:            sessionSvc,
		profileService:            user.NewProfileService(db, r.profileRepo, r.userRepo),
		userSettingService:        user.NewUserSettingService(db, r.userSettingRepo, r.userRepo),
		inviteService:             invite.NewInviteService(db, r.inviteRepo, r.clientRepo, r.roleRepo, r.emailTemplateRepo),
		forgotPasswordService:     authn.NewForgotPasswordService(db, r.userRepo, r.userTokenRepo, r.clientRepo, r.emailTemplateRepo),
		resetPasswordService:      authn.NewResetPasswordService(db, r.userRepo, r.userTokenRepo, r.clientRepo, r.securitySettingRepo, r.userPasswordHistoryRepo),
		emailVerificationService:  authn.NewEmailVerificationService(db, r.userRepo, r.userTokenRepo, r.clientRepo, r.emailTemplateRepo),
		magicLinkService:          authn.NewMagicLinkService(db, r.userRepo, r.userTokenRepo, r.clientRepo, r.userIdentityRepo, r.idpRepo, r.emailTemplateRepo),
		setupService:              setup.NewSetupService(db, r.userRepo, r.tenantRepo, r.tenantMemberRepo, r.clientRepo, r.idpRepo, r.roleRepo, r.userRoleRepo, r.userTokenRepo, r.userIdentityRepo, r.profileRepo),
		signupFlowService:         idp.NewSignupFlowService(db, r.signupFlowRepo, r.signupFlowRoleRepo, r.roleRepo, r.clientRepo),
		policyService:             iam.NewPolicyService(db, r.policyRepo, r.serviceRepo, r.apiRepo),
		apiKeyService:             client.NewAPIKeyService(db, r.apiKeyRepo, r.apiKeyAPIRepo, r.apiKeyPermissionRepo, r.apiRepo, r.userRepo, r.permissionRepo),
		securitySettingService:    secpolicy.NewSecuritySettingService(db, r.securitySettingRepo, r.securitySettingsAuditRepo),
		ipRestrictionRuleService:  secpolicy.NewIPRestrictionRuleService(db, r.ipRestrictionRuleRepo),
		emailTemplateService:      branding.NewEmailTemplateService(db, r.emailTemplateRepo),
		smsTemplateService:        branding.NewSMSTemplateService(db, r.smsTemplateRepo),
		loginTemplateService:      branding.NewLoginTemplateService(r.loginTemplateRepo),
		brandingService:           branding.NewBrandingService(r.brandingRepo),
		tenantSettingService:      tenant.NewTenantSettingService(r.tenantSettingRepo),
		emailConfigService:        notifier.NewEmailConfigService(r.emailConfigRepo),
		smsConfigService:          notifier.NewSMSConfigService(r.smsConfigRepo),
		webhookEndpointService:    webhook.NewWebhookEndpointService(r.webhookEndpointRepo),
		authEventService:          authEventSvc,
		oauthAuthorizeService:     oauth.NewOAuthAuthorizeService(db, r.clientRepo, r.clientURIRepo, r.oauthAuthCodeRepo, r.oauthConsentGrantRepo, r.oauthConsentChallengeRepo, authEventSvc),
		oauthTokenService:         oauth.NewOAuthTokenService(db, r.clientRepo, r.oauthAuthCodeRepo, r.oauthRefreshTokenRepo, r.userRepo, r.userIdentityRepo, authEventSvc),
		oauthConsentService:       oauth.NewOAuthConsentService(r.oauthConsentGrantRepo),
		oauthPARService:           oauth.NewOAuthPARService(db, r.clientRepo, r.clientURIRepo, r.oauthPARRequestRepo, authEventSvc),
		oauthDeviceService:        oauth.NewOAuthDeviceService(db, r.clientRepo, r.oauthDeviceCodeRepo, r.userRepo, r.userIdentityRepo, authEventSvc),
		oauthTokenExchangeService: oauth.NewOAuthTokenExchangeService(db, r.clientRepo, r.userRepo, authEventSvc),
		oauthSessionService:       oauth.NewOAuthSessionService(r.clientRepo, r.userRepo, r.oauthRefreshTokenRepo, authEventSvc),
		oauthCIBAService:          oauth.NewOAuthCIBAService(db, r.clientRepo, r.oauthCIBARequestRepo, r.userRepo, authEventSvc),
		oauthRegisterService:      oauth.NewOAuthRegisterService(db, r.clientRepo, r.clientURIRepo, r.tenantRepo, authEventSvc),
		accountService:            user.NewAccountService(db, r.userRepo, r.userTokenRepo, r.profileRepo, r.userSettingRepo, r.roleRepo, r.clientRepo, r.userBackupCodeRepo, r.userIdentityRepo, r.idpRepo, authEventSvc),
		smsLoginService:           authn.NewSMSLoginService(db, r.userRepo, r.smsOtpRepo, r.clientRepo, r.userIdentityRepo, r.idpRepo, authEventSvc),
		mfaService:                mfa.NewMFAService(db, r.userRepo, r.totpSecretRepo, r.webAuthnCredRepo, r.userBackupCodeRepo, r.securitySettingRepo, authEventSvc),
		federationService:         idp.NewFederationService(db, r.userRepo, r.userIdentityRepo, r.idpRepo, r.clientRepo, r.userRoleRepo, r.roleRepo, authEventSvc),
	}

	waSvc, err := mfa.NewWebAuthnService(db, r.userRepo, r.webAuthnCredRepo, appCache, authEventSvc)
	if err != nil {
		return nil, err
	}
	s.webAuthnService = waSvc
	return s, nil
}
