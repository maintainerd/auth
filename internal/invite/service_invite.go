package invite

import (
	"bytes"
	"context"
	"html/template"
	"os"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/maintainerd/maintainerd-auth/internal/branding"
	clientpkg "github.com/maintainerd/maintainerd-auth/internal/client"
	"github.com/maintainerd/maintainerd-auth/internal/platform/apperror"
	"github.com/maintainerd/maintainerd-auth/internal/platform/config"
	"github.com/maintainerd/maintainerd-auth/internal/platform/crypto"
	"github.com/maintainerd/maintainerd-auth/internal/platform/email"
	"github.com/maintainerd/maintainerd-auth/internal/platform/ptr"
	"github.com/maintainerd/maintainerd-auth/internal/platform/signedurl"
	"github.com/maintainerd/maintainerd-auth/internal/shared"
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
	SendInvite(ctx context.Context, tenantID int64, email string, userID int64, registrationFlowUUID *string, callbackURL *string) (*Invite, error)
	ResendInvite(ctx context.Context, inviteUUID uuid.UUID, tenantID int64) (*Invite, error)
	ListInvites(ctx context.Context, tenantID int64) ([]Invite, error)
	RevokeInvite(ctx context.Context, inviteUUID uuid.UUID, tenantID int64) error
	GetByUUID(ctx context.Context, inviteUUID uuid.UUID, tenantID int64) (*Invite, error)
	GetByToken(ctx context.Context, inviteToken string) (*Invite, error)
}

type inviteService struct {
	db                   *gorm.DB
	inviteRepo           InviteRepository
	clientRepo           ClientRepository
	emailTemplateRepo    branding.EmailTemplateRepository
	registrationFlowRepo RegistrationFlowRepository
}

func NewInviteService(
	db *gorm.DB,
	inviteRepo InviteRepository,
	clientRepo ClientRepository,
	emailTemplateRepo branding.EmailTemplateRepository,
	registrationFlowRepo RegistrationFlowRepository,
) InviteService {
	return &inviteService{
		db:                   db,
		inviteRepo:           inviteRepo,
		clientRepo:           clientRepo,
		emailTemplateRepo:    emailTemplateRepo,
		registrationFlowRepo: registrationFlowRepo,
	}
}

func (s *inviteService) SendInvite(
	ctx context.Context,
	tenantID int64,
	email string,
	userID int64,
	registrationFlowUUID *string,
	callbackURL *string,
) (*Invite, error) {
	_, span := otel.Tracer("service").Start(ctx, "invite.send")
	defer span.End()

	var invite *Invite
	var clientIdentifier string
	var clientIsSystem bool

	err := s.db.Transaction(func(tx *gorm.DB) error {
		clientRepo := s.clientRepo.WithTx(tx)
		registrationFlowRepo := s.registrationFlowRepo.WithTx(tx)
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
		clientIsSystem = Client.IsSystem

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

		if registrationFlowUUID != nil && *registrationFlowUUID != "" {
			registrationFlowUUIDParsed, err := uuid.Parse(*registrationFlowUUID)
			if err != nil {
				return apperror.NewValidation("invalid registration flow UUID")
			}
			registrationFlow, err := registrationFlowRepo.FindByUUIDAndTenantID(registrationFlowUUIDParsed, systemTenantID)
			if err != nil || registrationFlow == nil {
				return apperror.NewNotFoundWithReason("registration flow not found")
			}
			if registrationFlow.Status != shared.StatusActive {
				return apperror.NewValidation("registration flow is inactive")
			}
			invite.RegistrationFlowID = &registrationFlow.RegistrationFlowID

			// Use the flow's client for callback/branding resolution.
			var flowClientID int64
			if err := tx.Table("registration_flows").
				Select("client_id").
				Where("registration_flow_id = ?", registrationFlow.RegistrationFlowID).
				Scan(&flowClientID).Error; err == nil && flowClientID > 0 {
				var flowClientTenantID int64
				if err := tx.Table("clients").
					Select("tenant_id").
					Where("client_id = ?", flowClientID).
					Scan(&flowClientTenantID).Error; err != nil || flowClientTenantID != invite.TenantID {
					return apperror.NewNotFoundWithReason("registration flow client not found or access denied")
				}
				invite.ClientID = flowClientID
				clientIsSystem = false
				var flowClientIdentifier string
				if err := tx.Table("clients").
					Select("identifier").
					Where("client_id = ?", flowClientID).
					Scan(&flowClientIdentifier).Error; err == nil && flowClientIdentifier != "" {
					clientIdentifier = flowClientIdentifier
				}
			}
		}

		if invite.RegistrationFlowID != nil {
			registrationFlowID := *invite.RegistrationFlowID
			var flowRoleIDs []int64
			if err := tx.Table("registration_flow_roles").
				Where("registration_flow_id = ?", registrationFlowID).
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

		if callbackURL != nil && *callbackURL != "" {
			var uris []clientURI
			if err := tx.Where("client_id = ? AND type = ?", invite.ClientID, shared.ClientURITypeRedirect).Find(&uris).Error; err != nil {
				return err
			}
			if err := validateCallbackURL(callbackURL, uris); err != nil {
				return apperror.NewValidation("invalid callback URL: " + err.Error())
			}
			invite.CallbackURL = callbackURL
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

	// Generate signed invite URL (API domain) — all invites go to the identity app.
	params := map[string]string{
		"invite_token": invite.InviteToken,
		"email":        invite.InvitedEmail,
	}
	// Resolve the invite tenant. The tenant name is the DNS slug used both for
	// the tenant_id param (system-client invites) and for the per-tenant
	// identity subdomain the invite email links to.
	var inviteTenant TenantRecord
	if err := s.db.Select("name", "is_system").Where("tenant_id = ?", invite.TenantID).First(&inviteTenant).Error; err != nil || inviteTenant.Name == "" {
		return nil, apperror.NewInternal("failed to resolve invite tenant", err)
	}
	if clientIsSystem {
		params["tenant_id"] = inviteTenant.Name
	} else {
		params["client_id"] = clientIdentifier
	}
	apiBaseURL := config.AppPrivateHostname + "/register/invite"
	signedAPIURL, err := signedurl.GenerateSignedURL(apiBaseURL, params, inviteTTL())
	if err != nil {
		return nil, apperror.NewInternal("failed to generate signed invite URL", err)
	}

	frontendBaseURL := shared.FrontendURL(shared.FrontendSurfaceIdentity, inviteTenant.Name, inviteTenant.IsSystem, "/register/invite")
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

	// All invites go to the identity app. Determine surface param by client type.
	var clientIsSystem bool
	if err := s.db.Model(&Client{}).Select("is_system").Where("client_id = ?", existing.ClientID).Scan(&clientIsSystem).Error; err != nil {
		return nil, apperror.NewInternal("failed to resolve invite client", err)
	}
	var inviteTenant TenantRecord
	if err := s.db.Select("name", "is_system").Where("tenant_id = ?", existing.TenantID).First(&inviteTenant).Error; err != nil || inviteTenant.Name == "" {
		return nil, apperror.NewInternal("failed to resolve invite tenant", err)
	}
	if clientIsSystem {
		params["tenant_id"] = inviteTenant.Name
	} else {
		var clientIdentifier string
		if err := s.db.Model(&Client{}).Select("identifier").Where("client_id = ?", existing.ClientID).Scan(&clientIdentifier).Error; err != nil || clientIdentifier == "" {
			return nil, apperror.NewInternal("failed to resolve invite client", err)
		}
		params["client_id"] = clientIdentifier
	}
	frontendBaseURL := shared.FrontendURL(shared.FrontendSurfaceIdentity, inviteTenant.Name, inviteTenant.IsSystem, "/register/invite")
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

func (s *inviteService) GetByUUID(ctx context.Context, inviteUUID uuid.UUID, tenantID int64) (*Invite, error) {
	invite, err := s.inviteRepo.FindByUUIDAndTenantID(inviteUUID, tenantID)
	if err != nil {
		return nil, err
	}
	if invite == nil {
		return nil, apperror.NewNotFound("invite not found")
	}
	return invite, nil
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

func (s *inviteService) GetByToken(ctx context.Context, inviteToken string) (*Invite, error) {
	_, span := otel.Tracer("service").Start(ctx, "invite.get_by_token")
	defer span.End()
	invite, err := s.inviteRepo.FindByToken(inviteToken)
	if err != nil {
		return nil, err
	}
	if invite == nil {
		return nil, apperror.NewNotFoundWithReason("invite not found")
	}
	return invite, nil
}

type clientURI struct {
	URI  string
	Type string
}

func validateCallbackURL(callbackURL *string, uris []clientURI) error {
	if callbackURL == nil || *callbackURL == "" {
		return nil
	}
	matches := make([]clientpkg.RedirectURIMatch, len(uris))
	for i, u := range uris {
		matches[i] = clientpkg.RedirectURIMatch{URI: u.URI, Type: u.Type}
	}
	return clientpkg.MatchClientRedirectURI(matches, *callbackURL)
}
