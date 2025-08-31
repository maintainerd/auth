package service

import (
	"errors"
	"time"

	"github.com/maintainerd/auth/internal/dto"
	"github.com/maintainerd/auth/internal/model"
	"github.com/maintainerd/auth/internal/repository"
	"github.com/maintainerd/auth/internal/util"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type RegisterService interface {
	RegisterPublic(username, password, clientID string) (*dto.AuthResponseDto, error)
	RegisterPublicInvite(username, password, clientID, inviteToken string) (*dto.AuthResponseDto, error)
	RegisterPrivate(username, password string) (*dto.AuthResponseDto, error)
	RegisterPrivateInvite(username, password, inviteToken string) (*dto.AuthResponseDto, error)
}

type registerService struct {
	db             *gorm.DB
	authClientRepo repository.AuthClientRepository
	userRepo       repository.UserRepository
	userRoleRepo   repository.UserRoleRepository
	userTokenRepo  repository.UserTokenRepository
	roleRepo       repository.RoleRepository
	inviteRepo     repository.InviteRepository
}

func NewRegistrationService(
	db *gorm.DB,
	clientRepo repository.AuthClientRepository,
	userRepo repository.UserRepository,
	userRoleRepo repository.UserRoleRepository,
	userTokenRepo repository.UserTokenRepository,
	roleRepo repository.RoleRepository,
	inviteRepo repository.InviteRepository,
) RegisterService {
	return &registerService{
		db:             db,
		authClientRepo: clientRepo,
		userRepo:       userRepo,
		userRoleRepo:   userRoleRepo,
		userTokenRepo:  userTokenRepo,
		roleRepo:       roleRepo,
		inviteRepo:     inviteRepo,
	}
}

func (s *registerService) RegisterPublic(
	username,
	password,
	clientID string,
) (*dto.AuthResponseDto, error) {
	// Get and validate client id
	authClient, err := s.authClientRepo.FindByClientID(clientID)
	if err != nil {
		return nil, err
	}
	if authClient == nil || !authClient.IsActive || authClient.Domain == nil || *authClient.Domain == "" || authClient.AuthContainer == nil {
		return nil, errors.New("invalid client or identity provider")
	}

	// Check if username is an email
	isEmail := util.IsValidEmail(username)

	var createdUser *model.User

	// Transaction
	err = s.db.Transaction(func(tx *gorm.DB) error {
		// Get user by email or username
		var existingUser *model.User
		var err error

		if isEmail {
			existingUser, err = s.userRepo.FindByEmail(username, authClient.AuthContainerID)
		} else {
			existingUser, err = s.userRepo.FindByUsername(username, authClient.AuthContainerID)
		}

		if err != nil {
			return err
		}

		// Check if user already exists
		if existingUser != nil {
			if isEmail {
				return errors.New("email already registered")
			}
			return errors.New("username already taken")
		}

		// Hash password
		hashed, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		if err != nil {
			return err
		}

		// Create user
		newUser := &model.User{
			Username:        username,
			Email:           "",
			Password:        ptr(string(hashed)),
			AuthContainerID: authClient.AuthContainerID,
			OrganizationID:  authClient.AuthContainer.OrganizationID,
			IsActive:        true,
		}
		if isEmail {
			newUser.Email = username
		}

		if err := tx.Create(newUser).Error; err != nil {
			return err
		}

		createdUser = newUser

		// Generate OTP
		otp, err := util.GenerateOTP(6)
		if err != nil {
			return err
		}

		// Create user token
		userToken := &model.UserToken{
			UserID:    createdUser.UserID,
			TokenType: "user:email:verification",
			Token:     otp,
		}
		if err := tx.Create(userToken).Error; err != nil {
			return err
		}

		return nil // commit transaction
	})

	if err != nil {
		return nil, err
	}

	// Return token response
	return s.generateTokenResponse(createdUser.UserUUID.String(), authClient)
}

func (s *registerService) RegisterPrivate(
	username,
	password string,
) (*dto.AuthResponseDto, error) {
	// Get and validate client id (get default client)
	authClient, err := s.authClientRepo.FindDefault()
	if err != nil {
		return nil, err
	}
	if authClient == nil || !authClient.IsActive || authClient.Domain == nil || *authClient.Domain == "" || authClient.AuthContainer == nil {
		return nil, errors.New("invalid client or identity provider")
	}

	// Check if super admin already exists
	superAdmin, err := s.userRepo.FindSuperAdmin()
	if err != nil {
		return nil, err
	}
	if superAdmin != nil {
		return nil, errors.New("api is disabled")
	}

	// Check if username is an email
	isEmail := util.IsValidEmail(username)

	var createdUser *model.User

	// Transaction
	err = s.db.Transaction(func(tx *gorm.DB) error {
		// Get user by email or username
		var existingUser *model.User
		var err error

		if isEmail {
			existingUser, err = s.userRepo.FindByEmail(username, authClient.AuthContainerID)
		} else {
			existingUser, err = s.userRepo.FindByUsername(username, authClient.AuthContainerID)
		}

		if err != nil {
			return err
		}

		// Check if user already exists
		if existingUser != nil {
			if isEmail {
				return errors.New("email already registered")
			}
			return errors.New("username already taken")
		}

		// Hash password
		hashed, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		if err != nil {
			return err
		}

		// Create user
		newUser := &model.User{
			Username:        username,
			Email:           "",
			Password:        ptr(string(hashed)),
			AuthContainerID: authClient.AuthContainerID,
			OrganizationID:  authClient.AuthContainer.OrganizationID,
			IsActive:        true,
		}
		if isEmail {
			newUser.Email = username
		}

		if err := tx.Create(newUser).Error; err != nil {
			return err
		}

		createdUser = newUser

		userIdentity := &model.UserIdentity{
			UserID:       createdUser.UserID,
			AuthClientID: authClient.AuthClientID,
			Provider:     "default",
			Sub:          createdUser.UserUUID.String(),
		}
		if err := tx.Create(userIdentity).Error; err != nil {
			return err
		}

		// Assign super admin role if first user in the organization
		superAdminRole, err := s.roleRepo.FindByNameAndAuthContainerID("super-admin", authClient.AuthContainer.AuthContainerID)
		if err != nil {
			return err
		}
		if superAdminRole == nil {
			return errors.New("super-admin role not found")
		}

		// Assign role
		role := &model.UserRole{
			UserID:    createdUser.UserID,
			RoleID:    superAdminRole.RoleID,
			IsDefault: true,
		}
		if err := tx.Create(role).Error; err != nil {
			return err
		}

		// Generate OTP
		otp, err := util.GenerateOTP(6)
		if err != nil {
			return err
		}

		// Create user token
		userToken := &model.UserToken{
			UserID:    createdUser.UserID,
			TokenType: "user:email:verification",
			Token:     otp,
		}
		if err := tx.Create(userToken).Error; err != nil {
			return err
		}

		return nil // commit transaction
	})

	if err != nil {
		return nil, err
	}

	// Return token response
	return s.generateTokenResponse(createdUser.UserUUID.String(), authClient)
}

func (s *registerService) RegisterPrivateInvite(
	username,
	password,
	inviteToken string,
) (*dto.AuthResponseDto, error) {
	// Get and validate client id (get default client)
	authClient, err := s.authClientRepo.FindDefault()
	if err != nil {
		return nil, err
	}
	if authClient == nil || !authClient.IsActive || authClient.Domain == nil || *authClient.Domain == "" || authClient.AuthContainer == nil {
		return nil, errors.New("invalid client or identity provider")
	}

	// Validate invite
	invite, err := s.inviteRepo.FindByToken(inviteToken)
	if err != nil {
		return nil, errors.New("invalid invite token")
	}
	if invite == nil || invite.Status != "pending" {
		return nil, errors.New("invite is not valid or already used")
	}
	if invite.ExpiresAt != nil && time.Now().After(*invite.ExpiresAt) {
		return nil, errors.New("invite has expired")
	}
	if invite.InvitedEmail != username {
		return nil, errors.New("invalid email")
	}

	var createdUser *model.User

	// Transaction
	err = s.db.Transaction(func(tx *gorm.DB) error {
		// Get user by email or username
		var existingUser *model.User
		var err error

		existingUser, err = s.userRepo.FindByEmail(username, authClient.AuthContainerID)

		if err != nil {
			return err
		}

		// Check if user already exists
		if existingUser != nil {
			return errors.New("email already registered")
		}

		// Hash password
		hashed, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		if err != nil {
			return err
		}

		// Create user
		newUser := &model.User{
			Username:        username,
			Email:           username,
			Password:        ptr(string(hashed)),
			AuthContainerID: authClient.AuthContainerID,
			OrganizationID:  authClient.AuthContainer.OrganizationID,
			IsEmailVerified: true,
			IsActive:        true,
		}

		if err := tx.Create(newUser).Error; err != nil {
			return err
		}

		createdUser = newUser

		// Create user roles
		inviteRoles := invite.Roles
		if len(inviteRoles) > 0 {
			for _, ir := range inviteRoles {
				userRole := &model.UserRole{
					UserID:    createdUser.UserID,
					RoleID:    ir.RoleID,
					IsDefault: false,
				}
				if err := tx.Create(userRole).Error; err != nil {
					return err
				}
			}
		}

		// Mark invite as used
		if err := s.inviteRepo.MarkAsUsed(invite.InviteUUID); err != nil {
			return err
		}

		return nil // commit transaction
	})

	if err != nil {
		return nil, err
	}

	// Return token response
	return s.generateTokenResponse(createdUser.UserUUID.String(), authClient)
}

func (s *registerService) RegisterPublicInvite(
	username,
	password,
	clientID,
	inviteToken string,
) (*dto.AuthResponseDto, error) {
	// Get and validate client id
	authClient, err := s.authClientRepo.FindByClientID(clientID)
	if err != nil {
		return nil, err
	}
	if authClient == nil || !authClient.IsActive || authClient.Domain == nil || *authClient.Domain == "" || authClient.AuthContainer == nil {
		return nil, errors.New("invalid client or identity provider")
	}

	// Check if username is an email
	isEmail := util.IsValidEmail(username)

	var createdUser *model.User

	// Transaction
	err = s.db.Transaction(func(tx *gorm.DB) error {
		// Get user by email or username
		var existingUser *model.User
		var err error

		if isEmail {
			existingUser, err = s.userRepo.FindByEmail(username, authClient.AuthContainerID)
		} else {
			existingUser, err = s.userRepo.FindByUsername(username, authClient.AuthContainerID)
		}

		if err != nil {
			return err
		}

		// Check if user already exists
		if existingUser != nil {
			if isEmail {
				return errors.New("email already registered")
			}
			return errors.New("username already taken")
		}

		// Hash password
		hashed, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		if err != nil {
			return err
		}

		// Create user
		newUser := &model.User{
			Username:        username,
			Email:           "",
			Password:        ptr(string(hashed)),
			AuthContainerID: authClient.AuthContainerID,
			OrganizationID:  authClient.AuthContainer.OrganizationID,
			IsActive:        true,
		}
		if isEmail {
			newUser.Email = username
		}

		if err := tx.Create(newUser).Error; err != nil {
			return err
		}

		createdUser = newUser

		// Generate OTP
		otp, err := util.GenerateOTP(6)
		if err != nil {
			return err
		}

		// Create user token
		userToken := &model.UserToken{
			UserID:    createdUser.UserID,
			TokenType: "user:email:verification",
			Token:     otp,
		}
		if err := tx.Create(userToken).Error; err != nil {
			return err
		}

		return nil // commit transaction
	})

	if err != nil {
		return nil, err
	}

	// Return token response
	return s.generateTokenResponse(createdUser.UserUUID.String(), authClient)
}

func (s *registerService) generateTokenResponse(userUUID string, authClient *model.AuthClient) (*dto.AuthResponseDto, error) {
	accessToken, err := util.GenerateAccessToken(
		userUUID,
		"openid profile email",
		*authClient.Domain,
		*authClient.ClientID,
		authClient.AuthContainer.AuthContainerUUID,
		*authClient.ClientID,
		authClient.IdentityProvider.IdentityProviderUUID,
	)
	if err != nil {
		return nil, err
	}

	idToken, err := util.GenerateIDToken(userUUID, *authClient.Domain, *authClient.ClientID)
	if err != nil {
		return nil, err
	}

	refreshToken, err := util.GenerateRefreshToken(userUUID, *authClient.Domain, *authClient.ClientID)
	if err != nil {
		return nil, err
	}

	return &dto.AuthResponseDto{
		AccessToken:  accessToken,
		IDToken:      idToken,
		RefreshToken: refreshToken,
		ExpiresIn:    3600,
		TokenType:    "Bearer",
		IssuedAt:     time.Now().Unix(),
	}, nil
}

func ptr(s string) *string {
	return &s
}
