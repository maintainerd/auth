package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/maintainerd/maintainerd-auth/internal/authctx"
	"github.com/maintainerd/maintainerd-auth/internal/platform/jwt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOptionalUserContextMiddleware(t *testing.T) {
	initTestJWTKeys(t)

	userUUID := uuid.New()
	validToken, err := jwt.GenerateAccessToken(
		userUUID.String(), "openid", "https://auth.example.com",
		"https://api.example.com", "my-client", "provider-1",
	)
	require.NoError(t, err)

	resolvedUser := func(_, _ string) (*authctx.UserContext, error) {
		return &authctx.UserContext{User: &authctx.AuthUser{UserID: 1, UserUUID: userUUID}}, nil
	}

	cases := []struct {
		name     string
		setup    func(r *http.Request)
		findFn   func(sub, cID string) (*authctx.UserContext, error)
		wantUser bool
	}{
		{
			name:  "no token -> unauthenticated, no 401",
			setup: func(_ *http.Request) {},
		},
		{
			name: "invalid token -> unauthenticated",
			setup: func(r *http.Request) {
				r.AddCookie(&http.Cookie{Name: "access_token", Value: "garbage.token"})
			},
		},
		{
			name: "valid cookie token + user found -> populated",
			setup: func(r *http.Request) {
				r.AddCookie(&http.Cookie{Name: "access_token", Value: validToken})
			},
			findFn:   resolvedUser,
			wantUser: true,
		},
		{
			name: "valid bearer token + user found -> populated",
			setup: func(r *http.Request) {
				r.Header.Set("Authorization", "Bearer "+validToken)
			},
			findFn:   resolvedUser,
			wantUser: true,
		},
		{
			name: "valid token + user not found -> unauthenticated",
			setup: func(r *http.Request) {
				r.AddCookie(&http.Cookie{Name: "access_token", Value: validToken})
			},
			findFn: func(_, _ string) (*authctx.UserContext, error) { return nil, nil },
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var captured *authctx.AuthUser
			next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				captured = AuthFromRequest(r).User
				w.WriteHeader(http.StatusOK)
			})

			provider := &mockContextProvider{findFn: tc.findFn}
			mw := OptionalUserContextMiddleware(provider, newFakeCache())

			req := httptest.NewRequest(http.MethodGet, "/oauth/authorize", nil)
			tc.setup(req)
			rr := httptest.NewRecorder()
			mw(next).ServeHTTP(rr, req)

			// The middleware never rejects the request — absence of a session is
			// handled downstream (login_required), not by a 401 here.
			assert.Equal(t, http.StatusOK, rr.Code)
			if tc.wantUser {
				require.NotNil(t, captured)
				assert.Equal(t, userUUID, captured.UserUUID)
			} else {
				assert.Nil(t, captured)
			}
		})
	}
}
