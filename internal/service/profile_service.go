package service

import (
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/model"
	"github.com/maintainerd/auth/internal/repository"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type ProfileServiceDataResult struct {
	ProfileUUID uuid.UUID
	// Basic Identity Information
	FirstName   string
	MiddleName  *string
	LastName    *string
	Suffix      *string
	DisplayName *string
	Bio         *string
	// Personal Information
	Birthdate *time.Time
	Gender    *string
	// Contact Information
	Phone   *string
	Email   *string
	Address *string
	// Location Information
	City    *string
	Country *string
	// Preference
	Timezone *string
	Language *string
	// Media & Assets (auth-centric)
	ProfileURL *string
	// Extended data
	Metadata map[string]interface{}
	// System Fields
	CreatedAt time.Time
	UpdatedAt time.Time
}

type ProfileService interface {
	CreateOrUpdateProfile(
		userUUID uuid.UUID,
		firstName string,
		middleName, lastName, suffix, displayName, bio *string,
		birthdate *time.Time,
		gender *string,
		phone, email, address *string,
		city, country *string,
		timezone, language *string,
		profileURL *string,
		metadata map[string]interface{},
	) (*ProfileServiceDataResult, error)
	GetByUUID(profileUUID uuid.UUID) (*ProfileServiceDataResult, error)
	GetByUserUUID(userUUID uuid.UUID) (*ProfileServiceDataResult, error)
	DeleteByUUID(profileUUID uuid.UUID) (*ProfileServiceDataResult, error)
}

type profileService struct {
	db          *gorm.DB
	profileRepo repository.ProfileRepository
	userRepo    repository.UserRepository
}

func NewProfileService(
	db *gorm.DB,
	profileRepo repository.ProfileRepository,
	userRepo repository.UserRepository,
) ProfileService {
	return &profileService{
		db:          db,
		profileRepo: profileRepo,
		userRepo:    userRepo,
	}
}

func (s *profileService) CreateOrUpdateProfile(
	userUUID uuid.UUID,
	firstName string,
	middleName, lastName, suffix, displayName, bio *string,
	birthdate *time.Time,
	gender *string,
	phone, email, address *string,
	city, country *string,
	timezone, language *string,
	profileURL *string,
	metadata map[string]interface{},
) (*ProfileServiceDataResult, error) {
	var updatedProfile *model.Profile

	err := s.db.Transaction(func(tx *gorm.DB) error {
		// Step 1: Create transaction-aware repositories
		txProfileRepo := s.profileRepo.WithTx(tx)
		txUserRepo := s.userRepo.WithTx(tx)

		// Step 2: Find user by UUID to get userID
		user, err := txUserRepo.FindByUUID(userUUID)
		if err != nil || user == nil {
			return errors.New("user not found")
		}

		// Step 3: Try to find existing profile using repository
		existingProfile, err := txProfileRepo.FindByUserID(user.UserID)
		var profile model.Profile

		if err != nil {
			return err
		} else if existingProfile == nil {
			// Create new profile if not found
			profile = model.Profile{
				ProfileUUID: uuid.New(),
				UserID:      user.UserID,
			}
		} else {
			// Use existing profile
			profile = *existingProfile
		}

		// Step 2: Set all fields
		// Basic Identity Information
		profile.FirstName = firstName
		profile.MiddleName = middleName
		profile.LastName = lastName
		profile.Suffix = suffix
		profile.DisplayName = displayName
		profile.Bio = bio

		// Personal Information
		profile.Birthdate = birthdate
		profile.Gender = gender

		// Contact Information
		profile.Phone = phone
		profile.Email = email
		profile.Address = address

		// Location Information
		profile.City = city
		profile.Country = country

		// Preference
		profile.Timezone = timezone
		profile.Language = language

		// Media & Assets (auth-centric)
		profile.ProfileURL = profileURL

		// Extended data - convert map to JSONB
		if metadata != nil {
			metadataBytes, err := json.Marshal(metadata)
			if err != nil {
				return err
			}
			profile.Metadata = metadataBytes
		} else {
			profile.Metadata = datatypes.JSON([]byte("{}"))
		}

		// Step 4: Create or update using transaction-aware repository
		if profile.ProfileID == 0 {
			// Create new profile
			createdProfile, err := txProfileRepo.Create(&profile)
			if err != nil {
				return err
			}
			updatedProfile = createdProfile
		} else {
			// Update existing profile
			err := txProfileRepo.UpdateByUserID(user.UserID, &profile)
			if err != nil {
				return err
			}
			updatedProfile = &profile
		}

		// Step 5: Update user's is_profile_completed flag
		_, err = txUserRepo.UpdateByUUID(user.UserUUID, map[string]interface{}{
			"is_profile_completed": true,
		})
		if err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return toProfileServiceDataResult(updatedProfile), nil
}

func (s *profileService) GetByUUID(profileUUID uuid.UUID) (*ProfileServiceDataResult, error) {
	profile, err := s.profileRepo.FindByUUID(profileUUID)
	if err != nil || profile == nil {
		return nil, errors.New("profile not found")
	}

	return toProfileServiceDataResult(profile), nil
}

func (s *profileService) GetByUserUUID(userUUID uuid.UUID) (*ProfileServiceDataResult, error) {
	// Find user by UUID to get userID
	user, err := s.userRepo.FindByUUID(userUUID)
	if err != nil || user == nil {
		return nil, errors.New("user not found")
	}

	profile, err := s.profileRepo.FindByUserID(user.UserID)
	if err != nil || profile == nil {
		return nil, errors.New("profile not found")
	}

	return toProfileServiceDataResult(profile), nil
}

func (s *profileService) DeleteByUUID(profileUUID uuid.UUID) (*ProfileServiceDataResult, error) {
	// First get the profile to return it
	profile, err := s.profileRepo.FindByUUID(profileUUID)
	if err != nil || profile == nil {
		return nil, errors.New("profile not found")
	}

	// Delete the profile
	err = s.profileRepo.DeleteByUUID(profileUUID)
	if err != nil {
		return nil, err
	}

	return toProfileServiceDataResult(profile), nil
}

// Helper functions
func toProfileServiceDataResult(profile *model.Profile) *ProfileServiceDataResult {
	if profile == nil {
		return nil
	}

	// Convert metadata JSONB to map
	var metadata map[string]interface{}
	if len(profile.Metadata) > 0 {
		if err := json.Unmarshal(profile.Metadata, &metadata); err != nil {
			metadata = nil
		}
	}

	result := &ProfileServiceDataResult{
		ProfileUUID: profile.ProfileUUID,
		// Basic Identity Information
		FirstName:   profile.FirstName,
		MiddleName:  profile.MiddleName,
		LastName:    profile.LastName,
		Suffix:      profile.Suffix,
		DisplayName: profile.DisplayName,
		Bio:         profile.Bio,
		// Personal Information
		Birthdate: profile.Birthdate,
		Gender:    profile.Gender,
		// Contact Information
		Phone:   profile.Phone,
		Email:   profile.Email,
		Address: profile.Address,
		// Location Information
		City:    profile.City,
		Country: profile.Country,
		// Preference
		Timezone: profile.Timezone,
		Language: profile.Language,
		// Media & Assets (auth-centric)
		ProfileURL: profile.ProfileURL,
		// Extended data
		Metadata: metadata,
		// System Fields
		CreatedAt: profile.CreatedAt,
		UpdatedAt: profile.UpdatedAt,
	}

	return result
}
