package mfa

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-webauthn/webauthn/protocol"
	"github.com/maintainerd/maintainerd-auth/internal/platform/jwt"
	"github.com/maintainerd/maintainerd-auth/internal/platform/middleware"
	resp "github.com/maintainerd/maintainerd-auth/internal/platform/response"
)

// MFAHandler handles all MFA self-service endpoints (TOTP, backup codes, WebAuthn, step-up).
type MFAHandler struct {
	mfaSvc      MFAService
	webAuthnSvc WebAuthnService
}

var (
	parseWebAuthnCreationResponse = protocol.ParseCredentialCreationResponseBody
	parseWebAuthnRequestResponse  = protocol.ParseCredentialRequestResponseBody
)

func NewMFAHandler(mfaSvc MFAService, webAuthnSvc WebAuthnService) *MFAHandler {
	return &MFAHandler{mfaSvc: mfaSvc, webAuthnSvc: webAuthnSvc}
}

// RequireFreshStepUp gates a user's *own* destructive MFA self-service actions
// (download/delete a passkey, regenerate backup codes, disable TOTP/SMS/email
// OTP, self-reset every factor). It demands that THIS CALLER has completed a
// step-up: acr=2 within the tenant's step-up freshness window.
//
// It replaces a guard that also passed the request when the ACCOUNT merely had
// some factor enrolled. That conflated "this account owns a second factor" with
// "the caller in front of us just proved possession of one", so a single stolen
// acr=1 session cookie was enough to POST /mfa/reset (wiping TOTP, passkeys,
// SMS, email OTP and backup codes) or to mint ten fresh backup codes from
// /mfa/backup-codes/regenerate — MFA permanently defeated from a single-factor
// foothold. Enrollment routes take RequireStepUpForNewFactor instead — the same
// demand, but skipped while the account holds no factor, so a user with nothing
// enrolled can still bootstrap their first one. A user whose sole factor has
// since been disallowed by tenant policy (and so cannot step up) is recovered by
// an admin via /mfa/admin/users/{user_uuid}/reset.
//
// It must run after JWTAuthMiddleware and UserContextMiddleware so both the
// claims and the user context are present.
func (h *MFAHandler) RequireFreshStepUp(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user := middleware.AuthFromRequest(r).User
		if user == nil {
			resp.Error(w, http.StatusUnauthorized, "No valid authentication found")
			return
		}

		if !h.callerSteppedUp(w, r, user.UserID) {
			return
		}
		next.ServeHTTP(w, r)
	})
}

// RequireStepUpForNewFactor gates ENROLLING an MFA factor (TOTP, passkey, SMS,
// email OTP) on the caller having already stepped up — but only once the
// account holds at least one factor.
//
// Adding a factor is a credential-granting write, not a read: enrollment used to
// need nothing but "account:mfa:enroll:self", which every registered user holds.
// A hijacked acr=1 session could therefore enrol the ATTACKER's own authenticator
// on the victim's account, POST /mfa/step-up/verify with it to reach acr=2, and
// from there walk straight through every step-up gate on the account —
// /mfa/reset, backup-code regeneration, email/password change. The destructive
// siblings were already guarded; leaving the additive side open handed an
// attacker the key to the same doors.
//
// The gate is conditional on purpose. An account with NO factor cannot step up,
// so its FIRST enrollment must be reachable from an acr=1 session or nobody
// could ever turn MFA on. A user who has lost their only factor is recovered by
// an admin via /mfa/admin/users/{user_uuid}/reset, exactly as for the
// destructive routes.
//
// A failure to determine the account's MFA state fails CLOSED with 503: an
// unreadable enrollment state is not proof the account is unprotected.
//
// It must run after JWTAuthMiddleware and UserContextMiddleware so both the
// claims and the user context are present.
func (h *MFAHandler) RequireStepUpForNewFactor(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user := middleware.AuthFromRequest(r).User
		if user == nil {
			resp.Error(w, http.StatusUnauthorized, "No valid authentication found")
			return
		}

		enrolled, err := h.mfaSvc.UserHasMFA(r.Context(), user.UserID)
		if err != nil {
			resp.Error(w, http.StatusServiceUnavailable, "Could not verify the account's MFA state; please retry")
			return
		}
		if !enrolled {
			// Bootstrap the first factor: there is nothing to step up with yet.
			next.ServeHTTP(w, r)
			return
		}

		if !h.callerSteppedUp(w, r, user.UserID) {
			return
		}
		next.ServeHTTP(w, r)
	})
}

// callerSteppedUp reports whether THIS caller's token proves a step-up that is
// still inside the tenant's freshness window, writing the 403 itself when it
// does not. Shared by every step-up gate in this package so they cannot drift
// apart on what "fresh" means.
func (h *MFAHandler) callerSteppedUp(w http.ResponseWriter, r *http.Request, userID int64) bool {
	claims := middleware.JWTClaimsFromRequest(r)
	if claims == nil || claims.ACR != jwt.ACRLevel2 {
		resp.ErrorWithCode(w, http.StatusForbidden, "step_up_required", "Step-up authentication required")
		return false
	}
	ttl := h.mfaSvc.StepUpTTLSeconds(r.Context(), userID)
	if claims.Iat > 0 && time.Now().Unix()-claims.Iat > ttl {
		resp.ErrorWithCode(w, http.StatusForbidden, "step_up_required", "Step-up authentication has expired; please re-authenticate")
		return false
	}
	return true
}

// RequirePolicyStepUp gates a sensitive self-service action (e.g. email change)
// on a fresh step-up ONLY when the user's tenant policy
// require_mfa_for_sensitive_actions is enabled AND the user has an enrolled MFA
// factor. When the policy is off, or the user has no MFA enrolled, the request
// passes through unchanged — so password-only users are never blocked. When the
// gate applies, it enforces the same acr=2 + 5-minute freshness window as
// middleware.RequireStepUp. It must run after JWTAuthMiddleware and
// UserContextMiddleware so both the claims and the user context are present.
func (h *MFAHandler) RequirePolicyStepUp(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user := middleware.AuthFromRequest(r).User
		if user == nil {
			resp.Error(w, http.StatusUnauthorized, "No valid authentication found")
			return
		}

		required, err := h.mfaSvc.SensitiveActionStepUpRequired(r.Context(), user.UserID)
		if err != nil {
			// Fail CLOSED. This used to share the `!required` branch, so a
			// transient failure reading require_mfa_for_sensitive_actions
			// silently dropped the step-up gate entirely — a policy that is
			// unreadable is not a policy that is off. 503 rather than
			// step_up_required because the blocker is server-side: telling a
			// password-only user to step up would send them to a challenge they
			// cannot complete.
			resp.Error(w, http.StatusServiceUnavailable, "Could not verify the step-up policy; please retry")
			return
		}
		if !required {
			// Not applicable (policy off, or no MFA enrolled): do not block, so
			// password-only flows stay usable.
			next.ServeHTTP(w, r)
			return
		}

		if !h.callerSteppedUp(w, r, user.UserID) {
			return
		}
		next.ServeHTTP(w, r)
	})
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
	if allowed, err := h.mfaSvc.IsMethodAllowed(r.Context(), user.UserID, "webauthn"); err != nil {
		resp.HandleServiceError(w, r, "Failed to check MFA policy", err)
		return
	} else if !allowed {
		resp.Error(w, http.StatusForbidden, "WebAuthn MFA is not permitted by tenant policy")
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
	if allowed, err := h.mfaSvc.IsMethodAllowed(r.Context(), user.UserID, "webauthn"); err != nil {
		resp.HandleServiceError(w, r, "Failed to check MFA policy", err)
		return
	} else if !allowed {
		resp.Error(w, http.StatusForbidden, "WebAuthn MFA is not permitted by tenant policy")
		return
	}

	credName := r.URL.Query().Get("name")

	parsedResponse, err := parseWebAuthnCreationResponse(r.Body)
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
		Transport:      strings.Join([]string(cred.Transport), ","),
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
	if allowed, err := h.mfaSvc.IsMethodAllowed(r.Context(), user.UserID, "webauthn"); err != nil {
		resp.HandleServiceError(w, r, "Failed to check MFA policy", err)
		return
	} else if !allowed {
		resp.Error(w, http.StatusForbidden, "WebAuthn MFA is not permitted by tenant policy")
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
	if allowed, err := h.mfaSvc.IsMethodAllowed(r.Context(), user.UserID, "webauthn"); err != nil {
		resp.HandleServiceError(w, r, "Failed to check MFA policy", err)
		return
	} else if !allowed {
		resp.Error(w, http.StatusForbidden, "WebAuthn MFA is not permitted by tenant policy")
		return
	}

	parsedResponse, err := parseWebAuthnRequestResponse(r.Body)
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

	// Reconcile recovery state: if this was the user's last primary factor,
	// purge leftover backup codes so the account has no MFA.
	if err := h.mfaSvc.SyncMFAState(r.Context(), user.UserID); err != nil {
		resp.HandleServiceError(w, r, "Failed to reconcile MFA state", err)
		return
	}

	resp.Success(w, nil, "Credential deleted")
}

func (h *MFAHandler) WebAuthnDownloadCredential(w http.ResponseWriter, r *http.Request) {
	user := middleware.AuthFromRequest(r).User
	if user == nil {
		resp.Error(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	credUUID := chi.URLParam(r, "credential_uuid")
	result, err := h.webAuthnSvc.DownloadCredential(r.Context(), credUUID, user.UserID)
	if err != nil {
		resp.HandleServiceError(w, r, "Failed to download credential", err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Disposition", `attachment; filename="passkey-`+credUUID+`.json"`)
	resp.Success(w, result, "Credential downloaded")
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

	status, err := h.mfaSvc.GetMFAStatus(r.Context(), user.UserID)
	if err != nil {
		resp.HandleServiceError(w, r, "Failed to determine step-up methods", err)
		return
	}

	methods := buildStepUpMethods(status)
	result, err := h.mfaSvc.IssueStepUpChallenge(r.Context(), user.UserUUID.String(), methods)
	if err != nil {
		resp.HandleServiceError(w, r, "Failed to issue step-up challenge", err)
		return
	}

	resp.Success(w, result, "Step-up challenge issued")
}

// SendStepUpSMS sends an SMS OTP for step-up authentication.
//
// POST /mfa/step-up/send-sms
func (h *MFAHandler) SendStepUpSMS(w http.ResponseWriter, r *http.Request) {
	user := middleware.AuthFromRequest(r).User
	if user == nil {
		resp.Error(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	if err := h.mfaSvc.SendStepUpSMS(r.Context(), user.UserID); err != nil {
		resp.HandleServiceError(w, r, "Failed to send SMS step-up code", err)
		return
	}

	resp.Success(w, nil, "SMS step-up code sent")
}

func buildStepUpMethods(status *MFAStatusResponseDTO) []string {
	methods := make([]string, 0, 4)
	if status.IsTOTPEnabled {
		methods = append(methods, "totp")
	}
	if status.IsWebAuthnEnabled {
		methods = append(methods, "webauthn")
	}
	if status.BackupCodesCount > 0 {
		methods = append(methods, "backup_code")
	}
	if status.IsSMSEnabled {
		methods = append(methods, "sms")
	}
	if status.IsEmailOTPEnabled {
		methods = append(methods, "email_otp")
	}
	return methods
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
	auth := middleware.AuthFromRequest(r)
	actor := auth.User
	if actor == nil {
		resp.Error(w, http.StatusUnauthorized, "Unauthorized")
		return
	}
	tenant := auth.Tenant
	if tenant == nil {
		resp.Error(w, http.StatusUnauthorized, "Tenant not found in context")
		return
	}

	targetUUID := chi.URLParam(r, "user_uuid")

	if err := h.mfaSvc.AdminResetMFA(r.Context(), targetUUID, actor.UserID, tenant.TenantID); err != nil {
		resp.HandleServiceError(w, r, "Failed to reset MFA", err)
		return
	}

	resp.Success(w, nil, "MFA reset successfully")
}

// AdminResetMFAMethod resets a single MFA factor for a target user (admin only).
//
// POST /mfa/admin/users/{user_uuid}/reset/{method}
// where {method} is one of: totp, webauthn, sms, backup_code.
func (h *MFAHandler) AdminResetMFAMethod(w http.ResponseWriter, r *http.Request) {
	auth := middleware.AuthFromRequest(r)
	actor := auth.User
	if actor == nil {
		resp.Error(w, http.StatusUnauthorized, "Unauthorized")
		return
	}
	tenant := auth.Tenant
	if tenant == nil {
		resp.Error(w, http.StatusUnauthorized, "Tenant not found in context")
		return
	}

	targetUUID := chi.URLParam(r, "user_uuid")
	method := chi.URLParam(r, "method")

	if err := h.mfaSvc.AdminResetMFAMethod(r.Context(), targetUUID, method, actor.UserID, tenant.TenantID); err != nil {
		resp.HandleServiceError(w, r, "Failed to reset MFA method", err)
		return
	}

	resp.Success(w, nil, "MFA method reset successfully")
}

func (h *MFAHandler) EnrollSMS(w http.ResponseWriter, r *http.Request) {
	user := middleware.AuthFromRequest(r).User
	if user == nil {
		resp.Error(w, http.StatusUnauthorized, "Unauthorized")
		return
	}
	var req SMSEnrollRequestDTO
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		resp.BadRequestBody(w)
		return
	}
	if err := h.mfaSvc.EnrollSMS(r.Context(), user.UserID, req.Phone); err != nil {
		resp.HandleServiceError(w, r, "SMS enrollment failed", err)
		return
	}
	resp.Success(w, nil, "SMS enrollment code sent")
}

func (h *MFAHandler) VerifySMS(w http.ResponseWriter, r *http.Request) {
	user := middleware.AuthFromRequest(r).User
	if user == nil {
		resp.Error(w, http.StatusUnauthorized, "Unauthorized")
		return
	}
	var req SMSVerifyRequestDTO
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		resp.BadRequestBody(w)
		return
	}
	if err := h.mfaSvc.VerifySMS(r.Context(), user.UserID, req.Phone, req.Code); err != nil {
		resp.HandleServiceError(w, r, "SMS verification failed", err)
		return
	}
	resp.Success(w, nil, "SMS MFA enabled")
}

func (h *MFAHandler) DisableSMS(w http.ResponseWriter, r *http.Request) {
	user := middleware.AuthFromRequest(r).User
	if user == nil {
		resp.Error(w, http.StatusUnauthorized, "Unauthorized")
		return
	}
	if err := h.mfaSvc.DisableSMS(r.Context(), user.UserID); err != nil {
		resp.HandleServiceError(w, r, "Failed to disable SMS MFA", err)
		return
	}
	resp.Success(w, nil, "SMS MFA disabled")
}

func (h *MFAHandler) SendStepUpEmailOTP(w http.ResponseWriter, r *http.Request) {
	user := middleware.AuthFromRequest(r).User
	if user == nil {
		resp.Error(w, http.StatusUnauthorized, "Unauthorized")
		return
	}
	if err := h.mfaSvc.SendStepUpEmailOTP(r.Context(), user.UserID); err != nil {
		resp.HandleServiceError(w, r, "Failed to send email OTP", err)
		return
	}
	resp.Success(w, nil, "Email OTP sent")
}

func (h *MFAHandler) EnrollEmailOTP(w http.ResponseWriter, r *http.Request) {
	user := middleware.AuthFromRequest(r).User
	if user == nil {
		resp.Error(w, http.StatusUnauthorized, "Unauthorized")
		return
	}
	var req EmailOTPEnrollRequestDTO
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		resp.BadRequestBody(w)
		return
	}
	if err := h.mfaSvc.EnrollEmailOTP(r.Context(), user.UserID, req.Email); err != nil {
		resp.HandleServiceError(w, r, "Failed to enroll email OTP", err)
		return
	}
	resp.Success(w, nil, "Email OTP enrollment started — check your email for the code")
}

func (h *MFAHandler) VerifyEmailOTP(w http.ResponseWriter, r *http.Request) {
	user := middleware.AuthFromRequest(r).User
	if user == nil {
		resp.Error(w, http.StatusUnauthorized, "Unauthorized")
		return
	}
	var req EmailOTPVerifyRequestDTO
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		resp.BadRequestBody(w)
		return
	}
	if err := h.mfaSvc.VerifyEmailOTP(r.Context(), user.UserID, req.Email, req.Code); err != nil {
		resp.HandleServiceError(w, r, "Failed to verify email OTP", err)
		return
	}
	resp.Success(w, nil, "Email OTP MFA enabled")
}

func (h *MFAHandler) DisableEmailOTP(w http.ResponseWriter, r *http.Request) {
	user := middleware.AuthFromRequest(r).User
	if user == nil {
		resp.Error(w, http.StatusUnauthorized, "Unauthorized")
		return
	}
	if err := h.mfaSvc.DisableEmailOTP(r.Context(), user.UserID); err != nil {
		resp.HandleServiceError(w, r, "Failed to disable email OTP MFA", err)
		return
	}
	resp.Success(w, nil, "Email OTP MFA disabled")
}

// SelfResetMFA removes every MFA factor for the authenticated user (self-service).
// The target is always the caller — the user ID comes from the session, never
// from the request — so this can only ever reset the caller's own MFA.
//
// POST /mfa/reset
func (h *MFAHandler) SelfResetMFA(w http.ResponseWriter, r *http.Request) {
	user := middleware.AuthFromRequest(r).User
	if user == nil {
		resp.Error(w, http.StatusUnauthorized, "Unauthorized")
		return
	}
	if err := h.mfaSvc.SelfResetMFA(r.Context(), user.UserID); err != nil {
		resp.HandleServiceError(w, r, "Failed to reset MFA", err)
		return
	}
	resp.Success(w, nil, "MFA reset successfully")
}
