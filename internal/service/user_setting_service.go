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
	GetByUUID(userSettingUUID uuid.UUID) (*UserSettingServiceDataResult, error)
	GetByUserUUID(userUUID uuid.UUID) (*UserSettingServiceDataResult, error)
	DeleteByUUID(userSettingUUID uuid.UUID) (*UserSettingServiceDataResult, error)
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
	var updatedUserSetting *model.UserSetting

	err := s.db.Transaction(func(tx *gorm.DB) error {
		// Step 1: Create transaction-aware repositories
		txUserSettingRepo := s.userSettingRepo.WithTx(tx)
		txUserRepo := s.userRepo.WithTx(tx)

		// Step 2: Find user by UUID to get userID
		user, err := txUserRepo.FindByUUID(userUUID)
		if err != nil || user == nil {
			return errors.New("user not found")
		}

		// Step 3: Try to find existing user setting using repository
		existingUserSetting, err := txUserSettingRepo.FindByUserID(user.UserID)
		var userSetting model.UserSetting

		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				// Create new user setting if not found
				userSetting = model.UserSetting{
					UserSettingUUID: uuid.New(),
					UserID:          user.UserID,
				}
			} else {
				return err
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
				return errors.New("invalid social links format")
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
		return nil, err
	}

	return toUserSettingServiceDataResult(updatedUserSetting), nil
}

func (s *userSettingService) GetByUUID(userSettingUUID uuid.UUID) (*UserSettingServiceDataResult, error) {
	userSetting, err := s.userSettingRepo.FindByUUID(userSettingUUID)
	if err != nil || userSetting == nil {
		return nil, errors.New("user setting not found")
	}

	return toUserSettingServiceDataResult(userSetting), nil
}

func (s *userSettingService) GetByUserUUID(userUUID uuid.UUID) (*UserSettingServiceDataResult, error) {
	// Find user by UUID to get userID
	user, err := s.userRepo.FindByUUID(userUUID)
	if err != nil || user == nil {
		return nil, errors.New("user not found")
	}

	userSetting, err := s.userSettingRepo.FindByUserID(user.UserID)
	if err != nil || userSetting == nil {
		return nil, errors.New("user setting not found")
	}

	return toUserSettingServiceDataResult(userSetting), nil
}

func (s *userSettingService) DeleteByUUID(userSettingUUID uuid.UUID) (*UserSettingServiceDataResult, error) {
	// First get the user setting to return it
	userSetting, err := s.userSettingRepo.FindByUUID(userSettingUUID)
	if err != nil || userSetting == nil {
		return nil, errors.New("user setting not found")
	}

	// Delete the user setting
	err = s.userSettingRepo.DeleteByUUID(userSettingUUID)
	if err != nil {
		return nil, err
	}

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
