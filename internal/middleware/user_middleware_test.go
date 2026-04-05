package middleware

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/model"
	"github.com/maintainerd/auth/internal/repository"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// ---------------------------------------------------------------------------
// Mock: UserRepository (middleware package scope)
// ---------------------------------------------------------------------------

type mockUserRepoMW struct {
	findBySubAndClientIDFn func(sub, cID string) (*model.User, error)
}

func (m *mockUserRepoMW) WithTx(_ *gorm.DB) repository.UserRepository { return m }
func (m *mockUserRepoMW) Create(e *model.User) (*model.User, error)   { return e, nil }
func (m *mockUserRepoMW) CreateOrUpdate(e *model.User) (*model.User, error) {
	return e, nil
}
func (m *mockUserRepoMW) FindAll(p ...string) ([]model.User, error)                   { return nil, nil }
func (m *mockUserRepoMW) FindByUUID(id any, p ...string) (*model.User, error)         { return nil, nil }
func (m *mockUserRepoMW) FindByUUIDs(ids []string, p ...string) ([]model.User, error) { return nil, nil }
func (m *mockUserRepoMW) FindByID(id any, p ...string) (*model.User, error)           { return nil, nil }
func (m *mockUserRepoMW) UpdateByUUID(id, data any) (*model.User, error)              { return nil, nil }
func (m *mockUserRepoMW) UpdateByID(id, data any) (*model.User, error)                { return nil, nil }
func (m *mockUserRepoMW) DeleteByUUID(id any) error                                   { return nil }
func (m *mockUserRepoMW) DeleteByID(id any) error                                     { return nil }
func (m *mockUserRepoMW) Paginate(c map[string]any, pg, lim int, p ...string) (*repository.PaginationResult[model.User], error) {
	return nil, nil
}
func (m *mockUserRepoMW) FindByUsername(u string) (*model.User, error)  { return nil, nil }
func (m *mockUserRepoMW) FindByEmail(e string) (*model.User, error)     { return nil, nil }
func (m *mockUserRepoMW) FindByPhone(p string) (*model.User, error)     { return nil, nil }
func (m *mockUserRepoMW) FindSuperAdmin() (*model.User, error)          { return nil, nil }
func (m *mockUserRepoMW) FindRoles(userID int64) ([]model.Role, error)  { return nil, nil }
func (m *mockUserRepoMW) SetEmailVerified(id uuid.UUID, v bool) error   { return nil }
func (m *mockUserRepoMW) SetStatus(id uuid.UUID, s string) error        { return nil }
func (m *mockUserRepoMW) FindByEmailAndTenantID(e string, tID int64) (*model.User, error) {
	return nil, nil
}
func (m *mockUserRepoMW) FindPaginated(f repository.UserRepositoryGetFilter) (*repository.PaginationResult[model.User], error) {
	return &repository.PaginationResult[model.User]{}, nil
}
func (m *mockUserRepoMW) FindBySubAndClientID(sub, cID string) (*model.User, error) {
	if m.findBySubAndClientIDFn != nil {
		return m.findBySubAndClientIDFn(sub, cID)
	}
	return nil, nil
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// newFakeRedis returns a Redis client pointing at an invalid address so every
// cache operation fails immediately, exercising the database-fallback path.
func newFakeRedis() *redis.Client {
	return redis.NewClient(&redis.Options{
		Addr:        "localhost:0",
		DialTimeout: 20 * time.Millisecond,
		ReadTimeout: 20 * time.Millisecond,
	})
}

// withJWTContext injects sub and clientID (normally set by JWTAuthMiddleware).
func withJWTContext(r *http.Request, sub, clientID string) *http.Request {
	ctx := context.WithValue(r.Context(), SubKey, sub)
	ctx = context.WithValue(ctx, ClientIDKey, clientID)
	return r.WithContext(ctx)
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

func TestUserContextMiddleware(t *testing.T) {
	const sub = "user-sub-123"
	const clientID = "client-abc"
	userUUID := uuid.New()

	cases := []struct {
		name              string
		findBySubClientID func(sub, cID string) (*model.User, error)
		wantStatus        int
		checkContext      func(t *testing.T, captured *model.User)
	}{
		{
			name: "user found → context populated → 200",
			findBySubClientID: func(_, _ string) (*model.User, error) {
				return &model.User{UserID: 1, UserUUID: userUUID}, nil
			},
			wantStatus: http.StatusOK,
			checkContext: func(t *testing.T, captured *model.User) {
				require.NotNil(t, captured)
				assert.Equal(t, userUUID, captured.UserUUID)
			},
		},
		{
			name:              "user not found → 401",
			findBySubClientID: func(_, _ string) (*model.User, error) { return nil, nil },
			wantStatus:        http.StatusUnauthorized,
		},
		{
			name: "db error → 500",
			findBySubClientID: func(_, _ string) (*model.User, error) {
				return nil, errors.New("db error")
			},
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var captured *model.User
			next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				captured, _ = r.Context().Value(UserContextKey).(*model.User)
				w.WriteHeader(http.StatusOK)
			})

			repo := &mockUserRepoMW{findBySubAndClientIDFn: tc.findBySubClientID}
			mw := UserContextMiddleware(repo, newFakeRedis())

			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req = withJWTContext(req, sub, clientID)
			rr := httptest.NewRecorder()
			mw(next).ServeHTTP(rr, req)

			assert.Equal(t, tc.wantStatus, rr.Code)
			if tc.checkContext != nil {
				tc.checkContext(t, captured)
			}
		})
	}
}

