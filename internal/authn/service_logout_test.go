package authn

import (
	"context"
	"errors"
	"github.com/maintainerd/maintainerd-auth/internal/platform/apperror"
	"testing"
	"time"

	jwtlib "github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockLogoutUserRepo struct {
	findBySubFn      func(sub, clientID string) (*User, error)
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
	if m.findBySubFn != nil {
		return m.findBySubFn(sub, clientID)
	}
	return nil, nil
}
func (m *mockLogoutUserRepo) FindPaginated(filter UserRepositoryGetFilter) (*PaginationResult[User], error) {
	return nil, nil
}
func (m *mockLogoutUserRepo) SetEmailVerified(id uuid.UUID, verified bool) error    { return nil }
func (m *mockLogoutUserRepo) SetStatus(id uuid.UUID, status string) error           { return nil }
func (m *mockLogoutUserRepo) SetForcePasswordChange(id uuid.UUID, force bool) error { return nil }
func (m *mockLogoutUserRepo) ClearEmailChange(id uuid.UUID) error                   { return nil }
func (m *mockLogoutUserRepo) UpdateEmail(id uuid.UUID, email string) error          { return nil }
func (m *mockLogoutUserRepo) UpdateUsername(id uuid.UUID, username string) error    { return nil }
func (m *mockLogoutUserRepo) Create(e *User) (*User, error)                         { return e, nil }
func (m *mockLogoutUserRepo) CreateOrUpdate(e *User) (*User, error)                 { return e, nil }
func (m *mockLogoutUserRepo) FindAll(p ...string) ([]User, error)                   { return nil, nil }
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
	revokeSessionFn     func(int64, uuid.UUID) error
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
	if m.revokeSessionFn != nil {
		return m.revokeSessionFn(userID, sessionUUID)
	}
	return nil
}
func (m *mockLogoutSessionService) RevokeAllSessions(ctx context.Context, userID int64, reason string) error {
	if m.revokeAllSessionsFn != nil {
		return m.revokeAllSessionsFn(userID)
	}
	return nil
}
func (m *mockLogoutSessionService) CreateSession(ctx context.Context, userID, tenantID int64, ipAddress, userAgent string) (*UserSession, error) {
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

	t.Run("unresolvable sub returns nil", func(t *testing.T) {
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
			findBySubFn: func(string, string) (*User, error) { return nil, errors.New("db error") },
		}
		svc := &loginService{userRepo: repo, sessionService: &mockLogoutSessionService{}}
		err := svc.Logout(context.Background(), tokenStr)
		require.NoError(t, err)
	})

	t.Run("session revoke error is returned", func(t *testing.T) {
		token := jwtlib.NewWithClaims(jwtlib.SigningMethodHS256, jwtlib.MapClaims{
			"sub": userUUID.String(),
			"sid": uuid.New().String(),
		})
		tokenStr, _ := token.SignedString([]byte("secret"))

		repo := &mockLogoutUserRepo{
			findBySubFn: func(string, string) (*User, error) { return &User{UserID: 1}, nil },
		}
		sess := &mockLogoutSessionService{
			revokeSessionFn: func(int64, uuid.UUID) error { return errors.New("revoke error") },
		}
		svc := &loginService{userRepo: repo, sessionService: sess}
		err := svc.Logout(context.Background(), tokenStr)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "revoke error")
	})

	// A logout must never revoke a session it cannot identify.
	//
	// This used to fall back to RevokeAllSessions when the token carried no sid,
	// which signed the user out of every OTHER browser and their phone — alarming
	// behaviour, and it fired on every console logout because OAuth-minted tokens
	// had no sid at all. Tokens are session-stamped at /authorize now; when a sid
	// is genuinely absent the access token is already denylisted, and revoking
	// nothing beats guessing.
	t.Run("no sid revokes nothing rather than signing the user out everywhere", func(t *testing.T) {
		token := jwtlib.NewWithClaims(jwtlib.SigningMethodHS256, jwtlib.MapClaims{"sub": userUUID.String()})
		tokenStr, _ := token.SignedString([]byte("secret"))

		repo := &mockLogoutUserRepo{
			findBySubFn: func(string, string) (*User, error) { return &User{UserID: 42}, nil },
		}
		sess := &mockLogoutSessionService{
			revokeAllSessionsFn: func(int64) error {
				t.Fatal("logout must not revoke all sessions: it would sign the user out of other browsers and mobile")
				return nil
			},
			revokeSessionFn: func(int64, uuid.UUID) error {
				t.Fatal("no session is identifiable, so none should be revoked")
				return nil
			},
		}
		svc := &loginService{userRepo: repo, sessionService: sess}
		err := svc.Logout(context.Background(), tokenStr)
		require.NoError(t, err)
	})

	// Console and identity share one browser, one cookie domain and therefore one
	// user_sessions row, so whichever logs out first revokes it and the other is
	// signed out too. The second logout then finds it already gone — that is a
	// successful logout, not an error.
	t.Run("already-revoked session is a successful logout", func(t *testing.T) {
		token := jwtlib.NewWithClaims(jwtlib.SigningMethodHS256, jwtlib.MapClaims{
			"sub": userUUID.String(),
			"sid": uuid.New().String(),
		})
		tokenStr, _ := token.SignedString([]byte("secret"))

		repo := &mockLogoutUserRepo{
			findBySubFn: func(string, string) (*User, error) { return &User{UserID: 42}, nil },
		}
		sess := &mockLogoutSessionService{
			revokeSessionFn: func(int64, uuid.UUID) error {
				return apperror.NewNotFound("session not found")
			},
		}
		svc := &loginService{userRepo: repo, sessionService: sess}
		require.NoError(t, svc.Logout(context.Background(), tokenStr))
	})

	// The subject is a user_identities.sub, not users.user_uuid. A federated
	// subject (Google, Cognito) is usually not a UUID at all — parsing it as one
	// made logout a silent no-op for every federated user.
	t.Run("federated non-uuid sub still resolves the user and revokes its session", func(t *testing.T) {
		sessionUUID := uuid.New()
		token := jwtlib.NewWithClaims(jwtlib.SigningMethodHS256, jwtlib.MapClaims{
			"sub":       "109876543210987654321", // a Google-shaped subject
			"client_id": "acme-spa",
			"sid":       sessionUUID.String(),
		})
		tokenStr, _ := token.SignedString([]byte("secret"))

		var gotSub, gotClient string
		repo := &mockLogoutUserRepo{
			findBySubFn: func(sub, clientID string) (*User, error) {
				gotSub, gotClient = sub, clientID
				return &User{UserID: 7}, nil
			},
		}

		var revoked uuid.UUID
		var revokedUser int64
		sess := &mockLogoutSessionService{
			revokeSessionFn: func(uid int64, sid uuid.UUID) error {
				revokedUser, revoked = uid, sid
				return nil
			},
			revokeAllSessionsFn: func(int64) error {
				t.Fatal("must revoke only this session, not every device")
				return nil
			},
		}

		svc := &loginService{userRepo: repo, sessionService: sess}
		require.NoError(t, svc.Logout(context.Background(), tokenStr))

		assert.Equal(t, "109876543210987654321", gotSub)
		assert.Equal(t, "acme-spa", gotClient)
		assert.Equal(t, int64(7), revokedUser)
		assert.Equal(t, sessionUUID, revoked)
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
