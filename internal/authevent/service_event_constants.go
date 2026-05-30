package authevent

// AuthEvent category constants aligned with the OWASP Logging Vocabulary.
const (
	AuthEventCategoryAuthn   = "AUTHN"
	AuthEventCategoryAuthz   = "AUTHZ"
	AuthEventCategorySession = "SESSION"
	AuthEventCategoryUser    = "USER"
	AuthEventCategorySystem  = "SYSTEM"
)

// AuthEvent severity constants mapped from OWASP Logging Vocabulary levels.
const (
	AuthEventSeverityInfo     = "INFO"
	AuthEventSeverityWarn     = "WARN"
	AuthEventSeverityCritical = "CRITICAL"
)

// AuthEvent result constants required by PCI DSS 10.2.
const (
	AuthEventResultSuccess = "success"
	AuthEventResultFailure = "failure"
)

// OWASP Logging Vocabulary event type constants for the AUTHN category.
const (
	AuthEventTypeLoginSuccess          = "authn_login_success"
	AuthEventTypeLoginFail             = "authn_login_fail"
	AuthEventTypeLoginFailMax          = "authn_login_fail_max"
	AuthEventTypeLoginLock             = "authn_login_lock"
	AuthEventTypeLoginSuccessAfterFail = "authn_login_successafterfail"
	AuthEventTypePasswordChange        = "authn_password_change"
	AuthEventTypePasswordChangeFail    = "authn_password_change_fail"
	AuthEventTypeTokenCreated          = "authn_token_created"
	AuthEventTypeTokenRevoked          = "authn_token_revoked"
	AuthEventTypeTokenReuse            = "authn_token_reuse"
	AuthEventTypeTokenDelete           = "authn_token_delete"
	AuthEventTypeImpossibleTravel      = "authn_impossible_travel"
	AuthEventTypeOAuthAuthorize        = "authn_oauth_authorize"
	AuthEventTypeOAuthConsent          = "authn_oauth_consent"
	AuthEventTypeOAuthConsentDeny      = "authn_oauth_consent_deny"
	AuthEventTypeOAuthTokenExchange    = "authn_oauth_token_exchange"
	AuthEventTypeOAuthTokenRefresh     = "authn_oauth_token_refresh"
	AuthEventTypeOAuthTokenRevoke      = "authn_oauth_token_revoke"
	AuthEventTypeOAuthClientAuth       = "authn_oauth_client_auth"
	AuthEventTypeOAuthClientAuthFail   = "authn_oauth_client_auth_fail"
)

// OWASP Logging Vocabulary event type constants for the AUTHZ category.
const (
	AuthEventTypeAuthzFail   = "authz_fail"
	AuthEventTypeAuthzChange = "authz_change"
	AuthEventTypeAuthzAdmin  = "authz_admin"
)

// OWASP Logging Vocabulary event type constants for the SESSION category.
const (
	AuthEventTypeSessionCreated        = "session_created"
	AuthEventTypeSessionRenewed        = "session_renewed"
	AuthEventTypeSessionExpired        = "session_expired"
	AuthEventTypeSessionUseAfterExpire = "session_use_after_expire"
)

// OWASP Logging Vocabulary event type constants for the USER category.
const (
	AuthEventTypeUserCreated  = "user_created"
	AuthEventTypeUserUpdated  = "user_updated"
	AuthEventTypeUserArchived = "user_archived"
	AuthEventTypeUserDeleted  = "user_deleted"
)

// OWASP Logging Vocabulary event type constants for the PRIVILEGE category.
const (
	AuthEventTypePrivilegePermissionsChanged = "privilege_permissions_changed"
)

// OWASP Logging Vocabulary event type constants for the SYSTEM category.
const (
	AuthEventTypeSystemStartup  = "sys_startup"
	AuthEventTypeSystemShutdown = "sys_shutdown"
	AuthEventTypeSystemCrash    = "sys_crash"
)
