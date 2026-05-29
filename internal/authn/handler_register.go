package authn

import (
	"encoding/json"
	"net/http"
	"time"

	resp "github.com/maintainerd/auth/internal/platform/response"
	"github.com/maintainerd/auth/internal/platform/security"
)

type RegisterHandler struct {
	registerService          RegisterService
	emailVerificationService EmailVerificationService
}

func NewRegisterHandler(
	registerService RegisterService,
	emailVerificationService EmailVerificationService,
) *RegisterHandler {
	return &RegisterHandler{
		registerService:          registerService,
		emailVerificationService: emailVerificationService,
	}
}

func (h *RegisterHandler) RegisterPublic(w http.ResponseWriter, r *http.Request) {
	startTime := time.Now()
	sc := extractSecurityContext(r)
	clientIPStr, userAgentStr, requestIDStr := sc.clientIP, sc.userAgent, sc.requestID

	// Validate query parameters
	q := RegisterQueryDTO{
		ClientID:   r.URL.Query().Get("client_id"),
		ProviderID: r.URL.Query().Get("provider_id"),
	}

	if err := q.Validate(); err != nil {
		security.LogSecurityEvent(security.SecurityEvent{
			EventType: "registration_validation_failure",
			ClientIP:  clientIPStr,
			UserAgent: userAgentStr,
			RequestID: requestIDStr,
			Endpoint:  "/register",
			Method:    r.Method,
			Timestamp: startTime,
			Details:   "Query parameter validation failed",
			Severity:  "MEDIUM",
		})
		resp.ValidationError(w, err)
		return
	}

	// Validate User-Agent for suspicious patterns
	if !security.ValidateUserAgent(userAgentStr) {
		security.LogSecurityEvent(security.SecurityEvent{
			EventType: "suspicious_user_agent",
			ClientIP:  clientIPStr,
			UserAgent: userAgentStr,
			RequestID: requestIDStr,
			Endpoint:  "/register",
			Method:    r.Method,
			Timestamp: startTime,
			Details:   "Suspicious user agent detected",
			Severity:  "HIGH",
		})
		resp.Error(w, http.StatusBadRequest, "Invalid request")
		return
	}

	// Validate body payload
	var req RegisterRequestDTO
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		security.LogSecurityEvent(security.SecurityEvent{
			EventType: "registration_malformed_request",
			ClientIP:  clientIPStr,
			UserAgent: userAgentStr,
			RequestID: requestIDStr,
			Endpoint:  "/register",
			Method:    r.Method,
			Timestamp: startTime,
			Details:   "Malformed JSON request body",
			Severity:  "MEDIUM",
		})
		resp.Error(w, http.StatusBadRequest, "Invalid request format")
		return
	}

	// Validate using DTO convention for registration (includes sanitization and password strength)
	if err := req.ValidateForRegistration(); err != nil {
		// Determine event type based on error
		eventType := "registration_validation_failure"
		severity := "MEDIUM"
		if err.Error() == "password is too weak" ||
			err.Error() == "password must contain at least one uppercase letter" ||
			err.Error() == "password must contain at least one lowercase letter" ||
			err.Error() == "password must contain at least one digit" ||
			err.Error() == "password must contain at least one special character" ||
			err.Error() == "password contains a common weak password" {
			eventType = "registration_weak_password"
		}

		security.LogSecurityEvent(security.SecurityEvent{
			EventType: eventType,
			UserID:    req.Username,
			ClientIP:  clientIPStr,
			UserAgent: userAgentStr,
			RequestID: requestIDStr,
			Endpoint:  "/register",
			Method:    r.Method,
			Timestamp: startTime,
			Details:   "Request validation failed",
			Severity:  severity,
		})
		resp.ValidationError(w, err)
		return
	}

	// Public registration attempt (requires client_id and provider_id)
	tokenResponse, err := h.registerService.RegisterPublic(
		r.Context(), req.Username, req.Fullname, req.Password, req.Email, req.Phone, q.ClientID, q.ProviderID,
	)
	if err != nil {
		security.LogSecurityEvent(security.SecurityEvent{
			EventType: "registration_failure",
			UserID:    req.Username,
			ClientID:  q.ClientID,
			ClientIP:  clientIPStr,
			UserAgent: userAgentStr,
			RequestID: requestIDStr,
			Endpoint:  "/register",
			Method:    r.Method,
			Timestamp: startTime,
			Details:   "Registration failed",
			Severity:  "MEDIUM",
		})
		resp.HandleServiceError(w, r, "Registration failed", err)
		return
	}

	// authevent.Log successful registration
	security.LogSecurityEvent(security.SecurityEvent{
		EventType: "registration_success",
		UserID:    req.Username,
		ClientID:  q.ClientID,
		ClientIP:  clientIPStr,
		UserAgent: userAgentStr,
		RequestID: requestIDStr,
		Endpoint:  "/register",
		Method:    r.Method,
		Timestamp: startTime,
		Details:   "User successfully registered",
		Severity:  "LOW",
	})

	// Best-effort: send email verification OTP if an email was supplied.
	// Failures here must not fail the registration response.
	if req.Email != nil && *req.Email != "" && h.emailVerificationService != nil {
		clientIDPtr := q.ClientID
		providerIDPtr := q.ProviderID
		if _, sendErr := h.emailVerificationService.SendVerificationEmail(
			r.Context(), *req.Email, &clientIDPtr, &providerIDPtr,
		); sendErr != nil {
			security.LogSecurityEvent(security.SecurityEvent{
				EventType: "email_verification_send_failure",
				UserID:    req.Username,
				ClientID:  q.ClientID,
				ClientIP:  clientIPStr,
				UserAgent: userAgentStr,
				RequestID: requestIDStr,
				Endpoint:  "/register",
				Method:    r.Method,
				Timestamp: time.Now(),
				Details:   "Failed to send verification email after registration: " + sendErr.Error(),
				Severity:  "LOW",
			})
		}
	}

	// Response with optional cookie delivery based on X-Token-Delivery header
	resp.CreatedWithCookies(w, r, tokenResponse, "Registration successful")
}

func (h *RegisterHandler) Register(w http.ResponseWriter, r *http.Request) {
	startTime := time.Now()
	sc := extractSecurityContext(r)
	clientIPStr, userAgentStr, requestIDStr := sc.clientIP, sc.userAgent, sc.requestID

	// Parse optional query parameters (client_id and provider_id)
	var clientIDPtr, providerIDPtr *string
	if clientID := r.URL.Query().Get("client_id"); clientID != "" {
		clientIDPtr = &clientID
	}
	if providerID := r.URL.Query().Get("provider_id"); providerID != "" {
		providerIDPtr = &providerID
	}

	// Validate body payload
	var req RegisterRequestDTO
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		resp.Error(w, http.StatusBadRequest, "Invalid request")
		return
	}

	// Validate using DTO convention for registration (includes sanitization and password strength)
	if err := req.ValidateForRegistration(); err != nil {
		// Determine event type based on error
		eventType := "registration_validation_failure"
		severity := "MEDIUM"
		if err.Error() == "password is too weak" ||
			err.Error() == "password must contain at least one uppercase letter" ||
			err.Error() == "password must contain at least one lowercase letter" ||
			err.Error() == "password must contain at least one digit" ||
			err.Error() == "password must contain at least one special character" ||
			err.Error() == "password contains a common weak password" {
			eventType = "registration_weak_password"
		}

		security.LogSecurityEvent(security.SecurityEvent{
			EventType: eventType,
			UserID:    req.Username,
			ClientIP:  clientIPStr,
			UserAgent: userAgentStr,
			RequestID: requestIDStr,
			Endpoint:  "/register",
			Method:    r.Method,
			Timestamp: startTime,
			Details:   "Request validation failed",
			Severity:  severity,
		})
		resp.ValidationError(w, err)
		return
	}

	// Internal registration attempt (client_id/provider_id optional)
	tokenResponse, err := h.registerService.Register(
		r.Context(), req.Username, req.Fullname, req.Password, req.Email, req.Phone, clientIDPtr, providerIDPtr,
	)
	if err != nil {
		security.LogSecurityEvent(security.SecurityEvent{
			EventType: "registration_failure",
			UserID:    req.Username,
			ClientIP:  clientIPStr,
			UserAgent: userAgentStr,
			RequestID: requestIDStr,
			Endpoint:  "/register",
			Method:    r.Method,
			Timestamp: startTime,
			Details:   "Internal registration failed",
			Severity:  "MEDIUM",
		})
		resp.HandleServiceError(w, r, "Registration failed", err)
		return
	}

	// authevent.Log successful registration
	security.LogSecurityEvent(security.SecurityEvent{
		EventType: "registration_success",
		UserID:    req.Username,
		ClientIP:  clientIPStr,
		UserAgent: userAgentStr,
		RequestID: requestIDStr,
		Endpoint:  "/register",
		Method:    r.Method,
		Timestamp: startTime,
		Details:   "User successfully registered via internal endpoint",
		Severity:  "LOW",
	})

	// Response with optional cookie delivery based on X-Token-Delivery header
	resp.CreatedWithCookies(w, r, tokenResponse, "Registration successful")
}

func (h *RegisterHandler) RegisterInvite(w http.ResponseWriter, r *http.Request) {
	// Get invite token from query parameters
	inviteToken := r.URL.Query().Get("invite_token")
	if inviteToken == "" {
		resp.Error(w, http.StatusBadRequest, "Invite token is required")
		return
	}

	// Parse optional query parameters (client_id and provider_id)
	var clientIDPtr, providerIDPtr *string
	if clientID := r.URL.Query().Get("client_id"); clientID != "" {
		clientIDPtr = &clientID
	}
	if providerID := r.URL.Query().Get("provider_id"); providerID != "" {
		providerIDPtr = &providerID
	}

	// Validate body payload
	var req LoginRequestDTO
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		resp.Error(w, http.StatusBadRequest, "Invalid request")
		return
	}

	if err := req.Validate(); err != nil {
		resp.ValidationError(w, err)
		return
	}

	// Internal register with invite (client_id/provider_id optional)
	tokenResponse, err := h.registerService.RegisterInvite(
		r.Context(),
		req.Username,
		req.Password,
		inviteToken,
		clientIDPtr, providerIDPtr,
	)
	if err != nil {
		resp.HandleServiceError(w, r, "Registration failed", err)
		return
	}

	// Response with optional cookie delivery based on X-Token-Delivery header
	resp.CreatedWithCookies(w, r, tokenResponse, "Registration successful")
}

func (h *RegisterHandler) RegisterInvitePublic(w http.ResponseWriter, r *http.Request) {
	// Validate query parameters
	q := RegisterInviteQueryDTO{
		ClientID:    r.URL.Query().Get("client_id"),
		ProviderID:  r.URL.Query().Get("provider_id"),
		InviteToken: r.URL.Query().Get("invite_token"),
		Expires:     r.URL.Query().Get("expires"),
		Sig:         r.URL.Query().Get("sig"),
	}

	if err := q.Validate(); err != nil {
		resp.ValidationError(w, err)
		return
	}

	// Validate body payload
	var req LoginRequestDTO
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		resp.Error(w, http.StatusBadRequest, "Invalid request")
		return
	}

	if err := req.Validate(); err != nil {
		resp.ValidationError(w, err)
		return
	}

	// Public register with invite (requires client_id and provider_id)
	tokenResponse, err := h.registerService.RegisterInvitePublic(
		r.Context(),
		req.Username,
		req.Password,
		q.ClientID,
		q.ProviderID,
		q.InviteToken,
	)
	if err != nil {
		resp.HandleServiceError(w, r, "Registration failed", err)
		return
	}

	// Response with optional cookie delivery based on X-Token-Delivery header
	resp.CreatedWithCookies(w, r, tokenResponse, "Registration successful")
}
