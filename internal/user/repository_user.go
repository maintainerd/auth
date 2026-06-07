package user

import (
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/platform/database"
	"github.com/maintainerd/auth/internal/shared"
	"gorm.io/gorm"
)

type UserRepositoryGetFilter struct {
	Username  *string
	Email     *string
	Phone     *string
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
	WithTx(tx *gorm.DB) UserRepository
	FindByUsername(username string) (*User, error)
	FindByEmail(email string) (*User, error)
	// FindByEmailAndTenantID finds a user by email scoped to a specific tenant
	// via user_identities. Use this in preference to FindByEmail whenever a
	// tenantID is available to avoid cross-tenant data leakage.
	FindByEmailAndTenantID(email string, tenantID int64) (*User, error)
	FindByPhone(phone string) (*User, error)
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
	FindByPendingEmail(email string) (*User, error)
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

func (r *userRepository) FindByUsername(username string) (*User, error) {
	var user User
	err := r.DB().
		Where("username = ?", username).
		First(&user).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}

	return &user, nil
}

func (r *userRepository) FindByEmail(email string) (*User, error) {
	var user User
	err := r.DB().
		Where("email = ?", email).
		First(&user).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}

	return &user, nil
}

func (r *userRepository) FindByEmailAndTenantID(email string, tenantID int64) (*User, error) {
	var user User
	err := r.DB().
		Joins("JOIN user_identities ON users.user_id = user_identities.user_id").
		Where("users.email = ? AND user_identities.tenant_id = ?", email, tenantID).
		First(&user).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}

	return &user, nil
}

func (r *userRepository) FindByPhone(phone string) (*User, error) {
	var user User
	err := r.DB().
		Where("phone = ?", phone).
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
	query := r.DB().
		Model(&Role{}).
		Select("roles.*").
		Joins("JOIN user_roles ur ON ur.role_id = roles.role_id").
		Where("ur.user_id = ?", filter.UserID)

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
		Preload("UserIdentities.Client.IdentityProvider.Tenant").
		Preload("UserIdentities.Client.IdentityProvider").
		Preload("UserIdentities.Client").
		Preload("UserRoles.Role.RolePermissions.Permission").
		// Profile is preloaded so OIDC userinfo and other handlers can derive
		// the display name from profiles.first_name/last_name/display_name
		// (the users.fullname column was removed).
		Preload("Profile", "is_default = ?", true).
		Joins("JOIN user_identities ON users.user_id = user_identities.user_id").
		Joins("JOIN clients ON user_identities.client_id = clients.client_id").
		Where("user_identities.sub = ? AND clients.identifier = ?", sub, clientID).
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

func (r *userRepository) FindByPendingEmail(email string) (*User, error) {
	var user User
	err := r.DB().
		Where("pending_email = ?", email).
		First(&user).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &user, nil
}

func (r *userRepository) FindPaginated(filter UserRepositoryGetFilter) (*PaginationResult[User], error) {
	query := r.DB().Model(&User{})

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
	query = database.ApplyILike(query, "users.username", filter.Username)
	query = database.ApplyILike(query, "users.email", filter.Email)
	query = database.ApplyILike(query, "users.phone", filter.Phone)
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
