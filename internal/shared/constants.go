package shared

// Status constants shared across models.
// Use these instead of bare string literals to prevent typos and enable
// IDE-assisted refactoring.
const (
	// General entity statuses
	StatusActive    = "active"
	StatusInactive  = "inactive"
	StatusPending   = "pending"
	StatusSuspended = "suspended"

	// Service-specific statuses
	StatusMaintenance = "maintenance"
	StatusDeprecated  = "deprecated"

	// Invite-specific statuses (Invite.Status)
	StatusAccepted = "accepted"
	StatusRevoked  = "revoked"

	// User profile visibility (UserSetting.ProfileVisibility)
	VisibilityPublic  = "public"
	VisibilityPrivate = "private"
	VisibilityFriends = "friends"

	// Token types (UserToken.TokenType)
	TokenTypeEmailVerification = "user:email:verification"
	TokenTypePasswordReset     = "user:password:reset"
	TokenTypeMagicLink         = "user:magic_link"
	TokenTypeSession           = "user:session"
	TokenTypeMFATrustedDevice  = "user:mfa:trusted_device"

	// Default OAuth token response values.
	DefaultAccessTokenExpiresIn = 3600
	DefaultTokenScope           = "openid profile email"

	// Role names (Role.Name) — system-defined roles
	RoleSuperAdmin = "super-admin"
	RoleRegistered = "registered"

	// Tenant member roles (TenantMember.Role)
	TenantRoleOwner  = "owner"
	TenantRoleMember = "member"

	// Client types (Client.ClientType)
	ClientTypeTraditional = "traditional"
	ClientTypeSPA         = "spa"
	ClientTypeMobile      = "mobile"
	ClientTypeM2M         = "m2m"

	// Seeded first-party clients (Client.Name).
	SystemClientNameAuthConsole  = "auth-console"
	SystemClientNameAuthIdentity = "auth-identity"

	// Client URI types (ClientURI.Type)
	ClientURITypeRedirect   = "redirect-uri"
	ClientURITypeOrigin     = "origin-uri"
	ClientURITypeLogout     = "logout-uri"
	ClientURITypeLogin      = "login-uri"
	ClientURITypeCORSOrigin = "cors-origin-uri"

	// Gender values (Profile.Gender)
	GenderMale           = "male"
	GenderFemale         = "female"
	GenderOther          = "other"
	GenderPreferNotToSay = "prefer_not_to_say"

	// Preferred contact methods (UserSetting.PreferredContactMethod)
	ContactMethodEmail = "email"
	ContactMethodPhone = "phone"
	ContactMethodSMS   = "sms"

	// Identity provider names (UserIdentity.Provider)
	ProviderMaintainerd = "maintainerd"

	// Identity provider Provider values (IdentityProvider.Provider)
	IDPProviderMaintainerd = "maintainerd"
	IDPProviderCognito     = "cognito"
	IDPProviderAuth0       = "auth0"
	IDPProviderGoogle      = "google"
	IDPProviderFacebook    = "facebook"
	IDPProviderGitHub      = "github"
	IDPProviderMicrosoft   = "microsoft"
	IDPProviderApple       = "apple"
	IDPProviderLinkedIn    = "linkedin"
	IDPProviderTwitter     = "twitter"
	IDPProviderGitLab      = "gitlab"

	// Identity provider types (IdentityProvider.ProviderType)
	IDPTypeSystem     = "system"
	IDPTypeSocial     = "social"
	IDPTypeEnterprise = "enterprise"
	// IDPTypeIdentity is retained as a compatibility alias while callers move
	// from identity/social to system/social/enterprise.
	IDPTypeIdentity = IDPTypeEnterprise

	// IP restriction rule types (IPRestrictionRule.Type)
	IPRuleTypeAllow     = "allow"
	IPRuleTypeDeny      = "deny"
	IPRuleTypeWhitelist = "whitelist"
	IPRuleTypeBlacklist = "blacklist"

	// Login template styles (LoginTemplate.Template)
	LoginTemplateModern    = "modern"
	LoginTemplateClassic   = "classic"
	LoginTemplateMinimal   = "minimal"
	LoginTemplateCorporate = "corporate"
	LoginTemplateCreative  = "creative"
	LoginTemplateCustom    = "custom"

	// Branding layouts (Branding.Layout)
	BrandingLayoutCentered = "centered"
	BrandingLayoutFullPage = "full_page"
	BrandingLayoutSplit    = "split"

	// Policy statement effects (PolicyStatement.Effect)
	PolicyEffectAllow = "allow"
	PolicyEffectDeny  = "deny"

	// Transport/cache defaults.
	DefaultGRPCAddr             = ":50051"
	DefaultDiscoveryCacheMaxAge = "public, max-age=3600"

	// Auth flow destinations (AuthFlow.Destination)
	DestinationConsole  = "console"
	DestinationIdentity = "identity"
)
