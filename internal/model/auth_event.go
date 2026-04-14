package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

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

// AuthEvent represents a security event stored in the auth_events table.
// Events are immutable (append-only) following OWASP tamper-protection guidance.
type AuthEvent struct {
	AuthEventID   int64          `gorm:"column:auth_event_id;primaryKey;autoIncrement"`
	AuthEventUUID uuid.UUID      `gorm:"column:auth_event_uuid;type:uuid;uniqueIndex;not null"`
	TenantID      int64          `gorm:"column:tenant_id;not null"`
	ActorUserID   *int64         `gorm:"column:actor_user_id"`
	TargetUserID  *int64         `gorm:"column:target_user_id"`
	IPAddress     string         `gorm:"column:ip_address;type:varchar(45);not null"`
	UserAgent     *string        `gorm:"column:user_agent;type:text"`
	Category      string         `gorm:"column:category;type:varchar(20);not null"`
	EventType     string         `gorm:"column:event_type;type:varchar(60);not null"`
	Severity      string         `gorm:"column:severity;type:varchar(10);not null;default:INFO"`
	Result        string         `gorm:"column:result;type:varchar(10);not null"`
	Description   *string        `gorm:"column:description;type:text"`
	ErrorReason   *string        `gorm:"column:error_reason;type:varchar(255)"`
	TraceID       *string        `gorm:"column:trace_id;type:varchar(32)"`
	Metadata      datatypes.JSON `gorm:"column:metadata;type:jsonb;default:'{}'"`
	CreatedAt     time.Time      `gorm:"column:created_at;autoCreateTime;not null"`

	// Relationships
	Tenant     *Tenant `gorm:"foreignKey:TenantID;references:TenantID"`
	ActorUser  *User   `gorm:"foreignKey:ActorUserID;references:UserID"`
	TargetUser *User   `gorm:"foreignKey:TargetUserID;references:UserID"`
}

// TableName returns the database table name for GORM.
func (AuthEvent) TableName() string {
	return "auth_events"
}

// BeforeCreate generates a UUID if one is not already set.
func (ae *AuthEvent) BeforeCreate(_ *gorm.DB) error {
	if ae.AuthEventUUID == uuid.Nil {
		ae.AuthEventUUID = uuid.New()
	}
	return nil
}
