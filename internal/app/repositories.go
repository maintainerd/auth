package app

import (
	"github.com/maintainerd/maintainerd-auth/internal/auditlog"
	"github.com/maintainerd/maintainerd-auth/internal/authevent"
	"github.com/maintainerd/maintainerd-auth/internal/authn"
	"github.com/maintainerd/maintainerd-auth/internal/branding"
	"github.com/maintainerd/maintainerd-auth/internal/client"
	"github.com/maintainerd/maintainerd-auth/internal/event"
	"github.com/maintainerd/maintainerd-auth/internal/federation"
	"github.com/maintainerd/maintainerd-auth/internal/iam"
	"github.com/maintainerd/maintainerd-auth/internal/idp"
	"github.com/maintainerd/maintainerd-auth/internal/invite"
	"github.com/maintainerd/maintainerd-auth/internal/mfa"
	"github.com/maintainerd/maintainerd-auth/internal/notifier"
	"github.com/maintainerd/maintainerd-auth/internal/oauth"
	"github.com/maintainerd/maintainerd-auth/internal/secpolicy"
	"github.com/maintainerd/maintainerd-auth/internal/tenant"
	"github.com/maintainerd/maintainerd-auth/internal/user"
	"github.com/maintainerd/maintainerd-auth/internal/webhook"
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
	idpAllowedAudienceRepo    idp.IdentityProviderAllowedAudienceRepository
	roleRepo                  iam.RoleRepository
	rolePermissionRepo        iam.RolePermissionRepository
	clientRepo                client.ClientRepository
	clientPermissionRepo      client.ClientPermissionRepository
	clientAPIRepo             client.ClientAPIRepository
	clientRoleRepo            client.ClientRoleRepository
	clientURIRepo             client.ClientURIRepository
	userRepo                  user.UserRepository
	userIdentityRepo          user.UserIdentityRepository
	userRoleRepo              user.UserRoleRepository
	userTokenRepo             user.UserTokenRepository
	profileRepo               user.ProfileRepository
	profilePictureRepo        user.ProfilePictureRepository
	userSettingRepo           user.UserSettingRepository
	inviteRepo                invite.InviteRepository
	emailTemplateRepo         branding.EmailTemplateRepository
	smsTemplateRepo           branding.SMSTemplateRepository
	policyRepo                iam.PolicyRepository
	servicePolicyRepo         iam.ServicePolicyRepository
	registrationFlowRepo      idp.RegistrationFlowRepository
	registrationFlowRoleRepo  idp.RegistrationFlowRoleRepository
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
	mfaPhoneRepo              mfa.UserMFAPhoneRepository
	emailOTPRepo              mfa.UserMFAEmailRepository
	userBackupCodeRepo        mfa.UserMFABackupCodeRepository
	totpSecretRepo            mfa.UserMFATOTPSecretRepository
	mfaWebAuthnCredRepo       mfa.UserMFAWebAuthnCredentialRepository
	userPasswordHistoryRepo   user.UserPasswordHistoryRepository
	userLockoutRepo           authn.UserLockoutRepository
	userSessionRepo           authn.UserSessionRepository
	userConsentRepo           user.UserConsentRepository
	userTrustedDeviceRepo     user.UserTrustedDeviceRepository
	auditLogRepo              auditlog.ManagementAuditLogRepository
	webauthnChallengeRepo     mfa.WebAuthnChallengeRepository
	signingKeyRepo            oauth.SigningKeyRepository
	tokenRevocationRepo       oauth.OAuthTokenRevocationRepository
	tokenExchangeRepo         oauth.OAuthTokenExchangeRepository
	wifRepo                   federation.WorkloadIdentityFederationRepository
	dataErasureRequestRepo    user.DataErasureRequestRepository
	accountLinkRequestRepo    authn.AccountLinkRequestRepository
	policyVersionHistoryRepo  iam.PolicyVersionHistoryRepository
	oauthDPoPNonceRepo        oauth.OAuthDPoPNonceRepository
}

func initRepos(db *gorm.DB) *repos {
	return &repos{
		serviceRepo:               iam.NewServiceRepository(db),
		apiRepo:                   iam.NewAPIRepository(db),
		permissionRepo:            iam.NewPermissionRepository(db),
		tenantRepo:                tenant.NewTenantRepository(db),
		tenantMemberRepo:          tenant.NewTenantMemberRepository(db),
		idpRepo:                   idp.NewIdentityProviderRepository(db),
		idpAllowedAudienceRepo:    idp.NewIdentityProviderAllowedAudienceRepository(db),
		roleRepo:                  iam.NewRoleRepository(db),
		rolePermissionRepo:        iam.NewRolePermissionRepository(db),
		clientRepo:                client.NewClientRepository(db),
		clientPermissionRepo:      client.NewClientPermissionRepository(db),
		clientAPIRepo:             client.NewClientAPIRepository(db),
		clientRoleRepo:            client.NewClientRoleRepository(db),
		clientURIRepo:             client.NewClientURIRepository(db),
		userRepo:                  user.NewUserRepository(db),
		userIdentityRepo:          user.NewUserIdentityRepository(db),
		userRoleRepo:              user.NewUserRoleRepository(db),
		userTokenRepo:             user.NewUserTokenRepository(db),
		profileRepo:               user.NewProfileRepository(db),
		profilePictureRepo:        user.NewProfilePictureRepository(db),
		userSettingRepo:           user.NewUserSettingRepository(db),
		inviteRepo:                invite.NewInviteRepository(db),
		emailTemplateRepo:         branding.NewEmailTemplateRepository(db),
		smsTemplateRepo:           branding.NewSMSTemplateRepository(db),
		policyRepo:                iam.NewPolicyRepository(db),
		servicePolicyRepo:         iam.NewServicePolicyRepository(db),
		registrationFlowRepo:      idp.NewRegistrationFlowRepository(db),
		registrationFlowRoleRepo:  idp.NewRegistrationFlowRoleRepository(db),
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
		mfaPhoneRepo:              mfa.NewUserMFAPhoneRepository(db),
		emailOTPRepo:              mfa.NewUserMFAEmailRepository(db),
		userBackupCodeRepo:        mfa.NewUserMFABackupCodeRepository(db),
		totpSecretRepo:            mfa.NewUserMFATOTPSecretRepository(db),
		mfaWebAuthnCredRepo:       mfa.NewUserMFAWebAuthnCredentialRepository(db),
		userPasswordHistoryRepo:   user.NewUserPasswordHistoryRepository(db),
		userLockoutRepo:           authn.NewUserLockoutRepository(db),
		userSessionRepo:           authn.NewUserSessionRepository(db),
		userConsentRepo:           user.NewUserConsentRepository(db),
		userTrustedDeviceRepo:     user.NewUserTrustedDeviceRepository(db),
		auditLogRepo:              auditlog.NewManagementAuditLogRepository(db),
		webauthnChallengeRepo:     mfa.NewWebAuthnChallengeRepository(db),
		signingKeyRepo:            oauth.NewSigningKeyRepository(db),
		tokenRevocationRepo:       oauth.NewOAuthTokenRevocationRepository(db),
		tokenExchangeRepo:         oauth.NewOAuthTokenExchangeRepository(db),
		wifRepo:                   federation.NewWorkloadIdentityFederationRepository(db),
		dataErasureRequestRepo:    user.NewDataErasureRequestRepository(db),
		accountLinkRequestRepo:    authn.NewAccountLinkRequestRepository(db),
		policyVersionHistoryRepo:  iam.NewPolicyVersionHistoryRepository(db),
		oauthDPoPNonceRepo:        oauth.NewOAuthDPoPNonceRepository(db),
	}
}
