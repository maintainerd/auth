package seeder

import (
	"errors"
	"fmt"
	"log/slog"

	"github.com/maintainerd/auth/internal/event"
	"gorm.io/gorm"
)

// SeedEventTypes seeds the event_types catalog table with all v1.0.0 integration event types.
func SeedEventTypes(db *gorm.DB) error {
	specs := event.AllEventTypes()

	for _, spec := range specs {
		exists, err := eventTypeExists(db, spec.Key)
		if err != nil {
			return fmt.Errorf("failed to check event type %q: %w", spec.Key, err)
		}
		if exists {
			slog.Info("Event type already exists, skipping", "key", spec.Key)
			continue
		}

		et := event.EventType{
			Key:         spec.Key,
			Category:    spec.Category,
			Description: spec.Description,
			Version:     spec.Version,
			IsActive:    true,
		}

		if err := db.Create(&et).Error; err != nil {
			return fmt.Errorf("failed to seed event type %q: %w", spec.Key, err)
		}

		slog.Info("Event type seeded", "key", spec.Key)
	}

	return nil
}

func eventTypeExists(db *gorm.DB, key string) (bool, error) {
	var existing event.EventType
	err := db.Where("key = ?", key).First(&existing).Error
	if err == nil {
		return true, nil
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return false, nil
	}
	return false, err
}
