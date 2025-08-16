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
)

type AuthService interface {
	Register(username, email, password, clientID, identityProviderID string) (*dto.AuthResponse, error)
	Login(usernameOrEmail, password, clientID, identityProviderID string) (*dto.AuthResponse, error)
	GetUserByEmail(email string, authContainerID int64) (*model.User, error)
}

type authService struct {
	clientRepo    repository.AuthClientRepository
	userRepo      repository.UserRepository
	userTokenRepo repository.UserTokenRepository
}

func NewAuthService(
	clientRepo repository.AuthClientRepository,
	userRepo repository.UserRepository,
	userTokenRepo repository.UserTokenRepository,
) AuthService {
	return &authService{
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
	// Get auth client and check if exist and valid
	authClient, err := s.clientRepo.FindByClientIDAndIdentityProvider(clientID, identityProviderID)

	if err != nil {
		return nil, err
	}

	if authClient == nil || !authClient.IsActive || authClient.Domain == nil || *authClient.Domain == "" || authClient.AuthContainer == nil {
		return nil, errors.New("invalid client or identity provider")
	}

	// Check if username is an email
	isEmail := util.IsValidEmail(username)

	// Check if username or email already exists
	if isEmail {
		existingUser, err := s.userRepo.FindByEmail(username, authClient.AuthContainerID)
		if err != nil {
			return nil, err
		}
		if existingUser != nil {
			return nil, errors.New("email already registered")
		}
	} else {
		existingUser, err := s.userRepo.FindByUsername(username, authClient.AuthContainerID)
		if err != nil {
			return nil, err
		}
		if existingUser != nil {
			return nil, errors.New("username already taken")
		}
	}

	// Create password hash
	hashed, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
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

	createdUser, err := s.userRepo.Create(newUser)

	if err != nil {
		return nil, err
	}

	// Generate OTP
	otp, err := util.GenerateOTP(6)
	if err != nil {
		return nil, err
	}

	// Create user token
	userToken := &model.UserToken{
		TokenUUID: uuid.New(),
		UserID:    createdUser.UserID,
		TokenType: "user:email:verification",
		Token:     otp,
	}

	if _, err := s.userTokenRepo.Create(userToken); err != nil {
		return nil, err
	}

	// Return auth response
	return s.generateTokenResponse(createdUser.UserUUID.String(), authClient)
}

func (s *authService) Login(usernameOrEmail, password, clientID, identityProviderID string) (*dto.AuthResponse, error) {
	// Get auth client and check if exist and valid
	client, err := s.clientRepo.FindByClientIDAndIdentityProvider(clientID, identityProviderID)

	if err != nil || client == nil || !client.IsActive || client.Domain == nil || *client.Domain == "" || client.AuthContainer == nil {
		return nil, errors.New("invalid client or identity provider")
	}

	// Check if username or email exists
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

func (s *authService) generateTokenResponse(
	userUUID string,
	authClient *model.AuthClient,
) (*dto.AuthResponse, error) {
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
