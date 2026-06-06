package event

import (
	"time"

	"github.com/google/uuid"
)

// EventTypeServiceDataResult is the service-layer representation of an event type.
type EventTypeServiceDataResult struct {
	EventTypeUUID uuid.UUID
	Key           string
	Category      string
	Description   string
	Version       int
	IsActive      bool
	CreatedAt     time.Time
	UpdatedAt     time.Time
}
