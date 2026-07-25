package authn

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"time"

	"github.com/maintainerd/maintainerd-auth/internal/platform/middleware"
	resp "github.com/maintainerd/maintainerd-auth/internal/platform/response"
	"github.com/maintainerd/maintainerd-auth/internal/platform/security"
	"github.com/maintainerd/maintainerd-auth/internal/platform/signedurl"
)

type ResetPasswordHandler struct {
	resetPasswordService ResetPasswordService
}

func NewResetPasswordHandler(resetPasswordService ResetPasswordService) *ResetPasswordHandler {
	return &ResetPasswordHandler{
		resetPasswordService: resetPasswordService,
	}
}

func (h *ResetPasswordHandler) ResetPasswordPublic(w http.ResponseWriter, r *http.Request) {
	startTime := time.Now()

	// Extract security context
	clientIP := r.Context().Value(middleware.ClientIPKey)
	userAgent := r.Context().Value(middleware.UserAgentKey)
	requestID := r.Context().Value(middleware.RequestIDKey)

	// Convert context values to strings safely
	clientIPStr := ""
	userAgentStr := ""
	requestIDStr := ""

	if clientIP != nil {
		clientIPStr = clientIP.(string)
	}
	if userAgent != nil {
		userAgentStr = userAgent.(string)
	}
	if requestID != nil {
		requestIDStr = requestID.(string)
	}

	// Validate signed URL parameters first (security critical)
	signedParams, err := signedurl.ValidateSignedURL(r.URL.Query())
	if err != nil {
		security.LogSecurityEvent(security.SecurityEvent{
			EventType: "reset_password_invalid_signature",
			ClientIP:  clientIPStr,
			UserAgent: userAgentStr,
			RequestID: requestIDStr,
			Endpoint:  "/reset-password",
			Method:    r.Method,
			Timestamp: startTime,
			Details:   "Invalid signed URL",
			Severity:  "HIGH",
		})
		resp.Error(w, http.StatusBadRequest, "Invalid or expired reset link")
		return
	}

	// Extract validated parameters from signed URL
	clientID := signedParams["client_id"]
	tenantID := signedParams["tenant_id"]
	urlToken := signedParams["token"]

	if clientID == "" || tenantID != "" || urlToken == "" {
		security.LogSecurityEvent(security.SecurityEvent{
			EventType: "reset_password_missing_signed_params",
			ClientIP:  clientIPStr,
			UserAgent: userAgentStr,
			RequestID: requestIDStr,
			Endpoint:  "/reset-password",
			Method:    r.Method,
			Timestamp: startTime,
			Details:   "Missing required parameters in signed URL",
			Severity:  "HIGH",
		})
		resp.Error(w, http.StatusBadRequest, "Invalid reset link parameters")
		return
	}

	// Parse request body
	var req ResetPasswordRequestDTO
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		security.LogSecurityEvent(security.SecurityEvent{
			EventType: "reset_password_invalid_json",
			ClientIP:  clientIPStr,
			UserAgent: userAgentStr,
			RequestID: requestIDStr,
			Endpoint:  "/reset-password",
			Method:    r.Method,
			Timestamp: startTime,
			Details:   "Invalid request body",
			Severity:  "MEDIUM",
		})
		resp.BadRequestBody(w)
		return
	}

	// Validate request
	if err := req.Validate(); err != nil {
		security.LogSecurityEvent(security.SecurityEvent{
			EventType: "reset_password_validation_failure",
			UserID:    resetTokenLogRef(urlToken),
			ClientIP:  clientIPStr,
			UserAgent: userAgentStr,
			RequestID: requestIDStr,
			Endpoint:  "/reset-password",
			Method:    r.Method,
			Timestamp: startTime,
			Details:   "Request validation failed",
			Severity:  "MEDIUM",
		})
		resp.ValidationError(w, err)
		return
	}

	// Use token from signed URL (more secure) - no fallback needed for public route
	token := urlToken

	// Rate limiting check to prevent abuse
	if err := security.CheckRateLimit(token); err != nil {
		security.LogSecurityEvent(security.SecurityEvent{
			EventType: "reset_password_rate_limited",
			UserID:    resetTokenLogRef(token),
			ClientIP:  clientIPStr,
			UserAgent: userAgentStr,
			RequestID: requestIDStr,
			Endpoint:  "/reset-password",
			Method:    r.Method,
			Timestamp: startTime,
			Details:   "Rate limit exceeded for password reset",
			Severity:  "HIGH",
		})
		resp.Error(w, http.StatusTooManyRequests, "Too many requests. Please try again later.")
		return
	}

	// Process reset password request
	response, err := h.resetPasswordService.ResetPassword(r.Context(), token, req.NewPassword, stringPtrOrNil(clientID), nil)
	if err != nil {
		security.LogSecurityEvent(security.SecurityEvent{
			EventType: "reset_password_service_error",
			UserID:    resetTokenLogRef(token),
			ClientIP:  clientIPStr,
			UserAgent: userAgentStr,
			RequestID: requestIDStr,
			Endpoint:  "/reset-password",
			Method:    r.Method,
			Timestamp: startTime,
			Details:   "Password reset failed",
			Severity:  "HIGH",
		})
		resp.HandleServiceError(w, r, "Failed to reset password", err)
		return
	}

	// authevent.Log successful password reset
	security.LogSecurityEvent(security.SecurityEvent{
		EventType: "reset_password_success",
		UserID:    resetTokenLogRef(token),
		ClientIP:  clientIPStr,
		UserAgent: userAgentStr,
		RequestID: requestIDStr,
		Endpoint:  "/reset-password",
		Method:    r.Method,
		Timestamp: startTime,
		Details:   "Password reset completed successfully",
		Severity:  "INFO",
	})

	resp.Success(w, response, "Password reset successfully")
}

func (h *ResetPasswordHandler) ResetPassword(w http.ResponseWriter, r *http.Request) {
	startTime := time.Now()

	// Extract security context
	clientIP := r.Context().Value(middleware.ClientIPKey)
	userAgent := r.Context().Value(middleware.UserAgentKey)
	requestID := r.Context().Value(middleware.RequestIDKey)

	// Convert context values to strings safely
	clientIPStr := ""
	userAgentStr := ""
	requestIDStr := ""

	if clientIP != nil {
		clientIPStr = clientIP.(string)
	}
	if userAgent != nil {
		userAgentStr = userAgent.(string)
	}
	if requestID != nil {
		requestIDStr = requestID.(string)
	}

	// Always require signed URL validation for reset password
	signedParams, err := signedurl.ValidateSignedURL(r.URL.Query())
	if err != nil {
		security.LogSecurityEvent(security.SecurityEvent{
			EventType: "reset_password_invalid_signature",
			ClientIP:  clientIPStr,
			UserAgent: userAgentStr,
			RequestID: requestIDStr,
			Endpoint:  "/reset-password",
			Method:    r.Method,
			Timestamp: startTime,
			Details:   "Invalid signed URL",
			Severity:  "HIGH",
		})
		resp.Error(w, http.StatusBadRequest, "Invalid or expired reset link")
		return
	}

	// Extract validated parameters from signed URL
	tenantID := signedParams["tenant_id"]
	token := signedParams["token"]

	if token == "" || tenantID == "" || signedParams["client_id"] != "" {
		security.LogSecurityEvent(security.SecurityEvent{
			EventType: "reset_password_missing_token",
			ClientIP:  clientIPStr,
			UserAgent: userAgentStr,
			RequestID: requestIDStr,
			Endpoint:  "/reset-password",
			Method:    r.Method,
			Timestamp: startTime,
			Details:   "Missing token in signed URL",
			Severity:  "HIGH",
		})
		resp.Error(w, http.StatusBadRequest, "Invalid reset link")
		return
	}

	// Parse request body
	var req ResetPasswordRequestDTO
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		security.LogSecurityEvent(security.SecurityEvent{
			EventType: "reset_password_invalid_json",
			ClientIP:  clientIPStr,
			UserAgent: userAgentStr,
			RequestID: requestIDStr,
			Endpoint:  "/reset-password",
			Method:    r.Method,
			Timestamp: startTime,
			Details:   "Invalid request body",
			Severity:  "MEDIUM",
		})
		resp.BadRequestBody(w)
		return
	}

	// Validate request
	if err := req.Validate(); err != nil {
		security.LogSecurityEvent(security.SecurityEvent{
			EventType: "reset_password_validation_failure",
			UserID:    resetTokenLogRef(token),
			ClientIP:  clientIPStr,
			UserAgent: userAgentStr,
			RequestID: requestIDStr,
			Endpoint:  "/reset-password",
			Method:    r.Method,
			Timestamp: startTime,
			Details:   "Request validation failed",
			Severity:  "MEDIUM",
		})
		resp.ValidationError(w, err)
		return
	}

	// Token is always from signed URL - no need to validate against request body

	// Rate limiting check to prevent abuse
	if err := security.CheckRateLimit(token); err != nil {
		security.LogSecurityEvent(security.SecurityEvent{
			EventType: "reset_password_rate_limited",
			UserID:    resetTokenLogRef(token),
			ClientIP:  clientIPStr,
			UserAgent: userAgentStr,
			RequestID: requestIDStr,
			Endpoint:  "/reset-password",
			Method:    r.Method,
			Timestamp: startTime,
			Details:   "Rate limit exceeded for password reset",
			Severity:  "HIGH",
		})
		resp.Error(w, http.StatusTooManyRequests, "Too many requests. Please try again later.")
		return
	}

	// Process reset password request
	response, err := h.resetPasswordService.ResetPassword(r.Context(), token, req.NewPassword, nil, &tenantID)
	if err != nil {
		security.LogSecurityEvent(security.SecurityEvent{
			EventType: "reset_password_service_error",
			UserID:    resetTokenLogRef(token),
			ClientIP:  clientIPStr,
			UserAgent: userAgentStr,
			RequestID: requestIDStr,
			Endpoint:  "/reset-password",
			Method:    r.Method,
			Timestamp: startTime,
			Details:   "Password reset failed",
			Severity:  "HIGH",
		})
		resp.HandleServiceError(w, r, "Failed to reset password", err)
		return
	}

	// authevent.Log successful password reset
	security.LogSecurityEvent(security.SecurityEvent{
		EventType: "reset_password_success",
		UserID:    resetTokenLogRef(token),
		ClientIP:  clientIPStr,
		UserAgent: userAgentStr,
		RequestID: requestIDStr,
		Endpoint:  "/reset-password",
		Method:    r.Method,
		Timestamp: startTime,
		Details:   "Password reset completed successfully",
		Severity:  "INFO",
	})

	resp.Success(w, response, "Password reset successfully")
}

// resetTokenLogRef returns a short, non-reversible reference for a reset token.
//
// The raw token is a BEARER CREDENTIAL. Logging it — as this file did in four
// places — hands anyone with log or SIEM access a working account-takeover token
// while it is still valid. A truncated hash keeps log lines correlatable for a
// single attempt without being usable to perform the reset.
func resetTokenLogRef(token string) string {
	if token == "" {
		return "none"
	}
	sum := sha256.Sum256([]byte(token))
	return "reset:" + hex.EncodeToString(sum[:4])
}
