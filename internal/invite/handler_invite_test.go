package invite

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/maintainerd/maintainerd-auth/internal/shared"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInviteHandler_Send_NoTenant(t *testing.T) {
	h := NewInviteHandler(&mockInviteService{})
	r := httptest.NewRequest(http.MethodPost, "/invites", nil)
	w := httptest.NewRecorder()
	h.Send(w, r)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestInviteHandler_Send_NoUser(t *testing.T) {
	h := NewInviteHandler(&mockInviteService{})
	r := withTenant(httptest.NewRequest(http.MethodPost, "/invites", nil))
	w := httptest.NewRecorder()
	h.Send(w, r)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestInviteHandler_Send_BadJSON(t *testing.T) {
	h := NewInviteHandler(&mockInviteService{})
	r := withTenantAndUser(httptest.NewRequest(http.MethodPost, "/invites", nil))
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.Send(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestInviteHandler_Send_ServiceError(t *testing.T) {
	svc := &mockInviteService{
		sendInviteFn: func(tid int64, email string, uid int64, registrationFlowUUID *string, callbackURL *string) (*Invite, error) {
			return nil, assert.AnError
		},
	}
	h := NewInviteHandler(svc)
	r := withTenantAndUser(jsonReq(t, http.MethodPost, "/invites", map[string]interface{}{
		"email": "user@example.com",
	}))
	w := httptest.NewRecorder()
	h.Send(w, r)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestInviteHandler_Send_ValidationError(t *testing.T) {
	h := NewInviteHandler(&mockInviteService{})
	r := withTenantAndUser(jsonReq(t, http.MethodPost, "/invites", map[string]any{}))
	w := httptest.NewRecorder()
	h.Send(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestInviteHandler_Send_Success(t *testing.T) {
	svc := &mockInviteService{
		sendInviteFn: func(tid int64, email string, uid int64, registrationFlowUUID *string, callbackURL *string) (*Invite, error) {
			return &Invite{}, nil
		},
	}
	h := NewInviteHandler(svc)
	r := withTenantAndUser(jsonReq(t, http.MethodPost, "/invites", map[string]interface{}{
		"email": "user@example.com",
	}))
	w := httptest.NewRecorder()
	h.Send(w, r)
	assert.Equal(t, http.StatusOK, w.Code)
}

// GetInviteContext must return the RAW token the caller supplied, never the
// stored value: invites.invite_token now holds a digest, so echoing the record
// would hand the identity app a string that can never be redeemed — and would
// leak the storage form of a credential to an unauthenticated endpoint.
func TestInviteHandler_GetInviteContext_EchoesRawTokenNotStoredDigest(t *testing.T) {
	const rawToken = "raw-invite-token"
	expires := time.Now().Add(time.Hour)
	h := NewInviteHandler(&mockInviteService{
		getByTokenFn: func(got string) (*Invite, error) {
			// The service takes the raw token; hashing is the repository's job.
			assert.Equal(t, rawToken, got)
			return &Invite{
				InvitedEmail:    "ada@example.com",
				InviteTokenHash: hashInviteToken(rawToken),
				Status:          shared.StatusPending,
				ExpiresAt:       &expires,
			}, nil
		},
	})

	r := httptest.NewRequest(http.MethodGet, "/invite?invite_token="+rawToken, nil)
	w := httptest.NewRecorder()
	h.GetInviteContext(w, r)

	require.Equal(t, http.StatusOK, w.Code)
	var body struct {
		Data InviteContextResponseDTO `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, rawToken, body.Data.InviteToken)
	assert.NotContains(t, w.Body.String(), hashInviteToken(rawToken))
}

// The Send handler marshals the whole Invite into the audit-log "changes" blob.
// It used to serialise the raw invite token into that log, turning the audit
// trail into a second store of live account-creation credentials; the field is
// now both hashed and json:"-".
func TestInvite_IsNeverSerialisedIntoAuditPayload(t *testing.T) {
	blob, err := json.Marshal(map[string]any{"after": &Invite{
		InvitedEmail:    "ada@example.com",
		InviteTokenHash: hashInviteToken("raw-invite-token"),
	}})
	require.NoError(t, err)

	assert.NotContains(t, string(blob), "raw-invite-token")
	assert.NotContains(t, string(blob), hashInviteToken("raw-invite-token"))
	assert.NotContains(t, string(blob), "InviteToken")
}
