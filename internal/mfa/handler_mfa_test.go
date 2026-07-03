package mfa

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-webauthn/webauthn/protocol"
	"github.com/google/uuid"
	"github.com/maintainerd/maintainerd-auth/internal/authctx"
	"github.com/maintainerd/maintainerd-auth/internal/platform/apperror"
	"github.com/maintainerd/maintainerd-auth/internal/platform/jwt"
	"github.com/maintainerd/maintainerd-auth/internal/platform/middleware"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const mfaTestUserID int64 = 42

var (
	mfaTestUserUUID       = uuid.MustParse("00000000-0000-0000-0000-000000000042")
	mfaTestCredentialUUID = uuid.MustParse("00000000-0000-0000-0000-000000000099")
	mfaTestTenantID       = int64(1)
	mfaTestTenantUUID     = uuid.MustParse("00000000-0000-0000-0000-000000000001")
)

type mockMFAService struct {
	beginTOTPEnrollmentFn   func(context.Context, int64) (*TOTPEnrollResponseDTO, error)
	finishTOTPEnrollmentFn  func(context.Context, int64, string) ([]string, error)
	verifyTOTPFn            func(context.Context, int64, string) (bool, error)
	disableTOTPFn           func(context.Context, int64) error
	getBackupCodesCountFn   func(context.Context, int64) (int, error)
	regenerateBackupCodesFn func(context.Context, int64) ([]string, error)
	getMFAStatusFn          func(context.Context, int64) (*MFAStatusResponseDTO, error)
	getMFAPolicyFn          func(context.Context, int64) (*MFAPolicyDTO, error)
	isMFARequiredFn         func(context.Context, int64) (bool, error)
	isMethodAllowedFn       func(context.Context, int64, string) (bool, error)
	userHasMFAFn            func(context.Context, int64) (bool, error)

	sensitiveActionStepUpRequiredFn func(context.Context, int64) (bool, error)
	syncMFAStateFn                  func() error
	adminResetMFAFn                 func(context.Context, string, int64) error
	adminResetMFAMethodFn           func(context.Context, string, string, int64) error
	selfResetMFAFn                  func(context.Context, int64) error
	issueStepUpChallengeFn          func(context.Context, string, []string) (*StepUpChallengeResponseDTO, error)
	verifyStepUpFn                  func(context.Context, StepUpVerifyRequestDTO, int64) (*StepUpVerifyResponseDTO, error)
	sendStepUpSMSFn                 func(context.Context, int64) error
	enrollSMSFn                     func(context.Context, int64, string) error
	verifySMSFn                     func(context.Context, int64, string, string) error
	disableSMSFn                    func(context.Context, int64) error
	sendStepUpEmailOTPFn            func(context.Context, int64) error
	enrollEmailOTPFn                func(context.Context, int64, string) error
	verifyEmailOTPFn                func(context.Context, int64, string, string) error
	disableEmailOTPFn               func(context.Context, int64) error
	sendEmailOTPChallengeFn         func(context.Context, int64) error
}

func (m *mockMFAService) BeginTOTPEnrollment(ctx context.Context, userID int64) (*TOTPEnrollResponseDTO, error) {
	if m.beginTOTPEnrollmentFn != nil {
		return m.beginTOTPEnrollmentFn(ctx, userID)
	}
	return &TOTPEnrollResponseDTO{Secret: "secret", QRCodeURL: "otpauth://totp/test"}, nil
}

func (m *mockMFAService) FinishTOTPEnrollment(ctx context.Context, userID int64, code string) ([]string, error) {
	if m.finishTOTPEnrollmentFn != nil {
		return m.finishTOTPEnrollmentFn(ctx, userID, code)
	}
	return []string{"backup-1"}, nil
}

func (m *mockMFAService) VerifyTOTP(ctx context.Context, userID int64, code string) (bool, error) {
	if m.verifyTOTPFn != nil {
		return m.verifyTOTPFn(ctx, userID, code)
	}
	return true, nil
}

func (m *mockMFAService) DisableTOTP(ctx context.Context, userID int64) error {
	if m.disableTOTPFn != nil {
		return m.disableTOTPFn(ctx, userID)
	}
	return nil
}

func (m *mockMFAService) GetBackupCodesCount(ctx context.Context, userID int64) (int, error) {
	if m.getBackupCodesCountFn != nil {
		return m.getBackupCodesCountFn(ctx, userID)
	}
	return 3, nil
}

func (m *mockMFAService) RegenerateBackupCodes(ctx context.Context, userID int64) ([]string, error) {
	if m.regenerateBackupCodesFn != nil {
		return m.regenerateBackupCodesFn(ctx, userID)
	}
	return []string{"backup-1", "backup-2"}, nil
}

func (m *mockMFAService) GetMFAStatus(ctx context.Context, userID int64) (*MFAStatusResponseDTO, error) {
	if m.getMFAStatusFn != nil {
		return m.getMFAStatusFn(ctx, userID)
	}
	return &MFAStatusResponseDTO{IsTOTPEnabled: true, BackupCodesCount: 3}, nil
}

func (m *mockMFAService) GetMFAPolicy(ctx context.Context, tenantID int64) (*MFAPolicyDTO, error) {
	if m.getMFAPolicyFn != nil {
		return m.getMFAPolicyFn(ctx, tenantID)
	}
	return &MFAPolicyDTO{}, nil
}

func (m *mockMFAService) IsMFARequired(ctx context.Context, tenantID int64) (bool, error) {
	if m.isMFARequiredFn != nil {
		return m.isMFARequiredFn(ctx, tenantID)
	}
	return false, nil
}

func (m *mockMFAService) IsMethodAllowed(ctx context.Context, userID int64, method string) (bool, error) {
	if m.isMethodAllowedFn != nil {
		return m.isMethodAllowedFn(ctx, userID, method)
	}
	return true, nil
}

func (m *mockMFAService) UserHasMFA(ctx context.Context, userID int64) (bool, error) {
	if m.userHasMFAFn != nil {
		return m.userHasMFAFn(ctx, userID)
	}
	return true, nil
}

func (m *mockMFAService) SensitiveActionStepUpRequired(ctx context.Context, userID int64) (bool, error) {
	if m.sensitiveActionStepUpRequiredFn != nil {
		return m.sensitiveActionStepUpRequiredFn(ctx, userID)
	}
	return false, nil
}

func (m *mockMFAService) StepUpTTLSeconds(ctx context.Context, userID int64) int64 {
	return 300
}

func (m *mockMFAService) AdminResetMFA(ctx context.Context, targetUserUUID string, actorUserID int64, tenantID int64) error {
	if m.adminResetMFAFn != nil {
		return m.adminResetMFAFn(ctx, targetUserUUID, actorUserID)
	}
	return nil
}

func (m *mockMFAService) AdminResetMFAMethod(ctx context.Context, targetUserUUID, method string, actorUserID int64, tenantID int64) error {
	if m.adminResetMFAMethodFn != nil {
		return m.adminResetMFAMethodFn(ctx, targetUserUUID, method, actorUserID)
	}
	return nil
}

func (m *mockMFAService) SelfResetMFA(ctx context.Context, userID int64) error {
	if m.selfResetMFAFn != nil {
		return m.selfResetMFAFn(ctx, userID)
	}
	return nil
}

func (m *mockMFAService) IssueStepUpChallenge(ctx context.Context, userUUID string, allowedMethods []string) (*StepUpChallengeResponseDTO, error) {
	if m.issueStepUpChallengeFn != nil {
		return m.issueStepUpChallengeFn(ctx, userUUID, allowedMethods)
	}
	return &StepUpChallengeResponseDTO{ChallengeToken: "challenge", AllowedMethods: allowedMethods}, nil
}

func (m *mockMFAService) VerifyStepUp(ctx context.Context, req StepUpVerifyRequestDTO, userID int64) (*StepUpVerifyResponseDTO, error) {
	if m.verifyStepUpFn != nil {
		return m.verifyStepUpFn(ctx, req, userID)
	}
	return &StepUpVerifyResponseDTO{AccessToken: "elevated", ExpiresIn: 300}, nil
}

func (m *mockMFAService) SendStepUpSMS(ctx context.Context, userID int64) error {
	if m.sendStepUpSMSFn != nil {
		return m.sendStepUpSMSFn(ctx, userID)
	}
	return nil
}

func (m *mockMFAService) VerifyFactor(_ context.Context, _ int64, _, _ string, _ []byte) ([]string, error) {
	return []string{"pwd", "otp"}, nil
}

func (m *mockMFAService) SyncMFAState(_ context.Context, _ int64) error {
	if m.syncMFAStateFn != nil {
		return m.syncMFAStateFn()
	}
	return nil
}

func (m *mockMFAService) EnrolledMFAMethods(_ context.Context, _ int64) ([]string, error) {
	return []string{"totp", "backup_code"}, nil
}

func (m *mockMFAService) SendSMSChallenge(_ context.Context, _ int64) error { return nil }

func (m *mockMFAService) BeginWebAuthnLogin(_ context.Context, _ int64) (json.RawMessage, error) {
	return json.RawMessage(`{}`), nil
}

func (m *mockMFAService) EnrollSMS(ctx context.Context, userID int64, phone string) error {
	if m.enrollSMSFn != nil {
		return m.enrollSMSFn(ctx, userID, phone)
	}
	return nil
}

func (m *mockMFAService) VerifySMS(ctx context.Context, userID int64, phone, code string) error {
	if m.verifySMSFn != nil {
		return m.verifySMSFn(ctx, userID, phone, code)
	}
	return nil
}

func (m *mockMFAService) DisableSMS(ctx context.Context, userID int64) error {
	if m.disableSMSFn != nil {
		return m.disableSMSFn(ctx, userID)
	}
	return nil
}
func (m *mockMFAService) SendStepUpEmailOTP(ctx context.Context, userID int64) error {
	if m.sendStepUpEmailOTPFn != nil {
		return m.sendStepUpEmailOTPFn(ctx, userID)
	}
	return nil
}
func (m *mockMFAService) EnrollEmailOTP(ctx context.Context, userID int64, email string) error {
	if m.enrollEmailOTPFn != nil {
		return m.enrollEmailOTPFn(ctx, userID, email)
	}
	return nil
}
func (m *mockMFAService) VerifyEmailOTP(ctx context.Context, userID int64, email, code string) error {
	if m.verifyEmailOTPFn != nil {
		return m.verifyEmailOTPFn(ctx, userID, email, code)
	}
	return nil
}
func (m *mockMFAService) DisableEmailOTP(ctx context.Context, userID int64) error {
	if m.disableEmailOTPFn != nil {
		return m.disableEmailOTPFn(ctx, userID)
	}
	return nil
}
func (m *mockMFAService) SendEmailOTPChallenge(ctx context.Context, userID int64) error {
	if m.sendEmailOTPChallengeFn != nil {
		return m.sendEmailOTPChallengeFn(ctx, userID)
	}
	return nil
}

type mockWebAuthnService struct {
	beginRegistrationFn    func(context.Context, int64) (*protocol.CredentialCreation, error)
	finishRegistrationFn   func(context.Context, int64, string, *protocol.ParsedCredentialCreationData) (*UserMFAWebAuthnCredential, error)
	beginAuthenticationFn  func(context.Context, int64) (*protocol.CredentialAssertion, error)
	finishAuthenticationFn func(context.Context, int64, *protocol.ParsedCredentialAssertionData) (*UserMFAWebAuthnCredential, error)
	deleteCredentialFn     func(context.Context, string, int64) error
	downloadCredentialFn   func(context.Context, string, int64) (*WebAuthnCredentialDownloadDTO, error)
}

func (m *mockWebAuthnService) BeginRegistration(ctx context.Context, userID int64) (*protocol.CredentialCreation, error) {
	if m.beginRegistrationFn != nil {
		return m.beginRegistrationFn(ctx, userID)
	}
	return &protocol.CredentialCreation{}, nil
}

func (m *mockWebAuthnService) FinishRegistration(ctx context.Context, userID int64, credName string, response *protocol.ParsedCredentialCreationData) (*UserMFAWebAuthnCredential, error) {
	if m.finishRegistrationFn != nil {
		return m.finishRegistrationFn(ctx, userID, credName, response)
	}
	return &UserMFAWebAuthnCredential{CredentialUUID: mfaTestCredentialUUID, Name: credName, CreatedAt: time.Unix(1_700_000_000, 0).UTC()}, nil
}

func (m *mockWebAuthnService) BeginAuthentication(ctx context.Context, userID int64) (*protocol.CredentialAssertion, error) {
	if m.beginAuthenticationFn != nil {
		return m.beginAuthenticationFn(ctx, userID)
	}
	return &protocol.CredentialAssertion{}, nil
}

func (m *mockWebAuthnService) FinishAuthentication(ctx context.Context, userID int64, response *protocol.ParsedCredentialAssertionData) (*UserMFAWebAuthnCredential, error) {
	if m.finishAuthenticationFn != nil {
		return m.finishAuthenticationFn(ctx, userID, response)
	}
	return &UserMFAWebAuthnCredential{CredentialUUID: mfaTestCredentialUUID, Name: "Security Key"}, nil
}

func (m *mockWebAuthnService) DeleteCredential(ctx context.Context, credentialUUIDStr string, userID int64) error {
	if m.deleteCredentialFn != nil {
		return m.deleteCredentialFn(ctx, credentialUUIDStr, userID)
	}
	return nil
}

func (m *mockWebAuthnService) DownloadCredential(ctx context.Context, credentialUUIDStr string, userID int64) (*WebAuthnCredentialDownloadDTO, error) {
	if m.downloadCredentialFn != nil {
		return m.downloadCredentialFn(ctx, credentialUUIDStr, userID)
	}
	return &WebAuthnCredentialDownloadDTO{CredentialUUID: credentialUUIDStr, Name: "Security Key"}, nil
}

func TestMFAHandler_AuthRequired(t *testing.T) {
	h := NewMFAHandler(&mockMFAService{}, &mockWebAuthnService{})

	tests := []struct {
		name   string
		method string
		path   string
		call   func(http.ResponseWriter, *http.Request)
	}{
		{name: "status", method: http.MethodGet, path: "/mfa/status", call: h.GetStatus},
		{name: "begin totp", method: http.MethodPost, path: "/mfa/totp/enroll", call: h.BeginTOTPEnrollment},
		{name: "finish totp", method: http.MethodPost, path: "/mfa/totp/verify", call: h.FinishTOTPEnrollment},
		{name: "disable totp", method: http.MethodDelete, path: "/mfa/totp", call: h.DisableTOTP},
		{name: "backup count", method: http.MethodGet, path: "/mfa/backup-codes/count", call: h.GetBackupCodesCount},
		{name: "regenerate backup codes", method: http.MethodPost, path: "/mfa/backup-codes/regenerate", call: h.RegenerateBackupCodes},
		{name: "webauthn begin registration", method: http.MethodPost, path: "/mfa/webauthn/register/begin", call: h.WebAuthnBeginRegistration},
		{name: "webauthn finish registration", method: http.MethodPost, path: "/mfa/webauthn/register/finish", call: h.WebAuthnFinishRegistration},
		{name: "webauthn begin authentication", method: http.MethodPost, path: "/mfa/webauthn/auth/begin", call: h.WebAuthnBeginAuthentication},
		{name: "webauthn finish authentication", method: http.MethodPost, path: "/mfa/webauthn/auth/finish", call: h.WebAuthnFinishAuthentication},
		{name: "webauthn delete", method: http.MethodDelete, path: "/mfa/webauthn/" + mfaTestCredentialUUID.String(), call: h.WebAuthnDeleteCredential},
		{name: "step-up challenge", method: http.MethodPost, path: "/mfa/step-up/challenge", call: h.IssueStepUpChallenge},
		{name: "step-up verify", method: http.MethodPost, path: "/mfa/step-up/verify", call: h.VerifyStepUp},
		{name: "admin reset", method: http.MethodPost, path: "/mfa/admin/users/" + mfaTestUserUUID.String() + "/reset", call: h.AdminResetMFA},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			req := httptest.NewRequest(tt.method, tt.path, nil)

			tt.call(rec, req)

			assert.Equal(t, http.StatusUnauthorized, rec.Code)
			assert.JSONEq(t, `{"success":false,"error":"Unauthorized"}`, rec.Body.String())
		})
	}
}

func TestMFAHandler_ServiceBackedEndpoints(t *testing.T) {
	errUnauthorized := apperror.NewUnauthorized("mfa rejected")

	tests := []struct {
		name         string
		request      *http.Request
		handler      func(*MFAHandler, http.ResponseWriter, *http.Request)
		mfaSvc       *mockMFAService
		webAuthnSvc  *mockWebAuthnService
		wantStatus   int
		wantContains string
	}{
		{
			name:    "status success",
			request: authenticatedMFARequest(t, http.MethodGet, "/mfa/status", nil),
			handler: (*MFAHandler).GetStatus,
			mfaSvc: &mockMFAService{getMFAStatusFn: func(_ context.Context, userID int64) (*MFAStatusResponseDTO, error) {
				assert.Equal(t, mfaTestUserID, userID)
				return &MFAStatusResponseDTO{IsTOTPEnabled: true, BackupCodesCount: 2}, nil
			}},
			wantStatus:   http.StatusOK,
			wantContains: `"is_totp_enabled":true`,
		},
		{
			name:    "status service error maps",
			request: authenticatedMFARequest(t, http.MethodGet, "/mfa/status", nil),
			handler: (*MFAHandler).GetStatus,
			mfaSvc: &mockMFAService{getMFAStatusFn: func(context.Context, int64) (*MFAStatusResponseDTO, error) {
				return nil, errUnauthorized
			}},
			wantStatus:   http.StatusUnauthorized,
			wantContains: "mfa rejected",
		},
		{
			name:    "begin totp success",
			request: authenticatedMFARequest(t, http.MethodPost, "/mfa/totp/enroll", nil),
			handler: (*MFAHandler).BeginTOTPEnrollment,
			mfaSvc: &mockMFAService{beginTOTPEnrollmentFn: func(_ context.Context, userID int64) (*TOTPEnrollResponseDTO, error) {
				assert.Equal(t, mfaTestUserID, userID)
				return &TOTPEnrollResponseDTO{Secret: "secret", QRCodeURL: "otpauth://totp/test"}, nil
			}},
			wantStatus:   http.StatusOK,
			wantContains: "otpauth://totp/test",
		},
		{
			name:    "disable totp success",
			request: authenticatedMFARequest(t, http.MethodDelete, "/mfa/totp", nil),
			handler: (*MFAHandler).DisableTOTP,
			mfaSvc: &mockMFAService{disableTOTPFn: func(_ context.Context, userID int64) error {
				assert.Equal(t, mfaTestUserID, userID)
				return nil
			}},
			wantStatus:   http.StatusOK,
			wantContains: "TOTP disabled",
		},
		{
			name:    "backup count success",
			request: authenticatedMFARequest(t, http.MethodGet, "/mfa/backup-codes/count", nil),
			handler: (*MFAHandler).GetBackupCodesCount,
			mfaSvc: &mockMFAService{getBackupCodesCountFn: func(_ context.Context, userID int64) (int, error) {
				assert.Equal(t, mfaTestUserID, userID)
				return 7, nil
			}},
			wantStatus:   http.StatusOK,
			wantContains: `"remaining":7`,
		},
		{
			name:    "regenerate backup codes success",
			request: authenticatedMFARequest(t, http.MethodPost, "/mfa/backup-codes/regenerate", nil),
			handler: (*MFAHandler).RegenerateBackupCodes,
			mfaSvc: &mockMFAService{regenerateBackupCodesFn: func(_ context.Context, userID int64) ([]string, error) {
				assert.Equal(t, mfaTestUserID, userID)
				return []string{"code-1", "code-2"}, nil
			}},
			wantStatus:   http.StatusOK,
			wantContains: "code-1",
		},
		{
			name:    "webauthn begin registration success",
			request: authenticatedMFARequest(t, http.MethodPost, "/mfa/webauthn/register/begin", nil),
			handler: (*MFAHandler).WebAuthnBeginRegistration,
			webAuthnSvc: &mockWebAuthnService{beginRegistrationFn: func(_ context.Context, userID int64) (*protocol.CredentialCreation, error) {
				assert.Equal(t, mfaTestUserID, userID)
				return &protocol.CredentialCreation{}, nil
			}},
			wantStatus:   http.StatusOK,
			wantContains: "WebAuthn registration ceremony started",
		},
		{
			name:    "webauthn begin authentication success",
			request: authenticatedMFARequest(t, http.MethodPost, "/mfa/webauthn/auth/begin", nil),
			handler: (*MFAHandler).WebAuthnBeginAuthentication,
			webAuthnSvc: &mockWebAuthnService{beginAuthenticationFn: func(_ context.Context, userID int64) (*protocol.CredentialAssertion, error) {
				assert.Equal(t, mfaTestUserID, userID)
				return &protocol.CredentialAssertion{}, nil
			}},
			wantStatus:   http.StatusOK,
			wantContains: "WebAuthn authentication ceremony started",
		},
		{
			name: "webauthn delete success with chi param",
			request: authenticatedMFARequestWithParam(
				t,
				http.MethodDelete,
				"/mfa/webauthn/"+mfaTestCredentialUUID.String(),
				nil,
				"credential_uuid",
				mfaTestCredentialUUID.String(),
			),
			handler: (*MFAHandler).WebAuthnDeleteCredential,
			webAuthnSvc: &mockWebAuthnService{deleteCredentialFn: func(_ context.Context, credentialUUID string, userID int64) error {
				assert.Equal(t, mfaTestCredentialUUID.String(), credentialUUID)
				assert.Equal(t, mfaTestUserID, userID)
				return nil
			}},
			wantStatus:   http.StatusOK,
			wantContains: "Credential deleted",
		},
		{
			name:    "step-up challenge success",
			request: authenticatedMFARequest(t, http.MethodPost, "/mfa/step-up/challenge", nil),
			handler: (*MFAHandler).IssueStepUpChallenge,
			mfaSvc: &mockMFAService{issueStepUpChallengeFn: func(_ context.Context, userUUID string, allowed []string) (*StepUpChallengeResponseDTO, error) {
				assert.Equal(t, mfaTestUserUUID.String(), userUUID)
				assert.Equal(t, []string{"totp", "backup_code"}, allowed)
				return &StepUpChallengeResponseDTO{ChallengeToken: "challenge", AllowedMethods: allowed}, nil
			}},
			wantStatus:   http.StatusOK,
			wantContains: "challenge",
		},
		{
			name:    "admin reset success with chi param",
			request: authenticatedMFARequestWithParam(t, http.MethodPost, "/mfa/admin/users/"+mfaTestUserUUID.String()+"/reset", nil, "user_uuid", mfaTestUserUUID.String()),
			handler: (*MFAHandler).AdminResetMFA,
			mfaSvc: &mockMFAService{adminResetMFAFn: func(_ context.Context, targetUserUUID string, actorUserID int64) error {
				assert.Equal(t, mfaTestUserUUID.String(), targetUserUUID)
				assert.Equal(t, mfaTestUserID, actorUserID)
				return nil
			}},
			wantStatus:   http.StatusOK,
			wantContains: "MFA reset successfully",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.mfaSvc == nil {
				tt.mfaSvc = &mockMFAService{}
			}
			if tt.webAuthnSvc == nil {
				tt.webAuthnSvc = &mockWebAuthnService{}
			}
			h := NewMFAHandler(tt.mfaSvc, tt.webAuthnSvc)
			rec := httptest.NewRecorder()

			tt.handler(h, rec, tt.request)

			assert.Equal(t, tt.wantStatus, rec.Code)
			assert.Contains(t, rec.Body.String(), tt.wantContains)
		})
	}
}

func TestMFAHandler_JSONBodyEndpoints(t *testing.T) {
	tests := []struct {
		name         string
		request      *http.Request
		handler      func(*MFAHandler, http.ResponseWriter, *http.Request)
		mfaSvc       *mockMFAService
		wantStatus   int
		wantContains string
	}{
		{
			name:         "finish totp bad body",
			request:      authenticatedRawMFARequest(http.MethodPost, "/mfa/totp/verify", "{bad json"),
			handler:      (*MFAHandler).FinishTOTPEnrollment,
			mfaSvc:       &mockMFAService{},
			wantStatus:   http.StatusBadRequest,
			wantContains: "Invalid request body",
		},
		{
			name:    "finish totp service receives code",
			request: authenticatedMFARequest(t, http.MethodPost, "/mfa/totp/verify", TOTPVerifyRequestDTO{Code: "123456"}),
			handler: (*MFAHandler).FinishTOTPEnrollment,
			mfaSvc: &mockMFAService{finishTOTPEnrollmentFn: func(_ context.Context, userID int64, code string) ([]string, error) {
				assert.Equal(t, mfaTestUserID, userID)
				assert.Equal(t, "123456", code)
				return []string{"backup-1"}, nil
			}},
			wantStatus:   http.StatusOK,
			wantContains: "backup-1",
		},
		{
			name:         "verify step-up bad body",
			request:      authenticatedRawMFARequest(http.MethodPost, "/mfa/step-up/verify", "{bad json"),
			handler:      (*MFAHandler).VerifyStepUp,
			mfaSvc:       &mockMFAService{},
			wantStatus:   http.StatusBadRequest,
			wantContains: "Invalid request body",
		},
		{
			name:    "verify step-up service receives request",
			request: authenticatedMFARequest(t, http.MethodPost, "/mfa/step-up/verify", StepUpVerifyRequestDTO{ChallengeToken: "challenge", Method: "totp", Code: "123456"}),
			handler: (*MFAHandler).VerifyStepUp,
			mfaSvc: &mockMFAService{verifyStepUpFn: func(_ context.Context, req StepUpVerifyRequestDTO, userID int64) (*StepUpVerifyResponseDTO, error) {
				assert.Equal(t, mfaTestUserID, userID)
				assert.Equal(t, "challenge", req.ChallengeToken)
				assert.Equal(t, "totp", req.Method)
				assert.Equal(t, "123456", req.Code)
				return &StepUpVerifyResponseDTO{AccessToken: "elevated", ExpiresIn: 300}, nil
			}},
			wantStatus:   http.StatusOK,
			wantContains: "elevated",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := NewMFAHandler(tt.mfaSvc, &mockWebAuthnService{})
			rec := httptest.NewRecorder()

			tt.handler(h, rec, tt.request)

			assert.Equal(t, tt.wantStatus, rec.Code)
			assert.Contains(t, rec.Body.String(), tt.wantContains)
		})
	}
}

func TestMFAHandler_WebAuthnFinishBadBodies(t *testing.T) {
	h := NewMFAHandler(&mockMFAService{}, &mockWebAuthnService{
		finishRegistrationFn: func(context.Context, int64, string, *protocol.ParsedCredentialCreationData) (*UserMFAWebAuthnCredential, error) {
			t.Fatal("service should not be called for malformed registration response")
			return nil, nil
		},
		finishAuthenticationFn: func(context.Context, int64, *protocol.ParsedCredentialAssertionData) (*UserMFAWebAuthnCredential, error) {
			t.Fatal("service should not be called for malformed assertion response")
			return nil, nil
		},
	})

	tests := []struct {
		name       string
		request    *http.Request
		call       func(http.ResponseWriter, *http.Request)
		wantErrMsg string
	}{
		{
			name:       "finish registration malformed body",
			request:    authenticatedRawMFARequest(http.MethodPost, "/mfa/webauthn/register/finish?name=laptop", "{bad json"),
			call:       h.WebAuthnFinishRegistration,
			wantErrMsg: "Invalid WebAuthn credential response",
		},
		{
			name:       "finish authentication malformed body",
			request:    authenticatedRawMFARequest(http.MethodPost, "/mfa/webauthn/auth/finish", "{bad json"),
			call:       h.WebAuthnFinishAuthentication,
			wantErrMsg: "Invalid WebAuthn assertion response",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := httptest.NewRecorder()

			tt.call(rec, tt.request)

			assert.Equal(t, http.StatusBadRequest, rec.Code)
			assert.Contains(t, rec.Body.String(), tt.wantErrMsg)
		})
	}
}

func TestMFAHandler_WebAuthnPolicyGate(t *testing.T) {
	mfaSvc := &mockMFAService{
		isMethodAllowedFn: func(_ context.Context, userID int64, method string) (bool, error) {
			assert.Equal(t, mfaTestUserID, userID)
			assert.Equal(t, "webauthn", method)
			return false, nil
		},
	}
	webAuthnSvc := &mockWebAuthnService{
		beginRegistrationFn: func(context.Context, int64) (*protocol.CredentialCreation, error) {
			t.Fatal("registration ceremony should not start when WebAuthn is disallowed")
			return nil, nil
		},
		finishRegistrationFn: func(context.Context, int64, string, *protocol.ParsedCredentialCreationData) (*UserMFAWebAuthnCredential, error) {
			t.Fatal("registration ceremony should not finish when WebAuthn is disallowed")
			return nil, nil
		},
		beginAuthenticationFn: func(context.Context, int64) (*protocol.CredentialAssertion, error) {
			t.Fatal("authentication ceremony should not start when WebAuthn is disallowed")
			return nil, nil
		},
		finishAuthenticationFn: func(context.Context, int64, *protocol.ParsedCredentialAssertionData) (*UserMFAWebAuthnCredential, error) {
			t.Fatal("authentication ceremony should not finish when WebAuthn is disallowed")
			return nil, nil
		},
	}
	h := NewMFAHandler(mfaSvc, webAuthnSvc)

	tests := []struct {
		name    string
		request *http.Request
		call    func(http.ResponseWriter, *http.Request)
	}{
		{name: "begin registration", request: authenticatedMFARequest(t, http.MethodPost, "/mfa/webauthn/register/begin", nil), call: h.WebAuthnBeginRegistration},
		{name: "finish registration", request: authenticatedRawMFARequest(http.MethodPost, "/mfa/webauthn/register/finish", "{}"), call: h.WebAuthnFinishRegistration},
		{name: "begin authentication", request: authenticatedMFARequest(t, http.MethodPost, "/mfa/webauthn/auth/begin", nil), call: h.WebAuthnBeginAuthentication},
		{name: "finish authentication", request: authenticatedRawMFARequest(http.MethodPost, "/mfa/webauthn/auth/finish", "{}"), call: h.WebAuthnFinishAuthentication},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := httptest.NewRecorder()

			tt.call(rec, tt.request)

			assert.Equal(t, http.StatusForbidden, rec.Code)
			assert.Contains(t, rec.Body.String(), "WebAuthn MFA is not permitted by tenant policy")
		})
	}
}

func TestMFAHandler_WebAuthnFinishParserBackedBranches(t *testing.T) {
	originalCreation := parseWebAuthnCreationResponse
	originalRequest := parseWebAuthnRequestResponse
	t.Cleanup(func() {
		parseWebAuthnCreationResponse = originalCreation
		parseWebAuthnRequestResponse = originalRequest
	})

	parseWebAuthnCreationResponse = func(io.Reader) (*protocol.ParsedCredentialCreationData, error) {
		return &protocol.ParsedCredentialCreationData{}, nil
	}
	parseWebAuthnRequestResponse = func(io.Reader) (*protocol.ParsedCredentialAssertionData, error) {
		return &protocol.ParsedCredentialAssertionData{}, nil
	}

	tests := []struct {
		name         string
		request      *http.Request
		handler      func(*MFAHandler, http.ResponseWriter, *http.Request)
		webAuthnSvc  *mockWebAuthnService
		wantStatus   int
		wantContains string
	}{
		{
			name:    "finish registration success",
			request: authenticatedMFARequest(t, http.MethodPost, "/mfa/webauthn/register/finish?name=laptop", nil),
			handler: (*MFAHandler).WebAuthnFinishRegistration,
			webAuthnSvc: &mockWebAuthnService{finishRegistrationFn: func(_ context.Context, userID int64, name string, _ *protocol.ParsedCredentialCreationData) (*UserMFAWebAuthnCredential, error) {
				assert.Equal(t, mfaTestUserID, userID)
				assert.Equal(t, "laptop", name)
				return &UserMFAWebAuthnCredential{CredentialUUID: mfaTestCredentialUUID, Name: name, Transport: "usb", CreatedAt: time.Unix(1_700_000_000, 0).UTC()}, nil
			}},
			wantStatus:   http.StatusOK,
			wantContains: "Passkey registered successfully",
		},
		{
			name:    "finish registration service error",
			request: authenticatedMFARequest(t, http.MethodPost, "/mfa/webauthn/register/finish", nil),
			handler: (*MFAHandler).WebAuthnFinishRegistration,
			webAuthnSvc: &mockWebAuthnService{finishRegistrationFn: func(context.Context, int64, string, *protocol.ParsedCredentialCreationData) (*UserMFAWebAuthnCredential, error) {
				return nil, apperror.NewInternal("registration failed", errors.New("db down"))
			}},
			wantStatus:   http.StatusInternalServerError,
			wantContains: "WebAuthn registration failed",
		},
		{
			name:    "finish authentication success",
			request: authenticatedMFARequest(t, http.MethodPost, "/mfa/webauthn/auth/finish", nil),
			handler: (*MFAHandler).WebAuthnFinishAuthentication,
			webAuthnSvc: &mockWebAuthnService{finishAuthenticationFn: func(_ context.Context, userID int64, _ *protocol.ParsedCredentialAssertionData) (*UserMFAWebAuthnCredential, error) {
				assert.Equal(t, mfaTestUserID, userID)
				return &UserMFAWebAuthnCredential{CredentialUUID: mfaTestCredentialUUID, Name: "Security Key"}, nil
			}},
			wantStatus:   http.StatusOK,
			wantContains: "WebAuthn authentication succeeded",
		},
		{
			name:    "finish authentication service error",
			request: authenticatedMFARequest(t, http.MethodPost, "/mfa/webauthn/auth/finish", nil),
			handler: (*MFAHandler).WebAuthnFinishAuthentication,
			webAuthnSvc: &mockWebAuthnService{finishAuthenticationFn: func(context.Context, int64, *protocol.ParsedCredentialAssertionData) (*UserMFAWebAuthnCredential, error) {
				return nil, apperror.NewUnauthorized("assertion failed")
			}},
			wantStatus:   http.StatusUnauthorized,
			wantContains: "assertion failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := NewMFAHandler(&mockMFAService{}, tt.webAuthnSvc)
			rec := httptest.NewRecorder()

			tt.handler(h, rec, tt.request)

			assert.Equal(t, tt.wantStatus, rec.Code)
			assert.Contains(t, rec.Body.String(), tt.wantContains)
		})
	}
}

func TestMFAHandler_ServiceErrors(t *testing.T) {
	errInternal := errors.New("db down")

	tests := []struct {
		name         string
		request      *http.Request
		handler      func(*MFAHandler, http.ResponseWriter, *http.Request)
		mfaSvc       *mockMFAService
		webAuthnSvc  *mockWebAuthnService
		wantStatus   int
		wantContains string
	}{
		{
			name:    "finish totp validation error",
			request: authenticatedMFARequest(t, http.MethodPost, "/mfa/totp/verify", TOTPVerifyRequestDTO{Code: "000000"}),
			handler: (*MFAHandler).FinishTOTPEnrollment,
			mfaSvc: &mockMFAService{finishTOTPEnrollmentFn: func(context.Context, int64, string) ([]string, error) {
				return nil, apperror.NewValidation("invalid TOTP code")
			}},
			wantStatus:   http.StatusBadRequest,
			wantContains: "invalid TOTP code",
		},
		{
			name:    "disable totp internal error uses fallback",
			request: authenticatedMFARequest(t, http.MethodDelete, "/mfa/totp", nil),
			handler: (*MFAHandler).DisableTOTP,
			mfaSvc: &mockMFAService{disableTOTPFn: func(context.Context, int64) error {
				return apperror.NewInternal("disable failed", errInternal)
			}},
			wantStatus:   http.StatusInternalServerError,
			wantContains: "Failed to disable TOTP",
		},
		{
			name:    "begin totp service error",
			request: authenticatedMFARequest(t, http.MethodPost, "/mfa/totp/enroll", nil),
			handler: (*MFAHandler).BeginTOTPEnrollment,
			mfaSvc: &mockMFAService{beginTOTPEnrollmentFn: func(context.Context, int64) (*TOTPEnrollResponseDTO, error) {
				return nil, apperror.NewInternal("begin failed", errInternal)
			}},
			wantStatus:   http.StatusInternalServerError,
			wantContains: "Failed to begin TOTP enrollment",
		},
		{
			name:    "backup count service error",
			request: authenticatedMFARequest(t, http.MethodGet, "/mfa/backup-codes/count", nil),
			handler: (*MFAHandler).GetBackupCodesCount,
			mfaSvc: &mockMFAService{getBackupCodesCountFn: func(context.Context, int64) (int, error) {
				return 0, apperror.NewInternal("count failed", errInternal)
			}},
			wantStatus:   http.StatusInternalServerError,
			wantContains: "Failed to get backup codes count",
		},
		{
			name:    "regenerate backup codes service error",
			request: authenticatedMFARequest(t, http.MethodPost, "/mfa/backup-codes/regenerate", nil),
			handler: (*MFAHandler).RegenerateBackupCodes,
			mfaSvc: &mockMFAService{regenerateBackupCodesFn: func(context.Context, int64) ([]string, error) {
				return nil, apperror.NewInternal("regenerate failed", errInternal)
			}},
			wantStatus:   http.StatusInternalServerError,
			wantContains: "Failed to regenerate backup codes",
		},
		{
			name:    "webauthn begin registration service error",
			request: authenticatedMFARequest(t, http.MethodPost, "/mfa/webauthn/register/begin", nil),
			handler: (*MFAHandler).WebAuthnBeginRegistration,
			webAuthnSvc: &mockWebAuthnService{beginRegistrationFn: func(context.Context, int64) (*protocol.CredentialCreation, error) {
				return nil, apperror.NewInternal("begin registration failed", errInternal)
			}},
			wantStatus:   http.StatusInternalServerError,
			wantContains: "Failed to begin WebAuthn registration",
		},
		{
			name:    "webauthn begin auth service error",
			request: authenticatedMFARequest(t, http.MethodPost, "/mfa/webauthn/auth/begin", nil),
			handler: (*MFAHandler).WebAuthnBeginAuthentication,
			webAuthnSvc: &mockWebAuthnService{beginAuthenticationFn: func(context.Context, int64) (*protocol.CredentialAssertion, error) {
				return nil, apperror.NewValidation("no credentials")
			}},
			wantStatus:   http.StatusBadRequest,
			wantContains: "no credentials",
		},
		{
			name:    "step-up challenge service error",
			request: authenticatedMFARequest(t, http.MethodPost, "/mfa/step-up/challenge", nil),
			handler: (*MFAHandler).IssueStepUpChallenge,
			mfaSvc: &mockMFAService{issueStepUpChallengeFn: func(context.Context, string, []string) (*StepUpChallengeResponseDTO, error) {
				return nil, apperror.NewInternal("challenge failed", errInternal)
			}},
			wantStatus:   http.StatusInternalServerError,
			wantContains: "Failed to issue step-up challenge",
		},
		{
			name:    "delete credential not found",
			request: authenticatedMFARequestWithParam(t, http.MethodDelete, "/mfa/webauthn/"+mfaTestCredentialUUID.String(), nil, "credential_uuid", mfaTestCredentialUUID.String()),
			handler: (*MFAHandler).WebAuthnDeleteCredential,
			webAuthnSvc: &mockWebAuthnService{deleteCredentialFn: func(context.Context, string, int64) error {
				return apperror.NewNotFound("credential not found")
			}},
			wantStatus:   http.StatusNotFound,
			wantContains: "credential not found",
		},
		{
			name:    "verify step-up unauthorized",
			request: authenticatedMFARequest(t, http.MethodPost, "/mfa/step-up/verify", StepUpVerifyRequestDTO{ChallengeToken: "challenge", Method: "totp", Code: "bad"}),
			handler: (*MFAHandler).VerifyStepUp,
			mfaSvc: &mockMFAService{verifyStepUpFn: func(context.Context, StepUpVerifyRequestDTO, int64) (*StepUpVerifyResponseDTO, error) {
				return nil, apperror.NewUnauthorized("invalid TOTP code")
			}},
			wantStatus:   http.StatusUnauthorized,
			wantContains: "invalid TOTP code",
		},
		{
			name:    "admin reset not found",
			request: authenticatedMFARequestWithParam(t, http.MethodPost, "/mfa/admin/users/"+mfaTestUserUUID.String()+"/reset", nil, "user_uuid", mfaTestUserUUID.String()),
			handler: (*MFAHandler).AdminResetMFA,
			mfaSvc: &mockMFAService{adminResetMFAFn: func(context.Context, string, int64) error {
				return apperror.NewNotFound("target user not found")
			}},
			wantStatus:   http.StatusNotFound,
			wantContains: "target user not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.mfaSvc == nil {
				tt.mfaSvc = &mockMFAService{}
			}
			if tt.webAuthnSvc == nil {
				tt.webAuthnSvc = &mockWebAuthnService{}
			}
			h := NewMFAHandler(tt.mfaSvc, tt.webAuthnSvc)
			rec := httptest.NewRecorder()

			tt.handler(h, rec, tt.request)

			assert.Equal(t, tt.wantStatus, rec.Code)
			assert.Contains(t, rec.Body.String(), tt.wantContains)
		})
	}
}

func authenticatedMFARequest(t *testing.T, method, path string, body any) *http.Request {
	t.Helper()
	var buf bytes.Buffer
	if body != nil {
		require.NoError(t, json.NewEncoder(&buf).Encode(body))
	}
	req := httptest.NewRequest(method, path, &buf)
	req.Header.Set("Content-Type", "application/json")
	return withMFAUser(req)
}

func authenticatedRawMFARequest(method, path, body string) *http.Request {
	req := httptest.NewRequest(method, path, bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	return withMFAUser(req)
}

func authenticatedMFARequestWithParam(t *testing.T, method, path string, body any, key string, value string) *http.Request {
	t.Helper()
	req := authenticatedMFARequest(t, method, path, body)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add(key, value)
	return req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
}

func withMFAUser(req *http.Request) *http.Request {
	user := &authctx.AuthUser{UserID: mfaTestUserID, UserUUID: mfaTestUserUUID}
	tenant := &authctx.AuthTenant{TenantID: mfaTestTenantID, TenantUUID: mfaTestTenantUUID}
	return middleware.WithAuthContext(req, &authctx.AuthContext{User: user, Tenant: tenant})
}

func TestMFAHandler_RequireStepUpOrEnrolledMFA(t *testing.T) {
	tests := []struct {
		name       string
		request    func() *http.Request
		statusFn   func(context.Context, int64) (*MFAStatusResponseDTO, error)
		wantStatus int
		wantNext   bool
	}{
		{
			name: "stepped-up session passes without enrolled MFA",
			request: func() *http.Request {
				req := withMFAUser(httptest.NewRequest(http.MethodGet, "/mfa/webauthn/x/download", nil))
				return middleware.WithJWTClaims(req, &middleware.JWTClaims{ACR: jwt.ACRLevel2})
			},
			statusFn: func(context.Context, int64) (*MFAStatusResponseDTO, error) {
				return &MFAStatusResponseDTO{}, nil // no factors — must still pass via acr=2
			},
			wantStatus: http.StatusOK,
			wantNext:   true,
		},
		{
			name: "enrolled MFA factor passes without step-up",
			request: func() *http.Request {
				return withMFAUser(httptest.NewRequest(http.MethodGet, "/mfa/webauthn/x/download", nil))
			},
			statusFn: func(context.Context, int64) (*MFAStatusResponseDTO, error) {
				return &MFAStatusResponseDTO{IsWebAuthnEnabled: true}, nil
			},
			wantStatus: http.StatusOK,
			wantNext:   true,
		},
		{
			name: "no enrolled MFA and no step-up is forbidden",
			request: func() *http.Request {
				return withMFAUser(httptest.NewRequest(http.MethodGet, "/mfa/webauthn/x/download", nil))
			},
			statusFn: func(context.Context, int64) (*MFAStatusResponseDTO, error) {
				return &MFAStatusResponseDTO{}, nil
			},
			wantStatus: http.StatusForbidden,
			wantNext:   false,
		},
		{
			name: "missing user is unauthorized",
			request: func() *http.Request {
				return httptest.NewRequest(http.MethodGet, "/mfa/webauthn/x/download", nil)
			},
			statusFn:   func(context.Context, int64) (*MFAStatusResponseDTO, error) { return &MFAStatusResponseDTO{}, nil },
			wantStatus: http.StatusUnauthorized,
			wantNext:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := NewMFAHandler(&mockMFAService{getMFAStatusFn: tt.statusFn}, &mockWebAuthnService{})

			nextCalled := false
			next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				nextCalled = true
				w.WriteHeader(http.StatusOK)
			})

			rec := httptest.NewRecorder()
			h.RequireStepUpOrEnrolledMFA(next).ServeHTTP(rec, tt.request())

			assert.Equal(t, tt.wantStatus, rec.Code)
			assert.Equal(t, tt.wantNext, nextCalled)
		})
	}
}
