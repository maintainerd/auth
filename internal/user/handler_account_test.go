package user

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/authctx"
	"github.com/maintainerd/auth/internal/platform/middleware"
	"github.com/stretchr/testify/assert"
)

type mockAccountService struct {
	initiateEmailChangeFn func(userID int64, newEmail, currentPassword string) error
	verifyEmailChangeFn   func(userID int64, otp string) error
	changeUsernameFn      func(userID int64, newUsername, currentPassword string) error
	deleteAccountFn       func(userID int64, currentPassword string) error
	exportAccountDataFn   func(userID int64) (*AccountExportDTO, error)
	generateBackupCodesFn func(userID int64) (*GenerateBackupCodesResponseDTO, error)
	verifyBackupCodeFn    func(req VerifyBackupCodeDTO) (*LoginResponseDTO, error)
}

func (m *mockAccountService) InitiateEmailChange(_ context.Context, userID int64, newEmail, currentPassword string) error {
	if m.initiateEmailChangeFn != nil {
		return m.initiateEmailChangeFn(userID, newEmail, currentPassword)
	}
	return nil
}
func (m *mockAccountService) VerifyEmailChange(_ context.Context, userID int64, otp string) error {
	if m.verifyEmailChangeFn != nil {
		return m.verifyEmailChangeFn(userID, otp)
	}
	return nil
}
func (m *mockAccountService) ChangeUsername(_ context.Context, userID int64, newUsername, currentPassword string) error {
	if m.changeUsernameFn != nil {
		return m.changeUsernameFn(userID, newUsername, currentPassword)
	}
	return nil
}
func (m *mockAccountService) DeleteAccount(_ context.Context, userID int64, currentPassword string) error {
	if m.deleteAccountFn != nil {
		return m.deleteAccountFn(userID, currentPassword)
	}
	return nil
}
func (m *mockAccountService) ExportAccountData(_ context.Context, userID int64) (*AccountExportDTO, error) {
	if m.exportAccountDataFn != nil {
		return m.exportAccountDataFn(userID)
	}
	return &AccountExportDTO{}, nil
}
func (m *mockAccountService) GenerateBackupCodes(_ context.Context, userID int64) (*GenerateBackupCodesResponseDTO, error) {
	if m.generateBackupCodesFn != nil {
		return m.generateBackupCodesFn(userID)
	}
	return &GenerateBackupCodesResponseDTO{Codes: []string{"code1"}}, nil
}
func (m *mockAccountService) VerifyBackupCode(_ context.Context, req VerifyBackupCodeDTO) (*LoginResponseDTO, error) {
	if m.verifyBackupCodeFn != nil {
		return m.verifyBackupCodeFn(req)
	}
	return &LoginResponseDTO{AccessToken: "at"}, nil
}

type mockSessionService struct {
	listSessionsFn      func(userID int64) ([]*SessionDataResult, error)
	revokeSessionFn     func(userID int64, sessionUUID uuid.UUID) error
	revokeAllSessionsFn func(userID int64) error
	createSessionFn     func(userID int64, ipAddress, userAgent string) (*UserToken, error)
	enforceConcurrentFn func(userUUID uuid.UUID, userID int64) error
	validateAndTouchFn  func(sessionUUID uuid.UUID, userID int64) error
}

func (m *mockSessionService) ListSessions(_ context.Context, userID int64) ([]*SessionDataResult, error) {
	if m.listSessionsFn != nil {
		return m.listSessionsFn(userID)
	}
	return nil, nil
}
func (m *mockSessionService) RevokeSession(_ context.Context, userID int64, sessionUUID uuid.UUID) error {
	if m.revokeSessionFn != nil {
		return m.revokeSessionFn(userID, sessionUUID)
	}
	return nil
}
func (m *mockSessionService) RevokeAllSessions(_ context.Context, userID int64) error {
	if m.revokeAllSessionsFn != nil {
		return m.revokeAllSessionsFn(userID)
	}
	return nil
}
func (m *mockSessionService) CreateSession(_ context.Context, userID int64, ipAddress, userAgent string) (*UserToken, error) {
	if m.createSessionFn != nil {
		return m.createSessionFn(userID, ipAddress, userAgent)
	}
	return nil, nil
}
func (m *mockSessionService) EnforceConcurrentLimit(_ context.Context, userUUID uuid.UUID, userID int64) error {
	if m.enforceConcurrentFn != nil {
		return m.enforceConcurrentFn(userUUID, userID)
	}
	return nil
}
func (m *mockSessionService) ValidateAndTouch(_ context.Context, sessionUUID uuid.UUID, userID int64) error {
	if m.validateAndTouchFn != nil {
		return m.validateAndTouchFn(sessionUUID, userID)
	}
	return nil
}

func withAuthUser(r *http.Request) *http.Request {
	return middleware.WithAuthContext(r, &authctx.AuthContext{
		User: &authctx.AuthUser{UserUUID: testUserUUID, UserID: 42},
	})
}

func TestAccountHandler_InitiateEmailChange(t *testing.T) {
	t.Run("no user returns 401", func(t *testing.T) {
		r := jsonReq(t, http.MethodPost, "/account/email/change", map[string]string{"new_email": "new@example.com", "current_password": "pass"})
		w := httptest.NewRecorder()
		NewAccountHandler(&mockAccountService{}, &mockSessionService{}, nil).InitiateEmailChange(w, r)
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("invalid JSON returns 400", func(t *testing.T) {
		r := withAuthUser(badJSONReq(t, http.MethodPost, "/account/email/change"))
		w := httptest.NewRecorder()
		NewAccountHandler(&mockAccountService{}, &mockSessionService{}, nil).InitiateEmailChange(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("validation error returns 400", func(t *testing.T) {
		r := withAuthUser(jsonReq(t, http.MethodPost, "/account/email/change", map[string]string{"new_email": "", "current_password": ""}))
		w := httptest.NewRecorder()
		NewAccountHandler(&mockAccountService{}, &mockSessionService{}, nil).InitiateEmailChange(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("service error returns 500", func(t *testing.T) {
		svc := &mockAccountService{
			initiateEmailChangeFn: func(int64, string, string) error { return errors.New("db error") },
		}
		r := withAuthUser(jsonReq(t, http.MethodPost, "/account/email/change", map[string]string{"new_email": "new@example.com", "current_password": "pass"}))
		w := httptest.NewRecorder()
		NewAccountHandler(svc, &mockSessionService{}, nil).InitiateEmailChange(w, r)
		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})

	t.Run("success returns 200", func(t *testing.T) {
		r := withAuthUser(jsonReq(t, http.MethodPost, "/account/email/change", map[string]string{"new_email": "new@example.com", "current_password": "pass"}))
		w := httptest.NewRecorder()
		NewAccountHandler(&mockAccountService{}, &mockSessionService{}, nil).InitiateEmailChange(w, r)
		assert.Equal(t, http.StatusOK, w.Code)
	})
}

func TestAccountHandler_VerifyEmailChange(t *testing.T) {
	t.Run("no user returns 401", func(t *testing.T) {
		r := jsonReq(t, http.MethodPost, "/account/email/verify", map[string]string{"otp": "123456"})
		w := httptest.NewRecorder()
		NewAccountHandler(&mockAccountService{}, &mockSessionService{}, nil).VerifyEmailChange(w, r)
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("invalid JSON returns 400", func(t *testing.T) {
		r := withAuthUser(badJSONReq(t, http.MethodPost, "/account/email/verify"))
		w := httptest.NewRecorder()
		NewAccountHandler(&mockAccountService{}, &mockSessionService{}, nil).VerifyEmailChange(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("validation error returns 400", func(t *testing.T) {
		r := withAuthUser(jsonReq(t, http.MethodPost, "/account/email/verify", map[string]string{"otp": ""}))
		w := httptest.NewRecorder()
		NewAccountHandler(&mockAccountService{}, &mockSessionService{}, nil).VerifyEmailChange(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("service error returns 500", func(t *testing.T) {
		svc := &mockAccountService{
			verifyEmailChangeFn: func(int64, string) error { return errors.New("invalid otp") },
		}
		r := withAuthUser(jsonReq(t, http.MethodPost, "/account/email/verify", map[string]string{"otp": "123456"}))
		w := httptest.NewRecorder()
		NewAccountHandler(svc, &mockSessionService{}, nil).VerifyEmailChange(w, r)
		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})

	t.Run("success returns 200", func(t *testing.T) {
		r := withAuthUser(jsonReq(t, http.MethodPost, "/account/email/verify", map[string]string{"otp": "123456"}))
		w := httptest.NewRecorder()
		NewAccountHandler(&mockAccountService{}, &mockSessionService{}, nil).VerifyEmailChange(w, r)
		assert.Equal(t, http.StatusOK, w.Code)
	})
}

func TestAccountHandler_ChangeUsername(t *testing.T) {
	t.Run("no user returns 401", func(t *testing.T) {
		r := jsonReq(t, http.MethodPost, "/account/username", map[string]string{"new_username": "newuser", "current_password": "pass"})
		w := httptest.NewRecorder()
		NewAccountHandler(&mockAccountService{}, &mockSessionService{}, nil).ChangeUsername(w, r)
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("invalid JSON returns 400", func(t *testing.T) {
		r := withAuthUser(badJSONReq(t, http.MethodPut, "/account/username"))
		w := httptest.NewRecorder()
		NewAccountHandler(&mockAccountService{}, &mockSessionService{}, nil).ChangeUsername(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("validation error returns 400", func(t *testing.T) {
		r := withAuthUser(jsonReq(t, http.MethodPut, "/account/username", map[string]string{"new_username": "", "current_password": ""}))
		w := httptest.NewRecorder()
		NewAccountHandler(&mockAccountService{}, &mockSessionService{}, nil).ChangeUsername(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("service error returns 500", func(t *testing.T) {
		svc := &mockAccountService{
			changeUsernameFn: func(int64, string, string) error { return errors.New("db error") },
		}
		r := withAuthUser(jsonReq(t, http.MethodPut, "/account/username", map[string]string{"new_username": "newuser", "current_password": "pass"}))
		w := httptest.NewRecorder()
		NewAccountHandler(svc, &mockSessionService{}, nil).ChangeUsername(w, r)
		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})

	t.Run("success returns 200", func(t *testing.T) {
		r := withAuthUser(jsonReq(t, http.MethodPut, "/account/username", map[string]string{"new_username": "newuser", "current_password": "pass"}))
		w := httptest.NewRecorder()
		NewAccountHandler(&mockAccountService{}, &mockSessionService{}, nil).ChangeUsername(w, r)
		assert.Equal(t, http.StatusOK, w.Code)
	})
}

func TestAccountHandler_DeleteAccount(t *testing.T) {
	t.Run("no user returns 401", func(t *testing.T) {
		r := jsonReq(t, http.MethodPost, "/account", map[string]string{"current_password": "pass"})
		w := httptest.NewRecorder()
		NewAccountHandler(&mockAccountService{}, &mockSessionService{}, nil).DeleteAccount(w, r)
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("invalid JSON returns 400", func(t *testing.T) {
		r := withAuthUser(badJSONReq(t, http.MethodDelete, "/account"))
		w := httptest.NewRecorder()
		NewAccountHandler(&mockAccountService{}, &mockSessionService{}, nil).DeleteAccount(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("validation error returns 400", func(t *testing.T) {
		r := withAuthUser(jsonReq(t, http.MethodDelete, "/account", map[string]string{"current_password": ""}))
		w := httptest.NewRecorder()
		NewAccountHandler(&mockAccountService{}, &mockSessionService{}, nil).DeleteAccount(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("service error returns 500", func(t *testing.T) {
		svc := &mockAccountService{
			deleteAccountFn: func(int64, string) error { return errors.New("db error") },
		}
		r := withAuthUser(jsonReq(t, http.MethodDelete, "/account", map[string]string{"current_password": "pass"}))
		w := httptest.NewRecorder()
		NewAccountHandler(svc, &mockSessionService{}, nil).DeleteAccount(w, r)
		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})

	t.Run("success returns 200", func(t *testing.T) {
		r := withAuthUser(jsonReq(t, http.MethodDelete, "/account", map[string]string{"current_password": "pass"}))
		w := httptest.NewRecorder()
		NewAccountHandler(&mockAccountService{}, &mockSessionService{}, nil).DeleteAccount(w, r)
		assert.Equal(t, http.StatusOK, w.Code)
	})
}

func TestAccountHandler_ExportAccountData(t *testing.T) {
	t.Run("no user returns 401", func(t *testing.T) {
		r := httptest.NewRequest(http.MethodGet, "/account/export", nil)
		w := httptest.NewRecorder()
		NewAccountHandler(&mockAccountService{}, &mockSessionService{}, nil).ExportAccountData(w, r)
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("service error returns 500", func(t *testing.T) {
		svc := &mockAccountService{
			exportAccountDataFn: func(int64) (*AccountExportDTO, error) { return nil, errors.New("db error") },
		}
		r := withAuthUser(httptest.NewRequest(http.MethodGet, "/account/export", nil))
		w := httptest.NewRecorder()
		NewAccountHandler(svc, &mockSessionService{}, nil).ExportAccountData(w, r)
		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})

	t.Run("success returns 200", func(t *testing.T) {
		svc := &mockAccountService{
			exportAccountDataFn: func(int64) (*AccountExportDTO, error) {
				return &AccountExportDTO{Username: "alice"}, nil
			},
		}
		r := withAuthUser(httptest.NewRequest(http.MethodGet, "/account/export", nil))
		w := httptest.NewRecorder()
		NewAccountHandler(svc, &mockSessionService{}, nil).ExportAccountData(w, r)
		assert.Equal(t, http.StatusOK, w.Code)
	})
}

func TestAccountHandler_GenerateBackupCodes(t *testing.T) {
	t.Run("no user returns 401", func(t *testing.T) {
		r := httptest.NewRequest(http.MethodPost, "/account/backup-codes", nil)
		w := httptest.NewRecorder()
		NewAccountHandler(&mockAccountService{}, &mockSessionService{}, nil).GenerateBackupCodes(w, r)
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("service error returns 500", func(t *testing.T) {
		svc := &mockAccountService{
			generateBackupCodesFn: func(int64) (*GenerateBackupCodesResponseDTO, error) {
				return nil, errors.New("db error")
			},
		}
		r := withAuthUser(httptest.NewRequest(http.MethodPost, "/account/backup-codes", nil))
		w := httptest.NewRecorder()
		NewAccountHandler(svc, &mockSessionService{}, nil).GenerateBackupCodes(w, r)
		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})

	t.Run("success returns 200", func(t *testing.T) {
		r := withAuthUser(httptest.NewRequest(http.MethodPost, "/account/backup-codes", nil))
		w := httptest.NewRecorder()
		NewAccountHandler(&mockAccountService{}, &mockSessionService{}, nil).GenerateBackupCodes(w, r)
		assert.Equal(t, http.StatusOK, w.Code)
	})
}

func TestAccountHandler_VerifyBackupCode(t *testing.T) {
	t.Run("invalid JSON returns 400", func(t *testing.T) {
		r := badJSONReq(t, http.MethodPost, "/recovery/backup-code")
		w := httptest.NewRecorder()
		NewAccountHandler(&mockAccountService{}, &mockSessionService{}, nil).VerifyBackupCode(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("validation error returns 400", func(t *testing.T) {
		r := jsonReq(t, http.MethodPost, "/recovery/backup-code", map[string]string{})
		w := httptest.NewRecorder()
		NewAccountHandler(&mockAccountService{}, &mockSessionService{}, nil).VerifyBackupCode(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("service error returns 500", func(t *testing.T) {
		svc := &mockAccountService{
			verifyBackupCodeFn: func(req VerifyBackupCodeDTO) (*LoginResponseDTO, error) {
				return nil, errors.New("invalid code")
			},
		}
		r := jsonReq(t, http.MethodPost, "/recovery/backup-code", map[string]string{
			"email": "user@example.com", "code": "abc12345", "client_id": "app", "provider_id": "idp",
		})
		w := httptest.NewRecorder()
		NewAccountHandler(svc, &mockSessionService{}, nil).VerifyBackupCode(w, r)
		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})

	t.Run("success returns 200", func(t *testing.T) {
		svc := &mockAccountService{
			verifyBackupCodeFn: func(req VerifyBackupCodeDTO) (*LoginResponseDTO, error) {
				return &LoginResponseDTO{AccessToken: "at"}, nil
			},
		}
		r := jsonReq(t, http.MethodPost, "/recovery/backup-code", map[string]string{
			"email": "user@example.com", "code": "abc12345", "client_id": "app", "provider_id": "idp",
		})
		w := httptest.NewRecorder()
		NewAccountHandler(svc, &mockSessionService{}, nil).VerifyBackupCode(w, r)
		assert.Equal(t, http.StatusOK, w.Code)
	})
}

func TestAccountHandler_ListSessions(t *testing.T) {
	t.Run("no user returns 401", func(t *testing.T) {
		r := httptest.NewRequest(http.MethodGet, "/account/sessions", nil)
		w := httptest.NewRecorder()
		NewAccountHandler(&mockAccountService{}, &mockSessionService{}, nil).ListSessions(w, r)
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("service error returns 500", func(t *testing.T) {
		sessSvc := &mockSessionService{
			listSessionsFn: func(int64) ([]*SessionDataResult, error) {
				return nil, errors.New("db error")
			},
		}
		r := withAuthUser(httptest.NewRequest(http.MethodGet, "/account/sessions", nil))
		w := httptest.NewRecorder()
		NewAccountHandler(&mockAccountService{}, sessSvc, nil).ListSessions(w, r)
		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})

	t.Run("success returns 200", func(t *testing.T) {
		now := time.Now()
		sessSvc := &mockSessionService{
			listSessionsFn: func(int64) ([]*SessionDataResult, error) {
				return []*SessionDataResult{
					{SessionID: uuid.New().String(), CreatedAt: now},
				}, nil
			},
		}
		r := withAuthUser(httptest.NewRequest(http.MethodGet, "/account/sessions", nil))
		w := httptest.NewRecorder()
		NewAccountHandler(&mockAccountService{}, sessSvc, nil).ListSessions(w, r)
		assert.Equal(t, http.StatusOK, w.Code)
	})
}

func TestAccountHandler_RevokeSession(t *testing.T) {
	sessionUUID := uuid.New()

	t.Run("no user returns 401", func(t *testing.T) {
		r := withChiParam(httptest.NewRequest(http.MethodDelete, "/", nil), "session_uuid", sessionUUID.String())
		w := httptest.NewRecorder()
		NewAccountHandler(&mockAccountService{}, &mockSessionService{}, nil).RevokeSession(w, r)
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("invalid UUID returns 400", func(t *testing.T) {
		r := withAuthUser(withChiParam(httptest.NewRequest(http.MethodDelete, "/", nil), "session_uuid", "bad"))
		w := httptest.NewRecorder()
		NewAccountHandler(&mockAccountService{}, &mockSessionService{}, nil).RevokeSession(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("service error returns 500", func(t *testing.T) {
		sessSvc := &mockSessionService{
			revokeSessionFn: func(int64, uuid.UUID) error { return errors.New("db error") },
		}
		r := withAuthUser(withChiParam(httptest.NewRequest(http.MethodDelete, "/", nil), "session_uuid", sessionUUID.String()))
		w := httptest.NewRecorder()
		NewAccountHandler(&mockAccountService{}, sessSvc, nil).RevokeSession(w, r)
		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})

	t.Run("success returns 200", func(t *testing.T) {
		r := withAuthUser(withChiParam(httptest.NewRequest(http.MethodDelete, "/", nil), "session_uuid", sessionUUID.String()))
		w := httptest.NewRecorder()
		NewAccountHandler(&mockAccountService{}, &mockSessionService{}, nil).RevokeSession(w, r)
		assert.Equal(t, http.StatusOK, w.Code)
	})
}

func TestAccountHandler_RevokeAllSessions(t *testing.T) {
	t.Run("no user returns 401", func(t *testing.T) {
		r := httptest.NewRequest(http.MethodDelete, "/account/sessions", nil)
		w := httptest.NewRecorder()
		NewAccountHandler(&mockAccountService{}, &mockSessionService{}, nil).RevokeAllSessions(w, r)
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("service error returns 500", func(t *testing.T) {
		sessSvc := &mockSessionService{
			revokeAllSessionsFn: func(int64) error { return errors.New("db error") },
		}
		r := withAuthUser(httptest.NewRequest(http.MethodDelete, "/account/sessions", nil))
		w := httptest.NewRecorder()
		NewAccountHandler(&mockAccountService{}, sessSvc, nil).RevokeAllSessions(w, r)
		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})

	t.Run("success returns 200", func(t *testing.T) {
		r := withAuthUser(httptest.NewRequest(http.MethodDelete, "/account/sessions", nil))
		w := httptest.NewRecorder()
		NewAccountHandler(&mockAccountService{}, &mockSessionService{}, nil).RevokeAllSessions(w, r)
		assert.Equal(t, http.StatusOK, w.Code)
	})
}
