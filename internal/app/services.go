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

	sessionSvc := authn.NewSessionService(newAuthnUserTokenRepoAdapter(r.userTokenRepo))

	s := &svcs{
		tenantService:            tenant.NewTenantService(db, r.tenantRepo, nil),
		registerService:          authn.NewRegistrationService(db, newAuthnClientRepoAdapter(r.clientRepo), newAuthnUserRepoAdapter(r.userRepo), newAuthnUserRoleRepoAdapter(r.userRoleRepo), newAuthnUserTokenRepoAdapter(r.userTokenRepo), newAuthnUserIdentityRepoAdapter(r.userIdentityRepo), newAuthnRoleRepoAdapter(r.roleRepo), newAuthnInviteRepoAdapter(r.inviteRepo), newAuthnIDPRepoAdapter(r.idpRepo), r.securitySettingRepo, newAuthnPasswordHistoryRepoAdapter(r.userPasswordHistoryRepo)),
		loginService:             authn.NewLoginService(db, newAuthnClientRepoAdapter(r.clientRepo), newAuthnUserRepoAdapter(r.userRepo), newAuthnUserTokenRepoAdapter(r.userTokenRepo), newAuthnUserIdentityRepoAdapter(r.userIdentityRepo), newAuthnIDPRepoAdapter(r.idpRepo), authEventSvc, sessionSvc, r.securitySettingRepo),
		sessionService:           sessionSvc,
		profileService:           user.NewProfileService(db, r.profileRepo, r.userRepo),
		userSettingService:       user.NewUserSettingService(db, r.userSettingRepo, r.userRepo),
		forgotPasswordService:    authn.NewForgotPasswordService(db, newAuthnUserRepoAdapter(r.userRepo), newAuthnUserTokenRepoAdapter(r.userTokenRepo), newAuthnClientRepoAdapter(r.clientRepo), r.emailTemplateRepo),
		resetPasswordService:     authn.NewResetPasswordService(db, newAuthnUserRepoAdapter(r.userRepo), newAuthnUserTokenRepoAdapter(r.userTokenRepo), newAuthnClientRepoAdapter(r.clientRepo), r.securitySettingRepo, newAuthnPasswordHistoryRepoAdapter(r.userPasswordHistoryRepo)),
		emailVerificationService: authn.NewEmailVerificationService(db, newAuthnUserRepoAdapter(r.userRepo), newAuthnUserTokenRepoAdapter(r.userTokenRepo), newAuthnClientRepoAdapter(r.clientRepo), r.emailTemplateRepo),
		magicLinkService:         authn.NewMagicLinkService(db, newAuthnUserRepoAdapter(r.userRepo), newAuthnUserTokenRepoAdapter(r.userTokenRepo), newAuthnClientRepoAdapter(r.clientRepo), newAuthnUserIdentityRepoAdapter(r.userIdentityRepo), newAuthnIDPRepoAdapter(r.idpRepo), r.emailTemplateRepo),
		setupService:             setup.NewSetupService(db, r.userRepo, r.tenantRepo, r.tenantMemberRepo, r.clientRepo, r.idpRepo, r.roleRepo, r.userRoleRepo, r.userTokenRepo, r.userIdentityRepo, r.profileRepo),
		policyService:            iam.NewPolicyService(db, r.policyRepo, r.serviceRepo, r.apiRepo),
		securitySettingService:   secpolicy.NewSecuritySettingService(db, r.securitySettingRepo, r.securitySettingsAuditRepo),
		ipRestrictionRuleService: secpolicy.NewIPRestrictionRuleService(db, r.ipRestrictionRuleRepo),
		emailTemplateService:     branding.NewEmailTemplateService(db, r.emailTemplateRepo),
		smsTemplateService:       branding.NewSMSTemplateService(db, r.smsTemplateRepo),
		loginTemplateService:     branding.NewLoginTemplateService(r.loginTemplateRepo),
		brandingService:          branding.NewBrandingService(r.brandingRepo),
		tenantSettingService:     tenant.NewTenantSettingService(r.tenantSettingRepo),
		emailConfigService:       notifier.NewEmailConfigService(r.emailConfigRepo),
		smsConfigService:         notifier.NewSMSConfigService(r.smsConfigRepo),
		webhookEndpointService:   webhook.NewWebhookEndpointService(r.webhookEndpointRepo),
		authEventService:         authEventSvc,
		oauthConsentService:      oauth.NewOAuthConsentService(r.oauthConsentGrantRepo),
		smsLoginService:          authn.NewSMSLoginService(db, newAuthnUserRepoAdapter(r.userRepo), r.smsOtpRepo, newAuthnClientRepoAdapter(r.clientRepo), newAuthnUserIdentityRepoAdapter(r.userIdentityRepo), newAuthnIDPRepoAdapter(r.idpRepo), authEventSvc),
	}
	_ = appCache
	return s, nil
}
