package repository

import (
	"errors"
	"strings"

	"github.com/maintainerd/auth/internal/model"
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
	BaseRepositoryMethods[model.Profile]
	WithTx(tx *gorm.DB) ProfileRepository
	FindByUserID(userID int64) (*model.Profile, error)
	FindDefaultByUserID(userID int64) (*model.Profile, error)
	FindAllByUserID(filter ProfileRepositoryGetFilter) (*PaginationResult[model.Profile], error)
	UpdateByUserID(userID int64, updatedProfile *model.Profile) error
	DeleteByUserID(userID int64) error
	UnsetDefaultProfiles(userID int64) error
}

type profileRepository struct {
	*BaseRepository[model.Profile]
}

func NewProfileRepository(db *gorm.DB) ProfileRepository {
	return &profileRepository{
		BaseRepository: NewBaseRepository[model.Profile](db, "profile_uuid", "profile_id"),
	}
}

func (r *profileRepository) WithTx(tx *gorm.DB) ProfileRepository {
	return &profileRepository{
		BaseRepository: r.BaseRepository.WithTx(tx),
	}
}

func (r *profileRepository) FindByUserID(userID int64) (*model.Profile, error) {
	var profile model.Profile
	err := r.DB().Where("user_id = ?", userID).First(&profile).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil // Return nil profile when not found
		}
		return nil, err
	}
	return &profile, nil
}

func (r *profileRepository) FindDefaultByUserID(userID int64) (*model.Profile, error) {
	var profile model.Profile
	err := r.DB().Where("user_id = ? AND is_default = ?", userID, true).First(&profile).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil // Return nil profile when not found
		}
		return nil, err
	}
	return &profile, nil
}

func (r *profileRepository) FindAllByUserID(filter ProfileRepositoryGetFilter) (*PaginationResult[model.Profile], error) {
	var profiles []model.Profile
	var total int64

	query := r.DB().Model(&model.Profile{}).Where("user_id = ?", filter.UserID)

	// Apply filters
	if filter.FirstName != nil && *filter.FirstName != "" {
		query = query.Where("LOWER(first_name) LIKE ?", "%"+strings.ToLower(*filter.FirstName)+"%")
	}
	if filter.LastName != nil && *filter.LastName != "" {
		query = query.Where("LOWER(last_name) LIKE ?", "%"+strings.ToLower(*filter.LastName)+"%")
	}
	if filter.Email != nil && *filter.Email != "" {
		query = query.Where("LOWER(email) LIKE ?", "%"+strings.ToLower(*filter.Email)+"%")
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
	return &PaginationResult[model.Profile]{
		Data:       profiles,
		Total:      total,
		Page:       filter.Page,
		Limit:      filter.Limit,
		TotalPages: totalPages,
	}, nil
}

func (r *profileRepository) UpdateByUserID(userID int64, updatedProfile *model.Profile) error {
	return r.DB().Model(&model.Profile{}).
		Where("user_id = ?", userID).
		Updates(updatedProfile).Error
}

func (r *profileRepository) DeleteByUserID(userID int64) error {
	return r.DB().Where("user_id = ?", userID).Delete(&model.Profile{}).Error
}
func (r *profileRepository) UnsetDefaultProfiles(userID int64) error {
	return r.DB().Model(&model.Profile{}).
		Where("user_id = ? AND is_default = ?", userID, true).
		Update("is_default", false).Error
}
