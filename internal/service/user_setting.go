package service

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/apperror"
	"github.com/maintainerd/auth/internal/model"
	"github.com/maintainerd/auth/internal/repository"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type UserSettingServiceDataResult struct {
	UserSettingUUID          uuid.UUID
	Timezone                 *string
	PreferredLanguage        *string
	Locale                   *string
	SocialLinks              datatypes.JSON
	PreferredContactMethod   *string
	MarketingEmailConsent    bool
	SMSNotificationsConsent  bool
	PushNotificationsConsent bool
	ProfileVisibility        *string
	DataProcessingConsent    bool
	TermsAcceptedAt          *time.Time
	PrivacyPolicyAcceptedAt  *time.Time
	EmergencyContactName     *string
	EmergencyContactPhone    *string
	EmergencyContactEmail    *string
	EmergencyContactRelation *string
	CreatedAt                time.Time
	UpdatedAt                time.Time
}

type UserSettingService interface {
	CreateOrUpdateUserSetting(
		ctx context.Context,
		userUUID uuid.UUID,
		timezone, preferredLanguage, locale *string,
		socialLinks map[string]any,
		preferredContactMethod *string,
		marketingEmailConsent, smsNotificationsConsent, pushNotificationsConsent *bool,
		profileVisibility *string,
		dataProcessingConsent *bool,
		termsAcceptedAt, privacyPolicyAcceptedAt *time.Time,
		emergencyContactName, emergencyContactPhone, emergencyContactEmail, emergencyContactRelation *string,
	) (*UserSettingServiceDataResult, error)
	GetByUUID(ctx context.Context, userSettingUUID uuid.UUID) (*UserSettingServiceDataResult, error)
	GetByUserUUID(ctx context.Context, userUUID uuid.UUID) (*UserSettingServiceDataResult, error)
	DeleteByUUID(ctx context.Context, userSettingUUID uuid.UUID) (*UserSettingServiceDataResult, error)
}

type userSettingService struct {
	db              *gorm.DB
	userSettingRepo repository.UserSettingRepository
	userRepo        repository.UserRepository
}

func NewUserSettingService(
	db *gorm.DB,
	userSettingRepo repository.UserSettingRepository,
	userRepo repository.UserRepository,
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
	socialLinks map[string]any,
	preferredContactMethod *string,
	marketingEmailConsent, smsNotificationsConsent, pushNotificationsConsent *bool,
	profileVisibility *string,
	dataProcessingConsent *bool,
	termsAcceptedAt, privacyPolicyAcceptedAt *time.Time,
	emergencyContactName, emergencyContactPhone, emergencyContactEmail, emergencyContactRelation *string,
) (*UserSettingServiceDataResult, error) {
	_, span := otel.Tracer("service").Start(ctx, "user_setting.create_or_update")
	defer span.End()
	span.SetAttributes(attribute.String("user_uuid", userUUID.String()))

	var updatedUserSetting *model.UserSetting

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
		var userSetting model.UserSetting

		if err != nil {
			return err
		} else if existingUserSetting == nil {
			// Create new user setting if not found
			userSetting = model.UserSetting{
				UserSettingUUID: uuid.New(),
				UserID:          user.UserID,
			}
		} else {
			// Use existing user setting
			userSetting = *existingUserSetting
		}

		// Step 2: Parse social links JSON
		var socialLinksJSON datatypes.JSON
		if len(socialLinks) > 0 {
			socialLinksBytes, err := json.Marshal(socialLinks)
			if err != nil {
				return apperror.NewValidation("invalid social links format")
			}
			socialLinksJSON = socialLinksBytes
		}

		// Step 3: Set all fields
		// Internationalization
		userSetting.Timezone = timezone
		userSetting.PreferredLanguage = preferredLanguage
		userSetting.Locale = locale

		// Social Media & External Links
		if socialLinksJSON != nil {
			userSetting.SocialLinks = socialLinksJSON
		}

		// Communication Preferences
		userSetting.PreferredContactMethod = preferredContactMethod
		if marketingEmailConsent != nil {
			userSetting.MarketingEmailConsent = *marketingEmailConsent
		}
		if smsNotificationsConsent != nil {
			userSetting.SMSNotificationsConsent = *smsNotificationsConsent
		}
		if pushNotificationsConsent != nil {
			userSetting.PushNotificationsConsent = *pushNotificationsConsent
		}

		// Privacy & Compliance
		userSetting.ProfileVisibility = profileVisibility
		if dataProcessingConsent != nil {
			userSetting.DataProcessingConsent = *dataProcessingConsent
		}
		userSetting.TermsAcceptedAt = termsAcceptedAt
		userSetting.PrivacyPolicyAcceptedAt = privacyPolicyAcceptedAt

		// Emergency Contact
		userSetting.EmergencyContactName = emergencyContactName
		userSetting.EmergencyContactPhone = emergencyContactPhone
		userSetting.EmergencyContactEmail = emergencyContactEmail
		userSetting.EmergencyContactRelation = emergencyContactRelation

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

func (s *userSettingService) GetByUUID(ctx context.Context, userSettingUUID uuid.UUID) (*UserSettingServiceDataResult, error) {
	_, span := otel.Tracer("service").Start(ctx, "user_setting.get_by_uuid")
	defer span.End()
	span.SetAttributes(attribute.String("user_setting_uuid", userSettingUUID.String()))

	userSetting, err := s.userSettingRepo.FindByUUID(userSettingUUID)
	if err != nil || userSetting == nil {
		span.SetStatus(codes.Error, "user setting not found")
		return nil, apperror.NewNotFoundWithReason("user setting not found")
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

func (s *userSettingService) DeleteByUUID(ctx context.Context, userSettingUUID uuid.UUID) (*UserSettingServiceDataResult, error) {
	_, span := otel.Tracer("service").Start(ctx, "user_setting.delete_by_uuid")
	defer span.End()
	span.SetAttributes(attribute.String("user_setting_uuid", userSettingUUID.String()))

	// First get the user setting to return it
	userSetting, err := s.userSettingRepo.FindByUUID(userSettingUUID)
	if err != nil || userSetting == nil {
		span.SetStatus(codes.Error, "user setting not found")
		return nil, apperror.NewNotFoundWithReason("user setting not found")
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
func toUserSettingServiceDataResult(userSetting *model.UserSetting) *UserSettingServiceDataResult {
	if userSetting == nil {
		return nil
	}

	result := &UserSettingServiceDataResult{
		UserSettingUUID:          userSetting.UserSettingUUID,
		Timezone:                 userSetting.Timezone,
		PreferredLanguage:        userSetting.PreferredLanguage,
		Locale:                   userSetting.Locale,
		SocialLinks:              userSetting.SocialLinks,
		PreferredContactMethod:   userSetting.PreferredContactMethod,
		MarketingEmailConsent:    userSetting.MarketingEmailConsent,
		SMSNotificationsConsent:  userSetting.SMSNotificationsConsent,
		PushNotificationsConsent: userSetting.PushNotificationsConsent,
		ProfileVisibility:        userSetting.ProfileVisibility,
		DataProcessingConsent:    userSetting.DataProcessingConsent,
		TermsAcceptedAt:          userSetting.TermsAcceptedAt,
		PrivacyPolicyAcceptedAt:  userSetting.PrivacyPolicyAcceptedAt,
		EmergencyContactName:     userSetting.EmergencyContactName,
		EmergencyContactPhone:    userSetting.EmergencyContactPhone,
		EmergencyContactEmail:    userSetting.EmergencyContactEmail,
		EmergencyContactRelation: userSetting.EmergencyContactRelation,
		CreatedAt:                userSetting.CreatedAt,
		UpdatedAt:                userSetting.UpdatedAt,
	}

	return result
}
