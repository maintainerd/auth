package middleware

import (
	"context"
	"errors"
	"net/http"
	"strings"

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

// sessionlessSubjectTypes are `sub_type` values whose tokens structurally have
// no browser session to validate, so the absence of `sid` on them is correct
// rather than a hole:
//
//	client / service — machine principals from client_credentials; there is no
//	                   end user and nothing in user_sessions to look up.
//	device / ciba    — RFC 8628 device flow and CIBA. The user approves out of
//	                   band on a second device; the token is delivered to a
//	                   consumption device that was never in a browser session.
//	exchange         — RFC 8693 impersonation/delegation. The issued token
//	                   represents a subject the caller acts for, not a login.
//
// Every OTHER token reaching here authenticated an end user interactively and
// must carry the session it was minted from.
var sessionlessSubjectTypes = map[string]struct{}{
	"client":   {},
	"service":  {},
	"device":   {},
	"ciba":     {},
	"exchange": {},
}

func validateSessionWith(svc SessionValidator, w http.ResponseWriter, r *http.Request) bool {
	if svc == nil {
		return true
	}
	claims := JWTClaimsFromRequest(r)
	sessionID := ""
	if claims != nil {
		sessionID = claims.SessionID
	}
	if sessionID == "" {
		sessionID = r.Header.Get("X-Session-ID")
	}
	if sessionID == "" {
		// A JWT-authenticated request whose token carries no `sid` used to sail
		// straight past session validation, which made that token strictly more
		// powerful than one that had a sid: logout, "sign out everywhere",
		// session revocation and password change all operate on user_sessions, so
		// a token with nothing to look up survives every one of them for its full
		// lifetime. Fail closed instead, except for the grant types above that
		// have no session by construction.
		if claims != nil && !isSessionlessSubject(claims.SubjectType) {
			resp.Error(w, http.StatusUnauthorized, "Token is not bound to a session")
			return false
		}
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

// isSessionlessSubject reports whether a token's sub_type marks it as one that
// legitimately has no server-side session behind it.
func isSessionlessSubject(subjectType string) bool {
	_, ok := sessionlessSubjectTypes[strings.ToLower(strings.TrimSpace(subjectType))]
	return ok
}
