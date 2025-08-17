package service

import (
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/dto"
	"github.com/maintainerd/auth/internal/model"
	"github.com/maintainerd/auth/internal/repository"
	"github.com/maintainerd/auth/internal/util"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type AuthService interface {
	Register(username, email, password, clientID, identityProviderID string) (*dto.AuthResponse, error)
	Login(usernameOrEmail, password, clientID, identityProviderID string) (*dto.AuthResponse, error)
	GetUserByEmail(email string, authContainerID int64) (*model.User, error)
}

type authService struct {
	db            *gorm.DB
	clientRepo    repository.AuthClientRepository
	userRepo      repository.UserRepository
	userTokenRepo repository.UserTokenRepository
}

func NewAuthService(
	db *gorm.DB,
	clientRepo repository.AuthClientRepository,
	userRepo repository.UserRepository,
	userTokenRepo repository.UserTokenRepository,
) AuthService {
	return &authService{
		db:            db,
		clientRepo:    clientRepo,
		userRepo:      userRepo,
		userTokenRepo: userTokenRepo,
	}
}

func ptr(s string) *string {
	return &s
}

func (s *authService) Register(
	username,
	email,
	password,
	clientID,
	identityProviderID string,
) (*dto.AuthResponse, error) {
	// Step 1: Validate client
	authClient, err := s.clientRepo.FindByClientIDAndIdentityProvider(clientID, identityProviderID)
	if err != nil {
		return nil, err
	}
	if authClient == nil || !authClient.IsActive || authClient.Domain == nil || *authClient.Domain == "" || authClient.AuthContainer == nil {
		return nil, errors.New("invalid client or identity provider")
	}

	isEmail := util.IsValidEmail(username)

	var createdUser *model.User

	// Step 2: Use GORM Transaction (best practice)
	err = s.db.Transaction(func(tx *gorm.DB) error {
		// Check if username/email already exists
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
			UserUUID:        uuid.New(),
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
			TokenUUID: uuid.New(),
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

	// Step 3: Return token response
	return s.generateTokenResponse(createdUser.UserUUID.String(), authClient)
}

func (s *authService) Login(usernameOrEmail, password, clientID, identityProviderID string) (*dto.AuthResponse, error) {
	client, err := s.clientRepo.FindByClientIDAndIdentityProvider(clientID, identityProviderID)
	if err != nil || client == nil || !client.IsActive || client.Domain == nil || *client.Domain == "" || client.AuthContainer == nil {
		return nil, errors.New("invalid client or identity provider")
	}

	var user *model.User
	if util.IsValidEmail(usernameOrEmail) {
		user, err = s.userRepo.FindByEmail(usernameOrEmail, client.AuthContainerID)
	} else {
		user, err = s.userRepo.FindByUsername(usernameOrEmail, client.AuthContainerID)
	}

	if err != nil || user == nil || user.Password == nil {
		return nil, errors.New("invalid credentials")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(*user.Password), []byte(password)); err != nil {
		return nil, errors.New("invalid credentials")
	}

	return s.generateTokenResponse(user.UserUUID.String(), client)
}

func (s *authService) GetUserByEmail(email string, authContainerID int64) (*model.User, error) {
	return s.userRepo.FindByEmail(email, authContainerID)
}

func (s *authService) generateTokenResponse(userUUID string, authClient *model.AuthClient) (*dto.AuthResponse, error) {
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

	return &dto.AuthResponse{
		AccessToken:  accessToken,
		IDToken:      idToken,
		RefreshToken: refreshToken,
		ExpiresIn:    3600,
		TokenType:    "Bearer",
		IssuedAt:     time.Now().Unix(),
	}, nil
}
