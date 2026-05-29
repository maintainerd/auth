package authn

import (
	"github.com/go-chi/chi/v5"
	"github.com/maintainerd/auth/internal/rest/handler"
)

// SMSLoginRoute mounts unauthenticated SMS one-time-code login endpoints.
func SMSLoginRoute(
	r chi.Router,
	smsLoginHandler *handler.SMSLoginHandler,
) {
	r.Route("/sms-login", func(r chi.Router) {
		// Send OTP to phone number (unauthenticated)
		r.Post("/send", smsLoginHandler.SendOTP)
		// Verify OTP and obtain tokens (unauthenticated)
		r.Post("/verify", smsLoginHandler.VerifyOTP)
	})
}
