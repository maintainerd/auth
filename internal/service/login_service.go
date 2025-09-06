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

type LoginService interface {
	LoginPublic(usernameOrEmail, password, clientID string) (*dto.AuthResponseDto, error)
	LoginPrivate(usernameOrEmail, password string) (*dto.AuthResponseDto, error)
	GetUserByEmail(email string, authContainerID int64) (*model.User, error)
}

type loginService struct {
	db             *gorm.DB
	authClientRepo repository.AuthClientRepository
	userRepo       repository.UserRepository
	userTokenRepo  repository.UserTokenRepository
}

func NewLoginService(
	db *gorm.DB,
	authClientRepo repository.AuthClientRepository,
	userRepo repository.UserRepository,
	userTokenRepo repository.UserTokenRepository,
) LoginService {
	return &loginService{
		db:             db,
		authClientRepo: authClientRepo,
		userRepo:       userRepo,
		userTokenRepo:  userTokenRepo,
	}
}

func (s *loginService) LoginPublic(usernameOrEmail, password, clientID string) (*dto.AuthResponseDto, error) {
	// Get and validate client id (get default client)
	authClient, err := s.authClientRepo.FindByClientID(clientID)
	if err != nil {
		return nil, err
	}
	if authClient == nil ||
		!authClient.IsActive ||
		authClient.Domain == nil || *authClient.Domain == "" ||
		authClient.IdentityProvider == nil ||
		authClient.IdentityProvider.AuthContainer == nil ||
		authClient.IdentityProvider.AuthContainer.AuthContainerID == 0 {
		return nil, errors.New("invalid client or identity provider")
	}

	// Get container id
	authContainerId := authClient.IdentityProvider.AuthContainer.AuthContainerID

	// Get user
	var user *model.User
	if util.IsValidEmail(usernameOrEmail) {
		user, err = s.userRepo.FindByEmail(usernameOrEmail, authContainerId)
	} else {
		user, err = s.userRepo.FindByUsername(usernameOrEmail, authContainerId)
	}

	// Verify user and password
	if err != nil || user == nil || user.Password == nil {
		return nil, errors.New("invalid credentials")
	}

	// Compare password
	if err := bcrypt.CompareHashAndPassword([]byte(*user.Password), []byte(password)); err != nil {
		return nil, errors.New("invalid credentials")
	}

	// Generate token response
	return s.generateTokenResponse(user.UserUUID.String(), authClient)
}

func (s *loginService) LoginPrivate(username, password string) (*dto.AuthResponseDto, error) {
	// Get and validate client id (get default client)
	authClient, err := s.authClientRepo.FindDefault()
	if err != nil {
		return nil, err
	}
	if authClient == nil ||
		!authClient.IsActive ||
		authClient.Domain == nil || *authClient.Domain == "" ||
		authClient.IdentityProvider == nil ||
		authClient.IdentityProvider.AuthContainer == nil ||
		authClient.IdentityProvider.AuthContainer.AuthContainerID == 0 {
		return nil, errors.New("invalid client or identity provider")
	}

	// Get container id
	authContainerId := authClient.IdentityProvider.AuthContainer.AuthContainerID

	// Get user
	var user *model.User
	if util.IsValidEmail(username) {
		user, err = s.userRepo.FindByEmail(username, authContainerId)
	} else {
		user, err = s.userRepo.FindByUsername(username, authContainerId)
	}

	// Verify user and password
	if err != nil || user == nil || user.Password == nil {
		return nil, errors.New("invalid credentials")
	}

	// Compare password
	if err := bcrypt.CompareHashAndPassword([]byte(*user.Password), []byte(password)); err != nil {
		return nil, errors.New("invalid credentials")
	}

	// Generate token response
	return s.generateTokenResponse(user.UserUUID.String(), authClient)
}

func (s *loginService) GetUserByEmail(email string, authContainerID int64) (*model.User, error) {
	return s.userRepo.FindByEmail(email, authContainerID)
}

func (s *loginService) generateTokenResponse(userUUID string, authClient *model.AuthClient) (*dto.AuthResponseDto, error) {
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
