package dto

import (
	"encoding/json"
	"time"

	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/go-ozzo/ozzo-validation/v4/is"

	"github.com/maintainerd/auth/internal/model"
)

type UserSettingRequest struct {
	// Internationalization
	Timezone          *string `json:"timezone,omitempty"`
	PreferredLanguage *string `json:"preferred_language,omitempty"`
	Locale            *string `json:"locale,omitempty"`

	// Social Media & External Links
	SocialLinks map[string]string `json:"social_links,omitempty"`

	// Communication Preferences
	PreferredContactMethod   *string `json:"preferred_contact_method,omitempty"`
	MarketingEmailConsent    *bool   `json:"marketing_email_consent,omitempty"`
	SMSNotificationsConsent  *bool   `json:"sms_notifications_consent,omitempty"`
	PushNotificationsConsent *bool   `json:"push_notifications_consent,omitempty"`

	// Privacy & Compliance
	ProfileVisibility     *string `json:"profile_visibility,omitempty"`
	DataProcessingConsent *bool   `json:"data_processing_consent,omitempty"`

	// Emergency Contact
	EmergencyContactName     *string `json:"emergency_contact_name,omitempty"`
	EmergencyContactPhone    *string `json:"emergency_contact_phone,omitempty"`
	EmergencyContactEmail    *string `json:"emergency_contact_email,omitempty"`
	EmergencyContactRelation *string `json:"emergency_contact_relation,omitempty"`
}

func (r UserSettingRequest) Validate() error {
	return validation.ValidateStruct(&r,
		// Internationalization
		validation.Field(&r.Timezone,
			validation.NilOrNotEmpty,
			validation.RuneLength(0, 50).Error("Timezone must be at most 50 characters"),
		),
		validation.Field(&r.PreferredLanguage,
			validation.NilOrNotEmpty,
			validation.RuneLength(2, 10).Error("Preferred language must be 2-10 characters"),
		),
		validation.Field(&r.Locale,
			validation.NilOrNotEmpty,
			validation.RuneLength(2, 10).Error("Locale must be 2-10 characters"),
		),

		// Communication Preferences
		validation.Field(&r.PreferredContactMethod,
			validation.NilOrNotEmpty,
			validation.In("email", "phone", "sms").Error("Preferred contact method must be email, phone, or sms"),
		),

		// Privacy & Compliance
		validation.Field(&r.ProfileVisibility,
			validation.NilOrNotEmpty,
			validation.In("public", "private", "friends").Error("Profile visibility must be public, private, or friends"),
		),

		// Emergency Contact
		validation.Field(&r.EmergencyContactName,
			validation.NilOrNotEmpty,
			validation.RuneLength(0, 200).Error("Emergency contact name must be at most 200 characters"),
		),
		validation.Field(&r.EmergencyContactPhone,
			validation.NilOrNotEmpty,
			validation.RuneLength(0, 20).Error("Emergency contact phone must be at most 20 characters"),
		),
		validation.Field(&r.EmergencyContactEmail,
			validation.NilOrNotEmpty,
			is.Email.Error("Invalid emergency contact email format"),
			validation.RuneLength(0, 255).Error("Emergency contact email must be at most 255 characters"),
		),
		validation.Field(&r.EmergencyContactRelation,
			validation.NilOrNotEmpty,
			validation.RuneLength(0, 50).Error("Emergency contact relation must be at most 50 characters"),
		),
	)
}

type UserSettingResponse struct {
	UserSettingUUID string `json:"user_setting_uuid"`

	// Internationalization
	Timezone          *string `json:"timezone,omitempty"`
	PreferredLanguage *string `json:"preferred_language,omitempty"`
	Locale            *string `json:"locale,omitempty"`

	// Social Media & External Links
	SocialLinks map[string]any `json:"social_links,omitempty"`

	// Communication Preferences
	PreferredContactMethod   *string `json:"preferred_contact_method,omitempty"`
	MarketingEmailConsent    bool    `json:"marketing_email_consent"`
	SMSNotificationsConsent  bool    `json:"sms_notifications_consent"`
	PushNotificationsConsent bool    `json:"push_notifications_consent"`

	// Privacy & Compliance
	ProfileVisibility       *string    `json:"profile_visibility,omitempty"`
	DataProcessingConsent   bool       `json:"data_processing_consent"`
	TermsAcceptedAt         *time.Time `json:"terms_accepted_at,omitempty"`
	PrivacyPolicyAcceptedAt *time.Time `json:"privacy_policy_accepted_at,omitempty"`

	// Emergency Contact
	EmergencyContactName     *string `json:"emergency_contact_name,omitempty"`
	EmergencyContactPhone    *string `json:"emergency_contact_phone,omitempty"`
	EmergencyContactEmail    *string `json:"emergency_contact_email,omitempty"`
	EmergencyContactRelation *string `json:"emergency_contact_relation,omitempty"`

	// System Fields
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func NewUserSettingResponse(us *model.UserSetting) *UserSettingResponse {
	// Convert GORM JSON to map for social links
	var socialLinks map[string]any
	if len(us.SocialLinks) > 0 {
		socialLinks = make(map[string]any)
		if err := json.Unmarshal(us.SocialLinks, &socialLinks); err == nil {
			// Only include if unmarshaling was successful
		} else {
			socialLinks = nil
		}
	}

	return &UserSettingResponse{
		UserSettingUUID: us.UserSettingUUID.String(),

		// Internationalization
		Timezone:          us.Timezone,
		PreferredLanguage: us.PreferredLanguage,
		Locale:            us.Locale,

		// Social Media & External Links
		SocialLinks: socialLinks,

		// Communication Preferences
		PreferredContactMethod:   us.PreferredContactMethod,
		MarketingEmailConsent:    us.MarketingEmailConsent,
		SMSNotificationsConsent:  us.SMSNotificationsConsent,
		PushNotificationsConsent: us.PushNotificationsConsent,

		// Privacy & Compliance
		ProfileVisibility:       us.ProfileVisibility,
		DataProcessingConsent:   us.DataProcessingConsent,
		TermsAcceptedAt:         us.TermsAcceptedAt,
		PrivacyPolicyAcceptedAt: us.PrivacyPolicyAcceptedAt,

		// Emergency Contact
		EmergencyContactName:     us.EmergencyContactName,
		EmergencyContactPhone:    us.EmergencyContactPhone,
		EmergencyContactEmail:    us.EmergencyContactEmail,
		EmergencyContactRelation: us.EmergencyContactRelation,

		// System Fields
		CreatedAt: us.CreatedAt,
		UpdatedAt: us.UpdatedAt,
	}
}
