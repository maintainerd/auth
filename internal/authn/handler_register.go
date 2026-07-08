package authn

import (
	"encoding/json"
	"net/http"
	"time"

	resp "github.com/maintainerd/maintainerd-auth/internal/platform/response"
	"github.com/maintainerd/maintainerd-auth/internal/platform/security"
	"github.com/maintainerd/maintainerd-auth/internal/platform/signedurl"
)

type RegisterHandler struct {
	registerService RegisterService
}

func NewRegisterHandler(
	registerService RegisterService,
) *RegisterHandler {
	return &RegisterHandler{
		registerService: registerService,
	}
}

func (h *RegisterHandler) RegisterPublic(w http.ResponseWriter, r *http.Request) {
	startTime := time.Now()
	sc := extractSecurityContext(r)
	clientIPStr, userAgentStr, requestIDStr := sc.clientIP, sc.userAgent, sc.requestID

	q := RegisterQueryDTO{
		ClientID: r.URL.Query().Get("client_id"),
		TenantID: r.URL.Query().Get("tenant_id"),
	}
	clientIDPtr, tenantIDPtr, ok := authenticationContextQuery(r)
	if !ok {
		resp.Error(w, http.StatusBadRequest, "Public registration requires client_id and does not accept tenant_id")
		return
	}
	if err := q.Validate(); err != nil {
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
		resp.BadRequest(w)
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
		resp.BadRequestBody(w)
		return
	}

	// Validate using DTO convention for registration (includes sanitization and password strength)
	if err := req.ValidateForRegistration(); err != nil {
		eventType := "registration_validation_failure"
		severity := "MEDIUM"
		if security.IsPasswordStrengthError(err) {
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

	// Public registration attempt (client_id required; tenant_id rejected).
	ctx := contextWithRegistrationCaptchaToken(r.Context(), req.CaptchaToken)
	tokenResponse, err := h.registerService.RegisterPublic(
		ctx, req.Username, req.Fullname, req.Password, req.Email, req.Phone, clientIDPtr, tenantIDPtr, q.RegistrationFlow,
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

	// The verification email (when the tenant policy requires it) is sent by the
	// registration service inside its transaction — see RegisterService.RegisterPublic.
	// Sending it here as well would issue a second OTP and revoke the first.

	// Response with optional cookie delivery based on X-Token-Delivery header
	resp.CreatedWithCookies(w, r, tokenResponse, "Registration successful")
}

func (h *RegisterHandler) Register(w http.ResponseWriter, r *http.Request) {
	startTime := time.Now()
	sc := extractSecurityContext(r)
	clientIPStr, userAgentStr, requestIDStr := sc.clientIP, sc.userAgent, sc.requestID

	tenantID := r.URL.Query().Get("tenant_id")
	if tenantID == "" || r.URL.Query().Get("client_id") != "" {
		resp.Error(w, http.StatusBadRequest, "Internal registration requires tenant_id and does not accept client_id")
		return
	}
	tenantIDPtr := &tenantID

	// Validate body payload
	var req RegisterRequestDTO
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		resp.BadRequestBody(w)
		return
	}

	// Validate using DTO convention for registration (includes sanitization and password strength)
	if err := req.ValidateForRegistration(); err != nil {
		eventType := "registration_validation_failure"
		severity := "MEDIUM"
		if security.IsPasswordStrengthError(err) {
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

	// Internal registration attempt (client_id/tenant_id optional)
	ctx := contextWithRegistrationCaptchaToken(r.Context(), req.CaptchaToken)
	tokenResponse, err := h.registerService.Register(
		ctx, req.Username, req.Fullname, req.Password, req.Email, req.Phone, nil, tenantIDPtr, r.URL.Query().Get("registration_flow"),
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

func (h *RegisterHandler) RegisterInvitePublic(w http.ResponseWriter, r *http.Request) {
	// Validate query parameters
	q := RegisterInviteQueryDTO{
		ClientID:    r.URL.Query().Get("client_id"),
		TenantID:    r.URL.Query().Get("tenant_id"),
		InviteToken: r.URL.Query().Get("invite_token"),
		Expires:     r.URL.Query().Get("expires"),
		Sig:         r.URL.Query().Get("sig"),
	}
	if _, _, ok := authenticationContextQuery(r); !ok {
		resp.Error(w, http.StatusBadRequest, "Public invite registration requires client_id and does not accept tenant_id")
		return
	}

	if _, err := signedurl.ValidateSignedURL(r.URL.Query()); err != nil {
		resp.ValidationError(w, err)
		return
	}

	if err := q.Validate(); err != nil {
		resp.ValidationError(w, err)
		return
	}

	// Validate body payload
	var req LoginRequestDTO
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		resp.BadRequestBody(w)
		return
	}

	if err := req.Validate(); err != nil {
		resp.ValidationError(w, err)
		return
	}

	// Public register with invite (client_id only; tenant_id rejected by guard above)
	tokenResponse, err := h.registerService.RegisterInvitePublic(
		r.Context(),
		req.Username,
		req.Password,
		q.ClientID,
		"",
		q.InviteToken,
	)
	if err != nil {
		resp.HandleServiceError(w, r, "Registration failed", err)
		return
	}

	// Response with optional cookie delivery based on X-Token-Delivery header
	resp.CreatedWithCookies(w, r, tokenResponse, "Registration successful")
}

func (h *RegisterHandler) RegisterInvite(w http.ResponseWriter, r *http.Request) {
	tenantID := r.URL.Query().Get("tenant_id")
	if tenantID == "" || r.URL.Query().Get("client_id") != "" {
		resp.Error(w, http.StatusBadRequest, "Internal invite registration requires tenant_id and does not accept client_id")
		return
	}
	tenantIDPtr := &tenantID

	// Validate query parameters (client_id/tenant_id optional for internal)
	q := RegisterInviteQueryDTO{
		ClientID:    r.URL.Query().Get("client_id"),
		TenantID:    r.URL.Query().Get("tenant_id"),
		InviteToken: r.URL.Query().Get("invite_token"),
		Expires:     r.URL.Query().Get("expires"),
		Sig:         r.URL.Query().Get("sig"),
	}

	if _, err := signedurl.ValidateSignedURL(r.URL.Query()); err != nil {
		resp.ValidationError(w, err)
		return
	}

	if err := q.ValidateInternal(); err != nil {
		resp.ValidationError(w, err)
		return
	}

	// Validate body payload
	var req LoginRequestDTO
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		resp.BadRequestBody(w)
		return
	}

	if err := req.Validate(); err != nil {
		resp.ValidationError(w, err)
		return
	}

	// Internal register with invite (client_id/tenant_id optional)
	tokenResponse, err := h.registerService.RegisterInvite(
		r.Context(),
		req.Username,
		req.Password,
		nil,
		tenantIDPtr,
		q.InviteToken,
	)
	if err != nil {
		resp.HandleServiceError(w, r, "Registration failed", err)
		return
	}

	resp.CreatedWithCookies(w, r, tokenResponse, "Registration successful")
}
