package repository

import (
	"errors"

	"github.com/maintainerd/auth/internal/model"
	"gorm.io/gorm"
)

type ProfileRepository interface {
	BaseRepositoryMethods[model.Profile]
	WithTx(tx *gorm.DB) ProfileRepository
	FindByUserID(userID int64) (*model.Profile, error)
	UpdateByUserID(userID int64, updatedProfile *model.Profile) error
	DeleteByUserID(userID int64) error
}

type profileRepository struct {
	*BaseRepository[model.Profile]
	db *gorm.DB
}

func NewProfileRepository(db *gorm.DB) ProfileRepository {
	return &profileRepository{
		BaseRepository: NewBaseRepository[model.Profile](db, "profile_uuid", "profile_id"),
		db:             db,
	}
}

func (r *profileRepository) WithTx(tx *gorm.DB) ProfileRepository {
	return &profileRepository{
		BaseRepository: NewBaseRepository[model.Profile](tx, "profile_uuid", "profile_id"),
		db:             tx,
	}
}

func (r *profileRepository) FindByUserID(userID int64) (*model.Profile, error) {
	var profile model.Profile
	err := r.db.Where("user_id = ?", userID).First(&profile).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil // Return nil profile when not found
		}
		return nil, err
	}
	return &profile, nil
}

func (r *profileRepository) UpdateByUserID(userID int64, updatedProfile *model.Profile) error {
	return r.db.Model(&model.Profile{}).
		Where("user_id = ?", userID).
		Updates(updatedProfile).Error
}

func (r *profileRepository) DeleteByUserID(userID int64) error {
	return r.db.Where("user_id = ?", userID).Delete(&model.Profile{}).Error
}
