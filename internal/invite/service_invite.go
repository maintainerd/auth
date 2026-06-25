package invite

import (
	"bytes"
	"context"
	"html/template"
	"os"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/branding"
	"github.com/maintainerd/auth/internal/platform/apperror"
	"github.com/maintainerd/auth/internal/platform/config"
	"github.com/maintainerd/auth/internal/platform/crypto"
	"github.com/maintainerd/auth/internal/platform/email"
	"github.com/maintainerd/auth/internal/platform/ptr"
	"github.com/maintainerd/auth/internal/platform/signedurl"
	"github.com/maintainerd/auth/internal/shared"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	"gorm.io/gorm"
)

const defaultInviteTTL = 72 * time.Hour

func inviteTTL() time.Duration {
	if v := os.Getenv("INVITE_TTL_HOURS"); v != "" {
		if h, err := strconv.Atoi(v); err == nil && h > 0 {
			return time.Duration(h) * time.Hour
		}
	}
	return defaultInviteTTL
}

type InviteService interface {
	SendInvite(ctx context.Context, tenantID int64, email string, userID int64, authFlowUUID *string) (*Invite, error)
	ResendInvite(ctx context.Context, inviteUUID uuid.UUID, tenantID int64) (*Invite, error)
	ListInvites(ctx context.Context, tenantID int64) ([]Invite, error)
	RevokeInvite(ctx context.Context, inviteUUID uuid.UUID, tenantID int64) error
}

type inviteService struct {
	db                *gorm.DB
	inviteRepo        InviteRepository
	clientRepo        ClientRepository
	emailTemplateRepo branding.EmailTemplateRepository
	authFlowRepo      AuthFlowRepository
}

func NewInviteService(
	db *gorm.DB,
	inviteRepo InviteRepository,
	clientRepo ClientRepository,
	emailTemplateRepo branding.EmailTemplateRepository,
	authFlowRepo AuthFlowRepository,
) InviteService {
	return &inviteService{
		db:                db,
		inviteRepo:        inviteRepo,
		clientRepo:        clientRepo,
		emailTemplateRepo: emailTemplateRepo,
		authFlowRepo:      authFlowRepo,
	}
}

func (s *inviteService) SendInvite(
	ctx context.Context,
	tenantID int64,
	email string,
	userID int64,
	authFlowUUID *string,
) (*Invite, error) {
	_, span := otel.Tracer("service").Start(ctx, "invite.send")
	defer span.End()

	var invite *Invite
	var authFlowDestination string
	var authFlowIdentifier string
	var clientIdentifier string

	err := s.db.Transaction(func(tx *gorm.DB) error {
		clientRepo := s.clientRepo.WithTx(tx)
		authFlowRepo := s.authFlowRepo.WithTx(tx)
		inviteRepo := s.inviteRepo.WithTx(tx)

		Client, err := clientRepo.FindSystem()
		if err != nil {
			return err
		}
		if Client == nil ||
			Client.Status != shared.StatusActive ||
			Client.Domain == nil || *Client.Domain == "" ||
			Client.TenantID == 0 {
			return apperror.NewValidation("invalid client or identity provider")
		}
		clientIdentifier = *Client.Identifier

		// The system client's tenant is the tenant this invite belongs under.
		systemTenantID := Client.TenantID
		if tenantID != systemTenantID {
			return apperror.NewValidation("tenant mismatch: invite tenant must match the system tenant")
		}

		inviteToken, err := crypto.GenerateIdentifier(32)
		if err != nil {
			return err
		}
		expiresAt := ptr.TimePtr(time.Now().Add(inviteTTL()))

		invite = &Invite{
			TenantID:        tenantID,
			ClientID:        Client.ClientID,
			InvitedEmail:    email,
			InvitedByUserID: &userID,
			InviteToken:     inviteToken,
			Status:          shared.StatusPending,
			ExpiresAt:       expiresAt,
		}

		if authFlowUUID != nil && *authFlowUUID != "" {
			authFlowUUIDParsed, err := uuid.Parse(*authFlowUUID)
			if err != nil {
				return apperror.NewValidation("invalid auth flow UUID")
			}
			authFlow, err := authFlowRepo.FindByUUIDAndTenantID(authFlowUUIDParsed, systemTenantID)
			if err != nil || authFlow == nil {
				return apperror.NewNotFoundWithReason("auth flow not found")
			}
			invite.AuthFlowID = &authFlow.AuthFlowID

			authFlowDestination = authFlow.Destination
			authFlowIdentifier = authFlow.Identifier
		} else {
			defaultAuthFlow, err := authFlowRepo.FindByNameAndTenantID("system:onboarding:registered", systemTenantID)
			if err == nil && defaultAuthFlow != nil {
				invite.AuthFlowID = &defaultAuthFlow.AuthFlowID
				authFlowDestination = defaultAuthFlow.Destination
				authFlowIdentifier = defaultAuthFlow.Identifier
			}
		}

		if invite.AuthFlowID != nil {
			authFlowID := *invite.AuthFlowID
			var flowRoleIDs []int64
			if err := tx.Table("auth_flow_roles").
				Where("auth_flow_id = ?", authFlowID).
				Pluck("role_id", &flowRoleIDs).Error; err != nil {
				return err
			}
			for _, roleID := range flowRoleIDs {
				var hasRole int64
				if err := tx.Table("user_roles").
					Where("user_id = ? AND role_id = ?", userID, roleID).
					Count(&hasRole).Error; err != nil {
					return err
				}
				if hasRole == 0 {
					return apperror.NewValidation("you cannot grant roles you do not possess")
				}
			}
		}

		if _, err := inviteRepo.Create(invite); err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "transaction failed")
		return nil, err
	}

	// Generate signed invite URL (API domain)
	params := map[string]string{
		"invite_token": invite.InviteToken,
		"email":        invite.InvitedEmail,
	}
	if authFlowIdentifier != "" {
		params["auth_flow"] = authFlowIdentifier
	}
	if authFlowDestination == shared.DestinationConsole {
		var tenantIdentifier string
		if err := s.db.Model(&TenantRecord{}).Select("identifier").Where("tenant_id = ?", invite.TenantID).Scan(&tenantIdentifier).Error; err != nil || tenantIdentifier == "" {
			return nil, apperror.NewInternal("failed to resolve invite tenant", err)
		}
		params["tenant_id"] = tenantIdentifier
	} else {
		params["client_id"] = clientIdentifier
	}
	apiBaseURL := config.AppPrivateHostname + "/register/invite"
	signedAPIURL, err := signedurl.GenerateSignedURL(apiBaseURL, params, inviteTTL())
	if err != nil {
		return nil, apperror.NewInternal("failed to generate signed invite URL", err)
	}

	// Convert it to frontend URL — choose hostname based on auth flow's destination
	var frontendBaseURL string
	switch authFlowDestination {
	case shared.DestinationConsole:
		frontendBaseURL = config.AppFrontendConsoleHostname + "/register/invite"
	default:
		frontendBaseURL = config.AppFrontendIdentityHostname + "/register/invite"
	}
	inviteURL, err := signedurl.ConvertToFrontendURL(signedAPIURL, frontendBaseURL)
	if err != nil {
		return nil, apperror.NewInternal("failed to convert invite URL", err)
	}

	// Send invite email (scoped to the invite's tenant — from the request context)
	if err := s.sendInviteEmail(ctx, tenantID, email, inviteURL); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to send invite email")
		return nil, apperror.NewInternal("failed to send invite email", err)
	}

	span.SetStatus(codes.Ok, "")
	return invite, nil
}

func (s *inviteService) ResendInvite(
	ctx context.Context,
	inviteUUID uuid.UUID,
	tenantID int64,
) (*Invite, error) {
	_, span := otel.Tracer("service").Start(ctx, "invite.resend")
	defer span.End()

	existing, err := s.inviteRepo.FindByUUIDAndTenantID(inviteUUID, tenantID)
	if err != nil || existing == nil {
		return nil, apperror.NewNotFoundWithReason("invite not found")
	}

	inviteToken, err := crypto.GenerateIdentifier(32)
	if err != nil {
		return nil, err
	}
	expiresAt := ptr.TimePtr(time.Now().Add(inviteTTL()))

	if err := s.inviteRepo.ResetForResend(inviteUUID, inviteToken, *expiresAt); err != nil {
		return nil, err
	}

	params := map[string]string{
		"invite_token": inviteToken,
		"email":        existing.InvitedEmail,
	}
	var frontendBaseURL string
	var authFlowIdentifier string
	if existing.AuthFlowID != nil {
		authFlow, afErr := s.authFlowRepo.FindByID(*existing.AuthFlowID)
		if afErr == nil && authFlow != nil {
			authFlowIdentifier = authFlow.Identifier
			if authFlow.Destination == shared.DestinationConsole {
				frontendBaseURL = config.AppFrontendConsoleHostname + "/register/invite"
			}
		}
	}
	if frontendBaseURL == "" {
		frontendBaseURL = config.AppFrontendIdentityHostname + "/register/invite"
	}
	if authFlowIdentifier != "" {
		params["auth_flow"] = authFlowIdentifier
	}
	if frontendBaseURL == config.AppFrontendConsoleHostname+"/register/invite" {
		var tenantIdentifier string
		if err := s.db.Model(&TenantRecord{}).Select("identifier").Where("tenant_id = ?", existing.TenantID).Scan(&tenantIdentifier).Error; err != nil || tenantIdentifier == "" {
			return nil, apperror.NewInternal("failed to resolve invite tenant", err)
		}
		params["tenant_id"] = tenantIdentifier
	} else {
		var clientIdentifier string
		if err := s.db.Model(&Client{}).Select("identifier").Where("client_id = ?", existing.ClientID).Scan(&clientIdentifier).Error; err != nil || clientIdentifier == "" {
			return nil, apperror.NewInternal("failed to resolve invite client", err)
		}
		params["client_id"] = clientIdentifier
	}
	apiBaseURL := config.AppPrivateHostname + "/register/invite"
	signedAPIURL, err := signedurl.GenerateSignedURL(apiBaseURL, params, inviteTTL())
	if err != nil {
		return nil, apperror.NewInternal("failed to generate signed invite URL", err)
	}
	inviteURL, err := signedurl.ConvertToFrontendURL(signedAPIURL, frontendBaseURL)
	if err != nil {
		return nil, apperror.NewInternal("failed to convert invite URL", err)
	}

	if err := s.sendInviteEmail(ctx, existing.TenantID, existing.InvitedEmail, inviteURL); err != nil {
		span.RecordError(err)
		return nil, apperror.NewInternal("failed to send invite email", err)
	}

	span.SetStatus(codes.Ok, "")
	existing.InviteToken = inviteToken
	existing.ExpiresAt = expiresAt
	existing.Status = shared.StatusPending
	return existing, nil
}

func (s *inviteService) sendInviteEmail(ctx context.Context, tenantID int64, to, inviteURL string) error {
	var templateEntity *branding.EmailTemplate
	var err error
	templateEntity, err = s.emailTemplateRepo.FindByNameAndTenantID("user:invite", tenantID)
	if err != nil || templateEntity == nil {
		templateEntity, err = s.emailTemplateRepo.FindByName("user:invite")
	}
	if err != nil {
		return apperror.NewInternal("failed to fetch invite email template", err)
	}

	data := struct {
		InviteURL string
		LogoURL   string
	}{
		InviteURL: inviteURL,
	}

	tmpl, err := template.New("invite_html").Parse(templateEntity.BodyHTML)
	if err != nil {
		return apperror.NewInternal("failed to parse HTML invite template", err)
	}
	var bodyHTML bytes.Buffer
	if err := tmpl.Execute(&bodyHTML, data); err != nil {
		return apperror.NewInternal("failed to execute HTML invite template", err)
	}

	var bodyPlainStr string
	if templateEntity.BodyPlain != nil {
		tmplPlain, err := template.New("invite_plain").Parse(*templateEntity.BodyPlain)
		if err != nil {
			return apperror.NewInternal("failed to parse plain invite template", err)
		}
		var bodyPlain bytes.Buffer
		if err := tmplPlain.Execute(&bodyPlain, data); err != nil {
			return apperror.NewInternal("failed to execute plain invite template", err)
		}
		bodyPlainStr = bodyPlain.String()
	}

	return email.SendEmail(ctx, s.db, email.SendEmailParams{
		TenantID:  tenantID,
		To:        to,
		Subject:   templateEntity.Subject,
		BodyHTML:  bodyHTML.String(),
		BodyPlain: bodyPlainStr,
	})
}

// ListInvites returns all invitations for a tenant (admin view).
func (s *inviteService) ListInvites(ctx context.Context, tenantID int64) ([]Invite, error) {
	_, span := otel.Tracer("service").Start(ctx, "invite.list")
	defer span.End()
	return s.inviteRepo.FindAllByTenantID(tenantID)
}

// RevokeInvite marks a pending invitation as revoked. Scoped to the tenant so an
// admin can only revoke invites belonging to their own tenant.
func (s *inviteService) RevokeInvite(ctx context.Context, inviteUUID uuid.UUID, tenantID int64) error {
	_, span := otel.Tracer("service").Start(ctx, "invite.revoke")
	defer span.End()

	existing, err := s.inviteRepo.FindByUUIDAndTenantID(inviteUUID, tenantID)
	if err != nil {
		span.RecordError(err)
		return err
	}
	if existing == nil {
		return apperror.NewNotFoundWithReason("invite not found")
	}
	if err := s.inviteRepo.RevokeByUUID(inviteUUID); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "revoke invite failed")
		return apperror.NewInternal("failed to revoke invite", err)
	}
	span.SetStatus(codes.Ok, "")
	return nil
}
