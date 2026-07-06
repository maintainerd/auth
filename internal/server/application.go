package server

import (
	"github.com/maintainerd/maintainerd-auth/internal/auditlog"
	"github.com/maintainerd/maintainerd-auth/internal/authevent"
	"github.com/maintainerd/maintainerd-auth/internal/authn"
	"github.com/maintainerd/maintainerd-auth/internal/branding"
	"github.com/maintainerd/maintainerd-auth/internal/client"
	"github.com/maintainerd/maintainerd-auth/internal/event"
	"github.com/maintainerd/maintainerd-auth/internal/iam"
	"github.com/maintainerd/maintainerd-auth/internal/idp"
	"github.com/maintainerd/maintainerd-auth/internal/invite"
	"github.com/maintainerd/maintainerd-auth/internal/mfa"
	"github.com/maintainerd/maintainerd-auth/internal/notifier"
	"github.com/maintainerd/maintainerd-auth/internal/oauth"
	"github.com/maintainerd/maintainerd-auth/internal/platform/cache"
	"github.com/maintainerd/maintainerd-auth/internal/secpolicy"
	"github.com/maintainerd/maintainerd-auth/internal/setup"
	"github.com/maintainerd/maintainerd-auth/internal/tenant"
	"github.com/maintainerd/maintainerd-auth/internal/user"
	"github.com/maintainerd/maintainerd-auth/internal/webhook"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

// Application is the server package's dependency bundle. It intentionally
// mirrors the service surface the HTTP and gRPC transports need without
// depending on the app package's broader wiring graph.
type Application struct {
	DB          *gorm.DB
	RedisClient *redis.Client
	Cache       *cache.Cache

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
	UserConsentService           user.UserConsentService
	UserTrustedDeviceService     user.UserTrustedDeviceService
	UserRepo                     user.UserRepository
	InviteService                invite.InviteService
	ForgotPasswordService        authn.ForgotPasswordService
	ResetPasswordService         authn.ResetPasswordService
	EmailVerificationService     authn.EmailVerificationService
	MagicLinkService             authn.MagicLinkService
	SetupService                 setup.SetupService
	RegistrationFlowService      idp.RegistrationFlowService
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
	EventTypeService             event.EventTypeService
	TenantEventTypeConfigService event.TenantEventTypeConfigService
	EventRouteService            event.EventRouteService
	// Webhook management wiring (subscription + replay handlers, endpoint repo
	// for the per-tenant creation cap).
	WebhookEndpointRepo        webhook.WebhookEndpointRepository
	WebhookSubscriptionHandler *webhook.SubscriptionHandler
	WebhookReplayHandler       *webhook.ReplayHandler
	IPRestrictionRuleRepo      secpolicy.IPRestrictionRuleRepository
	AuditLogger                auditlog.ManagementAuditLogger
	AuditLogRepo               auditlog.ManagementAuditLogRepository
}
