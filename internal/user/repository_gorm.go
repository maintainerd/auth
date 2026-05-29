package user

import (
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/shared"
	"gorm.io/gorm"
)

type ProfileRepositoryGetFilter struct {
	UserID    int64
	FirstName *string
	LastName  *string
	Email     *string
	Phone     *string
	City      *string
	Country   *string
	IsDefault *bool
	Page      int
	Limit     int
	SortBy    string
	SortOrder string
}

type ProfileRepository interface {
	BaseRepositoryMethods[Profile]
	WithTx(tx *gorm.DB) ProfileRepository
	FindByUserID(userID int64) (*Profile, error)
	FindDefaultByUserID(userID int64) (*Profile, error)
	FindAllByUserID(filter ProfileRepositoryGetFilter) (*PaginationResult[Profile], error)
	UpdateByUserID(userID int64, updatedProfile *Profile) error
	DeleteByUserID(userID int64) error
	UnsetDefaultProfiles(userID int64) error
}

type profileRepository struct {
	*BaseRepository[Profile]
}

func NewProfileRepository(db *gorm.DB) ProfileRepository {
	return &profileRepository{
		BaseRepository: NewBaseRepository[Profile](db, "profile_uuid", "profile_id"),
	}
}

func (r *profileRepository) WithTx(tx *gorm.DB) ProfileRepository {
	return &profileRepository{
		BaseRepository: r.BaseRepository.WithTx(tx),
	}
}

func (r *profileRepository) FindByUserID(userID int64) (*Profile, error) {
	var profile Profile
	err := r.DB().Where("user_id = ?", userID).First(&profile).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil // Return nil profile when not found
		}
		return nil, err
	}
	return &profile, nil
}

func (r *profileRepository) FindDefaultByUserID(userID int64) (*Profile, error) {
	var profile Profile
	err := r.DB().Where("user_id = ? AND is_default = ?", userID, true).First(&profile).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil // Return nil profile when not found
		}
		return nil, err
	}
	return &profile, nil
}

func (r *profileRepository) FindAllByUserID(filter ProfileRepositoryGetFilter) (*PaginationResult[Profile], error) {
	var profiles []Profile
	var total int64

	query := r.DB().Model(&Profile{}).Where("user_id = ?", filter.UserID)

	// Apply filters
	if filter.FirstName != nil && *filter.FirstName != "" {
		query = query.Where("LOWER(first_name) LIKE ?", "%"+strings.ToLower(*filter.FirstName)+"%")
	}
	if filter.LastName != nil && *filter.LastName != "" {
		query = query.Where("LOWER(last_name) LIKE ?", "%"+strings.ToLower(*filter.LastName)+"%")
	}
	if filter.Email != nil && *filter.Email != "" {
		// Email lives on users (not profiles) since the column was removed —
		// join to users to preserve the existing filter API.
		query = query.Joins("JOIN users ON users.user_id = profiles.user_id").
			Where("LOWER(users.email) LIKE ?", "%"+strings.ToLower(*filter.Email)+"%")
	}
	if filter.Phone != nil && *filter.Phone != "" {
		query = query.Where("phone LIKE ?", "%"+*filter.Phone+"%")
	}
	if filter.City != nil && *filter.City != "" {
		query = query.Where("LOWER(city) LIKE ?", "%"+strings.ToLower(*filter.City)+"%")
	}
	if filter.Country != nil && *filter.Country != "" {
		query = query.Where("LOWER(country) = ?", strings.ToLower(*filter.Country))
	}
	if filter.IsDefault != nil {
		query = query.Where("is_default = ?", *filter.IsDefault)
	}

	// Count total
	if err := query.Count(&total).Error; err != nil {
		return nil, err
	}

	// Apply sorting — protected against SQL injection via allowlist
	query = query.Order(sanitizeOrder(filter.SortBy, filter.SortOrder, "is_default DESC, created_at DESC"))

	// Apply pagination
	if filter.Page < 1 {
		filter.Page = 1
	}
	if filter.Limit < 1 {
		filter.Limit = 20
	}
	offset := (filter.Page - 1) * filter.Limit
	if err := query.Offset(offset).Limit(filter.Limit).Find(&profiles).Error; err != nil {
		return nil, err
	}

	totalPages := int((total + int64(filter.Limit) - 1) / int64(filter.Limit))
	return &PaginationResult[Profile]{
		Data:       profiles,
		Total:      total,
		Page:       filter.Page,
		Limit:      filter.Limit,
		TotalPages: totalPages,
	}, nil
}

func (r *profileRepository) UpdateByUserID(userID int64, updatedProfile *Profile) error {
	return r.DB().Model(&Profile{}).
		Where("user_id = ?", userID).
		Updates(updatedProfile).Error
}

func (r *profileRepository) DeleteByUserID(userID int64) error {
	return r.DB().Where("user_id = ?", userID).Delete(&Profile{}).Error
}
func (r *profileRepository) UnsetDefaultProfiles(userID int64) error {
	return r.DB().Model(&Profile{}).
		Where("user_id = ? AND is_default = ?", userID, true).
		Update("is_default", false).Error
}

type UserRepositoryGetFilter struct {
	Username   *string
	Email      *string
	Phone      *string
	Status     []string
	TenantID   *int64
	RoleID     *int64
	ClientID   *int64
	UserPoolID *int64
	Page       int
	Limit      int
	SortBy     string
	SortOrder  string
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
		BaseRepository: NewBaseRepository[User](db, "user_uuid", "user_id"),
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
		Where("tenants.status = ? AND tenants.is_default = ?", shared.StatusActive, true).
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

func (r *userRepository) FindBySubAndClientID(sub string, clientID string) (*User, error) {
	var user User
	err := r.DB().
		Preload("UserIdentities.Tenant").
		Preload("UserIdentities.Client.IdentityProvider.Tenant").
		Preload("UserIdentities.Client.IdentityProvider").
		Preload("UserIdentities.Client").
		Preload("Roles.Permissions").
		// Profile is preloaded so OIDC userinfo and other handlers can derive
		// the display name from profiles.first_name/last_name/display_name
		// (the users.fullname column was removed).
		Preload("Profile", "is_default = ?", true).
		Joins("JOIN user_identities ON users.user_id = user_identities.user_id").
		Joins("JOIN clients ON user_identities.client_id = clients.client_id").
		Where("user_identities.sub = ? AND clients.client_id = ?", sub, clientID).
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
	var users []User
	var total int64

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
		// TODO(Phase 2): add user_identities.user_pool_id filter once the column exists.
	}

	// Apply filters
	if filter.Username != nil {
		query = query.Where("users.username ILIKE ?", "%"+*filter.Username+"%")
	}
	if filter.Email != nil {
		query = query.Where("users.email ILIKE ?", "%"+*filter.Email+"%")
	}
	if filter.Phone != nil {
		query = query.Where("users.phone ILIKE ?", "%"+*filter.Phone+"%")
	}
	if len(filter.Status) > 0 {
		query = query.Where("users.status IN ?", filter.Status)
	}
	if filter.RoleID != nil {
		query = query.Joins("JOIN user_roles ON users.user_id = user_roles.user_id").Where("user_roles.role_id = ?", *filter.RoleID)
	}

	// Count total records
	if err := query.Count(&total).Error; err != nil {
		return nil, err
	}

	// Apply sorting — protected against SQL injection via allowlist
	query = query.Order(sanitizeOrderPrefixed("users.", filter.SortBy, filter.SortOrder, "users.created_at DESC"))

	// Apply pagination
	filter.Page, filter.Limit = normalizePagination(filter.Page, filter.Limit)
	offset := (filter.Page - 1) * filter.Limit
	if err := query.Offset(offset).Limit(filter.Limit).Find(&users).Error; err != nil {
		return nil, err
	}

	totalPages := int((total + int64(filter.Limit) - 1) / int64(filter.Limit))

	return &PaginationResult[User]{
		Data:       users,
		Total:      total,
		Page:       filter.Page,
		Limit:      filter.Limit,
		TotalPages: totalPages,
	}, nil
}

type UserIdentityRepository interface {
	BaseRepositoryMethods[UserIdentity]
	WithTx(tx *gorm.DB) UserIdentityRepository
	FindByUserID(userID int64) ([]UserIdentity, error)
	FindByUserIDAndClientID(userID int64, clientID int64) (*UserIdentity, error)
	// FindByProviderAndSub looks up an identity by the provider slug and the external subject.
	// Used by federation to match an incoming OIDC token to a known user.
	FindByProviderAndSub(provider string, sub string) (*UserIdentity, error)
	// FindByUserIDAndProvider returns the first identity for a user with the given provider slug.
	FindByUserIDAndProvider(userID int64, provider string) (*UserIdentity, error)
	// FindByIdentityProviderID lists all identities linked to a configured IDP.
	FindByIdentityProviderID(idpID int64) ([]UserIdentity, error)
	DeleteByUserID(userID int64) error
}

type userIdentityRepository struct {
	*BaseRepository[UserIdentity]
}

func NewUserIdentityRepository(db *gorm.DB) UserIdentityRepository {
	return &userIdentityRepository{
		BaseRepository: NewBaseRepository[UserIdentity](db, "user_identity_uuid", "user_identity_id"),
	}
}

func (r *userIdentityRepository) WithTx(tx *gorm.DB) UserIdentityRepository {
	return &userIdentityRepository{
		BaseRepository: r.BaseRepository.WithTx(tx),
	}
}

func (r *userIdentityRepository) FindByUserID(userID int64) ([]UserIdentity, error) {
	var identities []UserIdentity
	err := r.DB().Where("user_id = ?", userID).Find(&identities).Error
	return identities, err
}

func (r *userIdentityRepository) FindByUserIDAndClientID(userID int64, clientID int64) (*UserIdentity, error) {
	var identity UserIdentity
	err := r.DB().Where("user_id = ? AND client_id = ?", userID, clientID).First(&identity).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &identity, nil
}

func (r *userIdentityRepository) FindByProviderAndSub(provider string, sub string) (*UserIdentity, error) {
	var identity UserIdentity
	err := r.DB().Where("provider = ? AND sub = ?", provider, sub).First(&identity).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &identity, nil
}

func (r *userIdentityRepository) FindByUserIDAndProvider(userID int64, provider string) (*UserIdentity, error) {
	var identity UserIdentity
	err := r.DB().Where("user_id = ? AND provider = ?", userID, provider).First(&identity).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &identity, nil
}

func (r *userIdentityRepository) FindByIdentityProviderID(idpID int64) ([]UserIdentity, error) {
	var identities []UserIdentity
	err := r.DB().Where("identity_provider_id = ?", idpID).Find(&identities).Error
	return identities, err
}

func (r *userIdentityRepository) DeleteByUserID(userID int64) error {
	return r.DB().Where("user_id = ?", userID).Delete(&UserIdentity{}).Error
}

// UserPasswordHistoryRepository manages previously used password hashes for
// a user so services can enforce PasswordPolicy.HistoryCount.
type UserPasswordHistoryRepository interface {
	WithTx(tx *gorm.DB) UserPasswordHistoryRepository
	// AddEntry inserts a new hash record for the user.
	AddEntry(userID int64, hash string) error
	// FindRecentHashes returns the most recent `count` hashes for the user,
	// ordered newest first.
	FindRecentHashes(userID int64, count int) ([]string, error)
	// PruneExcess deletes all but the most recent `keepCount` records for the user.
	PruneExcess(userID int64, keepCount int) error
}

type userPasswordHistoryRepository struct {
	db *gorm.DB
}

func NewUserPasswordHistoryRepository(db *gorm.DB) UserPasswordHistoryRepository {
	return &userPasswordHistoryRepository{db: db}
}

func (r *userPasswordHistoryRepository) WithTx(tx *gorm.DB) UserPasswordHistoryRepository {
	return &userPasswordHistoryRepository{db: tx}
}

func (r *userPasswordHistoryRepository) AddEntry(userID int64, hash string) error {
	entry := UserPasswordHistory{
		UserID:       userID,
		PasswordHash: hash,
	}
	return r.db.Create(&entry).Error
}

func (r *userPasswordHistoryRepository) FindRecentHashes(userID int64, count int) ([]string, error) {
	var entries []UserPasswordHistory
	err := r.db.
		Where("user_id = ?", userID).
		Order("created_at DESC").
		Limit(count).
		Find(&entries).Error
	if err != nil {
		return nil, err
	}
	hashes := make([]string, len(entries))
	for i, e := range entries {
		hashes[i] = e.PasswordHash
	}
	return hashes, nil
}

func (r *userPasswordHistoryRepository) PruneExcess(userID int64, keepCount int) error {
	// Delete all rows except the keepCount most recent ones.
	return r.db.Exec(`
DELETE FROM user_password_history
WHERE user_id = ?
  AND history_id NOT IN (
      SELECT history_id FROM user_password_history
      WHERE user_id = ?
      ORDER BY created_at DESC
      LIMIT ?
  )`, userID, userID, keepCount).Error
}

// UserPoolRepository provides read and write access to the user_pools table.
// A user pool is the isolation namespace for users, roles, and settings within
// a single tenant deployment.
type UserPoolRepository interface {
	BaseRepositoryMethods[UserPool]
	WithTx(tx *gorm.DB) UserPoolRepository
	FindByIdentifier(tenantID int64, identifier string) (*UserPool, error)
	FindDefault(tenantID int64) (*UserPool, error)
	FindSystem(tenantID int64) (*UserPool, error)
	FindAllByTenantID(tenantID int64) ([]UserPool, error)
}

type userPoolRepository struct {
	*BaseRepository[UserPool]
}

// NewUserPoolRepository returns a UserPoolRepository backed by the given gorm.DB.
func NewUserPoolRepository(db *gorm.DB) UserPoolRepository {
	return &userPoolRepository{
		BaseRepository: NewBaseRepository[UserPool](db, "user_pool_uuid", "user_pool_id"),
	}
}

// WithTx returns a new UserPoolRepository scoped to the given transaction.
func (r *userPoolRepository) WithTx(tx *gorm.DB) UserPoolRepository {
	return &userPoolRepository{
		BaseRepository: r.BaseRepository.WithTx(tx),
	}
}

// FindByIdentifier retrieves a user pool by its slug within a tenant.
func (r *userPoolRepository) FindByIdentifier(tenantID int64, identifier string) (*UserPool, error) {
	var pool UserPool
	err := r.DB().
		Where("tenant_id = ? AND identifier = ? AND deleted_at IS NULL", tenantID, identifier).
		First(&pool).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &pool, nil
}

// FindDefault retrieves the default user pool for the given tenant.
func (r *userPoolRepository) FindDefault(tenantID int64) (*UserPool, error) {
	var pool UserPool
	err := r.DB().
		Where("tenant_id = ? AND is_default = ? AND deleted_at IS NULL", tenantID, true).
		First(&pool).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &pool, nil
}

// FindSystem retrieves the system user pool for the given tenant.
func (r *userPoolRepository) FindSystem(tenantID int64) (*UserPool, error) {
	var pool UserPool
	err := r.DB().
		Where("tenant_id = ? AND is_system = ? AND deleted_at IS NULL", tenantID, true).
		First(&pool).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &pool, nil
}

// FindAllByTenantID retrieves all non-deleted user pools belonging to a tenant.
func (r *userPoolRepository) FindAllByTenantID(tenantID int64) ([]UserPool, error) {
	var pools []UserPool
	err := r.DB().
		Where("tenant_id = ? AND deleted_at IS NULL", tenantID).
		Find(&pools).Error
	return pools, err
}

type UserRoleRepository interface {
	BaseRepositoryMethods[UserRole]
	WithTx(tx *gorm.DB) UserRoleRepository
	FindByUserID(userID int64) ([]UserRole, error)
	FindByUserIDAndRoleID(userID int64, roleID int64) (*UserRole, error)
	FindDefaultRolesByUserID(userID int64) ([]UserRole, error)
	DeleteByUserID(userID int64) error
	DeleteByUserIDAndRoleID(userID int64, roleID int64) error
}

type userRoleRepository struct {
	*BaseRepository[UserRole]
}

func NewUserRoleRepository(db *gorm.DB) UserRoleRepository {
	return &userRoleRepository{
		BaseRepository: NewBaseRepository[UserRole](db, "user_role_uuid", "user_role_id"),
	}
}

func (r *userRoleRepository) WithTx(tx *gorm.DB) UserRoleRepository {
	return &userRoleRepository{
		BaseRepository: r.BaseRepository.WithTx(tx),
	}
}

func (r *userRoleRepository) FindByUserID(userID int64) ([]UserRole, error) {
	var userRoles []UserRole
	err := r.DB().Where("user_id = ?", userID).Find(&userRoles).Error
	return userRoles, err
}

func (r *userRoleRepository) FindByUserIDAndRoleID(userID int64, roleID int64) (*UserRole, error) {
	var ur UserRole
	err := r.DB().
		Where("user_id = ? AND role_id = ?", userID, roleID).
		First(&ur).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &ur, nil
}

func (r *userRoleRepository) FindDefaultRolesByUserID(userID int64) ([]UserRole, error) {
	var userRoles []UserRole
	err := r.DB().
		Where("user_id = ? AND is_default = true", userID).
		Find(&userRoles).Error
	return userRoles, err
}

func (r *userRoleRepository) DeleteByUserID(userID int64) error {
	return r.DB().Where("user_id = ?", userID).Delete(&UserRole{}).Error
}

func (r *userRoleRepository) DeleteByUserIDAndRoleID(userID int64, roleID int64) error {
	return r.DB().
		Where("user_id = ? AND role_id = ?", userID, roleID).
		Delete(&UserRole{}).Error
}

type UserSettingRepository interface {
	BaseRepositoryMethods[UserSetting]
	WithTx(tx *gorm.DB) UserSettingRepository
	FindByUserID(userID int64) (*UserSetting, error)
	UpdateByUserID(userID int64, updatedUserSetting *UserSetting) error
	DeleteByUserID(userID int64) error
}

type userSettingRepository struct {
	*BaseRepository[UserSetting]
}

func NewUserSettingRepository(db *gorm.DB) UserSettingRepository {
	return &userSettingRepository{
		BaseRepository: NewBaseRepository[UserSetting](db, "user_setting_uuid", "user_setting_id"),
	}
}

func (r *userSettingRepository) WithTx(tx *gorm.DB) UserSettingRepository {
	return &userSettingRepository{
		BaseRepository: r.BaseRepository.WithTx(tx),
	}
}

func (r *userSettingRepository) FindByUserID(userID int64) (*UserSetting, error) {
	var userSetting UserSetting
	err := r.DB().Where("user_id = ?", userID).First(&userSetting).Error
	return &userSetting, err
}

func (r *userSettingRepository) UpdateByUserID(userID int64, updatedUserSetting *UserSetting) error {
	return r.DB().Model(&UserSetting{}).
		Where("user_id = ?", userID).
		Updates(updatedUserSetting).Error
}

func (r *userSettingRepository) DeleteByUserID(userID int64) error {
	return r.DB().Where("user_id = ?", userID).Delete(&UserSetting{}).Error
}

type UserTokenRepository interface {
	BaseRepositoryMethods[UserToken]
	WithTx(tx *gorm.DB) UserTokenRepository
	FindByUserID(userID int64) ([]UserToken, error)
	FindActiveTokensByUserID(userID int64) ([]UserToken, error)
	FindByUserIDAndTokenType(userID int64, tokenType string) ([]UserToken, error)
	RevokeByUUID(tokenUUID uuid.UUID) error
	RevokeAllByUserID(userID int64) error
	DeleteByUserID(userID int64) error
	DeleteExpiredTokens(before time.Time) error

	// Session-specific methods
	FindActiveSessions(userID int64) ([]UserToken, error)
	FindActiveSessionByUUID(userID int64, sessionUUID uuid.UUID) (*UserToken, error)
	CountActiveSessions(userID int64) (int64, error)
	TouchSession(sessionUUID uuid.UUID, now time.Time) error
	RevokeSessionByUUID(userID int64, sessionUUID uuid.UUID) error
	RevokeAllSessionsByUserID(userID int64) error
}

type userTokenRepository struct {
	*BaseRepository[UserToken]
}

func NewUserTokenRepository(db *gorm.DB) UserTokenRepository {
	return &userTokenRepository{
		BaseRepository: NewBaseRepository[UserToken](db, "user_token_uuid", "user_token_id"),
	}
}

func (r *userTokenRepository) WithTx(tx *gorm.DB) UserTokenRepository {
	return &userTokenRepository{
		BaseRepository: r.BaseRepository.WithTx(tx),
	}
}

func (r *userTokenRepository) FindByUserID(userID int64) ([]UserToken, error) {
	var tokens []UserToken
	err := r.DB().Where("user_id = ?", userID).Find(&tokens).Error
	return tokens, err
}

func (r *userTokenRepository) FindActiveTokensByUserID(userID int64) ([]UserToken, error) {
	var tokens []UserToken
	err := r.DB().
		Where("user_id = ? AND is_revoked = false AND (expires_at IS NULL OR expires_at > ?)", userID, time.Now()).
		Find(&tokens).Error
	return tokens, err
}

func (r *userTokenRepository) FindByUserIDAndTokenType(userID int64, tokenType string) ([]UserToken, error) {
	var tokens []UserToken
	err := r.DB().
		Where("user_id = ? AND token_type = ?", userID, tokenType).
		Find(&tokens).Error
	return tokens, err
}

func (r *userTokenRepository) RevokeByUUID(tokenUUID uuid.UUID) error {
	return r.DB().Model(&UserToken{}).
		Where("user_token_uuid = ?", tokenUUID).
		Update("is_revoked", true).Error
}

func (r *userTokenRepository) RevokeAllByUserID(userID int64) error {
	return r.DB().Model(&UserToken{}).
		Where("user_id = ?", userID).
		Update("is_revoked", true).Error
}

func (r *userTokenRepository) DeleteByUserID(userID int64) error {
	return r.DB().Where("user_id = ?", userID).Delete(&UserToken{}).Error
}

func (r *userTokenRepository) DeleteExpiredTokens(before time.Time) error {
	return r.DB().
		Where("expires_at IS NOT NULL AND expires_at < ?", before).
		Delete(&UserToken{}).Error
}

// FindActiveSessions returns all non-revoked, non-expired session tokens for
// the given user (token_type = 'user:session'), ordered oldest-first.
func (r *userTokenRepository) FindActiveSessions(userID int64) ([]UserToken, error) {
	var tokens []UserToken
	now := time.Now()
	err := r.DB().
		Where("user_id = ? AND token_type = ? AND is_revoked = false AND (absolute_expires_at IS NULL OR absolute_expires_at > ?)",
			userID, shared.TokenTypeSession, now).
		Order("created_at ASC").
		Find(&tokens).Error
	return tokens, err
}

// FindActiveSessionByUUID looks up a single active session by UUID with
// ownership check (must belong to userID).
func (r *userTokenRepository) FindActiveSessionByUUID(userID int64, sessionUUID uuid.UUID) (*UserToken, error) {
	var token UserToken
	now := time.Now()
	err := r.DB().
		Where("user_id = ? AND user_token_uuid = ? AND token_type = ? AND is_revoked = false AND (absolute_expires_at IS NULL OR absolute_expires_at > ?)",
			userID, sessionUUID, shared.TokenTypeSession, now).
		First(&token).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	return &token, err
}

// CountActiveSessions returns the number of active (non-revoked, non-expired)
// session tokens for the given user.
func (r *userTokenRepository) CountActiveSessions(userID int64) (int64, error) {
	var count int64
	now := time.Now()
	err := r.DB().Model(&UserToken{}).
		Where("user_id = ? AND token_type = ? AND is_revoked = false AND (absolute_expires_at IS NULL OR absolute_expires_at > ?)",
			userID, shared.TokenTypeSession, now).
		Count(&count).Error
	return count, err
}

// TouchSession updates last_used_at for the session identified by sessionUUID.
// This implements the sliding idle timeout: callers invoke this on every
// authenticated request.
func (r *userTokenRepository) TouchSession(sessionUUID uuid.UUID, now time.Time) error {
	return r.DB().Model(&UserToken{}).
		Where("user_token_uuid = ? AND token_type = ? AND is_revoked = false", sessionUUID, shared.TokenTypeSession).
		Update("last_used_at", now).Error
}

// RevokeSessionByUUID revokes a single session token with an ownership check.
// Returns nil when the session is not found (idempotent).
func (r *userTokenRepository) RevokeSessionByUUID(userID int64, sessionUUID uuid.UUID) error {
	return r.DB().Model(&UserToken{}).
		Where("user_id = ? AND user_token_uuid = ? AND token_type = ?", userID, sessionUUID, shared.TokenTypeSession).
		Update("is_revoked", true).Error
}

// RevokeAllSessionsByUserID revokes all session tokens for a user without
// touching non-session token types (e.g. email verification, password reset).
func (r *userTokenRepository) RevokeAllSessionsByUserID(userID int64) error {
	return r.DB().Model(&UserToken{}).
		Where("user_id = ? AND token_type = ?", userID, shared.TokenTypeSession).
		Update("is_revoked", true).Error
}
