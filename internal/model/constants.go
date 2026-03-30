package model

// Status constants shared across models.
// Use these instead of bare string literals to prevent typos and enable
// IDE-assisted refactoring.
const (
	// General entity statuses
	StatusActive   = "active"
	StatusInactive = "inactive"
	StatusPending  = "pending"

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

	// Identity provider names (UserIdentity.Provider)
	ProviderDefault = "default"
)
