package authn

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// Repository projections and interfaces used by authn.
// ---------------------------------------------------------------------------
// Local aggregate structs — same underlying tables as owning domains.
// Cross-aggregate references are IDs only except for required navigational FKs.
// ---------------------------------------------------------------------------

type Tenant struct {
	TenantID    int64
	TenantUUID  uuid.UUID
	Name        string
	DisplayName string
	Description string
	Identifier  string
	Status      string
	IsSystem    bool
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

func (Tenant) TableName() string { return "tenants" }

type IdentityProvider struct {
	IdentityProviderID   int64
	IdentityProviderUUID uuid.UUID
	TenantID             int64
	Name                 string
	Provider             string
	ProviderType         string
	Identifier           string
	Status               string
	IsSystem             bool
	IsDefault            bool
	Tenant               *Tenant `gorm:"foreignKey:TenantID;references:TenantID"`
	CreatedAt            time.Time
	UpdatedAt            time.Time
}

func (IdentityProvider) TableName() string { return "identity_providers" }

type Client struct {
	ClientID               int64
	ClientUUID             uuid.UUID
	TenantID               int64
	IdentityProviderID     int64
	Name                   string
	DisplayName            string
	ClientType             string
	Domain                 *string
	Identifier             *string
	Status                 string
	IsDefault              bool
	IsSystem               bool
	AccessTokenTTL         *int
	RefreshTokenTTL        *int
	RequiredACR            *string
	RequirePKCE            *bool
	SessionIdleTimeout     *int
	SessionAbsoluteTimeout *int
	IdentityProvider       *IdentityProvider `gorm:"foreignKey:IdentityProviderID;references:IdentityProviderID"`
	ConnectedProviders     *[]ClientIdentityProvider
	CreatedAt              time.Time
	UpdatedAt              time.Time
}

func (Client) TableName() string { return "clients" }

type ClientIdentityProvider struct {
	ClientIdentityProviderID   int64
	ClientIdentityProviderUUID uuid.UUID
	TenantID                   int64
	ClientID                   int64
	IdentityProviderID         int64
	Enabled                    bool
	IsDefault                  bool
	IdentityProvider           *IdentityProvider
}

func (ClientIdentityProvider) TableName() string { return "client_identity_providers" }

type User struct {
	UserID                     int64
	UserUUID                   uuid.UUID
	TenantID                   int64
	Username                   string
	Fullname                   string `gorm:"-"`
	Email                      string
	Phone                      string
	Password                   *string
	IsEmailVerified            bool
	IsPhoneVerified            bool
	IsProfileCompleted         bool
	IsAccountCompleted         bool
	Status                     string
	ForcePasswordChange        bool
	PasswordChangedAt          *time.Time
	TemporaryPasswordExpiresAt *time.Time
	IsTOTPEnabled              bool
	IsWebAuthnEnabled          bool `gorm:"column:is_webauthn_enabled"`
	MFAEnabledAt               *time.Time
	CreatedAt                  time.Time
	UpdatedAt                  time.Time
}

func (User) TableName() string { return "users" }

type UserIdentity struct {
	UserIdentityID     int64
	UserIdentityUUID   uuid.UUID
	TenantID           int64
	UserID             int64
	ClientID           int64
	IdentityProviderID *int64 `gorm:"column:identity_provider_id"`
	Provider           string
	Sub                string
	Metadata           datatypes.JSON
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

func (UserIdentity) TableName() string { return "user_identities" }

type UserToken struct {
	UserTokenID        int64
	UserTokenUUID      uuid.UUID
	UserID             int64
	TokenType          string
	Token              string
	ExpiresAt          *time.Time
	IsRevoked          bool
	IPAddress          *string
	UserAgent          *string
	LastUsedAt         *time.Time
	IdleTimeoutSeconds *int
	AbsoluteExpiresAt  *time.Time
	CreatedAt          time.Time
	UpdatedAt          *time.Time
}

func (UserToken) TableName() string { return "user_tokens" }

type UserRole struct {
	UserRoleID int64
	UserID     int64
	RoleID     int64
	CreatedAt  time.Time
}

func (UserRole) TableName() string { return "user_roles" }

type Role struct {
	RoleID      int64
	RoleUUID    uuid.UUID
	TenantID    int64
	Name        string
	Description string
	Status      string
	IsDefault   bool
	IsSystem    bool
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

func (Role) TableName() string { return "roles" }

type Invite struct {
	InviteID     int64
	InviteUUID   uuid.UUID
	TenantID     int64
	InvitedEmail string
	AuthFlowID   *int64
	Status       string
	ExpiresAt    *time.Time
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

func (Invite) TableName() string { return "invites" }

// ---------------------------------------------------------------------------
// Repository filter types
// ---------------------------------------------------------------------------

type ClientRepositoryGetFilter struct {
	TenantID           int64
	IdentityProviderID int64
	Name               *string
	Status             *string
	Page               int
	Limit              int
	SortBy             string
	SortOrder          string
}

type UserRepositoryGetFilter struct {
	TenantID  int64
	Status    *string
	Page      int
	Limit     int
	SortBy    string
	SortOrder string
}

type IdentityProviderRepositoryGetFilter struct {
	TenantID  int64
	Status    *string
	Page      int
	Limit     int
	SortBy    string
	SortOrder string
}

type RoleRepositoryGetFilter struct {
	Name        *string
	Description *string
	IsDefault   *bool
	IsSystem    *bool
	Status      *string
	TenantID    int64
	Page        int
	Limit       int
	SortBy      string
	SortOrder   string
}

// ---------------------------------------------------------------------------
// Repository interfaces (consumer-defined: authn specifies only what it needs)
// ---------------------------------------------------------------------------

type ClientRepository interface {
	BaseRepositoryMethods[Client]
	WithTx(tx *gorm.DB) ClientRepository
	FindByClientIDAndIdentityProvider(clientID, providerID string) (*Client, error)
	FindByIdentifier(identifier string) (*Client, error)
	FindSystem() (*Client, error)
	FindSystemByTenantIdentifier(tenantIdentifier string) (*Client, error)
	FindSystemByTenantIdentifierAndName(tenantIdentifier, name string) (*Client, error)
	FindByUUIDAndTenantID(id uuid.UUID, tenantID int64) (*Client, error)
	FindByNameAndIdentityProvider(name string, ipID, tenantID int64) (*Client, error)
	FindByNameAndTenantID(name string, tenantID int64) (*Client, error)
	FindByClientID(clientID string, tenantID int64) (*Client, error)
	FindAllByTenantID(tenantID int64) ([]Client, error)
	FindDefaultByTenantID(tenantID int64) (*Client, error)
	FindPaginated(filter ClientRepositoryGetFilter) (*PaginationResult[Client], error)
	SetStatusByUUID(id uuid.UUID, tenantID int64, status string) error
	DeleteByUUIDAndTenantID(id uuid.UUID, tenantID int64) error
}

type UserRepository interface {
	BaseRepositoryMethods[User]
	FindByID(id any, preloads ...string) (*User, error)
	FindByUUID(uuid any, preloads ...string) (*User, error)
	UpdateByID(id any, updatedData any) (*User, error)
	WithTx(tx *gorm.DB) UserRepository
	FindByEmailAndTenantID(email string, tenantID int64) (*User, error)
	FindByUsernameAndTenantID(username string, tenantID int64) (*User, error)
	FindByPhoneAndTenantID(phone string, tenantID int64) (*User, error)
	FindByPendingEmailAndTenantID(email string, tenantID int64) (*User, error)
	FindSuperAdmin() (*User, error)
	FindRoles(userID int64) ([]Role, error)
	FindBySubAndClientID(sub, clientID string) (*User, error)
	FindPaginated(filter UserRepositoryGetFilter) (*PaginationResult[User], error)
	SetEmailVerified(id uuid.UUID, verified bool) error
	SetStatus(id uuid.UUID, status string) error
	SetForcePasswordChange(id uuid.UUID, force bool) error
	SetPendingEmail(id uuid.UUID, pendingEmail, token string, expiresAt time.Time) error
	ClearEmailChange(id uuid.UUID) error
	UpdateEmail(id uuid.UUID, email string) error
	UpdateUsername(id uuid.UUID, username string) error
}

type UserIdentityRepository interface {
	BaseRepositoryMethods[UserIdentity]
	WithTx(tx *gorm.DB) UserIdentityRepository
	FindByUserIDAndClientID(userID, clientID int64) (*UserIdentity, error)
	FindByUserID(userID int64) ([]UserIdentity, error)
	FindByUserIDAndProvider(userID int64, provider string) (*UserIdentity, error)
	FindByIdentityProviderID(idpID int64) ([]UserIdentity, error)
	DeleteByUserID(userID int64) error
}

type IdentityProviderRepository interface {
	BaseRepositoryMethods[IdentityProvider]
	WithTx(tx *gorm.DB) IdentityProviderRepository
	FindByIdentifier(identifier string) (*IdentityProvider, error)
	FindByName(name string, tenantID int64) (*IdentityProvider, error)
	FindDefaultByTenantID(tenantID int64) (*IdentityProvider, error)
	FindPaginated(filter IdentityProviderRepositoryGetFilter) (*PaginationResult[IdentityProvider], error)
	FindAllByTenantID(tenantID int64) ([]IdentityProvider, error)
	FindByTenantAndProvider(tenantID int64, provider string) (*IdentityProvider, error)
}

type UserTokenRepository interface {
	BaseRepositoryMethods[UserToken]
	WithTx(tx *gorm.DB) UserTokenRepository
	FindByUserID(userID int64) ([]UserToken, error)
	FindActiveTokensByUserID(userID int64) ([]UserToken, error)
	FindByUserIDAndTokenType(userID int64, tokenType string) ([]UserToken, error)
	RevokeByUUID(id uuid.UUID) error
	RevokeAllByUserID(userID int64) error
	DeleteByUserID(userID int64) error
	DeleteExpiredTokens(before time.Time) error
	FindActiveSessions(userID int64) ([]UserToken, error)
	FindActiveSessionByUUID(userID int64, sessionUUID uuid.UUID) (*UserToken, error)
	CountActiveSessions(userID int64) (int64, error)
	TouchSession(userID int64, sessionUUID uuid.UUID, now time.Time) error
	RevokeSessionByUUID(userID int64, sessionUUID uuid.UUID) error
	RevokeAllSessionsByUserID(userID int64) error
}

type RoleRepository interface {
	BaseRepositoryMethods[Role]
	WithTx(tx *gorm.DB) RoleRepository
	FindPaginated(filter RoleRepositoryGetFilter) (*PaginationResult[Role], error)
	FindByNameAndTenantID(name string, tenantID int64) (*Role, error)
}

type UserRoleRepository interface {
	BaseRepositoryMethods[UserRole]
	WithTx(tx *gorm.DB) UserRoleRepository
	FindByUserIDAndRoleID(userID, roleID int64) (*UserRole, error)
}

type InviteRepository interface {
	BaseRepositoryMethods[Invite]
	WithTx(tx *gorm.DB) InviteRepository
	FindByToken(token string) (*Invite, error)
	MarkAsUsed(inviteUUID uuid.UUID) error
}

type UserPasswordHistoryRepository interface {
	WithTx(tx *gorm.DB) UserPasswordHistoryRepository
	AddEntry(userID int64, hash string) error
	FindRecentHashes(userID int64, count int) ([]string, error)
	PruneExcess(userID int64, keepCount int) error
}

type AuthFlowRoleRepository interface {
	WithTx(tx *gorm.DB) AuthFlowRoleRepository
	FindRoleIDsByAuthFlowID(authFlowID int64) ([]int64, error)
}
