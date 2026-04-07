package model

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

	// Role names (Role.Name) — system-defined roles
	RoleSuperAdmin = "super-admin"
	RoleRegistered = "registered"

	// Client types (Client.ClientType)
	ClientTypeTraditional = "traditional"
	ClientTypeSPA         = "spa"
	ClientTypeMobile      = "mobile"
	ClientTypeM2M         = "m2m"

	// Client URI types (ClientURI.Type)
	ClientURITypeRedirect   = "redirect-uri"
	ClientURITypeOrigin     = "origin-uri"
	ClientURITypeLogout     = "logout-uri"
	ClientURITypeLogin      = "login-uri"
	ClientURITypeCORSOrigin = "cors-origin-uri"

	// API types (API.APIType)
	APITypeRest      = "rest"
	APITypeGRPC      = "grpc"
	APITypeGraphQL   = "graphql"
	APITypeSOAP      = "soap"
	APITypeWebhook   = "webhook"
	APITypeWebSocket = "websocket"
	APITypeRPC       = "rpc"

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
	ProviderDefault = "default"

	// Identity provider Provider values (IdentityProvider.Provider)
	IDPProviderInternal  = "internal"
	IDPProviderCognito   = "cognito"
	IDPProviderAuth0     = "auth0"
	IDPProviderGoogle    = "google"
	IDPProviderFacebook  = "facebook"
	IDPProviderGitHub    = "github"
	IDPProviderMicrosoft = "microsoft"
	IDPProviderApple     = "apple"
	IDPProviderLinkedIn  = "linkedin"
	IDPProviderTwitter   = "twitter"

	// Identity provider types (IdentityProvider.ProviderType)
	IDPTypeIdentity = "identity"
	IDPTypeSocial   = "social"

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

	// Policy statement effects (PolicyStatement.Effect)
	PolicyEffectAllow = "allow"
	PolicyEffectDeny  = "deny"
)
