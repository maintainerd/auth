package user

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// UserPasswordHistory records previously used password hashes for a user so
// that re-use can be blocked according to the tenant's PasswordPolicy.HistoryCount.
// This table is append-only; the DB-level trigger trg_deny_uph_update enforces
// that no row can be updated after insertion.
type UserPasswordHistory struct {
	HistoryID    int64     `gorm:"column:history_id;primaryKey;autoIncrement"`
	HistoryUUID  uuid.UUID `gorm:"column:history_uuid;not null;unique"`
	UserID       int64     `gorm:"column:user_id;not null"`
	PasswordHash string    `gorm:"column:password_hash;not null"`
	CreatedAt    time.Time `gorm:"column:created_at;not null;autoCreateTime"`
}

func (UserPasswordHistory) TableName() string {
	return "user_password_history"
}

func (h *UserPasswordHistory) BeforeCreate(_ *gorm.DB) error {
	if h.HistoryUUID == uuid.Nil {
		h.HistoryUUID = uuid.New()
	}
	return nil
}
