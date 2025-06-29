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
	Register(username, email, password string) (string, error)
	Login(email, password string) (string, error)
}

type authService struct {
	userRepo repository.UserRepository
}

func NewAuthService(userRepo repository.UserRepository) AuthService {
	return &authService{userRepo}
}

func (s *authService) Register(username, email, password string) (string, error) {
	hashed, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}

	user := &model.User{
		UserUUID: uuid.New(),
		Username: username,
		Email:    email,
		Password: ptr(string(hashed)),
	}

	if err := s.userRepo.Create(user); err != nil {
		return "", err
	}

	return util.GenerateToken(user.UserUUID.String())
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

	return util.GenerateToken(user.UserUUID.String())
}

func ptr(s string) *string {
	return &s
}
