package authevent

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAuthEventResponseDTO_JSONContract(t *testing.T) {
	metadata := map[string]any{"ip_risk": "low"}
	dto := AuthEventResponseDTO{
		AuthEventID: "event-1",
		IPAddress:   "10.0.0.1",
		Category:    AuthEventCategoryAuthn,
		EventType:   AuthEventTypeLoginSuccess,
		Severity:    AuthEventSeverityInfo,
		Result:      AuthEventResultSuccess,
		Metadata:    &metadata,
		CreatedAt:   time.Unix(100, 0).UTC(),
	}

	body, err := json.Marshal(dto)

	require.NoError(t, err)
	assert.Contains(t, string(body), `"auth_event_id":"event-1"`)
	assert.Contains(t, string(body), `"ip_risk":"low"`)
	// Internal integer identifiers must never be serialized outbound — only UUIDs
	// (auth_event_id) leave the service.
	assert.NotContains(t, string(body), `"tenant_id"`)
	assert.NotContains(t, string(body), `"actor_user_id"`)
	assert.NotContains(t, string(body), `"target_user_id"`)
}
