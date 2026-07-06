package user

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/maintainerd/maintainerd-auth/internal/platform/apperror"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"gorm.io/gorm"
)

type UserSettingServiceDataResult struct {
	UserSettingUUID   uuid.UUID
	Timezone          *string
	PreferredLanguage *string
	Locale            *string
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

type UserSettingService interface {
	CreateOrUpdateUserSetting(
		ctx context.Context,
		userUUID uuid.UUID,
		timezone, preferredLanguage, locale *string,
	) (*UserSettingServiceDataResult, error)
	GetByUUID(ctx context.Context, userSettingUUID uuid.UUID, userID int64) (*UserSettingServiceDataResult, error)
	GetByUserUUID(ctx context.Context, userUUID uuid.UUID) (*UserSettingServiceDataResult, error)
	DeleteByUUID(ctx context.Context, userSettingUUID uuid.UUID, userID int64) (*UserSettingServiceDataResult, error)
}

type userSettingService struct {
	db              *gorm.DB
	userSettingRepo UserSettingRepository
	userRepo        UserRepository
}

func NewUserSettingService(
	db *gorm.DB,
	userSettingRepo UserSettingRepository,
	userRepo UserRepository,
) UserSettingService {
	return &userSettingService{
		db:              db,
		userSettingRepo: userSettingRepo,
		userRepo:        userRepo,
	}
}

func (s *userSettingService) CreateOrUpdateUserSetting(
	ctx context.Context,
	userUUID uuid.UUID,
	timezone, preferredLanguage, locale *string,
) (*UserSettingServiceDataResult, error) {
	_, span := otel.Tracer("service").Start(ctx, "user_setting.create_or_update")
	defer span.End()
	span.SetAttributes(attribute.String("user_uuid", userUUID.String()))

	var updatedUserSetting *UserSetting

	err := s.db.Transaction(func(tx *gorm.DB) error {
		// Step 1: Create transaction-aware repositories
		txUserSettingRepo := s.userSettingRepo.WithTx(tx)
		txUserRepo := s.userRepo.WithTx(tx)

		// Step 2: Find user by UUID to get userID
		user, err := txUserRepo.FindByUUID(userUUID)
		if err != nil || user == nil {
			return apperror.NewNotFound("user not found")
		}

		// Step 3: Try to find existing user setting using repository
		existingUserSetting, err := txUserSettingRepo.FindByUserID(user.UserID)
		var userSetting UserSetting

		if err != nil {
			return err
		} else if existingUserSetting == nil {
			// Create new user setting if not found
			userSetting = UserSetting{
				UserSettingUUID: uuid.New(),
				UserID:          user.UserID,
			}
		} else {
			// Use existing user setting
			userSetting = *existingUserSetting
		}

		// Step 2: Set all fields
		// Internationalization. preferred_language was removed from the schema;
		// Locale (BCP-47) is the persisted source of truth. Fall back to the
		// caller-supplied preferredLanguage when locale is missing for
		// backward compatibility.
		userSetting.Timezone = timezone
		if locale != nil {
			userSetting.Locale = locale
		} else if preferredLanguage != nil {
			userSetting.Locale = preferredLanguage
		}

		// Step 4: Create or update using transaction-aware repository
		if userSetting.UserSettingID == 0 {
			// Create new user setting
			createdUserSetting, err := txUserSettingRepo.Create(&userSetting)
			if err != nil {
				return err
			}
			updatedUserSetting = createdUserSetting
		} else {
			// Update existing user setting
			err := txUserSettingRepo.UpdateByUserID(user.UserID, &userSetting)
			if err != nil {
				return err
			}
			updatedUserSetting = &userSetting
		}

		return nil
	})

	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "create or update user setting failed")
		return nil, err
	}

	span.SetStatus(codes.Ok, "")
	return toUserSettingServiceDataResult(updatedUserSetting), nil
}

func (s *userSettingService) GetByUUID(ctx context.Context, userSettingUUID uuid.UUID, userID int64) (*UserSettingServiceDataResult, error) {
	_, span := otel.Tracer("service").Start(ctx, "user_setting.get_by_uuid")
	defer span.End()
	span.SetAttributes(attribute.String("user_setting_uuid", userSettingUUID.String()))

	userSetting, err := s.userSettingRepo.FindByUUID(userSettingUUID)
	if err != nil || userSetting == nil {
		span.SetStatus(codes.Error, "user setting not found")
		return nil, apperror.NewNotFoundWithReason("user setting not found")
	}
	// Ownership guard: the user setting must belong to the requesting user,
	// preventing IDOR if this method is ever routed with a request-supplied UUID.
	if userSetting.UserID != userID {
		return nil, apperror.NewNotFoundWithReason("user setting not found or access denied")
	}

	span.SetStatus(codes.Ok, "")
	return toUserSettingServiceDataResult(userSetting), nil
}

func (s *userSettingService) GetByUserUUID(ctx context.Context, userUUID uuid.UUID) (*UserSettingServiceDataResult, error) {
	_, span := otel.Tracer("service").Start(ctx, "user_setting.get_by_user_uuid")
	defer span.End()
	span.SetAttributes(attribute.String("user_uuid", userUUID.String()))

	// Find user by UUID to get userID
	user, err := s.userRepo.FindByUUID(userUUID)
	if err != nil || user == nil {
		span.SetStatus(codes.Error, "user not found")
		return nil, apperror.NewNotFound("user not found")
	}

	userSetting, err := s.userSettingRepo.FindByUserID(user.UserID)
	if err != nil || userSetting == nil {
		span.SetStatus(codes.Error, "user setting not found")
		return nil, apperror.NewNotFoundWithReason("user setting not found")
	}

	span.SetStatus(codes.Ok, "")
	return toUserSettingServiceDataResult(userSetting), nil
}

func (s *userSettingService) DeleteByUUID(ctx context.Context, userSettingUUID uuid.UUID, userID int64) (*UserSettingServiceDataResult, error) {
	_, span := otel.Tracer("service").Start(ctx, "user_setting.delete_by_uuid")
	defer span.End()
	span.SetAttributes(attribute.String("user_setting_uuid", userSettingUUID.String()))

	// First get the user setting to return it
	userSetting, err := s.userSettingRepo.FindByUUID(userSettingUUID)
	if err != nil || userSetting == nil {
		span.SetStatus(codes.Error, "user setting not found")
		return nil, apperror.NewNotFoundWithReason("user setting not found")
	}
	// Ownership guard: the setting must belong to the requesting user, preventing
	// IDOR if this method is ever routed with a request-supplied UUID.
	if userSetting.UserID != userID {
		return nil, apperror.NewNotFoundWithReason("user setting not found or access denied")
	}

	// Delete the user setting
	err = s.userSettingRepo.DeleteByUUID(userSettingUUID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "delete user setting failed")
		return nil, err
	}

	span.SetStatus(codes.Ok, "")
	return toUserSettingServiceDataResult(userSetting), nil
}

// Helper functions
func toUserSettingServiceDataResult(userSetting *UserSetting) *UserSettingServiceDataResult {
	if userSetting == nil {
		return nil
	}

	// preferred_language column was removed — mirror Locale into the transient
	// PreferredLanguage field so API consumers still see a value.
	hydrateUserSettingTransients(userSetting)

	result := &UserSettingServiceDataResult{
		UserSettingUUID:   userSetting.UserSettingUUID,
		Timezone:          userSetting.Timezone,
		PreferredLanguage: userSetting.PreferredLanguage,
		Locale:            userSetting.Locale,
		CreatedAt:         userSetting.CreatedAt,
		UpdatedAt:         userSetting.UpdatedAt,
	}

	return result
}
