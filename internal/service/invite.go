package service

import (
	"bytes"
	"context"
	"html/template"
	"time"

	"github.com/maintainerd/auth/internal/apperror"
	"github.com/maintainerd/auth/internal/config"
	"github.com/maintainerd/auth/internal/crypto"
	"github.com/maintainerd/auth/internal/email"
	"github.com/maintainerd/auth/internal/model"
	"github.com/maintainerd/auth/internal/ptr"
	"github.com/maintainerd/auth/internal/repository"
	"github.com/maintainerd/auth/internal/signedurl"
	"gorm.io/gorm"
)

type InviteService interface {
	SendInvite(ctx context.Context, tenantID int64, email string, userID int64, roleUUIDs []string) (*model.Invite, error)
}

type inviteService struct {
	db                *gorm.DB
	inviteRepo        repository.InviteRepository
	clientRepo        repository.ClientRepository
	roleRepo          repository.RoleRepository
	emailTemplateRepo repository.EmailTemplateRepository
}

func NewInviteService(
	db *gorm.DB,
	inviteRepo repository.InviteRepository,
	clientRepo repository.ClientRepository,
	roleRepo repository.RoleRepository,
	emailTemplateRepo repository.EmailTemplateRepository,
) InviteService {
	return &inviteService{
		db:                db,
		inviteRepo:        inviteRepo,
		clientRepo:        clientRepo,
		roleRepo:          roleRepo,
		emailTemplateRepo: emailTemplateRepo,
	}
}

func (s *inviteService) SendInvite(
	ctx context.Context,
	tenantID int64,
	email string,
	userID int64,
	roleUUIDs []string,
) (*model.Invite, error) {
	var invite *model.Invite

	err := s.db.Transaction(func(tx *gorm.DB) error {
		clientRepo := s.clientRepo.WithTx(tx)
		roleRepo := s.roleRepo.WithTx(tx)
		inviteRepo := s.inviteRepo.WithTx(tx)

		Client, err := clientRepo.FindDefault()
		if err != nil {
			return err
		}
		if Client == nil ||
			Client.Status != model.StatusActive ||
			Client.Domain == nil || *Client.Domain == "" ||
			Client.IdentityProvider == nil ||
			Client.IdentityProvider.Tenant == nil ||
			Client.IdentityProvider.Tenant.TenantID == 0 {
			return apperror.NewValidation("invalid client or identity provider")
		}

		// Get tenant id
		tenantId := Client.IdentityProvider.Tenant.TenantID

		// Find roles by UUIDs
		foundRoles, err := roleRepo.FindByUUIDs(roleUUIDs)
		if err != nil {
			return err
		}

		// Validate all roles belong to the correct tenant
		if len(foundRoles) != len(roleUUIDs) {
			return apperror.NewNotFoundWithReason("one or more roles not found")
		}
		for _, role := range foundRoles {
			if role.TenantID != tenantId {
				return apperror.NewValidation("invalid role for the given client")
			}
		}

		inviteToken := crypto.GenerateIdentifier(32)
		expiresAt := ptr.TimePtr(time.Now().Add(72 * time.Hour))

		invite = &model.Invite{
			TenantID:        tenantID,
			ClientID:        Client.ClientID,
			InvitedEmail:    email,
			InvitedByUserID: userID,
			InviteToken:     inviteToken,
			Status:          model.StatusPending,
			ExpiresAt:       expiresAt,
		}

		if _, err := inviteRepo.Create(invite); err != nil {
			return err
		}

		// Create invite roles request
		var inviteRoles []model.InviteRole
		for _, role := range foundRoles {
			inviteRoles = append(inviteRoles, model.InviteRole{
				InviteID: invite.InviteID,
				RoleID:   role.RoleID,
			})
		}

		// Bulk create invite roles
		if err := tx.Create(&inviteRoles).Error; err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	// Generate signed invite URL (API domain)
	params := map[string]string{"invite_token": invite.InviteToken}
	apiBaseURL := config.AppPrivateHostname + "/register/invite"
	signedAPIURL, err := signedurl.GenerateSignedURL(apiBaseURL, params, 72*time.Hour)
	if err != nil {
		return nil, apperror.NewInternal("failed to generate signed invite URL", err)
	}

	// Convert it to frontend URL
	frontendBaseURL := config.AccountHostname + "/register/invite"
	inviteURL, err := signedurl.ConvertToFrontendURL(signedAPIURL, frontendBaseURL)
	if err != nil {
		return nil, apperror.NewInternal("failed to convert invite URL", err)
	}

	// Send invite email
	if err := s.sendInviteEmail(ctx, email, inviteURL); err != nil {
		return nil, apperror.NewInternal("failed to send invite email", err)
	}

	return invite, nil
}

func (s *inviteService) sendInviteEmail(ctx context.Context, to, inviteURL string) error {
	// Get email template from DB
	templateEntity, err := s.emailTemplateRepo.FindByName("internal:user:invite")
	if err != nil {
		return apperror.NewInternal("failed to fetch invite email template", err)
	}

	// Prepare data for the template
	data := struct {
		InviteURL string
		LogoURL   string
	}{
		InviteURL: inviteURL,
	}

	// Parse HTML template
	tmpl, err := template.New("invite_html").Parse(templateEntity.BodyHTML)
	if err != nil {
		return apperror.NewInternal("failed to parse HTML invite template", err)
	}
	var bodyHTML bytes.Buffer
	if err := tmpl.Execute(&bodyHTML, data); err != nil {
		return apperror.NewInternal("failed to execute HTML invite template", err)
	}

	// Parse plain-text template if available
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

	// Send email
	return email.SendEmail(ctx, email.SendEmailParams{
		To:        to,
		Subject:   templateEntity.Subject,
		BodyHTML:  bodyHTML.String(),
		BodyPlain: bodyPlainStr,
	})
}
