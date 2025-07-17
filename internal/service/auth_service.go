package service

import (
	"errors"

	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/model"
	"github.com/maintainerd/auth/internal/repository"
	"github.com/maintainerd/auth/internal/util"
	"golang.org/x/crypto/bcrypt"
)

type AuthService interface {
	Register(username, email, password, clientID, identityProviderID string) (string, error)
	Login(email, password string, authContainerID int64) (string, error)
	GetUserByEmail(email string, authContainerID int64) (*model.User, error)
}

type authService struct {
	userRepo   repository.UserRepository
	clientRepo repository.AuthClientRepository
}

func NewAuthService(userRepo repository.UserRepository, clientRepo repository.AuthClientRepository) AuthService {
	return &authService{
		userRepo:   userRepo,
		clientRepo: clientRepo,
	}
}

func ptr(s string) *string {
	return &s
}

func (s *authService) Register(
	username, email, password, clientID, identityProviderID string,
) (string, error) {

	client, err := s.clientRepo.FindByClientIDAndIdentityProvider(clientID, identityProviderID)
	if err != nil || !client.IsActive {
		return "", errors.New("invalid client or identity provider")
	}

	// Check for existing user based on username type (email or username)
	if util.IsValidEmail(username) {
		existingUser, err := s.userRepo.FindByEmail(username, client.AuthContainerID)
		if err == nil && existingUser != nil {
			return "", errors.New("email already registered")
		}
	} else {
		existingUser, err := s.userRepo.FindByUsername(username, client.AuthContainerID)
		if err == nil && existingUser != nil {
			return "", errors.New("username already taken")
		}
	}

	hashed, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}

	// Always store username; only store email if it's a valid email
	user := &model.User{
		UserUUID:        uuid.New(),
		Username:        username,
		Email:           "", // default empty
		Password:        ptr(string(hashed)),
		AuthContainerID: client.AuthContainerID,
		OrganizationID:  client.AuthContainer.OrganizationID,
		IsActive:        true,
	}

	if util.IsValidEmail(username) {
		user.Email = username
	}

	if err := s.userRepo.Create(user); err != nil {
		return "", err
	}

	return util.GenerateToken(user.UserUUID.String())
}

func (s *authService) Login(email, password string, authContainerID int64) (string, error) {
	user, err := s.userRepo.FindByEmail(email, authContainerID)
	if err != nil {
		return "", errors.New("invalid credentials")
	}

	if user.Password == nil {
		return "", errors.New("invalid credentials")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(*user.Password), []byte(password)); err != nil {
		return "", errors.New("invalid credentials")
	}

	return util.GenerateToken(user.UserUUID.String())
}

func (s *authService) GetUserByEmail(email string, authContainerID int64) (*model.User, error) {
	return s.userRepo.FindByEmail(email, authContainerID)
}
