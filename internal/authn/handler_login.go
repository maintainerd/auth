package authn

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/maintainerd/auth/internal/platform/cookie"
	"github.com/maintainerd/auth/internal/platform/middleware"
	resp "github.com/maintainerd/auth/internal/platform/response"
	"github.com/maintainerd/auth/internal/platform/security"
)

type LoginHandler struct {
	loginService LoginService
}

func NewLoginHandler(loginService LoginService) *LoginHandler {
	return &LoginHandler{
		loginService: loginService,
	}
}

// securityContext holds the values extracted from the request context that are
// set by SecurityContextMiddleware. It is used for audit logging in every handler.
type securityContext struct {
	clientIP  string
	userAgent string
	requestID string
}

// extractSecurityContext reads the security-related values that
// SecurityContextMiddleware stores in the request context.
func extractSecurityContext(r *http.Request) securityContext {
	strVal := func(key middleware.SecurityContextKey) string {
		if v := r.Context().Value(key); v != nil {
			if s, ok := v.(string); ok {
				return s
			}
		}
		return ""
	}
	return securityContext{
		clientIP:  strVal(middleware.ClientIPKey),
		userAgent: strVal(middleware.UserAgentKey),
		requestID: strVal(middleware.RequestIDKey),
	}
}

func (h *LoginHandler) LoginPublic(w http.ResponseWriter, r *http.Request) {
	startTime := time.Now()
	sc := extractSecurityContext(r)
	clientIPStr, userAgentStr, requestIDStr := sc.clientIP, sc.userAgent, sc.requestID

	// Validate query parameters
	q := LoginQueryDTO{
		ClientID: r.URL.Query().Get("client_id"),
		TenantID: r.URL.Query().Get("tenant_id"),
	}

	if err := q.Validate(); err != nil {
		security.LogSecurityEvent(security.SecurityEvent{
			EventType: "login_validation_failure",
			ClientIP:  clientIPStr,
			UserAgent: userAgentStr,
			RequestID: requestIDStr,
			Endpoint:  "/login",
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
			Endpoint:  "/login",
			Method:    r.Method,
			Timestamp: startTime,
			Details:   "Suspicious user agent detected",
			Severity:  "HIGH",
		})
		resp.BadRequest(w)
		return
	}

	// Validate body payload
	var req LoginRequestDTO
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		security.LogSecurityEvent(security.SecurityEvent{
			EventType: "login_malformed_request",
			ClientIP:  clientIPStr,
			UserAgent: userAgentStr,
			RequestID: requestIDStr,
			Endpoint:  "/login",
			Method:    r.Method,
			Timestamp: startTime,
			Details:   "Malformed JSON request body",
			Severity:  "MEDIUM",
		})
		resp.BadRequestBody(w)
		return
	}

	// Validate using DTO convention (includes sanitization)
	if err := req.Validate(); err != nil {
		security.LogSecurityEvent(security.SecurityEvent{
			EventType: "login_validation_failure",
			UserID:    req.Username,
			ClientIP:  clientIPStr,
			UserAgent: userAgentStr,
			RequestID: requestIDStr,
			Endpoint:  "/login",
			Method:    r.Method,
			Timestamp: startTime,
			Details:   "Request body validation failed",
			Severity:  "MEDIUM",
		})
		resp.ValidationError(w, err)
		return
	}

	// Public login attempt (client_id/tenant_id optional)
	ctx := contextWithTrustedDeviceToken(r.Context(), req.TrustedDeviceToken)
	var clIDPtr, tnIDPtr *string
	if q.ClientID != "" {
		clIDPtr = &q.ClientID
	}
	if q.TenantID != "" {
		tnIDPtr = &q.TenantID
	}
	tokenResponse, err := h.loginService.LoginPublic(
		ctx, req.Username, req.Password, clIDPtr, tnIDPtr,
	)
	if err != nil {
		security.LogSecurityEvent(security.SecurityEvent{
			EventType: "login_failure",
			UserID:    req.Username,
			ClientID:  q.ClientID,
			ClientIP:  clientIPStr,
			UserAgent: userAgentStr,
			RequestID: requestIDStr,
			Endpoint:  "/login",
			Method:    r.Method,
			Timestamp: startTime,
			Details:   "Authentication failed",
			Severity:  "MEDIUM",
		})
		resp.HandleServiceError(w, r, "Authentication failed", err)
		return
	}

	if tokenResponse == nil {
		resp.Error(w, http.StatusInternalServerError, "Login service returned an empty response")
		return
	}

	// authevent.Log successful login
	security.LogSecurityEvent(security.SecurityEvent{
		EventType: "login_success",
		UserID:    req.Username,
		ClientID:  q.ClientID,
		ClientIP:  clientIPStr,
		UserAgent: userAgentStr,
		RequestID: requestIDStr,
		Endpoint:  "/login",
		Method:    r.Method,
		Timestamp: startTime,
		Details:   "User successfully authenticated",
		Severity:  "LOW",
	})

	if tokenResponse.MFARequired {
		resp.Success(w, tokenResponse, "MFA verification required")
		return
	}
	if tokenResponse.RequirePasswordChange && tokenResponse.AccessToken == "" {
		resp.Success(w, tokenResponse, "Password change required")
		return
	}

	// Response with optional cookie delivery based on X-Token-Delivery header
	resp.SuccessWithCookies(w, r, tokenResponse, "Login successful")
}

// MFALoginVerify completes the login MFA second step and, on success, issues an
// acr=2 session. Works for both internal login (no client_id) and public login
// (client_id/tenant_id passed as query params).
func (h *LoginHandler) MFALoginVerify(w http.ResponseWriter, r *http.Request) {
	var req MFALoginVerifyRequestDTO
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		resp.BadRequestBody(w)
		return
	}

	clientID, tenantID := optionalClientQuery(r)
	ctx := contextWithRememberDevice(r.Context(), req.RememberDevice)
	tokenResponse, err := h.loginService.CompleteMFALogin(
		ctx, req.ChallengeToken, req.Method, req.Code, req.Assertion, clientID, tenantID,
	)
	if err != nil {
		resp.HandleServiceError(w, r, "MFA verification failed", err)
		return
	}

	resp.SuccessWithCookies(w, r, tokenResponse, "Login successful")
}

// MFALoginSendSMS sends an SMS OTP for the in-flight login MFA challenge.
func (h *LoginHandler) MFALoginSendSMS(w http.ResponseWriter, r *http.Request) {
	var req MFALoginChallengeRequestDTO
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		resp.BadRequestBody(w)
		return
	}
	if err := h.loginService.SendMFALoginSMS(r.Context(), req.ChallengeToken); err != nil {
		resp.HandleServiceError(w, r, "Failed to send SMS code", err)
		return
	}
	resp.Success(w, nil, "SMS code sent")
}

// MFALoginWebAuthnBegin starts a passkey assertion ceremony for the in-flight
// login MFA challenge and returns the assertion options.
func (h *LoginHandler) MFALoginWebAuthnBegin(w http.ResponseWriter, r *http.Request) {
	var req MFALoginChallengeRequestDTO
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		resp.BadRequestBody(w)
		return
	}
	options, err := h.loginService.BeginMFALoginWebAuthn(r.Context(), req.ChallengeToken)
	if err != nil {
		resp.HandleServiceError(w, r, "Failed to begin WebAuthn authentication", err)
		return
	}
	resp.Success(w, options, "WebAuthn authentication ceremony started")
}

// optionalClientQuery extracts client_id/tenant_id from the query string when
// present (public login); returns nils for internal login (system client).
func optionalClientQuery(r *http.Request) (clientID, tenantID *string) {
	if c := strings.TrimSpace(r.URL.Query().Get("client_id")); c != "" {
		clientID = &c
	}
	if t := strings.TrimSpace(r.URL.Query().Get("tenant_id")); t != "" {
		tenantID = &t
	}
	return clientID, tenantID
}

func (h *LoginHandler) Logout(w http.ResponseWriter, r *http.Request) {
	startTime := time.Now()
	sc := extractSecurityContext(r)
	clientIPStr, userAgentStr, requestIDStr := sc.clientIP, sc.userAgent, sc.requestID

	security.LogSecurityEvent(security.SecurityEvent{
		EventType: "logout",
		ClientIP:  clientIPStr,
		UserAgent: userAgentStr,
		RequestID: requestIDStr,
		Endpoint:  "/logout",
		Method:    r.Method,
		Timestamp: startTime,
		Details:   "User logout initiated",
		Severity:  "LOW",
	})

	accessToken := extractAccessToken(r)
	if accessToken != "" {
		if err := h.loginService.Logout(r.Context(), accessToken); err != nil {
			resp.Error(w, http.StatusInternalServerError, "Logout failed")
			return
		}
	}

	cookie.ClearAuthCookies(w)

	resp.Success(w, nil, "Logout successful")
}

func extractAccessToken(r *http.Request) string {
	for _, name := range []string{"access_token", "__Host-access_token"} {
		if cookie, err := r.Cookie(name); err == nil {
			return cookie.Value
		}
	}
	return ""
}

func extractRefreshToken(r *http.Request) string {
	for _, name := range []string{"refresh_token", "__Secure-refresh_token"} {
		if cookie, err := r.Cookie(name); err == nil {
			return cookie.Value
		}
	}
	return ""
}

// RefreshToken exchanges a refresh token for a fresh access/id/refresh token set.
//
// The refresh token is read from the JSON body ({"refresh_token": "..."}) or the
// refresh-token cookie. The session id is taken from the X-Session-ID header, or
// derived from the access_token cookie's sid claim, so the flow works for both
// bearer-token and cookie-based clients.
func (h *LoginHandler) RefreshToken(w http.ResponseWriter, r *http.Request) {
	startTime := time.Now()
	sc := extractSecurityContext(r)
	clientIPStr, userAgentStr, requestIDStr := sc.clientIP, sc.userAgent, sc.requestID

	// Refresh token: request body takes precedence, then the cookie. Track
	// whether it came from a cookie so we can deliver the rotated tokens the
	// same way (see cookie reissue below).
	var req RefreshTokenRequestDTO
	_ = json.NewDecoder(r.Body).Decode(&req) // body is optional in cookie mode
	refreshToken := strings.TrimSpace(req.RefreshToken)
	fromCookie := false
	if refreshToken == "" {
		refreshToken = extractRefreshToken(r)
		fromCookie = refreshToken != ""
	}
	if refreshToken == "" {
		resp.Error(w, http.StatusUnauthorized, "Refresh token is required")
		return
	}

	// Session id: explicit header wins, otherwise derive from the access token cookie.
	sessionID := strings.TrimSpace(r.Header.Get("X-Session-ID"))
	if sessionID == "" {
		sessionID = sessionIDFromAccessToken(extractAccessToken(r))
	}

	tokenResponse, err := h.loginService.RefreshToken(r.Context(), refreshToken, sessionID)
	if err != nil {
		security.LogSecurityEvent(security.SecurityEvent{
			EventType: "token_refresh_failure",
			ClientIP:  clientIPStr,
			UserAgent: userAgentStr,
			RequestID: requestIDStr,
			Endpoint:  "/refresh-token",
			Method:    r.Method,
			Timestamp: startTime,
			Details:   "Refresh token rejected",
			Severity:  "MEDIUM",
		})
		resp.HandleServiceError(w, r, "Token refresh failed", err)
		return
	}

	security.LogSecurityEvent(security.SecurityEvent{
		EventType: "token_refresh_success",
		ClientIP:  clientIPStr,
		UserAgent: userAgentStr,
		RequestID: requestIDStr,
		Endpoint:  "/refresh-token",
		Method:    r.Method,
		Timestamp: startTime,
		Details:   "Access token refreshed",
		Severity:  "LOW",
	})

	// Deliver the rotated tokens the same way they arrived. Because refresh
	// rotates (and revokes) the old refresh token, a cookie-based client MUST
	// receive the new cookies on this response — otherwise the browser keeps the
	// now-revoked cookie and the next refresh fails. So set cookies whenever the
	// token came from a cookie, or when the client explicitly asks for cookie
	// delivery. Cookies keep their HttpOnly/Secure/SameSite attributes.
	if fromCookie || r.Header.Get("X-Token-Delivery") == "cookie" {
		cookie.SetAuthCookies(w, tokenResponse)
	}
	resp.Success(w, tokenResponse, "Token refreshed successfully")
}

func (h *LoginHandler) Login(w http.ResponseWriter, r *http.Request) {
	startTime := time.Now()
	sc := extractSecurityContext(r)
	clientIPStr, userAgentStr, requestIDStr := sc.clientIP, sc.userAgent, sc.requestID

	// Parse optional query parameters (client_id and tenant_id)
	var clientIDPtr, tenantIDPtr *string
	if clientID := r.URL.Query().Get("client_id"); clientID != "" {
		clientIDPtr = &clientID
	}
	if tenantID := r.URL.Query().Get("tenant_id"); tenantID != "" {
		tenantIDPtr = &tenantID
	}

	// Validate body payload
	var req LoginRequestDTO
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		resp.BadRequestBody(w)
		return
	}

	// Validate using DTO convention (includes sanitization)
	if err := req.Validate(); err != nil {
		security.LogSecurityEvent(security.SecurityEvent{
			EventType: "login_validation_failure",
			UserID:    req.Username,
			ClientIP:  clientIPStr,
			UserAgent: userAgentStr,
			RequestID: requestIDStr,
			Endpoint:  "/login",
			Method:    r.Method,
			Timestamp: startTime,
			Details:   "Request body validation failed",
			Severity:  "MEDIUM",
		})
		resp.ValidationError(w, err)
		return
	}

	// Internal login attempt (client_id/tenant_id optional)
	ctx := contextWithTrustedDeviceToken(r.Context(), req.TrustedDeviceToken)
	tokenResponse, err := h.loginService.Login(
		ctx, req.Username, req.Password, clientIDPtr, tenantIDPtr,
	)
	if err != nil {
		security.LogSecurityEvent(security.SecurityEvent{
			EventType: "login_failure",
			UserID:    req.Username,
			ClientID:  "internal",
			ClientIP:  clientIPStr,
			UserAgent: userAgentStr,
			RequestID: requestIDStr,
			Endpoint:  "/login",
			Method:    r.Method,
			Timestamp: startTime,
			Details:   "Internal authentication failed",
			Severity:  "MEDIUM",
		})
		resp.HandleServiceError(w, r, "Authentication failed", err)
		return
	}

	if tokenResponse == nil {
		resp.Error(w, http.StatusInternalServerError, "Login service returned an empty response")
		return
	}

	// authevent.Log successful login
	security.LogSecurityEvent(security.SecurityEvent{
		EventType: "login_success",
		UserID:    req.Username,
		ClientID:  "internal",
		ClientIP:  clientIPStr,
		UserAgent: userAgentStr,
		RequestID: requestIDStr,
		Endpoint:  "/login",
		Method:    r.Method,
		Timestamp: startTime,
		Details:   "User successfully authenticated via internal endpoint",
		Severity:  "LOW",
	})

	if tokenResponse.MFARequired {
		resp.Success(w, tokenResponse, "MFA verification required")
		return
	}

	// Response with optional cookie delivery based on X-Token-Delivery header
	resp.SuccessWithCookies(w, r, tokenResponse, "Login successful")
}
