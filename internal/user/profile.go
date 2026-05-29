package user

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/platform/apperror"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
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
	// Profile Flags
	IsDefault bool
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
	Metadata map[string]any
	// System Fields
	CreatedAt time.Time
	UpdatedAt time.Time
}

type ProfileServiceListResult struct {
	Data       []ProfileServiceDataResult
	Total      int64
	Page       int
	Limit      int
	TotalPages int
}

type ProfileService interface {
	CreateOrUpdateProfile(
		ctx context.Context,
		userUUID uuid.UUID,
		firstName string,
		middleName, lastName, suffix, displayName, bio *string,
		birthdate *time.Time,
		gender *string,
		phone, email, address *string,
		city, country *string,
		timezone, language *string,
		profileURL *string,
		metadata map[string]any,
	) (*ProfileServiceDataResult, error)
	CreateOrUpdateSpecificProfile(
		ctx context.Context,
		profileUUID uuid.UUID,
		userUUID uuid.UUID,
		firstName string,
		middleName, lastName, suffix, displayName, bio *string,
		birthdate *time.Time,
		gender *string,
		phone, email, address *string,
		city, country *string,
		timezone, language *string,
		profileURL *string,
		metadata map[string]any,
	) (*ProfileServiceDataResult, error)
	GetByUUID(ctx context.Context, profileUUID uuid.UUID, userUUID uuid.UUID) (*ProfileServiceDataResult, error)
	GetByUserUUID(ctx context.Context, userUUID uuid.UUID) (*ProfileServiceDataResult, error)
	GetAll(ctx context.Context, userUUID uuid.UUID, firstName, lastName, email, phone, city, country *string, isDefault *bool, page, limit int, sortBy, sortOrder string) (*ProfileServiceListResult, error)
	SetDefaultProfile(ctx context.Context, profileUUID uuid.UUID, userUUID uuid.UUID) (*ProfileServiceDataResult, error)
	DeleteByUUID(ctx context.Context, profileUUID uuid.UUID, userUUID uuid.UUID) (*ProfileServiceDataResult, error)
}

type profileService struct {
	db          *gorm.DB
	profileRepo ProfileRepository
	userRepo    UserRepository
}

func NewProfileService(
	db *gorm.DB,
	profileRepo ProfileRepository,
	userRepo UserRepository,
) ProfileService {
	return &profileService{
		db:          db,
		profileRepo: profileRepo,
		userRepo:    userRepo,
	}
}

func (s *profileService) CreateOrUpdateProfile(
	ctx context.Context,
	userUUID uuid.UUID,
	firstName string,
	middleName, lastName, suffix, displayName, bio *string,
	birthdate *time.Time,
	gender *string,
	phone, email, address *string,
	city, country *string,
	timezone, language *string,
	profileURL *string,
	metadata map[string]any,
) (*ProfileServiceDataResult, error) {
	_, span := otel.Tracer("service").Start(ctx, "profile.createOrUpdate")
	defer span.End()
	span.SetAttributes(attribute.String("user.uuid", userUUID.String()))
	var updatedProfile *Profile

	err := s.db.Transaction(func(tx *gorm.DB) error {
		// Step 1: Create transaction-aware repositories
		txProfileRepo := s.profileRepo.WithTx(tx)
		txUserRepo := s.userRepo.WithTx(tx)

		// Step 2: Find user by UUID to get userID
		user, err := txUserRepo.FindByUUID(userUUID)
		if err != nil || user == nil {
			return apperror.NewNotFound("user not found")
		}

		// Step 3: Try to find existing default profile
		existingProfile, err := txProfileRepo.FindDefaultByUserID(user.UserID)
		var profile Profile

		if err != nil {
			return err
		} else if existingProfile == nil {
			// Create new profile if not found
			profile = Profile{
				ProfileUUID: uuid.New(),
				UserID:      user.UserID,
				IsDefault:   true, // First profile is always default
			}
		} else {
			// Use existing profile
			profile = *existingProfile
		}

		// Step 4: Set all fields
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
		_, err = txUserRepo.UpdateByUUID(user.UserUUID, map[string]any{
			"is_profile_completed": true,
		})
		if err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "create or update profile failed")
		return nil, err
	}

	span.SetStatus(codes.Ok, "")
	return toProfileServiceDataResult(updatedProfile), nil
}

func (s *profileService) CreateOrUpdateSpecificProfile(
	ctx context.Context,
	profileUUID uuid.UUID,
	userUUID uuid.UUID,
	firstName string,
	middleName, lastName, suffix, displayName, bio *string,
	birthdate *time.Time,
	gender *string,
	phone, email, address *string,
	city, country *string,
	timezone, language *string,
	profileURL *string,
	metadata map[string]any,
) (*ProfileServiceDataResult, error) {
	_, span := otel.Tracer("service").Start(ctx, "profile.createOrUpdateSpecific")
	defer span.End()
	span.SetAttributes(attribute.String("profile.uuid", profileUUID.String()), attribute.String("user.uuid", userUUID.String()))
	var updatedProfile *Profile

	err := s.db.Transaction(func(tx *gorm.DB) error {
		// Step 1: Create transaction-aware repositories
		txProfileRepo := s.profileRepo.WithTx(tx)
		txUserRepo := s.userRepo.WithTx(tx)

		// Step 2: Find user by UUID to get userID
		user, err := txUserRepo.FindByUUID(userUUID)
		if err != nil || user == nil {
			return apperror.NewNotFound("user not found")
		}

		// Step 3: Try to find existing profile by UUID
		existingProfile, err := txProfileRepo.FindByUUID(profileUUID)
		var profile Profile

		if err != nil {
			return err
		} else if existingProfile == nil {
			// Check if this is the first profile for the user
			existingUserProfile, err := txProfileRepo.FindByUserID(user.UserID)
			if err != nil {
				return err
			}

			// Create new profile with provided UUID
			profile = Profile{
				ProfileUUID: profileUUID,
				UserID:      user.UserID,
				IsDefault:   existingUserProfile == nil, // First profile is default
			}
		} else {
			// Verify profile belongs to user
			if existingProfile.UserID != user.UserID {
				return apperror.NewForbidden("profile does not belong to user")
			}
			// Use existing profile
			profile = *existingProfile
		}

		// Set all fields
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

		// Create or update profile
		if profile.ProfileID == 0 {
			// Create new profile
			createdProfile, err := txProfileRepo.Create(&profile)
			if err != nil {
				return err
			}
			updatedProfile = createdProfile

			// Update user's is_profile_completed flag on first profile creation
			_, err = txUserRepo.UpdateByUUID(user.UserUUID, map[string]any{
				"is_profile_completed": true,
			})
			if err != nil {
				return err
			}
		} else {
			// Update existing profile using CreateOrUpdate
			updated, err := txProfileRepo.CreateOrUpdate(&profile)
			if err != nil {
				return err
			}
			updatedProfile = updated
		}

		return nil
	})

	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "create or update specific profile failed")
		return nil, err
	}

	span.SetStatus(codes.Ok, "")
	return toProfileServiceDataResult(updatedProfile), nil
}

func (s *profileService) GetByUUID(ctx context.Context, profileUUID uuid.UUID, userUUID uuid.UUID) (*ProfileServiceDataResult, error) {
	_, span := otel.Tracer("service").Start(ctx, "profile.get")
	defer span.End()
	span.SetAttributes(attribute.String("profile.uuid", profileUUID.String()), attribute.String("user.uuid", userUUID.String()))
	// Find user by UUID to get userID
	user, err := s.userRepo.FindByUUID(userUUID)
	if err != nil || user == nil {
		span.SetStatus(codes.Error, "get profile failed")
		return nil, apperror.NewNotFound("user not found")
	}

	// Get profile by UUID
	profile, err := s.profileRepo.FindByUUID(profileUUID)
	if err != nil || profile == nil {
		span.SetStatus(codes.Error, "get profile failed")
		return nil, apperror.NewNotFound("profile not found")
	}

	// Verify ownership
	if profile.UserID != user.UserID {
		span.SetStatus(codes.Error, "get profile failed")
		return nil, apperror.NewForbidden("profile does not belong to user")
	}

	// Profile.Email column was removed — hydrate the transient field from
	// the owning user so API consumers still see the email.
	hydrateProfileTransients(profile, user, nil)

	span.SetStatus(codes.Ok, "")
	return toProfileServiceDataResult(profile), nil
}

func (s *profileService) GetByUserUUID(ctx context.Context, userUUID uuid.UUID) (*ProfileServiceDataResult, error) {
	_, span := otel.Tracer("service").Start(ctx, "profile.getByUser")
	defer span.End()
	span.SetAttributes(attribute.String("user.uuid", userUUID.String()))

	// Find user by UUID to get userID
	user, err := s.userRepo.FindByUUID(userUUID)
	if err != nil || user == nil {
		span.SetStatus(codes.Error, "get profile failed")
		return nil, apperror.NewNotFound("user not found")
	}

	// Find default profile by user ID
	profile, err := s.profileRepo.FindDefaultByUserID(user.UserID)
	if err != nil || profile == nil {
		span.SetStatus(codes.Error, "get profile failed")
		return nil, apperror.NewNotFound("profile not found")
	}

	// Profile.Email column was removed — hydrate the transient field.
	hydrateProfileTransients(profile, user, nil)

	span.SetStatus(codes.Ok, "")
	return toProfileServiceDataResult(profile), nil
}

func (s *profileService) GetAll(
	ctx context.Context,
	userUUID uuid.UUID,
	firstName, lastName, email, phone, city, country *string,
	isDefault *bool,
	page, limit int,
	sortBy, sortOrder string,
) (*ProfileServiceListResult, error) {
	_, span := otel.Tracer("service").Start(ctx, "profile.list")
	defer span.End()
	span.SetAttributes(attribute.String("user.uuid", userUUID.String()))

	// Find user by UUID to get userID
	user, err := s.userRepo.FindByUUID(userUUID)
	if err != nil || user == nil {
		span.SetStatus(codes.Error, "list profiles failed")
		return nil, apperror.NewNotFound("user not found")
	}

	// Build filter
	filter := ProfileRepositoryGetFilter{
		UserID:    user.UserID,
		FirstName: firstName,
		LastName:  lastName,
		Email:     email,
		Phone:     phone,
		City:      city,
		Country:   country,
		IsDefault: isDefault,
		Page:      page,
		Limit:     limit,
		SortBy:    sortBy,
		SortOrder: sortOrder,
	}

	// Get profiles
	result, err := s.profileRepo.FindAllByUserID(filter)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "list profiles failed")
		return nil, err
	}

	// Convert to service result. All profiles belong to the same user (filter
	// scopes by user_id), so we can hydrate them all with the same User.
	data := make([]ProfileServiceDataResult, len(result.Data))
	for i := range result.Data {
		hydrateProfileTransients(&result.Data[i], user, nil)
		if sr := toProfileServiceDataResult(&result.Data[i]); sr != nil {
			data[i] = *sr
		}
	}

	span.SetStatus(codes.Ok, "")
	return &ProfileServiceListResult{
		Data:       data,
		Total:      result.Total,
		Page:       result.Page,
		Limit:      result.Limit,
		TotalPages: result.TotalPages,
	}, nil
}

func (s *profileService) SetDefaultProfile(ctx context.Context, profileUUID uuid.UUID, userUUID uuid.UUID) (*ProfileServiceDataResult, error) {
	_, span := otel.Tracer("service").Start(ctx, "profile.setDefault")
	defer span.End()
	span.SetAttributes(attribute.String("profile.uuid", profileUUID.String()), attribute.String("user.uuid", userUUID.String()))
	var updatedProfile *Profile

	err := s.db.Transaction(func(tx *gorm.DB) error {
		// Step 1: Create transaction-aware repositories
		txProfileRepo := s.profileRepo.WithTx(tx)
		txUserRepo := s.userRepo.WithTx(tx)

		// Step 2: Find user by UUID to get userID
		user, err := txUserRepo.FindByUUID(userUUID)
		if err != nil || user == nil {
			return apperror.NewNotFound("user not found")
		}

		// Step 3: Get the profile to verify ownership
		profile, err := txProfileRepo.FindByUUID(profileUUID)
		if err != nil {
			return err
		}
		if profile == nil {
			return apperror.NewNotFound("profile not found")
		}

		// Verify profile belongs to user
		if profile.UserID != user.UserID {
			return apperror.NewForbidden("profile does not belong to user")
		}

		// Step 4: Unset all other default profiles for this user
		if err := txProfileRepo.UnsetDefaultProfiles(user.UserID); err != nil {
			return err
		}

		// Step 5: Set this profile as default
		profile.IsDefault = true
		updated, err := txProfileRepo.CreateOrUpdate(profile)
		if err != nil {
			return err
		}

		updatedProfile = updated
		return nil
	})

	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "set default profile failed")
		return nil, err
	}

	span.SetStatus(codes.Ok, "")
	return toProfileServiceDataResult(updatedProfile), nil
}

func (s *profileService) DeleteByUUID(ctx context.Context, profileUUID uuid.UUID, userUUID uuid.UUID) (*ProfileServiceDataResult, error) {
	_, span := otel.Tracer("service").Start(ctx, "profile.delete")
	defer span.End()
	span.SetAttributes(attribute.String("profile.uuid", profileUUID.String()), attribute.String("user.uuid", userUUID.String()))

	// Find user by UUID to get userID
	user, err := s.userRepo.FindByUUID(userUUID)
	if err != nil || user == nil {
		span.SetStatus(codes.Error, "delete profile failed")
		return nil, apperror.NewNotFound("user not found")
	}

	// Get the profile to verify ownership and return it
	profile, err := s.profileRepo.FindByUUID(profileUUID)
	if err != nil || profile == nil {
		span.SetStatus(codes.Error, "delete profile failed")
		return nil, apperror.NewNotFound("profile not found")
	}

	// Verify ownership
	if profile.UserID != user.UserID {
		span.SetStatus(codes.Error, "delete profile failed")
		return nil, apperror.NewForbidden("profile does not belong to user")
	}

	// Prevent deletion of default profile
	if profile.IsDefault {
		span.SetStatus(codes.Error, "delete profile failed")
		return nil, apperror.NewValidation("cannot delete default profile")
	}

	// Delete the profile
	err = s.profileRepo.DeleteByUUID(profileUUID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "delete profile failed")
		return nil, err
	}

	span.SetStatus(codes.Ok, "")
	return toProfileServiceDataResult(profile), nil
}

// Helper functions
func toProfileServiceDataResult(profile *Profile) *ProfileServiceDataResult {
	if profile == nil {
		return nil
	}

	// Convert metadata JSONB to map
	var metadata map[string]any
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
		// Profile Flags
		IsDefault: profile.IsDefault,
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

// Profile holds biographical/PII data for a user. Multiple profiles per user
// are supported (is_default marks the active one). Removed columns:
//   - email     → use users.email
//   - timezone  → use user_settings.timezone
//   - language  → use user_settings.locale
type Profile struct {
	ProfileID   int64     `gorm:"column:profile_id;primaryKey"`
	ProfileUUID uuid.UUID `gorm:"column:profile_uuid;unique;not null"`
	UserID      int64     `gorm:"column:user_id;not null"`

	// Basic Identity Information
	FirstName   string  `gorm:"column:first_name;not null"`
	MiddleName  *string `gorm:"column:middle_name"`
	LastName    *string `gorm:"column:last_name"`
	Suffix      *string `gorm:"column:suffix"`
	DisplayName *string `gorm:"column:display_name"`
	Bio         *string `gorm:"column:bio"`

	// Profile Flags
	IsDefault bool `gorm:"column:is_default;default:false"`

	// Personal Information
	Birthdate *time.Time `gorm:"column:birthdate"`
	Gender    *string    `gorm:"column:gender"`

	// Contact Information (profile-level contact, distinct from users.phone which is the login phone)
	Phone   *string `gorm:"column:phone"`
	Address *string `gorm:"column:address"`

	// Email is NOT persisted on profiles — it lives in users.email.
	// Kept as a transient field for API compatibility.
	Email *string `gorm:"-"`

	// Location Information
	City    *string `gorm:"column:city"`
	Country *string `gorm:"column:country"`

	// Timezone/Language are NOT persisted on profiles — they live in user_settings.
	// Kept as transient fields for API compatibility (Language maps to user_settings.locale).
	Timezone *string `gorm:"-"`
	Language *string `gorm:"-"`

	// Media & Assets (auth-centric)
	ProfileURL *string `gorm:"column:profile_url"`

	// Extended data
	Metadata datatypes.JSON `gorm:"column:metadata;type:jsonb;default:'{}'"`

	// Audit
	CreatedBy *int64 `gorm:"column:created_by"`
	UpdatedBy *int64 `gorm:"column:updated_by"`

	// System Fields
	CreatedAt time.Time      `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt time.Time      `gorm:"column:updated_at;autoUpdateTime"`
	DeletedAt gorm.DeletedAt `gorm:"column:deleted_at;index"`

	// Relationships
	User *User `gorm:"foreignKey:UserID;references:UserID"`
}

func (Profile) TableName() string {
	return "profiles"
}

func (p *Profile) BeforeCreate(tx *gorm.DB) (err error) {
	if p.ProfileUUID == uuid.Nil {
		p.ProfileUUID = uuid.New()
	}
	return
}
