package user

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// tenant.CreateByUserUUID only accepts SYSTEM-tenant users, and nothing used to
// expose that set — so the console's Add Member picker offered the caller's own
// tenant users and every choice 403'd. These pin the endpoint that closes it,
// and the boundary that keeps it from becoming a cross-tenant user list.
func TestUserHandler_ListMembershipCandidates(t *testing.T) {
	t.Run("no tenant returns 401", func(t *testing.T) {
		r := httptest.NewRequest(http.MethodGet, "/", nil)
		w := httptest.NewRecorder()
		NewUserHandler(&mockUserService{}).ListMembershipCandidates(w, r)
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("returns candidates with a reduced projection", func(t *testing.T) {
		candidateUUID := uuid.New()
		svc := &mockUserService{
			listMembershipCandidatesFn: func(_ *string, _, _ int) ([]MembershipCandidateDTO, int64, error) {
				return []MembershipCandidateDTO{{
					UserUUID: candidateUUID,
					Username: "ada",
					Email:    "ada@example.com",
					Fullname: "Ada Lovelace",
				}}, 1, nil
			},
		}
		r := withTenant(httptest.NewRequest(http.MethodGet, "/?page=1&limit=10", nil))
		w := httptest.NewRecorder()
		NewUserHandler(svc).ListMembershipCandidates(w, r)
		require.Equal(t, http.StatusOK, w.Code)

		// The projection must not leak fields beyond what a picker needs — these
		// are users from another tenant as far as most callers are concerned.
		var body struct {
			Data struct {
				Rows []map[string]any `json:"rows"`
			} `json:"data"`
		}
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
		require.Len(t, body.Data.Rows, 1)
		for key := range body.Data.Rows[0] {
			assert.Contains(t, []string{"user_id", "username", "email", "fullname"}, key,
				"unexpected field %q in the candidate projection", key)
		}
	})

	t.Run("the search term reaches the service", func(t *testing.T) {
		var gotSearch *string
		svc := &mockUserService{
			listMembershipCandidatesFn: func(search *string, _, _ int) ([]MembershipCandidateDTO, int64, error) {
				gotSearch = search
				return nil, 0, nil
			},
		}
		r := withTenant(httptest.NewRequest(http.MethodGet, "/?search=ada", nil))
		NewUserHandler(svc).ListMembershipCandidates(httptest.NewRecorder(), r)
		require.NotNil(t, gotSearch)
		assert.Equal(t, "ada", *gotSearch)
	})

	t.Run("service error is surfaced, not swallowed", func(t *testing.T) {
		svc := &mockUserService{
			listMembershipCandidatesFn: func(*string, int, int) ([]MembershipCandidateDTO, int64, error) {
				return nil, 0, errors.New("boom")
			},
		}
		r := withTenant(httptest.NewRequest(http.MethodGet, "/", nil))
		w := httptest.NewRecorder()
		NewUserHandler(svc).ListMembershipCandidates(w, r)
		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}
