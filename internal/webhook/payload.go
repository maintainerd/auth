package webhook

import (
	"time"

	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/model"
	"gorm.io/datatypes"
)

type eventPayload struct {
	ID        uuid.UUID `json:"id"`
	Timestamp time.Time `json:"timestamp"`
	Type      string    `json:"type"`
	Data      eventData `json:"data"`
}

type eventData struct {
	EventUUID    uuid.UUID      `json:"event_uuid"`
	TenantID     int64          `json:"tenant_id"`
	Category     string         `json:"category"`
	Severity     string         `json:"severity"`
	Result       string         `json:"result"`
	ActorUserID  *int64         `json:"actor_user_id,omitempty"`
	TargetUserID *int64         `json:"target_user_id,omitempty"`
	IPAddress    string         `json:"ip_address"`
	UserAgent    *string        `json:"user_agent,omitempty"`
	Description  *string        `json:"description,omitempty"`
	Metadata     datatypes.JSON `json:"metadata,omitempty"`
	CreatedAt    time.Time      `json:"created_at"`
}

func buildPayload(event *model.AuthEvent) eventPayload {
	return eventPayload{
		ID:        uuid.New(),
		Timestamp: time.Now().UTC(),
		Type:      event.EventType,
		Data: eventData{
			EventUUID:    event.AuthEventUUID,
			TenantID:     event.TenantID,
			Category:     event.Category,
			Severity:     event.Severity,
			Result:       event.Result,
			ActorUserID:  event.ActorUserID,
			TargetUserID: event.TargetUserID,
			IPAddress:    event.IPAddress,
			UserAgent:    event.UserAgent,
			Description:  event.Description,
			Metadata:     event.Metadata,
			CreatedAt:    event.CreatedAt,
		},
	}
}
