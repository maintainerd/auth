package user

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// Profile holds biographical/PII data for a user.
type Profile struct {
	ProfileID   int64          `gorm:"column:profile_id;primaryKey"`
	ProfileUUID uuid.UUID      `gorm:"column:profile_uuid;unique;not null"`
	UserID      int64          `gorm:"column:user_id;not null"`
	FirstName   string         `gorm:"column:first_name;not null"`
	MiddleName  *string        `gorm:"column:middle_name"`
	LastName    *string        `gorm:"column:last_name"`
	DisplayName *string        `gorm:"column:display_name"`
	Birthdate   *time.Time     `gorm:"column:birthdate"`
	Gender      *string        `gorm:"column:gender"`
	IsDefault   bool           `gorm:"column:is_default;not null;default:false"`
	Email       *string        `gorm:"-"`
	Timezone    *string        `gorm:"-"`
	Language    *string        `gorm:"-"`
	ProfileURL  *string        `gorm:"column:profile_url"`
	Metadata    datatypes.JSON `gorm:"column:metadata;type:jsonb;default:'{}'"`
	CreatedBy   *int64         `gorm:"column:created_by"`
	UpdatedBy   *int64         `gorm:"column:updated_by"`
	CreatedAt   time.Time      `gorm:"column:created_at;not null;autoCreateTime"`
	UpdatedAt   time.Time      `gorm:"column:updated_at;not null;autoUpdateTime"`
	DeletedAt   gorm.DeletedAt `gorm:"column:deleted_at;index"`

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
