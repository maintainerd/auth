package server

import (
	"github.com/maintainerd/auth/internal/authevent"
	"github.com/maintainerd/auth/internal/authn"
	"github.com/maintainerd/auth/internal/branding"
	"github.com/maintainerd/auth/internal/client"
	"github.com/maintainerd/auth/internal/dashboard"
	"github.com/maintainerd/auth/internal/event"
	"github.com/maintainerd/auth/internal/iam"
	"github.com/maintainerd/auth/internal/idp"
	"github.com/maintainerd/auth/internal/invite"
	"github.com/maintainerd/auth/internal/mfa"
	"github.com/maintainerd/auth/internal/notifier"
	"github.com/maintainerd/auth/internal/oauth"
	"github.com/maintainerd/auth/internal/secpolicy"
	"github.com/maintainerd/auth/internal/setup"
	"github.com/maintainerd/auth/internal/tenant"
	"github.com/maintainerd/auth/internal/user"
	"github.com/maintainerd/auth/internal/webhook"
)

// handlers holds every REST handler instance. Created once per server start.
type handlers struct {
	service            *iam.ServiceHandler
	api                *iam.APIHandler
	permission         *iam.PermissionHandler
	policy             *iam.PolicyHandler
	tenant             *tenant.TenantHandler
	identityProvider   *idp.IdentityProviderHandler
	client             *client.ClientHandler
	role               *iam.RoleHandler
	user               *user.UserHandler
	register           *authn.RegisterHandler
	login              *authn.LoginHandler
	profile            *user.ProfileHandler
	userSetting        *user.UserSettingHandler
	invite             *invite.InviteHandler
	forgotPassword     *authn.ForgotPasswordHandler
	resetPassword      *authn.ResetPasswordHandler
	emailVerification  *authn.EmailVerificationHandler
	magicLink          *authn.MagicLinkHandler
	setup              *setup.SetupHandler
	apiKey             *client.APIKeyHandler
	authFlow           *idp.AuthFlowHandler
	securitySetting    *secpolicy.SecuritySettingHandler
	ipRestrictionRule  *secpolicy.IPRestrictionRuleHandler
	emailTemplate      *branding.EmailTemplateHandler
	smsTemplate        *branding.SMSTemplateHandler
	branding           *branding.BrandingHandler
	tenantSetting      *tenant.TenantSettingHandler
	emailConfig        *notifier.EmailConfigHandler
	smsConfig          *notifier.SMSConfigHandler
	webhookEndpoint    *webhook.WebhookEndpointHandler
	authEvent          *authevent.AuthEventHandler
	authorization      *iam.AuthorizationHandler
	oauthAuthorize     *oauth.OAuthAuthorizeHandler
	oauthConnections   *oauth.OAuthConnectionsHandler
	oauthToken         *oauth.OAuthTokenHandler
	oauthTokenExchange *oauth.OAuthTokenExchangeHandler
	oauthConsent       *oauth.OAuthConsentHandler
	oauthDiscovery     *oauth.OAuthDiscoveryHandler
	oauthUserInfo      *oauth.OAuthUserInfoHandler
	oauthPAR           *oauth.OAuthPARHandler
	oauthDevice        *oauth.OAuthDeviceHandler
	oauthSession       *oauth.OAuthSessionHandler
	oauthCIBA          *oauth.OAuthCIBAHandler
	oauthRegister      *oauth.OAuthRegisterHandler
	account            *user.AccountHandler
	smsLogin           *authn.SMSLoginHandler
	mfa                *mfa.MFAHandler
	federation         *idp.FederationHandler
	eventConfig        *event.ConfigHandler
	eventManagement    *event.ManagementHandler
	dashboard          *dashboard.Handler
}

func initHandlers(application *Application) *handlers {
	return &handlers{
		service:            iam.NewServiceHandler(application.ServiceService),
		api:                iam.NewAPIHandler(application.APIService),
		permission:         iam.NewPermissionHandler(application.PermissionService),
		policy:             iam.NewPolicyHandler(application.PolicyService),
		tenant:             tenant.NewTenantHandler(application.TenantService, application.TenantMemberService, application.BrandingService, application.SecuritySettingService),
		identityProvider:   idp.NewIdentityProviderHandler(application.IdentityProviderService),
		client:             client.NewClientHandler(application.ClientService),
		role:               iam.NewRoleHandler(application.RoleService),
		user:               user.NewUserHandler(application.UserService),
		register:           authn.NewRegisterHandler(application.RegisterService),
		login:              authn.NewLoginHandler(application.LoginService),
		profile:            user.NewProfileHandler(application.ProfileService),
		userSetting:        user.NewUserSettingHandler(application.UserSettingService),
		invite:             invite.NewInviteHandler(application.InviteService),
		forgotPassword:     authn.NewForgotPasswordHandler(application.ForgotPasswordService),
		resetPassword:      authn.NewResetPasswordHandler(application.ResetPasswordService),
		emailVerification:  authn.NewEmailVerificationHandler(application.EmailVerificationService),
		magicLink:          authn.NewMagicLinkHandler(application.MagicLinkService),
		setup:              setup.NewSetupHandler(application.SetupService),
		apiKey:             client.NewAPIKeyHandler(application.APIKeyService),
		authFlow:           idp.NewAuthFlowHandler(application.AuthFlowService),
		securitySetting:    secpolicy.NewSecuritySettingHandler(application.SecuritySettingService),
		ipRestrictionRule:  secpolicy.NewIPRestrictionRuleHandler(application.IPRestrictionRuleService),
		emailTemplate:      branding.NewEmailTemplateHandler(application.EmailTemplateService),
		smsTemplate:        branding.NewSMSTemplateHandler(application.SMSTemplateService),
		branding:           branding.NewBrandingHandler(application.BrandingService),
		tenantSetting:      tenant.NewTenantSettingHandler(application.TenantSettingService, application.AuthEventService),
		emailConfig:        notifier.NewEmailConfigHandler(application.EmailConfigService),
		smsConfig:          notifier.NewSMSConfigHandler(application.SMSConfigService),
		webhookEndpoint:    webhook.NewWebhookEndpointHandler(application.WebhookEndpointService),
		authEvent:          authevent.NewAuthEventHandler(application.AuthEventService),
		authorization:      iam.NewAuthorizationHandler(application.AuthorizationService),
		oauthAuthorize:     oauth.NewOAuthAuthorizeHandler(application.OAuthAuthorizeService),
		oauthConnections:   oauth.NewOAuthConnectionsHandler(application.OAuthConnectionsService),
		oauthToken:         oauth.NewOAuthTokenHandler(application.OAuthTokenService, nil, nil),
		oauthTokenExchange: oauth.NewOAuthTokenExchangeHandler(application.OAuthTokenExchangeService),
		oauthConsent:       oauth.NewOAuthConsentHandler(application.OAuthConsentService),
		oauthDiscovery:     oauth.NewOAuthDiscoveryHandler(),
		oauthUserInfo:      oauth.NewOAuthUserInfoHandler(),
		oauthPAR:           oauth.NewOAuthPARHandler(application.OAuthPARService),
		oauthDevice:        oauth.NewOAuthDeviceHandler(application.OAuthDeviceService),
		oauthSession:       oauth.NewOAuthSessionHandler(application.OAuthSessionService),
		oauthCIBA:          oauth.NewOAuthCIBAHandler(application.OAuthCIBAService),
		oauthRegister:      oauth.NewOAuthRegisterHandler(application.OAuthRegisterService),
		account:            user.NewAccountHandler(application.AccountService, newUserSessionServiceAdapter(application.SessionService), application.ProfileRepo),
		smsLogin:           authn.NewSMSLoginHandler(application.SMSLoginService),
		mfa:                mfa.NewMFAHandler(application.MFAService, application.WebAuthnService),
		federation:         idp.NewFederationHandler(application.FederationService),
		eventConfig:        event.NewConfigHandler(application.EventTypeService, application.TenantEventTypeConfigService),
		eventManagement:    event.NewManagementHandler(application.EventRouteService),
		dashboard:          dashboard.NewHandler(dashboard.NewService(application.DB)),
	}
}
