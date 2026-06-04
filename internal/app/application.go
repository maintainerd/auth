package app

import "github.com/maintainerd/auth/internal/server"

// ServerApplication adapts the legacy app bundle to the server package's
// transport-focused dependency bundle.
func (a *App) ServerApplication() *server.Application {
	if a == nil {
		return nil
	}

	return &server.Application{
		DB:                        a.DB,
		RedisClient:               a.RedisClient,
		Cache:                     a.Cache,
		ServiceService:            a.ServiceService,
		APIService:                a.APIService,
		PermissionService:         a.PermissionService,
		PolicyService:             a.PolicyService,
		TenantService:             a.TenantService,
		TenantMemberService:       a.TenantMemberService,
		IdentityProviderService:   a.IdentityProviderService,
		ClientService:             a.ClientService,
		RoleService:               a.RoleService,
		UserService:               a.UserService,
		RegisterService:           a.RegisterService,
		LoginService:              a.LoginService,
		ProfileService:            a.ProfileService,
		UserSettingService:        a.UserSettingService,
		InviteService:             a.InviteService,
		ForgotPasswordService:     a.ForgotPasswordService,
		ResetPasswordService:      a.ResetPasswordService,
		EmailVerificationService:  a.EmailVerificationService,
		MagicLinkService:          a.MagicLinkService,
		SetupService:              a.SetupService,
		SignupFlowService:         a.SignupFlowService,
		APIKeyService:             a.APIKeyService,
		SecuritySettingService:    a.SecuritySettingService,
		IPRestrictionRuleService:  a.IPRestrictionRuleService,
		EmailTemplateService:      a.EmailTemplateService,
		SMSTemplateService:        a.SMSTemplateService,
		LoginTemplateService:      a.LoginTemplateService,
		BrandingService:           a.BrandingService,
		TenantSettingService:      a.TenantSettingService,
		EmailConfigService:        a.EmailConfigService,
		SMSConfigService:          a.SMSConfigService,
		WebhookEndpointService:    a.WebhookEndpointService,
		AuthEventService:          a.AuthEventService,
		AuthorizationService:      a.AuthorizationService,
		OAuthAuthorizeService:     a.OAuthAuthorizeService,
		OAuthTokenService:         a.OAuthTokenService,
		OAuthConsentService:       a.OAuthConsentService,
		OAuthPARService:           a.OAuthPARService,
		OAuthDeviceService:        a.OAuthDeviceService,
		OAuthTokenExchangeService: a.OAuthTokenExchangeService,
		OAuthSessionService:       a.OAuthSessionService,
		OAuthCIBAService:          a.OAuthCIBAService,
		OAuthRegisterService:      a.OAuthRegisterService,
		AccountService:            a.AccountService,
		SessionService:            a.SessionService,
		SMSLoginService:           a.SMSLoginService,
		MFAService:                a.MFAService,
		WebAuthnService:           a.WebAuthnService,
		FederationService:         a.FederationService,
	}
}
