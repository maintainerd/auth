package service

import (
	"bytes"
	"errors"
	"fmt"
	"html/template"
	"time"

	"github.com/maintainerd/auth/internal/config"
	"github.com/maintainerd/auth/internal/model"
	"github.com/maintainerd/auth/internal/repository"
	"github.com/maintainerd/auth/internal/util"
	"gorm.io/gorm"
)

type InviteService interface {
	SendInvite(email string, userID int64, roleUUIDs []string) (*model.Invite, error)
}

type inviteService struct {
	db                *gorm.DB
	inviteRepo        repository.InviteRepository
	authClientRepo    repository.AuthClientRepository
	roleRepo          repository.RoleRepository
	emailTemplateRepo repository.EmailTemplateRepository
}

func NewInviteService(
	db *gorm.DB,
	inviteRepo repository.InviteRepository,
	authClientRepo repository.AuthClientRepository,
	roleRepo repository.RoleRepository,
	emailTemplateRepo repository.EmailTemplateRepository,
) InviteService {
	return &inviteService{
		db:                db,
		inviteRepo:        inviteRepo,
		authClientRepo:    authClientRepo,
		roleRepo:          roleRepo,
		emailTemplateRepo: emailTemplateRepo,
	}
}

func (s *inviteService) SendInvite(
	email string,
	userID int64,
	roleUUIDs []string,
) (*model.Invite, error) {
	var invite *model.Invite

	err := s.db.Transaction(func(tx *gorm.DB) error {
		authClientRepo := repository.NewAuthClientRepository(tx)
		roleRepo := repository.NewRoleRepository(tx)
		inviteRepo := repository.NewInviteRepository(tx)

		authClient, err := authClientRepo.FindDefault()
		if err != nil {
			return err
		}
		if authClient == nil ||
			!authClient.IsActive ||
			authClient.Domain == nil || *authClient.Domain == "" ||
			authClient.IdentityProvider == nil ||
			authClient.IdentityProvider.AuthContainer == nil ||
			authClient.IdentityProvider.AuthContainer.AuthContainerID == 0 {
			return errors.New("invalid client or identity provider")
		}

		// Get container id
		authContainerId := authClient.IdentityProvider.AuthContainer.AuthContainerID

		// Find roles by UUIDs
		foundRoles, err := roleRepo.FindByUUIDs(roleUUIDs)
		if err != nil {
			return err
		}

		// Validate all roles belong to the correct auth container
		if len(foundRoles) != len(roleUUIDs) {
			return errors.New("one or more roles not found")
		}
		for _, role := range foundRoles {
			if role.AuthContainerID != authContainerId {
				return errors.New("invalid role for the given client")
			}
		}

		inviteToken := util.GenerateIdentifier(32)
		expiresAt := util.TimePtr(time.Now().Add(72 * time.Hour))

		invite = &model.Invite{
			AuthClientID:    authClient.AuthClientID,
			InvitedEmail:    email,
			InvitedByUserID: userID,
			InviteToken:     inviteToken,
			Status:          "pending",
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
	signedAPIURL, err := util.GenerateSignedURL(apiBaseURL, params, 72*time.Hour)
	if err != nil {
		return nil, fmt.Errorf("failed to generate signed invite URL: %w", err)
	}

	// Convert it to frontend URL
	frontendBaseURL := config.AccountHostname + "/register/invite"
	inviteURL, err := util.ConvertToFrontendURL(signedAPIURL, frontendBaseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to convert invite URL: %w", err)
	}

	// Send invite email
	if err := s.sendInviteEmail(email, inviteURL); err != nil {
		return nil, fmt.Errorf("failed to send invite email: %w", err)
	}

	return invite, nil
}

func (s *inviteService) sendInviteEmail(to, inviteURL string) error {
	// Get email template from DB
	templateEntity, err := s.emailTemplateRepo.FindByName("internal:user:invite")
	if err != nil {
		return fmt.Errorf("failed to fetch invite email template: %w", err)
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
		return fmt.Errorf("failed to parse HTML invite template: %w", err)
	}
	var bodyHTML bytes.Buffer
	if err := tmpl.Execute(&bodyHTML, data); err != nil {
		return fmt.Errorf("failed to execute HTML invite template: %w", err)
	}

	// Parse plain-text template if available
	var bodyPlainStr string
	if templateEntity.BodyPlain != nil {
		tmplPlain, err := template.New("invite_plain").Parse(*templateEntity.BodyPlain)
		if err != nil {
			return fmt.Errorf("failed to parse plain invite template: %w", err)
		}
		var bodyPlain bytes.Buffer
		if err := tmplPlain.Execute(&bodyPlain, data); err != nil {
			return fmt.Errorf("failed to execute plain invite template: %w", err)
		}
		bodyPlainStr = bodyPlain.String()
	}

	// Send email
	return util.SendEmail(util.SendEmailParams{
		To:        to,
		Subject:   templateEntity.Subject,
		BodyHTML:  bodyHTML.String(),
		BodyPlain: bodyPlainStr,
	})
}
