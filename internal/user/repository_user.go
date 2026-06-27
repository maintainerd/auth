package user

import (
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/platform/database"
	"github.com/maintainerd/auth/internal/shared"
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
	FindByPendingEmailAndTenantID(email string, tenantID int64) (*User, error)
	FindSuperAdmin() (*User, error)
	FindRoles(userID int64) ([]Role, error)
	FindRolesPaginated(filter GetUserRolesFilter) (*PaginationResult[Role], error)
	FindBySubAndClientID(sub string, clientID string) (*User, error)
	FindPaginated(filter UserRepositoryGetFilter) (*PaginationResult[User], error)
	SetEmailVerified(userUUID uuid.UUID, verified bool) error
	SetStatus(userUUID uuid.UUID, status string) error
	// Feature: Force password change
	SetForcePasswordChange(userUUID uuid.UUID, force bool) error
	// Feature: Email change with OTP re-verification
	SetPendingEmail(userUUID uuid.UUID, pendingEmail, otp string, expiresAt time.Time) error
	ClearEmailChange(userUUID uuid.UUID) error
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

func (r *userRepository) FindByPendingEmailAndTenantID(email string, tenantID int64) (*User, error) {
	var user User
	err := r.DB().
		Where("pending_email = ? AND tenant_id = ?", email, tenantID).
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

	return database.PaginateQuery[Role](query, filter.Page, filter.Limit)
}

func (r *userRepository) FindBySubAndClientID(sub string, clientID string) (*User, error) {
	var user User
	err := r.DB().
		Preload("UserIdentities.Tenant").
		Preload("UserIdentities.Client").
		Preload("UserRoles.Role.RolePermissions.Permission").
		// Profile is preloaded so OIDC userinfo and other handlers can derive
		// the display name from profiles.first_name/last_name/display_name
		// (the users.fullname column was removed).
		Preload("Profile", "is_default = ?", true).
		Joins("JOIN user_identities ON users.user_id = user_identities.user_id").
		Joins("JOIN clients ON clients.identifier = ? AND clients.status = ?", clientID, shared.StatusActive).
		Where("user_identities.sub = ?", sub).
		Where(`clients.tenant_id = user_identities.tenant_id AND (
				user_identities.client_id = clients.client_id OR EXISTS (
					SELECT 1
					FROM client_identity_providers cip
					WHERE cip.client_id = clients.client_id
						AND cip.identity_provider_id = user_identities.identity_provider_id
						AND cip.enabled = TRUE
						AND cip.deleted_at IS NULL
				)
			)`).
		First(&user).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &user, nil
}

func (r *userRepository) SetEmailVerified(userUUID uuid.UUID, verified bool) error {
	return r.DB().Model(&User{}).
		Where("user_uuid = ?", userUUID).
		Update("is_email_verified", verified).Error
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

func (r *userRepository) SetPendingEmail(userUUID uuid.UUID, pendingEmail, otp string, expiresAt time.Time) error {
	return r.DB().Model(&User{}).
		Where("user_uuid = ?", userUUID).
		Updates(map[string]interface{}{
			"pending_email":               pendingEmail,
			"email_change_otp":            otp,
			"email_change_otp_expires_at": expiresAt,
		}).Error
}

func (r *userRepository) ClearEmailChange(userUUID uuid.UUID) error {
	return r.DB().Model(&User{}).
		Where("user_uuid = ?", userUUID).
		Updates(map[string]interface{}{
			"pending_email":               nil,
			"email_change_otp":            nil,
			"email_change_otp_expires_at": nil,
		}).Error
}

func (r *userRepository) UpdateEmail(userUUID uuid.UUID, email string) error {
	return r.DB().Model(&User{}).
		Where("user_uuid = ?", userUUID).
		Update("email", email).Error
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
	// column was removed; without this the derived name is always empty. Scoped to
	// the default profile to match FindBySubAndClientID and keep the has-one load
	// deterministic when a user has multiple profiles.
	query = query.Preload("Profile", "is_default = ?", true)

	// Filter by user_identities fields (tenant, client) — join once to avoid duplicates.
	if filter.TenantID != nil || filter.ClientID != nil {
		query = query.Joins("JOIN user_identities ON users.user_id = user_identities.user_id")
		if filter.TenantID != nil {
			query = query.Where("user_identities.tenant_id = ?", *filter.TenantID)
		}
		if filter.ClientID != nil {
			query = query.Where("user_identities.client_id = ?", *filter.ClientID)
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

	return database.PaginateQuery[User](query, filter.Page, filter.Limit)
}
