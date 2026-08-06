package user

import (
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/maintainerd/maintainerd-auth/internal/platform/database"
	"github.com/maintainerd/maintainerd-auth/internal/shared"
	"gorm.io/gorm"
)

type UserRepositoryGetFilter struct {
	Search    *string
	Username  *string
	Email     *string
	Phone     *string
	Fullname  *string
	Status    []string
	TenantID  *int64
	RoleID    *int64
	ClientID  *int64
	Page      int
	Limit     int
	SortBy    string
	SortOrder string
	Cursor    *int64
}

type GetUserRolesFilter struct {
	UserID      int64
	Name        *string
	Description *string
	Status      *string
	Page        int
	Limit       int
	SortBy      string
	SortOrder   string
}

type UserRepository interface {
	BaseRepositoryMethods[User]
	FindByID(id any, preloads ...string) (*User, error)
	FindByUUID(uuid any, preloads ...string) (*User, error)
	FindByUUIDs(uuids []string, preloads ...string) ([]User, error)
	FindAll(preloads ...string) ([]User, error)
	UpdateByID(id any, updatedData any) (*User, error)
	UpdateByUUID(uuid any, updatedData any) (*User, error)
	DeleteByUUID(uuid any) error
	DeleteByID(id any) error
	Paginate(conditions map[string]any, page int, limit int, preloads ...string) (*PaginationResult[User], error)
	WithTx(tx *gorm.DB) UserRepository
	// Tenant-scoped lookups. Users are isolated per tenant (email/username are
	// unique per tenant), so unscoped variants are deliberately not exposed.
	FindByEmailAndTenantID(email string, tenantID int64) (*User, error)
	FindByUsernameAndTenantID(username string, tenantID int64) (*User, error)
	FindByPhoneAndTenantID(phone string, tenantID int64) (*User, error)
	FindSuperAdmin() (*User, error)
	FindRoles(userID int64) ([]Role, error)
	// EffectivePermissionNames returns the permission names a user actually holds
	// in a tenant, through active, non-deleted roles and permissions. It is the
	// authority for "may this actor grant this?".
	EffectivePermissionNames(userID, tenantID int64) ([]string, error)
	FindRolesPaginated(filter GetUserRolesFilter) (*PaginationResult[Role], error)
	FindBySubAndClientID(sub string, clientID string) (*User, error)
	FindPaginated(filter UserRepositoryGetFilter) (*PaginationResult[User], error)
	SetEmailVerified(userUUID uuid.UUID, verified bool) error
	SetStatus(userUUID uuid.UUID, status string) error
	// Feature: Force password change
	SetForcePasswordChange(userUUID uuid.UUID, force bool) error
	// Feature: Email change with OTP re-verification. The OTP + pending address
	// live in user_otps (channel='email_change'); only the final apply remains here.
	UpdateEmail(userUUID uuid.UUID, email string) error
	UpdateUsername(userUUID uuid.UUID, username string) error
}

type userRepository struct {
	*BaseRepository[User]
}

func NewUserRepository(db *gorm.DB) UserRepository {
	return &userRepository{
		BaseRepository: database.NewBaseRepository[User](db, "user_uuid", "user_id"),
	}
}

func (r *userRepository) WithTx(tx *gorm.DB) UserRepository {
	return &userRepository{
		BaseRepository: r.BaseRepository.WithTx(tx),
	}
}

func (r *userRepository) FindByEmailAndTenantID(email string, tenantID int64) (*User, error) {
	var user User
	err := r.DB().
		Where("email = ? AND tenant_id = ?", email, tenantID).
		First(&user).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}

	return &user, nil
}

func (r *userRepository) FindByUsernameAndTenantID(username string, tenantID int64) (*User, error) {
	var user User
	err := r.DB().
		Where("username = ? AND tenant_id = ?", username, tenantID).
		First(&user).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}

	return &user, nil
}

func (r *userRepository) FindByPhoneAndTenantID(phone string, tenantID int64) (*User, error) {
	var user User
	err := r.DB().
		Where("phone = ? AND tenant_id = ?", phone, tenantID).
		First(&user).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}

	return &user, nil
}

func (r *userRepository) FindSuperAdmin() (*User, error) {
	var user User
	err := r.DB().
		Joins("JOIN user_identities ON users.user_id = user_identities.user_id").
		Joins("JOIN tenants ON user_identities.tenant_id = tenants.tenant_id").
		Joins("JOIN user_roles ON users.user_id = user_roles.user_id").
		Joins("JOIN roles ON user_roles.role_id = roles.role_id").
		Where("tenants.status = ? AND tenants.is_system = ?", shared.StatusActive, true).
		Where("roles.name = ?", shared.RoleSuperAdmin).
		First(&user).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}

	return &user, nil
}

func (r *userRepository) FindRoles(userID int64) ([]Role, error) {
	var roles []Role
	err := r.DB().
		Model(&Role{}).
		Select("roles.*").
		Joins("JOIN user_roles ur ON ur.role_id = roles.role_id").
		Where("ur.user_id = ?", userID).
		Find(&roles).Error
	return roles, err
}

func (r *userRepository) FindRolesPaginated(filter GetUserRolesFilter) (*PaginationResult[Role], error) {
	// Filter by a subquery rather than a JOIN+Select. A custom Select("roles.*")
	// makes GORM's Count emit COUNT("roles"."*") (a quoted "*"), which Postgres
	// rejects; the subquery keeps Count as a plain count(*) and avoids the join
	// column ambiguity.
	roleIDs := r.DB().Table("user_roles").Select("role_id").Where("user_id = ?", filter.UserID)
	query := r.DB().
		Model(&Role{}).
		Where("roles.role_id IN (?)", roleIDs)

	query = database.ApplyILike(query, "roles.name", filter.Name)
	query = database.ApplyILike(query, "roles.description", filter.Description)
	if filter.Status != nil && *filter.Status != "" {
		query = query.Where("roles.status = ?", *filter.Status)
	}

	query = query.Order(database.SanitizeOrderPrefixed("roles.", filter.SortBy, filter.SortOrder, "roles.created_at DESC"))

	query = query.Preload("RolePermissions.Permission")
	return database.PaginateQuery[Role](query, filter.Page, filter.Limit)
}

// FindBySubAndClientID deliberately does NOT filter on users.status.
//
// It is tempting to add `users.status = 'active'` here, since this is the one
// resolver every sub+client lookup funnels through — but it is not only an
// authentication resolver. Logout (authn.Logout) and RP-initiated/backchannel
// logout (oauth session EndSession) resolve the user through it purely to find
// the sessions and refresh tokens to revoke. Filtering here would make those
// return nil for a suspended account, so suspending a user and then signing
// them out would silently revoke nothing and leave their sessions live —
// the exact opposite of the intent.
//
// The status gate therefore lives at the authentication decision points, where
// refusing is the correct outcome: userService.FindBySubAndClientID and
// userService.FindByUserID (see isAuthenticatable) for the request path, and
// the login/refresh services for token issuance.
func (r *userRepository) FindBySubAndClientID(sub string, clientID string) (*User, error) {
	var user User
	err := r.DB().
		Preload("UserIdentities.Tenant").
		Preload("UserIdentities.IdentityProvider").
		Preload("UserRoles.Role.RolePermissions.Permission").
		// Profile is preloaded so OIDC userinfo and other handlers can derive
		// the display name from profiles.first_name/last_name/display_name
		// (the users.fullname column was removed).
		Preload("Profile", "is_default = ?", true).
		Joins("JOIN user_identities ON users.user_id = user_identities.user_id").
		Joins("JOIN clients ON clients.identifier = ? AND clients.status = ? AND clients.deleted_at IS NULL", clientID, shared.StatusActive).
		Where("user_identities.sub = ?", sub).
		// Reachability is ONLY an enabled client_identity_providers connection.
		// There is deliberately no second branch matching the identity directly
		// to the client: that used to let a client keep authenticating a user
		// after its connection to the identity's provider had been disabled.
		Where(`clients.tenant_id = user_identities.tenant_id AND EXISTS (
				SELECT 1
				FROM client_identity_providers cip
				JOIN identity_providers idp
					ON idp.identity_provider_id = cip.identity_provider_id
				WHERE cip.client_id = clients.client_id
					AND cip.identity_provider_id = user_identities.identity_provider_id
					AND cip.tenant_id = user_identities.tenant_id
					AND cip.enabled = TRUE
					AND cip.deleted_at IS NULL
					AND idp.deleted_at IS NULL
					AND idp.status = ?
			)`, shared.StatusActive).
		First(&user).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &user, nil
}

// EffectivePermissionNames resolves what the actor can actually do in a tenant
// right now, filtered exactly the way the request auth context is filtered so a
// deleted or deactivated grant cannot satisfy the escalation guard.
func (r *userRepository) EffectivePermissionNames(userID, tenantID int64) ([]string, error) {
	var names []string
	err := r.DB().
		Table("user_roles").
		Joins("JOIN roles ON roles.role_id = user_roles.role_id").
		Joins("JOIN role_permissions ON role_permissions.role_id = roles.role_id").
		Joins("JOIN permissions ON permissions.permission_id = role_permissions.permission_id").
		Where("user_roles.user_id = ?", userID).
		Where("roles.tenant_id = ? AND roles.deleted_at IS NULL AND roles.status = ?", tenantID, shared.StatusActive).
		Where("permissions.deleted_at IS NULL AND permissions.status = ?", shared.StatusActive).
		Distinct().
		Pluck("permissions.name", &names).Error
	return names, err
}

func (r *userRepository) SetEmailVerified(userUUID uuid.UUID, verified bool) error {
	updates := map[string]any{"is_email_verified": verified}
	if verified {
		updates["email_verified_at"] = time.Now()
	}
	return r.DB().Model(&User{}).
		Where("user_uuid = ?", userUUID).
		Updates(updates).Error
}

func (r *userRepository) SetStatus(userUUID uuid.UUID, status string) error {
	return r.DB().Model(&User{}).
		Where("user_uuid = ?", userUUID).
		Update("status", status).Error
}

func (r *userRepository) SetForcePasswordChange(userUUID uuid.UUID, force bool) error {
	return r.DB().Model(&User{}).
		Where("user_uuid = ?", userUUID).
		Update("force_password_change", force).Error
}

// UpdateEmail moves the account to a new address and advances its verification
// state in the same statement.
//
// The only caller is the OTP-confirmed email change, where the user has just
// proved control of the new mailbox. Writing the email column alone left
// is_email_verified / email_verified_at describing the OLD address: a verified
// user silently became "verified" for a mailbox they no longer own, and an
// unverified one stayed blocked by enforceLoginEmailVerification with no way
// forward — the verification they had just completed did not count. Atomic
// because a half-applied change is exactly the state that strands the account.
func (r *userRepository) UpdateEmail(userUUID uuid.UUID, email string) error {
	return r.DB().Model(&User{}).
		Where("user_uuid = ?", userUUID).
		Updates(map[string]any{
			"email":             email,
			"is_email_verified": true,
			"email_verified_at": time.Now(),
		}).Error
}

func (r *userRepository) UpdateUsername(userUUID uuid.UUID, username string) error {
	return r.DB().Model(&User{}).
		Where("user_uuid = ?", userUUID).
		Update("username", username).Error
}

func (r *userRepository) FindPaginated(filter UserRepositoryGetFilter) (*PaginationResult[User], error) {
	query := r.DB().Model(&User{})

	// Preload the default Profile so the computed `fullname` (Profile.display_name
	// → first_name/last_name) is populated on every listed user. The users.fullname
	// column was removed; without this the derived name is always empty.
	query = query.Preload("Profile", "is_default = ?", true)

	// Filter by user_identities fields (tenant, client) — join once to avoid duplicates.
	if filter.TenantID != nil || filter.ClientID != nil {
		query = query.Joins("JOIN user_identities ON users.user_id = user_identities.user_id")
		if filter.TenantID != nil {
			query = query.Where("user_identities.tenant_id = ?", *filter.TenantID)
		}
		if filter.ClientID != nil {
			// "Users of this client" = users holding an identity from a provider the
			// client has an enabled connection to. Identities are not owned by a
			// client, so there is no column to compare against. DISTINCT because a
			// user may reach one client through several providers.
			query = query.Distinct("users.*").
				Joins(`JOIN client_identity_providers cip
					ON cip.identity_provider_id = user_identities.identity_provider_id
					AND cip.tenant_id = user_identities.tenant_id
					AND cip.enabled = TRUE
					AND cip.deleted_at IS NULL`).
				Joins(`JOIN identity_providers idp
					ON idp.identity_provider_id = user_identities.identity_provider_id
					AND idp.deleted_at IS NULL
					AND idp.status = ?`, shared.StatusActive).
				Where("cip.client_id = ?", *filter.ClientID)
		}
	}

	// Apply filters
	if filter.Search != nil && *filter.Search != "" {
		like := "%" + strings.ToLower(*filter.Search) + "%"
		query = query.Joins("LEFT JOIN profiles ON users.user_id = profiles.user_id AND profiles.is_default = true AND profiles.deleted_at IS NULL")
		query = query.Where(
			"LOWER(users.username) LIKE ? OR LOWER(users.email) LIKE ? OR users.phone LIKE ? OR LOWER(profiles.first_name) LIKE ? OR LOWER(profiles.last_name) LIKE ? OR LOWER(profiles.display_name) LIKE ?",
			like, like, like, like, like, like,
		)
	} else {
		query = database.ApplyILike(query, "users.username", filter.Username)
		query = database.ApplyILike(query, "users.email", filter.Email)
		query = database.ApplyILike(query, "users.phone", filter.Phone)
		if filter.Fullname != nil && *filter.Fullname != "" {
			query = query.Joins("JOIN profiles ON users.user_id = profiles.user_id AND profiles.is_default = true")
			like := "%" + strings.ToLower(*filter.Fullname) + "%"
			query = query.Where(
				"LOWER(profiles.first_name) LIKE ? OR LOWER(profiles.last_name) LIKE ? OR LOWER(profiles.display_name) LIKE ?",
				like, like, like,
			)
		}
	}
	if len(filter.Status) > 0 {
		query = query.Where("users.status IN ?", filter.Status)
	}
	if filter.RoleID != nil {
		query = query.Joins("JOIN user_roles ON users.user_id = user_roles.user_id").Where("user_roles.role_id = ?", *filter.RoleID)
	}

	// Apply sorting — protected against SQL injection via allowlist
	query = query.Order(database.SanitizeOrderPrefixed("users.", filter.SortBy, filter.SortOrder, "users.created_at DESC"))

	afterID := int64(0)
	if filter.Cursor != nil {
		afterID = *filter.Cursor
	}
	// Qualified: the tenant/client filters join user_identities, which also has a
	// user_id column, so a bare "user_id" makes the keyset predicate ambiguous
	// (42702) and every page after the first fails.
	rows, nextCursor, err := database.PaginateKeyset[User](query, afterID, filter.Limit, "users.user_id", func(u User) int64 { return u.UserID })
	if err != nil {
		return nil, err
	}

	return &PaginationResult[User]{
		Data:       rows,
		Limit:      filter.Limit,
		NextCursor: nextCursor,
	}, nil
}
