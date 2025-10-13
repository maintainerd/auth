package service

import (
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/model"
	"github.com/maintainerd/auth/internal/repository"
	"gorm.io/gorm"
)

type ProfileServiceDataResult struct {
	ProfileUUID   uuid.UUID
	FirstName     string
	MiddleName    *string
	LastName      *string
	Suffix        *string
	DisplayName   *string
	Birthdate     *time.Time
	Gender        *string
	Bio           *string
	Phone         *string
	Email         *string
	AddressLine1  *string
	AddressLine2  *string
	City          *string
	StateProvince *string
	PostalCode    *string
	Country       *string
	CountryName   *string
	Company       *string
	JobTitle      *string
	Department    *string
	Industry      *string
	WebsiteURL    *string
	AvatarURL     *string
	CoverURL      *string
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

type ProfileService interface {
	CreateOrUpdateProfile(
		userUUID uuid.UUID,
		firstName string,
		middleName, lastName, suffix, displayName *string,
		birthdate *time.Time,
		gender, bio *string,
		phone, email *string,
		addressLine1, addressLine2, city, stateProvince, postalCode, country, countryName *string,
		company, jobTitle, department, industry, websiteURL *string,
		avatarURL, coverURL *string,
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
	middleName, lastName, suffix, displayName *string,
	birthdate *time.Time,
	gender, bio *string,
	phone, email *string,
	addressLine1, addressLine2, city, stateProvince, postalCode, country, countryName *string,
	company, jobTitle, department, industry, websiteURL *string,
	avatarURL, coverURL *string,
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
			if errors.Is(err, gorm.ErrRecordNotFound) {
				// Create new profile if not found
				profile = model.Profile{
					ProfileUUID: uuid.New(),
					UserID:      user.UserID,
				}
			} else {
				return err
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

		// Personal Information
		profile.Birthdate = birthdate
		profile.Gender = gender
		profile.Bio = bio

		// Contact Information
		profile.Phone = phone
		profile.Email = email

		// Address Information
		profile.AddressLine1 = addressLine1
		profile.AddressLine2 = addressLine2
		profile.City = city
		profile.StateProvince = stateProvince
		profile.PostalCode = postalCode
		profile.Country = country
		profile.CountryName = countryName

		// Professional Information
		profile.Company = company
		profile.JobTitle = jobTitle
		profile.Department = department
		profile.Industry = industry
		profile.WebsiteURL = websiteURL

		// Media & Assets
		profile.AvatarURL = avatarURL
		profile.CoverURL = coverURL

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

	result := &ProfileServiceDataResult{
		ProfileUUID:   profile.ProfileUUID,
		FirstName:     profile.FirstName,
		MiddleName:    profile.MiddleName,
		LastName:      profile.LastName,
		Suffix:        profile.Suffix,
		DisplayName:   profile.DisplayName,
		Birthdate:     profile.Birthdate,
		Gender:        profile.Gender,
		Bio:           profile.Bio,
		Phone:         profile.Phone,
		Email:         profile.Email,
		AddressLine1:  profile.AddressLine1,
		AddressLine2:  profile.AddressLine2,
		City:          profile.City,
		StateProvince: profile.StateProvince,
		PostalCode:    profile.PostalCode,
		Country:       profile.Country,
		CountryName:   profile.CountryName,
		Company:       profile.Company,
		JobTitle:      profile.JobTitle,
		Department:    profile.Department,
		Industry:      profile.Industry,
		WebsiteURL:    profile.WebsiteURL,
		AvatarURL:     profile.AvatarURL,
		CoverURL:      profile.CoverURL,
		CreatedAt:     profile.CreatedAt,
		UpdatedAt:     profile.UpdatedAt,
	}

	return result
}
