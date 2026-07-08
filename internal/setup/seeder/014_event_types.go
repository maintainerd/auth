package seeder

import (
	"errors"
	"fmt"
	"log/slog"

	"github.com/maintainerd/maintainerd-auth/internal/event"
	"gorm.io/gorm"
)

// SeedEventTypes seeds the event_types catalog for a tenant with all v1.0.0
// integration event types. Event types are tenant-scoped: every tenant gets its
// own copy of the canonical catalog.
func SeedEventTypes(db *gorm.DB, tenantID int64) error {
	specs := event.AllEventTypes()

	for _, spec := range specs {
		exists, err := eventTypeExists(db, tenantID, spec.Key)
		if err != nil {
			return fmt.Errorf("failed to check event type %q: %w", spec.Key, err)
		}
		if exists {
			slog.Info("Event type already exists, skipping", "key", spec.Key, "tenant_id", tenantID)
			continue
		}

		et := event.EventType{
			TenantID:    tenantID,
			Key:         spec.Key,
			Category:    spec.Category,
			Description: spec.Description,
			Version:     spec.Version,
			IsActive:    true,
		}

		if err := db.Create(&et).Error; err != nil {
			return fmt.Errorf("failed to seed event type %q: %w", spec.Key, err)
		}

		slog.Info("Event type seeded", "key", spec.Key, "tenant_id", tenantID)
	}

	return nil
}

func eventTypeExists(db *gorm.DB, tenantID int64, key string) (bool, error) {
	var existing event.EventType
	err := db.Where("key = ? AND tenant_id = ?", key, tenantID).First(&existing).Error
	if err == nil {
		return true, nil
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return false, nil
	}
	return false, err
}
