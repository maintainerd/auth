package service

import (
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/dto"
	"github.com/maintainerd/auth/internal/model"
	"github.com/maintainerd/auth/internal/repository"
	"gorm.io/gorm"
)

type ProfileService interface {
	CreateOrUpdateProfile(userID int64, req *dto.ProfileRequest) (*model.Profile, error)
	GetProfileByUserID(userID int64) (*model.Profile, error)
	DeleteProfile(userID int64) error
}

type profileService struct {
	db          *gorm.DB
	profileRepo repository.ProfileRepository
}

func NewProfileService(
	db *gorm.DB,
	profileRepo repository.ProfileRepository,
) ProfileService {
	return &profileService{
		db:          db,
		profileRepo: profileRepo,
	}
}

func (s *profileService) CreateOrUpdateProfile(userID int64, req *dto.ProfileRequest) (*model.Profile, error) {
	var updatedProfile *model.Profile

	err := s.db.Transaction(func(tx *gorm.DB) error {
		// Step 1: Try to find existing profile
		var profile model.Profile
		err := tx.Where("user_id = ?", userID).First(&profile).Error
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				// create new profile if not found
				profile = model.Profile{
					ProfileUUID: uuid.New(),
					UserID:      userID,
				}
			} else {
				return err
			}
		}

		// Step 2: Parse birthdate
		birthdate, err := parseBirthdate(req.Birthdate)
		if err != nil {
			return err
		}

		// Step 3: Set fields
		profile.FirstName = req.FirstName
		profile.MiddleName = req.MiddleName
		profile.LastName = req.LastName
		profile.Suffix = req.Suffix
		profile.Birthdate = birthdate
		profile.Gender = req.Gender
		profile.Phone = req.Phone
		profile.Email = req.Email
		profile.Address = req.Address
		profile.AvatarURL = req.AvatarURL
		profile.CoverURL = req.CoverURL

		// Step 4: Create or update
		if profile.ProfileID == 0 {
			if err := tx.Create(&profile).Error; err != nil {
				return err
			}
		} else {
			if err := tx.Model(&profile).Where("user_id = ?", userID).Updates(profile).Error; err != nil {
				return err
			}
		}

		updatedProfile = &profile
		return nil
	})

	if err != nil {
		return nil, err
	}

	return updatedProfile, nil
}

func (s *profileService) GetProfileByUserID(userID int64) (*model.Profile, error) {
	return s.profileRepo.FindByUserID(userID)
}

func (s *profileService) DeleteProfile(userID int64) error {
	return s.profileRepo.DeleteByUserID(userID)
}

// Helper for parsing birthdate string to *time.Time
func parseBirthdate(birthdateStr *string) (*time.Time, error) {
	if birthdateStr == nil || *birthdateStr == "" {
		return nil, nil
	}
	parsed, err := time.Parse("2006-01-02", *birthdateStr)
	if err != nil {
		return nil, errors.New("invalid birthdate format, must be YYYY-MM-DD")
	}
	return &parsed, nil
}
