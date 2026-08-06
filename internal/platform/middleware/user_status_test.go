package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/maintainerd/maintainerd-auth/internal/authctx"
	"github.com/maintainerd/maintainerd-auth/internal/shared"
)

// Deactivating, suspending or soft-deleting an account must take effect on the
// NEXT request. Before this, status was never consulted after authentication, so
// a disabled account stayed fully usable until its access token happened to
// expire — and a cached user context extended that even further.
func TestUserStatusGrantsAccess(t *testing.T) {
	cases := []struct {
		name   string
		user   *authctx.AuthUser
		allow  bool
		status int
	}{
		{"active user is allowed", &authctx.AuthUser{Status: shared.StatusActive}, true, http.StatusOK},
		{"inactive user is refused", &authctx.AuthUser{Status: "inactive"}, false, http.StatusUnauthorized},
		{"suspended user is refused", &authctx.AuthUser{Status: "suspended"}, false, http.StatusUnauthorized},
		{"pending user is refused", &authctx.AuthUser{Status: "pending"}, false, http.StatusUnauthorized},
		// Several projections legitimately omit the column; failing closed there
		// would lock every user out rather than deny one.
		{"unset status is treated as active", &authctx.AuthUser{}, true, http.StatusOK},
		{"nil user defers to the caller's own nil handling", nil, true, http.StatusOK},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			got := userStatusGrantsAccess(w, tc.user)
			if got != tc.allow {
				t.Fatalf("expected allow=%v, got %v", tc.allow, got)
			}
			if !tc.allow && w.Code != tc.status {
				t.Fatalf("expected %d on refusal, got %d", tc.status, w.Code)
			}
		})
	}
}
