package authn

import (
	"encoding/json"
	"net/http"

	"github.com/maintainerd/auth/internal/dto"
	resp "github.com/maintainerd/auth/internal/platform/response"
	"github.com/maintainerd/auth/internal/service"
)

// SMSLoginHandler handles SMS one-time-code login flows.
type SMSLoginHandler struct {
	smsLoginService service.SMSLoginService
}

func NewSMSLoginHandler(smsLoginService service.SMSLoginService) *SMSLoginHandler {
	return &SMSLoginHandler{smsLoginService: smsLoginService}
}

// SendOTP sends a one-time SMS code to the given phone number.
//
// POST /sms-login/send
func (h *SMSLoginHandler) SendOTP(w http.ResponseWriter, r *http.Request) {
	var req dto.SMSLoginSendDTO
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		resp.Error(w, http.StatusBadRequest, "Invalid JSON format")
		return
	}
	if err := req.Validate(); err != nil {
		resp.ValidationError(w, err)
		return
	}

	if err := h.smsLoginService.SendOTP(r.Context(), req); err != nil {
		resp.HandleServiceError(w, r, "Failed to send OTP", err)
		return
	}

	resp.Success(w, nil, "If a matching account exists, a verification code has been sent")
}

// VerifyOTP validates the submitted OTP and returns tokens on success.
//
// POST /sms-login/verify
func (h *SMSLoginHandler) VerifyOTP(w http.ResponseWriter, r *http.Request) {
	var req dto.SMSLoginVerifyDTO
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		resp.Error(w, http.StatusBadRequest, "Invalid JSON format")
		return
	}
	if err := req.Validate(); err != nil {
		resp.ValidationError(w, err)
		return
	}

	tokens, err := h.smsLoginService.VerifyOTP(r.Context(), req)
	if err != nil {
		resp.HandleServiceError(w, r, "OTP verification failed", err)
		return
	}

	resp.Success(w, tokens, "Authenticated successfully")
}
