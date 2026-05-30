package app

import (
	"github.com/maintainerd/auth/internal/authevent"
	"github.com/maintainerd/auth/internal/branding"
	"github.com/maintainerd/auth/internal/client"
	"github.com/maintainerd/auth/internal/iam"
	"github.com/maintainerd/auth/internal/idp"
	"github.com/maintainerd/auth/internal/invite"
	"github.com/maintainerd/auth/internal/mfa"
	"github.com/maintainerd/auth/internal/notifier"
	"github.com/maintainerd/auth/internal/oauth"
	"github.com/maintainerd/auth/internal/secpolicy"
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
	userPoolRepo              user.UserPoolRepository
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
	loginTemplateRepo         branding.LoginTemplateRepository
	policyRepo                iam.PolicyRepository
	servicePolicyRepo         iam.ServicePolicyRepository
	apiKeyRepo                client.APIKeyRepository
	apiKeyAPIRepo             client.APIKeyAPIRepository
	apiKeyPermissionRepo      client.APIKeyPermissionRepository
	signupFlowRepo            idp.SignupFlowRepository
	signupFlowRoleRepo        idp.SignupFlowRoleRepository
	securitySettingRepo       secpolicy.SecuritySettingRepository
	securitySettingsAuditRepo secpolicy.SecuritySettingsAuditRepository
	ipRestrictionRuleRepo     secpolicy.IPRestrictionRuleRepository
	brandingRepo              branding.BrandingRepository
	tenantSettingRepo         tenant.TenantSettingRepository
	emailConfigRepo           notifier.EmailConfigRepository
	smsConfigRepo             notifier.SMSConfigRepository
	webhookEndpointRepo       webhook.WebhookEndpointRepository
	authEventRepo             authevent.AuthEventRepository
	oauthAuthCodeRepo         oauth.OAuthAuthorizationCodeRepository
	oauthRefreshTokenRepo     oauth.OAuthRefreshTokenRepository
	oauthConsentGrantRepo     oauth.OAuthConsentGrantRepository
	oauthConsentChallengeRepo oauth.OAuthConsentChallengeRepository
	oauthPARRequestRepo       oauth.OAuthPARRequestRepository
	oauthDeviceCodeRepo       oauth.OAuthDeviceCodeRepository
	oauthCIBARequestRepo      oauth.OAuthCIBARequestRepository
	smsOtpRepo                notifier.SMSOtpRepository
	userBackupCodeRepo        mfa.UserBackupCodeRepository
	totpSecretRepo            mfa.UserTOTPSecretRepository
	webAuthnCredRepo          mfa.UserWebAuthnCredentialRepository
	userPasswordHistoryRepo   user.UserPasswordHistoryRepository
}

func initRepos(db *gorm.DB) *repos {
	return &repos{
		serviceRepo:               iam.NewServiceRepository(db),
		apiRepo:                   iam.NewAPIRepository(db),
		permissionRepo:            iam.NewPermissionRepository(db),
		tenantRepo:                tenant.NewTenantRepository(db),
		tenantMemberRepo:          tenant.NewTenantMemberRepository(db),
		userPoolRepo:              user.NewUserPoolRepository(db),
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
		loginTemplateRepo:         branding.NewLoginTemplateRepository(db),
		policyRepo:                iam.NewPolicyRepository(db),
		servicePolicyRepo:         iam.NewServicePolicyRepository(db),
		apiKeyRepo:                client.NewAPIKeyRepository(db),
		apiKeyAPIRepo:             client.NewAPIKeyAPIRepository(db),
		apiKeyPermissionRepo:      client.NewAPIKeyPermissionRepository(db),
		signupFlowRepo:            idp.NewSignupFlowRepository(db),
		signupFlowRoleRepo:        idp.NewSignupFlowRoleRepository(db),
		securitySettingRepo:       secpolicy.NewSecuritySettingRepository(db),
		securitySettingsAuditRepo: secpolicy.NewSecuritySettingsAuditRepository(db),
		ipRestrictionRuleRepo:     secpolicy.NewIPRestrictionRuleRepository(db),
		brandingRepo:              branding.NewBrandingRepository(db),
		tenantSettingRepo:         tenant.NewTenantSettingRepository(db),
		emailConfigRepo:           notifier.NewEmailConfigRepository(db),
		smsConfigRepo:             notifier.NewSMSConfigRepository(db),
		webhookEndpointRepo:       webhook.NewWebhookEndpointRepository(db),
		authEventRepo:             authevent.NewAuthEventRepository(db),
		oauthAuthCodeRepo:         oauth.NewOAuthAuthorizationCodeRepository(db),
		oauthRefreshTokenRepo:     oauth.NewOAuthRefreshTokenRepository(db),
		oauthConsentGrantRepo:     oauth.NewOAuthConsentGrantRepository(db),
		oauthConsentChallengeRepo: oauth.NewOAuthConsentChallengeRepository(db),
		oauthPARRequestRepo:       oauth.NewOAuthPARRequestRepository(db),
		oauthDeviceCodeRepo:       oauth.NewOAuthDeviceCodeRepository(db),
		oauthCIBARequestRepo:      oauth.NewOAuthCIBARequestRepository(db),
		smsOtpRepo:                notifier.NewSMSOtpRepository(db),
		userBackupCodeRepo:        mfa.NewUserBackupCodeRepository(db),
		totpSecretRepo:            mfa.NewUserTOTPSecretRepository(db),
		webAuthnCredRepo:          mfa.NewUserWebAuthnCredentialRepository(db),
		userPasswordHistoryRepo:   user.NewUserPasswordHistoryRepository(db),
	}
}
