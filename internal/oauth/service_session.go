package oauth

import (
	"context"
	"net/url"

	"github.com/maintainerd/auth/internal/authevent"
	"github.com/maintainerd/auth/internal/platform/apperror"
	"github.com/maintainerd/auth/internal/platform/jwt"
	"github.com/maintainerd/auth/internal/platform/middleware"
	"github.com/maintainerd/auth/internal/platform/ptr"
	"github.com/maintainerd/auth/internal/platform/security"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"gorm.io/gorm"
)

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
		claims, err := jwt.ValidateToken(req.IDTokenHint)
		if err == nil {
			if sub, ok := claims["sub"].(string); ok && sub != "" {
				user, _ := s.userRepo.FindBySubAndClientID(sub, req.ClientID)
				if user != nil {
					userID = &user.UserID
					// Revoke all refresh tokens for this user.
					_, _ = s.refreshTokenRepo.RevokeByUserID(user.UserID)
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
					postLogoutRedirectURI = req.PostLogoutRedirectURI
					if req.State != "" {
						postLogoutRedirectURI += "?state=" + url.QueryEscape(req.State)
					}
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
	claims, err := jwt.ValidateToken(req.LogoutToken)
	if err != nil {
		span.SetStatus(codes.Error, "invalid logout token")
		return apperror.NewOAuthInvalidRequest("logout_token is invalid or expired")
	}

	// Confirm the token has a non-empty sub claim.
	sub, ok := claims["sub"].(string)
	if !ok || sub == "" {
		span.SetStatus(codes.Error, "missing sub claim")
		return apperror.NewOAuthInvalidRequest("logout_token is missing the sub claim")
	}

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
		_, _ = s.refreshTokenRepo.RevokeByUserID(user.UserID)

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
	client, err := findActiveClientByIdentifier(s.db, clientID)
	if err != nil || client == nil || client.ClientURIs == nil {
		return false
	}
	for _, uri := range *client.ClientURIs {
		if uri.URI == redirectURI {
			return true
		}
	}
	return false
}
