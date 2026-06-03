package tenant

import (
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTenantResponseDTO_JSONContract(t *testing.T) {
	id := uuid.New()
	dto := TenantResponseDTO{
		TenantUUID:  id,
		Name:        "acme",
		DisplayName: "Acme",
	}

	body, err := json.Marshal(dto)

	require.NoError(t, err)
	assert.Contains(t, string(body), `"tenant_id":"`+id.String()+`"`)
	assert.Contains(t, string(body), `"display_name":"Acme"`)
}

func TestTenantMemberDTO_JSONContract(t *testing.T) {
	memberID := uuid.New()
	userID := uuid.New()
	dto := TenantMemberResponseDTO{
		TenantMemberUUID: memberID,
		Role:             "owner",
		User:             &MemberUserResponseDTO{UserUUID: userID, Email: "owner@example.com"},
	}

	body, err := json.Marshal(dto)

	require.NoError(t, err)
	assert.Contains(t, string(body), `"tenant_member_id":"`+memberID.String()+`"`)
	assert.Contains(t, string(body), `"user_id":"`+userID.String()+`"`)
}
