package app

import (
	"github.com/maintainerd/auth/internal/authevent"
	"github.com/maintainerd/auth/internal/branding"
	"github.com/maintainerd/auth/internal/client"
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
	"gorm.io/gorm"
)

// repos holds every repository instance. It is private to the app package and
// is only passed between the three init functions below.
type repos struct {
	serviceRepo               iam.ServiceRepository
	apiRepo                   iam.APIRepository
	permissionRepo            iam.PermissionRepository
	tenantRepo                tenant.TenantRepository
	tenantMemberRepo          tenant.TenantMemberRepository
	idpRepo                   idp.IdentityProviderRepository
	roleRepo                  iam.RoleRepository
	rolePermissionRepo        iam.RolePermissionRepository
	clientRepo                client.ClientRepository
	clientPermissionRepo      client.ClientPermissionRepository
	clientAPIRepo             client.ClientAPIRepository
	clientURIRepo             client.ClientURIRepository
	userRepo                  user.UserRepository
	userIdentityRepo          user.UserIdentityRepository
	userRoleRepo              user.UserRoleRepository
	userTokenRepo             user.UserTokenRepository
	profileRepo               user.ProfileRepository
	userSettingRepo           user.UserSettingRepository
	inviteRepo                invite.InviteRepository
	emailTemplateRepo         branding.EmailTemplateRepository
	smsTemplateRepo           branding.SMSTemplateRepository
	policyRepo                iam.PolicyRepository
	servicePolicyRepo         iam.ServicePolicyRepository
	apiKeyRepo                client.APIKeyRepository
	apiKeyAPIRepo             client.APIKeyAPIRepository
	apiKeyPermissionRepo      client.APIKeyPermissionRepository
	authFlowRepo            idp.AuthFlowRepository
	authFlowRoleRepo        idp.AuthFlowRoleRepository
	authFlowCallbackURIRepo idp.AuthFlowCallbackURIRepository
	securitySettingRepo       secpolicy.SecuritySettingRepository
	securitySettingsAuditRepo secpolicy.SecuritySettingsAuditRepository
	ipRestrictionRuleRepo     secpolicy.IPRestrictionRuleRepository
	brandingRepo              branding.BrandingRepository
	tenantSettingRepo         tenant.TenantSettingRepository
	emailConfigRepo           notifier.EmailConfigRepository
	smsConfigRepo             notifier.SMSConfigRepository
	webhookEndpointRepo       webhook.WebhookEndpointRepository
	webhookEndpointEventRepo  webhook.WebhookEndpointEventRepository
	deliveryHistoryRepo       webhook.DeliveryHistoryRepository
	authEventRepo             authevent.AuthEventRepository
	eventTypeRepo             event.EventTypeRepository
	eventRouteRepo            event.EventRouteRepository
	tenantEventTypeRepo       event.TenantEventTypeRepository
	outboxRepo                event.OutboxRepository
	oauthAuthCodeRepo         oauth.OAuthAuthorizationCodeRepository
	oauthRefreshTokenRepo     oauth.OAuthRefreshTokenRepository
	oauthConsentGrantRepo     oauth.OAuthConsentGrantRepository
	oauthConsentChallengeRepo oauth.OAuthConsentChallengeRepository
	oauthPARRequestRepo       oauth.OAuthPARRequestRepository
	oauthDeviceCodeRepo       oauth.OAuthDeviceCodeRepository
	oauthCIBARequestRepo      oauth.OAuthCIBARequestRepository
	smsOtpRepo                notifier.UserOTPRepository
	smsPhoneRepo              mfa.UserSMSPhoneRepository
	userBackupCodeRepo        mfa.UserBackupCodeRepository
	totpSecretRepo            mfa.UserTOTPSecretRepository
	webAuthnCredRepo          mfa.UserWebAuthnCredentialRepository
	userPasswordHistoryRepo   user.UserPasswordHistoryRepository
	setupStateRepo            setup.SetupStateRepository
}

func initRepos(db *gorm.DB) *repos {
	return &repos{
		serviceRepo:               iam.NewServiceRepository(db),
		apiRepo:                   iam.NewAPIRepository(db),
		permissionRepo:            iam.NewPermissionRepository(db),
		tenantRepo:                tenant.NewTenantRepository(db),
		tenantMemberRepo:          tenant.NewTenantMemberRepository(db),
		idpRepo:                   idp.NewIdentityProviderRepository(db),
		roleRepo:                  iam.NewRoleRepository(db),
		rolePermissionRepo:        iam.NewRolePermissionRepository(db),
		clientRepo:                client.NewClientRepository(db),
		clientPermissionRepo:      client.NewClientPermissionRepository(db),
		clientAPIRepo:             client.NewClientAPIRepository(db),
		clientURIRepo:             client.NewClientURIRepository(db),
		userRepo:                  user.NewUserRepository(db),
		userIdentityRepo:          user.NewUserIdentityRepository(db),
		userRoleRepo:              user.NewUserRoleRepository(db),
		userTokenRepo:             user.NewUserTokenRepository(db),
		profileRepo:               user.NewProfileRepository(db),
		userSettingRepo:           user.NewUserSettingRepository(db),
		inviteRepo:                invite.NewInviteRepository(db),
		emailTemplateRepo:         branding.NewEmailTemplateRepository(db),
		smsTemplateRepo:           branding.NewSMSTemplateRepository(db),
		policyRepo:                iam.NewPolicyRepository(db),
		servicePolicyRepo:         iam.NewServicePolicyRepository(db),
		apiKeyRepo:                client.NewAPIKeyRepository(db),
		apiKeyAPIRepo:             client.NewAPIKeyAPIRepository(db),
		apiKeyPermissionRepo:      client.NewAPIKeyPermissionRepository(db),
		authFlowRepo:            idp.NewAuthFlowRepository(db),
		authFlowRoleRepo:        idp.NewAuthFlowRoleRepository(db),
		authFlowCallbackURIRepo: idp.NewAuthFlowCallbackURIRepository(db),
		securitySettingRepo:       secpolicy.NewSecuritySettingRepository(db),
		securitySettingsAuditRepo: secpolicy.NewSecuritySettingsAuditRepository(db),
		ipRestrictionRuleRepo:     secpolicy.NewIPRestrictionRuleRepository(db),
		brandingRepo:              branding.NewBrandingRepository(db),
		tenantSettingRepo:         tenant.NewTenantSettingRepository(db),
		emailConfigRepo:           notifier.NewEmailConfigRepository(db),
		smsConfigRepo:             notifier.NewSMSConfigRepository(db),
		webhookEndpointRepo:       webhook.NewWebhookEndpointRepository(db),
		webhookEndpointEventRepo:  webhook.NewWebhookEndpointEventRepository(db),
		deliveryHistoryRepo:       webhook.NewDeliveryHistoryRepository(db),
		authEventRepo:             authevent.NewAuthEventRepository(db),
		eventTypeRepo:             event.NewEventTypeRepository(db),
		eventRouteRepo:            event.NewEventRouteRepository(db),
		tenantEventTypeRepo:       event.NewTenantEventTypeRepository(db),
		outboxRepo:                event.NewOutboxRepository(db),
		oauthAuthCodeRepo:         oauth.NewOAuthAuthorizationCodeRepository(db),
		oauthRefreshTokenRepo:     oauth.NewOAuthRefreshTokenRepository(db),
		oauthConsentGrantRepo:     oauth.NewOAuthConsentGrantRepository(db),
		oauthConsentChallengeRepo: oauth.NewOAuthConsentChallengeRepository(db),
		oauthPARRequestRepo:       oauth.NewOAuthPARRequestRepository(db),
		oauthDeviceCodeRepo:       oauth.NewOAuthDeviceCodeRepository(db),
		oauthCIBARequestRepo:      oauth.NewOAuthCIBARequestRepository(db),
		smsOtpRepo:                notifier.NewUserOTPRepository(db),
		smsPhoneRepo:              mfa.NewUserSMSPhoneRepository(db),
		userBackupCodeRepo:        mfa.NewUserBackupCodeRepository(db),
		totpSecretRepo:            mfa.NewUserTOTPSecretRepository(db),
		webAuthnCredRepo:          mfa.NewUserWebAuthnCredentialRepository(db),
		userPasswordHistoryRepo:   user.NewUserPasswordHistoryRepository(db),
		setupStateRepo:            setup.NewSetupStateRepository(db),
	}
}
