package user

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type UserSetting struct {
	UserSettingID            int64          `gorm:"column:user_setting_id;primaryKey"`
	UserSettingUUID          uuid.UUID      `gorm:"column:user_setting_uuid;unique;not null"`
	UserID                   int64          `gorm:"column:user_id;not null;unique"`
	Timezone          *string   `gorm:"column:timezone"`
	Locale            *string   `gorm:"column:locale"`
	PreferredLanguage *string   `gorm:"-"`
	CreatedAt         time.Time `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt         time.Time `gorm:"column:updated_at;autoUpdateTime"`

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
