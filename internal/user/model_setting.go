package user

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type UserSetting struct {
	UserSettingID            int64          `gorm:"column:user_setting_id;primaryKey"`
	UserSettingUUID          uuid.UUID      `gorm:"column:user_setting_uuid;unique;not null"`
	UserID                   int64          `gorm:"column:user_id;not null;unique"`
	Timezone                 *string        `gorm:"column:timezone"`
	Locale                   *string        `gorm:"column:locale"`
	PreferredLanguage        *string        `gorm:"-"`
	SocialLinks              datatypes.JSON `gorm:"column:social_links"`
	PreferredContactMethod   *string        `gorm:"column:preferred_contact_method"`
	MarketingEmailConsent    bool           `gorm:"column:marketing_email_consent;default:false"`
	SMSNotificationsConsent  bool           `gorm:"column:sms_notifications_consent;default:false"`
	PushNotificationsConsent bool           `gorm:"column:push_notifications_consent;default:false"`
	ProfileVisibility        *string        `gorm:"column:profile_visibility;default:'private'"`
	DataProcessingConsent    bool           `gorm:"column:data_processing_consent;default:false"`
	TermsAcceptedAt          *time.Time     `gorm:"column:terms_accepted_at"`
	PrivacyPolicyAcceptedAt  *time.Time     `gorm:"column:privacy_policy_accepted_at"`
	EmergencyContactName     *string        `gorm:"column:emergency_contact_name"`
	EmergencyContactPhone    *string        `gorm:"column:emergency_contact_phone"`
	EmergencyContactEmail    *string        `gorm:"column:emergency_contact_email"`
	EmergencyContactRelation *string        `gorm:"column:emergency_contact_relation"`
	CreatedAt                time.Time      `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt                time.Time      `gorm:"column:updated_at;autoUpdateTime"`

	User *User `gorm:"foreignKey:UserID;references:UserID"`
}

func (UserSetting) TableName() string {
	return "user_settings"
}

func (us *UserSetting) BeforeCreate(tx *gorm.DB) (err error) {
	if us.UserSettingUUID == uuid.Nil {
		us.UserSettingUUID = uuid.New()
	}
	return
}
