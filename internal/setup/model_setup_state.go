package setup

import "time"

const SetupStateBootstrap = "bootstrap"

type SetupState struct {
	SetupStateID int64      `gorm:"column:setup_state_id;primaryKey"`
	Key          string     `gorm:"column:key;uniqueIndex;not null"`
	IsComplete   bool       `gorm:"column:is_complete;not null;default:false"`
	CompletedAt  *time.Time `gorm:"column:completed_at"`
	CreatedAt    time.Time  `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt    time.Time  `gorm:"column:updated_at;autoUpdateTime"`
}

func (SetupState) TableName() string {
	return "setup_states"
}
