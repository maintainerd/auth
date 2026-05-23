package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type UserBackupCode struct {
	BackupCodeID   int64      `gorm:"column:backup_code_id;primaryKey"`
	BackupCodeUUID uuid.UUID  `gorm:"column:backup_code_uuid;unique"`
	UserID         int64      `gorm:"column:user_id"`
	CodeHash       string     `gorm:"column:code_hash"`
	Used           bool       `gorm:"column:used;default:false"`
	UsedAt         *time.Time `gorm:"column:used_at"`
	CreatedAt      time.Time  `gorm:"column:created_at;autoCreateTime"`
}

func (UserBackupCode) TableName() string { return "user_backup_codes" }

func (b *UserBackupCode) BeforeCreate(tx *gorm.DB) error {
	if b.BackupCodeUUID == uuid.Nil {
		b.BackupCodeUUID = uuid.New()
	}
	return nil
}
