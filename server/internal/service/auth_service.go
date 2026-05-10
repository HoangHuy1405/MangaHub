package service

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"golang.org/x/crypto/bcrypt"

	"mangahub/internal/repository"
	"mangahub/pkg/models"
)

type AuthService interface {
	Register(input models.RegisterInput) (int64, error)
	Login(input models.LoginInput) (token string, user *models.User, err error)
}

type authServiceImpl struct {
	repo      repository.AuthRepository
	jwtSecret string
}

func NewAuthService(repo repository.AuthRepository, jwtSecret string) AuthService {
	return &authServiceImpl{repo: repo, jwtSecret: jwtSecret}
}

func (s *authServiceImpl) Register(input models.RegisterInput) (int64, error) {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
	if err != nil {
		return 0, fmt.Errorf("hash password: %w", err)
	}

	id, err := s.repo.CreateUser(input.Username, input.Email, string(hashedPassword))
	if err != nil {
		return 0, ErrConflict
	}

	return id, nil
}

func (s *authServiceImpl) Login(input models.LoginInput) (string, *models.User, error) {
	var user *models.User
	var err error

	if input.Email != "" {
		user, err = s.repo.FindByEmail(input.Email)
	} else {
		user, err = s.repo.FindByUsername(input.Username)
	}

	if err != nil {
		return "", nil, fmt.Errorf("find user: %w", err)
	}
	if user == nil {
		return "", nil, ErrUnauthorized
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(input.Password)); err != nil {
		return "", nil, ErrUnauthorized
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id":  user.ID,
		"username": user.Username,
		"exp":      time.Now().Add(24 * time.Hour).Unix(),
		"iat":      time.Now().Unix(),
	})

	tokenString, err := token.SignedString([]byte(s.jwtSecret))
	if err != nil {
		return "", nil, fmt.Errorf("sign jwt token: %w", err)
	}

	return tokenString, user, nil
}
