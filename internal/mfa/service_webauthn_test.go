package mfa

import (
	"context"
	"encoding/base64"
	"errors"
	"testing"
	"time"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/webauthn"
	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/maintainerd/maintainerd-auth/internal/platform/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestWebAuthnUserAdapter(t *testing.T) {
	creds := []webauthn.Credential{{ID: []byte("credential-id")}}
	user := &webAuthnUser{
		user:  &User{UserID: 42, Email: "user@example.com"},
		creds: creds,
	}

	assert.Equal(t, []byte("42"), user.WebAuthnID())
	assert.Equal(t, "user@example.com", user.WebAuthnName())
	assert.Equal(t, "user@example.com", user.WebAuthnDisplayName())
	assert.Equal(t, creds, user.WebAuthnCredentials())
}

func TestRPIDFromHostname(t *testing.T) {
	tests := []struct {
		name     string
		hostname string
		want     string
	}{
		{name: "https hostname", hostname: "https://auth.example.com", want: "auth.example.com"},
		{name: "http hostname", hostname: "http://localhost:8080", want: "localhost"},
		{name: "bare hostname", hostname: "auth.example.com", want: "auth.example.com"},
		{name: "bare hostname with port", hostname: "auth.example.com:8443", want: "auth.example.com"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, rpIDFromHostname(tt.hostname))
		})
	}
}

func TestNewWebAuthnService(t *testing.T) {
	original := config.AppPublicHostname
	t.Cleanup(func() { config.AppPublicHostname = original })
	db, _ := newMockGormDB(t)

	config.AppPublicHostname = "https://auth.example.com"
	got, err := NewWebAuthnService(db, &mockUserRepo{}, &mockMFAWebAuthnCredentialRepo{}, &mockWebAuthnSessionStore{}, &mockAuthEventService{}, nil)

	require.NoError(t, err)
	assert.IsType(t, &webAuthnService{}, got)

	config.AppPublicHostname = "not a url with spaces"
	got, err = NewWebAuthnService(db, &mockUserRepo{}, &mockMFAWebAuthnCredentialRepo{}, &mockWebAuthnSessionStore{}, &mockAuthEventService{}, nil)
	require.Error(t, err)
	assert.Nil(t, got)
}

func TestWebAuthnService_LoadWebAuthnUser(t *testing.T) {
	encodedID := base64.RawURLEncoding.EncodeToString([]byte("cred-id"))
	tests := []struct {
		name    string
		user    *User
		userErr error
		creds   []UserMFAWebAuthnCredential
		credErr error
		wantErr string
		wantLen int
	}{
		{name: "maps valid credentials and skips invalid ids", user: &User{UserID: mfaTestUserID, Email: "user@example.com"}, creds: []UserMFAWebAuthnCredential{{CredentialKeyID: encodedID, PublicKey: []byte("public"), SignCount: 9, IsBackupEligible: true}, {CredentialKeyID: "%"}}, wantLen: 1},
		{name: "missing user", wantErr: "user not found"},
		{name: "user repo error", userErr: errors.New("db down"), wantErr: "user not found"},
		{name: "credential lookup error", user: &User{UserID: mfaTestUserID}, credErr: errors.New("db down"), wantErr: "credential lookup failed"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &webAuthnService{
				userRepo:            &mockUserRepo{findByID: tt.user, findByIDErr: tt.userErr},
				mfaWebAuthnCredRepo: &mockMFAWebAuthnCredentialRepo{findByUserID: tt.creds, findByUserIDErr: tt.credErr},
			}

			got, err := svc.loadWebAuthnUser(mfaTestUserID)

			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
				return
			}
			require.NoError(t, err)
			assert.Len(t, got.creds, tt.wantLen)
		})
	}
}

func TestWebAuthnService_SessionHelpers(t *testing.T) {
	store := &mockWebAuthnSessionStore{}
	svc := &webAuthnService{sessionStore: store}
	session := &webauthn.SessionData{Challenge: "challenge", UserID: []byte("42")}

	assert.Equal(t, "webauthn:session:42:reg", svc.sessionKey(42, "reg"))
	require.NoError(t, svc.storeSession(t.Context(), 42, "reg", session))
	got, err := svc.loadSession(t.Context(), 42, "reg")
	require.NoError(t, err)
	assert.Equal(t, session.Challenge, got.Challenge)
	require.NoError(t, svc.deleteSession(t.Context(), 42, "reg"))
	_, err = svc.loadSession(t.Context(), 42, "reg")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "session expired")
}

func TestWebAuthnService_BeginCeremoniesErrorPaths(t *testing.T) {
	t.Run("begin registration load user error", func(t *testing.T) {
		svc := &webAuthnService{userRepo: &mockUserRepo{}, mfaWebAuthnCredRepo: &mockMFAWebAuthnCredentialRepo{}}
		_, err := svc.BeginRegistration(t.Context(), mfaTestUserID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "user not found")
	})

	t.Run("begin authentication load user error", func(t *testing.T) {
		svc := &webAuthnService{userRepo: &mockUserRepo{}, mfaWebAuthnCredRepo: &mockMFAWebAuthnCredentialRepo{}}
		_, err := svc.BeginAuthentication(t.Context(), mfaTestUserID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "user not found")
	})

	t.Run("finish registration load session error", func(t *testing.T) {
		svc := &webAuthnService{
			userRepo:            &mockUserRepo{findByID: &User{UserID: mfaTestUserID}},
			mfaWebAuthnCredRepo: &mockMFAWebAuthnCredentialRepo{},
			sessionStore:        &mockWebAuthnSessionStore{getErr: errors.New("missing")},
		}
		_, err := svc.FinishRegistration(t.Context(), mfaTestUserID, "", nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "session expired")
	})

	t.Run("finish registration load user error", func(t *testing.T) {
		svc := &webAuthnService{userRepo: &mockUserRepo{}, mfaWebAuthnCredRepo: &mockMFAWebAuthnCredentialRepo{}}
		_, err := svc.FinishRegistration(t.Context(), mfaTestUserID, "", nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "user not found")
	})

	t.Run("finish authentication load session error", func(t *testing.T) {
		svc := &webAuthnService{
			userRepo:            &mockUserRepo{findByID: &User{UserID: mfaTestUserID}},
			mfaWebAuthnCredRepo: &mockMFAWebAuthnCredentialRepo{},
			sessionStore:        &mockWebAuthnSessionStore{getErr: errors.New("missing")},
		}
		_, err := svc.FinishAuthentication(t.Context(), mfaTestUserID, nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "session expired")
	})

	t.Run("finish authentication load user error", func(t *testing.T) {
		svc := &webAuthnService{userRepo: &mockUserRepo{}, mfaWebAuthnCredRepo: &mockMFAWebAuthnCredentialRepo{}}
		_, err := svc.FinishAuthentication(t.Context(), mfaTestUserID, nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "user not found")
	})
}

func TestWebAuthnService_RegistrationAndAuthenticationCeremonies(t *testing.T) {
	originalBeginReg := beginWebAuthnRegistration
	originalCreate := createWebAuthnCredential
	originalBeginLogin := beginWebAuthnLogin
	originalValidate := validateWebAuthnLogin
	t.Cleanup(func() {
		beginWebAuthnRegistration = originalBeginReg
		createWebAuthnCredential = originalCreate
		beginWebAuthnLogin = originalBeginLogin
		validateWebAuthnLogin = originalValidate
	})

	t.Run("begin registration success and library error", func(t *testing.T) {
		beginWebAuthnRegistration = func(*webauthn.WebAuthn, webauthn.User) (*protocol.CredentialCreation, *webauthn.SessionData, error) {
			return &protocol.CredentialCreation{}, &webauthn.SessionData{Challenge: "challenge"}, nil
		}
		svc := &webAuthnService{
			userRepo:            &mockUserRepo{findByID: &User{UserID: mfaTestUserID}},
			mfaWebAuthnCredRepo: &mockMFAWebAuthnCredentialRepo{},
			sessionStore:        &mockWebAuthnSessionStore{},
		}
		got, err := svc.BeginRegistration(t.Context(), mfaTestUserID)
		require.NoError(t, err)
		assert.NotNil(t, got)

		beginWebAuthnRegistration = func(*webauthn.WebAuthn, webauthn.User) (*protocol.CredentialCreation, *webauthn.SessionData, error) {
			return nil, nil, errors.New("browser ceremony failed")
		}
		_, err = svc.BeginRegistration(t.Context(), mfaTestUserID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "registration initiation failed")

		beginWebAuthnRegistration = func(*webauthn.WebAuthn, webauthn.User) (*protocol.CredentialCreation, *webauthn.SessionData, error) {
			return &protocol.CredentialCreation{}, &webauthn.SessionData{Challenge: "challenge"}, nil
		}
		svc.sessionStore = &mockWebAuthnSessionStore{setErr: errors.New("cache down")}
		_, err = svc.BeginRegistration(t.Context(), mfaTestUserID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "cache down")
	})

	t.Run("begin authentication success and library error", func(t *testing.T) {
		beginWebAuthnLogin = func(*webauthn.WebAuthn, webauthn.User) (*protocol.CredentialAssertion, *webauthn.SessionData, error) {
			return &protocol.CredentialAssertion{}, &webauthn.SessionData{Challenge: "challenge"}, nil
		}
		svc := &webAuthnService{
			userRepo:            &mockUserRepo{findByID: &User{UserID: mfaTestUserID}},
			mfaWebAuthnCredRepo: &mockMFAWebAuthnCredentialRepo{},
			sessionStore:        &mockWebAuthnSessionStore{},
		}
		got, err := svc.BeginAuthentication(t.Context(), mfaTestUserID)
		require.NoError(t, err)
		assert.NotNil(t, got)

		beginWebAuthnLogin = func(*webauthn.WebAuthn, webauthn.User) (*protocol.CredentialAssertion, *webauthn.SessionData, error) {
			return nil, nil, errors.New("browser ceremony failed")
		}
		_, err = svc.BeginAuthentication(t.Context(), mfaTestUserID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "authentication initiation failed")

		beginWebAuthnLogin = func(*webauthn.WebAuthn, webauthn.User) (*protocol.CredentialAssertion, *webauthn.SessionData, error) {
			return &protocol.CredentialAssertion{}, &webauthn.SessionData{Challenge: "challenge"}, nil
		}
		svc.sessionStore = &mockWebAuthnSessionStore{setErr: errors.New("cache down")}
		_, err = svc.BeginAuthentication(t.Context(), mfaTestUserID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "cache down")
	})

	t.Run("finish registration success and persistence errors", func(t *testing.T) {
		createWebAuthnCredential = func(*webauthn.WebAuthn, webauthn.User, webauthn.SessionData, *protocol.ParsedCredentialCreationData) (*webauthn.Credential, error) {
			return &webauthn.Credential{
				ID:        []byte("cred-id"),
				PublicKey: []byte("public"),
				Authenticator: webauthn.Authenticator{
					SignCount: 7,
				},
				Transport: []protocol.AuthenticatorTransport{protocol.USB},
				Flags: webauthn.CredentialFlags{
					BackupEligible: true,
					BackupState:    true,
				},
			}, nil
		}
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		expectMFAUpdate(mock, "users").WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectCommit()
		events := &mockAuthEventService{}
		svc := &webAuthnService{
			db:                  db,
			userRepo:            &mockUserRepo{findByID: &User{UserID: mfaTestUserID}},
			mfaWebAuthnCredRepo: &mockMFAWebAuthnCredentialRepo{},
			sessionStore: &mockWebAuthnSessionStore{values: map[string]*webauthn.SessionData{
				"webauthn:session:42:reg": {Challenge: "challenge"},
			}},
			authEventService: events,
		}
		got, err := svc.FinishRegistration(t.Context(), mfaTestUserID, "", nil)
		require.NoError(t, err)
		assert.Equal(t, "Security Key", got.Name)
		assert.Equal(t, pq.StringArray{"usb"}, got.Transport)
		assert.Len(t, events.inputs, 1)
		assertExpectationsMet(t, mock)

		require.NoError(t, svc.storeSession(t.Context(), mfaTestUserID, "reg", &webauthn.SessionData{Challenge: "challenge"}))
		createWebAuthnCredential = func(*webauthn.WebAuthn, webauthn.User, webauthn.SessionData, *protocol.ParsedCredentialCreationData) (*webauthn.Credential, error) {
			return nil, errors.New("bad attestation")
		}
		_, err = svc.FinishRegistration(t.Context(), mfaTestUserID, "laptop", nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "WebAuthn registration failed")

		createWebAuthnCredential = func(*webauthn.WebAuthn, webauthn.User, webauthn.SessionData, *protocol.ParsedCredentialCreationData) (*webauthn.Credential, error) {
			return &webauthn.Credential{ID: []byte("cred-id"), PublicKey: []byte("public")}, nil
		}
		require.NoError(t, svc.storeSession(t.Context(), mfaTestUserID, "reg", &webauthn.SessionData{Challenge: "challenge"}))
		svc.mfaWebAuthnCredRepo = &mockMFAWebAuthnCredentialRepo{createErr: errors.New("db down")}
		_, err = svc.FinishRegistration(t.Context(), mfaTestUserID, "laptop", nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to persist")

		dbErr, mockErr := newMockGormDB(t)
		mockErr.ExpectBegin()
		expectMFAUpdate(mockErr, "users").WillReturnError(errors.New("db down"))
		mockErr.ExpectRollback()
		svc.db = dbErr
		svc.mfaWebAuthnCredRepo = &mockMFAWebAuthnCredentialRepo{}
		require.NoError(t, svc.storeSession(t.Context(), mfaTestUserID, "reg", &webauthn.SessionData{Challenge: "challenge"}))
		_, err = svc.FinishRegistration(t.Context(), mfaTestUserID, "laptop", nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to update user WebAuthn state")
		assertExpectationsMet(t, mockErr)

		dbOK, mockOK := newMockGormDB(t)
		mockOK.ExpectBegin()
		expectMFAUpdate(mockOK, "users").WillReturnResult(sqlmock.NewResult(0, 1))
		mockOK.ExpectCommit()
		svc.db = dbOK
		svc.sessionStore = &mockWebAuthnSessionStore{values: map[string]*webauthn.SessionData{
			"webauthn:session:42:reg": {Challenge: "challenge"},
		}, delErr: errors.New("cache down")}
		_, err = svc.FinishRegistration(t.Context(), mfaTestUserID, "laptop", nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to clear WebAuthn registration session")
		assertExpectationsMet(t, mockOK)
	})

	t.Run("finish authentication success and validation errors", func(t *testing.T) {
		validateWebAuthnLogin = func(*webauthn.WebAuthn, webauthn.User, webauthn.SessionData, *protocol.ParsedCredentialAssertionData) (*webauthn.Credential, error) {
			return &webauthn.Credential{ID: []byte("cred-id"), Authenticator: webauthn.Authenticator{SignCount: 8}}, nil
		}
		stored := &UserMFAWebAuthnCredential{CredentialID: 1, CredentialUUID: mfaTestCredentialUUID, CredentialKeyID: base64.RawURLEncoding.EncodeToString([]byte("cred-id")), SignCount: 7}
		events := &mockAuthEventService{}
		svc := &webAuthnService{
			userRepo:            &mockUserRepo{findByID: &User{UserID: mfaTestUserID, TenantID: mfaTestTenantID}},
			mfaWebAuthnCredRepo: &mockMFAWebAuthnCredentialRepo{findByKeyID: stored},
			sessionStore: &mockWebAuthnSessionStore{values: map[string]*webauthn.SessionData{
				"webauthn:session:42:auth": {Challenge: "challenge"},
			}},
			authEventService: events,
		}
		got, err := svc.FinishAuthentication(t.Context(), mfaTestUserID, nil)
		require.NoError(t, err)
		assert.Equal(t, stored, got)
		assert.Len(t, events.inputs, 1)
		assert.Equal(t, mfaTestTenantID, events.inputs[0].TenantID)

		require.NoError(t, svc.storeSession(t.Context(), mfaTestUserID, "auth", &webauthn.SessionData{Challenge: "challenge"}))
		validateWebAuthnLogin = func(*webauthn.WebAuthn, webauthn.User, webauthn.SessionData, *protocol.ParsedCredentialAssertionData) (*webauthn.Credential, error) {
			return nil, errors.New("bad assertion")
		}
		_, err = svc.FinishAuthentication(t.Context(), mfaTestUserID, nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "WebAuthn authentication failed")

		validateWebAuthnLogin = func(*webauthn.WebAuthn, webauthn.User, webauthn.SessionData, *protocol.ParsedCredentialAssertionData) (*webauthn.Credential, error) {
			return &webauthn.Credential{ID: []byte("cred-id"), Authenticator: webauthn.Authenticator{SignCount: 8}}, nil
		}
		require.NoError(t, svc.storeSession(t.Context(), mfaTestUserID, "auth", &webauthn.SessionData{Challenge: "challenge"}))
		svc.mfaWebAuthnCredRepo = &mockMFAWebAuthnCredentialRepo{findByKeyIDErr: errors.New("db down")}
		_, err = svc.FinishAuthentication(t.Context(), mfaTestUserID, nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "credential not found after validation")

		require.NoError(t, svc.storeSession(t.Context(), mfaTestUserID, "auth", &webauthn.SessionData{Challenge: "challenge"}))
		svc.mfaWebAuthnCredRepo = &mockMFAWebAuthnCredentialRepo{findByKeyID: &UserMFAWebAuthnCredential{CredentialID: 1, SignCount: 9}}
		_, err = svc.FinishAuthentication(t.Context(), mfaTestUserID, nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "sign count regression")

		require.NoError(t, svc.storeSession(t.Context(), mfaTestUserID, "auth", &webauthn.SessionData{Challenge: "challenge"}))
		svc.mfaWebAuthnCredRepo = &mockMFAWebAuthnCredentialRepo{findByKeyID: stored, signCountErr: errors.New("db down")}
		_, err = svc.FinishAuthentication(t.Context(), mfaTestUserID, nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to update WebAuthn sign count")

		require.NoError(t, svc.storeSession(t.Context(), mfaTestUserID, "auth", &webauthn.SessionData{Challenge: "challenge"}))
		svc.mfaWebAuthnCredRepo = &mockMFAWebAuthnCredentialRepo{findByKeyID: stored, lastUsedErr: errors.New("db down")}
		_, err = svc.FinishAuthentication(t.Context(), mfaTestUserID, nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to update WebAuthn last-used")

		svc.sessionStore = &mockWebAuthnSessionStore{values: map[string]*webauthn.SessionData{
			"webauthn:session:42:auth": {Challenge: "challenge"},
		}, delErr: errors.New("cache down")}
		svc.mfaWebAuthnCredRepo = &mockMFAWebAuthnCredentialRepo{findByKeyID: stored}
		_, err = svc.FinishAuthentication(t.Context(), mfaTestUserID, nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to clear WebAuthn authentication session")
	})
}

func TestWebAuthnService_DeleteCredential(t *testing.T) {
	credUUID := uuid.MustParse("00000000-0000-0000-0000-000000000777")

	t.Run("success leaves WebAuthn enabled when remaining credentials exist", func(t *testing.T) {
		repo := &mockMFAWebAuthnCredentialRepo{
			findByUserID: []UserMFAWebAuthnCredential{{CredentialID: 1, CredentialUUID: credUUID}},
		}
		svc := &webAuthnService{mfaWebAuthnCredRepo: repo}

		require.NoError(t, svc.DeleteCredential(t.Context(), credUUID.String(), mfaTestUserID))
	})

	t.Run("success disables WebAuthn when no credentials remain", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		expectMFAUpdate(mock, "users").WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectCommit()
		repo := &sequencedWebAuthnCredentialRepo{
			sequences: [][]UserMFAWebAuthnCredential{{{CredentialID: 1, CredentialUUID: credUUID}}, {}},
		}
		svc := &webAuthnService{db: db, mfaWebAuthnCredRepo: repo}

		require.NoError(t, svc.DeleteCredential(t.Context(), credUUID.String(), mfaTestUserID))
		assertExpectationsMet(t, mock)
	})

	tests := []struct {
		name    string
		repo    UserMFAWebAuthnCredentialRepository
		dbErr   bool
		wantErr string
	}{
		{name: "lookup error", repo: &mockMFAWebAuthnCredentialRepo{findByUserIDErr: errors.New("db down")}, wantErr: "credential lookup failed"},
		{name: "not found", repo: &mockMFAWebAuthnCredentialRepo{findByUserID: []UserMFAWebAuthnCredential{}}, wantErr: "credential not found"},
		{name: "delete error", repo: &mockMFAWebAuthnCredentialRepo{findByUserID: []UserMFAWebAuthnCredential{{CredentialID: 1, CredentialUUID: credUUID}}, deleteErr: errors.New("db down")}, wantErr: "failed to delete credential"},
		{name: "remaining lookup error", repo: &sequencedWebAuthnCredentialRepo{sequences: [][]UserMFAWebAuthnCredential{{{CredentialID: 1, CredentialUUID: credUUID}}}, errAt: 2}, wantErr: "failed to list remaining"},
		{name: "user update error", repo: &sequencedWebAuthnCredentialRepo{sequences: [][]UserMFAWebAuthnCredential{{{CredentialID: 1, CredentialUUID: credUUID}}, {}}}, dbErr: true, wantErr: "failed to update user WebAuthn state"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock := newMockGormDB(t)
			if tt.dbErr {
				mock.ExpectBegin()
				expectMFAUpdate(mock, "users").WillReturnError(errors.New("db down"))
				mock.ExpectRollback()
			}
			svc := &webAuthnService{db: db, mfaWebAuthnCredRepo: tt.repo}

			err := svc.DeleteCredential(t.Context(), credUUID.String(), mfaTestUserID)

			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.wantErr)
			assertExpectationsMet(t, mock)
		})
	}
}

type mockWebAuthnSessionStore struct {
	values map[string]*webauthn.SessionData
	setErr error
	getErr error
	delErr error
}

func (m *mockWebAuthnSessionStore) SetSession(_ context.Context, key string, value any, _ time.Duration) error {
	if m.setErr != nil {
		return m.setErr
	}
	if m.values == nil {
		m.values = map[string]*webauthn.SessionData{}
	}
	m.values[key] = value.(*webauthn.SessionData)
	return nil
}

func (m *mockWebAuthnSessionStore) GetSession(_ context.Context, key string, dest any) error {
	if m.getErr != nil {
		return m.getErr
	}
	value, ok := m.values[key]
	if !ok {
		return errors.New("missing")
	}
	*(dest.(*webauthn.SessionData)) = *value
	return nil
}

func (m *mockWebAuthnSessionStore) DeleteSession(_ context.Context, key string) error {
	if m.delErr != nil {
		return m.delErr
	}
	delete(m.values, key)
	return nil
}

type sequencedWebAuthnCredentialRepo struct {
	mockBaseRepositoryMethods[UserMFAWebAuthnCredential]
	sequences [][]UserMFAWebAuthnCredential
	calls     int
	errAt     int
}

func (m *sequencedWebAuthnCredentialRepo) WithTx(*gorm.DB) UserMFAWebAuthnCredentialRepository {
	return m
}

func (m *sequencedWebAuthnCredentialRepo) FindByUserID(int64) ([]UserMFAWebAuthnCredential, error) {
	m.calls++
	if m.errAt == m.calls {
		return nil, errors.New("db down")
	}
	if len(m.sequences) == 0 {
		return nil, nil
	}
	result := m.sequences[0]
	m.sequences = m.sequences[1:]
	return result, nil
}

func (m *sequencedWebAuthnCredentialRepo) FindByCredentialKeyID(string) (*UserMFAWebAuthnCredential, error) {
	return nil, nil
}
func (m *sequencedWebAuthnCredentialRepo) CreateCredential(*UserMFAWebAuthnCredential) error {
	return nil
}
func (m *sequencedWebAuthnCredentialRepo) UpdateSignCount(int64, int64) error { return nil }
func (m *sequencedWebAuthnCredentialRepo) UpdateLastUsed(int64) error         { return nil }
func (m *sequencedWebAuthnCredentialRepo) DeleteCredentialByID(int64, int64) error {
	return nil
}
func (m *sequencedWebAuthnCredentialRepo) DeleteAllByUserID(int64) error { return nil }
