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
	iamTenantServiceRepo := newIAMTenantServiceRepo(db)
	iamClientRepo := newIAMClientRepo(db)
	iamTenantRepo := newIAMTenantRepo(db)
	iamUserRepo := newIAMUserRepo(db)
	clientAPIRepo := newClientAPIRepo(db)
	clientPermissionRepo := newClientPermissionRepo(db)
	clientIDPRepo := newClientIDPRepo(db)
	clientTenantRepo := newClientTenantRepo(db)
	clientUserRepo := newClientUserRepo(db)
	userTenantRepo := newUserTenantRepo(db)
	userRoleRepo := newUserRoleRepo(db)
	userClientRepo := newUserClientRepo(db)
	userIDPRepo := newUserIDPRepo(db)
	userBackupCodeRepo := newUserBackupCodeRepo(db)
	idpTenantRepo := newIDPTenantRepo(db)
	idpUserRepo := newIDPUserRepo(db)
	idpUserIdentityRepo := newIDPUserIdentityRepo(db)
	idpClientRepo := newIDPClientRepo(db)
	idpUserRoleRepo := newIDPUserRoleRepo(db)
	idpRoleRepo := newIDPRoleRepo(db)
	mfaUserRepo := newMFAUserRepo(db)
	oauthClientRepo := newOAuthClientRepo(db)
	oauthClientURIRepo := newOAuthClientURIRepo(db)
	oauthTenantRepo := newOAuthTenantRepo(db)
	oauthUserRepo := newOAuthUserRepo(db)
	oauthUserIdentityRepo := newOAuthUserIdentityRepo(db)

	webAuthnSvc, err := mfa.NewWebAuthnService(db, mfaUserRepo, r.webAuthnCredRepo, appCache, authEventSvc)
	if err != nil {
		return nil, err
	}

	s := &svcs{
		serviceService:            iam.NewServiceService(db, r.serviceRepo, iamTenantServiceRepo, r.apiRepo, r.servicePolicyRepo, r.policyRepo),
		apiService:                iam.NewAPIService(db, r.apiRepo, r.serviceRepo, iamTenantServiceRepo),
		permissionService:         iam.NewPermissionService(db, r.permissionRepo, r.apiRepo, r.roleRepo, iamClientRepo, appCache),
		tenantService:             tenant.NewTenantService(db, r.tenantRepo, tenantCascadeModels()),
		tenantMemberService:       tenant.NewTenantMemberService(db, r.tenantMemberRepo, newTenantUserReader(r.userRepo), r.tenantRepo),
		idpService:                idp.NewIdentityProviderService(db, r.idpRepo, idpTenantRepo, idpUserRepo),
		clientService:             client.NewClientService(db, r.clientRepo, r.clientURIRepo, clientIDPRepo, clientPermissionRepo, r.clientPermissionRepo, r.clientAPIRepo, clientAPIRepo, clientUserRepo, clientTenantRepo, authEventSvc),
		roleService:               iam.NewRoleService(db, r.roleRepo, r.permissionRepo, r.rolePermissionRepo, iamUserRepo, iamTenantRepo, appCache, authEventSvc),
		userService:               user.NewUserService(db, r.userRepo, r.userIdentityRepo, r.userRoleRepo, userRoleRepo, userTenantRepo, userIDPRepo, userClientRepo, r.userPoolRepo, appCache, r.userTokenRepo, r.securitySettingRepo, r.userPasswordHistoryRepo, authEventSvc),
		registerService:           authn.NewRegistrationService(db, newAuthnClientRepoAdapter(r.clientRepo), newAuthnUserRepoAdapter(r.userRepo), newAuthnUserRoleRepoAdapter(r.userRoleRepo), newAuthnUserTokenRepoAdapter(r.userTokenRepo), newAuthnUserIdentityRepoAdapter(r.userIdentityRepo), newAuthnRoleRepoAdapter(r.roleRepo), newAuthnInviteRepoAdapter(r.inviteRepo), newAuthnIDPRepoAdapter(r.idpRepo), r.securitySettingRepo, newAuthnPasswordHistoryRepoAdapter(r.userPasswordHistoryRepo)),
		loginService:              authn.NewLoginService(db, newAuthnClientRepoAdapter(r.clientRepo), newAuthnUserRepoAdapter(r.userRepo), newAuthnUserTokenRepoAdapter(r.userTokenRepo), newAuthnUserIdentityRepoAdapter(r.userIdentityRepo), newAuthnIDPRepoAdapter(r.idpRepo), authEventSvc, sessionSvc, r.securitySettingRepo),
		sessionService:            sessionSvc,
		profileService:            user.NewProfileService(db, r.profileRepo, r.userRepo),
		userSettingService:        user.NewUserSettingService(db, r.userSettingRepo, r.userRepo),
		inviteService:             invite.NewInviteService(db, r.inviteRepo, newInviteClientRepo(db), newInviteRoleRepo(db), r.emailTemplateRepo),
		forgotPasswordService:     authn.NewForgotPasswordService(db, newAuthnUserRepoAdapter(r.userRepo), newAuthnUserTokenRepoAdapter(r.userTokenRepo), newAuthnClientRepoAdapter(r.clientRepo), r.emailTemplateRepo),
		resetPasswordService:      authn.NewResetPasswordService(db, newAuthnUserRepoAdapter(r.userRepo), newAuthnUserTokenRepoAdapter(r.userTokenRepo), newAuthnClientRepoAdapter(r.clientRepo), r.securitySettingRepo, newAuthnPasswordHistoryRepoAdapter(r.userPasswordHistoryRepo)),
		emailVerificationService:  authn.NewEmailVerificationService(db, newAuthnUserRepoAdapter(r.userRepo), newAuthnUserTokenRepoAdapter(r.userTokenRepo), newAuthnClientRepoAdapter(r.clientRepo), r.emailTemplateRepo),
		magicLinkService:          authn.NewMagicLinkService(db, newAuthnUserRepoAdapter(r.userRepo), newAuthnUserTokenRepoAdapter(r.userTokenRepo), newAuthnClientRepoAdapter(r.clientRepo), newAuthnUserIdentityRepoAdapter(r.userIdentityRepo), newAuthnIDPRepoAdapter(r.idpRepo), r.emailTemplateRepo),
		setupService:              setup.NewSetupService(db, r.userRepo, r.tenantRepo, r.tenantMemberRepo, r.clientRepo, r.roleRepo, r.userRoleRepo, r.userIdentityRepo, r.profileRepo),
		signupFlowService:         idp.NewSignupFlowService(db, r.signupFlowRepo, r.signupFlowRoleRepo, idpRoleRepo, idpClientRepo),
		policyService:             iam.NewPolicyService(db, r.policyRepo, r.serviceRepo, r.apiRepo),
		apiKeyService:             client.NewAPIKeyService(db, r.apiKeyRepo, r.apiKeyAPIRepo, r.apiKeyPermissionRepo, clientAPIRepo, clientUserRepo, clientPermissionRepo),
		securitySettingService:    secpolicy.NewSecuritySettingService(db, r.securitySettingRepo, r.securitySettingsAuditRepo),
		ipRestrictionRuleService:  secpolicy.NewIPRestrictionRuleService(db, r.ipRestrictionRuleRepo),
		emailTemplateService:      branding.NewEmailTemplateService(r.emailTemplateRepo),
		smsTemplateService:        branding.NewSMSTemplateService(r.smsTemplateRepo),
		loginTemplateService:      branding.NewLoginTemplateService(r.loginTemplateRepo),
		brandingService:           branding.NewBrandingService(r.brandingRepo),
		tenantSettingService:      tenant.NewTenantSettingService(r.tenantSettingRepo),
		emailConfigService:        notifier.NewEmailConfigService(r.emailConfigRepo),
		smsConfigService:          notifier.NewSMSConfigService(r.smsConfigRepo),
		webhookEndpointService:    webhook.NewWebhookEndpointService(r.webhookEndpointRepo),
		authEventService:          authEventSvc,
		oauthAuthorizeService:     oauth.NewOAuthAuthorizeService(db, oauthClientRepo, oauthClientURIRepo, r.oauthAuthCodeRepo, r.oauthConsentGrantRepo, r.oauthConsentChallengeRepo, authEventSvc),
		oauthTokenService:         oauth.NewOAuthTokenService(db, oauthClientRepo, r.oauthAuthCodeRepo, r.oauthRefreshTokenRepo, oauthUserRepo, oauthUserIdentityRepo, authEventSvc),
		oauthConsentService:       oauth.NewOAuthConsentService(r.oauthConsentGrantRepo),
		oauthPARService:           oauth.NewOAuthPARService(db, oauthClientRepo, oauthClientURIRepo, r.oauthPARRequestRepo, authEventSvc),
		oauthDeviceService:        oauth.NewOAuthDeviceService(db, oauthClientRepo, r.oauthDeviceCodeRepo, oauthUserRepo, oauthUserIdentityRepo, authEventSvc),
		oauthTokenExchangeService: oauth.NewOAuthTokenExchangeService(db, oauthClientRepo, oauthUserRepo, authEventSvc),
		oauthSessionService:       oauth.NewOAuthSessionService(db, oauthClientRepo, oauthUserRepo, r.oauthRefreshTokenRepo, authEventSvc),
		oauthCIBAService:          oauth.NewOAuthCIBAService(db, oauthClientRepo, r.oauthCIBARequestRepo, oauthUserRepo, authEventSvc),
		oauthRegisterService:      oauth.NewOAuthRegisterService(db, oauthClientRepo, oauthClientURIRepo, oauthTenantRepo, authEventSvc),
		accountService:            user.NewAccountService(db, r.userRepo, r.userTokenRepo, r.profileRepo, r.userSettingRepo, userRoleRepo, userClientRepo, userBackupCodeRepo, r.userIdentityRepo, userIDPRepo, authEventSvc),
		smsLoginService:           authn.NewSMSLoginService(db, newAuthnUserRepoAdapter(r.userRepo), r.smsOtpRepo, newAuthnClientRepoAdapter(r.clientRepo), newAuthnUserIdentityRepoAdapter(r.userIdentityRepo), newAuthnIDPRepoAdapter(r.idpRepo), authEventSvc),
		mfaService:                mfa.NewMFAService(db, mfaUserRepo, r.totpSecretRepo, r.webAuthnCredRepo, r.userBackupCodeRepo, r.securitySettingRepo, authEventSvc),
		webAuthnService:           webAuthnSvc,
		federationService:         idp.NewFederationService(db, idpUserRepo, idpUserIdentityRepo, r.idpRepo, idpClientRepo, idpUserRoleRepo, idpRoleRepo, authEventSvc),
	}
	return s, nil
}

func tenantCascadeModels() []any {
	return []any{
		&oauth.OAuthCIBARequest{},
		&oauth.OAuthDeviceCode{},
		&oauth.OAuthPARRequest{},
		&oauth.OAuthConsentChallenge{},
		&oauth.OAuthConsentGrant{},
		&oauth.OAuthRefreshToken{},
		&oauth.OAuthAuthorizationCode{},

		&webhook.WebhookEndpoint{},
		&notifier.SMSConfig{},
		&notifier.EmailConfig{},
		&branding.SMSTemplate{},
		&branding.LoginTemplate{},
		&branding.EmailTemplate{},
		&branding.Branding{},

		&secpolicy.SecuritySettingsAudit{},
		&secpolicy.IPRestrictionRule{},
		&secpolicy.SecuritySetting{},

		&invite.Invite{},
		&idp.SignupFlow{},
		&idp.IdentityProvider{},

		&client.ClientURI{},
		&client.APIKey{},
		&client.Client{},

		&iam.Permission{},
		&iam.API{},
		&iam.Policy{},
		&iam.Role{},
		&iam.Service{},
		&iam.TenantService{},

		&user.UserIdentity{},
		&user.UserPool{},

		&tenant.TenantServiceLink{},
		&tenant.TenantSetting{},
		&tenant.TenantMember{},
	}
}
