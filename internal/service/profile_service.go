package service

import (
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/dto"
	"github.com/maintainerd/auth/internal/model"
	"github.com/maintainerd/auth/internal/repository"
)

type ProfileService interface {
	CreateOrUpdateProfile(userID int64, req *dto.ProfileRequest) (*model.Profile, error)
	GetProfileByUserID(userID int64) (*model.Profile, error)
	DeleteProfile(userID int64) error
}

type profileService struct {
	profileRepo repository.ProfileRepository
}

func NewProfileService(profileRepo repository.ProfileRepository) ProfileService {
	return &profileService{profileRepo}
}

func (s *profileService) CreateOrUpdateProfile(userID int64, req *dto.ProfileRequest) (*model.Profile, error) {
	profile, err := s.profileRepo.FindByUserID(userID)
	if err != nil || profile == nil {
		profile = &model.Profile{
			ProfileUUID: uuid.New(),
			UserID:      userID,
		}
	}

	birthdate, err := parseBirthdate(req.Birthdate)
	if err != nil {
		return nil, err
	}

	// Set profile fields from request
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

	if profile.ProfileID == 0 {
		createdProfile, err := s.profileRepo.Create(profile)
		if err != nil {
			return nil, err
		}
		profile = createdProfile
	} else {
		if err := s.profileRepo.UpdateByUserID(userID, profile); err != nil {
			return nil, err
		}
	}

	return profile, nil
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
