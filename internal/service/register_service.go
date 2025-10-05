package service

import (
	"errors"
	"strconv"
	"time"

	"github.com/maintainerd/auth/internal/dto"
	"github.com/maintainerd/auth/internal/model"
	"github.com/maintainerd/auth/internal/repository"
	"github.com/maintainerd/auth/internal/util"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type RegisterService interface {
	Register(username, password, authClientID, authContainerID string) (*dto.AuthResponseDto, error)
	RegisterInvite(username, password, authClientID, authContainerID, inviteToken string) (*dto.AuthResponseDto, error)
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

func (s *registerService) Register(
	username,
	password,
	authClientID,
	authContainerID string,
) (*dto.AuthResponseDto, error) {
	// Get and validate auth client
	authClient, err := s.authClientRepo.FindByClientID(authClientID)
	if err != nil {
		return nil, err
	}
	if authClient == nil ||
		!authClient.IsActive ||
		authClient.Domain == nil || *authClient.Domain == "" {
		return nil, errors.New("invalid or inactive auth client")
	}

	// Parse auth container ID
	authContainerId, err := strconv.ParseInt(authContainerID, 10, 64)
	if err != nil {
		return nil, errors.New("invalid auth container ID format")
	}

	// Check if username is an email
	isEmail := util.IsValidEmail(username)

	var createdUser *model.User

	// Transaction
	err = s.db.Transaction(func(tx *gorm.DB) error {
		txUserRepo := s.userRepo.WithTx(tx)
		txUserTokenRepo := s.userTokenRepo.WithTx(tx)

		// Get user by email or username
		var existingUser *model.User
		var txErr error

		if isEmail {
			existingUser, txErr = txUserRepo.FindByEmail(username, authContainerId)
		} else {
			existingUser, txErr = txUserRepo.FindByUsername(username, authContainerId)
		}

		if txErr != nil {
			return txErr
		}

		// Check if user already exists
		if existingUser != nil {
			if isEmail {
				return errors.New("email already registered")
			}
			return errors.New("username already taken")
		}

		// Hash password
		hashed, txErr := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		if txErr != nil {
			return txErr
		}

		// Create user
		newUser := &model.User{
			Username:        username,
			Email:           "",
			Password:        util.Ptr(string(hashed)),
			AuthContainerID: authContainerId,
			IsActive:        true,
		}
		if isEmail {
			newUser.Email = username
		}

		createdUser, txErr = txUserRepo.Create(newUser)
		if txErr != nil {
			return txErr
		}

		// Generate OTP
		otp, txErr := util.GenerateOTP(6)
		if txErr != nil {
			return txErr
		}

		// Create user token
		userToken := &model.UserToken{
			UserID:    createdUser.UserID,
			TokenType: "user:email:verification",
			Token:     otp,
		}
		_, txErr = txUserTokenRepo.Create(userToken)
		if txErr != nil {
			return txErr
		}

		return nil // commit transaction
	})

	if err != nil {
		return nil, err
	}

	// Return token response
	return s.generateTokenResponse(createdUser.UserUUID.String(), createdUser, authClient)
}

func (s *registerService) RegisterInvite(
	username,
	password,
	authClientID,
	authContainerID,
	inviteToken string,
) (*dto.AuthResponseDto, error) {
	// Get and validate auth client
	authClient, err := s.authClientRepo.FindByClientID(authClientID)
	if err != nil {
		return nil, err
	}
	if authClient == nil ||
		!authClient.IsActive ||
		authClient.Domain == nil || *authClient.Domain == "" {
		return nil, errors.New("invalid or inactive auth client")
	}

	// Parse auth container ID
	authContainerId, err := strconv.ParseInt(authContainerID, 10, 64)
	if err != nil {
		return nil, errors.New("invalid auth container ID format")
	}

	// Validate invite token
	invite, err := s.inviteRepo.FindByToken(inviteToken)
	if err != nil {
		return nil, errors.New("invalid invite token")
	}
	if invite == nil {
		return nil, errors.New("invite not found")
	}

	// Check invite status and expiration
	if invite.Status != "pending" {
		return nil, errors.New("invite has already been used or is no longer valid")
	}
	if invite.ExpiresAt != nil && time.Now().After(*invite.ExpiresAt) {
		return nil, errors.New("invite has expired")
	}

	// Check if username is an email
	isEmail := util.IsValidEmail(username)

	// If username is email, validate it matches the invited email
	if isEmail && username != invite.InvitedEmail {
		return nil, errors.New("email must match the invited email address")
	}

	var createdUser *model.User

	// Transaction
	err = s.db.Transaction(func(tx *gorm.DB) error {
		txUserRepo := s.userRepo.WithTx(tx)
		txUserRoleRepo := s.userRoleRepo.WithTx(tx)
		txInviteRepo := s.inviteRepo.WithTx(tx)

		// Get user by email or username
		var existingUser *model.User
		var txErr error

		if isEmail {
			existingUser, txErr = txUserRepo.FindByEmail(username, authContainerId)
		} else {
			existingUser, txErr = txUserRepo.FindByUsername(username, authContainerId)
		}

		if txErr != nil {
			return txErr
		}

		// Check if user already exists
		if existingUser != nil {
			if isEmail {
				return errors.New("email already registered")
			}
			return errors.New("username already taken")
		}

		// Hash password
		hashed, txErr := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		if txErr != nil {
			return txErr
		}

		// Create user
		newUser := &model.User{
			Username:        username,
			Email:           invite.InvitedEmail, // Always use the invited email
			Password:        util.Ptr(string(hashed)),
			AuthContainerID: authContainerId,
			IsActive:        true,
			IsEmailVerified: true, // Auto-verify email for invited users
		}
		if !isEmail {
			// If username is not email, set the username but keep invited email
			newUser.Username = username
		}

		createdUser, txErr = txUserRepo.Create(newUser)
		if txErr != nil {
			return txErr
		}

		// Assign all roles from the invite to the user
		for _, role := range invite.Roles {
			// Check if user already has this role (shouldn't happen for new user, but safety check)
			existingUserRole, txErr := txUserRoleRepo.FindByUserIDAndRoleID(createdUser.UserID, role.RoleID)
			if txErr != nil && !errors.Is(txErr, gorm.ErrRecordNotFound) {
				return txErr
			}
			if existingUserRole != nil {
				continue // Skip if already assigned
			}

			// Create user-role association
			userRole := &model.UserRole{
				UserID: createdUser.UserID,
				RoleID: role.RoleID,
			}

			_, txErr = txUserRoleRepo.Create(userRole)
			if txErr != nil {
				return txErr
			}
		}

		// Mark invite as used (using repository method)
		txErr = txInviteRepo.MarkAsUsed(invite.InviteUUID)
		if txErr != nil {
			return txErr
		}

		return nil // commit transaction
	})

	if err != nil {
		return nil, err
	}

	// Return token response
	return s.generateTokenResponse(createdUser.UserUUID.String(), createdUser, authClient)
}

func (s *registerService) generateTokenResponse(userUUID string, user *model.User, authClient *model.AuthClient) (*dto.AuthResponseDto, error) {
	accessToken, err := util.GenerateAccessToken(
		userUUID,
		"openid profile email",
		*authClient.Domain,
		*authClient.ClientID,
		authClient.IdentityProvider.AuthContainer.AuthContainerUUID,
		*authClient.ClientID,
		authClient.IdentityProvider.IdentityProviderUUID,
	)
	if err != nil {
		return nil, err
	}

	// Create user profile for ID token (populate from user data)
	profile := &util.UserProfile{
		Email:         user.Email,
		EmailVerified: user.IsEmailVerified,
		Phone:         user.Phone,
		PhoneVerified: user.IsPhoneVerified,
	}

	// Generate ID token with user profile (no nonce for registration flow)
	idToken, err := util.GenerateIDToken(userUUID, *authClient.Domain, *authClient.ClientID, profile, "")
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
