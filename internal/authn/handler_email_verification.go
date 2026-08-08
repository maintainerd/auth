package authn

import (
	"encoding/json"
	"net/http"
	"time"

	resp "github.com/maintainerd/maintainerd-auth/internal/platform/response"
	"github.com/maintainerd/maintainerd-auth/internal/platform/security"
)

type EmailVerificationHandler struct {
	emailVerificationService EmailVerificationService
}

func NewEmailVerificationHandler(emailVerificationService EmailVerificationService) *EmailVerificationHandler {
	return &EmailVerificationHandler{
		emailVerificationService: emailVerificationService,
	}
}

// SendVerificationEmailPublic handles client-scoped public resend requests.
func (h *EmailVerificationHandler) SendVerificationEmailPublic(w http.ResponseWriter, r *http.Request) {
	startTime := time.Now()
	sc := extractSecurityContext(r)
	clientIPStr, userAgentStr, requestIDStr := sc.clientIP, sc.userAgent, sc.requestID

	clientID, tenantID, ok := authenticationContextQuery(r)
	if !ok {
		security.LogSecurityEvent(security.SecurityEvent{
			EventType: "email_verification_missing_params",
			ClientIP:  clientIPStr,
			UserAgent: userAgentStr,
			RequestID: requestIDStr,
			Endpoint:  "/email-verification/send",
			Method:    r.Method,
			Timestamp: startTime,
			Details:   "Missing or ambiguous authentication context",
			Severity:  "MEDIUM",
		})
		resp.Error(w, http.StatusBadRequest, "Public email verification requires client_id and does not accept tenant_id")
		return
	}

	h.handleSendVerification(w, r, clientID, tenantID, startTime, sc)
}

func (h *EmailVerificationHandler) handleSendVerification(
	w http.ResponseWriter,
	r *http.Request,
	clientID, providerID *string,
	startTime time.Time,
	sc securityContext,
) {
	clientIPStr, userAgentStr, requestIDStr := sc.clientIP, sc.userAgent, sc.requestID

	var req SendEmailVerificationRequestDTO
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		security.LogSecurityEvent(security.SecurityEvent{
			EventType: "email_verification_invalid_json",
			ClientIP:  clientIPStr,
			UserAgent: userAgentStr,
			RequestID: requestIDStr,
			Endpoint:  "/email-verification/send",
			Method:    r.Method,
			Timestamp: startTime,
			Details:   "Invalid request body",
			Severity:  "MEDIUM",
		})
		resp.BadRequestBody(w)
		return
	}

	if err := req.Validate(); err != nil {
		security.LogSecurityEvent(security.SecurityEvent{
			EventType: "email_verification_validation_failure",
			UserID:    req.Email,
			ClientIP:  clientIPStr,
			UserAgent: userAgentStr,
			RequestID: requestIDStr,
			Endpoint:  "/email-verification/send",
			Method:    r.Method,
			Timestamp: startTime,
			Details:   "Request validation failed",
			Severity:  "MEDIUM",
		})
		resp.ValidationError(w, err)
		return
	}

	if err := security.CheckRateLimit(req.Email); err != nil {
		security.LogSecurityEvent(security.SecurityEvent{
			EventType: "email_verification_rate_limited",
			UserID:    req.Email,
			ClientIP:  clientIPStr,
			UserAgent: userAgentStr,
			RequestID: requestIDStr,
			Endpoint:  "/email-verification/send",
			Method:    r.Method,
			Timestamp: startTime,
			Details:   "Rate limit exceeded for email verification send",
			Severity:  "HIGH",
		})
		resp.Error(w, http.StatusTooManyRequests, "Too many requests. Please try again later.")
		return
	}

	response, err := h.emailVerificationService.SendVerificationEmail(r.Context(), req.Email, clientID, providerID) // nosemgrep
	if err != nil {
		security.LogSecurityEvent(security.SecurityEvent{
			EventType: "email_verification_service_error",
			UserID:    req.Email,
			ClientIP:  clientIPStr,
			UserAgent: userAgentStr,
			RequestID: requestIDStr,
			Endpoint:  "/email-verification/send",
			Method:    r.Method,
			Timestamp: startTime,
			Details:   "Email verification send failed",
			Severity:  "HIGH",
		})
		resp.HandleServiceError(w, r, "Failed to send verification email", err)
		return
	}

	security.LogSecurityEvent(security.SecurityEvent{
		EventType: "email_verification_request",
		UserID:    req.Email,
		ClientIP:  clientIPStr,
		UserAgent: userAgentStr,
		RequestID: requestIDStr,
		Endpoint:  "/email-verification/send",
		Method:    r.Method,
		Timestamp: startTime,
		Details:   "Email verification send processed",
		Severity:  "INFO",
	})

	resp.Success(w, response, "Verification email sent")
}

func (h *EmailVerificationHandler) VerifyEmailPublic(w http.ResponseWriter, r *http.Request) {
	clientID, tenantID, ok := authenticationContextQuery(r)
	if !ok {
		resp.Error(w, http.StatusBadRequest, "Public email verification requires client_id and does not accept tenant_id")
		return
	}
	h.verifyEmail(w, r, clientID, tenantID)
}

func (h *EmailVerificationHandler) verifyEmail(w http.ResponseWriter, r *http.Request, clientID, tenantID *string) {
	startTime := time.Now()
	sc := extractSecurityContext(r)
	clientIPStr, userAgentStr, requestIDStr := sc.clientIP, sc.userAgent, sc.requestID

	var req VerifyEmailRequestDTO
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		security.LogSecurityEvent(security.SecurityEvent{
			EventType: "email_verification_invalid_json",
			ClientIP:  clientIPStr,
			UserAgent: userAgentStr,
			RequestID: requestIDStr,
			Endpoint:  "/email-verification/verify",
			Method:    r.Method,
			Timestamp: startTime,
			Details:   "Invalid request body",
			Severity:  "MEDIUM",
		})
		resp.BadRequestBody(w)
		return
	}

	if err := req.Validate(); err != nil {
		security.LogSecurityEvent(security.SecurityEvent{
			EventType: "email_verification_validation_failure",
			UserID:    req.Email,
			ClientIP:  clientIPStr,
			UserAgent: userAgentStr,
			RequestID: requestIDStr,
			Endpoint:  "/email-verification/verify",
			Method:    r.Method,
			Timestamp: startTime,
			Details:   "Request validation failed",
			Severity:  "MEDIUM",
		})
		resp.ValidationError(w, err)
		return
	}

	if err := security.CheckRateLimit(req.Email); err != nil {
		security.LogSecurityEvent(security.SecurityEvent{
			EventType: "email_verification_rate_limited",
			UserID:    req.Email,
			ClientIP:  clientIPStr,
			UserAgent: userAgentStr,
			RequestID: requestIDStr,
			Endpoint:  "/email-verification/verify",
			Method:    r.Method,
			Timestamp: startTime,
			Details:   "Rate limit exceeded for email verification consume",
			Severity:  "HIGH",
		})
		resp.Error(w, http.StatusTooManyRequests, "Too many requests. Please try again later.")
		return
	}

	response, err := h.emailVerificationService.VerifyEmail(r.Context(), req.Email, req.OTP, clientID, tenantID) // nosemgrep
	if err != nil {
		resp.HandleServiceError(w, r, "Failed to verify email", err)
		return
	}

	resp.Success(w, response, "Email verified")
}
