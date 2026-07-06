package authn

import (
	"context"
	"errors"
	"testing"
	"time"

	jwtlib "github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockLogoutUserRepo struct {
	findByUUIDFn     func(uuid.UUID) (*User, error)
	findByIDFn       func(interface{}, ...string) (*User, error)
	findByUsernameFn func(string) (*User, error)
	findByEmailFn    func(string) (*User, error)
}

func (m *mockLogoutUserRepo) WithTx(_ *gorm.DB) UserRepository { return m }
func (m *mockLogoutUserRepo) FindByUUID(id any, p ...string) (*User, error) {
	if m.findByUUIDFn != nil {
		return m.findByUUIDFn(id.(uuid.UUID))
	}
	return nil, nil
}
func (m *mockLogoutUserRepo) FindByID(id interface{}, p ...string) (*User, error) {
	if m.findByIDFn != nil {
		return m.findByIDFn(id, p...)
	}
	return nil, nil
}
func (m *mockLogoutUserRepo) FindByUsername(username string) (*User, error) {
	if m.findByUsernameFn != nil {
		return m.findByUsernameFn(username)
	}
	return nil, nil
}
func (m *mockLogoutUserRepo) FindByEmail(email string) (*User, error) {
	if m.findByEmailFn != nil {
		return m.findByEmailFn(email)
	}
	return nil, nil
}
func (m *mockLogoutUserRepo) FindByEmailAndTenantID(email string, tenantID int64) (*User, error) {
	return nil, nil
}
func (m *mockLogoutUserRepo) FindByUsernameAndTenantID(username string, tenantID int64) (*User, error) {
	if m.findByUsernameFn != nil {
		return m.findByUsernameFn(username)
	}
	return nil, nil
}
func (m *mockLogoutUserRepo) FindByPhoneAndTenantID(phone string, tenantID int64) (*User, error) {
	return nil, nil
}
func (m *mockLogoutUserRepo) FindByPhone(phone string) (*User, error) { return nil, nil }
func (m *mockLogoutUserRepo) FindSuperAdmin() (*User, error)          { return nil, nil }
func (m *mockLogoutUserRepo) FindRoles(userID int64) ([]Role, error)  { return nil, nil }
func (m *mockLogoutUserRepo) FindBySubAndClientID(sub, clientID string) (*User, error) {
	return nil, nil
}
func (m *mockLogoutUserRepo) FindPaginated(filter UserRepositoryGetFilter) (*PaginationResult[User], error) {
	return nil, nil
}
func (m *mockLogoutUserRepo) SetEmailVerified(id uuid.UUID, verified bool) error    { return nil }
func (m *mockLogoutUserRepo) SetStatus(id uuid.UUID, status string) error           { return nil }
func (m *mockLogoutUserRepo) SetForcePasswordChange(id uuid.UUID, force bool) error { return nil }
func (m *mockLogoutUserRepo) ClearEmailChange(id uuid.UUID) error { return nil }
func (m *mockLogoutUserRepo) UpdateEmail(id uuid.UUID, email string) error       { return nil }
func (m *mockLogoutUserRepo) UpdateUsername(id uuid.UUID, username string) error { return nil }
func (m *mockLogoutUserRepo) Create(e *User) (*User, error) { return e, nil }
func (m *mockLogoutUserRepo) CreateOrUpdate(e *User) (*User, error)              { return e, nil }
func (m *mockLogoutUserRepo) FindAll(p ...string) ([]User, error)                { return nil, nil }
func (m *mockLogoutUserRepo) FindByUUIDs(ids []string, p ...string) ([]User, error) {
	return nil, nil
}
func (m *mockLogoutUserRepo) UpdateByUUID(id, data any) (*User, error) { return nil, nil }
func (m *mockLogoutUserRepo) UpdateByID(id, data any) (*User, error)   { return nil, nil }
func (m *mockLogoutUserRepo) DeleteByUUID(id any) error                { return nil }
func (m *mockLogoutUserRepo) DeleteByID(id any) error                  { return nil }
func (m *mockLogoutUserRepo) Paginate(c map[string]any, pg, lim int, p ...string) (*PaginationResult[User], error) {
	return nil, nil
}

type mockLogoutSessionService struct {
	revokeAllSessionsFn func(int64) error
}

type recordingLogoutJTIDenylister struct {
	jti string
	ttl time.Duration
	err error
}

func (r *recordingLogoutJTIDenylister) DenyJTI(_ context.Context, jti string, ttl time.Duration) error {
	r.jti = jti
	r.ttl = ttl
	return r.err
}

func (r *recordingLogoutJTIDenylister) IsJTIDenied(_ context.Context, _ string) (bool, error) {
	return false, nil
}

func (m *mockLogoutSessionService) ListSessions(ctx context.Context, userID int64) ([]*SessionDataResult, error) {
	return nil, nil
}
func (m *mockLogoutSessionService) RevokeSession(ctx context.Context, userID int64, sessionUUID uuid.UUID) error {
	return nil
}
func (m *mockLogoutSessionService) RevokeAllSessions(ctx context.Context, userID int64) error {
	if m.revokeAllSessionsFn != nil {
		return m.revokeAllSessionsFn(userID)
	}
	return nil
}
func (m *mockLogoutSessionService) CreateSession(ctx context.Context, userID int64, ipAddress, userAgent string) (*UserSession, error) {
	return nil, nil
}
func (m *mockLogoutSessionService) EnforceConcurrentLimit(ctx context.Context, userUUID uuid.UUID, userID int64) error {
	return nil
}
func (m *mockLogoutSessionService) ValidateAndTouch(ctx context.Context, sessionUUID uuid.UUID, userID int64) error {
	return nil
}

func TestLoginService_Logout(t *testing.T) {
	userUUID := uuid.New()

	t.Run("empty access token returns nil", func(t *testing.T) {
		svc := &loginService{userRepo: &mockLogoutUserRepo{}, sessionService: &mockLogoutSessionService{}}
		err := svc.Logout(context.Background(), "")
		require.NoError(t, err)
	})

	t.Run("nil sessionService returns nil", func(t *testing.T) {
		svc := &loginService{userRepo: &mockLogoutUserRepo{}}
		err := svc.Logout(context.Background(), "some-token")
		require.NoError(t, err)
	})

	t.Run("malformed JWT returns nil", func(t *testing.T) {
		svc := &loginService{userRepo: &mockLogoutUserRepo{}, sessionService: &mockLogoutSessionService{}}
		err := svc.Logout(context.Background(), "not-a-jwt")
		require.NoError(t, err)
	})

	t.Run("JWT without sub claim returns nil", func(t *testing.T) {
		token := jwtlib.NewWithClaims(jwtlib.SigningMethodHS256, jwtlib.MapClaims{"no": "sub"})
		tokenStr, _ := token.SignedString([]byte("secret"))
		svc := &loginService{userRepo: &mockLogoutUserRepo{}, sessionService: &mockLogoutSessionService{}}
		err := svc.Logout(context.Background(), tokenStr)
		require.NoError(t, err)
	})

	t.Run("JWT with non map claims returns nil", func(t *testing.T) {
		token := jwtlib.NewWithClaims(jwtlib.SigningMethodHS256, jwtlib.RegisteredClaims{Subject: userUUID.String()})
		tokenStr, _ := token.SignedString([]byte("secret"))
		svc := &loginService{userRepo: &mockLogoutUserRepo{}, sessionService: &mockLogoutSessionService{}}
		err := svc.Logout(context.Background(), tokenStr)
		require.NoError(t, err)
	})

	t.Run("JWT with empty sub returns nil", func(t *testing.T) {
		token := jwtlib.NewWithClaims(jwtlib.SigningMethodHS256, jwtlib.MapClaims{"sub": ""})
		tokenStr, _ := token.SignedString([]byte("secret"))
		svc := &loginService{userRepo: &mockLogoutUserRepo{}, sessionService: &mockLogoutSessionService{}}
		err := svc.Logout(context.Background(), tokenStr)
		require.NoError(t, err)
	})

	t.Run("invalid UUID sub returns nil", func(t *testing.T) {
		token := jwtlib.NewWithClaims(jwtlib.SigningMethodHS256, jwtlib.MapClaims{"sub": "not-a-uuid"})
		tokenStr, _ := token.SignedString([]byte("secret"))
		svc := &loginService{userRepo: &mockLogoutUserRepo{}, sessionService: &mockLogoutSessionService{}}
		err := svc.Logout(context.Background(), tokenStr)
		require.NoError(t, err)
	})

	t.Run("user not found returns nil", func(t *testing.T) {
		token := jwtlib.NewWithClaims(jwtlib.SigningMethodHS256, jwtlib.MapClaims{"sub": userUUID.String()})
		tokenStr, _ := token.SignedString([]byte("secret"))

		repo := &mockLogoutUserRepo{
			findByUUIDFn: func(uuid.UUID) (*User, error) { return nil, nil },
		}
		svc := &loginService{userRepo: repo, sessionService: &mockLogoutSessionService{}}
		err := svc.Logout(context.Background(), tokenStr)
		require.NoError(t, err)
	})

	t.Run("user lookup error returns nil", func(t *testing.T) {
		token := jwtlib.NewWithClaims(jwtlib.SigningMethodHS256, jwtlib.MapClaims{"sub": userUUID.String()})
		tokenStr, _ := token.SignedString([]byte("secret"))

		repo := &mockLogoutUserRepo{
			findByUUIDFn: func(uuid.UUID) (*User, error) { return nil, errors.New("db error") },
		}
		svc := &loginService{userRepo: repo, sessionService: &mockLogoutSessionService{}}
		err := svc.Logout(context.Background(), tokenStr)
		require.NoError(t, err)
	})

	t.Run("session revoke error is returned", func(t *testing.T) {
		token := jwtlib.NewWithClaims(jwtlib.SigningMethodHS256, jwtlib.MapClaims{"sub": userUUID.String()})
		tokenStr, _ := token.SignedString([]byte("secret"))

		repo := &mockLogoutUserRepo{
			findByUUIDFn: func(uuid.UUID) (*User, error) { return &User{UserID: 1}, nil },
		}
		sess := &mockLogoutSessionService{
			revokeAllSessionsFn: func(int64) error { return errors.New("revoke error") },
		}
		svc := &loginService{userRepo: repo, sessionService: sess}
		err := svc.Logout(context.Background(), tokenStr)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "revoke error")
	})

	t.Run("success revokes all sessions", func(t *testing.T) {
		token := jwtlib.NewWithClaims(jwtlib.SigningMethodHS256, jwtlib.MapClaims{"sub": userUUID.String()})
		tokenStr, _ := token.SignedString([]byte("secret"))

		called := false
		repo := &mockLogoutUserRepo{
			findByUUIDFn: func(uuid.UUID) (*User, error) { return &User{UserID: 42}, nil },
		}
		sess := &mockLogoutSessionService{
			revokeAllSessionsFn: func(uid int64) error {
				called = true
				require.Equal(t, int64(42), uid)
				return nil
			},
		}
		svc := &loginService{userRepo: repo, sessionService: sess}
		err := svc.Logout(context.Background(), tokenStr)
		require.NoError(t, err)
		require.True(t, called)
	})

	t.Run("success denylists access token jti", func(t *testing.T) {
		token := jwtlib.NewWithClaims(jwtlib.SigningMethodHS256, jwtlib.MapClaims{
			"sub": userUUID.String(),
			"jti": "logout-jti",
			"exp": time.Now().Add(time.Hour).Unix(),
		})
		tokenStr, _ := token.SignedString([]byte("secret"))

		denylist := &recordingLogoutJTIDenylister{}
		repo := &mockLogoutUserRepo{
			findByUUIDFn: func(uuid.UUID) (*User, error) { return &User{UserID: 42}, nil },
		}
		sess := &mockLogoutSessionService{
			revokeAllSessionsFn: func(int64) error { return nil },
		}
		svc := &loginService{userRepo: repo, sessionService: sess, jtiDenylist: denylist}

		err := svc.Logout(context.Background(), tokenStr)

		require.NoError(t, err)
		assert.Equal(t, "logout-jti", denylist.jti)
		assert.Positive(t, denylist.ttl)
	})

	t.Run("denylist error is returned before session revoke", func(t *testing.T) {
		token := jwtlib.NewWithClaims(jwtlib.SigningMethodHS256, jwtlib.MapClaims{
			"sub": userUUID.String(),
			"jti": "logout-jti",
			"exp": time.Now().Add(time.Hour).Unix(),
		})
		tokenStr, _ := token.SignedString([]byte("secret"))

		denylist := &recordingLogoutJTIDenylister{err: errors.New("denylist error")}
		svc := &loginService{userRepo: &mockLogoutUserRepo{}, sessionService: &mockLogoutSessionService{}, jtiDenylist: denylist}

		err := svc.Logout(context.Background(), tokenStr)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "denylist error")
	})
}
