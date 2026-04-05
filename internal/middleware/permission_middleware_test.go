package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/maintainerd/auth/internal/model"
	"github.com/stretchr/testify/assert"
)

// userWithPermissions returns a minimal User fixture with the given permission names.
func userWithPermissions(perms ...string) *model.User {
	var permissions []model.Permission
	for _, p := range perms {
		permissions = append(permissions, model.Permission{Name: p})
	}
	return &model.User{
		UserID: 1,
		Roles:  []model.Role{{Permissions: permissions}},
	}
}

func TestPermissionMiddleware(t *testing.T) {
	cases := []struct {
		name         string
		required     []string
		setupContext func(r *http.Request) *http.Request
		wantStatus   int
	}{
		{
			name:         "no user in context → 401",
			required:     []string{"read"},
			setupContext: func(r *http.Request) *http.Request { return r },
			wantStatus:   http.StatusUnauthorized,
		},
		{
			name:     "invalid type in context → 500",
			required: []string{"read"},
			setupContext: func(r *http.Request) *http.Request {
				ctx := context.WithValue(r.Context(), UserContextKey, "not-a-user")
				return r.WithContext(ctx)
			},
			wantStatus: http.StatusInternalServerError,
		},
		{
			name:     "user lacks required permission → 403",
			required: []string{"write"},
			setupContext: func(r *http.Request) *http.Request {
				ctx := context.WithValue(r.Context(), UserContextKey, userWithPermissions("read"))
				return r.WithContext(ctx)
			},
			wantStatus: http.StatusForbidden,
		},
		{
			name:     "user has required permission → 200",
			required: []string{"read"},
			setupContext: func(r *http.Request) *http.Request {
				ctx := context.WithValue(r.Context(), UserContextKey, userWithPermissions("read", "write"))
				return r.WithContext(ctx)
			},
			wantStatus: http.StatusOK,
		},
		{
			name:     "user has one of multiple required permissions → 200",
			required: []string{"admin", "write"},
			setupContext: func(r *http.Request) *http.Request {
				ctx := context.WithValue(r.Context(), UserContextKey, userWithPermissions("write"))
				return r.WithContext(ctx)
			},
			wantStatus: http.StatusOK,
		},
		{
			name:     "empty required list → 403",
			required: []string{},
			setupContext: func(r *http.Request) *http.Request {
				ctx := context.WithValue(r.Context(), UserContextKey, userWithPermissions("read"))
				return r.WithContext(ctx)
			},
			wantStatus: http.StatusForbidden,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req = tc.setupContext(req)
			rr := httptest.NewRecorder()
			PermissionMiddleware(tc.required)(okHandler()).ServeHTTP(rr, req)
			assert.Equal(t, tc.wantStatus, rr.Code)
		})
	}
}

func TestHasAnyPermission(t *testing.T) {
	t.Run("no roles → false", func(t *testing.T) {
		assert.False(t, hasAnyPermission(&model.User{}, []string{"read"}))
	})

	t.Run("has matching permission → true", func(t *testing.T) {
		assert.True(t, hasAnyPermission(userWithPermissions("read"), []string{"write", "read"}))
	})

	t.Run("no matching permission → false", func(t *testing.T) {
		assert.False(t, hasAnyPermission(userWithPermissions("read"), []string{"write", "admin"}))
	})

	t.Run("empty required list → false", func(t *testing.T) {
		assert.False(t, hasAnyPermission(userWithPermissions("read"), []string{}))
	})

	t.Run("multiple roles, permission in second role → true", func(t *testing.T) {
		user := &model.User{
			Roles: []model.Role{
				{Permissions: []model.Permission{{Name: "read"}}},
				{Permissions: []model.Permission{{Name: "admin"}}},
			},
		}
		assert.True(t, hasAnyPermission(user, []string{"admin"}))
	})
}

