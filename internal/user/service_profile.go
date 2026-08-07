package user

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/maintainerd/maintainerd-auth/internal/platform/apperror"
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
	DisplayName *string
	// Personal Information
	Birthdate *time.Time
	Gender    *string
	// Contact Information (transient)
	Email *string
	// Preference
	Timezone *string
	Language *string
	// Media & Assets (auth-centric)
	ProfileURL *string
	// Extended data
	Metadata map[string]any
	// Profile state
	IsDefault bool
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
	// Avatar handling. Set/Clear take the caller's user id and refuse a profile
	// belonging to anyone else, so a UUID alone is not authority to change
	// someone's picture. SetProfilePicture returns the URL the profile now
	// points at.
	SetProfilePicture(ctx context.Context, profileUUID uuid.UUID, userID int64, picture *ProfilePicture) (string, error)
	ClearProfilePicture(ctx context.Context, profileUUID uuid.UUID, userID int64) error
	GetProfilePicture(ctx context.Context, profileUUID uuid.UUID) (*ProfilePicture, error)
	GetProfilePictureETag(ctx context.Context, profileUUID uuid.UUID) (string, error)
	// EnsureProfileOwnedBy gates the avatar READ for callers who do not
	// administer users.
	EnsureProfileOwnedBy(ctx context.Context, profileUUID uuid.UUID, userID int64) error
	CreateOrUpdateProfile(
		ctx context.Context,
		userUUID uuid.UUID,
		firstName string,
		middleName, lastName, displayName *string,
		birthdate *time.Time,
		gender *string,
		email *string,
		timezone, language *string,
		profileURL *string,
		metadata map[string]any,
	) (*ProfileServiceDataResult, error)
	CreateOrUpdateSpecificProfile(
		ctx context.Context,
		profileUUID uuid.UUID,
		userUUID uuid.UUID,
		firstName string,
		middleName, lastName, displayName *string,
		birthdate *time.Time,
		gender *string,
		email *string,
		timezone, language *string,
		profileURL *string,
		metadata map[string]any,
	) (*ProfileServiceDataResult, error)
	GetByUUID(ctx context.Context, profileUUID uuid.UUID, userUUID uuid.UUID) (*ProfileServiceDataResult, error)
	GetByUserUUID(ctx context.Context, userUUID uuid.UUID) (*ProfileServiceDataResult, error)
	GetAll(ctx context.Context, userUUID uuid.UUID, firstName, lastName, email *string, page, limit int, sortBy, sortOrder string) (*ProfileServiceListResult, error)
	SetDefault(ctx context.Context, profileUUID uuid.UUID, userUUID uuid.UUID) (*ProfileServiceDataResult, error)
	DeleteByUUID(ctx context.Context, profileUUID uuid.UUID, userUUID uuid.UUID) (*ProfileServiceDataResult, error)
}

type profileService struct {
	db          *gorm.DB
	profileRepo ProfileRepository
	userRepo    UserRepository
	// profilePictureRepo is the ONLY route to avatar bytes. Optional so a caller
	// that does not serve avatars need not wire it; the picture methods refuse
	// rather than panic when it is absent.
	profilePictureRepo ProfilePictureRepository
}

func NewProfileService(
	db *gorm.DB,
	profileRepo ProfileRepository,
	userRepo UserRepository,
	opts ...ProfileServiceOption,
) ProfileService {
	s := &profileService{
		db:          db,
		profileRepo: profileRepo,
		userRepo:    userRepo,
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

// ProfileServiceOption configures optional collaborators.
type ProfileServiceOption func(*profileService)

// WithProfilePictureRepository enables avatar upload and serving.
func WithProfilePictureRepository(repo ProfilePictureRepository) ProfileServiceOption {
	return func(s *profileService) { s.profilePictureRepo = repo }
}

// applyProfileFields copies the optional fields onto profile, skipping the ones
// the caller omitted.
//
// The assignments used to be unconditional. Every optional field arrives as a
// pointer that JSON decodes to nil when the key is absent, and the row is then
// persisted with a full-struct Save — so a client that PUT only {first_name,
// last_name} silently erased the user's gender, middle name, birthdate and the
// whole metadata blob (which carries the OIDC `address` claim). A partial update
// must not be a destructive one. nil therefore means "not sent, leave alone";
// clearing a field is not expressible through these DTOs, which mark every
// optional field `omitempty` and so cannot distinguish an explicit null anyway.
//
// firstName is a plain string and Required at the DTO layer, so it is always
// present and always assigned.
func applyProfileFields(
	profile *Profile,
	firstName string,
	middleName, lastName, displayName *string,
	birthdate *time.Time,
	gender *string,
	timezone, language *string,
	profileURL *string,
) {
	profile.FirstName = firstName
	if middleName != nil {
		profile.MiddleName = middleName
	}
	if lastName != nil {
		profile.LastName = lastName
	}
	if displayName != nil {
		profile.DisplayName = displayName
	}
	if birthdate != nil {
		profile.Birthdate = birthdate
	}
	if gender != nil {
		profile.Gender = gender
	}
	if timezone != nil {
		profile.Timezone = timezone
	}
	if language != nil {
		profile.Language = language
	}
	if profileURL != nil {
		profile.ProfileURL = profileURL
	}
}

// applyProfileMetadata encodes metadata onto profile. An omitted metadata map
// leaves any stored blob intact; only a brand-new row is seeded with "{}" so the
// JSONB column is never NULL.
func applyProfileMetadata(profile *Profile, metadata map[string]any) error {
	if metadata != nil {
		metadataBytes, err := json.Marshal(metadata)
		if err != nil {
			return err
		}
		profile.Metadata = metadataBytes
		return nil
	}
	if len(profile.Metadata) == 0 {
		profile.Metadata = datatypes.JSON([]byte("{}"))
	}
	return nil
}

func (s *profileService) CreateOrUpdateProfile(
	ctx context.Context,
	userUUID uuid.UUID,
	firstName string,
	middleName, lastName, displayName *string,
	birthdate *time.Time,
	gender *string,
	email *string,
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

		// Step 3: Try to find existing default profile for this user
		existingProfile, err := txProfileRepo.FindDefaultByUserID(user.UserID)
		var profile Profile

		if err != nil {
			return err
		} else if existingProfile == nil {
			// No profile yet — create one.
			profile = Profile{
				ProfileUUID: uuid.New(),
				UserID:      user.UserID,
				IsDefault:   true,
			}
		} else {
			// Use existing profile
			profile = *existingProfile
		}

		// Step 4: Apply the fields the caller actually sent.
		applyProfileFields(&profile, firstName, middleName, lastName, displayName, birthdate, gender, timezone, language, profileURL)
		if err := applyProfileMetadata(&profile, metadata); err != nil {
			return err
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
	middleName, lastName, displayName *string,
	birthdate *time.Time,
	gender *string,
	email *string,
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
			// Check if user already has a profile — if not, this is the first.
			anyProfile, err := txProfileRepo.FindByUserID(user.UserID)
			if err != nil {
				return err
			}
			profile = Profile{
				ProfileUUID: profileUUID,
				UserID:      user.UserID,
				IsDefault:   anyProfile == nil,
			}
		} else {
			// Verify profile belongs to user
			if existingProfile.UserID != user.UserID {
				return apperror.NewForbidden("profile does not belong to user")
			}
			// Use existing profile
			profile = *existingProfile
		}

		// Apply the fields the caller actually sent.
		applyProfileFields(&profile, firstName, middleName, lastName, displayName, birthdate, gender, timezone, language, profileURL)
		// Contact Information (transient — not persisted on profile)
		if email != nil {
			profile.Email = email
		}
		if err := applyProfileMetadata(&profile, metadata); err != nil {
			return err
		}

		// Create or update profile
		if profile.ProfileID == 0 {
			// Create new profile
			createdProfile, err := txProfileRepo.Create(&profile)
			if err != nil {
				return err
			}
			updatedProfile = createdProfile
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
	firstName, lastName, email *string,
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

	// Delete the profile. If it was the default, promote another of the user's
	// remaining profiles so the "exactly one default" invariant holds: the row is
	// soft-deleted, so it drops out of the live-row unique index and the promoted
	// profile can safely take the default slot. All in one transaction.
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		txProfileRepo := s.profileRepo.WithTx(tx)
		if err := txProfileRepo.DeleteByUUID(profileUUID); err != nil {
			return err
		}
		if profile.IsDefault {
			// Oldest remaining live profile (soft-deleted rows are excluded).
			next, err := txProfileRepo.FindByUserID(user.UserID)
			if err != nil {
				return err
			}
			if next != nil {
				// Clear stray defaults first (idempotent), then promote — keeps the
				// promotion safe against the partial unique index.
				if err := txProfileRepo.UnsetDefaultProfiles(user.UserID); err != nil {
					return err
				}
				if err := tx.Model(&Profile{}).
					Where("profile_id = ?", next.ProfileID).
					Update("is_default", true).Error; err != nil {
					return err
				}
			}
		}
		return nil
	})
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
		DisplayName: profile.DisplayName,
		// Personal Information
		Birthdate: profile.Birthdate,
		Gender:    profile.Gender,
		// Contact Information (transient)
		Email: profile.Email,
		// Preference
		Timezone: profile.Timezone,
		Language: profile.Language,
		// Media & Assets (auth-centric)
		ProfileURL: profile.ProfileURL,
		// Extended data
		Metadata: metadata,
		// Profile state
		IsDefault: profile.IsDefault,
		// System Fields
		CreatedAt: profile.CreatedAt,
		UpdatedAt: profile.UpdatedAt,
	}

	return result
}

func (s *profileService) SetDefault(ctx context.Context, profileUUID uuid.UUID, userUUID uuid.UUID) (*ProfileServiceDataResult, error) {
	_, span := otel.Tracer("service").Start(ctx, "profile.set_default")
	defer span.End()
	span.SetAttributes(attribute.String("profile.uuid", profileUUID.String()), attribute.String("user.uuid", userUUID.String()))

	user, err := s.userRepo.FindByUUID(userUUID)
	if err != nil || user == nil {
		return nil, apperror.NewNotFound("user not found")
	}

	profile, err := s.profileRepo.FindByUUID(profileUUID)
	if err != nil || profile == nil {
		return nil, apperror.NewNotFound("profile not found")
	}
	if profile.UserID != user.UserID {
		return nil, apperror.NewForbidden("profile does not belong to user")
	}

	profileID := profile.ProfileID
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := s.profileRepo.WithTx(tx).UnsetDefaultProfiles(user.UserID); err != nil {
			return err
		}
		return tx.Model(&Profile{}).Where("profile_id = ?", profileID).Update("is_default", true).Error
	})
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "set default profile failed")
		return nil, err
	}

	updated, err := s.profileRepo.FindByUUID(profileUUID)
	if err != nil || updated == nil {
		return nil, apperror.NewNotFound("profile not found after update")
	}
	hydrateProfileTransients(updated, user, nil)
	span.SetStatus(codes.Ok, "")
	return toProfileServiceDataResult(updated), nil
}
