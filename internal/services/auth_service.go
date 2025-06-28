package services

import (
	"errors"

	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/models"
	"github.com/maintainerd/auth/internal/repositories"
	"github.com/maintainerd/auth/internal/utils"
	"golang.org/x/crypto/bcrypt"
)

type AuthService interface {
	Register(username, email, password string) (string, error)
	Login(email, password string) (string, error)
}

type authService struct {
	userRepo repositories.UserRepository
}

func NewAuthService(userRepo repositories.UserRepository) AuthService {
	return &authService{userRepo}
}

func (s *authService) Register(username, email, password string) (string, error) {
	hashed, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}

	user := &models.User{
		UserUUID: uuid.New(),
		Username: username,
		Email:    email,
		Password: ptr(string(hashed)),
	}

	if err := s.userRepo.Create(user); err != nil {
		return "", err
	}

	return utils.GenerateToken(user.UserUUID.String())
}

func (s *authService) Login(email, password string) (string, error) {
	user, err := s.userRepo.FindByEmail(email)
	if err != nil {
		return "", errors.New("invalid credentials")
	}

	if user.Password == nil {
		return "", errors.New("invalid credentials")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(*user.Password), []byte(password)); err != nil {
		return "", errors.New("invalid credentials")
	}

	return utils.GenerateToken(user.UserUUID.String())
}

func ptr(s string) *string {
	return &s
}
