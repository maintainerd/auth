package mfa

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-webauthn/webauthn/protocol"
	"github.com/maintainerd/auth/internal/platform/middleware"
	resp "github.com/maintainerd/auth/internal/platform/response"
)

// MFAHandler handles all MFA self-service endpoints (TOTP, backup codes, WebAuthn, step-up).
type MFAHandler struct {
	mfaSvc      MFAService
	webAuthnSvc WebAuthnService
}

func NewMFAHandler(mfaSvc MFAService, webAuthnSvc WebAuthnService) *MFAHandler {
	return &MFAHandler{mfaSvc: mfaSvc, webAuthnSvc: webAuthnSvc}
}

// ──────────────────────────────────────────────────────────────────────────────
// Status
// ──────────────────────────────────────────────────────────────────────────────

// GetStatus returns the current MFA enrollment state for the authenticated user.
//
// GET /mfa/status
func (h *MFAHandler) GetStatus(w http.ResponseWriter, r *http.Request) {
	user := middleware.AuthFromRequest(r).User
	if user == nil {
		resp.Error(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	status, err := h.mfaSvc.GetMFAStatus(r.Context(), user.UserID)
	if err != nil {
		resp.HandleServiceError(w, r, "Failed to retrieve MFA status", err)
		return
	}

	resp.Success(w, status, "MFA status retrieved")
}

// ──────────────────────────────────────────────────────────────────────────────
// TOTP
// ──────────────────────────────────────────────────────────────────────────────

// BeginTOTPEnrollment generates a TOTP secret and QR code URL.
//
// POST /mfa/totp/enroll
func (h *MFAHandler) BeginTOTPEnrollment(w http.ResponseWriter, r *http.Request) {
	user := middleware.AuthFromRequest(r).User
	if user == nil {
		resp.Error(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	result, err := h.mfaSvc.BeginTOTPEnrollment(r.Context(), user.UserID)
	if err != nil {
		resp.HandleServiceError(w, r, "Failed to begin TOTP enrollment", err)
		return
	}

	resp.Success(w, result, "TOTP enrollment started")
}

// FinishTOTPEnrollment verifies a TOTP code and activates the secret.
//
// POST /mfa/totp/verify
func (h *MFAHandler) FinishTOTPEnrollment(w http.ResponseWriter, r *http.Request) {
	user := middleware.AuthFromRequest(r).User
	if user == nil {
		resp.Error(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	var req TOTPVerifyRequestDTO
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		resp.BadRequestBody(w)
		return
	}

	codes, err := h.mfaSvc.FinishTOTPEnrollment(r.Context(), user.UserID, req.Code)
	if err != nil {
		resp.HandleServiceError(w, r, "TOTP enrollment failed", err)
		return
	}

	resp.Success(w, BackupCodesResponseDTO{Codes: codes}, "TOTP enrolled successfully — save your backup codes")
}

// DisableTOTP removes TOTP enrollment for the authenticated user.
//
// DELETE /mfa/totp
func (h *MFAHandler) DisableTOTP(w http.ResponseWriter, r *http.Request) {
	user := middleware.AuthFromRequest(r).User
	if user == nil {
		resp.Error(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	if err := h.mfaSvc.DisableTOTP(r.Context(), user.UserID); err != nil {
		resp.HandleServiceError(w, r, "Failed to disable TOTP", err)
		return
	}

	resp.Success(w, nil, "TOTP disabled")
}

// ──────────────────────────────────────────────────────────────────────────────
// Backup Codes
// ──────────────────────────────────────────────────────────────────────────────

// GetBackupCodesCount returns the count of unused backup codes.
//
// GET /mfa/backup-codes/count
func (h *MFAHandler) GetBackupCodesCount(w http.ResponseWriter, r *http.Request) {
	user := middleware.AuthFromRequest(r).User
	if user == nil {
		resp.Error(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	count, err := h.mfaSvc.GetBackupCodesCount(r.Context(), user.UserID)
	if err != nil {
		resp.HandleServiceError(w, r, "Failed to get backup codes count", err)
		return
	}

	resp.Success(w, map[string]int{"remaining": count}, "")
}

// RegenerateBackupCodes issues a fresh set of backup codes, invalidating old ones.
//
// POST /mfa/backup-codes/regenerate
func (h *MFAHandler) RegenerateBackupCodes(w http.ResponseWriter, r *http.Request) {
	user := middleware.AuthFromRequest(r).User
	if user == nil {
		resp.Error(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	codes, err := h.mfaSvc.RegenerateBackupCodes(r.Context(), user.UserID)
	if err != nil {
		resp.HandleServiceError(w, r, "Failed to regenerate backup codes", err)
		return
	}

	resp.Success(w, BackupCodesResponseDTO{Codes: codes}, "New backup codes generated — save them now")
}

// ──────────────────────────────────────────────────────────────────────────────
// WebAuthn registration
// ──────────────────────────────────────────────────────────────────────────────

// WebAuthnBeginRegistration starts a passkey registration ceremony.
//
// POST /mfa/webauthn/register/begin
func (h *MFAHandler) WebAuthnBeginRegistration(w http.ResponseWriter, r *http.Request) {
	user := middleware.AuthFromRequest(r).User
	if user == nil {
		resp.Error(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	creation, err := h.webAuthnSvc.BeginRegistration(r.Context(), user.UserID)
	if err != nil {
		resp.HandleServiceError(w, r, "Failed to begin WebAuthn registration", err)
		return
	}

	resp.Success(w, creation, "WebAuthn registration ceremony started")
}

// WebAuthnFinishRegistration completes a passkey registration ceremony.
//
// POST /mfa/webauthn/register/finish
func (h *MFAHandler) WebAuthnFinishRegistration(w http.ResponseWriter, r *http.Request) {
	user := middleware.AuthFromRequest(r).User
	if user == nil {
		resp.Error(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	credName := r.URL.Query().Get("name")

	parsedResponse, err := protocol.ParseCredentialCreationResponseBody(r.Body)
	if err != nil {
		resp.Error(w, http.StatusBadRequest, "Invalid WebAuthn credential response")
		return
	}

	cred, svcErr := h.webAuthnSvc.FinishRegistration(r.Context(), user.UserID, credName, parsedResponse)
	if svcErr != nil {
		resp.HandleServiceError(w, r, "WebAuthn registration failed", svcErr)
		return
	}

	resp.Success(w, WebAuthnCredentialSummaryDTO{
		CredentialUUID: cred.CredentialUUID.String(),
		Name:           cred.Name,
		Transport:      cred.Transport,
		CreatedAt:      cred.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}, "Passkey registered successfully")
}

// ──────────────────────────────────────────────────────────────────────────────
// WebAuthn authentication
// ──────────────────────────────────────────────────────────────────────────────

// WebAuthnBeginAuthentication starts a passkey assertion ceremony.
//
// POST /mfa/webauthn/auth/begin
func (h *MFAHandler) WebAuthnBeginAuthentication(w http.ResponseWriter, r *http.Request) {
	user := middleware.AuthFromRequest(r).User
	if user == nil {
		resp.Error(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	assertion, err := h.webAuthnSvc.BeginAuthentication(r.Context(), user.UserID)
	if err != nil {
		resp.HandleServiceError(w, r, "Failed to begin WebAuthn authentication", err)
		return
	}

	resp.Success(w, assertion, "WebAuthn authentication ceremony started")
}

// WebAuthnFinishAuthentication completes a passkey assertion ceremony.
//
// POST /mfa/webauthn/auth/finish
func (h *MFAHandler) WebAuthnFinishAuthentication(w http.ResponseWriter, r *http.Request) {
	user := middleware.AuthFromRequest(r).User
	if user == nil {
		resp.Error(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	parsedResponse, err := protocol.ParseCredentialRequestResponseBody(r.Body)
	if err != nil {
		resp.Error(w, http.StatusBadRequest, "Invalid WebAuthn assertion response")
		return
	}

	cred, svcErr := h.webAuthnSvc.FinishAuthentication(r.Context(), user.UserID, parsedResponse)
	if svcErr != nil {
		resp.HandleServiceError(w, r, "WebAuthn authentication failed", svcErr)
		return
	}

	resp.Success(w, WebAuthnCredentialSummaryDTO{
		CredentialUUID: cred.CredentialUUID.String(),
		Name:           cred.Name,
	}, "WebAuthn authentication succeeded")
}

// WebAuthnDeleteCredential removes a registered passkey.
//
// DELETE /mfa/webauthn/{credential_uuid}
func (h *MFAHandler) WebAuthnDeleteCredential(w http.ResponseWriter, r *http.Request) {
	user := middleware.AuthFromRequest(r).User
	if user == nil {
		resp.Error(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	credUUID := chi.URLParam(r, "credential_uuid")

	if err := h.webAuthnSvc.DeleteCredential(r.Context(), credUUID, user.UserID); err != nil {
		resp.HandleServiceError(w, r, "Failed to delete credential", err)
		return
	}

	resp.Success(w, nil, "Credential deleted")
}

// ──────────────────────────────────────────────────────────────────────────────
// Step-up authentication
// ──────────────────────────────────────────────────────────────────────────────

// IssueStepUpChallenge issues a short-lived challenge token for step-up auth.
//
// POST /mfa/step-up/challenge
func (h *MFAHandler) IssueStepUpChallenge(w http.ResponseWriter, r *http.Request) {
	user := middleware.AuthFromRequest(r).User
	if user == nil {
		resp.Error(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	result, err := h.mfaSvc.IssueStepUpChallenge(r.Context(), user.UserUUID.String(), []string{"totp", "backup_code", "webauthn"})
	if err != nil {
		resp.HandleServiceError(w, r, "Failed to issue step-up challenge", err)
		return
	}

	resp.Success(w, result, "Step-up challenge issued")
}

// VerifyStepUp verifies a step-up factor and returns an elevated access token.
//
// POST /mfa/step-up/verify
func (h *MFAHandler) VerifyStepUp(w http.ResponseWriter, r *http.Request) {
	user := middleware.AuthFromRequest(r).User
	if user == nil {
		resp.Error(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	var req StepUpVerifyRequestDTO
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		resp.BadRequestBody(w)
		return
	}

	result, err := h.mfaSvc.VerifyStepUp(r.Context(), req, user.UserID)
	if err != nil {
		resp.HandleServiceError(w, r, "Step-up verification failed", err)
		return
	}

	resp.Success(w, result, "Step-up authentication succeeded")
}

// ──────────────────────────────────────────────────────────────────────────────
// Admin
// ──────────────────────────────────────────────────────────────────────────────

// AdminResetMFA resets all MFA factors for a target user (admin only).
//
// POST /mfa/admin/users/{user_uuid}/reset
func (h *MFAHandler) AdminResetMFA(w http.ResponseWriter, r *http.Request) {
	actor := middleware.AuthFromRequest(r).User
	if actor == nil {
		resp.Error(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	targetUUID := chi.URLParam(r, "user_uuid")

	if err := h.mfaSvc.AdminResetMFA(r.Context(), targetUUID, actor.UserID); err != nil {
		resp.HandleServiceError(w, r, "Failed to reset MFA", err)
		return
	}

	resp.Success(w, nil, "MFA reset successfully")
}
