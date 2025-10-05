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
		// Get user by email or username
		var existingUser *model.User
		var err error

		if isEmail {
			existingUser, err = s.userRepo.FindByEmail(username, authContainerId)
		} else {
			existingUser, err = s.userRepo.FindByUsername(username, authContainerId)
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
			AuthContainerID: authContainerId,
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

	// Check if username is an email
	isEmail := util.IsValidEmail(username)

	var createdUser *model.User

	// Transaction
	err = s.db.Transaction(func(tx *gorm.DB) error {
		// Get user by email or username
		var existingUser *model.User
		var err error

		if isEmail {
			existingUser, err = s.userRepo.FindByEmail(username, authContainerId)
		} else {
			existingUser, err = s.userRepo.FindByUsername(username, authContainerId)
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
			AuthContainerID: authContainerId,
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
