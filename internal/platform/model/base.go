package model

import (
	"time"

	"gorm.io/gorm"
)

type Base struct {
	CreatedBy *int64          `gorm:"column:created_by"`
	UpdatedBy *int64          `gorm:"column:updated_by"`
	CreatedAt time.Time       `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt time.Time       `gorm:"column:updated_at;autoUpdateTime"`
	DeletedAt gorm.DeletedAt  `gorm:"column:deleted_at;index"`
}
