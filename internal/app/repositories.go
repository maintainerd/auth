package app

import (
	"github.com/maintainerd/auth/internal/repository"
	"gorm.io/gorm"
)

// repos holds every repository instance. It is private to the app package and
// is only passed between the three init functions below.
type repos struct {
	serviceRepo               repository.ServiceRepository
	tenantServiceRepo         repository.TenantServiceRepository
	apiRepo                   repository.APIRepository
	permissionRepo            repository.PermissionRepository
	tenantRepo                repository.TenantRepository
	tenantMemberRepo          repository.TenantMemberRepository
	userPoolRepo              repository.UserPoolRepository
	idpRepo                   repository.IdentityProviderRepository
	roleRepo                  repository.RoleRepository
	rolePermissionRepo        repository.RolePermissionRepository
	clientRepo                repository.ClientRepository
	clientPermissionRepo      repository.ClientPermissionRepository
	clientAPIRepo             repository.ClientAPIRepository
	clientURIRepo             repository.ClientURIRepository
	userRepo                  repository.UserRepository
	userIdentityRepo          repository.UserIdentityRepository
	userRoleRepo              repository.UserRoleRepository
	userTokenRepo             repository.UserTokenRepository
	profileRepo               repository.ProfileRepository
	userSettingRepo           repository.UserSettingRepository
	inviteRepo                repository.InviteRepository
	emailTemplateRepo         repository.EmailTemplateRepository
	smsTemplateRepo           repository.SMSTemplateRepository
	loginTemplateRepo         repository.LoginTemplateRepository
	policyRepo                repository.PolicyRepository
	servicePolicyRepo         repository.ServicePolicyRepository
	apiKeyRepo                repository.APIKeyRepository
	apiKeyAPIRepo             repository.APIKeyAPIRepository
	apiKeyPermissionRepo      repository.APIKeyPermissionRepository
	signupFlowRepo            repository.SignupFlowRepository
	signupFlowRoleRepo        repository.SignupFlowRoleRepository
	securitySettingRepo       repository.SecuritySettingRepository
	securitySettingsAuditRepo repository.SecuritySettingsAuditRepository
	ipRestrictionRuleRepo     repository.IPRestrictionRuleRepository
	brandingRepo              repository.BrandingRepository
	tenantSettingRepo         repository.TenantSettingRepository
	emailConfigRepo           repository.EmailConfigRepository
	smsConfigRepo             repository.SMSConfigRepository
	webhookEndpointRepo       repository.WebhookEndpointRepository
	authEventRepo             repository.AuthEventRepository
	oauthAuthCodeRepo         repository.OAuthAuthorizationCodeRepository
	oauthRefreshTokenRepo     repository.OAuthRefreshTokenRepository
	oauthConsentGrantRepo     repository.OAuthConsentGrantRepository
	oauthConsentChallengeRepo repository.OAuthConsentChallengeRepository
}

func initRepos(db *gorm.DB) *repos {
	return &repos{
		serviceRepo:               repository.NewServiceRepository(db),
		tenantServiceRepo:         repository.NewTenantServiceRepository(db),
		apiRepo:                   repository.NewAPIRepository(db),
		permissionRepo:            repository.NewPermissionRepository(db),
		tenantRepo:                repository.NewTenantRepository(db),
		tenantMemberRepo:          repository.NewTenantMemberRepository(db),
		userPoolRepo:              repository.NewUserPoolRepository(db),
		idpRepo:                   repository.NewIdentityProviderRepository(db),
		roleRepo:                  repository.NewRoleRepository(db),
		rolePermissionRepo:        repository.NewRolePermissionRepository(db),
		clientRepo:                repository.NewClientRepository(db),
		clientPermissionRepo:      repository.NewClientPermissionRepository(db),
		clientAPIRepo:             repository.NewClientAPIRepository(db),
		clientURIRepo:             repository.NewClientURIRepository(db),
		userRepo:                  repository.NewUserRepository(db),
		userIdentityRepo:          repository.NewUserIdentityRepository(db),
		userRoleRepo:              repository.NewUserRoleRepository(db),
		userTokenRepo:             repository.NewUserTokenRepository(db),
		profileRepo:               repository.NewProfileRepository(db),
		userSettingRepo:           repository.NewUserSettingRepository(db),
		inviteRepo:                repository.NewInviteRepository(db),
		emailTemplateRepo:         repository.NewEmailTemplateRepository(db),
		smsTemplateRepo:           repository.NewSMSTemplateRepository(db),
		loginTemplateRepo:         repository.NewLoginTemplateRepository(db),
		policyRepo:                repository.NewPolicyRepository(db),
		servicePolicyRepo:         repository.NewServicePolicyRepository(db),
		apiKeyRepo:                repository.NewAPIKeyRepository(db),
		apiKeyAPIRepo:             repository.NewAPIKeyAPIRepository(db),
		apiKeyPermissionRepo:      repository.NewAPIKeyPermissionRepository(db),
		signupFlowRepo:            repository.NewSignupFlowRepository(db),
		signupFlowRoleRepo:        repository.NewSignupFlowRoleRepository(db),
		securitySettingRepo:       repository.NewSecuritySettingRepository(db),
		securitySettingsAuditRepo: repository.NewSecuritySettingsAuditRepository(db),
		ipRestrictionRuleRepo:     repository.NewIPRestrictionRuleRepository(db),
		brandingRepo:              repository.NewBrandingRepository(db),
		tenantSettingRepo:         repository.NewTenantSettingRepository(db),
		emailConfigRepo:           repository.NewEmailConfigRepository(db),
		smsConfigRepo:             repository.NewSMSConfigRepository(db),
		webhookEndpointRepo:       repository.NewWebhookEndpointRepository(db),
		authEventRepo:             repository.NewAuthEventRepository(db),
		oauthAuthCodeRepo:         repository.NewOAuthAuthorizationCodeRepository(db),
		oauthRefreshTokenRepo:     repository.NewOAuthRefreshTokenRepository(db),
		oauthConsentGrantRepo:     repository.NewOAuthConsentGrantRepository(db),
		oauthConsentChallengeRepo: repository.NewOAuthConsentChallengeRepository(db),
	}
}
