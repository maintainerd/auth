package user

import (
	"errors"
	"strings"

	"github.com/maintainerd/auth/internal/platform/database"
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
	FindByUUID(uuid any, preloads ...string) (*Profile, error)
	DeleteByUUID(uuid any) error
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
		BaseRepository: database.NewBaseRepository[Profile](db, "profile_uuid", "profile_id"),
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
	query := r.DB().Model(&Profile{}).Where("user_id = ?", filter.UserID)

	// Apply filters
	if filter.FirstName != nil && *filter.FirstName != "" {
		query = database.ApplyILike(query, "first_name", filter.FirstName)
	}
	if filter.LastName != nil && *filter.LastName != "" {
		query = database.ApplyILike(query, "last_name", filter.LastName)
	}
	if filter.Email != nil && *filter.Email != "" {
		// Email lives on users (not profiles) since the column was removed —
		// join to users to preserve the existing filter API.
		query = query.Joins("JOIN users ON users.user_id = profiles.user_id")
		query = database.ApplyILike(query, "users.email", filter.Email)
	}
	if filter.Phone != nil && *filter.Phone != "" {
		query = database.ApplyILike(query, "phone", filter.Phone)
	}
	if filter.City != nil && *filter.City != "" {
		query = database.ApplyILike(query, "city", filter.City)
	}
	if filter.Country != nil && *filter.Country != "" {
		query = query.Where("LOWER(country) = ?", strings.ToLower(*filter.Country))
	}
	if filter.IsDefault != nil {
		query = query.Where("is_default = ?", *filter.IsDefault)
	}

	// Apply sorting — protected against SQL injection via allowlist
	query = query.Order(database.SanitizeOrder(filter.SortBy, filter.SortOrder, "is_default DESC, created_at DESC"))

	return database.PaginateQuery[Profile](query, filter.Page, filter.Limit)
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
