package oauth

import (
	"context"
	"log/slog"
	"net/url"
	"time"

	jwtlib "github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/maintainerd/maintainerd-auth/internal/authevent"
	"github.com/maintainerd/maintainerd-auth/internal/platform/apperror"
	"github.com/maintainerd/maintainerd-auth/internal/platform/cache"
	"github.com/maintainerd/maintainerd-auth/internal/platform/jwt"
	"github.com/maintainerd/maintainerd-auth/internal/platform/middleware"
	"github.com/maintainerd/maintainerd-auth/internal/platform/ptr"
	"github.com/maintainerd/maintainerd-auth/internal/platform/security"
	"github.com/maintainerd/maintainerd-auth/internal/shared"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"gorm.io/gorm"
)

var oauthSessionValidateTokenWithContext = jwt.ValidateTokenWithContext

// logoutTokenReplayGuard remembers spent logout-token `jti` values.
// OIDC Back-Channel Logout §2.6 point 7 requires it: without replay protection a
// captured logout token keeps terminating the user's sessions forever, which is
// a denial-of-service against that account.
var logoutTokenReplayGuard = &clientAssertionReplayGuard{seen: map[string]time.Time{}}

// validateLogoutTokenShape applies OIDC Back-Channel Logout 1.0 §2.6 to an
// already signature-verified logout token.
//
// Signature verification alone does not distinguish a logout token from any
// other token this server signs — all of them use one process-wide key — so the
// shape rules below are what make "this is a logout instruction" checkable:
//
//	§2.4/§2.6 point 4: a sub, a sid, or both.
//	§2.4/§2.6 point 5: an `events` claim carrying the backchannel-logout member.
//	§2.4/§2.6 point 6: NO `nonce` — a nonce is what marks an ID token, and
//	                   forbidding it is precisely how the spec stops an ID token
//	                   being replayed here.
//	§2.6 point 7:      a jti, used exactly once.
func validateLogoutTokenShape(ctx context.Context, claims jwtlib.MapClaims) *apperror.OAuthError {
	sub, _ := claims["sub"].(string)
	sid, _ := claims["sid"].(string)
	if sub == "" && sid == "" {
		return apperror.NewOAuthInvalidRequest("logout_token must contain a sub or a sid claim")
	}
	// This server can only act on a token whose subject it can resolve.
	if sub == "" {
		return apperror.NewOAuthInvalidRequest("logout_token is missing the sub claim")
	}

	if _, present := claims["nonce"]; present {
		return apperror.NewOAuthInvalidRequest("logout_token must not contain a nonce claim")
	}

	events, ok := claims["events"].(map[string]any)
	if !ok {
		return apperror.NewOAuthInvalidRequest("logout_token is missing the events claim")
	}
	if _, ok := events[jwt.BackchannelLogoutEventURI]; !ok {
		return apperror.NewOAuthInvalidRequest("logout_token events claim does not declare a back-channel logout")
	}

	jti, _ := claims["jti"].(string)
	if jti == "" {
		return apperror.NewOAuthInvalidRequest("logout_token is missing the jti claim")
	}
	if !rememberLogoutTokenJTI(ctx, jti) {
		return apperror.NewOAuthInvalidRequest("logout_token has already been used")
	}

	return nil
}

// logoutTokenReplayStore is the SHARED store used to enforce logout-token
// single-use. Set from the composition root; nil falls back to the in-process
// guard.
var logoutTokenReplayStore cache.JTIDenylister

// SetLogoutTokenReplayStore wires the shared replay store.
//
// OIDC Back-Channel Logout 1.0 §2.6 requires a logout token's `jti` to be
// single-use. The in-process guard cannot deliver that across replicas: the same
// token replayed against a different instance sees an empty map and is accepted,
// so the property silently disappears the moment the service is scaled — which
// packaging it as an image invites. Redis is already the shared JTI denylist for
// access-token revocation, so it is the natural home.
func SetLogoutTokenReplayStore(store cache.JTIDenylister) { logoutTokenReplayStore = store }

// rememberLogoutTokenJTI records a logout token's jti and reports whether it was
// previously unseen.
//
// Fails CLOSED on a store error: an unreachable Redis must not turn single-use
// into unlimited-use for a token that ends sessions.
func rememberLogoutTokenJTI(ctx context.Context, jti string) bool {
	if logoutTokenReplayStore == nil {
		return logoutTokenReplayGuard.remember(jti, time.Now())
	}
	key := "logout_token:" + jti
	seen, err := logoutTokenReplayStore.IsJTIDenied(ctx, key)
	if err != nil {
		return false
	}
	if seen {
		return false
	}
	if err := logoutTokenReplayStore.DenyJTI(ctx, key, logoutTokenReplayTTL); err != nil {
		return false
	}
	return true
}

// logoutTokenReplayTTL bounds how long a spent jti is remembered. It only has to
// outlive the token itself, which is short-lived by construction.
const logoutTokenReplayTTL = 15 * time.Minute

// OAuthSessionService handles RP-Initiated Logout (OIDC Session Mgmt 1.0) and
// OIDC Back-Channel Logout 1.0.
type OAuthSessionService interface {
	// EndSession processes a GET /oauth/end_session request. Revokes the user's
	// refresh tokens and returns the post-logout redirect URI (if registered).
	EndSession(ctx context.Context, req OAuthEndSessionRequestDTO) (string, *apperror.OAuthError)

	// BackchannelLogout processes a POST /oauth/logout/backchannel request.
	// Validates the logout_token JWT and revokes all sessions for the identified
	// user/client combination.
	BackchannelLogout(ctx context.Context, req OAuthBackchannelLogoutRequestDTO) *apperror.OAuthError
}

type oauthSessionService struct {
	db               *gorm.DB
	clientRepo       ClientRepository
	userRepo         UserRepository
	refreshTokenRepo OAuthRefreshTokenRepository
	authEventService authevent.AuthEventService
}

// NewOAuthSessionService creates a new OAuthSessionService.
func NewOAuthSessionService(
	db *gorm.DB,
	clientRepo ClientRepository,
	userRepo UserRepository,
	refreshTokenRepo OAuthRefreshTokenRepository,
	authEventService authevent.AuthEventService,
) OAuthSessionService {
	return &oauthSessionService{
		db:               db,
		clientRepo:       clientRepo,
		userRepo:         userRepo,
		refreshTokenRepo: refreshTokenRepo,
		authEventService: authEventService,
	}
}

// EndSession implements OAuthSessionService.
func (s *oauthSessionService) EndSession(ctx context.Context, req OAuthEndSessionRequestDTO) (string, *apperror.OAuthError) {
	_, span := otel.Tracer("service").Start(ctx, "oauth_session.end_session")
	defer span.End()

	var userID *int64

	// If an id_token_hint was provided, identify the user from it.
	if req.IDTokenHint != "" {
		claims, err := oauthSessionValidateTokenWithContext(ctx, req.IDTokenHint)
		if err == nil {
			if sub, ok := claims["sub"].(string); ok && sub != "" {
				user, _ := s.userRepo.FindBySubAndClientID(sub, req.ClientID)
				if user != nil {
					userID = &user.UserID
					sid, _ := claims["sid"].(string)
					// Terminating the SESSION is what makes this a logout. Revoking
					// refresh tokens alone left the user_sessions row live, so the very
					// next /oauth/authorize saw a valid session and issued a fresh code
					// with no re-authentication — the browser was never logged out.
					s.terminateSessions(ctx, user.UserID, sid)
					if sessionUUID, parseErr := uuid.Parse(sid); parseErr == nil {
						// Scoped to the session the RP asked to end. Revoking every
						// refresh token for the user would also sign them out on their
						// other browsers and devices, which is what "sign out everywhere"
						// is for and RP-initiated logout is not.
						_, _ = s.refreshTokenRepo.RevokeBySession(sessionUUID)
					} else {
						// No usable sid: fall back to the whole user rather than leaving
						// refresh tokens alive after an explicit logout.
						_, _ = s.refreshTokenRepo.RevokeByUserID(user.UserID)
					}
				}
			}
		}
		// Validation failure is silently ignored per OIDC Session Mgmt §5.
	}

	if userID != nil {
		s.authEventService.Log(ctx, authevent.AuthEventInput{
			ActorUserID: userID,
			IPAddress:   middleware.ClientIPFromContext(ctx),
			UserAgent:   ptr.PtrOrNil(middleware.UserAgentFromContext(ctx)),
			Category:    authevent.AuthEventCategorySession,
			EventType:   authevent.AuthEventTypeSessionExpired,
			Severity:    authevent.AuthEventSeverityInfo,
			Result:      authevent.AuthEventResultSuccess,
			Description: ptr.Ptr("RP-initiated logout"),
		})
		span.SetAttributes(attribute.Int64("user.id", *userID))
	}

	// Build the post-logout redirect URI if provided and validated.
	postLogoutRedirectURI := ""
	if req.PostLogoutRedirectURI != "" {
		if _, err := url.ParseRequestURI(req.PostLogoutRedirectURI); err == nil {
			if err := security.ValidateRedirectURI(req.PostLogoutRedirectURI); err == nil {
				if s.validateClientPostLogoutRedirect(req.ClientID, req.PostLogoutRedirectURI) {
					postLogoutRedirectURI = appendLogoutState(req.PostLogoutRedirectURI, req.State)
				}
			}
		}
	}

	span.SetStatus(codes.Ok, "")
	return postLogoutRedirectURI, nil
}

// BackchannelLogout implements OAuthSessionService.
func (s *oauthSessionService) BackchannelLogout(ctx context.Context, req OAuthBackchannelLogoutRequestDTO) *apperror.OAuthError {
	_, span := otel.Tracer("service").Start(ctx, "oauth_session.backchannel_logout")
	defer span.End()

	// Validate the logout token as a JWT.
	claims, err := oauthSessionValidateTokenWithContext(ctx, req.LogoutToken)
	if err != nil {
		span.SetStatus(codes.Error, "invalid logout token")
		return apperror.NewOAuthInvalidRequest("logout_token is invalid or expired")
	}

	// OIDC Back-Channel Logout 1.0 §2.6. The endpoint is unauthenticated, so this
	// validation IS the authentication: without it any JWT this server signed that
	// merely carried a `sub` — including an ID token every relying party is handed
	// by design — revoked all of that user's refresh tokens across every client.
	if oerr := validateLogoutTokenShape(ctx, claims); oerr != nil {
		span.SetStatus(codes.Error, "logout token failed §2.6 validation")
		return oerr
	}

	sub, _ := claims["sub"].(string)
	sid, _ := claims["sid"].(string)

	// Resolve the client from the aud claim.
	clientID := ""
	if aud, ok := claims["client_id"].(string); ok {
		clientID = aud
	}

	// Locate the user and revoke their refresh tokens.
	user, err := s.userRepo.FindBySubAndClientID(sub, clientID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "user lookup failed")
		return apperror.NewOAuthServerError("an unexpected error occurred")
	}

	if user != nil {
		s.terminateSessions(ctx, user.UserID, sid)
		if sessionUUID, parseErr := uuid.Parse(sid); parseErr == nil {
			_, _ = s.refreshTokenRepo.RevokeBySession(sessionUUID)
		} else {
			_, _ = s.refreshTokenRepo.RevokeByUserID(user.UserID)
		}

		s.authEventService.Log(ctx, authevent.AuthEventInput{
			ActorUserID: &user.UserID,
			IPAddress:   middleware.ClientIPFromContext(ctx),
			UserAgent:   ptr.PtrOrNil(middleware.UserAgentFromContext(ctx)),
			Category:    authevent.AuthEventCategorySession,
			EventType:   authevent.AuthEventTypeSessionExpired,
			Severity:    authevent.AuthEventSeverityInfo,
			Result:      authevent.AuthEventResultSuccess,
			Description: ptr.Ptr("Backchannel logout processed"),
		})

		span.SetAttributes(attribute.Int64("user.id", user.UserID))
	}

	span.SetStatus(codes.Ok, "")
	return nil
}

func (s *oauthSessionService) validateClientPostLogoutRedirect(clientID string, redirectURI string) bool {
	if clientID == "" {
		return false
	}
	var client Client
	err := s.db.
		Preload("ClientURIs").
		Where("identifier = ? AND status = ?", clientID, "active").
		First(&client).Error
	if err != nil || client.ClientURIs == nil {
		return false
	}
	for _, uri := range *client.ClientURIs {
		// The type filter is the point: client_uris holds five distinct kinds
		// (redirect_uri, origin_uri, logout_uri, login_uri, cors_origin_uri) and the
		// match used to ignore it, so a cors_origin_uri registered purely so a
		// browser could call the API became a valid place to land a user after
		// logout. Only logout_uri is registered as a post-logout landing page —
		// compare client.redirectMatches, which applies the same rule to
		// redirect_uri.
		if uri.Type == shared.ClientURITypeLogout && uri.URI == redirectURI {
			return true
		}
	}
	return false
}

// terminateSessions marks the user's browser session(s) revoked.
//
// It writes to user_sessions, which is otherwise owned by the authentication
// layer (see repository_user_session_auth.go). Logout is the one place the OAuth
// layer has to: the whole point of RP-initiated and back-channel logout is to
// end the session, and everything downstream — session middleware's
// ValidateAndTouch, and the authorize endpoint's "already signed in" path — keys
// off the revoked_at column.
//
// When sid names a session, only that session ends; a user's other browsers and
// devices are untouched. With no usable sid, every live session for the user is
// ended, because an unscoped logout that ends nothing is worse than one that
// ends too much.
func (s *oauthSessionService) terminateSessions(ctx context.Context, userID int64, sid string) {
	if s.db == nil {
		return
	}

	q := s.db.WithContext(ctx).
		Table("user_sessions").
		Where("user_id = ? AND revoked_at IS NULL", userID)

	if sessionUUID, err := uuid.Parse(sid); err == nil {
		q = q.Where("user_session_uuid = ?", sessionUUID)
	}

	if err := q.Updates(map[string]any{
		"revoked_at":     time.Now(),
		"revoked_reason": "logout",
	}).Error; err != nil {
		slog.WarnContext(ctx, "oauth logout could not revoke the user session; the browser stays signed in",
			"user_id", userID, "error", err)
	}
}

// appendLogoutState adds the RP's `state` to the post-logout redirect.
//
// The old code appended "?state=" unconditionally, which produced a second '?'
// on any registered logout URI that already carried a query string — the RP then
// saw a single parameter literally named "…?state" and lost both its own
// parameters and its CSRF state.
func appendLogoutState(redirectURI, state string) string {
	if state == "" {
		return redirectURI
	}
	u, err := url.Parse(redirectURI)
	if err != nil {
		return redirectURI
	}
	q := u.Query()
	q.Set("state", state)
	u.RawQuery = q.Encode()
	return u.String()
}
