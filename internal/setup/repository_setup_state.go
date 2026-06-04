package setup

import (
	"errors"
	"time"

	"gorm.io/gorm"
)

type SetupStateRepository interface {
	WithTx(tx *gorm.DB) SetupStateRepository
	FindByKey(key string) (*SetupState, error)
	IsComplete(key string) (bool, error)
	MarkComplete(key string, completedAt time.Time) (*SetupState, error)
}

type setupStateRepository struct {
	db *gorm.DB
}

type openSetupStateRepository struct{}

func NewSetupStateRepository(db *gorm.DB) SetupStateRepository {
	return &setupStateRepository{db: db}
}

func NewOpenSetupStateRepository() SetupStateRepository {
	return openSetupStateRepository{}
}

func (r *setupStateRepository) WithTx(tx *gorm.DB) SetupStateRepository {
	return &setupStateRepository{db: tx}
}

func (r *setupStateRepository) FindByKey(key string) (*SetupState, error) {
	var state SetupState
	err := r.db.Where("key = ?", key).First(&state).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &state, nil
}

func (r *setupStateRepository) IsComplete(key string) (bool, error) {
	state, err := r.FindByKey(key)
	if err != nil || state == nil {
		return false, err
	}
	return state.IsComplete, nil
}

func (r *setupStateRepository) MarkComplete(key string, completedAt time.Time) (*SetupState, error) {
	state := &SetupState{
		Key:         key,
		IsComplete:  true,
		CompletedAt: &completedAt,
	}
	err := r.db.Where("key = ?", key).Assign(map[string]any{
		"is_complete":  true,
		"completed_at": completedAt,
	}).FirstOrCreate(state).Error
	if err != nil {
		return nil, err
	}
	return state, nil
}

func (openSetupStateRepository) WithTx(_ *gorm.DB) SetupStateRepository {
	return openSetupStateRepository{}
}

func (openSetupStateRepository) FindByKey(_ string) (*SetupState, error) {
	return nil, nil
}

func (openSetupStateRepository) IsComplete(_ string) (bool, error) {
	return false, nil
}

func (openSetupStateRepository) MarkComplete(key string, completedAt time.Time) (*SetupState, error) {
	return &SetupState{Key: key, IsComplete: true, CompletedAt: &completedAt}, nil
}
