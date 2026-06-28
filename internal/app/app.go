package app

import (
	"fmt"

	"github.com/maintainerd/auth/internal/authevent"
	"github.com/maintainerd/auth/internal/authn"
	"github.com/maintainerd/auth/internal/branding"
	"github.com/maintainerd/auth/internal/client"
	"github.com/maintainerd/auth/internal/event"
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
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

type App struct {
	DB          *gorm.DB
	RedisClient *redis.Client
	Cache       *cache.Cache
	// Services
	ServiceService               iam.ServiceService
	APIService                   iam.APIService
	PermissionService            iam.PermissionService
	PolicyService                iam.PolicyService
	TenantService                tenant.TenantService
	TenantMemberService          tenant.TenantMemberService
	IdentityProviderService      idp.IdentityProviderService
	ClientService                client.ClientService
	RoleService                  iam.RoleService
	UserService                  user.UserService
	RegisterService              authn.RegisterService
	LoginService                 authn.LoginService
	ProfileService               user.ProfileService
	ProfileRepo                  user.ProfileRepository
	UserSettingService           user.UserSettingService
	InviteService                invite.InviteService
	ForgotPasswordService        authn.ForgotPasswordService
	ResetPasswordService         authn.ResetPasswordService
	EmailVerificationService     authn.EmailVerificationService
	MagicLinkService             authn.MagicLinkService
	SetupService                 setup.SetupService
	AuthFlowService              idp.AuthFlowService
	APIKeyService                client.APIKeyService
	SecuritySettingService       secpolicy.SecuritySettingService
	IPRestrictionRuleService     secpolicy.IPRestrictionRuleService
	EmailTemplateService         branding.EmailTemplateService
	SMSTemplateService           branding.SMSTemplateService
	BrandingService              branding.BrandingService
	TenantSettingService         tenant.TenantSettingService
	EmailConfigService           notifier.EmailConfigService
	SMSConfigService             notifier.SMSConfigService
	WebhookEndpointService       webhook.WebhookEndpointService
	AuthEventService             authevent.AuthEventService
	AuthorizationService         iam.ServiceAuthorizationService
	OAuthAuthorizeService        oauth.OAuthAuthorizeService
	OAuthConnectionsService      oauth.OAuthConnectionsService
	OAuthTokenService            oauth.OAuthTokenService
	OAuthConsentService          oauth.OAuthConsentService
	OAuthPARService              oauth.OAuthPARService
	OAuthDeviceService           oauth.OAuthDeviceService
	OAuthTokenExchangeService    oauth.OAuthTokenExchangeService
	OAuthSessionService          oauth.OAuthSessionService
	OAuthCIBAService             oauth.OAuthCIBAService
	OAuthRegisterService         oauth.OAuthRegisterService
	AccountService               user.AccountService
	SessionService               authn.SessionService
	SMSLoginService              authn.SMSLoginService
	MFAService                   mfa.MFAService
	WebAuthnService              mfa.WebAuthnService
	FederationService            idp.FederationService
	IDPRepo                      idp.IdentityProviderRepository
	IDPAllowedAudienceRepo       idp.IdentityProviderAllowedAudienceRepository
	EventService                 event.EventService
	EventTypeService             event.EventTypeService
	TenantEventTypeConfigService event.TenantEventTypeConfigService
	EventRouteService            event.EventRouteService
	// Webhook management wiring used by the REST router.
	WebhookEndpointRepo        webhook.WebhookEndpointRepository
	WebhookSubscriptionHandler *webhook.SubscriptionHandler
	WebhookReplayHandler       *webhook.ReplayHandler
	IPRestrictionRuleRepo      secpolicy.IPRestrictionRuleRepository
}

// NewApp wires the full dependency graph in two focused steps:
//  1. initRepos    — every repository, bound to db
//  2. initServices — every service, consuming repos
//
// Handler creation is delegated to transport packages (rest, grpcserver).
func NewApp(db *gorm.DB, redisClient *redis.Client) (*App, error) {
	r := initRepos(db)
	appCache := cache.New(redisClient)
	s, err := initServices(db, r, appCache, redisClient)
	if err != nil {
		return nil, fmt.Errorf("service init failed: %w", err)
	}

	return &App{
		DB:          db,
		RedisClient: redisClient,
		Cache:       appCache,
		// Services
		ServiceService:               s.serviceService,
		APIService:                   s.apiService,
		PermissionService:            s.permissionService,
		PolicyService:                s.policyService,
		TenantService:                s.tenantService,
		TenantMemberService:          s.tenantMemberService,
		IdentityProviderService:      s.idpService,
		ClientService:                s.clientService,
		RoleService:                  s.roleService,
		UserService:                  s.userService,
		RegisterService:              s.registerService,
		LoginService:                 s.loginService,
		ProfileService:               s.profileService,
		ProfileRepo:                  s.profileRepo,
		UserSettingService:           s.userSettingService,
		InviteService:                s.inviteService,
		ForgotPasswordService:        s.forgotPasswordService,
		ResetPasswordService:         s.resetPasswordService,
		EmailVerificationService:     s.emailVerificationService,
		MagicLinkService:             s.magicLinkService,
		SetupService:                 s.setupService,
		AuthFlowService:              s.authFlowService,
		APIKeyService:                s.apiKeyService,
		SecuritySettingService:       s.securitySettingService,
		IPRestrictionRuleService:     s.ipRestrictionRuleService,
		EmailTemplateService:         s.emailTemplateService,
		SMSTemplateService:           s.smsTemplateService,
		BrandingService:              s.brandingService,
		TenantSettingService:         s.tenantSettingService,
		EmailConfigService:           s.emailConfigService,
		SMSConfigService:             s.smsConfigService,
		WebhookEndpointService:       s.webhookEndpointService,
		AuthEventService:             s.authEventService,
		AuthorizationService:         s.authorizationService,
		OAuthAuthorizeService:        s.oauthAuthorizeService,
		OAuthConnectionsService:      s.oauthConnectionsService,
		OAuthTokenService:            s.oauthTokenService,
		OAuthConsentService:          s.oauthConsentService,
		OAuthPARService:              s.oauthPARService,
		OAuthDeviceService:           s.oauthDeviceService,
		OAuthTokenExchangeService:    s.oauthTokenExchangeService,
		OAuthSessionService:          s.oauthSessionService,
		OAuthCIBAService:             s.oauthCIBAService,
		OAuthRegisterService:         s.oauthRegisterService,
		AccountService:               s.accountService,
		SessionService:               s.sessionService,
		SMSLoginService:              s.smsLoginService,
		MFAService:                   s.mfaService,
		WebAuthnService:              s.webAuthnService,
		FederationService:            s.federationService,
		IDPRepo:                      r.idpRepo,
		IDPAllowedAudienceRepo:       r.idpAllowedAudienceRepo,
		EventService:                 s.eventService,
		EventTypeService:             s.eventTypeService,
		TenantEventTypeConfigService: s.tenantEventTypeConfigService,
		EventRouteService:            s.eventRouteService,
		WebhookEndpointRepo:          s.webhookEndpointRepo,
		WebhookSubscriptionHandler:   s.webhookSubscriptionHandler,
		WebhookReplayHandler:         s.webhookReplayHandler,
		IPRestrictionRuleRepo:        s.ipRestrictionRuleRepo,
	}, nil
}
