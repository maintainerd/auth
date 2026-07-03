package authn

import (
	"encoding/json"
	"net/http"
	"time"

	resp "github.com/maintainerd/maintainerd-auth/internal/platform/response"
	"github.com/maintainerd/maintainerd-auth/internal/platform/security"
	"github.com/maintainerd/maintainerd-auth/internal/platform/signedurl"
)

type MagicLinkHandler struct {
	magicLinkService MagicLinkService
}

func NewMagicLinkHandler(magicLinkService MagicLinkService) *MagicLinkHandler {
	return &MagicLinkHandler{
		magicLinkService: magicLinkService,
	}
}

// SendMagicLinkPublic handles public magic-link requests. client_id identifies
// the external app client.
// Mounted on the public surface (port 8081); uses the account-facing frontend hostname.
func (h *MagicLinkHandler) SendMagicLinkPublic(w http.ResponseWriter, r *http.Request) {
	startTime := time.Now()
	sc := extractSecurityContext(r)
	clientIPStr, userAgentStr, requestIDStr := sc.clientIP, sc.userAgent, sc.requestID

	clientID, tenantID, ok := authenticationContextQuery(r)
	if !ok {
		security.LogSecurityEvent(security.SecurityEvent{
			EventType: "magic_link_missing_params",
			ClientIP:  clientIPStr,
			UserAgent: userAgentStr,
			RequestID: requestIDStr,
			Endpoint:  "/magic-link/send",
			Method:    r.Method,
			Timestamp: startTime,
			Details:   "Missing or ambiguous authentication context",
			Severity:  "MEDIUM",
		})
		resp.Error(w, http.StatusBadRequest, "Public magic-link login requires client_id and does not accept tenant_id")
		return
	}

	h.handleSendMagicLink(w, r, clientID, tenantID, startTime, sc)
}

func (h *MagicLinkHandler) handleSendMagicLink(
	w http.ResponseWriter,
	r *http.Request,
	clientID,
	tenantID *string,
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

	response, err := h.magicLinkService.SendMagicLink(r.Context(), req.Email, clientID, tenantID, false)
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
// magic-link token is verified. Public links must contain client_id.
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
	if clientID == nil || tenantID != nil {
		resp.Error(w, http.StatusBadRequest, "Magic link must contain client_id and must not contain tenant_id")
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
