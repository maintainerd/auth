package handler

import (
	"encoding/json"
	"net/http"

	"github.com/maintainerd/auth/internal/dto"
	"github.com/maintainerd/auth/internal/middleware"
	resp "github.com/maintainerd/auth/internal/rest/response"
	"github.com/maintainerd/auth/internal/service"
)

// AccountHandler handles self-service account management operations.
type AccountHandler struct {
	accountService service.AccountService
}

func NewAccountHandler(accountService service.AccountService) *AccountHandler {
	return &AccountHandler{accountService: accountService}
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

	var req dto.ChangeEmailRequestDTO
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		resp.Error(w, http.StatusBadRequest, "Invalid JSON format")
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

	var req dto.VerifyEmailChangeDTO
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		resp.Error(w, http.StatusBadRequest, "Invalid JSON format")
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

	var req dto.ChangeUsernameDTO
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		resp.Error(w, http.StatusBadRequest, "Invalid JSON format")
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

	var req dto.AccountDeleteDTO
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		resp.Error(w, http.StatusBadRequest, "Invalid JSON format")
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
	var req dto.VerifyBackupCodeDTO
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		resp.Error(w, http.StatusBadRequest, "Invalid JSON format")
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
