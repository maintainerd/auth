package user

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/maintainerd/maintainerd-auth/internal/auditlog"
	"github.com/maintainerd/maintainerd-auth/internal/platform/middleware"
	resp "github.com/maintainerd/maintainerd-auth/internal/platform/response"
	"github.com/maintainerd/maintainerd-auth/internal/platform/security"
)

// AccountHandler handles self-service account management operations.
type AccountHandler struct {
	accountService AccountService
	sessionService SessionService
	profileRepo    ProfileRepository
	auditLogger    auditlog.ManagementAuditLogger
}

func NewAccountHandler(accountService AccountService, sessionService SessionService, profileRepo ProfileRepository) *AccountHandler {
	return &AccountHandler{accountService: accountService, sessionService: sessionService, profileRepo: profileRepo}
}

// SetAuditLogger injects the audit logger (called by the wiring layer).
func (h *AccountHandler) SetAuditLogger(l auditlog.ManagementAuditLogger) { h.auditLogger = l }

func (h *AccountHandler) logAudit(r *http.Request, tenantID int64, actorUserID *int64, action, resourceType, resourceID string, resourceUUID *uuid.UUID, changes, outcome string) {
	if h.auditLogger == nil {
		return
	}
	_ = h.auditLogger.Log(r.Context(), auditlog.LogEntry{
		TenantID:     tenantID,
		ActorUserID:  actorUserID,
		Action:       action,
		ResourceType: resourceType,
		ResourceID:   resourceID,
		ResourceUUID: resourceUUID,
		Changes:      changes,
		Outcome:      outcome,
	})
}

// GetAccount returns consolidated user information: profile, roles, permissions, tenant.
//
// GET /account
func (h *AccountHandler) GetAccount(w http.ResponseWriter, r *http.Request) {
	auth := middleware.AuthFromRequest(r)
	if auth.User == nil {
		resp.Error(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	var roleNames []string
	permSet := map[string]bool{}

	for _, role := range auth.User.Roles {
		roleNames = append(roleNames, role.Name)
		for _, perm := range role.Permissions {
			permSet[perm.Name] = true
		}
	}

	permissions := make([]string, 0, len(permSet))
	for name := range permSet {
		permissions = append(permissions, name)
	}

	tenant := AccountTenantDTO{}
	if auth.Tenant != nil {
		tenant = AccountTenantDTO{
			UUID:        auth.Tenant.TenantUUID.String(),
			Name:        auth.Tenant.Name,
			DisplayName: auth.Tenant.DisplayName,
			// Tenant identifier was dropped; the DNS-safe name is the slug.
			Identifier: auth.Tenant.Name,
		}
	}

	var profiles []AccountProfileDTO
	if h.profileRepo != nil {
		if result, err := h.profileRepo.FindAllByUserID(ProfileRepositoryGetFilter{UserID: auth.User.UserID}); err == nil {
			for _, p := range result.Data {
				profiles = append(profiles, AccountProfileDTO{
					ProfileID:   p.ProfileUUID.String(),
					FirstName:   p.FirstName,
					LastName:    p.LastName,
					DisplayName: p.DisplayName,
					Default:     true,
				})
			}
		}
	}

	resp.Success(w, AccountResponseDTO{
		UserID:        auth.User.UserUUID.String(),
		Email:         auth.User.Email,
		Phone:         auth.User.Phone,
		EmailVerified: auth.User.IsEmailVerified,
		PhoneVerified: auth.User.IsPhoneVerified,
		Profiles:      profiles,
		Roles:         roleNames,
		Permissions:   permissions,
		Tenant:        tenant,
	}, "Account retrieved")
}

type AccountResponseDTO struct {
	UserID        string              `json:"user_id"`
	Email         string              `json:"email"`
	Phone         string              `json:"phone"`
	EmailVerified bool                `json:"email_verified"`
	PhoneVerified bool                `json:"phone_verified"`
	Profiles      []AccountProfileDTO `json:"profiles"`
	Roles         []string            `json:"roles"`
	Permissions   []string            `json:"permissions"`
	Tenant        AccountTenantDTO    `json:"tenant"`
}

type AccountProfileDTO struct {
	ProfileID   string  `json:"profile_id"`
	FirstName   string  `json:"first_name"`
	LastName    *string `json:"last_name,omitempty"`
	DisplayName *string `json:"display_name,omitempty"`
	Default     bool    `json:"default"`
}

type AccountTenantDTO struct {
	UUID        string `json:"tenant_id"`
	Name        string `json:"name"`
	DisplayName string `json:"display_name"`
	Identifier  string `json:"identifier"`
}

// InitiateEmailChange starts the email change flow by sending an OTP to the new address.
//
// POST /account/email/change
func (h *AccountHandler) InitiateEmailChange(w http.ResponseWriter, r *http.Request) {
	user := middleware.AuthFromRequest(r).User
	if user == nil {
		resp.Error(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	var req ChangeEmailRequestDTO
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		resp.BadRequestBody(w)
		return
	}
	if err := req.Validate(); err != nil {
		resp.ValidationError(w, err)
		return
	}

	if err := h.accountService.InitiateEmailChange(r.Context(), user.UserID, req.NewEmail, req.CurrentPassword); err != nil {
		resp.HandleServiceError(w, r, "Failed to initiate email change", err)
		return
	}

	tenantIDIEC := int64(0)
	if t := middleware.AuthFromRequest(r).Tenant; t != nil {
		tenantIDIEC = t.TenantID
	}
	actorUserIDIEC := &user.UserID
	changesJSONIEC, _ := json.Marshal(map[string]any{"update": map[string]any{"new_email": req.NewEmail}})
	userUUIDIEC := user.UserUUID
	h.logAudit(r, tenantIDIEC, actorUserIDIEC, "account.initiate_email_change", "account", userUUIDIEC.String(), &userUUIDIEC, string(changesJSONIEC), "success")

	resp.Success(w, nil, "Verification code sent to new email address")
}

// VerifyEmailChange confirms the OTP and applies the new email address.
//
// POST /account/email/verify
func (h *AccountHandler) VerifyEmailChange(w http.ResponseWriter, r *http.Request) {
	user := middleware.AuthFromRequest(r).User
	if user == nil {
		resp.Error(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	var req VerifyEmailChangeDTO
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		resp.BadRequestBody(w)
		return
	}
	if err := req.Validate(); err != nil {
		resp.ValidationError(w, err)
		return
	}

	if err := h.accountService.VerifyEmailChange(r.Context(), user.UserID, req.OTP); err != nil {
		resp.HandleServiceError(w, r, "Failed to verify email change", err)
		return
	}

	tenantIDVEC := int64(0)
	if t := middleware.AuthFromRequest(r).Tenant; t != nil {
		tenantIDVEC = t.TenantID
	}
	actorUserIDVEC := &user.UserID
	changesJSONVEC, _ := json.Marshal(map[string]any{"update": map[string]any{"email_changed": true}})
	userUUIDVEC := user.UserUUID
	h.logAudit(r, tenantIDVEC, actorUserIDVEC, "account.verify_email_change", "account", userUUIDVEC.String(), &userUUIDVEC, string(changesJSONVEC), "success")

	resp.Success(w, nil, "Email address updated successfully")
}

// ChangeUsername updates the authenticated user's username.
//
// PUT /account/username
func (h *AccountHandler) ChangeUsername(w http.ResponseWriter, r *http.Request) {
	user := middleware.AuthFromRequest(r).User
	if user == nil {
		resp.Error(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	var req ChangeUsernameDTO
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		resp.BadRequestBody(w)
		return
	}
	if err := req.Validate(); err != nil {
		resp.ValidationError(w, err)
		return
	}

	if err := h.accountService.ChangeUsername(r.Context(), user.UserID, req.NewUsername, req.CurrentPassword); err != nil {
		resp.HandleServiceError(w, r, "Failed to change username", err)
		return
	}

	tenantIDCU := int64(0)
	if t := middleware.AuthFromRequest(r).Tenant; t != nil {
		tenantIDCU = t.TenantID
	}
	actorUserIDCU := &user.UserID
	changesJSONCU, _ := json.Marshal(map[string]any{"update": map[string]any{"new_username": req.NewUsername}})
	userUUIDCU := user.UserUUID
	h.logAudit(r, tenantIDCU, actorUserIDCU, "account.change_username", "account", userUUIDCU.String(), &userUUIDCU, string(changesJSONCU), "success")

	resp.Success(w, nil, "Username updated successfully")
}

// ChangePassword rotates the authenticated user's own password.
//
// PUT /account/password
func (h *AccountHandler) ChangePassword(w http.ResponseWriter, r *http.Request) {
	user := middleware.AuthFromRequest(r).User
	if user == nil {
		resp.Error(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	var req ChangePasswordDTO
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		resp.BadRequestBody(w)
		return
	}
	if err := req.Validate(); err != nil {
		resp.ValidationError(w, err)
		return
	}

	// Throttle wrong-current-password attempts per user. Without this the
	// endpoint is an unmetered password-verification oracle for anyone holding a
	// stolen access token: /account is not mounted under the strict auth rate
	// limit group, only the global per-IP one.
	//
	// The key is deliberately NOT the login lockout key. Feeding this into login
	// lockout would let a stolen token lock the victim out of logging in at all,
	// turning a confidentiality problem into a denial of service.
	throttleKey := "pwdchange:" + user.UserUUID.String()
	if err := security.CheckRateLimit(throttleKey); err != nil {
		resp.Error(w, http.StatusTooManyRequests, "Too many password change attempts. Please try again later.")
		return
	}

	// The caller's own session, so it can be spared when the others are revoked.
	// Absent (no sid claim) means everything gets revoked — see the service.
	var callerSession *uuid.UUID
	if claims := middleware.JWTClaimsFromRequest(r); claims != nil && claims.SessionID != "" {
		if parsed, err := uuid.Parse(claims.SessionID); err == nil {
			callerSession = &parsed
		}
	}

	result, err := h.accountService.ChangePassword(r.Context(), user.UserID, req.CurrentPassword, req.NewPassword, callerSession)
	if err != nil {
		security.RecordFailedAttempt(throttleKey)
		resp.HandleServiceError(w, r, "Failed to change password", err)
		return
	}
	security.ResetFailedAttempts(throttleKey)

	tenantIDCP := int64(0)
	if t := middleware.AuthFromRequest(r).Tenant; t != nil {
		tenantIDCP = t.TenantID
	}
	actorUserIDCP := &user.UserID
	// The audit record carries no password material of any kind — not the value,
	// not its length, not a prefix.
	changesJSONCP, _ := json.Marshal(map[string]any{"update": map[string]any{"password_changed": true}})
	userUUIDCP := user.UserUUID
	h.logAudit(r, tenantIDCP, actorUserIDCP, "account.change_password", "account", userUUIDCP.String(), &userUUIDCP, string(changesJSONCP), "success")

	message := "Password changed successfully"
	if result.ReauthenticationRequired {
		message = "Password changed successfully. Please sign in again."
	}
	resp.Success(w, result, message)
}

// DeleteAccount permanently deletes the authenticated user's account.
//
// DELETE /account
func (h *AccountHandler) DeleteAccount(w http.ResponseWriter, r *http.Request) {
	user := middleware.AuthFromRequest(r).User
	if user == nil {
		resp.Error(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	var req AccountDeleteDTO
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		resp.BadRequestBody(w)
		return
	}
	if err := req.Validate(); err != nil {
		resp.ValidationError(w, err)
		return
	}

	if err := h.accountService.DeleteAccount(r.Context(), user.UserID, req.CurrentPassword); err != nil {
		resp.HandleServiceError(w, r, "Failed to delete account", err)
		return
	}

	tenantIDDA := int64(0)
	if t := middleware.AuthFromRequest(r).Tenant; t != nil {
		tenantIDDA = t.TenantID
	}
	actorUserIDDA := &user.UserID
	changesJSONDA, _ := json.Marshal(map[string]any{"before": map[string]any{"id": user.UserUUID.String()}})
	userUUIDDA := user.UserUUID
	h.logAudit(r, tenantIDDA, actorUserIDDA, "account.delete", "account", userUUIDDA.String(), &userUUIDDA, string(changesJSONDA), "success")

	resp.Success(w, nil, "Account deleted successfully")
}

// ExportAccountData returns all personal data for the authenticated user.
//
// GET /account/export
func (h *AccountHandler) ExportAccountData(w http.ResponseWriter, r *http.Request) {
	user := middleware.AuthFromRequest(r).User
	if user == nil {
		resp.Error(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	data, err := h.accountService.ExportAccountData(r.Context(), user.UserID)
	if err != nil {
		resp.HandleServiceError(w, r, "Failed to export account data", err)
		return
	}

	resp.Success(w, data, "Account data exported successfully")
}

// GenerateBackupCodes generates and returns 10 fresh backup codes for the authenticated user.
//
// POST /account/backup-codes
func (h *AccountHandler) GenerateBackupCodes(w http.ResponseWriter, r *http.Request) {
	user := middleware.AuthFromRequest(r).User
	if user == nil {
		resp.Error(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	result, err := h.accountService.GenerateBackupCodes(r.Context(), user.UserID)
	if err != nil {
		resp.HandleServiceError(w, r, "Failed to generate backup codes", err)
		return
	}

	tenantIDGBC := int64(0)
	if t := middleware.AuthFromRequest(r).Tenant; t != nil {
		tenantIDGBC = t.TenantID
	}
	actorUserIDGBC := &user.UserID
	changesJSONGBC, _ := json.Marshal(map[string]any{"update": map[string]any{"backup_codes_regenerated": true}})
	userUUIDGBC := user.UserUUID
	h.logAudit(r, tenantIDGBC, actorUserIDGBC, "account.generate_backup_codes", "account", userUUIDGBC.String(), &userUUIDGBC, string(changesJSONGBC), "success")

	resp.Success(w, result, "Backup codes generated. Store them somewhere safe — they will not be shown again.")
}

// VerifyBackupCode recovers account access using a backup code (unauthenticated).
//
// POST /recovery/backup-code
func (h *AccountHandler) VerifyBackupCode(w http.ResponseWriter, r *http.Request) {
	var req VerifyBackupCodeDTO
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		resp.BadRequestBody(w)
		return
	}
	if err := req.Validate(); err != nil {
		resp.ValidationError(w, err)
		return
	}

	tokens, err := h.accountService.VerifyBackupCode(r.Context(), req)
	if err != nil {
		resp.HandleServiceError(w, r, "Backup code verification failed", err)
		return
	}

	changesJSONVBC, _ := json.Marshal(map[string]any{"update": map[string]any{"backup_code_used": true}})
	h.logAudit(r, 0, nil, "account.verify_backup_code", "account", "", nil, string(changesJSONVBC), "success")

	resp.Success(w, tokens, "Account recovered successfully")
}

// SendPhoneVerification sends an SMS OTP to the given phone so the authenticated
// user can verify ownership of the number.
//
// POST /account/phone/send-verification
func (h *AccountHandler) SendPhoneVerification(w http.ResponseWriter, r *http.Request) {
	user := middleware.AuthFromRequest(r).User
	if user == nil {
		resp.Error(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	var req SendPhoneVerificationDTO
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		resp.BadRequestBody(w)
		return
	}
	if err := req.Validate(); err != nil {
		resp.ValidationError(w, err)
		return
	}

	if err := h.accountService.SendPhoneVerification(r.Context(), user.UserID, req.Phone); err != nil {
		resp.HandleServiceError(w, r, "Failed to send phone verification code", err)
		return
	}

	tenantIDSPV := int64(0)
	if t := middleware.AuthFromRequest(r).Tenant; t != nil {
		tenantIDSPV = t.TenantID
	}
	actorUserIDSPV := &user.UserID
	changesJSONSPV, _ := json.Marshal(map[string]any{"update": map[string]any{"phone": req.Phone}})
	userUUIDSPV := user.UserUUID
	h.logAudit(r, tenantIDSPV, actorUserIDSPV, "account.send_phone_verification", "account", userUUIDSPV.String(), &userUUIDSPV, string(changesJSONSPV), "success")

	resp.Success(w, nil, "Verification code sent to phone number")
}

// VerifyPhone confirms an SMS OTP and marks the authenticated user's phone verified.
//
// POST /account/phone/verify
func (h *AccountHandler) VerifyPhone(w http.ResponseWriter, r *http.Request) {
	user := middleware.AuthFromRequest(r).User
	if user == nil {
		resp.Error(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	var req VerifyPhoneDTO
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		resp.BadRequestBody(w)
		return
	}
	if err := req.Validate(); err != nil {
		resp.ValidationError(w, err)
		return
	}

	if err := h.accountService.VerifyPhone(r.Context(), user.UserID, req.Phone, req.Code); err != nil {
		resp.HandleServiceError(w, r, "Failed to verify phone number", err)
		return
	}

	tenantIDVPh := int64(0)
	if t := middleware.AuthFromRequest(r).Tenant; t != nil {
		tenantIDVPh = t.TenantID
	}
	actorUserIDVPh := &user.UserID
	changesJSONVPh, _ := json.Marshal(map[string]any{"update": map[string]any{"phone": req.Phone, "is_phone_verified": true}})
	userUUIDVPh := user.UserUUID
	h.logAudit(r, tenantIDVPh, actorUserIDVPh, "account.verify_phone", "account", userUUIDVPh.String(), &userUUIDVPh, string(changesJSONVPh), "success")

	resp.Success(w, nil, "Phone number verified successfully")
}

// ListSessions returns all active sessions for the authenticated user.
//
// GET /account/sessions
func (h *AccountHandler) ListSessions(w http.ResponseWriter, r *http.Request) {
	user := middleware.AuthFromRequest(r).User
	if user == nil {
		resp.Error(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	sessions, err := h.sessionService.ListSessions(r.Context(), user.UserID)
	if err != nil {
		resp.HandleServiceError(w, r, "Failed to list sessions", err)
		return
	}

	resp.Success(w, sessions, "Sessions retrieved successfully")
}

// RevokeSession revokes a single session by UUID for the authenticated user.
//
// DELETE /account/sessions/{session_uuid}
func (h *AccountHandler) RevokeSession(w http.ResponseWriter, r *http.Request) {
	user := middleware.AuthFromRequest(r).User
	if user == nil {
		resp.Error(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	sessionUUID, err := uuid.Parse(chi.URLParam(r, "session_uuid"))
	if err != nil {
		resp.Error(w, http.StatusBadRequest, "Invalid session UUID")
		return
	}

	if err := h.sessionService.RevokeSession(r.Context(), user.UserID, sessionUUID); err != nil {
		resp.HandleServiceError(w, r, "Failed to revoke session", err)
		return
	}

	tenantIDRS := int64(0)
	if t := middleware.AuthFromRequest(r).Tenant; t != nil {
		tenantIDRS = t.TenantID
	}
	actorUserIDRS := &user.UserID
	changesJSONRS, _ := json.Marshal(map[string]any{"update": map[string]any{"session_uuid": sessionUUID.String()}})
	sessionUUIDRef := sessionUUID
	h.logAudit(r, tenantIDRS, actorUserIDRS, "account.revoke_session", "session", sessionUUID.String(), &sessionUUIDRef, string(changesJSONRS), "success")

	resp.Success(w, nil, "Session revoked successfully")
}

// RevokeAllSessions revokes every active session for the authenticated user.
//
// DELETE /account/sessions
func (h *AccountHandler) RevokeAllSessions(w http.ResponseWriter, r *http.Request) {
	user := middleware.AuthFromRequest(r).User
	if user == nil {
		resp.Error(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	if err := h.sessionService.RevokeAllSessions(r.Context(), user.UserID, "user_revoke"); err != nil {
		resp.HandleServiceError(w, r, "Failed to revoke all sessions", err)
		return
	}

	tenantIDRAS := int64(0)
	if t := middleware.AuthFromRequest(r).Tenant; t != nil {
		tenantIDRAS = t.TenantID
	}
	actorUserIDRAS := &user.UserID
	changesJSONRAS, _ := json.Marshal(map[string]any{"update": map[string]any{"all_sessions_revoked": true}})
	userUUIDRAS := user.UserUUID
	h.logAudit(r, tenantIDRAS, actorUserIDRAS, "account.revoke_all_sessions", "account", userUUIDRAS.String(), &userUUIDRAS, string(changesJSONRAS), "success")

	resp.Success(w, nil, "All sessions revoked successfully")
}
