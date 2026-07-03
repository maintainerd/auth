package middleware

import (
	"context"
	"errors"
	"net/http"

	"github.com/google/uuid"
	"github.com/maintainerd/maintainerd-auth/internal/platform/apperror"
	resp "github.com/maintainerd/maintainerd-auth/internal/platform/response"
)

// SessionValidator is the minimal interface required by SessionValidationMiddleware.
// It matches SessionService.ValidateAndTouch.
type SessionValidator interface {
	ValidateAndTouch(ctx context.Context, sessionUUID uuid.UUID, userID int64) error
}

var defaultSessionValidator SessionValidator

func SetSessionValidator(svc SessionValidator) {
	defaultSessionValidator = svc
}

func ValidateSessionFromRequest(w http.ResponseWriter, r *http.Request) bool {
	return validateSessionWith(defaultSessionValidator, w, r)
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
			if !validateSessionWith(svc, w, r) {
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func validateSessionWith(svc SessionValidator, w http.ResponseWriter, r *http.Request) bool {
	if svc == nil {
		return true
	}
	sessionID := ""
	if claims := JWTClaimsFromRequest(r); claims != nil {
		sessionID = claims.SessionID
	}
	if sessionID == "" {
		sessionID = r.Header.Get("X-Session-ID")
	}
	if sessionID == "" {
		return true
	}

	sessionUUID, err := uuid.Parse(sessionID)
	if err != nil {
		resp.Error(w, http.StatusBadRequest, "Invalid session ID format")
		return false
	}

	auth := AuthFromRequest(r)
	if auth.User == nil {
		resp.Error(w, http.StatusUnauthorized, "Unauthorized")
		return false
	}

	if err := svc.ValidateAndTouch(r.Context(), sessionUUID, auth.User.UserID); err != nil {
		var unauth *apperror.UnauthorizedError
		if errors.As(err, &unauth) {
			resp.Error(w, http.StatusUnauthorized, err.Error())
			return false
		}
		resp.Error(w, http.StatusInternalServerError, "Session validation failed")
		return false
	}
	return true
}
