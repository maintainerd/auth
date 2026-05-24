package middleware

import (
	"context"
	"errors"
	"net/http"

	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/apperror"
	resp "github.com/maintainerd/auth/internal/rest/response"
)

// SessionValidator is the minimal interface required by SessionValidationMiddleware.
// It matches SessionService.ValidateAndTouch.
type SessionValidator interface {
	ValidateAndTouch(ctx context.Context, sessionUUID uuid.UUID, userID int64) error
}

// SessionValidationMiddleware enforces idle timeout and absolute session lifetime
// on every authenticated request. It must run after UserContextMiddleware so that
// the authenticated user (and therefore the userID) is available in the context.
//
// Clients send their session ID in the X-Session-ID request header. If the header
// is absent the middleware passes the request through unchanged — session
// validation is opt-in so that paths without a session ID are not broken.
func SessionValidationMiddleware(svc SessionValidator) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			sessionIDHeader := r.Header.Get("X-Session-ID")
			if sessionIDHeader == "" {
				next.ServeHTTP(w, r)
				return
			}

			sessionUUID, err := uuid.Parse(sessionIDHeader)
			if err != nil {
				resp.Error(w, http.StatusBadRequest, "Invalid X-Session-ID format")
				return
			}

			auth := AuthFromRequest(r)
			if auth.User == nil {
				resp.Error(w, http.StatusUnauthorized, "Unauthorized")
				return
			}

			if err := svc.ValidateAndTouch(r.Context(), sessionUUID, auth.User.UserID); err != nil {
				var unauth *apperror.UnauthorizedError
				if errors.As(err, &unauth) {
					resp.Error(w, http.StatusUnauthorized, err.Error())
					return
				}
				resp.Error(w, http.StatusInternalServerError, "Session validation failed")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
