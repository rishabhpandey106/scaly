package service

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

// UserRepository defines the user persistence operations required by AuthService.
type UserRepository interface {
	Create(email, hashedPassword string) (int, error)
	GetByEmail(email string) (int, string, error)
}

type AuthService struct {
	userRepo    UserRepository
	jwtSecret   []byte
	tokenExpiry time.Duration
}

func NewAuthService(userRepo UserRepository, jwtSecret string, tokenExpiry time.Duration) *AuthService {
	return &AuthService{
		userRepo:    userRepo,
		jwtSecret:   []byte(jwtSecret),
		tokenExpiry: tokenExpiry,
	}
}

func (s *AuthService) Signup(email, password string) (int, string, error) {

	_, _, err := s.userRepo.GetByEmail(email)
	if err == nil {
		return 0, "", errors.New("email already registered")
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return 0, "", err
	}

	userID, err := s.userRepo.Create(email, string(hashedPassword))
	if err != nil {
		return 0, "", err
	}

	token, err := s.generateToken(userID)
	if err != nil {
		return 0, "", err
	}

	return userID, token, nil
}

func (s *AuthService) Login(email, password string) (int, string, error) {
	userID, hashedPassword, err := s.userRepo.GetByEmail(email)
	if err != nil {
		return 0, "", errors.New("invalid credentials")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password)); err != nil {
		return 0, "", errors.New("invalid credentials")
	}

	token, err := s.generateToken(userID)
	if err != nil {
		return 0, "", err
	}

	return userID, token, nil
}

func (s *AuthService) generateToken(userID int) (string, error) {
	claims := jwt.MapClaims{
		"user_id": userID,
		"exp":     time.Now().Add(s.tokenExpiry).Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(s.jwtSecret)
}

func (s *AuthService) ValidateToken(tokenString string) (int, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return s.jwtSecret, nil
	})

	if err != nil {
		return 0, err
	}

	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		if floatUserID, ok := claims["user_id"].(float64); ok {
			return int(floatUserID), nil
		}
		return 0, errors.New("invalid token claims")
	}

	return 0, errors.New("invalid token")
}
