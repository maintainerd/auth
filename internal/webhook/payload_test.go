package webhook

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/maintainerd/maintainerd-auth/internal/authevent"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/datatypes"
)

func TestBuildPayload(t *testing.T) {
	t.Run("success with all fields", func(t *testing.T) {
		actorID := int64(10)
		targetID := int64(20)
		userAgent := "Maintainerd Tests"
		description := "user logged in"
		createdAt := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
		eventUUID := uuid.New()

		payload := buildPayload(&authevent.AuthEvent{
			AuthEventUUID: eventUUID,
			TenantID:      1,
			ActorUserID:   &actorID,
			TargetUserID:  &targetID,
			IPAddress:     "203.0.113.10",
			UserAgent:     &userAgent,
			Category:      authevent.AuthEventCategoryAuthn,
			EventType:     authevent.AuthEventTypeLoginSuccess,
			Severity:      authevent.AuthEventSeverityInfo,
			Result:        authevent.AuthEventResultSuccess,
			Description:   &description,
			Metadata:      datatypes.JSON([]byte(`{"request_id":"req_123"}`)),
			CreatedAt:     createdAt,
		})

		require.NotEqual(t, uuid.Nil, payload.ID)
		assert.WithinDuration(t, time.Now().UTC(), payload.Timestamp, time.Second)
		assert.Equal(t, authevent.AuthEventTypeLoginSuccess, payload.Type)
		assert.Equal(t, eventUUID, payload.Data.EventUUID)
		assert.Equal(t, int64(1), payload.Data.TenantID)
		assert.Equal(t, actorID, *payload.Data.ActorUserID)
		assert.Equal(t, targetID, *payload.Data.TargetUserID)
		assert.Equal(t, "203.0.113.10", payload.Data.IPAddress)
		assert.Equal(t, userAgent, *payload.Data.UserAgent)
		assert.Equal(t, description, *payload.Data.Description)
		assert.JSONEq(t, `{"request_id":"req_123"}`, string(payload.Data.Metadata))
		assert.Equal(t, createdAt, payload.Data.CreatedAt)
	})
}
