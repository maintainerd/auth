package model

// Status constants shared across models.
// Use these instead of bare string literals to prevent typos and enable
// IDE-assisted refactoring.
const (
	// General entity statuses
	StatusActive   = "active"
	StatusInactive = "inactive"

	// Service-specific statuses
	StatusMaintenance = "maintenance"
	StatusDeprecated  = "deprecated"

	// User profile visibility (UserSetting.ProfileVisibility)
	VisibilityPublic  = "public"
	VisibilityPrivate = "private"
	VisibilityFriends = "friends"

	// Token types (UserToken.TokenType)
	TokenTypeEmailVerification = "user:email:verification"
	TokenTypePasswordReset     = "user:password:reset"
)
