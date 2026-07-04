package authn

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type UserLockout struct {
	UserLockoutID   int64      `gorm:"column:user_lockout_id;primaryKey;autoIncrement"`
	UserLockoutUUID uuid.UUID  `gorm:"column:user_lockout_uuid;type:uuid;uniqueIndex;not null;default:gen_random_uuid()"`
	TenantID        int64      `gorm:"column:tenant_id;not null"`
	UserID          *int64     `gorm:"column:user_id"`
	Identifier      string     `gorm:"column:identifier;not null"`
	FailedCount     int        `gorm:"column:failed_count;not null;default:0"`
	LastFailedAt    *time.Time `gorm:"column:last_failed_at"`
	LockedUntil     *time.Time `gorm:"column:locked_until"`
	IPAddress       *string    `gorm:"column:ip_address"`
	CreatedAt       time.Time  `gorm:"column:created_at;autoCreateTime;not null"`
	UpdatedAt       time.Time  `gorm:"column:updated_at;autoUpdateTime;not null"`
}

func (UserLockout) TableName() string { return "user_lockouts" }

func (l *UserLockout) BeforeCreate(_ *gorm.DB) error {
	if l.UserLockoutUUID == uuid.Nil {
		l.UserLockoutUUID = uuid.New()
	}
	return nil
}
