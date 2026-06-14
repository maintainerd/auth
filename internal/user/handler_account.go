package user

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/platform/middleware"
	resp "github.com/maintainerd/auth/internal/platform/response"
)

// AccountHandler handles self-service account management operations.
type AccountHandler struct {
	accountService AccountService
	sessionService SessionService
	profileRepo    ProfileRepository
}

func NewAccountHandler(accountService AccountService, sessionService SessionService, profileRepo ProfileRepository) *AccountHandler {
	return &AccountHandler{accountService: accountService, sessionService: sessionService, profileRepo: profileRepo}
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
			Identifier:  auth.Tenant.Identifier,
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
					Default:     p.IsDefault,
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
		Roles:           roleNames,
		Permissions:     permissions,
		Tenant:          tenant,
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

	resp.Success(w, nil, "Username updated successfully")
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

	resp.Success(w, tokens, "Account recovered successfully")
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

	if err := h.sessionService.RevokeAllSessions(r.Context(), user.UserID); err != nil {
		resp.HandleServiceError(w, r, "Failed to revoke all sessions", err)
		return
	}

	resp.Success(w, nil, "All sessions revoked successfully")
}
