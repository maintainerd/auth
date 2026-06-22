package authn

import (
	"encoding/json"
	"net/http"
	"time"

	resp "github.com/maintainerd/auth/internal/platform/response"
	"github.com/maintainerd/auth/internal/platform/security"
	"github.com/maintainerd/auth/internal/platform/signedurl"
)

type MagicLinkHandler struct {
	magicLinkService MagicLinkService
}

func NewMagicLinkHandler(magicLinkService MagicLinkService) *MagicLinkHandler {
	return &MagicLinkHandler{
		magicLinkService: magicLinkService,
	}
}

// SendMagicLinkPublic handles public magic-link requests; client_id determines
// both the client and tenant.
// Mounted on the public surface (port 8081); uses the account-facing frontend hostname.
func (h *MagicLinkHandler) SendMagicLinkPublic(w http.ResponseWriter, r *http.Request) {
	startTime := time.Now()
	sc := extractSecurityContext(r)
	clientIPStr, userAgentStr, requestIDStr := sc.clientIP, sc.userAgent, sc.requestID

	clientID := r.URL.Query().Get("client_id")
	if clientID == "" || r.URL.Query().Get("tenant_id") != "" {
		security.LogSecurityEvent(security.SecurityEvent{
			EventType: "magic_link_missing_params",
			ClientIP:  clientIPStr,
			UserAgent: userAgentStr,
			RequestID: requestIDStr,
			Endpoint:  "/magic-link/send",
			Method:    r.Method,
			Timestamp: startTime,
			Details:   "Missing client_id or unexpected tenant_id parameter",
			Severity:  "MEDIUM",
		})
		resp.Error(w, http.StatusBadRequest, "Public magic-link login requires client_id and does not accept tenant_id")
		return
	}

	h.handleSendMagicLink(w, r, &clientID, nil, nil, false, startTime, sc)
}

// SendMagicLink handles internal magic-link requests; tenant_id selects the
// tenant's designated system client.
// Mounted on the management surface (port 8080); uses the auth-facing frontend hostname.
func (h *MagicLinkHandler) SendMagicLink(w http.ResponseWriter, r *http.Request) {
	startTime := time.Now()
	sc := extractSecurityContext(r)

	tenantID := r.URL.Query().Get("tenant_id")
	if tenantID == "" || r.URL.Query().Get("client_id") != "" {
		resp.Error(w, http.StatusBadRequest, "Internal magic-link login requires tenant_id and does not accept client_id")
		return
	}
	h.handleSendMagicLink(w, r, nil, nil, &tenantID, true, startTime, sc)
}

func (h *MagicLinkHandler) handleSendMagicLink(
	w http.ResponseWriter,
	r *http.Request,
	clientID, providerID, tenantID *string,
	isInternal bool,
	startTime time.Time,
	sc securityContext,
) {
	clientIPStr, userAgentStr, requestIDStr := sc.clientIP, sc.userAgent, sc.requestID

	var req SendMagicLinkRequestDTO
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		security.LogSecurityEvent(security.SecurityEvent{
			EventType: "magic_link_invalid_json",
			ClientIP:  clientIPStr,
			UserAgent: userAgentStr,
			RequestID: requestIDStr,
			Endpoint:  "/magic-link/send",
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
			EventType: "magic_link_validation_failure",
			UserID:    req.Email,
			ClientIP:  clientIPStr,
			UserAgent: userAgentStr,
			RequestID: requestIDStr,
			Endpoint:  "/magic-link/send",
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
			EventType: "magic_link_rate_limited",
			UserID:    req.Email,
			ClientIP:  clientIPStr,
			UserAgent: userAgentStr,
			RequestID: requestIDStr,
			Endpoint:  "/magic-link/send",
			Method:    r.Method,
			Timestamp: startTime,
			Details:   "Rate limit exceeded for magic link send",
			Severity:  "HIGH",
		})
		resp.Error(w, http.StatusTooManyRequests, "Too many requests. Please try again later.")
		return
	}

	var response *SendMagicLinkResponseDTO
	var err error
	if tenantID != nil {
		response, err = h.magicLinkService.SendMagicLinkForTenant(r.Context(), req.Email, *tenantID, isInternal)
	} else {
		response, err = h.magicLinkService.SendMagicLink(r.Context(), req.Email, clientID, providerID, isInternal)
	}
	if err != nil {
		resp.HandleServiceError(w, r, "Failed to send sign-in link", err)
		return
	}

	security.LogSecurityEvent(security.SecurityEvent{
		EventType: "magic_link_request",
		UserID:    req.Email,
		ClientIP:  clientIPStr,
		UserAgent: userAgentStr,
		RequestID: requestIDStr,
		Endpoint:  "/magic-link/send",
		Method:    r.Method,
		Timestamp: startTime,
		Details:   "Magic link send processed",
		Severity:  "INFO",
	})

	resp.Success(w, response, "Sign-in link sent")
}

// VerifyMagicLink consumes a magic-link token and exchanges it for a session.
// The signed URL (with signature + expiration) is validated first, then the
// magic-link token is verified. Accepts client_id or tenant_id query params
// (same pattern as login).
func (h *MagicLinkHandler) VerifyMagicLink(w http.ResponseWriter, r *http.Request) {
	startTime := time.Now()
	sc := extractSecurityContext(r)
	clientIPStr, userAgentStr, requestIDStr := sc.clientIP, sc.userAgent, sc.requestID

	signedParams, err := signedurl.ValidateSignedURL(r.URL.Query())
	if err != nil {
		security.LogSecurityEvent(security.SecurityEvent{
			EventType: "magic_link_invalid_signature",
			ClientIP:  clientIPStr,
			UserAgent: userAgentStr,
			RequestID: requestIDStr,
			Endpoint:  "/magic-link/verify",
			Method:    r.Method,
			Timestamp: startTime,
			Details:   "Invalid or expired signed URL: " + err.Error(),
			Severity:  "HIGH",
		})
		resp.Error(w, http.StatusBadRequest, "Invalid or expired magic link")
		return
	}

	token := signedParams["token"]
	if token == "" {
		resp.Error(w, http.StatusBadRequest, "Missing token parameter")
		return
	}

	var clientID, tenantID *string
	if c := signedParams["client_id"]; c != "" {
		clientID = &c
	}
	if t := signedParams["tenant_id"]; t != "" {
		tenantID = &t
	}
	if (clientID == nil) == (tenantID == nil) {
		resp.Error(w, http.StatusBadRequest, "Magic link must contain exactly one authentication context")
		return
	}

	if err := security.CheckRateLimit(clientIPStr); err != nil {
		security.LogSecurityEvent(security.SecurityEvent{
			EventType: "magic_link_rate_limited",
			ClientIP:  clientIPStr,
			UserAgent: userAgentStr,
			RequestID: requestIDStr,
			Endpoint:  "/magic-link/verify",
			Method:    r.Method,
			Timestamp: startTime,
			Details:   "Rate limit exceeded for magic link verify",
			Severity:  "HIGH",
		})
		resp.Error(w, http.StatusTooManyRequests, "Too many requests. Please try again later.")
		return
	}

	response, err := h.magicLinkService.LoginWithMagicLink(r.Context(), token, clientID, tenantID)
	if err != nil {
		security.LogSecurityEvent(security.SecurityEvent{
			EventType: "magic_link_login_failure",
			ClientIP:  clientIPStr,
			UserAgent: userAgentStr,
			RequestID: requestIDStr,
			Endpoint:  "/magic-link/verify",
			Method:    r.Method,
			Timestamp: startTime,
			Details:   "Magic link verification failed",
			Severity:  "MEDIUM",
		})
		resp.HandleServiceError(w, r, "Failed to sign in", err)
		return
	}

	if response.MFARequired {
		resp.Success(w, response, "MFA verification required")
		return
	}
	resp.SuccessWithCookies(w, r, response, "Signed in")
}

func (h *MagicLinkHandler) AdminSendMagicLink(w http.ResponseWriter, r *http.Request) {
	h.handleAdminSendMagicLink(w, r, true)
}

func (h *MagicLinkHandler) AdminSendMagicLinkPublic(w http.ResponseWriter, r *http.Request) {
	h.handleAdminSendMagicLink(w, r, false)
}

func (h *MagicLinkHandler) handleAdminSendMagicLink(w http.ResponseWriter, r *http.Request, isInternal bool) {
	var req AdminSendMagicLinkRequestDTO
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		resp.BadRequestBody(w)
		return
	}
	if req.UserUUID == "" {
		resp.Error(w, http.StatusBadRequest, "user_uuid is required")
		return
	}

	response, err := h.magicLinkService.AdminSendMagicLink(r.Context(), req.UserUUID, isInternal)
	if err != nil {
		resp.HandleServiceError(w, r, "Failed to send magic link", err)
		return
	}

	resp.Success(w, response, "Magic link sent")
}
