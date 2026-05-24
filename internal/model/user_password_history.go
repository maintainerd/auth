package model

import "time"

// UserPasswordHistory records previously used password hashes for a user so
// that re-use can be blocked according to the tenant's PasswordPolicy.HistoryCount.
type UserPasswordHistory struct {
	HistoryID    int64     `gorm:"column:history_id;primaryKey;autoIncrement"`
	UserID       int64     `gorm:"column:user_id;not null"`
	PasswordHash string    `gorm:"column:password_hash;not null"`
	CreatedAt    time.Time `gorm:"column:created_at;autoCreateTime"`
}

func (UserPasswordHistory) TableName() string {
	return "user_password_history"
}
